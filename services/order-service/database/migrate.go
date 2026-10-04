package database

import (
	"log"

	"github.com/adibfahimi/moda-style/services/order-service/models"
)

// Migrate creates or updates the order, order item and payment transaction
// tables through gorm AutoMigrate. It runs on startup right after Connect and
// exits the process on failure so a schema mismatch never serves traffic
// silently.
func Migrate() {
	err := DB.AutoMigrate(
		&models.Order{},
		&models.OrderItem{},
		&models.PaymentTransaction{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migrated successfully")
}
