package gofusretrodb

import (
	"time"
)

// Item represents a DOFUS item (from SWF parser)
type Item struct {
	ID           int               `json:"id"`
	TypeID       int               `json:"type_id"`
	Level        int               `json:"level"`
	Requirements string            `json:"requirements"`
	Stats        string            `json:"stats"`
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
	Stats        string    `json:"stats" gorm:"type:text"`
	Price        int       `json:"price" gorm:"default:0"`
	Weight       int       `json:"weight" gorm:"default:0"`
	GfxID        int       `json:"gfx_id" gorm:"default:0"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	// Note: Type relationship will be handled manually via TypeAnkaId
	Translations []ItemTranslationModel `json:"translations" gorm:"foreignKey:ItemID"`
	Effects      []ItemEffectModel      `json:"effects" gorm:"foreignKey:ItemID"`
	Conditions   []ItemConditionModel   `json:"conditions" gorm:"foreignKey:ItemID"`
	Recipe       *RecipeModel           `json:"recipe,omitempty" gorm:"foreignKey:ItemID"`
	Ingredients  []IngredientModel      `json:"ingredients,omitempty" gorm:"foreignKey:ItemID"`
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

// ItemEffectModel represents item effects/stats
type ItemEffectModel struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ItemID     uint      `json:"item_id" gorm:"not null"`
	EffectType int       `json:"effect_type" gorm:"not null"`
	MinValue   int       `json:"min_value" gorm:"default:0"`
	MaxValue   int       `json:"max_value" gorm:"default:0"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Item       ItemModel `json:"item" gorm:"foreignKey:ItemID"`
}

func (ItemEffectModel) TableName() string {
	return "item_effects"
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
