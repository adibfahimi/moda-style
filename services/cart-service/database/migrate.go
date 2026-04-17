package database

import (
	"log"

	"github.com/adibfahimi/moda-style/services/cart-service/models"
)

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
