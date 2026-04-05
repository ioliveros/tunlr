package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/ioliveros/tunlr/internal/config"
	"github.com/ioliveros/tunlr/internal/db"
	"github.com/ioliveros/tunlr/internal/handler"
	"github.com/ioliveros/tunlr/internal/middleware"
)

func main() {
	cfg := config.Load()
	_ = db.Connect(cfg)

	r := gin.Default()
	r.Use(middleware.CORS(cfg))

	r.GET("/health", handler.Health)

	log.Printf("tunlr API running on :%s", cfg.Port)
	log.Fatal(r.Run(":" + cfg.Port))
}
