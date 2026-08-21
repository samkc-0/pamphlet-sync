package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/samkc-0/pamphlet-sync/internal/auth"
	"github.com/samkc-0/pamphlet-sync/internal/config"
	"github.com/samkc-0/pamphlet-sync/internal/middleware"
	"github.com/samkc-0/pamphlet-sync/internal/models"
)

const (
	oauthStateCookie = "pamphlet_oauth_state"
	sessionTTL       = 30 * 24 * time.Hour
)

type AuthHandler struct {
	DB          *gorm.DB
	OAuthConfig *oauth2.Config
	FrontendURL string
}

func NewAuthHandler(db *gorm.DB, cfg config.Config) *AuthHandler {
	return &AuthHandler{
		DB:          db,
		OAuthConfig: auth.NewGoogleOAuthConfig(cfg),
		FrontendURL: cfg.FrontendURL,
	}
}

// GoogleLogin redirects the browser to Google's consent screen.
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	state, err := auth.NewRandomToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start login"})
		return
	}

	c.SetCookie(oauthStateCookie, state, 600, "/", "", false, true)
	c.Redirect(http.StatusFound, h.OAuthConfig.AuthCodeURL(state))
}

// GoogleCallback completes the login, issues a session, and redirects back
// to the frontend with the session token.
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	expectedState, err := c.Cookie(oauthStateCookie)
	if err != nil || expectedState == "" || c.Query("state") != expectedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}
	c.SetCookie(oauthStateCookie, "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	googleUser, err := auth.FetchGoogleUser(c.Request.Context(), h.OAuthConfig, code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "google login failed"})
		return
	}

	user, err := h.upsertUser(googleUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save user"})
		return
	}

	token, err := auth.NewRandomToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start session"})
		return
	}

	session := models.Session{
		Token:     token,
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	if err := h.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
		return
	}

	c.Redirect(http.StatusFound, h.FrontendURL+"/?session="+token)
}

func (h *AuthHandler) upsertUser(googleUser *auth.GoogleUser) (models.User, error) {
	var user models.User
	err := h.DB.First(&user, "google_sub = ?", googleUser.Sub).Error

	if err == nil {
		if user.Name != googleUser.Name || user.Email != googleUser.Email {
			user.Name = googleUser.Name
			user.Email = googleUser.Email
			h.DB.Save(&user)
		}
		return user, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, err
	}

	user = models.User{
		ID:        uuid.NewString(),
		GoogleSub: googleUser.Sub,
		Email:     googleUser.Email,
		Name:      googleUser.Name,
		CreatedAt: time.Now(),
	}
	if err := h.DB.Create(&user).Error; err != nil {
		return models.User{}, err
	}

	return user, nil
}

// Me returns the currently authenticated user.
func (h *AuthHandler) Me(c *gin.Context) {
	user := c.MustGet(middleware.CurrentUserKey).(models.User)
	c.JSON(http.StatusOK, user)
}

// Logout deletes the current session so its token can no longer be used.
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.MustGet(middleware.CurrentTokenKey).(string)
	h.DB.Delete(&models.Session{}, "token = ?", token)
	c.Status(http.StatusNoContent)
}
