package models

import (
	"time"

	"gorm.io/gorm"
)

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

// CalculateStock computes total stock from all sizes
func (p *Product) CalculateStock() {
	total := 0
	for _, size := range p.Sizes {
		total += size.Stock
	}
	p.Stock = total
}

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
