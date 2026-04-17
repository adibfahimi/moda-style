package database

import (
	"log"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/auth-service/models"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	config := common.GetDatabaseConfig()
	DB = common.MustConnectDatabase(config)
}

func Migrate() {
	err := DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migrated successfully")
}
