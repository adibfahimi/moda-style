package database

import (
	"log"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/admin-service/models"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	config := common.GetDatabaseConfig()
	DB = common.MustConnectDatabase(config)
}

func Migrate() {
	err := DB.AutoMigrate(
		&models.ActivityLog{},
		&models.Order{},
		&models.OrderItem{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Admin service database migrated successfully")
}
