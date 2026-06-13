package db

import (
	"log"

	"github.com/ioliveros/tunlr/internal/config"
	"github.com/ioliveros/tunlr/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Connect opens the SQLite database, enables foreign keys, and runs schema
// migrations. It fails fast: a desktop app with no working store is unusable.
func Connect(cfg *config.Config) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(cfg.DBPath+"?_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := model.Migrate(database); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	log.Println("database connected:", cfg.DBPath)
	return database
}
