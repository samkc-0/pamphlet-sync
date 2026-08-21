package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/samkc-0/pamphlet-sync/internal/config"
	"github.com/samkc-0/pamphlet-sync/internal/db"
	"github.com/samkc-0/pamphlet-sync/internal/routes"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(conn); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	r := gin.Default()
	routes.Setup(r, conn, cfg)

	log.Printf("listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
