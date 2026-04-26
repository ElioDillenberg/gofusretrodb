package gofusretrodb

import (
	"time"
)

// ServerModel represents a Dofus Retro game server
type ServerModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:50;not null"`         // Display name (e.g. "Boune 2")
	Code      string    `json:"code" gorm:"size:50;uniqueIndex;not null"` // URL-safe code (e.g. "boune-2")
	IsActive  bool      `json:"is_active" gorm:"default:true;not null"`
	CreatedAt time.Time `json:"created_at"`
}

func (ServerModel) TableName() string {
	return "servers"
}

// ServerSeedData contains the initial list of Dofus Retro servers
var ServerSeedData = []ServerModel{
	{ID: 1, Name: "Boune", Code: "boune", IsActive: true},
	{ID: 2, Name: "Boune 2", Code: "boune-2", IsActive: true},
	{ID: 3, Name: "Boune 3", Code: "boune-3", IsActive: true},
	{ID: 4, Name: "Boune 4", Code: "boune-4", IsActive: true},
	{ID: 5, Name: "Allisteria", Code: "allisteria", IsActive: true},
	{ID: 6, Name: "Allisteria 2", Code: "allisteria-2", IsActive: true},
	{ID: 7, Name: "Allisteria 3", Code: "allisteria-3", IsActive: true},
	{ID: 8, Name: "Fallanster", Code: "fallanster", IsActive: true},
	{ID: 9, Name: "Fallanster 2", Code: "fallanster-2", IsActive: true},
}

// UserItemPriceModel stores the current price a user has set for an item on a server.
// This is upserted on every price change (only the latest value is kept).
type UserItemPriceModel struct {
	ID        uint        `json:"id" gorm:"primaryKey"`
	UserID    uint        `json:"user_id" gorm:"not null;uniqueIndex:idx_user_server_item"`
	ServerID  uint        `json:"server_id" gorm:"not null;uniqueIndex:idx_user_server_item"`
	ItemID    uint        `json:"item_id" gorm:"not null;uniqueIndex:idx_user_server_item"`
	Price     int         `json:"price" gorm:"not null;default:0"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	User      UserModel   `json:"user" gorm:"foreignKey:UserID"`
	Server    ServerModel `json:"server" gorm:"foreignKey:ServerID"`
	Item      ItemModel   `json:"item" gorm:"foreignKey:ItemID;references:AnkaId"`
}

func (UserItemPriceModel) TableName() string {
	return "user_item_prices"
}

// ItemPriceHistoryModel stores an append-only log of price changes.
// Only created for pro/admin users. Each row is immutable.
type ItemPriceHistoryModel struct {
	ID        uint        `json:"id" gorm:"primaryKey"`
	UserID    uint        `json:"user_id" gorm:"not null;index:idx_price_history_lookup"`
	ServerID  uint        `json:"server_id" gorm:"not null;index:idx_price_history_lookup"`
	ItemID    uint        `json:"item_id" gorm:"not null;index:idx_price_history_lookup"`
	Price     int         `json:"price" gorm:"not null;default:0"`
	CreatedAt time.Time   `json:"created_at" gorm:"not null;index:idx_price_history_lookup"`
	User      UserModel   `json:"user" gorm:"foreignKey:UserID"`
	Server    ServerModel `json:"server" gorm:"foreignKey:ServerID"`
	Item      ItemModel   `json:"item" gorm:"foreignKey:ItemID;references:AnkaId"`
}

func (ItemPriceHistoryModel) TableName() string {
	return "item_price_history"
}


