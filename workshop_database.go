package gofusretrodb

import (
	"fmt"
	"sort"
	"time"
)

// ==================== Workshop List Management ====================

// CreateWorkshopList creates a new workshop list for a user
func (ds *DatabaseService) CreateWorkshopList(userID uint, name, description string) (*WorkshopListModel, error) {
	list := &WorkshopListModel{
		UserID:      userID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := ds.db.Create(list).Error; err != nil {
		return nil, fmt.Errorf("failed to create workshop list: %v", err)
	}

	return list, nil
}

// GetWorkshopListsByUser retrieves all workshop lists for a user
func (ds *DatabaseService) GetWorkshopListsByUser(userID uint) ([]WorkshopListModel, error) {
	var lists []WorkshopListModel
	err := ds.db.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&lists).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get workshop lists: %v", err)
	}
	return lists, nil
}

// GetWorkshopListByID retrieves a workshop list by ID with its items
func (ds *DatabaseService) GetWorkshopListByID(listID uint, language string) (*WorkshopListModel, error) {
	var list WorkshopListModel
	err := ds.db.
		Preload("Items.Item.Translations", "language = ?", language).
		Preload("Items.Item.Type.Translations", "language = ?", language).
		Preload("Items.Item.Stats.StatType.Translations", "language = ?", language).
		Preload("Items.Item.Stats.StatType.Runes.Item.Translations", "language = ?", language).
		First(&list, listID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get workshop list: %v", err)
	}

	// Load recipes for each item
	for i := range list.Items {
		if err := ds.LoadRecipeRecursive(&list.Items[i].Item, language, 3, 0); err != nil {
			// Don't fail if recipe loading fails, just continue
			continue
		}
	}

	return &list, nil
}

// UpdateWorkshopList updates a workshop list's name and description
func (ds *DatabaseService) UpdateWorkshopList(listID uint, name, description string) error {
	return ds.db.Model(&WorkshopListModel{}).
		Where("id = ?", listID).
		Updates(map[string]interface{}{
			"name":        name,
			"description": description,
			"updated_at":  time.Now(),
		}).Error
}

// DeleteWorkshopList deletes a workshop list and all its items
func (ds *DatabaseService) DeleteWorkshopList(listID uint) error {
	// Delete all items in the list first
	if err := ds.db.Where("workshop_list_id = ?", listID).Delete(&WorkshopListItemModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete workshop list items: %v", err)
	}

	// Delete the list itself
	if err := ds.db.Delete(&WorkshopListModel{}, listID).Error; err != nil {
		return fmt.Errorf("failed to delete workshop list: %v", err)
	}

	return nil
}

// IsWorkshopListOwner checks if a user owns a workshop list
func (ds *DatabaseService) IsWorkshopListOwner(listID, userID uint) (bool, error) {
	var count int64
	err := ds.db.Model(&WorkshopListModel{}).
		Where("id = ? AND user_id = ?", listID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ==================== Workshop List Items ====================

// AddItemToWorkshopList adds an item to a workshop list
func (ds *DatabaseService) AddItemToWorkshopList(listID, itemID uint, quantity int, notes string) (*WorkshopListItemModel, error) {
	if quantity < 1 {
		quantity = 1
	}

	// Check if item already exists in the list
	var existingItem WorkshopListItemModel
	err := ds.db.Where("workshop_list_id = ? AND item_id = ?", listID, itemID).First(&existingItem).Error
	if err == nil {
		// Item already exists, update quantity
		existingItem.Quantity += quantity
		existingItem.UpdatedAt = time.Now()
		if notes != "" {
			existingItem.Notes = notes
		}
		if err := ds.db.Save(&existingItem).Error; err != nil {
			return nil, fmt.Errorf("failed to update workshop list item: %v", err)
		}
		return &existingItem, nil
	}

	// Create new item
	item := &WorkshopListItemModel{
		WorkshopListID: listID,
		ItemID:         itemID,
		Quantity:       quantity,
		Notes:          notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := ds.db.Create(item).Error; err != nil {
		return nil, fmt.Errorf("failed to add item to workshop list: %v", err)
	}

	// Update the list's updated_at
	ds.db.Model(&WorkshopListModel{}).Where("id = ?", listID).Update("updated_at", time.Now())

	return item, nil
}

// UpdateWorkshopListItem updates an item's quantity and notes
func (ds *DatabaseService) UpdateWorkshopListItem(itemID uint, quantity int, notes string) error {
	if quantity < 1 {
		quantity = 1
	}

	return ds.db.Model(&WorkshopListItemModel{}).
		Where("id = ?", itemID).
		Updates(map[string]interface{}{
			"quantity":   quantity,
			"notes":      notes,
			"updated_at": time.Now(),
		}).Error
}

// RemoveItemFromWorkshopList removes an item from a workshop list
func (ds *DatabaseService) RemoveItemFromWorkshopList(itemID uint) error {
	// Get the list ID before deleting
	var item WorkshopListItemModel
	if err := ds.db.First(&item, itemID).Error; err != nil {
		return fmt.Errorf("workshop list item not found: %v", err)
	}

	listID := item.WorkshopListID

	if err := ds.db.Delete(&WorkshopListItemModel{}, itemID).Error; err != nil {
		return fmt.Errorf("failed to remove item from workshop list: %v", err)
	}

	// Update the list's updated_at
	ds.db.Model(&WorkshopListModel{}).Where("id = ?", listID).Update("updated_at", time.Now())

	return nil
}

// GetWorkshopListItemCount returns the number of items in a workshop list
func (ds *DatabaseService) GetWorkshopListItemCount(listID uint) (int64, error) {
	var count int64
	err := ds.db.Model(&WorkshopListItemModel{}).Where("workshop_list_id = ?", listID).Count(&count).Error
	return count, err
}

// ==================== Resource Calculations ====================

// ResourceRequirement represents a resource needed for crafting
type ResourceRequirement struct {
	ItemID                   uint
	ItemAnkaID               int
	TypeAnkaID               int
	GfxID                    int
	Name                     string
	TotalNeeded              int
	AuctionHouseID           *uint
	AuctionHouseName         string
	AuctionHouseDisplayOrder int
}

// GetAllResourcesForList calculates all unique resources needed for a workshop list
func (ds *DatabaseService) GetAllResourcesForList(listID uint, language string) ([]ResourceRequirement, error) {
	list, err := ds.GetWorkshopListByID(listID, language)
	if err != nil {
		return nil, err
	}

	// Aggregate all resources from all items
	resourceMap := make(map[uint]*ResourceRequirement)

	for _, listItem := range list.Items {
		if listItem.Item.Recipe == nil {
			continue
		}

		// Calculate resources for this item * quantity
		ds.aggregateRecipeResources(listItem.Item.Recipe, listItem.Quantity, resourceMap)
	}

	// Convert map to slice
	var resources []ResourceRequirement
	for _, req := range resourceMap {
		resources = append(resources, *req)
	}

	return resources, nil
}

// GetResourcesGroupedByAuctionHouse returns resources grouped by auction house
// The key is the auction house name (empty string for items without an auction house)
// Resources within each group are sorted alphabetically by name
// Auction houses are sorted by their display order
func (ds *DatabaseService) GetResourcesGroupedByAuctionHouse(listID uint, language string) (map[string][]ResourceRequirement, []string, error) {
	resources, err := ds.GetAllResourcesForList(listID, language)
	if err != nil {
		return nil, nil, err
	}

	grouped := make(map[string][]ResourceRequirement)
	// Track auction house display order for sorting
	ahDisplayOrder := make(map[string]int)

	for _, res := range resources {
		key := res.AuctionHouseName
		grouped[key] = append(grouped[key], res)
		// Store display order (only need to store once per auction house)
		if _, exists := ahDisplayOrder[key]; !exists {
			ahDisplayOrder[key] = res.AuctionHouseDisplayOrder
		}
	}

	// Build order slice from auction house names (excluding empty string)
	var order []string
	for key := range grouped {
		if key != "" {
			order = append(order, key)
		}
	}

	// Sort auction houses by display order
	sort.Slice(order, func(i, j int) bool {
		return ahDisplayOrder[order[i]] < ahDisplayOrder[order[j]]
	})

	// Add empty string key at the end if there are items without auction house
	if len(grouped[""]) > 0 {
		order = append(order, "")
	}

	// Sort resources within each group alphabetically by name
	for key := range grouped {
		sort.Slice(grouped[key], func(i, j int) bool {
			return grouped[key][i].Name < grouped[key][j].Name
		})
	}

	return grouped, order, nil
}

// aggregateRecipeResources recursively adds up all resources needed (including craftable items)
func (ds *DatabaseService) aggregateRecipeResources(recipe *RecipeModel, multiplier int, resources map[uint]*ResourceRequirement) {
	if recipe == nil {
		return
	}

	for _, ingredient := range recipe.Ingredients {
		needed := ingredient.Quantity * multiplier

		// Always add the ingredient to resources (even if it's craftable)
		if existing, ok := resources[ingredient.ItemID]; ok {
			existing.TotalNeeded += needed
		} else {
			name := ""
			if len(ingredient.Item.Translations) > 0 {
				name = ingredient.Item.Translations[0].Name
			}

			// Get auction house info from preloaded item type
			var ahID *uint
			var ahName string
			var ahDisplayOrder int
			if ingredient.Item.Type != nil && ingredient.Item.Type.AuctionHouse != nil {
				ahID = &ingredient.Item.Type.AuctionHouse.ID
				ahDisplayOrder = ingredient.Item.Type.AuctionHouse.DisplayOrder
				if len(ingredient.Item.Type.AuctionHouse.Translations) > 0 {
					ahName = ingredient.Item.Type.AuctionHouse.Translations[0].Name
				}
			}

			resources[ingredient.ItemID] = &ResourceRequirement{
				ItemID:                   ingredient.ItemID,
				ItemAnkaID:               ingredient.Item.AnkaId,
				TypeAnkaID:               ingredient.Item.TypeAnkaId,
				GfxID:                    ingredient.Item.GfxID,
				Name:                     name,
				TotalNeeded:              needed,
				AuctionHouseID:           ahID,
				AuctionHouseName:         ahName,
				AuctionHouseDisplayOrder: ahDisplayOrder,
			}
		}

		// If ingredient has a recipe, also recurse into it to get sub-ingredients
		if ingredient.Item.Recipe != nil {
			ds.aggregateRecipeResources(ingredient.Item.Recipe, needed, resources)
		}
	}
}

// ItemHasRecipe checks if an item has a recipe (is craftable)
func (ds *DatabaseService) ItemHasRecipe(itemID uint) (bool, error) {
	var count int64
	err := ds.db.Model(&RecipeModel{}).Where("item_id = ?", itemID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsItemInWorkshopList checks if an item is already in a workshop list
func (ds *DatabaseService) IsItemInWorkshopList(listID, itemID uint) (bool, error) {
	var count int64
	err := ds.db.Model(&WorkshopListItemModel{}).
		Where("workshop_list_id = ? AND item_id = ?", listID, itemID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// RemoveItemFromWorkshopListByItemID removes an item from a list using list_id and item_id
func (ds *DatabaseService) RemoveItemFromWorkshopListByItemID(listID, itemID uint) error {
	if err := ds.db.Where("workshop_list_id = ? AND item_id = ?", listID, itemID).
		Delete(&WorkshopListItemModel{}).Error; err != nil {
		return fmt.Errorf("failed to remove item from workshop list: %v", err)
	}

	// Update the list's updated_at
	ds.db.Model(&WorkshopListModel{}).Where("id = ?", listID).Update("updated_at", time.Now())

	return nil
}

// ==================== Rune Calculations ====================

// RuneRequirement represents a unique rune that can be obtained from breaking items in the list
type RuneRequirement struct {
	RuneID     int
	ItemAnkaID int    // The rune item's AnkaID (for price store)
	TypeAnkaID int    // Type AnkaID for image path
	GfxID      int    // GfxID for image path
	Name       string // Translated rune name
	Code       string // Rune code (e.g., "fo", "pa_fo")
	Tier       string // "ba", "pa", "ra", or "single"
	Weight     float64
}

// GetUniqueRunesForList returns all unique runes that can be obtained from breaking items in a workshop list
func (ds *DatabaseService) GetUniqueRunesForList(listID uint, language string) ([]RuneRequirement, error) {
	list, err := ds.GetWorkshopListByID(listID, language)
	if err != nil {
		return nil, err
	}

	// Map to track unique runes by their ID
	runeMap := make(map[int]*RuneRequirement)

	for _, listItem := range list.Items {
		// Get runes from the item's stats
		for _, stat := range listItem.Item.Stats {
			if len(stat.StatType.Runes) == 0 {
				continue
			}

			for _, rune := range stat.StatType.Runes {
				if _, exists := runeMap[rune.ID]; exists {
					continue
				}

				// Get the rune name from its linked Item
				name := ""
				typeAnkaID := 0
				gfxID := 0
				itemAnkaID := 0

				if rune.Item != nil {
					itemAnkaID = rune.Item.AnkaId
					typeAnkaID = rune.Item.TypeAnkaId
					gfxID = rune.Item.GfxID

					for _, t := range rune.Item.Translations {
						if t.Language == language && t.Name != "" {
							name = t.Name
							break
						}
					}
					// Fallback to first translation
					if name == "" && len(rune.Item.Translations) > 0 {
						name = rune.Item.Translations[0].Name
					}
				}

				if name == "" {
					name = rune.Code
				}

				runeMap[rune.ID] = &RuneRequirement{
					RuneID:     rune.ID,
					ItemAnkaID: itemAnkaID,
					TypeAnkaID: typeAnkaID,
					GfxID:      gfxID,
					Name:       name,
					Code:       rune.Code,
					Tier:       rune.Tier,
					Weight:     rune.Weight,
				}
			}
		}
	}

	// Convert map to slice
	var runes []RuneRequirement
	for _, req := range runeMap {
		runes = append(runes, *req)
	}

	// Sort by weight descending (higher weight = more valuable runes first)
	sort.Slice(runes, func(i, j int) bool {
		return runes[i].Weight > runes[j].Weight
	})

	return runes, nil
}
