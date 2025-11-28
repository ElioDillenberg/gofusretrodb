package gofusretrodb

import (
	"time"
)

// Item represents a DOFUS item (from SWF parser)
type Item struct {
	ID           int               `json:"id"`
	TypeID       int               `json:"type_id"`
	Level        int               `json:"level"`
	Price        int               `json:"price"`
	Weight       int               `json:"weight"`
	GfxID        int               `json:"gfx_id"`
	Requirements string            `json:"requirements"`
	Translations []ItemTranslation `json:"translations"`
}

// ItemTranslation represents item text in a specific language (from SWF parser)
type ItemTranslation struct {
	Language    string `json:"language"`
	Name        string `json:"name"`
	NameUpper   string `json:"name_upper"`
	Description string `json:"description"`
}

// Database models
type ItemModel struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	AnkaId       int       `json:"anka_id" gorm:"default:0"`
	TypeAnkaId   int       `json:"type_anka_id" gorm:"default:0"` // References ItemType.AnkaId
	Level        int       `json:"level" gorm:"default:0"`
	Requirements string    `json:"requirements" gorm:"type:text"`
	StatsFormula string    `json:"stats_formula" gorm:"type:text"`
	Price        int       `json:"price" gorm:"default:0"`
	Weight       int       `json:"weight" gorm:"default:0"`
	GfxID        int       `json:"gfx_id" gorm:"default:0"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	// Type relationship: TypeAnkaId -> ItemTypeModel.AnkaId
	Type         *ItemTypeModel         `json:"type,omitempty" gorm:"foreignKey:TypeAnkaId;references:AnkaId"`
	Translations []ItemTranslationModel `json:"translations" gorm:"foreignKey:ItemID"`
	Conditions   []ItemConditionModel   `json:"conditions" gorm:"foreignKey:ItemID"`
	Recipe       *RecipeModel           `json:"recipe,omitempty" gorm:"foreignKey:ItemID"`
	Ingredients  []IngredientModel      `json:"ingredients,omitempty" gorm:"foreignKey:ItemID"`
	Stats        []ItemStatModel        `json:"itemstats,omitempty" gorm:"foreignKey:ItemID"`
}

func (ItemModel) TableName() string {
	return "items"
}

type ItemTranslationModel struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ItemID      uint      `json:"item_id" gorm:"not null"`
	Language    string    `json:"language" gorm:"size:5;not null"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	NameUpper   string    `json:"name_upper" gorm:"size:255"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Item        ItemModel `json:"item" gorm:"foreignKey:ItemID"`
}

func (ItemTranslationModel) TableName() string {
	return "item_translations"
}

type ItemTypeModel struct {
	ID           uint                       `json:"id" gorm:"primaryKey"`
	AnkaId       int                        `json:"anka_id" gorm:"uniqueIndex;default:0"` // Original SWF type ID
	KeyName      string                     `json:"key_name" gorm:"size:50"`
	Translations []ItemTypeTranslationModel `json:"translations" gorm:"foreignKey:ItemTypeID"`
}

func (ItemTypeModel) TableName() string {
	return "item_types"
}

type ItemTypeTranslationModel struct {
	ID         uint          `json:"id" gorm:"primaryKey"`
	ItemTypeID uint          `json:"item_type_id" gorm:"not null"`
	Language   string        `json:"language" gorm:"size:5;not null"`
	Name       string        `json:"name" gorm:"size:255;not null"`
	ItemType   ItemTypeModel `json:"item_type" gorm:"foreignKey:ItemTypeID"`
}

func (ItemTypeTranslationModel) TableName() string {
	return "item_type_translations"
}

//// ItemEffectModel represents item effects/stats
//type ItemEffectModel struct {
//	ID         uint      `json:"id" gorm:"primaryKey"`
//	ItemID     uint      `json:"item_id" gorm:"not null"`
//	EffectType int       `json:"effect_type" gorm:"not null"`
//	MinValue   int       `json:"min_value" gorm:"default:0"`
//	MaxValue   int       `json:"max_value" gorm:"default:0"`
//	CreatedAt  time.Time `json:"created_at"`
//	UpdatedAt  time.Time `json:"updated_at"`
//	Item       ItemModel `json:"item" gorm:"foreignKey:ItemID"`
//}
//
//func (ItemEffectModel) TableName() string {
//	return "item_effects"
//}

// ItemConditionModel represents item usage conditions
type ItemConditionModel struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ItemID        uint      `json:"item_id" gorm:"not null"`
	ConditionType int       `json:"condition_type" gorm:"not null"`
	ConditionSign int       `json:"condition_sign" gorm:"not null"`
	Value         int       `json:"value" gorm:"not null"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Item          ItemModel `json:"item" gorm:"foreignKey:ItemID"`
}

func (ItemConditionModel) TableName() string {
	return "item_conditions"
}

// ItemSetModel represents equipment sets
type ItemSetModel struct {
	ID           uint                      `json:"id" gorm:"primaryKey"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
	Items        []ItemModel               `json:"items" gorm:"many2many:item_set_items;"`
	Translations []ItemSetTranslationModel `json:"translations" gorm:"foreignKey:ItemSetID"`
}

func (ItemSetModel) TableName() string {
	return "item_sets"
}

// ItemSetTranslationModel represents set names in different languages
type ItemSetTranslationModel struct {
	ID        uint         `json:"id" gorm:"primaryKey"`
	ItemSetID uint         `json:"item_set_id" gorm:"not null"`
	Language  string       `json:"language" gorm:"size:5;not null"`
	Name      string       `json:"name" gorm:"size:255;not null"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	ItemSet   ItemSetModel `json:"item_set" gorm:"foreignKey:ItemSetID"`
}

func (ItemSetTranslationModel) TableName() string {
	return "item_set_translations"
}

func (StatTypeModel) TableName() string {
	return "stat_types"
}

func (StatTypeTranslationModel) TableName() string {
	return "stat_type_translations"
}

func (ItemStatModel) TableName() string {
	return "item_stats"
}

// RecipeModel represents crafting recipes
type RecipeModel struct {
	ID          uint              `json:"id" gorm:"primaryKey"`
	ItemID      uint              `json:"item_id" gorm:"not null"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Item        ItemModel         `json:"item" gorm:"foreignKey:ItemID"`
	Ingredients []IngredientModel `json:"ingredients" gorm:"foreignKey:RecipeID"`
}

func (RecipeModel) TableName() string {
	return "recipes"
}

// IngredientModel represents recipe ingredients
type IngredientModel struct {
	ID        uint        `json:"id" gorm:"primaryKey"`
	RecipeID  uint        `json:"recipe_id" gorm:"not null"`
	ItemID    uint        `json:"item_id" gorm:"not null"`
	Quantity  int         `json:"quantity" gorm:"not null"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Recipe    RecipeModel `json:"recipe" gorm:"foreignKey:RecipeID"`
	Item      ItemModel   `json:"item" gorm:"foreignKey:ItemID"`
}

func (IngredientModel) TableName() string {
	return "ingredients"
}

// Recipe represents a parsed crafting recipe (from SWF parser)
type Recipe struct {
	ItemID      int          `json:"item_id"`
	Ingredients []Ingredient `json:"ingredients"`
}

// Ingredient represents a recipe ingredient (from SWF parser)
type Ingredient struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

// ItemTypeDefinition represents an item type extracted from SWF files
type ItemTypeDefinition struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Category int    `json:"category"`
}

type StatTypeModel struct {
	ID           int                        `json:"id" gorm:"primaryKey"` // The hex code as integer (e.g., 100 for 0x64)
	Code         string                     `json:"code"`                 // Internal key like "vitality", "wisdom"
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
	DisplayOrder int                        `json:"display_order"`
	Translations []StatTypeTranslationModel `json:"translations" gorm:"foreignKey:StatTypeID"`
}

type ItemStat struct {
	StatTypeId int    `json:"item_stat_id"`
	ItemAnkaId int    `json:"item_anka_id"`
	MinValue   int    `json:"min_value"`
	MaxValue   int    `json:"max_value"`
	Formula    string `json:"formula"`
}

type ItemStatModel struct {
	ID         int           `json:"id" gorm:"primaryKey"`
	ItemID     int           `json:"item_id"`      // Foreign key to items.anka_id
	StatTypeID int           `json:"stat_type_id"` // Foreign key to stat_types.id
	StatType   StatTypeModel `json:"stat_type" gorm:"foreignKey:StatTypeID;references:ID"`
	MinValue   *int          `json:"min_value"`
	MaxValue   *int          `json:"max_value"`
	Formula    string        `json:"formula"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

type StatTypeTranslationModel struct {
	ID         int       `json:"id" db:"id"`
	StatTypeID int       `json:"stat_type_id" db:"stat_type_id"`
	Language   string    `json:"language" db:"language"` // "fr", "en", "es", etc.
	Name       string    `json:"name" db:"name"`         // Localized name
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// StatTypeSeedData contains the reference data for stat types
var StatTypeSeedData = []StatTypeModel{

	// Characteristics
	{ID: 0x7d, Code: "vitality", DisplayOrder: 17},     // svg icon ok
	{ID: 0x7b, Code: "chance", DisplayOrder: 18},       // svg icon ok
	{ID: 0x7e, Code: "intelligence", DisplayOrder: 19}, // svg icon ok
	{ID: 0x76, Code: "strength", DisplayOrder: 20},     // svg icon ok
	{ID: 0x77, Code: "agility", DisplayOrder: 21},      // svg icon ok
	{ID: 0x7c, Code: "wisdom", DisplayOrder: 23},       // svg icon ok

	// Combat Stats
	{ID: 0xb6, Code: "summon", DisplayOrder: 33},         // svg icon ok
	{ID: 0x80, Code: "mp", DisplayOrder: 14},             // svg icon ok
	{ID: 0x6f, Code: "ap", DisplayOrder: 13},             // svg icon ok
	{ID: 0x70, Code: "damage", DisplayOrder: 25},         // svg icon ok
	{ID: 0x8a, Code: "damage_percent", DisplayOrder: 22}, // svg icon ok
	{ID: 0x73, Code: "critical_hit", DisplayOrder: 26},   // svg icon ok
	{ID: 0x75, Code: "range", DisplayOrder: 15},          // svg icon ok
	{ID: 0xb2, Code: "heal", DisplayOrder: 32},           // svg icon ok
	//{ID: 0x73, Code: "critical_miss"},   // svg icon ok

	// Misc
	{ID: 0x9e, Code: "pods", DisplayOrder: 0},
	{ID: 0xae, Code: "initiative", DisplayOrder: 16},  // svg icon ok
	{ID: 0xb0, Code: "prospecting", DisplayOrder: 24}, // svg icon ok

	// Resistances
	{ID: 0xf4, Code: "neutral_resist", DisplayOrder: 34}, // svg icon ok
	{ID: 0xf1, Code: "water_resist", DisplayOrder: 35},   // svg icon ok
	{ID: 0xf3, Code: "fire_resist", DisplayOrder: 36},    // svg icon ok
	{ID: 0xf2, Code: "air_resist", DisplayOrder: 38},     // svg icon ok
	{ID: 0xf0, Code: "earth_resist", DisplayOrder: 37},   // svg icon ok

	// Resistances Percentage
	{ID: 0xd6, Code: "neutral_resist_percent", DisplayOrder: 27}, // svg icon ok
	{ID: 0xd3, Code: "water_resist_percent", DisplayOrder: 28},   // svg icon ok
	{ID: 0xd5, Code: "fire_resist_percent", DisplayOrder: 29},    // svg icon ok
	{ID: 0xd2, Code: "earth_resist_percent", DisplayOrder: 30},   // svg icon ok
	{ID: 0xd4, Code: "air_resist_percent", DisplayOrder: 31},     // svg icon ok

	// Weapon damage
	{ID: 0x64, Code: "neutral_damage", DisplayOrder: 1}, // svg icon ok
	{ID: 0x61, Code: "water_damage", DisplayOrder: 2},   // svg icon ok
	{ID: 0x63, Code: "fire_damage", DisplayOrder: 3},    // svg icon ok
	{ID: 0x60, Code: "earth_damage", DisplayOrder: 4},   // svg icon ok
	{ID: 0x62, Code: "air_damage", DisplayOrder: 5},     // svg icon ok

	{ID: 0x5f, Code: "neutral_life_steal", DisplayOrder: 6}, // svg icon ok
	{ID: 0x5b, Code: "water_life_steal", DisplayOrder: 7},   // svg icon ok
	{ID: 0x5e, Code: "fire_life_steal", DisplayOrder: 8},    // svg icon ok
	{ID: 0x5c, Code: "earth_life_steal", DisplayOrder: 9},   // svg icon ok
	{ID: 0x5d, Code: "air_life_steal", DisplayOrder: 10},    // svg icon ok

	{ID: 0x82, Code: "gold_steal", DisplayOrder: 12},
	{ID: 0x65, Code: "ap_kick", DisplayOrder: 11},

	//{ID: 0x65, Code: "ap_kick_resistance"},
	//{ID: 0x65, Code: "mp_kick_resistance"},

	// Special Stats
	{ID: 0xdc, Code: "reflect_damage", DisplayOrder: 38}, // svg icon ok
	{ID: 0xe1, Code: "trap_damage", DisplayOrder: 39},    // svg icon ok
	{ID: 0xe2, Code: "trap_damage_percent", DisplayOrder: 40},
	{ID: 0x86f, Code: "final_damage", DisplayOrder: 41},
	{ID: 0x31b, Code: "hunting_weapon", DisplayOrder: 42},
}

// StatTypeTranslations contains multilingual translations for stat types
var StatTypeTranslations = map[string]map[string]string{
	"gold_steal":             {"fr": "Vol de Kamas", "en": "Kamas steal", "es": "Robo de Kamas"},
	"ap_kick":                {"fr": "PA perdus à la cible", "en": "Lost AP for the target", "es": "PA perdidos por el blanco"},
	"hunting_weapon":         {"fr": "Arme de chasse", "en": "Hunting weapon", "es": "Arma de caza"},
	"vitality":               {"fr": "Vitalité", "en": "Vitality", "es": "Vitalidad"},
	"wisdom":                 {"fr": "Sagesse", "en": "Wisdom", "es": "Sabiduría"},
	"strength":               {"fr": "Force", "en": "Strength", "es": "Fuerza"},
	"intelligence":           {"fr": "Intelligence", "en": "Intelligence", "es": "Inteligencia"},
	"chance":                 {"fr": "Chance", "en": "Chance", "es": "Suerte"},
	"agility":                {"fr": "Agilité", "en": "Agility", "es": "Agilidad"},
	"ap":                     {"fr": "PA", "en": "AP", "es": "PA"},
	"mp":                     {"fr": "PM", "en": "MP", "es": "PM"},
	"range":                  {"fr": "Portée", "en": "Range", "es": "Alcance"},
	"initiative":             {"fr": "Initiative", "en": "Initiative", "es": "Iniciativa"},
	"prospecting":            {"fr": "Prospection", "en": "Prospecting", "es": "Prospección"},
	"pods":                   {"fr": "Pods", "en": "Pods", "es": "Pods"},
	"critical_hit":           {"fr": "Coups Critiques", "en": "Critical Hit", "es": "Golpe Crítico"},
	"neutral_resist":         {"fr": "Résistance Neutre", "en": "Neutral Resistance", "es": "Resistencia Neutral"},
	"earth_resist":           {"fr": "Résistance Terre", "en": "Earth Resistance", "es": "Resistencia Tierra"},
	"fire_resist":            {"fr": "Résistance Feu", "en": "Fire Resistance", "es": "Resistencia Fuego"},
	"water_resist":           {"fr": "Résistance Eau", "en": "Water Resistance", "es": "Resistencia Agua"},
	"air_resist":             {"fr": "Résistance Air", "en": "Air Resistance", "es": "Resistencia Aire"},
	"neutral_resist_percent": {"fr": "Résistance Neutre (%)", "en": "Neutral Resistance (%)", "es": "Resistencia Neutral (%)"},
	"earth_resist_percent":   {"fr": "Résistance Terre (%)", "en": "Earth Resistance (%)", "es": "Resistencia Tierra (%)"},
	"fire_resist_percent":    {"fr": "Résistance Feu (%)", "en": "Fire Resistance (%)", "es": "Resistencia Fuego (%)"},
	"water_resist_percent":   {"fr": "Résistance Eau (%)", "en": "Water Resistance (%)", "es": "Resistencia Agua (%)"},
	"air_resist_percent":     {"fr": "Résistance Air (%)", "en": "Air Resistance (%)", "es": "Resistencia Aire (%)"},
	"neutral_damage":         {"fr": "Dommages Neutre", "en": "Neutral Damage", "es": "Daño Neutral"},
	"earth_damage":           {"fr": "Dommages Terre", "en": "Earth Damage", "es": "Daño Tierra"},
	"fire_damage":            {"fr": "Dommages Feu", "en": "Fire Damage", "es": "Daño Fuego"},
	"water_damage":           {"fr": "Dommages Eau", "en": "Water Damage", "es": "Daño Agua"},
	"air_damage":             {"fr": "Dommages Air", "en": "Air Damage", "es": "Daño Aire"},
	"neutral_life_steal":     {"fr": "Vol de vie Neutre", "en": "Neutral life steal", "es": "Robo de vida Neutral"},
	"earth_life_steal":       {"fr": "Vol de vie Terre", "en": "Earth life steal", "es": "Robo de vida Tierra"},
	"fire_life_steal":        {"fr": "Vol de vie Feu", "en": "Fire life steal", "es": "Robo de vida Fuego"},
	"water_life_steal":       {"fr": "Vol de vie Eau", "en": "Water life steal", "es": "Robo de vida Agua"},
	"air_life_steal":         {"fr": "Vol de vie Air", "en": "Air life steal", "es": "Robo de vida Aire"},
	"heal":                   {"fr": "Soins", "en": "Heals", "es": "Curas"},
	"summon":                 {"fr": "Invocations", "en": "Summons", "es": "Invocaciones"},
	"reflect_damage":         {"fr": "Renvoie de Dommages", "en": "Reflect Damage", "es": "Reflejo de Daño"},
	"trap_damage":            {"fr": "Dommages Pièges", "en": "Trap Damage", "es": "Daño de Trampas"},
	"trap_damage_percent":    {"fr": "Dommages Pièges (%)", "en": "Trap Damage (%)", "es": "Daño de Trampas (%)"},
	"damage":                 {"fr": "Dommages", "en": "Damage", "es": "Daño"},
	"damage_percent":         {"fr": "Dommages (%)", "en": "Damage (%)", "es": "Daño (%)"},
	"final_damage":           {"fr": "Dommages finaux", "en": "Final damage", "es": "Daño final"},
}
