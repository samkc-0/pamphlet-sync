package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestNavigationHandler_UpsertAndGet(t *testing.T) {
	db := newTestDB(t)
	h := NewNavigationHandler(db)
	user := testUser("u1")

	c, w := newTestContext(user, http.MethodPost, "/navigation", upsertNavigationStateRequest{
		ActiveRowID:       "hash1",
		OpenContentHashes: []string{"hash1", "hash2"},
		LibraryPageID:     "books-2",
		SettingsPageID:    "main",
		UpdatedAt:         time.Now(),
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/navigation", nil)
	invoke(c, h.Get)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var state navigationStateResponse
	json.Unmarshal(w.Body.Bytes(), &state)
	if state.ActiveRowID != "hash1" || len(state.OpenContentHashes) != 2 {
		t.Errorf("unexpected navigation state: %+v", state)
	}
}

func TestNavigationHandler_Get_NotFoundWhenNeverSaved(t *testing.T) {
	db := newTestDB(t)
	h := NewNavigationHandler(db)

	c, w := newTestContext(testUser("u1"), http.MethodGet, "/navigation", nil)
	invoke(c, h.Get)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestNavigationHandler_Upsert_OlderWriteIsIgnored(t *testing.T) {
	db := newTestDB(t)
	h := NewNavigationHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/navigation", upsertNavigationStateRequest{
		ActiveRowID: "hash1", UpdatedAt: now,
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/navigation", upsertNavigationStateRequest{
		ActiveRowID: "library", UpdatedAt: now.Add(-time.Hour),
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodGet, "/navigation", nil)
	invoke(c, h.Get)
	var state navigationStateResponse
	json.Unmarshal(w.Body.Bytes(), &state)
	if state.ActiveRowID != "hash1" {
		t.Errorf("older write should not have overwritten newer state, got %q", state.ActiveRowID)
	}
}
