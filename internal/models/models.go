// Package models contains GORM model definitions.
package models

import "time"

// User is an account identified by a Google OAuth login.
type User struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	GoogleSub string    `gorm:"uniqueIndex;not null" json:"-"`
	Email     string    `gorm:"not null" json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// Session is an opaque bearer token issued after a successful login.
type Session struct {
	Token     string    `gorm:"primaryKey" json:"-"`
	UserID    string    `gorm:"not null;index" json:"-"`
	CreatedAt time.Time `json:"-"`
	ExpiresAt time.Time `json:"-"`
}

// Book is a user's uploaded book, stored as its extracted plain-text
// chapters (see internal/handlers.BookChapter) rather than the original
// EPUB file. ContentHash identifies the book across devices: the same
// extracted text always hashes to the same value.
type Book struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"not null;uniqueIndex:idx_user_content_hash" json:"-"`
	ContentHash string    `gorm:"not null;uniqueIndex:idx_user_content_hash" json:"contentHash"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Language    string    `json:"language"`
	Content     string    `json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ReadingProgress is a user's last-known reading position in a book,
// identified by Book.ContentHash. UpdatedAt is client-supplied and used to
// resolve conflicting writes from multiple devices (last write wins).
type ReadingProgress struct {
	UserID         string    `gorm:"primaryKey" json:"-"`
	ContentHash    string    `gorm:"primaryKey" json:"contentHash"`
	ChapterID      string    `gorm:"not null" json:"chapterId"`
	ParagraphIndex int       `gorm:"not null" json:"paragraphIndex"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// PinnedWord tracks whether a user has pinned a word in a given language.
// Unpinning sets Pinned to false rather than deleting the row, so the
// UpdatedAt timestamp survives for last-write-wins conflict resolution.
type PinnedWord struct {
	UserID       string    `gorm:"primaryKey" json:"-"`
	LanguageCode string    `gorm:"primaryKey" json:"languageCode"`
	Word         string    `gorm:"primaryKey" json:"word"`
	Pinned       bool      `json:"pinned"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// All returns every model that should be included in auto-migration.
func All() []interface{} {
	return []interface{}{
		&User{},
		&Session{},
		&Book{},
		&ReadingProgress{},
		&PinnedWord{},
	}
}
