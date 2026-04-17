package models

import (
	"time"

	"gorm.io/gorm"
)

type CartItem struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	ProductID uint           `gorm:"not null;index" json:"product_id"`
	SizeID    uint           `gorm:"not null" json:"size_id"`
	Quantity  int            `gorm:"not null;default:1" json:"quantity"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// CartItemWithProduct is used for API responses with product details
type CartItemWithProduct struct {
	ID        uint    `json:"id"`
	ProductID uint    `json:"product_id"`
	Name      string  `json:"name"`
	ImageURL  string  `json:"image_url"`
	Price     float64 `json:"price"`
	Size      string  `json:"size"`
	Color     string  `json:"color"`
	SizeID    uint    `json:"size_id"`
	Quantity  int     `json:"quantity"`
	Stock     int     `json:"stock"`
}

type WishlistItem struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	ProductID uint           `gorm:"not null;index" json:"product_id"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// WishlistItemWithProduct is used for API responses with product details
type WishlistItemWithProduct struct {
	ID        uint    `json:"id"`
	ProductID uint    `json:"product_id"`
	Name      string  `json:"name"`
	ImageURL  string  `json:"image_url"`
	Price     float64 `json:"price"`
	InStock   bool    `json:"in_stock"`
}
