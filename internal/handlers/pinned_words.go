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

type PinnedWordHandler struct {
	DB *gorm.DB
}

func NewPinnedWordHandler(db *gorm.DB) *PinnedWordHandler {
	return &PinnedWordHandler{DB: db}
}

type setPinnedWordRequest struct {
	LanguageCode string    `json:"languageCode" binding:"required"`
	Word         string    `json:"word" binding:"required"`
	Pinned       bool      `json:"pinned"`
	UpdatedAt    time.Time `json:"updatedAt" binding:"required"`
}

// Set pins or unpins a word for the current user. Unpinning updates the
// existing row rather than deleting it, so UpdatedAt survives for
// last-write-wins conflict resolution against other devices.
func (h *PinnedWordHandler) Set(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var req setPinnedWordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var existing models.PinnedWord
	err := h.DB.
		Where("user_id = ? AND language_code = ? AND word = ?", user.ID, req.LanguageCode, req.Word).
		First(&existing).Error

	switch {
	case err == nil:
		if !req.UpdatedAt.After(existing.UpdatedAt) {
			c.Status(http.StatusNoContent)
			return
		}
		existing.Pinned = req.Pinned
		existing.UpdatedAt = req.UpdatedAt
		if err := h.DB.Save(&existing).Error; err != nil {
			log.Printf("set pinned word: update for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save pinned word"})
			return
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		word := models.PinnedWord{
			UserID:       user.ID,
			LanguageCode: req.LanguageCode,
			Word:         req.Word,
			Pinned:       req.Pinned,
			UpdatedAt:    req.UpdatedAt,
		}
		if err := h.DB.Create(&word).Error; err != nil {
			log.Printf("set pinned word: create for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save pinned word"})
			return
		}
	default:
		log.Printf("set pinned word: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save pinned word"})
		return
	}

	c.Status(http.StatusNoContent)
}

// List returns every word the current user currently has pinned.
func (h *PinnedWordHandler) List(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var words []models.PinnedWord
	err := h.DB.Where("user_id = ? AND pinned = ?", user.ID, true).Find(&words).Error
	if err != nil {
		log.Printf("list pinned words: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pinned words"})
		return
	}

	c.JSON(http.StatusOK, words)
}
