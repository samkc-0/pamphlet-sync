package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samkc-0/pamphlet-sync/internal/models"
)

func TestProgressHandler_UpsertAndList(t *testing.T) {
	db := newTestDB(t)
	h := NewProgressHandler(db)
	user := testUser("u1")

	c, w := newTestContext(user, http.MethodPost, "/progress/hash1", upsertProgressRequest{
		ChapterID:      "ch1",
		ParagraphIndex: 3,
		UpdatedAt:      time.Now(),
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/progress", nil)
	invoke(c, h.List)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var progress []models.ReadingProgress
	if err := json.Unmarshal(w.Body.Bytes(), &progress); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(progress) != 1 || progress[0].ParagraphIndex != 3 {
		t.Fatalf("unexpected progress list: %+v", progress)
	}
}

func TestProgressHandler_Upsert_OlderWriteIsIgnored(t *testing.T) {
	db := newTestDB(t)
	h := NewProgressHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/progress/hash1", upsertProgressRequest{
		ChapterID: "ch5", ParagraphIndex: 10, UpdatedAt: now,
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// An older write should be silently ignored, not overwrite the newer one.
	c, w = newTestContext(user, http.MethodPost, "/progress/hash1", upsertProgressRequest{
		ChapterID: "ch1", ParagraphIndex: 0, UpdatedAt: now.Add(-time.Hour),
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.ReadingProgress
	db.Where("user_id = ? AND content_hash = ?", user.ID, "hash1").First(&stored)
	if stored.ChapterID != "ch5" || stored.ParagraphIndex != 10 {
		t.Errorf("older write should not have overwritten newer progress, got %+v", stored)
	}
}

func TestProgressHandler_Upsert_NewerWriteWins(t *testing.T) {
	db := newTestDB(t)
	h := NewProgressHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/progress/hash1", upsertProgressRequest{
		ChapterID: "ch1", ParagraphIndex: 0, UpdatedAt: now,
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/progress/hash1", upsertProgressRequest{
		ChapterID: "ch5", ParagraphIndex: 10, UpdatedAt: now.Add(time.Hour),
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.ReadingProgress
	db.Where("user_id = ? AND content_hash = ?", user.ID, "hash1").First(&stored)
	if stored.ChapterID != "ch5" || stored.ParagraphIndex != 10 {
		t.Errorf("newer write should have won, got %+v", stored)
	}
}

func TestProgressHandler_List_ScopedToUser(t *testing.T) {
	db := newTestDB(t)
	h := NewProgressHandler(db)
	userA := testUser("a")
	userB := testUser("b")

	for _, u := range []models.User{userA, userB} {
		c, w := newTestContext(u, http.MethodPost, "/progress/hash1", upsertProgressRequest{
			ChapterID: "ch1", ParagraphIndex: 1, UpdatedAt: time.Now(),
		})
		c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
		invoke(c, h.Upsert)
		if w.Code != http.StatusNoContent {
			t.Fatalf("seed upsert failed: %d", w.Code)
		}
	}

	c, w := newTestContext(userA, http.MethodGet, "/progress", nil)
	invoke(c, h.List)
	var progress []models.ReadingProgress
	json.Unmarshal(w.Body.Bytes(), &progress)
	if len(progress) != 1 {
		t.Fatalf("expected only userA's single progress record, got %d: %+v", len(progress), progress)
	}

	var countForB int64
	db.Model(&models.ReadingProgress{}).Where("user_id = ?", userB.ID).Count(&countForB)
	if countForB != 1 {
		t.Fatalf("expected userB's own row to still exist untouched, count=%d", countForB)
	}
}
