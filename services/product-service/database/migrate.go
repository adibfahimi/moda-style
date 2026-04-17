package database

import (
	"log"

	"github.com/adibfahimi/moda-style/services/product-service/models"
)

func Migrate() {
	err := DB.AutoMigrate(
		&models.Category{},
		&models.Product{},
		&models.Size{},
		&models.Review{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Allow duplicate category names under different parents in hierarchical mode.
	if err := DB.Exec("DROP INDEX IF EXISTS idx_categories_name").Error; err != nil {
		log.Printf("Warning: failed to drop legacy category name unique index: %v", err)
	}

	// Allow duplicate category slugs under different parents in hierarchical mode.
	if err := DB.Exec("DROP INDEX IF EXISTS idx_categories_slug").Error; err != nil {
		log.Printf("Warning: failed to drop legacy category slug unique index: %v", err)
	}
	if err := DB.Exec("DROP INDEX IF EXISTS categories_slug_key").Error; err != nil {
		log.Printf("Warning: failed to drop legacy category slug unique key: %v", err)
	}

	log.Println("Database migrated successfully")
}
