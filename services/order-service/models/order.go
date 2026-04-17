package models

import (
	"time"

	"gorm.io/gorm"
)

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
	PaymentIntentID string         `gorm:"size:100" json:"payment_intent_id"`               // Stripe payment intent ID (or fake ID for testing)
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
	SizeID      *uint     `json:"size_id,omitempty"`
	Size        string    `gorm:"size:10" json:"size"`
	Color       string    `gorm:"size:50" json:"color"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	Price       float64   `gorm:"not null" json:"price"`
	Subtotal    float64   `gorm:"not null" json:"subtotal"`
	CreatedAt   time.Time `json:"created_at"`
}

// PaymentTransaction represents a payment attempt
type PaymentTransaction struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	OrderID         uint           `gorm:"not null;index" json:"order_id"`
	PaymentIntentID string         `gorm:"size:100" json:"payment_intent_id"`
	Amount          float64        `gorm:"not null" json:"amount"`
	Currency        string         `gorm:"size:3;default:'usd'" json:"currency"`
	Status          string         `gorm:"size:20" json:"status"` // pending, succeeded, failed
	PaymentMethod   string         `gorm:"size:50" json:"payment_method"`
	ErrorMessage    string         `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}
