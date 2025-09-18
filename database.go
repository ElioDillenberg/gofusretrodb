package gofusretrodb

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DatabaseService handles database operations
type DatabaseService struct {
	db *gorm.DB
}

// NewDatabaseService creates a new database service
func NewDatabaseService(dsn string) (*DatabaseService, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	service := &DatabaseService{db: db}

	// Initialize schema
	if err := service.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}

	return service, nil
}

// Close closes the database connection
func (ds *DatabaseService) Close() error {
	sqlDB, err := ds.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB returns the underlying GORM database instance
func (ds *DatabaseService) GetDB() *gorm.DB {
	return ds.db
}

// initSchema creates the database tables
func (ds *DatabaseService) initSchema() error {
	// Auto-migrate the schema (creates tables if they don't exist)
	err := ds.db.AutoMigrate(
		&ItemTypeModel{},
		&ItemTypeTranslationModel{},
		&ItemModel{},
		&ItemTranslationModel{},
		&ItemEffectModel{},
		&ItemConditionModel{},
		&ItemSetModel{},
		&ItemSetTranslationModel{},
		&RecipeModel{},
		&IngredientModel{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto-migrate schema: %v", err)
	}

	// Create unique constraints and indexes after auto-migration
	ds.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_item_type_translations_unique ON item_type_translations(item_type_id, language)")
	ds.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_item_translations_unique ON item_translations(item_id, language)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_item_translations_language ON item_translations(language)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_item_translations_name ON item_translations(name)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_items_type_anka_id ON items(type_anka_id)")
	// Create index on anka_id, but allow multiple zeros for existing records
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_items_anka_id ON items(anka_id)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_item_effects_item_id ON item_effects(item_id)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_item_effects_type ON item_effects(effect_type)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_item_conditions_item_id ON item_conditions(item_id)")
	ds.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_item_set_translations_unique ON item_set_translations(item_set_id, language)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_recipes_item_id ON recipes(item_id)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_ingredients_recipe_id ON ingredients(recipe_id)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_ingredients_item_id ON ingredients(item_id)")

	return nil
}

// ClearAllData removes all existing item data from the database
func (ds *DatabaseService) ClearAllData() error {
	ds.db.Exec("DELETE FROM item_effects")
	ds.db.Exec("DELETE FROM item_conditions")
	ds.db.Exec("DELETE FROM item_translations")
	ds.db.Exec("DELETE FROM ingredients")
	ds.db.Exec("DELETE FROM recipes")
	ds.db.Exec("DELETE FROM items")
	return nil
}

// SaveItems saves parsed items to the database
func (ds *DatabaseService) SaveItems(allItems map[string][]Item) error {
	// Begin transaction
	tx := ds.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Clear existing data before inserting new data
	if err := ds.ClearAllData(); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear existing data: %v", err)
	}

	// Step 1: Use French as master language to create items based on AnkaId
	// Then add translations from other languages
	itemMap := make(map[int]*ItemModel)                             // AnkaId -> ItemModel
	translationMap := make(map[int]map[string]ItemTranslationModel) // AnkaId -> language -> translation

	// First pass: Process French items to create the base items
	if frenchItems, exists := allItems["fr"]; exists {
		for _, item := range frenchItems {
			if len(item.Translations) == 0 || item.ID == 0 {
				continue
			}

			translation := item.Translations[0]
			itemMap[item.ID] = &ItemModel{
				AnkaId:       item.ID,     // Store original DOFUS item ID
				TypeAnkaId:   item.TypeID, // Store original DOFUS type ID (references ItemType.AnkaId)
				Level:        item.Level,
				Requirements: item.Requirements,
				Stats:        item.Stats,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			// Initialize translation map for this item
			translationMap[item.ID] = make(map[string]ItemTranslationModel)

			// Add French translation
			translationMap[item.ID]["fr"] = ItemTranslationModel{
				Language:    "fr",
				Name:        translation.Name,
				NameUpper:   translation.NameUpper,
				Description: translation.Description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
		}
	}

	// Second pass: Add translations from other languages (en, es) matching by AnkaId
	for language, items := range allItems {
		if language == "fr" {
			continue // Already processed French
		}

		for _, item := range items {
			if len(item.Translations) == 0 || item.ID == 0 {
				continue
			}

			// Check if we have a French item with this AnkaId
			if existingItem, exists := itemMap[item.ID]; exists {
				// Add translation for this language
				translation := item.Translations[0]
				translationMap[item.ID][language] = ItemTranslationModel{
					Language:    language,
					Name:        translation.Name,
					NameUpper:   translation.NameUpper,
					Description: translation.Description,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}

				// Update item data with more complete information if available
				if item.Level > existingItem.Level {
					existingItem.Level = item.Level
				}
				if item.Requirements != "" && existingItem.Requirements == "" {
					existingItem.Requirements = item.Requirements
				}
				if item.Stats != "{}" && item.Stats != "" && (existingItem.Stats == "{}" || existingItem.Stats == "") {
					existingItem.Stats = item.Stats
				}
			} else {
				// This item exists in non-French language but not in French
				// Create it anyway but log this situation
				fmt.Printf("Warning: Item AnkaId %d exists in %s but not in French\n", item.ID, language)

				translation := item.Translations[0]
				itemMap[item.ID] = &ItemModel{
					AnkaId:       item.ID,
					TypeAnkaId:   item.TypeID,
					Level:        item.Level,
					Requirements: item.Requirements,
					Stats:        item.Stats,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				translationMap[item.ID] = make(map[string]ItemTranslationModel)
				translationMap[item.ID][language] = ItemTranslationModel{
					Language:    language,
					Name:        translation.Name,
					NameUpper:   translation.NameUpper,
					Description: translation.Description,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
			}
		}
	}

	// Insert items and their translations
	itemsInserted := 0
	for ankaId, item := range itemMap {
		// Create item
		if err := tx.Create(item).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert item with AnkaId %d: %v", ankaId, err)
		}

		// Debug: log first few AnkaIds
		if itemsInserted < 5 {
			fmt.Printf("Debug: Inserted item with AnkaId=%d, PostgresID=%d\n", item.AnkaId, item.ID)
		}
		itemsInserted++

		// Insert translations
		for _, translation := range translationMap[ankaId] {
			translation.ItemID = item.ID
			if err := tx.Create(&translation).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to insert translation for AnkaId %d: %v", ankaId, err)
			}
		}
	}

	fmt.Printf("Successfully inserted %d items with translations\n", itemsInserted)
	return tx.Commit().Error
}

// GetItemsByLanguage retrieves items for a specific language
func (ds *DatabaseService) GetItemsByLanguage(language string) ([]map[string]interface{}, error) {
	var results []struct {
		ItemModel
		Translation ItemTranslationModel `gorm:"embedded;embeddedPrefix:translation_"`
		TypeName    string               `gorm:"column:type_name"`
	}

	err := ds.db.Table("items i").
		Select("i.*, it.language as translation_language, it.name as translation_name, it.name_upper as translation_name_upper, it.description as translation_description, it.created_at as translation_created_at, it.updated_at as translation_updated_at, it.id as translation_id, it.item_id as translation_item_id, tt.name as type_name").
		Joins("JOIN item_translations it ON i.id = it.item_id").
		Joins("LEFT JOIN item_types itype ON i.type_anka_id = itype.anka_id").
		Joins("LEFT JOIN item_type_translations tt ON itype.id = tt.item_type_id AND tt.language = it.language").
		Where("it.language = ?", language).
		Order("i.type_anka_id, it.name").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query items: %v", err)
	}

	var items []map[string]interface{}
	for _, result := range results {
		// Parse stats JSON
		var statsMap map[string]interface{}
		if result.Stats != "" && result.Stats != "{}" {
			json.Unmarshal([]byte(result.Stats), &statsMap)
		}

		item := map[string]interface{}{
			"id":           result.ID,
			"anka_id":      result.AnkaId,
			"type_anka_id": result.TypeAnkaId,
			"type_name":    result.TypeName,
			"level":        result.Level,
			"requirements": result.Requirements,
			"stats":        statsMap,
			"name":         result.Translation.Name,
			"name_upper":   result.Translation.NameUpper,
			"description":  result.Translation.Description,
			"language":     language,
		}

		items = append(items, item)
	}

	return items, nil
}

func (ds *DatabaseService) GetItemsSearch(search string, language string) ([]ItemModel, error) {
	var items []ItemModel
	var err error

	trimmedSearch := strings.TrimSpace(search)

	// Handle empty search - return empty result or limit results
	if trimmedSearch == "" {
		err = ds.db.Preload("Translations", "language = ?", language).
			Preload("Ingredients").
			Preload("Recipe").
			Joins("JOIN item_translations it ON items.id = it.item_id").
			Limit(50).
			Find(&items).Error

		return items, err
	}

	err = ds.db.Preload("Translations", "language = ?", language).
		Preload("Ingredients").
		Preload("Recipe").
		Joins("JOIN item_translations it ON items.id = it.item_id").
		Where("it.language = ? AND LOWER(it.name) LIKE LOWER(?)", language, "%"+trimmedSearch+"%").
		Find(&items).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search items: %v", err)
	}

	return items, nil
}

// GetItemPrimaryKeyByAnkaId finds the PostgreSQL primary key for an item by its original DOFUS ID
func (ds *DatabaseService) GetItemPrimaryKeyByAnkaId(ankaId int) (uint, error) {
	var item ItemModel
	err := ds.db.Select("id").Where("anka_id = ?", ankaId).First(&item).Error
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

// SaveRecipes saves recipes to the database using AnkaId mapping
func (ds *DatabaseService) SaveRecipes(recipes []Recipe) error {
	if len(recipes) == 0 {
		return nil
	}

	fmt.Printf("Saving %d recipes to database...\n", len(recipes))

	// Begin transaction
	tx := ds.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Clear existing recipes
	if err := tx.Exec("DELETE FROM ingredients").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear ingredients: %v", err)
	}
	if err := tx.Exec("DELETE FROM recipes").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear recipes: %v", err)
	}

	// Insert recipes
	successfulRecipes := 0
	for _, recipe := range recipes {
		// Find the PostgreSQL primary key for the recipe item
		itemPK, err := ds.GetItemPrimaryKeyByAnkaId(recipe.ItemID)
		if err != nil {
			// Skip recipes for items that don't exist
			continue
		}

		recipeModel := RecipeModel{
			ItemID:    itemPK, // Use PostgreSQL primary key
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := tx.Create(&recipeModel).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert recipe: %v", err)
		}

		// Insert ingredients
		for _, ingredient := range recipe.Ingredients {
			// Find the PostgreSQL primary key for the ingredient item
			ingredientPK, err := ds.GetItemPrimaryKeyByAnkaId(ingredient.ItemID)
			if err != nil {
				// Skip ingredients for items that don't exist
				continue
			}

			ingredientModel := IngredientModel{
				RecipeID:  recipeModel.ID,
				ItemID:    ingredientPK, // Use PostgreSQL primary key
				Quantity:  ingredient.Quantity,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := tx.Create(&ingredientModel).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to insert ingredient: %v", err)
			}
		}
		successfulRecipes++
	}

	fmt.Printf("Successfully saved %d recipes (skipped %d recipes for missing items)\n", successfulRecipes, len(recipes)-successfulRecipes)
	return tx.Commit().Error
}

// SaveItemTypes saves dynamically extracted item types to the database
func (ds *DatabaseService) SaveItemTypes(allItemTypes map[string][]ItemTypeDefinition) error {
	if len(allItemTypes) == 0 {
		return nil
	}

	fmt.Printf("Saving item types from %d languages to database...\n", len(allItemTypes))

	// Check if we already have item types
	var existingTypeCount int64
	if err := ds.db.Model(&ItemTypeModel{}).Count(&existingTypeCount).Error; err != nil {
		return fmt.Errorf("failed to count existing item types: %v", err)
	}

	if existingTypeCount > 0 {
		fmt.Printf("Found %d existing item types. Using upsert strategy instead of clearing...\n", existingTypeCount)
		return ds.upsertItemTypes(allItemTypes)
	}

	// Begin transaction for fresh insertion
	tx := ds.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Collect all unique item type IDs across languages
	allTypeIDs := make(map[int]int) // ID -> category
	for _, itemTypes := range allItemTypes {
		for _, itemType := range itemTypes {
			allTypeIDs[itemType.ID] = itemType.Category
		}
	}

	// Insert item types (one record per type ID)
	for typeID, category := range allTypeIDs {
		itemType := ItemTypeModel{
			AnkaId:  typeID,                         // Store original SWF type ID
			KeyName: fmt.Sprintf("type_%d", typeID), // Dynamic key name
		}
		if err := tx.Create(&itemType).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert item type %d: %v", typeID, err)
		}

		// Keep category for potential future use
		_ = category
	}

	// Insert all translations - need to find the database ID for each AnkaId
	for language, itemTypes := range allItemTypes {
		for _, itemType := range itemTypes {
			// Find the database primary key for this AnkaId
			var dbItemType ItemTypeModel
			if err := tx.Where("anka_id = ?", itemType.ID).First(&dbItemType).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to find item type with AnkaId %d: %v", itemType.ID, err)
			}

			translation := ItemTypeTranslationModel{
				ItemTypeID: dbItemType.ID, // Use database primary key
				Language:   language,
				Name:       itemType.Name,
			}
			if err := tx.Create(&translation).Error; err != nil {
				// Skip duplicates but continue
				if !strings.Contains(err.Error(), "duplicate key") && !strings.Contains(err.Error(), "violates unique constraint") {
					tx.Rollback()
					return fmt.Errorf("failed to insert item type translation: %v", err)
				}
			}
		}
	}

	fmt.Printf("Successfully saved %d item types with translations in %d languages\n", len(allTypeIDs), len(allItemTypes))
	return tx.Commit().Error
}

// upsertItemTypes updates existing item types or inserts new ones
func (ds *DatabaseService) upsertItemTypes(allItemTypes map[string][]ItemTypeDefinition) error {
	fmt.Println("Upserting item types and translations...")

	// Collect all unique item type IDs across languages
	allTypeIDs := make(map[int]int) // ID -> category
	for _, itemTypes := range allItemTypes {
		for _, itemType := range itemTypes {
			allTypeIDs[itemType.ID] = itemType.Category
		}
	}

	// Upsert item types
	for typeID, category := range allTypeIDs {
		itemType := ItemTypeModel{
			AnkaId:  typeID,
			KeyName: fmt.Sprintf("type_%d", typeID),
		}

		// Use GORM's FirstOrCreate to handle existing records by AnkaId
		if err := ds.db.FirstOrCreate(&itemType, "anka_id = ?", typeID).Error; err != nil {
			return fmt.Errorf("failed to upsert item type %d: %v", typeID, err)
		}

		_ = category // Keep for potential future use
	}

	// Upsert translations
	for language, itemTypes := range allItemTypes {
		for _, itemType := range itemTypes {
			// Find the database primary key for this AnkaId
			var dbItemType ItemTypeModel
			if err := ds.db.Where("anka_id = ?", itemType.ID).First(&dbItemType).Error; err != nil {
				return fmt.Errorf("failed to find item type with AnkaId %d: %v", itemType.ID, err)
			}

			translation := ItemTypeTranslationModel{
				ItemTypeID: dbItemType.ID, // Use database primary key
				Language:   language,
				Name:       itemType.Name,
			}

			// Use FirstOrCreate for translations
			if err := ds.db.FirstOrCreate(&translation, "item_type_id = ? AND language = ?", dbItemType.ID, language).Error; err != nil {
				return fmt.Errorf("failed to upsert item type translation: %v", err)
			}
		}
	}

	fmt.Printf("Successfully upserted %d item types with translations in %d languages\n", len(allTypeIDs), len(allItemTypes))
	return nil
}

// GetRecipeByItemID retrieves the recipe for a specific item by AnkaId
func (ds *DatabaseService) GetRecipeByItemID(ankaId int, language string) (*RecipeModel, error) {
	// First get the PostgreSQL primary key for the item
	itemPK, err := ds.GetItemPrimaryKeyByAnkaId(ankaId)
	if err != nil {
		return nil, fmt.Errorf("item not found: %v", err)
	}

	var recipe RecipeModel
	err = ds.db.Preload("Item").
		Preload("Ingredients").
		Preload("Ingredients.Item").
		Preload("Ingredients.Item.Translations", "language = ?", language).
		Where("item_id = ?", itemPK).
		First(&recipe).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No recipe found (item is not craftable)
		}
		return nil, fmt.Errorf("failed to get recipe: %v", err)
	}

	return &recipe, nil
}

// GetItemByIDAndLanguage retrieves a specific item by AnkaId with translation for a specific language
func (ds *DatabaseService) GetItemByIDAndLanguage(ankaId int, language string) (map[string]interface{}, error) {
	// Query for item with translation directly - this handles duplicate anka_ids correctly
	var result struct {
		ItemTranslationModel
		TypeName         string `gorm:"column:type_name"`
		ItemAnkaId       int    `gorm:"column:item_anka_id"`
		ItemLevel        int    `gorm:"column:item_level"`
		ItemRequirements string `gorm:"column:item_requirements"`
		ItemStats        string `gorm:"column:item_stats"`
		TypeAnkaId       int    `gorm:"column:type_anka_id"`
	}

	err := ds.db.Table("item_translations it").
		Select("it.*, tt.name as type_name, i.anka_id as item_anka_id, i.level as item_level, i.requirements as item_requirements, i.stats as item_stats, i.type_anka_id as type_anka_id").
		Joins("JOIN items i ON it.item_id = i.id").
		Joins("LEFT JOIN item_types itype ON i.type_anka_id = itype.anka_id").
		Joins("LEFT JOIN item_type_translations tt ON itype.id = tt.item_type_id AND tt.language = it.language").
		Where("i.anka_id = ? AND it.language = ?", ankaId, language).
		First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No translation found for this language
		}
		return nil, fmt.Errorf("failed to query item %d for language %s: %v", ankaId, language, err)
	}

	// Parse stats JSON
	var statsMap map[string]interface{}
	if result.ItemStats != "" && result.ItemStats != "{}" {
		json.Unmarshal([]byte(result.ItemStats), &statsMap)
	}

	// Build result with single language
	item := map[string]interface{}{
		"id":           result.ItemID,
		"anka_id":      result.ItemAnkaId,
		"type_anka_id": result.TypeAnkaId,
		"level":        result.ItemLevel,
		"requirements": result.ItemRequirements,
		"stats":        statsMap,
		"name":         result.Name,
		"name_upper":   result.NameUpper,
		"description":  result.Description,
		"type_name":    result.TypeName,
		"language":     language,
	}

	return item, nil
}
