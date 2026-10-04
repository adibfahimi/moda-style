// Package database owns the gorm handle of the cart service. The handle is
// shared with the other services through the same PostgreSQL instance, which is
// how cart lines can be joined with the product catalogue.
package database

import (
	"github.com/adibfahimi/moda-style/common"
	"gorm.io/gorm"
)

// DB is the process-wide gorm handle. It is nil until Connect runs and is
// replaced by an in-memory database in tests.
var DB *gorm.DB

// Connect reads the database settings from the environment via
// common.GetDatabaseConfig and opens the shared connection pool, terminating the
// process when the database is unreachable.
func Connect() {
	config := common.GetDatabaseConfig()
	DB = common.MustConnectDatabase(config)
}
