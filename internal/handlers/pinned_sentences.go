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

type PinnedSentenceHandler struct {
	DB *gorm.DB
}

func NewPinnedSentenceHandler(db *gorm.DB) *PinnedSentenceHandler {
	return &PinnedSentenceHandler{DB: db}
}

type setPinnedSentenceRequest struct {
	LanguageCode string    `json:"languageCode" binding:"required"`
	Sentence     string    `json:"sentence" binding:"required"`
	Pinned       bool      `json:"pinned"`
	UpdatedAt    time.Time `json:"updatedAt" binding:"required"`
}

// Set pins or unpins a sentence for the current user. Unpinning updates the
// existing row rather than deleting it, so UpdatedAt survives for
// last-write-wins conflict resolution against other devices.
func (h *PinnedSentenceHandler) Set(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var req setPinnedSentenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var existing models.PinnedSentence
	err := h.DB.
		Where("user_id = ? AND language_code = ? AND sentence = ?", user.ID, req.LanguageCode, req.Sentence).
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
			log.Printf("set pinned sentence: update for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save pinned sentence"})
			return
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		sentence := models.PinnedSentence{
			UserID:       user.ID,
			LanguageCode: req.LanguageCode,
			Sentence:     req.Sentence,
			Pinned:       req.Pinned,
			UpdatedAt:    req.UpdatedAt,
		}
		if err := h.DB.Create(&sentence).Error; err != nil {
			log.Printf("set pinned sentence: create for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save pinned sentence"})
			return
		}
	default:
		log.Printf("set pinned sentence: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save pinned sentence"})
		return
	}

	c.Status(http.StatusNoContent)
}

// List returns every sentence the current user currently has pinned.
func (h *PinnedSentenceHandler) List(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var sentences []models.PinnedSentence
	err := h.DB.Where("user_id = ? AND pinned = ?", user.ID, true).Find(&sentences).Error
	if err != nil {
		log.Printf("list pinned sentences: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pinned sentences"})
		return
	}

	c.JSON(http.StatusOK, sentences)
}
