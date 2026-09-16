package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samkc-0/pamphlet-sync/internal/models"
)

func TestNotebookHandler_UpsertAndGet(t *testing.T) {
	db := newTestDB(t)
	h := NewNotebookHandler(db)
	user := testUser("u1")

	body := upsertNotebookRequest{
		ID:           "nb1",
		Title:        "Vocabulary",
		Content:      "le chat\n\nla maison",
		LanguageCode: "fr",
		FontFamily:   "serif",
		UpdatedAt:    time.Now(),
	}

	c, w := newTestContext(user, http.MethodPost, "/notebooks", body)
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 creating notebook, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/notebooks/nb1", nil)
	c.Params = gin.Params{{Key: "id", Value: "nb1"}}
	invoke(c, h.Get)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 getting notebook, got %d: %s", w.Code, w.Body.String())
	}

	var resp notebookContentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Title != "Vocabulary" || resp.Content != "le chat\n\nla maison" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestNotebookHandler_Upsert_UpdatesExistingWhenNewer(t *testing.T) {
	db := newTestDB(t)
	h := NewNotebookHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/notebooks", upsertNotebookRequest{
		ID:        "nb1",
		Title:     "Original",
		Content:   "first draft",
		UpdatedAt: now,
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("seed create failed: %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/notebooks", upsertNotebookRequest{
		ID:        "nb1",
		Title:     "Updated",
		Content:   "second draft",
		UpdatedAt: now.Add(time.Minute),
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on update, got %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodGet, "/notebooks/nb1", nil)
	c.Params = gin.Params{{Key: "id", Value: "nb1"}}
	invoke(c, h.Get)

	var resp notebookContentResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Title != "Updated" || resp.Content != "second draft" {
		t.Errorf("expected the newer write to win, got %+v", resp)
	}
}

func TestNotebookHandler_Upsert_OlderWriteIsIgnored(t *testing.T) {
	db := newTestDB(t)
	h := NewNotebookHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/notebooks", upsertNotebookRequest{
		ID:        "nb1",
		Title:     "Current",
		Content:   "current draft",
		UpdatedAt: now,
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("seed create failed: %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/notebooks", upsertNotebookRequest{
		ID:        "nb1",
		Title:     "Stale",
		Content:   "stale draft",
		UpdatedAt: now.Add(-time.Hour),
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.Notebook
	db.Where("user_id = ? AND id = ?", user.ID, "nb1").First(&stored)
	if stored.Title != "Current" {
		t.Errorf("an older write should have been ignored, got title %q", stored.Title)
	}
}

func TestNotebookHandler_List_ExcludesContentAndOtherUsers(t *testing.T) {
	db := newTestDB(t)
	h := NewNotebookHandler(db)
	userA := testUser("a")
	userB := testUser("b")

	create := func(user models.User, id string) {
		c, w := newTestContext(user, http.MethodPost, "/notebooks", upsertNotebookRequest{
			ID:        id,
			Title:     "Notebook " + id,
			Content:   "secret content",
			UpdatedAt: time.Now(),
		})
		invoke(c, h.Upsert)
		if w.Code != http.StatusNoContent {
			t.Fatalf("seed create failed: %d", w.Code)
		}
	}
	create(userA, "nbA")
	create(userB, "nbB")

	c, w := newTestContext(userA, http.MethodGet, "/notebooks", nil)
	invoke(c, h.List)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var notebooks []models.Notebook
	if err := json.Unmarshal(w.Body.Bytes(), &notebooks); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(notebooks) != 1 || notebooks[0].ID != "nbA" {
		t.Fatalf("expected only userA's notebook, got %+v", notebooks)
	}
	if notebooks[0].Content != "" {
		t.Error("expected List to never include Content")
	}
}

func TestNotebookHandler_Get_RejectsOtherUsersNotebook(t *testing.T) {
	db := newTestDB(t)
	h := NewNotebookHandler(db)
	owner := testUser("owner")
	attacker := testUser("attacker")

	c, w := newTestContext(owner, http.MethodPost, "/notebooks", upsertNotebookRequest{
		ID:        "private-nb",
		Title:     "Private",
		Content:   "secret",
		UpdatedAt: time.Now(),
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("seed create failed: %d", w.Code)
	}

	c, w = newTestContext(attacker, http.MethodGet, "/notebooks/private-nb", nil)
	c.Params = gin.Params{{Key: "id", Value: "private-nb"}}
	invoke(c, h.Get)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for another user's notebook, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotebookHandler_Get_NotFoundForUnknownID(t *testing.T) {
	db := newTestDB(t)
	h := NewNotebookHandler(db)

	c, w := newTestContext(testUser("u1"), http.MethodGet, "/notebooks/does-not-exist", nil)
	c.Params = gin.Params{{Key: "id", Value: "does-not-exist"}}
	invoke(c, h.Get)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestNotebookHandler_Delete_MarksDeletedAndListReflectsIt(t *testing.T) {
	db := newTestDB(t)
	h := NewNotebookHandler(db)
	user := testUser("u1")

	c, w := newTestContext(user, http.MethodPost, "/notebooks", upsertNotebookRequest{
		ID:        "nb1",
		Title:     "A Notebook",
		Content:   "text",
		UpdatedAt: time.Now(),
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("seed create failed: %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/notebooks/nb1/delete", deleteNotebookRequest{
		UpdatedAt: time.Now(),
	})
	c.Params = gin.Params{{Key: "id", Value: "nb1"}}
	invoke(c, h.Delete)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting notebook, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/notebooks", nil)
	invoke(c, h.List)
	var notebooks []models.Notebook
	json.Unmarshal(w.Body.Bytes(), &notebooks)
	if len(notebooks) != 1 || !notebooks[0].Deleted {
		t.Fatalf("expected List to positively confirm the deletion, got %+v", notebooks)
	}

	c, w = newTestContext(user, http.MethodGet, "/notebooks/nb1", nil)
	c.Params = gin.Params{{Key: "id", Value: "nb1"}}
	invoke(c, h.Get)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected Get to reject a deleted notebook, got %d", w.Code)
	}
}

func TestNotebookHandler_Delete_OlderDeleteIsIgnored(t *testing.T) {
	db := newTestDB(t)
	h := NewNotebookHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/notebooks", upsertNotebookRequest{
		ID:        "nb1",
		Content:   "text",
		UpdatedAt: now,
	})
	invoke(c, h.Upsert)
	if w.Code != http.StatusNoContent {
		t.Fatalf("seed create failed: %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/notebooks/nb1/delete", deleteNotebookRequest{
		UpdatedAt: now.Add(-time.Hour),
	})
	c.Params = gin.Params{{Key: "id", Value: "nb1"}}
	invoke(c, h.Delete)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.Notebook
	db.Where("user_id = ? AND id = ?", user.ID, "nb1").First(&stored)
	if stored.Deleted {
		t.Error("a delete request older than the notebook's creation should be ignored")
	}
}

func TestNotebookHandler_Delete_NonexistentNotebookIsNoOp(t *testing.T) {
	db := newTestDB(t)
	h := NewNotebookHandler(db)

	c, w := newTestContext(testUser("u1"), http.MethodPost, "/notebooks/does-not-exist/delete", deleteNotebookRequest{
		UpdatedAt: time.Now(),
	})
	c.Params = gin.Params{{Key: "id", Value: "does-not-exist"}}
	invoke(c, h.Delete)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 no-op for a nonexistent notebook, got %d", w.Code)
	}
}
