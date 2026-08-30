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

type ProgressHandler struct {
	DB *gorm.DB
}

func NewProgressHandler(db *gorm.DB) *ProgressHandler {
	return &ProgressHandler{DB: db}
}

type upsertProgressRequest struct {
	ChapterID      string    `json:"chapterId" binding:"required"`
	ParagraphIndex int       `json:"paragraphIndex"`
	UpdatedAt      time.Time `json:"updatedAt" binding:"required"`
}

// Upsert records the current user's reading position in a book, identified
// by content hash. A write older than (or equal to) what's already stored
// is silently ignored, so devices racing to sync can't let a stale write
// clobber a newer one.
func (h *ProgressHandler) Upsert(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)
	hash := c.Param("hash")

	var req upsertProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var existing models.ReadingProgress
	err := h.DB.Where("user_id = ? AND content_hash = ?", user.ID, hash).First(&existing).Error

	switch {
	case err == nil:
		if !req.UpdatedAt.After(existing.UpdatedAt) {
			c.Status(http.StatusNoContent)
			return
		}
		existing.ChapterID = req.ChapterID
		existing.ParagraphIndex = req.ParagraphIndex
		existing.UpdatedAt = req.UpdatedAt
		if err := h.DB.Save(&existing).Error; err != nil {
			log.Printf("upsert progress: update %s for user %s: %v", hash, user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save progress"})
			return
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		progress := models.ReadingProgress{
			UserID:         user.ID,
			ContentHash:    hash,
			ChapterID:      req.ChapterID,
			ParagraphIndex: req.ParagraphIndex,
			UpdatedAt:      req.UpdatedAt,
		}
		if err := h.DB.Create(&progress).Error; err != nil {
			log.Printf("upsert progress: create %s for user %s: %v", hash, user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save progress"})
			return
		}
	default:
		log.Printf("upsert progress: query %s for user %s: %v", hash, user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save progress"})
		return
	}

	c.Status(http.StatusNoContent)
}

// List returns every reading-progress record for the current user, for bulk
// pulling on sign-in.
func (h *ProgressHandler) List(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var progress []models.ReadingProgress
	if err := h.DB.Where("user_id = ?", user.ID).Find(&progress).Error; err != nil {
		log.Printf("list progress: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}
