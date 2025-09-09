# GOFUS Retro DB

A shared Go library containing GORM models and database operations for DOFUS Retro data. :D

## Features

- GORM models for all DOFUS entities (items, recipes, translations, etc.)
- Database service with CRUD operations
- Multi-language support for item names and descriptions
- Recipe/crafting system support
- PostgreSQL database backend

## Usage

```go
import "github.com/eliodillenberg/gofusretrodb"

// Create database service
db, err := gofusretrodb.NewDatabaseService("postgres://...")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Get items by language
items, err := db.GetItemsByLanguage("fr")
if err != nil {
    log.Fatal(err)
}

// Get specific item with recipe
item, err := db.GetItemByIDAndLanguage(472, "en")
if err != nil {
    log.Fatal(err)
}

recipe, err := db.GetRecipeByItemID(472, "en")
if err != nil {
    log.Fatal(err)
}
```

## Models

- `ItemModel` - Game items with stats, requirements, etc.
- `ItemTranslationModel` - Multi-language names and descriptions
- `ItemTypeModel` - Item categories (weapon, armor, etc.)
- `RecipeModel` - Crafting recipes
- `IngredientModel` - Recipe ingredients
- `ItemSetModel` - Equipment sets

## Database Schema

The library automatically creates and manages the database schema with proper indexes and foreign key constraints.
