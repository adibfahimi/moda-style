// Package database owns the process-wide GORM connection of the auth service.
//
// The connection parameters come from common.GetDatabaseConfig, which reads
// them from the environment, so the same binary works against any PostgreSQL
// instance without code changes.
package database

import (
	"log"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/auth-service/models"
	"gorm.io/gorm"
)

// DB is the shared GORM handle. It is nil until Connect has been called and is
// used by every handler in this service.
var DB *gorm.DB

// Connect opens the PostgreSQL connection described by the environment and
// stores it in DB. It terminates the process when the database is unreachable,
// because the service cannot serve requests without it.
func Connect() {
	config := common.GetDatabaseConfig()
	DB = common.MustConnectDatabase(config)
}

// Migrate creates or updates the tables the auth service owns (currently only
// the users table). It terminates the process when the schema cannot be
// applied.
func Migrate() {
	err := DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migrated successfully")
}
