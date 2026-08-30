package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/samkc-0/pamphlet-sync/internal/middleware"
	"github.com/samkc-0/pamphlet-sync/internal/models"
)

type BookMetadataHandler struct {
	DB *gorm.DB
}

func NewBookMetadataHandler(db *gorm.DB) *BookMetadataHandler {
	return &BookMetadataHandler{DB: db}
}

type upsertBookMetadataRequest struct {
	Title                  string    `json:"title"`
	Author                 string    `json:"author"`
	LanguageCode           string    `json:"languageCode"`
	DictionaryLanguageCode string    `json:"dictionaryLanguageCode"`
	FontFamily             string    `json:"fontFamily"`
	SpanishVoiceRegion     string    `json:"spanishVoiceRegion"`
	UpdatedAt              time.Time `json:"updatedAt" binding:"required"`
}

// Upsert saves the current user's display overrides for one book,
// identified by content hash. A write older than (or equal to) what's
// already stored is silently ignored.
func (h *BookMetadataHandler) Upsert(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)
	hash := c.Param("hash")

	var req upsertBookMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var existing models.BookMetadataOverride
	err := h.DB.Where("user_id = ? AND content_hash = ?", user.ID, hash).First(&existing).Error

	switch {
	case err == nil:
		if !req.UpdatedAt.After(existing.UpdatedAt) {
			c.Status(http.StatusNoContent)
			return
		}
		existing.Title = req.Title
		existing.Author = req.Author
		existing.LanguageCode = req.LanguageCode
		existing.DictionaryLanguageCode = req.DictionaryLanguageCode
		existing.FontFamily = req.FontFamily
		existing.SpanishVoiceRegion = req.SpanishVoiceRegion
		existing.UpdatedAt = req.UpdatedAt
		if err := h.DB.Save(&existing).Error; err != nil {
			log.Printf("upsert book metadata: update %s for user %s: %v", hash, user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save book metadata"})
			return
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		override := models.BookMetadataOverride{
			UserID:                 user.ID,
			ContentHash:            hash,
			Title:                  req.Title,
			Author:                 req.Author,
			LanguageCode:           req.LanguageCode,
			DictionaryLanguageCode: req.DictionaryLanguageCode,
			FontFamily:             req.FontFamily,
			SpanishVoiceRegion:     req.SpanishVoiceRegion,
			UpdatedAt:              req.UpdatedAt,
		}
		if err := h.DB.Create(&override).Error; err != nil {
			log.Printf("upsert book metadata: create %s for user %s: %v", hash, user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save book metadata"})
			return
		}
	default:
		log.Printf("upsert book metadata: query %s for user %s: %v", hash, user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save book metadata"})
		return
	}

	c.Status(http.StatusNoContent)
}

// List returns every book-metadata override for the current user, for bulk
// pulling on sign-in.
func (h *BookMetadataHandler) List(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var overrides []models.BookMetadataOverride
	err := h.DB.Where("user_id = ?", user.ID).Find(&overrides).Error
	if err != nil {
		log.Printf("list book metadata: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list book metadata"})
		return
	}

	c.JSON(http.StatusOK, overrides)
}
