package models

import (
	"time"

	"gorm.io/gorm"
)

// DashboardStats represents the analytics data for admin dashboard
type DashboardStats struct {
	TotalUsers       int64   `json:"total_users"`
	TotalProducts    int64   `json:"total_products"`
	TotalCategories  int64   `json:"total_categories"`
	TotalReviews     int64   `json:"total_reviews"`
	LowStockProducts int64   `json:"low_stock_products"`
	AverageRating    float64 `json:"average_rating"`
}

// RecentActivity represents recent activity in the system
type RecentActivity struct {
	ID          uint      `json:"id"`
	Type        string    `json:"type"` // user_registered, product_created, review_added, etc.
	Description string    `json:"description"`
	UserID      *uint     `json:"user_id,omitempty"`
	UserName    string    `json:"user_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ActivityLog stores admin actions for audit trail
type ActivityLog struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	AdminID     uint           `gorm:"not null;index" json:"admin_id"`
	AdminName   string         `gorm:"size:100" json:"admin_name"`
	Action      string         `gorm:"size:100;not null" json:"action"`  // created, updated, deleted
	Resource    string         `gorm:"size:50;not null" json:"resource"` // product, user, category, etc.
	ResourceID  uint           `json:"resource_id"`
	Description string         `gorm:"type:text" json:"description"`
	IPAddress   string         `gorm:"size:45" json:"ip_address"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// UserStats represents detailed user statistics
type UserStats struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	Banned        bool      `json:"banned"`
	ReviewCount   int64     `json:"review_count"`
	WishlistCount int64     `json:"wishlist_count"`
	LastLoginAt   time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ProductStats represents detailed product statistics
type ProductStats struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	CategoryID    uint      `json:"category_id"`
	CategoryName  string    `json:"category_name"`
	Price         float64   `json:"price"`
	Stock         int       `json:"stock"`
	ImageURL      string    `json:"image_url"`
	ReviewCount   int64     `json:"review_count"`
	AverageRating float64   `json:"average_rating"`
	WishlistCount int64     `json:"wishlist_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// Order represents a customer order
type Order struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          uint           `gorm:"not null;index" json:"user_id"`
	OrderNumber     string         `gorm:"size:50;uniqueIndex;not null" json:"order_number"`
	Status          string         `gorm:"size:20;not null;default:'pending'" json:"status"` // pending, processing, shipped, delivered, cancelled
	TotalAmount     float64        `gorm:"not null" json:"total_amount"`
	ShippingAddress string         `gorm:"type:text" json:"shipping_address"`
	PaymentMethod   string         `gorm:"size:50" json:"payment_method"`
	PaymentStatus   string         `gorm:"size:20;default:'pending'" json:"payment_status"` // pending, paid, failed, refunded
	Notes           string         `gorm:"type:text" json:"notes"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Items []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OrderID     uint      `gorm:"not null;index" json:"order_id"`
	ProductID   uint      `gorm:"not null" json:"product_id"`
	ProductName string    `gorm:"size:200" json:"product_name"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	Price       float64   `gorm:"not null" json:"price"`
	Subtotal    float64   `gorm:"not null" json:"subtotal"`
	CreatedAt   time.Time `json:"created_at"`
}

// OrderStats represents order statistics for admin view
type OrderStats struct {
	ID              uint      `json:"id"`
	OrderNumber     string    `json:"order_number"`
	UserID          uint      `json:"user_id"`
	UserName        string    `json:"user_name"`
	UserEmail       string    `json:"user_email"`
	Status          string    `json:"status"`
	PaymentStatus   string    `json:"payment_status"`
	PaymentMethod   string    `json:"payment_method"`
	TotalAmount     float64   `json:"total_amount"`
	ItemCount       int64     `json:"item_count"`
	ShippingAddress string    `json:"shipping_address"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
