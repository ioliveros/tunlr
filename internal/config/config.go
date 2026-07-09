package config

import (
	"log"
	"os"
	"path/filepath"
)


type Config struct {
	DBPath string
}

func Load() *Config {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath()
	}
	return &Config{DBPath: dbPath}
}

func defaultDBPath() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	dir := filepath.Join(base, "tunlr")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("could not create config dir %s: %v", dir, err)
		return "tunlr.db"
	}
	return filepath.Join(dir, "tunlr.db")
}
