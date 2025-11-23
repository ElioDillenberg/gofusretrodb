# Item Stats Implementation Specification

## Overview

This document specifies the database models and API methods needed in the `gofusretrodb` project to support item stats extraction and storage. The item stats data comes from `itemstats_fr_1259.swf` files and contains approximately 13,000+ stat entries for items ranging from ID #2 to #20254.

## Data Source Format

The stats are extracted from ActionScript files in the format:
```actionscript
ISTA[927] = "64#15#23#1d15+20,7e#1a#23#1d10+25,76#1a#23#1d10+25,73#a##0d0+10";
ISTA[1050] = "64#10#19#1d10+15,7e#1a#23#1d10+25,76#1a#23#1d10+25";
```

### Format Explanation:
- **ISTA[X]**: Item ID (X)
- **Stat Format**: `{stat_type_hex}#{min}#{max}#{formula}`
- **Formula**: 
  - `1d10+15` means "roll 1 dice with 10 sides and add 15" (dynamic range)
  - `0d0+10` means "fixed value of 10" (no randomness)
  - Empty min/max (`##`) means the values are computed from formula only
- **Multiple Stats**: Separated by commas

### Example Parsing:
```
"64#15#23#1d15+20" = Vitality stat with min=15, max=23, formula="1d15+20"
"7e#1a#23#1d10+25" = Wisdom stat with min=26 (0x1a), max=35 (0x23), formula="1d10+25"
"73#a##0d0+10" = AP stat with min=10 (0xa), max=empty, formula="0d0+10" (fixed +10 AP)
```

## Required Database Models

### 1. `ItemStat` Model

Main table to store individual stats for each item.

```go
type ItemStat struct {
    ID            int       `json:"id" db:"id"`
    ItemID        int       `json:"item_id" db:"item_id"`           // Foreign key to items.anka_id
    StatTypeID    int       `json:"stat_type_id" db:"stat_type_id"` // Foreign key to stat_types.id
    MinValue      *int      `json:"min_value" db:"min_value"`       // Nullable - minimum stat value (hex decoded)
    MaxValue      *int      `json:"max_value" db:"max_value"`       // Nullable - maximum stat value (hex decoded)
    Formula       string    `json:"formula" db:"formula"`           // e.g., "1d10+15" or "0d0+10"
    DisplayOrder  int       `json:"display_order" db:"display_order"` // Order of stat in item description
    CreatedAt     time.Time `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}
```

**Database Schema:**
```sql
CREATE TABLE item_stats (
    id SERIAL PRIMARY KEY,
    item_id INTEGER NOT NULL REFERENCES items(anka_id) ON DELETE CASCADE,
    stat_type_id INTEGER NOT NULL REFERENCES stat_types(id),
    min_value INTEGER NULL,
    max_value INTEGER NULL,
    formula VARCHAR(50) NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_item_stats_item_id ON item_stats(item_id);
CREATE INDEX idx_item_stats_stat_type_id ON item_stats(stat_type_id);
```

### 2. `StatType` Model

Lookup table for stat types (characteristics, resistances, damage types, etc.).

```go
type StatType struct {
    ID          int                  `json:"id" db:"id"`          // The hex code as integer (e.g., 100 for 0x64)
    Key         string               `json:"key" db:"key"`        // Internal key like "vitality", "wisdom"
    Category    string               `json:"category" db:"category"` // "characteristic", "resistance", "damage", "special"
    CreatedAt   time.Time            `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time            `json:"updated_at" db:"updated_at"`
    Translations []StatTypeTranslation `json:"translations,omitempty"`
}
```

**Database Schema:**
```sql
CREATE TABLE stat_types (
    id INTEGER PRIMARY KEY,  -- Use the decimal value of the hex code
    key VARCHAR(50) NOT NULL UNIQUE,
    category VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 3. `StatTypeTranslation` Model

Multilingual names for stat types.

```go
type StatTypeTranslation struct {
    ID         int       `json:"id" db:"id"`
    StatTypeID int       `json:"stat_type_id" db:"stat_type_id"`
    Language   string    `json:"language" db:"language"` // "fr", "en", "es", etc.
    Name       string    `json:"name" db:"name"`         // Localized name
    CreatedAt  time.Time `json:"created_at" db:"created_at"`
    UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
```

**Database Schema:**
```sql
CREATE TABLE stat_type_translations (
    id SERIAL PRIMARY KEY,
    stat_type_id INTEGER NOT NULL REFERENCES stat_types(id) ON DELETE CASCADE,
    language VARCHAR(2) NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(stat_type_id, language)
);

CREATE INDEX idx_stat_type_translations_stat_type_id ON stat_type_translations(stat_type_id);
CREATE INDEX idx_stat_type_translations_language ON stat_type_translations(language);
```

## Stat Type Reference Data

Here are the known stat types that should be seeded into the `stat_types` table:

```go
var StatTypeSeedData = []StatType{
    // Characteristics
    {ID: 0x64, Key: "vitality", Category: "characteristic"},       // 100
    {ID: 0x73, Key: "action_points", Category: "characteristic"},  // 115
    {ID: 0x76, Key: "wisdom", Category: "characteristic"},         // 118
    {ID: 0x77, Key: "strength", Category: "characteristic"},       // 119
    {ID: 0x7b, Key: "agility", Category: "characteristic"},        // 123
    {ID: 0x7c, Key: "intelligence", Category: "characteristic"},   // 124
    {ID: 0x7d, Key: "chance", Category: "characteristic"},         // 125
    {ID: 0x7e, Key: "wisdom_alt", Category: "characteristic"},     // 126 (alternative wisdom?)
    {ID: 0x7f, Key: "pods", Category: "characteristic"},           // 127
    {ID: 0x80, Key: "prospecting", Category: "characteristic"},    // 128
    
    // Combat Stats
    {ID: 0x60, Key: "damage", Category: "damage"},                 // 96
    {ID: 0x62, Key: "damage_percent", Category: "damage"},         // 98
    {ID: 0x6e, Key: "critical_hit", Category: "combat"},           // 110
    {ID: 0x6f, Key: "initiative", Category: "combat"},             // 111
    {ID: 0x70, Key: "range", Category: "combat"},                  // 112
    {ID: 0x73, Key: "ap", Category: "combat"},                     // 115
    {ID: 0x75, Key: "summon", Category: "combat"},                 // 117
    {ID: 0x99, Key: "range_bonus", Category: "combat"},            // 153
    
    // Resistances
    {ID: 0x98, Key: "neutral_resist", Category: "resistance"},     // 152
    {ID: 0x9a, Key: "earth_resist", Category: "resistance"},       // 154
    {ID: 0x9b, Key: "fire_resist", Category: "resistance"},        // 155
    {ID: 0x9c, Key: "water_resist", Category: "resistance"},       // 156
    {ID: 0x9d, Key: "air_resist", Category: "resistance"},         // 157
    {ID: 0x9e, Key: "neutral_resist_percent", Category: "resistance"}, // 158
    {ID: 0xae, Key: "earth_resist_percent", Category: "resistance"},   // 174
    
    // Elemental Damage
    {ID: 0xf0, Key: "neutral_damage", Category: "damage"},         // 240
    {ID: 0xf1, Key: "earth_damage", Category: "damage"},           // 241
    {ID: 0xf2, Key: "fire_damage", Category: "damage"},            // 242
    {ID: 0xf3, Key: "water_damage", Category: "damage"},           // 243
    {ID: 0xf4, Key: "air_damage", Category: "damage"},             // 244
    
    // Special Stats
    {ID: 0x8a, Key: "heal", Category: "special"},                  // 138
    {ID: 0x8b, Key: "reflect_damage", Category: "special"},        // 139
    {ID: 0x209, Key: "ap_reduction", Category: "special"},         // 521
    {ID: 0x25b, Key: "trap_damage", Category: "special"},          // 603
    {ID: 0x259, Key: "trap_power", Category: "special"},           // 601
    {ID: 0x31b, Key: "dodge", Category: "special"},                // 795
    {ID: 0x320, Key: "lock", Category: "special"},                 // 800
    {ID: 0x834, Key: "mp_reduction", Category: "special"},         // 2100
    
    // Add more as discovered during extraction...
}
```

## Required Database Service Methods

Add these methods to the `DatabaseService` in gofusretrodb:

### 1. Save Item Stats

```go
// SaveItemStats saves item stats for multiple items
// Input: map[itemID][]ItemStat
func (db *DatabaseService) SaveItemStats(stats map[int][]ItemStat) error
```

**Implementation Notes:**
- Batch insert for performance (use transactions)
- Clear existing stats for an item before inserting new ones (UPDATE operation)
- Handle nullable min/max values properly
- Set display_order based on array position

### 2. Get Item Stats by Item ID

```go
// GetItemStatsByItemID retrieves all stats for a specific item
func (db *DatabaseService) GetItemStatsByItemID(itemID int) ([]ItemStat, error)
```

**Implementation Notes:**
- Include stat type information (join with stat_types)
- Order by display_order
- Return empty array if no stats found (not error)

### 3. Get Item with Stats

```go
// GetItemWithStats retrieves an item with all its stats and translations
func (db *DatabaseService) GetItemWithStats(itemID int, language string) (*ItemWithStats, error)

type ItemWithStats struct {
    Item  *Item       `json:"item"`
    Stats []ItemStatWithType `json:"stats"`
}

type ItemStatWithType struct {
    ItemStat
    StatType     StatType `json:"stat_type"`
}
```

### 4. Seed Stat Types

```go
// SeedStatTypes initializes the stat_types and stat_type_translations tables
func (db *DatabaseService) SeedStatTypes() error
```

**Implementation Notes:**
- Use UPSERT (ON CONFLICT DO UPDATE) to avoid duplicates
- Should be idempotent (can run multiple times safely)
- Include translations for at least French and English

### 5. Get Stat Type by ID

```go
// GetStatTypeByID retrieves a stat type with translations
func (db *DatabaseService) GetStatTypeByID(id int) (*StatType, error)
```

## Translation Data for Stat Types

Provide at least French translations (English can be added later):

```go
var StatTypeTranslations = map[string]map[string]string{
    "vitality":        {"fr": "Vitalité", "en": "Vitality"},
    "wisdom":          {"fr": "Sagesse", "en": "Wisdom"},
    "strength":        {"fr": "Force", "en": "Strength"},
    "intelligence":    {"fr": "Intelligence", "en": "Intelligence"},
    "chance":          {"fr": "Chance", "en": "Chance"},
    "agility":         {"fr": "Agilité", "en": "Agility"},
    "action_points":   {"fr": "PA", "en": "AP"},
    "range":           {"fr": "Portée", "en": "Range"},
    "initiative":      {"fr": "Initiative", "en": "Initiative"},
    "prospecting":     {"fr": "Prospection", "en": "Prospecting"},
    "pods":            {"fr": "Pods", "en": "Pods"},
    "critical_hit":    {"fr": "Coups Critiques", "en": "Critical Hit"},
    "neutral_resist":  {"fr": "Résistance Neutre", "en": "Neutral Resistance"},
    "earth_resist":    {"fr": "Résistance Terre", "en": "Earth Resistance"},
    "fire_resist":     {"fr": "Résistance Feu", "en": "Fire Resistance"},
    "water_resist":    {"fr": "Résistance Eau", "en": "Water Resistance"},
    "air_resist":      {"fr": "Résistance Air", "en": "Air Resistance"},
    "neutral_damage":  {"fr": "Dommages Neutre", "en": "Neutral Damage"},
    "earth_damage":    {"fr": "Dommages Terre", "en": "Earth Damage"},
    "fire_damage":     {"fr": "Dommages Feu", "en": "Fire Damage"},
    "water_damage":    {"fr": "Dommages Eau", "en": "Water Damage"},
    "air_damage":      {"fr": "Dommages Air", "en": "Air Damage"},
    "heal":            {"fr": "Soins", "en": "Heals"},
    "summon":          {"fr": "Invocations", "en": "Summons"},
    "reflect_damage":  {"fr": "Renvoie de Dommages", "en": "Reflect Damage"},
    "trap_damage":     {"fr": "Dommages Pièges", "en": "Trap Damage"},
    "trap_power":      {"fr": "Puissance Pièges", "en": "Trap Power"},
    "dodge":           {"fr": "Esquive", "en": "Dodge"},
    "lock":            {"fr": "Tacle", "en": "Lock"},
    "damage":          {"fr": "Dommages", "en": "Damage"},
    "damage_percent":  {"fr": "Dommages (%)", "en": "Damage (%)"},
    // Add more as needed...
}
```

## Expected Usage from go-retro-lang-getter

Once implemented, the extraction code will:

1. **Parse the itemstats SWF files**:
   ```go
   parser := NewItemStatsParser()
   stats := parser.ParseItemStatsFile("./data/lang/swf/itemstats_fr_1259.swf")
   ```

2. **Convert hex values to integers**:
   ```go
   // "64#15#23#1d15+20" → ItemStat{StatTypeID: 100, MinValue: 21, MaxValue: 35, Formula: "1d15+20"}
   ```

3. **Save to database**:
   ```go
   err := db.SaveItemStats(stats)
   ```

4. **Query items with stats**:
   ```go
   itemWithStats, err := db.GetItemWithStats(927, "fr")
   // Returns item #927 with all its stats and localized stat names
   ```

## Migration Order

1. Create `stat_types` table
2. Create `stat_type_translations` table
3. Create `item_stats` table
4. Run seed data for stat_types and translations
5. Extract and save item stats from SWF files

## Testing Checklist

- [ ] Create all three tables successfully
- [ ] Seed stat types with at least 30+ known stat types
- [ ] Seed translations for French and English
- [ ] Save item stats for test item (e.g., item #927)
- [ ] Query item stats by item ID
- [ ] Query item with stats including translations
- [ ] Verify nullable fields work correctly (min/max can be NULL)
- [ ] Verify display_order maintains stat sequence
- [ ] Verify cascade delete works (deleting item deletes its stats)

## Data Volume Estimates

- **Items with stats**: ~13,000 items
- **Average stats per item**: 3-7 stats
- **Total stat records**: ~50,000-90,000 rows in item_stats table
- **Stat types**: ~50-100 unique stat types
- **Translations**: ~100-200 rows (2 languages × ~50 stat types)

## Questions to Consider

1. Should we store the raw stat string from SWF as well for debugging?
2. Do we need versioning for stat changes over time?
3. Should we validate formula format on insert?
4. Do we need indexes on formula field for searching?

## Additional Notes

- The hex stat IDs are used as primary keys in `stat_types` to maintain consistency with game data
- Not all items have stats (quest items, resources, etc. won't appear in ISTA)
- Some stat IDs may be unknown - these should be stored but flagged for manual review
- Formula parsing will be handled in go-retro-lang-getter, database just stores the string

---

**Document Version**: 1.0  
**Date**: November 22, 2025  
**Author**: GitHub Copilot  
**For Project**: gofusretrodb

