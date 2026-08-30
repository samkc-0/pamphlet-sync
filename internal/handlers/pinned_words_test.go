package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/samkc-0/pamphlet-sync/internal/models"
)

func TestPinnedWordHandler_SetAndList(t *testing.T) {
	db := newTestDB(t)
	h := NewPinnedWordHandler(db)
	user := testUser("u1")

	c, w := newTestContext(user, http.MethodPost, "/pinned-words", setPinnedWordRequest{
		LanguageCode: "fr", Word: "chagrin", Pinned: true, UpdatedAt: time.Now(),
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/pinned-words", nil)
	invoke(c, h.List)
	var words []models.PinnedWord
	json.Unmarshal(w.Body.Bytes(), &words)
	if len(words) != 1 || words[0].Word != "chagrin" {
		t.Fatalf("unexpected pinned words: %+v", words)
	}
}

func TestPinnedWordHandler_UnpinUpdatesRowRatherThanDeleting(t *testing.T) {
	db := newTestDB(t)
	h := NewPinnedWordHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/pinned-words", setPinnedWordRequest{
		LanguageCode: "fr", Word: "chagrin", Pinned: true, UpdatedAt: now,
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/pinned-words", setPinnedWordRequest{
		LanguageCode: "fr", Word: "chagrin", Pinned: false, UpdatedAt: now.Add(time.Minute),
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.PinnedWord
	err := db.Where("user_id = ? AND language_code = ? AND word = ?", user.ID, "fr", "chagrin").First(&stored).Error
	if err != nil {
		t.Fatalf("expected row to still exist after unpinning, got error: %v", err)
	}
	if stored.Pinned {
		t.Error("expected Pinned to be false after unpinning")
	}

	c, w = newTestContext(user, http.MethodGet, "/pinned-words", nil)
	invoke(c, h.List)
	var words []models.PinnedWord
	json.Unmarshal(w.Body.Bytes(), &words)
	if len(words) != 0 {
		t.Errorf("expected List to exclude unpinned words, got %+v", words)
	}
}

func TestPinnedWordHandler_Set_OlderWriteIsIgnored(t *testing.T) {
	db := newTestDB(t)
	h := NewPinnedWordHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/pinned-words", setPinnedWordRequest{
		LanguageCode: "fr", Word: "chagrin", Pinned: false, UpdatedAt: now,
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// A stale "pin" arriving after a newer "unpin" must not resurrect it.
	c, w = newTestContext(user, http.MethodPost, "/pinned-words", setPinnedWordRequest{
		LanguageCode: "fr", Word: "chagrin", Pinned: true, UpdatedAt: now.Add(-time.Hour),
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.PinnedWord
	db.Where("user_id = ? AND language_code = ? AND word = ?", user.ID, "fr", "chagrin").First(&stored)
	if stored.Pinned {
		t.Error("stale pin should not have overwritten the newer unpin")
	}
}
