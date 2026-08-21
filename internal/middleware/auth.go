// Package middleware contains gin middleware.
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/samkc-0/pamphlet-sync/internal/models"
)

const (
	CurrentUserKey  = "currentUser"
	CurrentTokenKey = "currentSessionToken"
)

// RequireSession looks up the bearer token from the Authorization header,
// rejecting the request if it is missing, unknown, or expired. On success
// the authenticated user is stored on the gin context under CurrentUserKey.
func RequireSession(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		var session models.Session
		if err := db.First(&session, "token = ?", token).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}

		if time.Now().After(session.ExpiresAt) {
			db.Delete(&session)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return
		}

		var user models.User
		if err := db.First(&user, "id = ?", session.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}

		c.Set(CurrentUserKey, user)
		c.Set(CurrentTokenKey, token)
		c.Next()
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimPrefix(header, prefix)
}
