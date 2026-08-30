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
	r.GET("/version", handlers.Version)

	authHandler := handlers.NewAuthHandler(db, cfg)
	r.GET("/auth/google/login", authHandler.GoogleLogin)
	r.GET("/auth/google/callback", authHandler.GoogleCallback)

	bookHandler := handlers.NewBookHandler(db)
	progressHandler := handlers.NewProgressHandler(db)
	pinnedWordHandler := handlers.NewPinnedWordHandler(db)
	settingsHandler := handlers.NewSettingsHandler(db)
	bookMetadataHandler := handlers.NewBookMetadataHandler(db)
	navigationHandler := handlers.NewNavigationHandler(db)

	protected := r.Group("/")
	protected.Use(middleware.RequireSession(db))
	protected.GET("/me", authHandler.Me)
	protected.POST("/auth/logout", authHandler.Logout)
	protected.POST("/books", bookHandler.Create)
	protected.GET("/books", bookHandler.List)
	protected.GET("/books/:hash", bookHandler.Get)
	protected.POST("/books/:hash/delete", bookHandler.Delete)
	protected.POST("/progress/:hash", progressHandler.Upsert)
	protected.GET("/progress", progressHandler.List)
	protected.POST("/pinned-words", pinnedWordHandler.Set)
	protected.GET("/pinned-words", pinnedWordHandler.List)
	protected.POST("/settings", settingsHandler.Upsert)
	protected.GET("/settings", settingsHandler.Get)
	protected.POST("/book-metadata/:hash", bookMetadataHandler.Upsert)
	protected.GET("/book-metadata", bookMetadataHandler.List)
	protected.POST("/navigation", navigationHandler.Upsert)
	protected.GET("/navigation", navigationHandler.Get)
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
