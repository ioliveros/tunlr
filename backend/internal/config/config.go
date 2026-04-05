package config

import (
	"log"
	"os"
	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DBPath      string
	FrontendURL string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}
	return &Config{
		Port:        getEnv("PORT", "8082"),
		DBPath:      getEnv("DB_PATH", "tunlr.db"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
