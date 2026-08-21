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

// All returns every model that should be included in auto-migration.
func All() []interface{} {
	return []interface{}{
		&User{},
		&Session{},
	}
}
