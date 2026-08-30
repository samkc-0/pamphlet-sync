package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/samkc-0/pamphlet-sync/internal/config"
)

func newTestAuthHandler() *AuthHandler {
	return NewAuthHandler(nil, config.Config{GinMode: "debug"})
}

func TestGoogleCallback_RejectsMissingStateCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestAuthHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=anything&code=abc", nil)

	h.GoogleCallback(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 with no state cookie, got %d", w.Code)
	}
}

func TestGoogleCallback_RejectsMismatchedState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestAuthHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=wrong&code=abc", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "expected"})
	c.Request = req

	h.GoogleCallback(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on state mismatch, got %d", w.Code)
	}
}

func TestGoogleCallback_RejectsMissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestAuthHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=matching", nil)
	req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "matching"})
	c.Request = req

	h.GoogleCallback(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 with no code param, got %d", w.Code)
	}
}
