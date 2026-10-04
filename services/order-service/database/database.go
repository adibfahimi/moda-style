// Package database owns the gorm handle of the order service. The handle points
// at the same PostgreSQL instance as the other services, which is how checkout
// can read cart lines and adjust product stock.
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
