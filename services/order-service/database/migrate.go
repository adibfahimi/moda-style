package database

import (
	"log"

	"github.com/adibfahimi/moda-style/services/order-service/models"
)

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
