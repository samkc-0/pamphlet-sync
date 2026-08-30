package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samkc-0/pamphlet-sync/internal/models"
)

func TestBookMetadataHandler_UpsertAndList(t *testing.T) {
	db := newTestDB(t)
	h := NewBookMetadataHandler(db)
	user := testUser("u1")

	c, w := newTestContext(user, http.MethodPost, "/book-metadata/hash1", upsertBookMetadataRequest{
		Title:      "Custom Title",
		FontFamily: "serif",
		UpdatedAt:  time.Now(),
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/book-metadata", nil)
	invoke(c, h.List)
	var overrides []models.BookMetadataOverride
	json.Unmarshal(w.Body.Bytes(), &overrides)
	if len(overrides) != 1 || overrides[0].Title != "Custom Title" {
		t.Fatalf("unexpected overrides: %+v", overrides)
	}
}

func TestBookMetadataHandler_Upsert_OlderWriteIsIgnored(t *testing.T) {
	db := newTestDB(t)
	h := NewBookMetadataHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/book-metadata/hash1", upsertBookMetadataRequest{
		Title: "Newer Title", UpdatedAt: now,
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/book-metadata/hash1", upsertBookMetadataRequest{
		Title: "Stale Title", UpdatedAt: now.Add(-time.Hour),
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.BookMetadataOverride
	db.Where("user_id = ? AND content_hash = ?", user.ID, "hash1").First(&stored)
	if stored.Title != "Newer Title" {
		t.Errorf("older write should not have overwritten newer metadata, got %q", stored.Title)
	}
}

func TestBookMetadataHandler_List_ScopedToUser(t *testing.T) {
	db := newTestDB(t)
	h := NewBookMetadataHandler(db)
	userA := testUser("a")
	userB := testUser("b")

	for _, u := range []models.User{userA, userB} {
		c, w := newTestContext(u, http.MethodPost, "/book-metadata/hash1", upsertBookMetadataRequest{
			Title: "Title", UpdatedAt: time.Now(),
		})
		c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
		invoke(c, h.Upsert)
		if w.Code != http.StatusNoContent {
			t.Fatalf("seed upsert failed: %d", w.Code)
		}
	}

	c, w := newTestContext(userA, http.MethodGet, "/book-metadata", nil)
	invoke(c, h.List)
	var overrides []models.BookMetadataOverride
	json.Unmarshal(w.Body.Bytes(), &overrides)
	if len(overrides) != 1 {
		t.Fatalf("expected only userA's override, got %d: %+v", len(overrides), overrides)
	}
}
