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
	ID             uint                       `json:"id" gorm:"primaryKey"`
	AnkaId         int                        `json:"anka_id" gorm:"uniqueIndex;default:0"` // Original SWF type ID
	KeyName        string                     `json:"key_name" gorm:"size:50"`
	AuctionHouseID *uint                      `json:"auction_house_id" gorm:"index"`
	AuctionHouse   *AuctionHouseModel         `json:"auction_house,omitempty" gorm:"foreignKey:AuctionHouseID"`
	Translations   []ItemTypeTranslationModel `json:"translations" gorm:"foreignKey:ItemTypeID"`
}

func (ItemTypeModel) TableName() string {
	return "item_types"
}

// AuctionHouseModel represents an in-game auction house location
type AuctionHouseModel struct {
	ID           uint                           `json:"id" gorm:"primaryKey"`
	Code         string                         `json:"code" gorm:"size:50;uniqueIndex;not null"`
	DisplayOrder int                            `json:"display_order" gorm:"default:0"`
	CreatedAt    time.Time                      `json:"created_at"`
	UpdatedAt    time.Time                      `json:"updated_at"`
	Translations []AuctionHouseTranslationModel `json:"translations" gorm:"foreignKey:AuctionHouseID"`
}

func (AuctionHouseModel) TableName() string {
	return "auction_houses"
}

// AuctionHouseTranslationModel represents auction house names in different languages
type AuctionHouseTranslationModel struct {
	ID             uint              `json:"id" gorm:"primaryKey"`
	AuctionHouseID uint              `json:"auction_house_id" gorm:"not null"`
	Language       string            `json:"language" gorm:"size:5;not null"`
	Name           string            `json:"name" gorm:"size:255;not null"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	AuctionHouse   AuctionHouseModel `json:"auction_house" gorm:"foreignKey:AuctionHouseID"`
}

func (AuctionHouseTranslationModel) TableName() string {
	return "auction_house_translations"
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

func (StatTypeCategoryModel) TableName() string {
	return "stat_type_categories"
}

func (StatTypeCategoryTranslationModel) TableName() string {
	return "stat_type_category_translations"
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

// StatTypeCategoryModel represents a category grouping for stat types
type StatTypeCategoryModel struct {
	ID           int                                `json:"id" gorm:"primaryKey"`
	Code         string                             `json:"code" gorm:"size:50;uniqueIndex;not null"` // e.g., "main", "resistance", "resistance_percentage", "misc"
	DisplayOrder int                                `json:"display_order"`
	CreatedAt    time.Time                          `json:"created_at"`
	UpdatedAt    time.Time                          `json:"updated_at"`
	Translations []StatTypeCategoryTranslationModel `json:"translations" gorm:"foreignKey:CategoryID"`
}

// StatTypeCategoryTranslationModel represents translations for stat type categories
type StatTypeCategoryTranslationModel struct {
	ID         int                   `json:"id" gorm:"primaryKey"`
	CategoryID int                   `json:"category_id" gorm:"not null"`
	Language   string                `json:"language" gorm:"size:5;not null"`
	Name       string                `json:"name" gorm:"size:255;not null"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
	Category   StatTypeCategoryModel `json:"category" gorm:"foreignKey:CategoryID"`
}

type StatTypeModel struct {
	ID           int                        `json:"id" gorm:"primaryKey"` // The hex code as integer (e.g., 100 for 0x64)
	Code         string                     `json:"code"`                 // Internal key like "vitality", "wisdom"
	CategoryID   int                        `json:"category_id"`          // Foreign key to stat_type_categories.id
	Category     *StatTypeCategoryModel     `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:ID"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
	DisplayOrder int                        `json:"display_order"`
	Translations []StatTypeTranslationModel `json:"translations" gorm:"foreignKey:StatTypeID"`
	Runes        []RuneModel                `json:"runes,omitempty" gorm:"foreignKey:StatTypeID;references:ID"` // Associated runes for this stat type
}

type ItemStat struct {
	StatTypeId int    `json:"item_stat_id"`
	ItemAnkaId int    `json:"item_anka_id"`
	MinValue   *int   `json:"min_value"`
	MaxValue   *int   `json:"max_value"`
	Formula    string `json:"formula"`
}

type ItemStatModel struct {
	ID         int           `json:"id" gorm:"primaryKey"`
	ItemID     uint          `json:"item_id"`      // Foreign key to items.id (primary key)
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

// StatTypeCategorySeedData contains the reference data for stat type categories
var StatTypeCategorySeedData = []StatTypeCategoryModel{
	{ID: 1, Code: "main", DisplayOrder: 1},
	{ID: 2, Code: "resistance", DisplayOrder: 2},
	{ID: 3, Code: "resistance_percentage", DisplayOrder: 3},
	{ID: 4, Code: "misc", DisplayOrder: 4},
	{ID: 5, Code: "weapon", DisplayOrder: 5},
	{ID: 6, Code: "combat", DisplayOrder: 6},
}

// StatTypeCategoryTranslations contains multilingual translations for stat type categories
var StatTypeCategoryTranslations = map[string]map[string]string{
	"main":                  {"fr": "Caractéristiques", "en": "Main Stats", "es": "Características"},
	"resistance":            {"fr": "Résistances", "en": "Resistances", "es": "Resistencias"},
	"resistance_percentage": {"fr": "Résistances (%)", "en": "Resistances (%)", "es": "Resistencias (%)"},
	"misc":                  {"fr": "Divers", "en": "Miscellaneous", "es": "Varios"},
	"weapon":                {"fr": "Arme", "en": "Weapon", "es": "Arma"},
	"combat":                {"fr": "Combat", "en": "Combat", "es": "Combate"},
}

// StatTypeSeedData contains the reference data for stat types
var StatTypeSeedData = []StatTypeModel{
	// Characteristics (main category)
	{ID: 0x7d, Code: "vitality", CategoryID: 1, DisplayOrder: 17},     // svg icon ok
	{ID: 0x7b, Code: "chance", CategoryID: 1, DisplayOrder: 18},       // svg icon ok
	{ID: 0x7e, Code: "intelligence", CategoryID: 1, DisplayOrder: 19}, // svg icon ok
	{ID: 0x76, Code: "strength", CategoryID: 1, DisplayOrder: 20},     // svg icon ok
	{ID: 0x77, Code: "agility", CategoryID: 1, DisplayOrder: 21},      // svg icon ok
	{ID: 0x7c, Code: "wisdom", CategoryID: 1, DisplayOrder: 23},       // svg icon ok

	// Combat Stats (combat category)
	{ID: 0xb6, Code: "summon", CategoryID: 6, DisplayOrder: 33},         // svg icon ok
	{ID: 0x80, Code: "mp", CategoryID: 6, DisplayOrder: 14},             // svg icon ok
	{ID: 0x6f, Code: "ap", CategoryID: 6, DisplayOrder: 13},             // svg icon ok
	{ID: 0x70, Code: "damage", CategoryID: 6, DisplayOrder: 25},         // svg icon ok
	{ID: 0x8a, Code: "damage_percent", CategoryID: 6, DisplayOrder: 22}, // svg icon ok
	{ID: 0x73, Code: "critical_hit", CategoryID: 6, DisplayOrder: 26},   // svg icon ok
	{ID: 0x75, Code: "range", CategoryID: 6, DisplayOrder: 15},          // svg icon ok
	{ID: 0xb2, Code: "heal", CategoryID: 6, DisplayOrder: 32},           // svg icon ok
	//{ID: 0x73, Code: "critical_miss"},   // svg icon ok

	// Misc (misc category)
	{ID: 0x9e, Code: "pods", CategoryID: 4, DisplayOrder: 0},
	{ID: 0xae, Code: "initiative", CategoryID: 4, DisplayOrder: 16},  // svg icon ok
	{ID: 0xb0, Code: "prospecting", CategoryID: 4, DisplayOrder: 24}, // svg icon ok

	// Resistances (resistance category)
	{ID: 0xf4, Code: "neutral_resist", CategoryID: 2, DisplayOrder: 34}, // svg icon ok
	{ID: 0xf1, Code: "water_resist", CategoryID: 2, DisplayOrder: 35},   // svg icon ok
	{ID: 0xf3, Code: "fire_resist", CategoryID: 2, DisplayOrder: 36},    // svg icon ok
	{ID: 0xf2, Code: "air_resist", CategoryID: 2, DisplayOrder: 38},     // svg icon ok
	{ID: 0xf0, Code: "earth_resist", CategoryID: 2, DisplayOrder: 37},   // svg icon ok

	// Resistances Percentage (resistance_percentage category)
	{ID: 0xd6, Code: "neutral_resist_percent", CategoryID: 3, DisplayOrder: 27}, // svg icon ok
	{ID: 0xd3, Code: "water_resist_percent", CategoryID: 3, DisplayOrder: 28},   // svg icon ok
	{ID: 0xd5, Code: "fire_resist_percent", CategoryID: 3, DisplayOrder: 29},    // svg icon ok
	{ID: 0xd2, Code: "earth_resist_percent", CategoryID: 3, DisplayOrder: 30},   // svg icon ok
	{ID: 0xd4, Code: "air_resist_percent", CategoryID: 3, DisplayOrder: 31},     // svg icon ok

	// Weapon damage (weapon category)
	{ID: 0x64, Code: "neutral_damage", CategoryID: 5, DisplayOrder: 1}, // svg icon ok
	{ID: 0x60, Code: "water_damage", CategoryID: 5, DisplayOrder: 2},   // svg icon ok
	{ID: 0x63, Code: "fire_damage", CategoryID: 5, DisplayOrder: 3},    // svg icon ok
	{ID: 0x61, Code: "earth_damage", CategoryID: 5, DisplayOrder: 4},   // svg icon ok
	{ID: 0x62, Code: "air_damage", CategoryID: 5, DisplayOrder: 5},     // svg icon ok

	{ID: 0x5f, Code: "neutral_life_steal", CategoryID: 5, DisplayOrder: 6}, // svg icon ok
	{ID: 0x5b, Code: "water_life_steal", CategoryID: 5, DisplayOrder: 7},   // svg icon ok
	{ID: 0x5e, Code: "fire_life_steal", CategoryID: 5, DisplayOrder: 8},    // svg icon ok
	{ID: 0x5c, Code: "earth_life_steal", CategoryID: 5, DisplayOrder: 9},   // svg icon ok
	{ID: 0x5d, Code: "air_life_steal", CategoryID: 5, DisplayOrder: 10},    // svg icon ok

	{ID: 0x82, Code: "gold_steal", CategoryID: 5, DisplayOrder: 12},
	{ID: 0x65, Code: "ap_kick", CategoryID: 5, DisplayOrder: 11},

	//{ID: 0x65, Code: "ap_kick_resistance"},
	//{ID: 0x65, Code: "mp_kick_resistance"},

	// Special Stats (combat category)
	{ID: 0xdc, Code: "reflect_damage", CategoryID: 6, DisplayOrder: 38}, // svg icon ok
	{ID: 0xe1, Code: "trap_damage", CategoryID: 6, DisplayOrder: 39},    // svg icon ok
	{ID: 0xe2, Code: "trap_damage_percent", CategoryID: 6, DisplayOrder: 40},
	{ID: 0x86f, Code: "final_damage", CategoryID: 6, DisplayOrder: 41},
	{ID: 0x31b, Code: "hunting_weapon", CategoryID: 5, DisplayOrder: 42},
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
	"neutral_damage":         {"fr": "Dégats Neutre", "en": "Neutral Damage", "es": "Daño Neutral"},
	"earth_damage":           {"fr": "Dégats Terre", "en": "Earth Damage", "es": "Daño Tierra"},
	"fire_damage":            {"fr": "Dégats Feu", "en": "Fire Damage", "es": "Daño Fuego"},
	"water_damage":           {"fr": "Dégats Eau", "en": "Water Damage", "es": "Daño Agua"},
	"air_damage":             {"fr": "Dégats Air", "en": "Air Damage", "es": "Daño Aire"},
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

// AuctionHouseSeedData contains the reference data for auction houses
// These represent the different Hôtel de vente (Hôtel de Vente) locations in-game
var AuctionHouseSeedData = []AuctionHouseModel{
	{ID: 1, Code: "resources", DisplayOrder: 1},
	{ID: 2, Code: "alchemist", DisplayOrder: 2},
	{ID: 3, Code: "jeweller", DisplayOrder: 3},
	{ID: 4, Code: "butcher&hunter", DisplayOrder: 4},
	{ID: 5, Code: "baker", DisplayOrder: 5},
	{ID: 6, Code: "handymen", DisplayOrder: 6},
	{ID: 7, Code: "lumberjack", DisplayOrder: 7},
	{ID: 8, Code: "shoemaker", DisplayOrder: 8},
	{ID: 9, Code: "documents", DisplayOrder: 9},
	{ID: 10, Code: "blacksmith", DisplayOrder: 10},
	{ID: 11, Code: "miners", DisplayOrder: 11},
	{ID: 12, Code: "linkedparchments", DisplayOrder: 12},
	{ID: 13, Code: "farmer", DisplayOrder: 13},
	{ID: 14, Code: "fishermen&fishmonger", DisplayOrder: 14},
	{ID: 15, Code: "runes", DisplayOrder: 15},
	{ID: 16, Code: "carver", DisplayOrder: 16},
	{ID: 17, Code: "tailor", DisplayOrder: 17},
	{ID: 18, Code: "souls", DisplayOrder: 18},
	{ID: 19, Code: "animals", DisplayOrder: 19},
	{ID: 20, Code: "shields", DisplayOrder: 19},
}

// AuctionHouseTranslations contains multilingual translations for auction houses
var AuctionHouseTranslations = map[string]map[string]string{
	"resources":            {"fr": "Hôtel de vente des ressources", "en": "Resource Market", "es": "Mercadillo de recursos"},
	"alchemist":            {"fr": "Hôtel de vente des alchimistes", "en": "Alchemists' Market", "es": "Mercadillo de alquimistas"},
	"jeweller":             {"fr": "Hôtel de vente des bijoutiers", "en": "Jewellers' Market", "es": "Mercadillo de joyeros"},
	"butcher&hunter":       {"fr": "Hôtel de vente des bouchers et des chasseurs", "en": "Butchers and Hunters' Market", "es": "Mercadillo de carniceros y cazadores"},
	"baker":                {"fr": "Hôtel de vente des boulangers", "en": "Bakers' Market", "es": "Mercadillo de panaderos"},
	"handymen":             {"fr": "Hôtel de vente des bricoleurs", "en": "Handymens' Market", "es": "Mercadillo de manitas"},
	"lumberjack":           {"fr": "Hôtel de vente des bûcherons", "en": "Lumberjacks' Market", "es": "Mercadillo de leñadores"},
	"shoemaker":            {"fr": "Hôtel de vente des cordonniers", "en": "Shoemakers' Market", "es": "Mercadillo de zapateros"},
	"documents":            {"fr": "Hôtel de vente des documents", "en": "Scroll Market", "es": "Mercadillo de documentos"},
	"blacksmith":           {"fr": "Hôtel de vente des forgerons", "en": "Smiths' Market", "es": "Mercadillo de herreros"},
	"miners":               {"fr": "Hôtel de vente des mineurs", "en": "Miners' Market", "es": "Mercadillo de mineros"},
	"linkedparchments":     {"fr": "Hôtel de vente des parchemins liés", "en": "Linked Scroll Market", "es": "Mercadillo de pergaminos ligados"},
	"farmer":               {"fr": "Hôtel de vente des paysans", "en": "Farmers' Market", "es": "Mercadillo de campesinos"},
	"fishermen&fishmonger": {"fr": "Hôtel de vente des poissoniers et des pêcheurs", "en": "Fishermen and Fishmongers' Market", "es": "Mercadillo de pescaderos y pescadores"},
	"runes":                {"fr": "Hôtel de vente des runes", "en": "Rune Market", "es": "Mercadillo de runas"},
	"carver":               {"fr": "Hôtel de vente des sculpteurs", "en": "Carvers' Market", "es": "Mercadillo de escultores"},
	"tailor":               {"fr": "Hôtel de vente des tailleurs", "en": "Tailors' Market", "es": "Mercadillo de sastres"},
	"souls":                {"fr": "Hôtel de vente des âmes", "en": "Soul Market", "es": "Mercadillo de almas"},
	"animals":              {"fr": "Hôtel de vente des animaux", "en": "Pet Market", "es": "Mercadillo de animales"},
	"shields":              {"fr": "Hôtel de vente des boucliers", "en": "Shield Market", "es": "Mercadillo de escudos"},
}

// AuctionHouseItemTypeMapping maps auction house codes to ItemType.AnkaId values
// Fill in the AnkaIds for each auction house based on which item types are sold there
var AuctionHouseItemTypeMapping = map[string][]int{
	"resources": {
		15, 35, 36, 46, 47, 48, 53, 54, 55, 56, 57, 58, 59, 65, 68, 103, 104, 105, 106, 107, 109, 110, 111,
	},
	"alchemist": {
		12, 14, 26, 43, 44, 45, 66, 70, 71, 74, 86,
	},
	"jeweller": {
		1, 9,
	},
	"butcher&hunter": {
		63, 64, 69,
	},
	"baker": {
		33, 42,
	},
	"handymen": {
		84, 93, 112,
	},
	"lumberjack": {
		38, 95, 96, 98, 108,
	},
	"shoemaker": {
		10, 11,
	},
	"documents": {
		13, 25, 73, 75, 76,
	},
	"blacksmith": {
		5, 6, 7, 8, 19, 20, 21, 22,
	},
	"miners": {
		39, 40, 50, 51, 88,
	},
	"linkedparchments": {
		87,
	},
	"farmer": {
		34, 52, 60,
	},
	"fishermen&fishmonger": {
		41, 49, 62,
	},
	"runes": {
		78,
	},
	"carver": {
		2, 3, 4,
	},
	"tailor": {
		16, 17, 81,
	},
	"souls": {
		83, 85, 124, 125,
	},
	"animals": {
		18, 77, 91, 90, 97, 113, 116,
	},
	"shields": {
		82,
	},
}

// RuneModel represents a forgemagie rune that can be obtained by breaking items
type RuneModel struct {
	ID         int            `json:"id" gorm:"primaryKey"`
	Code       string         `json:"code" gorm:"size:50;not null"` // e.g., "fo", "pa_fo", "ra_fo", "ga_pa"
	StatTypeID int            `json:"stat_type_id"`                 // References stat_types.id (e.g., strength)
	StatType   *StatTypeModel `json:"stat_type,omitempty" gorm:"foreignKey:StatTypeID;references:ID"`
	Tier       string         `json:"tier" gorm:"size:10;not null"` // "ba", "pa", "ra", or "single" for AP/MP/Range
	Weight     float64        `json:"weight" gorm:"not null"`       // The "poids" value used in calculations
	PowerValue int            `json:"power_value" gorm:"not null"`  // How much stat the rune adds (1 for ba, 3 for pa, 10 for ra)
	ItemAnkaID int            `json:"item_anka_id" gorm:"index"`    // The game's AnkaID for seeding - used to resolve ItemID
	ItemID     *uint          `json:"item_id" gorm:"index"`         // References items.id (the physical rune item)
	Item       *ItemModel     `json:"item,omitempty" gorm:"foreignKey:ItemID;references:ID"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

func (RuneModel) TableName() string {
	return "runes"
}

// RuneTier constants
const (
	RuneTierBa     = "ba"     // Base rune (e.g., Fo)
	RuneTierPa     = "pa"     // Pa rune (e.g., Pa Fo)
	RuneTierRa     = "ra"     // Ra rune (e.g., Ra Fo)
	RuneTierSingle = "single" // Single tier runes (Ga Pa, Ga Pme, Po)
)

// Rune weights (poids) - used in drop chance calculations
const (
	RuneWeightGaPa         = 100.0 // AP rune
	RuneWeightGaPme        = 90.0  // MP rune
	RuneWeightPo           = 51.0  // Range rune
	RuneWeightCriInvoDoRen = 30.0  // Critical/Summon/Reflect damage
	RuneWeightDoSo         = 20.0  // Damage/Heals
	RuneWeightDoPer        = 2.0
	RuneWeightSaProspec    = 3.0 // Wisdom/Prospecting
	RuneWeightRePer        = 4.0
	RuneWeightRe           = 5.0
	RuneWeightDoPi         = 15.0
	RuneWeightStat         = 1.0 // Main stats (Fo, Ine, Cha, Age)
)

// RuneSeedData contains the reference data for runes
// ItemAnkaID will be set to 0 initially and updated when you provide the actual item IDs
var RuneSeedData = []RuneModel{
	// === MAIN STATS (Weight = 1) ===
	// Strength (Force) runes
	{ID: 1, Code: "fo", StatTypeID: 0x76, Tier: RuneTierBa, Weight: RuneWeightStat, PowerValue: 1, ItemAnkaID: 1519},
	{ID: 2, Code: "pa_fo", StatTypeID: 0x76, Tier: RuneTierPa, Weight: RuneWeightStat, PowerValue: 3, ItemAnkaID: 1545},
	{ID: 3, Code: "ra_fo", StatTypeID: 0x76, Tier: RuneTierRa, Weight: RuneWeightStat, PowerValue: 10, ItemAnkaID: 1551},

	// Intelligence runes
	{ID: 4, Code: "ine", StatTypeID: 0x7e, Tier: RuneTierBa, Weight: RuneWeightStat, PowerValue: 1, ItemAnkaID: 1522},
	{ID: 5, Code: "pa_ine", StatTypeID: 0x7e, Tier: RuneTierPa, Weight: RuneWeightStat, PowerValue: 3, ItemAnkaID: 1547},
	{ID: 6, Code: "ra_ine", StatTypeID: 0x7e, Tier: RuneTierRa, Weight: RuneWeightStat, PowerValue: 10, ItemAnkaID: 1553},

	// Chance runes
	{ID: 7, Code: "cha", StatTypeID: 0x7b, Tier: RuneTierBa, Weight: RuneWeightStat, PowerValue: 1, ItemAnkaID: 1525},
	{ID: 8, Code: "pa_cha", StatTypeID: 0x7b, Tier: RuneTierPa, Weight: RuneWeightStat, PowerValue: 3, ItemAnkaID: 1550},
	{ID: 9, Code: "ra_cha", StatTypeID: 0x7b, Tier: RuneTierRa, Weight: RuneWeightStat, PowerValue: 10, ItemAnkaID: 1556},

	// Agility runes
	{ID: 10, Code: "age", StatTypeID: 0x77, Tier: RuneTierBa, Weight: RuneWeightStat, PowerValue: 1, ItemAnkaID: 1524},
	{ID: 11, Code: "pa_age", StatTypeID: 0x77, Tier: RuneTierPa, Weight: RuneWeightStat, PowerValue: 3, ItemAnkaID: 1549},
	{ID: 12, Code: "ra_age", StatTypeID: 0x77, Tier: RuneTierRa, Weight: RuneWeightStat, PowerValue: 10, ItemAnkaID: 1555},

	// === WISDOM / PROSPECTING (Weight = 3) ===
	// Wisdom (Sagesse) runes
	{ID: 13, Code: "sa", StatTypeID: 0x7c, Tier: RuneTierBa, Weight: RuneWeightSaProspec, PowerValue: 1, ItemAnkaID: 1521},
	{ID: 14, Code: "pa_sa", StatTypeID: 0x7c, Tier: RuneTierPa, Weight: RuneWeightSaProspec, PowerValue: 3, ItemAnkaID: 1546},
	{ID: 15, Code: "ra_sa", StatTypeID: 0x7c, Tier: RuneTierRa, Weight: RuneWeightSaProspec, PowerValue: 10, ItemAnkaID: 1552},

	// Prospecting runes
	{ID: 16, Code: "prospe", StatTypeID: 0xb0, Tier: RuneTierBa, Weight: RuneWeightSaProspec, PowerValue: 1, ItemAnkaID: 7451},
	{ID: 17, Code: "pa_prospe", StatTypeID: 0xb0, Tier: RuneTierPa, Weight: RuneWeightSaProspec, PowerValue: 3, ItemAnkaID: 10662},

	// === VITALITY (Weight = 0.25, but uses special thresholds) ===
	// Vitality runes - special weight, uses different thresholds (Pa Vi = 27, Ra Vi = 104)
	{ID: 18, Code: "vi", StatTypeID: 0x7d, Tier: RuneTierBa, Weight: 0.25, PowerValue: 4, ItemAnkaID: 1523},
	{ID: 19, Code: "pa_vi", StatTypeID: 0x7d, Tier: RuneTierPa, Weight: 0.25, PowerValue: 10, ItemAnkaID: 1548},
	{ID: 20, Code: "ra_vi", StatTypeID: 0x7d, Tier: RuneTierRa, Weight: 0.25, PowerValue: 30, ItemAnkaID: 1554},

	// === INITIATIVE / PODS (Weight = 0.1) ===
	// Initiative runes
	{ID: 21, Code: "ini", StatTypeID: 0xae, Tier: RuneTierBa, Weight: 0.1, PowerValue: 10, ItemAnkaID: 7448},
	{ID: 22, Code: "pa_ini", StatTypeID: 0xae, Tier: RuneTierPa, Weight: 0.1, PowerValue: 30, ItemAnkaID: 7449},
	{ID: 23, Code: "ra_ini", StatTypeID: 0xae, Tier: RuneTierRa, Weight: 0.1, PowerValue: 100, ItemAnkaID: 7450},

	// Pods runes
	{ID: 24, Code: "pod", StatTypeID: 0x9e, Tier: RuneTierBa, Weight: 0.1, PowerValue: 10, ItemAnkaID: 7443},
	{ID: 25, Code: "pa_pod", StatTypeID: 0x9e, Tier: RuneTierPa, Weight: 0.1, PowerValue: 30, ItemAnkaID: 7444},
	{ID: 26, Code: "ra_pod", StatTypeID: 0x9e, Tier: RuneTierRa, Weight: 0.1, PowerValue: 100, ItemAnkaID: 7445},

	// === SINGLE TIER RUNES (AP, MP, Range) ===
	// AP rune (Ga Pa) - single tier, level-based chance
	{ID: 27, Code: "ga_pa", StatTypeID: 0x6f, Tier: RuneTierSingle, Weight: RuneWeightGaPa, PowerValue: 1, ItemAnkaID: 1557},

	// MP rune (Ga Pme) - single tier, level-based chance
	{ID: 28, Code: "ga_pme", StatTypeID: 0x80, Tier: RuneTierSingle, Weight: RuneWeightGaPme, PowerValue: 1, ItemAnkaID: 1558},

	// Range rune (Po) - single tier, level-based chance
	{ID: 29, Code: "po", StatTypeID: 0x75, Tier: RuneTierSingle, Weight: RuneWeightPo, PowerValue: 1, ItemAnkaID: 7438},

	// === DAMAGE RUNES (Weight = 20) ===
	// Damage (Do) rune
	{ID: 30, Code: "do", StatTypeID: 0x70, Tier: RuneTierSingle, Weight: RuneWeightDoSo, PowerValue: 1, ItemAnkaID: 7435},

	// Heals (So) rune
	{ID: 31, Code: "so", StatTypeID: 0xb2, Tier: RuneTierSingle, Weight: RuneWeightDoSo, PowerValue: 1, ItemAnkaID: 7434},

	// === CRITICAL / SUMMONS (Weight = 30) ===
	// Critical hit (Cri) rune
	{ID: 32, Code: "cri", StatTypeID: 0x73, Tier: RuneTierSingle, Weight: RuneWeightCriInvoDoRen, PowerValue: 1, ItemAnkaID: 7433},

	// Summons (Invo) rune
	{ID: 33, Code: "invo", StatTypeID: 0xb6, Tier: RuneTierBa, Weight: RuneWeightCriInvoDoRen, PowerValue: 1, ItemAnkaID: 7442},

	// Reflect damage (Do Ren) rune
	{ID: 34, Code: "do_ren", StatTypeID: 0xdc, Tier: RuneTierBa, Weight: RuneWeightCriInvoDoRen, PowerValue: 1, ItemAnkaID: 7437},

	// === FIXED RESISTANCES (Weight = 5) ===
	// Neutral resistance
	{ID: 35, Code: "re_neu", StatTypeID: 0xf4, Tier: RuneTierSingle, Weight: RuneWeightRe, PowerValue: 1, ItemAnkaID: 7456},
	// Earth resistance
	{ID: 36, Code: "re_ter", StatTypeID: 0xf0, Tier: RuneTierSingle, Weight: RuneWeightRe, PowerValue: 1, ItemAnkaID: 7455},
	// Fire resistance
	{ID: 37, Code: "re_feu", StatTypeID: 0xf3, Tier: RuneTierSingle, Weight: RuneWeightRe, PowerValue: 1, ItemAnkaID: 7452},
	// Water resistance
	{ID: 38, Code: "re_eau", StatTypeID: 0xf1, Tier: RuneTierSingle, Weight: RuneWeightRe, PowerValue: 1, ItemAnkaID: 7454},
	// Air resistance
	{ID: 39, Code: "re_air", StatTypeID: 0xf2, Tier: RuneTierSingle, Weight: RuneWeightRe, PowerValue: 1, ItemAnkaID: 7453},

	// === PERCENTAGE RESISTANCES (Weight = 10) ===
	// Neutral resistance %
	{ID: 40, Code: "re_per_neu", StatTypeID: 0xd6, Tier: RuneTierSingle, Weight: RuneWeightRePer, PowerValue: 1, ItemAnkaID: 7460},
	// Earth resistance %
	{ID: 41, Code: "re_per_ter", StatTypeID: 0xd2, Tier: RuneTierSingle, Weight: RuneWeightRePer, PowerValue: 1, ItemAnkaID: 7459},
	// Fire resistance %
	{ID: 42, Code: "re_per_feu", StatTypeID: 0xd5, Tier: RuneTierSingle, Weight: RuneWeightRePer, PowerValue: 1, ItemAnkaID: 7457},
	// Water resistance %
	{ID: 43, Code: "re_per_eau", StatTypeID: 0xd3, Tier: RuneTierSingle, Weight: RuneWeightRePer, PowerValue: 1, ItemAnkaID: 7560},
	// Air resistance %
	{ID: 44, Code: "re_per_air", StatTypeID: 0xd4, Tier: RuneTierSingle, Weight: RuneWeightRePer, PowerValue: 1, ItemAnkaID: 7458},

	// === DAMAGE % (Do Per / Pui) (Weight = 3) ===
	{ID: 45, Code: "do_per", StatTypeID: 0x8a, Tier: RuneTierBa, Weight: RuneWeightDoPer, PowerValue: 1, ItemAnkaID: 7436},
	{ID: 46, Code: "pa_do_per", StatTypeID: 0x8a, Tier: RuneTierPa, Weight: RuneWeightDoPer, PowerValue: 3, ItemAnkaID: 10618},
	{ID: 47, Code: "ra_do_per", StatTypeID: 0x8a, Tier: RuneTierRa, Weight: RuneWeightDoPer, PowerValue: 10, ItemAnkaID: 10619},

	// Rune de chasse (hunting weapon rune)
	{ID: 48, Code: "chasse", StatTypeID: 0x31b, Tier: RuneTierBa, Weight: 5, PowerValue: 1, ItemAnkaID: 10057},

	// Rune Trap Damage
	{ID: 49, Code: "pi", StatTypeID: 0xe1, Tier: RuneTierBa, Weight: RuneWeightDoPi, PowerValue: 1, ItemAnkaID: 7446},
	{ID: 50, Code: "pa_pi", StatTypeID: 0xe1, Tier: RuneTierPa, Weight: RuneWeightDoPi, PowerValue: 3, ItemAnkaID: 10613},

	// Rune Pi Per (AP reduction %)
	{ID: 51, Code: "pi_per", StatTypeID: 0xe2, Tier: RuneTierBa, Weight: RuneWeightDoPer, PowerValue: 1, ItemAnkaID: 7447},
	{ID: 52, Code: "pa_pi_per", StatTypeID: 0xe2, Tier: RuneTierPa, Weight: RuneWeightDoPer, PowerValue: 3, ItemAnkaID: 10615},
	{ID: 53, Code: "ra_pi_per", StatTypeID: 0xe2, Tier: RuneTierRa, Weight: RuneWeightDoPer, PowerValue: 10, ItemAnkaID: 10616},
}

// AP rune drop chances by item level (level -> percentage)
// Max is 66.66%, caps at level 119+
var APRuneDropChanceByLevel = map[int]float64{
	1: 0.0, 2: 0.02, 3: 0.04, 4: 0.08, 5: 0.12, 6: 0.17, 7: 0.23, 8: 0.30, 9: 0.38, 10: 0.47,
	11: 0.57, 12: 0.68, 13: 0.80, 14: 0.93, 15: 1.07, 16: 1.21, 17: 1.37, 18: 1.54, 19: 1.71, 20: 1.90,
	21: 2.09, 22: 2.30, 23: 2.51, 24: 2.73, 25: 2.96, 26: 3.21, 27: 3.46, 28: 3.72, 29: 3.99, 30: 4.27,
	31: 4.56, 32: 4.86, 33: 5.17, 34: 5.48, 35: 5.81, 36: 6.15, 37: 6.49, 38: 6.85, 39: 7.21, 40: 7.59,
	41: 7.97, 42: 8.37, 43: 8.77, 44: 9.18, 45: 9.61, 46: 10.04, 47: 10.48, 48: 10.93, 49: 11.39, 50: 11.86,
	51: 12.34, 52: 12.83, 53: 13.32, 54: 13.83, 55: 14.35, 56: 14.88, 57: 15.41, 58: 15.96, 59: 16.51, 60: 17.08,
	61: 17.65, 62: 18.23, 63: 18.83, 64: 19.43, 65: 20.04, 66: 20.66, 67: 21.29, 68: 21.93, 69: 22.58, 70: 23.24,
	71: 23.91, 72: 24.59, 73: 25.28, 74: 25.97, 75: 26.68, 76: 27.40, 77: 28.12, 78: 28.86, 79: 29.60, 80: 30.36,
	81: 31.12, 82: 31.89, 83: 32.68, 84: 33.47, 85: 34.27, 86: 35.08, 87: 35.90, 88: 36.73, 89: 37.57, 90: 38.42,
	91: 39.28, 92: 40.15, 93: 41.03, 94: 41.91, 95: 42.81, 96: 43.72, 97: 44.63, 98: 45.56, 99: 46.49, 100: 47.43,
	101: 48.39, 102: 49.35, 103: 50.32, 104: 51.30, 105: 52.30, 106: 53.30, 107: 54.31, 108: 55.33, 109: 56.36, 110: 57.40,
	111: 58.44, 112: 59.50, 113: 60.57, 114: 61.65, 115: 62.73, 116: 63.83, 117: 64.93, 118: 66.05, 119: 66.66, 120: 66.66,
}

// MP rune drop chances by item level (level -> percentage)
// Max is 66.66%, caps at level 111+
var MPRuneDropChanceByLevel = map[int]float64{
	1: 0.01, 2: 0.02, 3: 0.05, 4: 0.09, 5: 0.14, 6: 0.19, 7: 0.27, 8: 0.35, 9: 0.44, 10: 0.54,
	11: 0.65, 12: 0.78, 13: 0.91, 14: 1.06, 15: 1.22, 16: 1.39, 17: 1.56, 18: 1.75, 19: 1.95, 20: 2.16,
	21: 2.39, 22: 2.62, 23: 2.86, 24: 3.12, 25: 3.38, 26: 3.66, 27: 3.94, 28: 4.24, 29: 4.55, 30: 4.87,
	31: 5.20, 32: 5.54, 33: 5.89, 34: 6.26, 35: 6.63, 36: 7.01, 37: 7.41, 38: 7.81, 39: 8.23, 40: 8.66,
	41: 9.10, 42: 9.55, 43: 10.01, 44: 10.48, 45: 10.96, 46: 11.45, 47: 11.95, 48: 12.47, 49: 12.99, 50: 13.53,
	51: 14.07, 52: 14.63, 53: 15.20, 54: 15.78, 55: 16.37, 56: 16.97, 57: 17.58, 58: 18.20, 59: 18.84, 60: 19.48,
	61: 20.13, 62: 20.80, 63: 21.48, 64: 22.16, 65: 22.86, 66: 23.57, 67: 24.29, 68: 25.02, 69: 25.76, 70: 26.51,
	71: 27.28, 72: 28.05, 73: 28.84, 74: 29.63, 75: 30.44, 76: 31.25, 77: 32.08, 78: 32.92, 79: 33.77, 80: 34.63,
	81: 35.50, 82: 36.38, 83: 37.28, 84: 38.18, 85: 39.10, 86: 40.02, 87: 40.96, 88: 41.90, 89: 42.86, 90: 43.83,
	91: 44.81, 92: 45.80, 93: 46.80, 94: 47.81, 95: 48.84, 96: 49.87, 97: 50.91, 98: 51.97, 99: 53.03, 100: 54.11,
	101: 55.20, 102: 56.30, 103: 57.41, 104: 58.53, 105: 59.66, 106: 60.80, 107: 61.95, 108: 63.12, 109: 64.29, 110: 65.47,
	111: 66.66, 112: 66.66,
}

// MaxRuneDropChance is the maximum drop chance for any rune (2/3)
const MaxRuneDropChance = 66.66

// RuneThreshold defines the stat value thresholds for guaranteed rune drops
type RuneThreshold struct {
	StatCode       string // e.g., "strength", "vitality"
	BaThreshold    int    // Minimum jet for 100% Ba rune
	PaThreshold    int    // Minimum jet for 100% Pa rune
	RaThreshold    int    // Minimum jet for 100% Ra rune
	IntermediatePa int    // Intermediate value for Pa calculation
	IntermediateRa int    // Intermediate value for Ra calculation
}

// RuneThresholds contains the threshold data for different stat types
// Formula: 100% of rune X = [intermediate_threshold / (2/3)] / 0.9
var RuneThresholds = map[string]RuneThreshold{
	// Main stats (Fo, Ine, Cha, Age) - Weight = 1
	"strength":     {StatCode: "strength", BaThreshold: 2, PaThreshold: 9, RaThreshold: 34, IntermediatePa: 5, IntermediateRa: 20},
	"intelligence": {StatCode: "intelligence", BaThreshold: 2, PaThreshold: 9, RaThreshold: 34, IntermediatePa: 5, IntermediateRa: 20},
	"chance":       {StatCode: "chance", BaThreshold: 2, PaThreshold: 9, RaThreshold: 34, IntermediatePa: 5, IntermediateRa: 20},
	"agility":      {StatCode: "agility", BaThreshold: 2, PaThreshold: 9, RaThreshold: 34, IntermediatePa: 5, IntermediateRa: 20},

	// Wisdom/Prospecting - Weight = 3
	"wisdom":      {StatCode: "wisdom", BaThreshold: 2, PaThreshold: 9, RaThreshold: 34, IntermediatePa: 5, IntermediateRa: 20},
	"prospecting": {StatCode: "prospecting", BaThreshold: 2, PaThreshold: 9, RaThreshold: 34, IntermediatePa: 5, IntermediateRa: 20},

	// Vitality - special thresholds
	"vitality": {StatCode: "vitality", BaThreshold: 5, PaThreshold: 27, RaThreshold: 104, IntermediatePa: 16, IntermediateRa: 62},

	// Initiative/Pods - thresholds * 10
	"initiative": {StatCode: "initiative", BaThreshold: 17, PaThreshold: 84, RaThreshold: 334, IntermediatePa: 50, IntermediateRa: 200},
	"pods":       {StatCode: "pods", BaThreshold: 17, PaThreshold: 84, RaThreshold: 334, IntermediatePa: 50, IntermediateRa: 200},

	// Damage % (Pui)
	"damage_percent": {StatCode: "damage_percent", BaThreshold: 2, PaThreshold: 9, RaThreshold: 34, IntermediatePa: 5, IntermediateRa: 20},
}

// GetAPRuneDropChance returns the drop chance for AP rune based on item level
func GetAPRuneDropChance(level int) float64 {
	if level <= 0 {
		return 0
	}
	if level >= 119 {
		return MaxRuneDropChance
	}
	if chance, exists := APRuneDropChanceByLevel[level]; exists {
		return chance
	}
	return 0
}

// GetMPRuneDropChance returns the drop chance for MP rune based on item level
func GetMPRuneDropChance(level int) float64 {
	if level <= 0 {
		return 0
	}
	if level >= 111 {
		return MaxRuneDropChance
	}
	if chance, exists := MPRuneDropChanceByLevel[level]; exists {
		return chance
	}
	return 0
}

// UserModel represents a user in the system
type UserModel struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	Username       *string    `json:"username" gorm:"size:100;uniqueIndex"`           // Optional, can be set later in settings
	EmailHash      string     `json:"email_hash" gorm:"size:64;uniqueIndex;not null"` // SHA-256 hash of email for lookups
	EncryptedEmail string     `json:"encrypted_email" gorm:"size:255"`                // AES-GCM encrypted email for when we need to use it
	DiscordID      *string    `json:"discord_id" gorm:"size:20;uniqueIndex"`          // Discord user ID for OAuth linking
	IsAdmin        bool       `json:"is_admin" gorm:"default:false"`
	IsDeleted      bool       `json:"is_deleted" gorm:"default:false"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at"`
}

func (UserModel) TableName() string {
	return "users"
}

// SessionModel represents a user session
type SessionModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Token     string    `json:"token" gorm:"size:255;uniqueIndex;not null"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	User      UserModel `json:"user" gorm:"foreignKey:UserID"`
}

func (SessionModel) TableName() string {
	return "sessions"
}

// MagicLinkModel represents a magic link for passwordless authentication
type MagicLinkModel struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	Token     string     `json:"token" gorm:"size:255;uniqueIndex;not null"`
	Email     string     `json:"email" gorm:"size:255;not null;index"`
	UserID    *uint      `json:"user_id" gorm:"index"` // Nullable for new users
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	Used      bool       `json:"used" gorm:"default:false"`
	CreatedAt time.Time  `json:"created_at"`
	User      *UserModel `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (MagicLinkModel) TableName() string {
	return "magic_links"
}

// PasskeyCredentialModel represents a WebAuthn passkey credential
type PasskeyCredentialModel struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	CredentialID   []byte    `json:"credential_id" gorm:"type:bytea;uniqueIndex;not null"` // Unique credential identifier
	PublicKey      []byte    `json:"public_key" gorm:"type:bytea;not null"`                // COSE-encoded public key
	AAGUID         []byte    `json:"aa_guid" gorm:"type:bytea"`                            // Authenticator attestation GUID
	SignCount      uint32    `json:"sign_count" gorm:"default:0"`                          // Signature counter for clone detection
	BackupEligible bool      `json:"backup_eligible" gorm:"default:false"`                 // Whether credential can be backed up
	BackupState    bool      `json:"backup_state" gorm:"default:false"`                    // Whether credential is currently backed up
	Name           string    `json:"name" gorm:"size:100;not null"`                        // User-provided name (e.g., "MacBook Pro")
	CreatedAt      time.Time `json:"created_at"`
	User           UserModel `json:"user" gorm:"foreignKey:UserID"`
}

func (PasskeyCredentialModel) TableName() string {
	return "passkey_credentials"
}

// WebAuthnChallengeModel stores temporary challenges for WebAuthn ceremonies
type WebAuthnChallengeModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SessionID string    `json:"session_id" gorm:"size:255;uniqueIndex;not null"` // Temporary session ID for the ceremony
	Challenge []byte    `json:"challenge" gorm:"type:bytea;not null"`            // The random challenge bytes
	UserID    *uint     `json:"user_id" gorm:"index"`                            // User ID (for registration, nullable for login)
	Type      string    `json:"type" gorm:"size:20;not null"`                    // "registration" or "login"
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`                      // 5 minute expiry
	CreatedAt time.Time `json:"created_at"`
}

func (WebAuthnChallengeModel) TableName() string {
	return "webauthn_challenges"
}

// OAuthStateModel stores temporary OAuth state for CSRF protection
type OAuthStateModel struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	State       string    `json:"state" gorm:"size:64;uniqueIndex;not null"` // Random state parameter
	Provider    string    `json:"provider" gorm:"size:20;not null"`          // "discord", "google", etc.
	RedirectURL string    `json:"redirect_url" gorm:"size:255"`              // Where to redirect after auth
	ExpiresAt   time.Time `json:"expires_at" gorm:"not null"`                // 10 minute expiry
	CreatedAt   time.Time `json:"created_at"`
}

func (OAuthStateModel) TableName() string {
	return "oauth_states"
}
