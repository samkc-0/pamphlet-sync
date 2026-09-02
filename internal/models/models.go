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
//
// Deleted marks a book removed on some device; the row is kept (not
// hard-deleted) so UpdatedAt survives for last-write-wins conflict
// resolution, and so other devices can positively confirm a deletion
// happened rather than inferring it from the book's absence from List.
type Book struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"not null;uniqueIndex:idx_user_content_hash" json:"-"`
	ContentHash string    `gorm:"not null;uniqueIndex:idx_user_content_hash" json:"contentHash"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Language    string    `json:"language"`
	Content     string    `json:"-"`
	Deleted     bool      `json:"deleted"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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

// PinnedSentence tracks whether a user has pinned a sentence in a given
// language, the same way PinnedWord tracks pinned words. Unpinning sets
// Pinned to false rather than deleting the row, so the UpdatedAt timestamp
// survives for last-write-wins conflict resolution.
type PinnedSentence struct {
	UserID       string    `gorm:"primaryKey" json:"-"`
	LanguageCode string    `gorm:"primaryKey" json:"languageCode"`
	Sentence     string    `gorm:"primaryKey" json:"sentence"`
	Pinned       bool      `json:"pinned"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// UserSettings holds a user's global app preferences (not tied to any
// particular book). One row per user.
type UserSettings struct {
	UserID                     string    `gorm:"primaryKey" json:"-"`
	AnimationsEnabled          bool      `json:"animationsEnabled"`
	AutoPlayWordAudio          bool      `json:"autoPlayWordAudio"`
	IsDarkMode                 bool      `json:"isDarkMode"`
	LastDictionaryLanguageCode string    `json:"lastDictionaryLanguageCode"`
	LastSpanishVoiceRegion     string    `json:"lastSpanishVoiceRegion"`
	UpdatedAt                  time.Time `json:"updatedAt"`
}

// BookMetadataOverride holds a user's display overrides for one book
// (title/author/language corrections, reading font, voice choice),
// identified by Book.ContentHash.
type BookMetadataOverride struct {
	UserID                 string    `gorm:"primaryKey" json:"-"`
	ContentHash            string    `gorm:"primaryKey" json:"contentHash"`
	Title                  string    `json:"title"`
	Author                 string    `json:"author"`
	LanguageCode           string    `json:"languageCode"`
	DictionaryLanguageCode string    `json:"dictionaryLanguageCode"`
	FontFamily             string    `json:"fontFamily"`
	SpanishVoiceRegion     string    `json:"spanishVoiceRegion"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

// NavigationState is which screen a user was looking at: the focused row
// (a book's content hash, "library", or "settings"), which books were open
// in the swipe grid, and which library-grid page was showing. One row per
// user. A book's page *within* a book is deliberately not stored here -
// that's ReadingProgress's job, since a ReaderPage.id is viewport-dependent
// and meaningless on another device, while a library-grid page index is
// portable as-is.
type NavigationState struct {
	UserID            string    `gorm:"primaryKey" json:"-"`
	ActiveRowID       string    `json:"activeRowId"`
	OpenContentHashes string    `json:"openContentHashes" gorm:"type:text"`
	LibraryPageID     string    `json:"libraryPageId"`
	SettingsPageID    string    `json:"settingsPageId"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// All returns every model that should be included in auto-migration.
func All() []interface{} {
	return []interface{}{
		&User{},
		&Session{},
		&Book{},
		&ReadingProgress{},
		&PinnedWord{},
		&PinnedSentence{},
		&UserSettings{},
		&BookMetadataOverride{},
		&NavigationState{},
	}
}
