package gofusretrodb

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ==================== Server Management ====================

// SeedServers inserts or updates the predefined server list
func (ds *DatabaseService) SeedServers() error {
	fmt.Println("Seeding servers...")

	for _, server := range ServerSeedData {
		existing := ServerModel{}
		err := ds.db.Where("code = ?", server.Code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			server.CreatedAt = time.Now()
			if err := ds.db.Create(&server).Error; err != nil {
				return fmt.Errorf("failed to create server %s: %v", server.Code, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check server %s: %v", server.Code, err)
		} else {
			// Update name and active status
			ds.db.Model(&existing).Updates(map[string]interface{}{
				"name":      server.Name,
				"is_active": server.IsActive,
			})
		}
	}

	fmt.Printf("Successfully seeded %d servers\n", len(ServerSeedData))
	return nil
}

// GetActiveServers returns all active game servers
func (ds *DatabaseService) GetActiveServers() ([]ServerModel, error) {
	var servers []ServerModel
	err := ds.db.Where("is_active = ?", true).Order("id ASC").Find(&servers).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get active servers: %v", err)
	}
	return servers, nil
}

// GetServerByID returns a server by ID
func (ds *DatabaseService) GetServerByID(id uint) (*ServerModel, error) {
	var server ServerModel
	if err := ds.db.First(&server, id).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

// SetUserServer sets the user's selected game server in user_preferences
func (ds *DatabaseService) SetUserServer(userID, serverID uint) error {
	prefs, err := ds.GetOrCreateUserPreferences(userID)
	if err != nil {
		return err
	}
	return ds.db.Model(prefs).Update("server_id", serverID).Error
}

// GetUserServer returns the user's selected server (nil if none set)
func (ds *DatabaseService) GetUserServer(userID uint) (*ServerModel, error) {
	prefs, err := ds.GetOrCreateUserPreferences(userID)
	if err != nil {
		return nil, err
	}
	if prefs.ServerID == nil {
		return nil, nil
	}
	return ds.GetServerByID(*prefs.ServerID)
}

// GetOrCreateUserPreferences returns the user_preferences row for the given user,
// creating a default row (browser mode, no server) if it doesn't exist yet.
func (ds *DatabaseService) GetOrCreateUserPreferences(userID uint) (*UserPreferencesModel, error) {
	var prefs UserPreferencesModel
	result := ds.db.Where(UserPreferencesModel{UserID: userID}).
		Attrs(UserPreferencesModel{PriceSaveMode: "browser"}).
		FirstOrCreate(&prefs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get/create user preferences for user %d: %v", userID, result.Error)
	}
	return &prefs, nil
}

// GetUserWithPreferences fetches a user and their preferences in two queries.
// Returns the user and a guaranteed non-nil preferences (created if missing).
func (ds *DatabaseService) GetUserWithPreferences(userID uint) (*UserModel, *UserPreferencesModel, error) {
	user, err := ds.GetUserByID(userID)
	if err != nil {
		return nil, nil, err
	}
	prefs, err := ds.GetOrCreateUserPreferences(userID)
	if err != nil {
		return nil, nil, err
	}
	return user, prefs, nil
}

// SetPriceSaveMode updates the price_save_mode preference for a user.
// Only "browser" and "cloud" are valid values.
func (ds *DatabaseService) SetPriceSaveMode(userID uint, mode string) error {
	if mode != "browser" && mode != "cloud" {
		return fmt.Errorf("invalid price_save_mode %q: must be 'browser' or 'cloud'", mode)
	}
	prefs, err := ds.GetOrCreateUserPreferences(userID)
	if err != nil {
		return err
	}
	return ds.db.Model(prefs).Update("price_save_mode", mode).Error
}

// MigrateServerIDToPreferences copies non-null users.server_id values into
// user_preferences rows. Safe to call multiple times (skips users that already
// have a preferences row with a server set). Called once at startup.
func (ds *DatabaseService) MigrateServerIDToPreferences() error {
	// Find all users with a non-null server_id
	var users []UserModel
	if err := ds.db.Where("server_id IS NOT NULL").Find(&users).Error; err != nil {
		return fmt.Errorf("MigrateServerIDToPreferences: failed to query users: %v", err)
	}

	migrated := 0
	for _, u := range users {
		if u.ServerID == nil {
			continue
		}
		// Only set if not already set in preferences
		prefs, err := ds.GetOrCreateUserPreferences(u.ID)
		if err != nil {
			log.Printf("MigrateServerIDToPreferences: skipping user %d: %v", u.ID, err)
			continue
		}
		if prefs.ServerID != nil {
			// Already has a server in preferences — leave it alone
			continue
		}
		if err := ds.db.Model(prefs).Update("server_id", *u.ServerID).Error; err != nil {
			log.Printf("MigrateServerIDToPreferences: failed to set server for user %d: %v", u.ID, err)
			continue
		}
		migrated++
	}

	if migrated > 0 {
		log.Printf("MigrateServerIDToPreferences: migrated %d user(s)", migrated)
	}
	return nil
}

// ==================== Price Management ====================

// UpsertUserItemPrices upserts current prices for a user on a server.
// Returns the map of item IDs whose price actually changed (old price differs from new).
func (ds *DatabaseService) UpsertUserItemPrices(userID, serverID uint, prices map[uint]int) (changedItems map[uint]int, err error) {
	changedItems = make(map[uint]int)

	if len(prices) == 0 {
		return changedItems, nil
	}

	// Collect item IDs
	itemIDs := make([]uint, 0, len(prices))
	for id := range prices {
		itemIDs = append(itemIDs, id)
	}

	// Fetch existing prices to detect changes
	var existing []UserItemPriceModel
	if err := ds.db.Where("user_id = ? AND server_id = ? AND item_id IN ?", userID, serverID, itemIDs).
		Find(&existing).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing prices: %v", err)
	}

	existingMap := make(map[uint]int, len(existing))
	for _, e := range existing {
		existingMap[e.ItemID] = e.Price
	}

	// Determine which items actually changed
	for itemID, newPrice := range prices {
		if oldPrice, exists := existingMap[itemID]; !exists || oldPrice != newPrice {
			changedItems[itemID] = newPrice
		}
	}

	if len(changedItems) == 0 {
		return changedItems, nil
	}

	// Upsert only changed items
	now := time.Now()
	records := make([]UserItemPriceModel, 0, len(changedItems))
	for itemID, price := range changedItems {
		records = append(records, UserItemPriceModel{
			UserID:    userID,
			ServerID:  serverID,
			ItemID:    itemID,
			Price:     price,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Use ON CONFLICT to upsert
	if err := ds.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "server_id"}, {Name: "item_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"price", "updated_at"}),
	}).Create(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to upsert prices: %v", err)
	}

	return changedItems, nil
}

// SaveUserPrices saves prices for a user. Always upserts current prices.
// If the user is pro/admin, also appends changed prices to the history log.
func (ds *DatabaseService) SaveUserPrices(role string, userID, serverID uint, prices map[uint]int) error {
	changedItems, err := ds.UpsertUserItemPrices(userID, serverID, prices)
	if err != nil {
		return err
	}

	// Only append to history for pro/admin users, and only for items that changed
	if (role == RolePro || role == RoleAdmin) && len(changedItems) > 0 {
		if err := ds.InsertPriceHistory(userID, serverID, changedItems); err != nil {
			// Log but don't fail the whole operation
			fmt.Printf("Warning: failed to insert price history: %v\n", err)
		}
	}

	return nil
}

// InsertPriceHistory appends price entries to the history log (pro/admin only)
func (ds *DatabaseService) InsertPriceHistory(userID, serverID uint, prices map[uint]int) error {
	if len(prices) == 0 {
		return nil
	}

	now := time.Now()
	records := make([]ItemPriceHistoryModel, 0, len(prices))
	for itemID, price := range prices {
		records = append(records, ItemPriceHistoryModel{
			UserID:    userID,
			ServerID:  serverID,
			ItemID:    itemID,
			Price:     price,
			CreatedAt: now,
		})
	}

	if err := ds.db.Create(&records).Error; err != nil {
		return fmt.Errorf("failed to insert price history: %v", err)
	}

	return nil
}

// GetLatestUserItemPrices returns the current prices for a user on a server for the given item IDs.
// If itemIDs is nil or empty, returns all prices for the user on the server.
func (ds *DatabaseService) GetLatestUserItemPrices(userID, serverID uint, itemIDs []uint) ([]UserItemPriceModel, error) {
	query := ds.db.Where("user_id = ? AND server_id = ?", userID, serverID)
	if len(itemIDs) > 0 {
		query = query.Where("item_id IN ?", itemIDs)
	}

	var prices []UserItemPriceModel
	if err := query.Find(&prices).Error; err != nil {
		return nil, fmt.Errorf("failed to get user item prices: %v", err)
	}
	return prices, nil
}

// GetItemPriceHistory returns the price history for a specific item (pro/admin feature)
func (ds *DatabaseService) GetItemPriceHistory(userID, serverID, itemID uint, limit int) ([]ItemPriceHistoryModel, error) {
	var history []ItemPriceHistoryModel
	query := ds.db.Where("user_id = ? AND server_id = ? AND item_id = ?", userID, serverID, itemID).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to get price history: %v", err)
	}
	return history, nil
}




