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

type SettingsHandler struct {
	DB *gorm.DB
}

func NewSettingsHandler(db *gorm.DB) *SettingsHandler {
	return &SettingsHandler{DB: db}
}

type upsertSettingsRequest struct {
	AnimationsEnabled          bool      `json:"animationsEnabled"`
	AutoPlayWordAudio          bool      `json:"autoPlayWordAudio"`
	IsDarkMode                 bool      `json:"isDarkMode"`
	LastDictionaryLanguageCode string    `json:"lastDictionaryLanguageCode"`
	LastSpanishVoiceRegion     string    `json:"lastSpanishVoiceRegion"`
	UpdatedAt                  time.Time `json:"updatedAt" binding:"required"`
}

// Upsert saves the current user's app-wide settings. A write older than
// (or equal to) what's already stored is silently ignored.
func (h *SettingsHandler) Upsert(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var req upsertSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var existing models.UserSettings
	err := h.DB.Where("user_id = ?", user.ID).First(&existing).Error

	switch {
	case err == nil:
		if !req.UpdatedAt.After(existing.UpdatedAt) {
			c.Status(http.StatusNoContent)
			return
		}
		existing.AnimationsEnabled = req.AnimationsEnabled
		existing.AutoPlayWordAudio = req.AutoPlayWordAudio
		existing.IsDarkMode = req.IsDarkMode
		existing.LastDictionaryLanguageCode = req.LastDictionaryLanguageCode
		existing.LastSpanishVoiceRegion = req.LastSpanishVoiceRegion
		existing.UpdatedAt = req.UpdatedAt
		if err := h.DB.Save(&existing).Error; err != nil {
			log.Printf("upsert settings: update for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
			return
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		settings := models.UserSettings{
			UserID:                     user.ID,
			AnimationsEnabled:          req.AnimationsEnabled,
			AutoPlayWordAudio:          req.AutoPlayWordAudio,
			IsDarkMode:                 req.IsDarkMode,
			LastDictionaryLanguageCode: req.LastDictionaryLanguageCode,
			LastSpanishVoiceRegion:     req.LastSpanishVoiceRegion,
			UpdatedAt:                  req.UpdatedAt,
		}
		if err := h.DB.Create(&settings).Error; err != nil {
			log.Printf("upsert settings: create for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
			return
		}
	default:
		log.Printf("upsert settings: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Get returns the current user's settings, or 404 if none have been saved
// yet (the frontend keeps its local defaults in that case).
func (h *SettingsHandler) Get(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var settings models.UserSettings
	err := h.DB.Where("user_id = ?", user.ID).First(&settings).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("get settings: query for user %s: %v", user.ID, err)
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "no settings saved"})
		return
	}

	c.JSON(http.StatusOK, settings)
}
