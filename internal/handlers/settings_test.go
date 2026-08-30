package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/samkc-0/pamphlet-sync/internal/models"
)

func TestSettingsHandler_UpsertAndGet(t *testing.T) {
	db := newTestDB(t)
	h := NewSettingsHandler(db)
	user := testUser("u1")

	c, w := newTestContext(user, http.MethodPost, "/settings", upsertSettingsRequest{
		AnimationsEnabled:          false,
		IsDarkMode:                 true,
		LastDictionaryLanguageCode: "fr",
		LastSpanishVoiceRegion:     "es-MX",
		UpdatedAt:                  time.Now(),
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/settings", nil)
	invoke(c, h.Get)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var settings models.UserSettings
	json.Unmarshal(w.Body.Bytes(), &settings)
	if !settings.IsDarkMode || settings.LastDictionaryLanguageCode != "fr" {
		t.Errorf("unexpected settings: %+v", settings)
	}
}

func TestSettingsHandler_Get_NotFoundWhenNeverSaved(t *testing.T) {
	db := newTestDB(t)
	h := NewSettingsHandler(db)

	c, w := newTestContext(testUser("u1"), http.MethodGet, "/settings", nil)
	invoke(c, h.Get)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestSettingsHandler_Upsert_OlderWriteIsIgnored(t *testing.T) {
	db := newTestDB(t)
	h := NewSettingsHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/settings", upsertSettingsRequest{
		IsDarkMode: true,
		UpdatedAt:  now,
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/settings", upsertSettingsRequest{
		IsDarkMode: false,
		UpdatedAt:  now.Add(-time.Hour),
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.UserSettings
	db.Where("user_id = ?", user.ID).First(&stored)
	if !stored.IsDarkMode {
		t.Error("older write should not have overwritten newer settings")
	}
}
