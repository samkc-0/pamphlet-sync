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

type NotebookHandler struct {
	DB *gorm.DB
}

func NewNotebookHandler(db *gorm.DB) *NotebookHandler {
	return &NotebookHandler{DB: db}
}

type upsertNotebookRequest struct {
	ID           string    `json:"id" binding:"required"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	LanguageCode string    `json:"languageCode"`
	FontFamily   string    `json:"fontFamily"`
	UpdatedAt    time.Time `json:"updatedAt" binding:"required"`
}

// Upsert creates a notebook or updates an existing one for the current user.
// Unlike Book, a notebook's content is mutable, so this is a conditional
// last-write-wins write (like PinnedWord), not upsert-if-absent: an update
// older than what's already stored is silently ignored.
func (h *NotebookHandler) Upsert(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var req upsertNotebookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var existing models.Notebook
	err := h.DB.
		Where("user_id = ? AND id = ?", user.ID, req.ID).
		First(&existing).Error

	switch {
	case err == nil:
		if !req.UpdatedAt.After(existing.UpdatedAt) {
			c.Status(http.StatusNoContent)
			return
		}
		existing.Title = req.Title
		existing.Content = req.Content
		existing.LanguageCode = req.LanguageCode
		existing.FontFamily = req.FontFamily
		existing.Deleted = false
		existing.UpdatedAt = req.UpdatedAt
		if err := h.DB.Save(&existing).Error; err != nil {
			log.Printf("upsert notebook: update for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save notebook"})
			return
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		notebook := models.Notebook{
			ID:           req.ID,
			UserID:       user.ID,
			Title:        req.Title,
			Content:      req.Content,
			LanguageCode: req.LanguageCode,
			FontFamily:   req.FontFamily,
			UpdatedAt:    req.UpdatedAt,
		}
		if err := h.DB.Create(&notebook).Error; err != nil {
			log.Printf("upsert notebook: create for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save notebook"})
			return
		}
	default:
		log.Printf("upsert notebook: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save notebook"})
		return
	}

	c.Status(http.StatusNoContent)
}

// List returns metadata for every notebook the current user has synced,
// without its content, so the frontend can diff its local catalog against
// it - the same shape as BookHandler.List.
func (h *NotebookHandler) List(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var notebooks []models.Notebook
	err := h.DB.
		Select("id", "user_id", "title", "language_code", "font_family", "deleted", "created_at", "updated_at").
		Where("user_id = ?", user.ID).
		Find(&notebooks).Error
	if err != nil {
		log.Printf("list notebooks: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notebooks"})
		return
	}

	c.JSON(http.StatusOK, notebooks)
}

type notebookContentResponse struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	LanguageCode string    `json:"languageCode"`
	FontFamily   string    `json:"fontFamily"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Get returns one notebook's full content by ID, scoped to the current
// user. Notebook.Content is tagged json:"-" so List (which serializes the
// model directly) never leaks it - so unlike List, Get can't just serialize
// the model either; it needs its own response type that does include it.
func (h *NotebookHandler) Get(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)
	id := c.Param("id")

	var notebook models.Notebook
	err := h.DB.Where("user_id = ? AND id = ?", user.ID, id).First(&notebook).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("get notebook: query %s for user %s: %v", id, user.ID, err)
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "notebook not found"})
		return
	}
	if notebook.Deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "notebook not found"})
		return
	}

	c.JSON(http.StatusOK, notebookContentResponse{
		ID:           notebook.ID,
		Title:        notebook.Title,
		Content:      notebook.Content,
		LanguageCode: notebook.LanguageCode,
		FontFamily:   notebook.FontFamily,
		UpdatedAt:    notebook.UpdatedAt,
	})
}

type deleteNotebookRequest struct {
	UpdatedAt time.Time `json:"updatedAt" binding:"required"`
}

// Delete marks a notebook deleted for the current user, so other devices can
// positively confirm the deletion via List rather than inferring it from the
// notebook's absence. The row is kept (not hard-deleted) so UpdatedAt
// survives for last-write-wins: a delete older than what's already recorded
// is silently ignored.
func (h *NotebookHandler) Delete(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)
	id := c.Param("id")

	var req deleteNotebookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var notebook models.Notebook
	err := h.DB.Where("user_id = ? AND id = ?", user.ID, id).First(&notebook).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusNoContent)
			return
		}
		log.Printf("delete notebook: query %s for user %s: %v", id, user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete notebook"})
		return
	}

	if !req.UpdatedAt.After(notebook.UpdatedAt) {
		c.Status(http.StatusNoContent)
		return
	}

	notebook.Deleted = true
	notebook.UpdatedAt = req.UpdatedAt
	if err := h.DB.Save(&notebook).Error; err != nil {
		log.Printf("delete notebook: update %s for user %s: %v", id, user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete notebook"})
		return
	}

	c.Status(http.StatusNoContent)
}
