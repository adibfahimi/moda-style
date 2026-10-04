package database

import (
	"log"

	"github.com/adibfahimi/moda-style/services/cart-service/models"
)

// Migrate creates or updates the cart and wishlist tables through gorm
// AutoMigrate. It runs on startup right after Connect and exits the process on
// failure so a schema mismatch never serves traffic silently.
func Migrate() {
	err := DB.AutoMigrate(
		&models.CartItem{},
		&models.WishlistItem{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migrated successfully")
}
