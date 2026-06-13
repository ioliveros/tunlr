package config

import (
	"log"
	"os"
	"path/filepath"
)

// Config holds runtime configuration for the desktop app.
type Config struct {
	// DBPath is the absolute path to the SQLite database file.
	DBPath string
}

// Load resolves configuration, defaulting the database into the
// per-user application-support directory so a packaged .app stores
// its state in the conventional macOS location. DB_PATH overrides it
// (handy for tests and development).
func Load() *Config {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath()
	}
	return &Config{DBPath: dbPath}
}

// defaultDBPath returns <user-config-dir>/tunlr/tunlr.db, creating the
// directory if needed. On macOS that resolves to
// ~/Library/Application Support/tunlr/tunlr.db.
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
