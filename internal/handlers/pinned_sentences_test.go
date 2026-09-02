package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/samkc-0/pamphlet-sync/internal/models"
)

func TestPinnedSentenceHandler_SetAndList(t *testing.T) {
	db := newTestDB(t)
	h := NewPinnedSentenceHandler(db)
	user := testUser("u1")

	c, w := newTestContext(user, http.MethodPost, "/pinned-sentences", setPinnedSentenceRequest{
		LanguageCode: "fr", Sentence: "Il y avait un homme.", Pinned: true, UpdatedAt: time.Now(),
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/pinned-sentences", nil)
	invoke(c, h.List)
	var sentences []models.PinnedSentence
	json.Unmarshal(w.Body.Bytes(), &sentences)
	if len(sentences) != 1 || sentences[0].Sentence != "Il y avait un homme." {
		t.Fatalf("unexpected pinned sentences: %+v", sentences)
	}
}

func TestPinnedSentenceHandler_UnpinUpdatesRowRatherThanDeleting(t *testing.T) {
	db := newTestDB(t)
	h := NewPinnedSentenceHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/pinned-sentences", setPinnedSentenceRequest{
		LanguageCode: "fr", Sentence: "Bonjour.", Pinned: true, UpdatedAt: now,
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/pinned-sentences", setPinnedSentenceRequest{
		LanguageCode: "fr", Sentence: "Bonjour.", Pinned: false, UpdatedAt: now.Add(time.Minute),
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.PinnedSentence
	err := db.Where("user_id = ? AND language_code = ? AND sentence = ?", user.ID, "fr", "Bonjour.").First(&stored).Error
	if err != nil {
		t.Fatalf("expected row to still exist after unpinning, got error: %v", err)
	}
	if stored.Pinned {
		t.Error("expected Pinned to be false after unpinning")
	}

	c, w = newTestContext(user, http.MethodGet, "/pinned-sentences", nil)
	invoke(c, h.List)
	var sentences []models.PinnedSentence
	json.Unmarshal(w.Body.Bytes(), &sentences)
	if len(sentences) != 0 {
		t.Errorf("expected List to exclude unpinned sentences, got %+v", sentences)
	}
}

func TestPinnedSentenceHandler_Set_OlderWriteIsIgnored(t *testing.T) {
	db := newTestDB(t)
	h := NewPinnedSentenceHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/pinned-sentences", setPinnedSentenceRequest{
		LanguageCode: "fr", Sentence: "Bonjour.", Pinned: false, UpdatedAt: now,
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// A stale "pin" arriving after a newer "unpin" must not resurrect it.
	c, w = newTestContext(user, http.MethodPost, "/pinned-sentences", setPinnedSentenceRequest{
		LanguageCode: "fr", Sentence: "Bonjour.", Pinned: true, UpdatedAt: now.Add(-time.Hour),
	})
	invoke(c, h.Set)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.PinnedSentence
	db.Where("user_id = ? AND language_code = ? AND sentence = ?", user.ID, "fr", "Bonjour.").First(&stored)
	if stored.Pinned {
		t.Error("stale pin should not have overwritten the newer unpin")
	}
}
