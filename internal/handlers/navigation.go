package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/samkc-0/pamphlet-sync/internal/middleware"
	"github.com/samkc-0/pamphlet-sync/internal/models"
)

type NavigationHandler struct {
	DB *gorm.DB
}

func NewNavigationHandler(db *gorm.DB) *NavigationHandler {
	return &NavigationHandler{DB: db}
}

type upsertNavigationStateRequest struct {
	ActiveRowID       string    `json:"activeRowId"`
	OpenContentHashes []string  `json:"openContentHashes"`
	LibraryPageID     string    `json:"libraryPageId"`
	SettingsPageID    string    `json:"settingsPageId"`
	UpdatedAt         time.Time `json:"updatedAt" binding:"required"`
}

type navigationStateResponse struct {
	ActiveRowID       string    `json:"activeRowId"`
	OpenContentHashes []string  `json:"openContentHashes"`
	LibraryPageID     string    `json:"libraryPageId"`
	SettingsPageID    string    `json:"settingsPageId"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// Upsert saves which screen the current user was looking at. A write older
// than (or equal to) what's already stored is silently ignored.
func (h *NavigationHandler) Upsert(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var req upsertNavigationStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	openContentHashes, err := json.Marshal(req.OpenContentHashes)
	if err != nil {
		log.Printf("upsert navigation state: marshal open hashes for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save navigation state"})
		return
	}

	var existing models.NavigationState
	queryErr := h.DB.Where("user_id = ?", user.ID).First(&existing).Error

	switch {
	case queryErr == nil:
		if !req.UpdatedAt.After(existing.UpdatedAt) {
			c.Status(http.StatusNoContent)
			return
		}
		existing.ActiveRowID = req.ActiveRowID
		existing.OpenContentHashes = string(openContentHashes)
		existing.LibraryPageID = req.LibraryPageID
		existing.SettingsPageID = req.SettingsPageID
		existing.UpdatedAt = req.UpdatedAt
		if err := h.DB.Save(&existing).Error; err != nil {
			log.Printf("upsert navigation state: update for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save navigation state"})
			return
		}
	case errors.Is(queryErr, gorm.ErrRecordNotFound):
		state := models.NavigationState{
			UserID:            user.ID,
			ActiveRowID:       req.ActiveRowID,
			OpenContentHashes: string(openContentHashes),
			LibraryPageID:     req.LibraryPageID,
			SettingsPageID:    req.SettingsPageID,
			UpdatedAt:         req.UpdatedAt,
		}
		if err := h.DB.Create(&state).Error; err != nil {
			log.Printf("upsert navigation state: create for user %s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save navigation state"})
			return
		}
	default:
		log.Printf("upsert navigation state: query for user %s: %v", user.ID, queryErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save navigation state"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Get returns the current user's last-known screen, or 404 if none has
// been saved yet.
func (h *NavigationHandler) Get(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var state models.NavigationState
	err := h.DB.Where("user_id = ?", user.ID).First(&state).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("get navigation state: query for user %s: %v", user.ID, err)
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "no navigation state saved"})
		return
	}

	var openContentHashes []string
	if err := json.Unmarshal([]byte(state.OpenContentHashes), &openContentHashes); err != nil {
		openContentHashes = []string{}
	}

	c.JSON(http.StatusOK, navigationStateResponse{
		ActiveRowID:       state.ActiveRowID,
		OpenContentHashes: openContentHashes,
		LibraryPageID:     state.LibraryPageID,
		SettingsPageID:    state.SettingsPageID,
		UpdatedAt:         state.UpdatedAt,
	})
}
