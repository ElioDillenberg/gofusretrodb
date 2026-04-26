package gofusretrodb

import (
	"fmt"
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

// SetUserServer sets the user's selected game server
func (ds *DatabaseService) SetUserServer(userID, serverID uint) error {
	return ds.db.Model(&UserModel{}).Where("id = ?", userID).Update("server_id", serverID).Error
}

// GetUserServer returns the user's selected server (nil if none set)
func (ds *DatabaseService) GetUserServer(userID uint) (*ServerModel, error) {
	var user UserModel
	if err := ds.db.Preload("Server").First(&user, userID).Error; err != nil {
		return nil, err
	}
	return user.Server, nil // nil if no server selected
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




