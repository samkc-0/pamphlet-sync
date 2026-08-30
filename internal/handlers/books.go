package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/samkc-0/pamphlet-sync/internal/middleware"
	"github.com/samkc-0/pamphlet-sync/internal/models"
)

// BookChapter is one chapter of a book's extracted plain-text content, as
// produced by the frontend's EPUB parser (images/scripts/styles already
// stripped, paragraphs in reading order).
type BookChapter struct {
	ID           string   `json:"id"`
	ChapterTitle string   `json:"chapterTitle,omitempty"`
	Paragraphs   []string `json:"paragraphs"`
}

type BookHandler struct {
	DB *gorm.DB
}

func NewBookHandler(db *gorm.DB) *BookHandler {
	return &BookHandler{DB: db}
}

type createBookRequest struct {
	ContentHash string        `json:"contentHash" binding:"required"`
	Title       string        `json:"title"`
	Author      string        `json:"author"`
	Language    string        `json:"language"`
	Chapters    []BookChapter `json:"chapters" binding:"required"`
}

// Create stores a book's extracted content, scoped to the current user.
// Content at a given hash is immutable, so this is upsert-if-absent: a book
// already stored under this hash for this user is left untouched.
func (h *BookHandler) Create(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var req createBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	content, err := json.Marshal(req.Chapters)
	if err != nil {
		log.Printf("create book: marshal chapters for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save book"})
		return
	}

	book := models.Book{
		ID:          uuid.NewString(),
		UserID:      user.ID,
		ContentHash: req.ContentHash,
		Title:       req.Title,
		Author:      req.Author,
		Language:    req.Language,
		Content:     string(content),
		CreatedAt:   time.Now(),
	}

	err = h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "content_hash"}},
		DoNothing: true,
	}).Create(&book).Error
	if err != nil {
		log.Printf("create book: save for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save book"})
		return
	}

	c.Status(http.StatusNoContent)
}

// List returns metadata for every book the current user has synced, without
// their content, so the frontend can diff its local catalog against it.
func (h *BookHandler) List(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)

	var books []models.Book
	err := h.DB.
		Select("id", "user_id", "content_hash", "title", "author", "language", "created_at").
		Where("user_id = ?", user.ID).
		Find(&books).Error
	if err != nil {
		log.Printf("list books: query for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list books"})
		return
	}

	c.JSON(http.StatusOK, books)
}

type bookContentResponse struct {
	Title    string        `json:"title"`
	Author   string        `json:"author"`
	Language string        `json:"language"`
	Chapters []BookChapter `json:"chapters"`
}

// Get returns one book's full extracted content by content hash, scoped to
// the current user.
func (h *BookHandler) Get(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)
	hash := c.Param("hash")

	var book models.Book
	err := h.DB.Where("user_id = ? AND content_hash = ?", user.ID, hash).First(&book).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("get book: query %s for user %s: %v", hash, user.ID, err)
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	var chapters []BookChapter
	if err := json.Unmarshal([]byte(book.Content), &chapters); err != nil {
		log.Printf("get book: unmarshal content %s for user %s: %v", hash, user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load book"})
		return
	}

	c.JSON(http.StatusOK, bookContentResponse{
		Title:    book.Title,
		Author:   book.Author,
		Language: book.Language,
		Chapters: chapters,
	})
}
