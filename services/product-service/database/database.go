package database

import (
	"github.com/adibfahimi/moda-style/common"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	config := common.GetDatabaseConfig()
	DB = common.MustConnectDatabase(config)
}
