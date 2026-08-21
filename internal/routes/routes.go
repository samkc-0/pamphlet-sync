// Package routes wires up gin route handlers.
package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/samkc-0/pamphlet-sync/internal/config"
	"github.com/samkc-0/pamphlet-sync/internal/handlers"
	"github.com/samkc-0/pamphlet-sync/internal/middleware"
)

// Setup registers all application routes on the given gin engine.
func Setup(r *gin.Engine, db *gorm.DB, cfg config.Config) {
	r.Use(corsMiddleware(cfg.FrontendURL))

	health := handlers.NewHealthHandler(db)
	r.GET("/healthz", health.Check)

	authHandler := handlers.NewAuthHandler(db, cfg)
	r.GET("/auth/google/login", authHandler.GoogleLogin)
	r.GET("/auth/google/callback", authHandler.GoogleCallback)

	protected := r.Group("/")
	protected.Use(middleware.RequireSession(db))
	protected.GET("/me", authHandler.Me)
	protected.POST("/auth/logout", authHandler.Logout)
}

func corsMiddleware(frontendURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", frontendURL)
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
