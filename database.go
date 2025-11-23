package gofusretrodb

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseService handles database operations
type DatabaseService struct {
	db *gorm.DB
}

// NewDatabaseService creates a new database service
func NewDatabaseService(dsn string) (*DatabaseService, error) {
	// Configure GORM logger to suppress "record not found" errors
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Warn, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Enable color
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
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
		&ItemStatModel{},
		&StatTypeModel{},
		&StatTypeTranslationModel{},
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
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_item_stats_item_id ON item_effects(item_id)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_item_stats_type ON item_effects(effect_type)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_item_conditions_item_id ON item_conditions(item_id)")
	ds.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_item_set_translations_unique ON item_set_translations(item_set_id, language)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_recipes_item_id ON recipes(item_id)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_ingredients_recipe_id ON ingredients(recipe_id)")
	ds.db.Exec("CREATE INDEX IF NOT EXISTS idx_ingredients_item_id ON ingredients(item_id)")

	return nil
}

// ClearAllData removes all existing item data from the database
func (ds *DatabaseService) ClearAllData() error {
	ds.db.Exec("DELETE FROM item_stats")
	ds.db.Exec("DELETE FROM item_stat_types")
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
				GfxID:        item.GfxID,
				Price:        item.Price,
				Weight:       item.Weight,
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
			} else {
				translation := item.Translations[0]
				itemMap[item.ID] = &ItemModel{
					AnkaId:       item.ID,
					TypeAnkaId:   item.TypeID,
					Level:        item.Level,
					GfxID:        item.GfxID,
					Price:        item.Price,
					Weight:       item.Weight,
					Requirements: item.Requirements,
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
		item := map[string]interface{}{
			"id":           result.ID,
			"anka_id":      result.AnkaId,
			"type_anka_id": result.TypeAnkaId,
			"type_name":    result.TypeName,
			"level":        result.Level,
			"requirements": result.Requirements,
			"name":         result.Translation.Name,
			"name_upper":   result.Translation.NameUpper,
			"description":  result.Translation.Description,
			"language":     language,
		}

		items = append(items, item)
	}

	return items, nil
}

func (ds *DatabaseService) GetItemsSearch(search string, language string, typeAnkaIDs []int) ([]ItemModel, error) {
	var items []ItemModel
	var err error

	trimmedSearch := strings.TrimSpace(search)

	// Handle empty search - return empty result or limit results
	if trimmedSearch == "" {
		query := ds.db.Preload("Translations", "language = ?", language).
			Preload("Recipe.Ingredients.Item.Translations").
			Joins("JOIN item_translations it ON items.id = it.item_id").
			Limit(50)

		// Add type filter if provided
		if len(typeAnkaIDs) > 0 {
			query = query.Where("items.type_anka_id IN ?", typeAnkaIDs)
		}

		err = query.Find(&items).Error
		return items, err
	}

	query := ds.db.Preload("Translations", "language = ?", language).
		Preload("Type.Translations", "language = ?", language).
		Preload("Recipe.Ingredients.Item.Translations", "language = ?", language).
		Joins("JOIN item_translations it ON items.id = it.item_id").
		Where("it.language = ? AND LOWER(it.name) LIKE LOWER(?)", language, "%"+trimmedSearch+"%")

	// Add type filter if provided
	if len(typeAnkaIDs) > 0 {
		query = query.Where("items.type_anka_id IN ?", typeAnkaIDs)
	}

	err = query.Find(&items).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search items: %v", err)
	}

	return items, nil
}

// GetItemsSearchPaginated retrieves items with pagination and priority sorting at the database level
func (ds *DatabaseService) GetItemsSearchPaginated(searchValue, language string, typeAnkaIDs []int, limit, offset int) (items []ItemModel, totalCount int, err error) {
	trimmedSearch := strings.TrimSpace(searchValue)

	// Build the base query
	baseQuery := ds.db.Table("items").
		Joins("JOIN item_translations it ON items.id = it.item_id").
		Where("it.language = ?", language)

	// Add search filter if provided
	if trimmedSearch != "" {
		baseQuery = baseQuery.Where("LOWER(it.name) LIKE LOWER(?)", "%"+trimmedSearch+"%")
	}

	// Add type filter if provided
	if len(typeAnkaIDs) > 0 {
		baseQuery = baseQuery.Where("items.type_anka_id IN ?", typeAnkaIDs)
	}

	// Get total count
	var count int64
	countQuery := baseQuery.Count(&count)
	if countQuery.Error != nil {
		return nil, 0, fmt.Errorf("failed to count items: %v", countQuery.Error)
	}
	totalCount = int(count)

	// Build the main query with priority sorting
	query := ds.db.
		Preload("Translations", "language = ?", language).
		Preload("Type.Translations", "language = ?", language).
		Joins("JOIN item_translations it ON items.id = it.item_id").
		Where("it.language = ?", language)

	// Add search filter if provided
	if trimmedSearch != "" {
		query = query.Where("LOWER(it.name) LIKE LOWER(?)", "%"+trimmedSearch+"%")

		// Priority sorting: items starting with search term come first
		query = query.Order(fmt.Sprintf(
			"CASE WHEN LOWER(it.name) LIKE LOWER('%s%%') THEN 0 ELSE 1 END",
			strings.ReplaceAll(trimmedSearch, "'", "''"), // Escape single quotes for SQL safety
		))
	}

	// Add type filter if provided
	if len(typeAnkaIDs) > 0 {
		query = query.Where("items.type_anka_id IN ?", typeAnkaIDs)
	}

	// Add secondary sorting by name and apply pagination
	query = query.Order("it.name ASC").
		Limit(limit).
		Offset(offset).
		Find(&items)

	if query.Error != nil {
		return nil, 0, fmt.Errorf("failed to search items: %v", query.Error)
	}

	// Recursively load full recipe trees for all items (max depth 10)
	for i := range items {
		if err := ds.LoadRecipeRecursive(&items[i], language, 3, 0); err != nil {
			return nil, 0, fmt.Errorf("failed to load recipe tree for item %d: %v", items[i].ID, err)
		}
	}

	return items, totalCount, nil
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

// GetItemTypesByIDs retrieves item types by their AnkaIDs with translations for a specific language
func (ds *DatabaseService) GetItemTypesByIDs(ankaIDs []int, language string) ([]ItemTypeModel, error) {
	var itemTypes []ItemTypeModel

	err := ds.db.
		Preload("Translations", "language = ?", language).
		Where("anka_id IN ?", ankaIDs).
		Find(&itemTypes).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get item types: %v", err)
	}

	return itemTypes, nil
}

// DiagnoseRecipes checks if recipes exist and tests preloading
func (ds *DatabaseService) DiagnoseRecipes(language string) error {
	// Check total recipes count
	var recipeCount int64
	if err := ds.db.Model(&RecipeModel{}).Count(&recipeCount).Error; err != nil {
		return fmt.Errorf("failed to count recipes: %v", err)
	}
	fmt.Printf("Total recipes in database: %d\n", recipeCount)

	// Check total ingredients count
	var ingredientCount int64
	if err := ds.db.Model(&IngredientModel{}).Count(&ingredientCount).Error; err != nil {
		return fmt.Errorf("failed to count ingredients: %v", err)
	}
	fmt.Printf("Total ingredients in database: %d\n", ingredientCount)

	// Find first 5 items that have recipes
	var items []ItemModel
	err := ds.db.Preload("Translations", "language = ?", language).
		Preload("Recipe").
		Preload("Recipe.Ingredients").
		Preload("Recipe.Ingredients.Item").
		Preload("Recipe.Ingredients.Item.Translations", "language = ?", language).
		Joins("INNER JOIN recipes ON recipes.item_id = items.id").
		Limit(5).
		Find(&items).Error

	if err != nil {
		return fmt.Errorf("failed to query items with recipes: %v", err)
	}

	fmt.Printf("\nFound %d items with recipes (showing first 5):\n", len(items))
	for _, item := range items {
		if len(item.Translations) > 0 {
			fmt.Printf("- Item AnkaID=%d, Name=%s", item.AnkaId, item.Translations[0].Name)
			if item.Recipe != nil {
				fmt.Printf(" -> Recipe with %d ingredients\n", len(item.Recipe.Ingredients))
				for _, ing := range item.Recipe.Ingredients {
					if len(ing.Item.Translations) > 0 {
						fmt.Printf("    * %dx %s\n", ing.Quantity, ing.Item.Translations[0].Name)
					}
				}
			} else {
				fmt.Printf(" -> Recipe is nil (NOT LOADED)\n")
			}
		}
	}

	return nil
}

// LoadRecipeRecursive recursively loads the recipe and all ingredient recipes to build a complete crafting tree
func (ds *DatabaseService) LoadRecipeRecursive(item *ItemModel, language string, maxDepth int, currentDepth int) error {
	// Prevent infinite recursion
	if currentDepth >= maxDepth {
		return nil
	}

	// Load the recipe for this item if it exists
	var recipe RecipeModel
	err := ds.db.Preload("Ingredients").
		Where("item_id = ?", item.ID).
		First(&recipe).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No recipe found - this is a base material
			return nil
		}
		return fmt.Errorf("failed to load recipe: %v", err)
	}

	// Attach recipe to item
	item.Recipe = &recipe

	// For each ingredient, load the item details and recursively load its recipe
	for i := range recipe.Ingredients {
		ingredient := &recipe.Ingredients[i]

		// Load the ingredient item with translations
		var ingredientItem ItemModel
		err := ds.db.Preload("Translations", "language = ?", language).
			Preload("Type.Translations", "language = ?", language).
			Where("id = ?", ingredient.ItemID).
			First(&ingredientItem).Error

		if err != nil {
			return fmt.Errorf("failed to load ingredient item %d: %v", ingredient.ItemID, err)
		}

		// Recursively load the recipe for this ingredient item
		if err := ds.LoadRecipeRecursive(&ingredientItem, language, maxDepth, currentDepth+1); err != nil {
			return err
		}

		// Attach the fully loaded item to the ingredient
		ingredient.Item = ingredientItem
	}

	return nil
}

// GetItemWithFullRecipeTree retrieves an item by AnkaId with its complete recipe tree recursively loaded
func (ds *DatabaseService) GetItemWithFullRecipeTree(ankaId int, language string, maxDepth int) (*ItemModel, error) {
	if maxDepth <= 0 {
		maxDepth = 10 // Default max depth to prevent infinite loops
	}

	// Load the base item with translations
	var item ItemModel
	err := ds.db.Preload("Translations", "language = ?", language).
		Preload("Type.Translations", "language = ?", language).
		Where("anka_id = ?", ankaId).
		First(&item).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load item: %v", err)
	}

	// Recursively load all recipes and ingredients
	if err := ds.LoadRecipeRecursive(&item, language, maxDepth, 0); err != nil {
		return nil, err
	}

	return &item, nil
}

// GetItemsWithRecipeTree retrieves multiple items with their complete recipe trees
func (ds *DatabaseService) GetItemsWithRecipeTree(ankaIds []int, language string, maxDepth int) ([]ItemModel, error) {
	if maxDepth <= 0 {
		maxDepth = 10
	}

	var items []ItemModel
	err := ds.db.Preload("Translations", "language = ?", language).
		Preload("Type.Translations", "language = ?", language).
		Where("anka_id IN ?", ankaIds).
		Find(&items).Error

	if err != nil {
		return nil, fmt.Errorf("failed to load items: %v", err)
	}

	// Recursively load recipes for each item
	for i := range items {
		if err := ds.LoadRecipeRecursive(&items[i], language, maxDepth, 0); err != nil {
			return nil, err
		}
	}

	return items, nil
}

func (ds *DatabaseService) SaveItemStats(itemStatsMap map[int][]ItemStat) error {
	if len(itemStatsMap) == 0 {
		fmt.Println("No item stats to save")
		return nil
	}

	fmt.Printf("Saving item stats for %d items...\n", len(itemStatsMap))

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

	// Clear existing item stats
	if err := tx.Exec("DELETE FROM item_stats").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to clear item stats: %v", err)
	}

	totalStats := 0
	skippedItems := 0
	skippedStats := 0

	// Iterate through each item's stats
	for itemAnkaId, stats := range itemStatsMap {
		// Find the PostgreSQL primary key for this item
		itemPK, err := ds.GetItemPrimaryKeyByAnkaId(itemAnkaId)
		if err != nil {
			// Skip items that don't exist in the database
			skippedItems++
			continue
		}

		// Insert each stat for this item
		for _, stat := range stats {
			// Verify that the stat type exists (the hex code should match a StatType ID)
			var statTypeExists int64
			if err := tx.Model(&StatTypeModel{}).Where("id = ?", stat.StatTypeId).Count(&statTypeExists).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to check stat type existence: %v", err)
			}

			if statTypeExists == 0 {
				// Skip stats with unknown stat type IDs
				skippedStats++
				continue
			}

			itemStatModel := ItemStatModel{
				ItemID:     int(itemPK),     // Use the PostgreSQL primary key
				StatTypeID: stat.StatTypeId, // Use the hex code as the stat type ID
				MinValue:   &stat.MinValue,
				MaxValue:   &stat.MaxValue,
				Formula:    stat.Formula,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			if err := tx.Create(&itemStatModel).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to insert item stat for item %d, stat type 0x%x: %v", itemAnkaId, stat.StatTypeId, err)
			}

			totalStats++
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("Successfully saved %d item stats (skipped %d items not in DB, %d unknown stat types)\n", totalStats, skippedItems, skippedStats)
	return nil
}

func (ds *DatabaseService) SeedStatTypes() error {
	fmt.Println("Seeding stat types...")

	// Check if stat types already exist
	var count int64
	if err := ds.db.Model(&StatTypeModel{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count stat types: %v", err)
	}

	if count > 0 {
		fmt.Printf("Found %d existing stat types, skipping seed\n", count)
		return nil
	}

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

	// Insert stat types with their hexadecimal IDs
	for i, statType := range StatTypeSeedData {
		statTypeModel := StatTypeModel{
			ID:           statType.ID, // Use the hexadecimal ID directly
			Code:         statType.Code,
			DisplayOrder: i + 1,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := tx.Create(&statTypeModel).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert stat type %s (0x%x): %v", statType.Code, statType.ID, err)
		}

		// Insert translations for this stat type
		if translations, exists := StatTypeTranslations[statType.Code]; exists {
			for language, name := range translations {
				translation := StatTypeTranslationModel{
					StatTypeID: statType.ID,
					Language:   language,
					Name:       name,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}

				if err := tx.Create(&translation).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to insert translation for stat type %s (%s): %v", statType.Code, language, err)
				}
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("Successfully seeded %d stat types with translations\n", len(StatTypeSeedData))
	return nil
}
