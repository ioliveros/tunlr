package db

import (
	"log"
	"github.com/ioliveros/tunlr/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	log.Println("Database connected:", cfg.DBPath)
	return database
}
