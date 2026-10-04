// Package models defines the product-service persistence entities: the category
// tree, products, their size/colour variants with per-variant stock and the
// customer reviews attached to products.
//
// See database/migrate.go for the schema ownership of each entity.
package models

import (
	"time"

	"gorm.io/gorm"
)

// Category is a node of the category tree. ParentID is nil for root categories;
// Parent and Children expose the tree to clients, and soft deletes keep products
// pointing at their original category.
type Category struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100;not null" json:"name"`
	Slug      string         `gorm:"size:100;not null" json:"slug"`
	ParentID  *uint          `gorm:"index" json:"parent_id,omitempty"`
	Parent    *Category      `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children  []Category     `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Product is a catalogue item.
//
// Stock is intentionally not persisted (gorm:"-"): it is derived from the
// product's sizes via CalculateStock, so availability always reflects the
// per-variant rows written by the admin service.
type Product struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Price       float64        `gorm:"not null" json:"price"`
	CategoryID  uint           `gorm:"not null" json:"category_id"`
	Category    Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	ImageURL    string         `gorm:"size:255" json:"image_url"`
	Stock       int            `gorm:"-" json:"stock"` // Computed from sizes
	Sizes       []Size         `gorm:"foreignKey:ProductID" json:"sizes,omitempty"`
	Reviews     []Review       `gorm:"foreignKey:ProductID" json:"reviews,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// CalculateStock sums the stock of every size variant into the transient Stock
// field. Callers must have preloaded Sizes, otherwise the sum is zero.
func (p *Product) CalculateStock() {
	total := 0
	for _, size := range p.Sizes {
		total += size.Stock
	}
	p.Stock = total
}

// Size is one purchasable variant of a product: a size/colour combination with
// its own stock counter.
type Size struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ProductID uint           `gorm:"not null;index" json:"product_id"`
	Size      string         `gorm:"size:10;not null" json:"size"` // S, M, L, XL, etc.
	Color     string         `gorm:"size:50;not null" json:"color"`
	Stock     int            `gorm:"not null;default:0" json:"stock"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Review is a customer rating for a product.
//
// UserName is denormalised on purpose so review listings never need to query the
// users table, which lives in a different service. The database enforces a
// CHECK constraint on Rating so values outside 1-5 cannot be stored even if a
// caller bypasses the request validator.
type Review struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ProductID uint           `gorm:"not null;index" json:"product_id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	UserName  string         `gorm:"size:255" json:"user_name"` // Store user name for display
	Rating    int            `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Comment   string         `gorm:"type:text" json:"comment"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
