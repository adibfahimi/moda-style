// Package models defines the persistence and response types of the cart
// service.
//
// CartItem and WishlistItem are the migrated tables; the *WithProduct structs
// are read-only projections joined against the products/sizes tables for API
// responses.
package models

import (
	"time"

	"gorm.io/gorm"
)

// CartItem is one line of a user's cart: a product plus the size variant the
// customer picked and how many units they want.
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

// CartItemWithProduct is the API shape of a cart line. It flattens the product
// name, image and price together with the selected size, colour and the stock
// left for that variant.
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

// WishlistItem records that a user saved a product for later. A user can save a
// product at most once; deleting it uses soft deletes.
type WishlistItem struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	ProductID uint           `gorm:"not null;index" json:"product_id"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// WishlistItemWithProduct is the API shape of a wishlist entry: the product
// summary plus an InStock flag that is true when any size variant still has
// stock.
type WishlistItemWithProduct struct {
	ID        uint    `json:"id"`
	ProductID uint    `json:"product_id"`
	Name      string  `json:"name"`
	ImageURL  string  `json:"image_url"`
	Price     float64 `json:"price"`
	InStock   bool    `json:"in_stock"`
}
