// Package database owns the process-wide GORM connection of the product service.
//
// Connection parameters are read from the environment by
// common.GetDatabaseConfig so the same binary works against any PostgreSQL
// instance. Schema changes live in migrate.go.
package database

import (
	"github.com/adibfahimi/moda-style/common"
	"gorm.io/gorm"
)

// DB is the shared GORM handle used by every handler; it is nil until Connect
// has been called.
var DB *gorm.DB

// Connect opens the database described by the environment and stores the handle
// in DB, terminating the process when the connection cannot be established.
func Connect() {
	config := common.GetDatabaseConfig()
	DB = common.MustConnectDatabase(config)
}
