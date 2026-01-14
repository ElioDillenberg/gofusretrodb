package gofusretrodb

import (
	"time"
)

// WorkshopListModel represents a user's workshop list
type WorkshopListModel struct {
	ID          uint                    `json:"id" gorm:"primaryKey"`
	UserID      uint                    `json:"user_id" gorm:"not null;index"`
	Name        string                  `json:"name" gorm:"size:255;not null"`
	Description string                  `json:"description" gorm:"type:text"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
	User        UserModel               `json:"user" gorm:"foreignKey:UserID"`
	Items       []WorkshopListItemModel `json:"items" gorm:"foreignKey:WorkshopListID"`
}

func (WorkshopListModel) TableName() string {
	return "workshop_lists"
}

// WorkshopListItemModel represents an item in a workshop list
type WorkshopListItemModel struct {
	ID             uint              `json:"id" gorm:"primaryKey"`
	WorkshopListID uint              `json:"workshop_list_id" gorm:"not null;index"`
	ItemID         uint              `json:"item_id" gorm:"not null;index"`
	Quantity       int               `json:"quantity" gorm:"default:1"`
	Notes          string            `json:"notes" gorm:"type:text"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	WorkshopList   WorkshopListModel `json:"workshop_list" gorm:"foreignKey:WorkshopListID"`
	Item           ItemModel         `json:"item" gorm:"foreignKey:ItemID"`
}

func (WorkshopListItemModel) TableName() string {
	return "workshop_list_items"
}
