package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/samkc-0/pamphlet-sync/internal/middleware"
	"github.com/samkc-0/pamphlet-sync/internal/models"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(models.All()...); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

// newTestContext builds a gin context for the given user with an optional
// JSON request body. Pass a nil body for GET-style requests.
func newTestContext(user models.User, method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		encoded, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(encoded))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req
	c.Set(middleware.CurrentUserKey, user)
	return c, w
}

func testUser(id string) models.User {
	return models.User{ID: id, GoogleSub: "sub-" + id, Email: id + "@example.com"}
}

// invoke runs a handler directly (bypassing gin's engine) and forces its
// deferred response header to flush. Handlers that respond via c.JSON write
// a body immediately, which already forces this - but a handler that only
// calls c.Status() (no body) defers the actual header write until gin's
// engine finishes the request, which never happens when a handler is
// called directly like this. Without the explicit flush, w.Code would
// stay at httptest's default 200 regardless of what c.Status() set it to.
func invoke(c *gin.Context, handler func(*gin.Context)) {
	handler(c)
	c.Writer.WriteHeaderNow()
}

func TestBookHandler_CreateAndGet(t *testing.T) {
	db := newTestDB(t)
	h := NewBookHandler(db)
	user := testUser("u1")

	body := createBookRequest{
		ContentHash: "hash1",
		Title:       "Le Colonel Chabert",
		Author:      "Balzac",
		Language:    "fr",
		Chapters:    []BookChapter{{ID: "ch1", Paragraphs: []string{"Il y avait..."}}},
	}

	c, w := newTestContext(user, http.MethodPost, "/books", body)
	invoke(c, h.Create)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 creating book, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/books/hash1", nil)
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Get)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 getting book, got %d: %s", w.Code, w.Body.String())
	}

	var resp bookContentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Title != "Le Colonel Chabert" || len(resp.Chapters) != 1 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestBookHandler_Create_IsIdempotentOnDuplicateHash(t *testing.T) {
	db := newTestDB(t)
	h := NewBookHandler(db)
	user := testUser("u1")

	body := createBookRequest{
		ContentHash: "hash1",
		Title:       "Original Title",
		Chapters:    []BookChapter{{ID: "ch1", Paragraphs: []string{"text"}}},
	}

	c, w := newTestContext(user, http.MethodPost, "/books", body)
	invoke(c, h.Create)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on first create, got %d", w.Code)
	}

	body.Title = "Different Title"
	c, w = newTestContext(user, http.MethodPost, "/books", body)
	invoke(c, h.Create)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on duplicate create, got %d", w.Code)
	}

	var count int64
	db.Model(&models.Book{}).Where("user_id = ? AND content_hash = ?", user.ID, "hash1").Count(&count)
	if count != 1 {
		t.Errorf("expected exactly one stored book, got %d", count)
	}
}

func TestBookHandler_List_ExcludesContentAndOtherUsers(t *testing.T) {
	db := newTestDB(t)
	h := NewBookHandler(db)
	userA := testUser("a")
	userB := testUser("b")

	create := func(user models.User, hash string) {
		c, w := newTestContext(user, http.MethodPost, "/books", createBookRequest{
			ContentHash: hash,
			Title:       "Book " + hash,
			Chapters:    []BookChapter{{ID: "ch1", Paragraphs: []string{"secret content"}}},
		})
		invoke(c, h.Create)
		if w.Code != http.StatusNoContent {
			t.Fatalf("seed create failed: %d", w.Code)
		}
	}
	create(userA, "hashA")
	create(userB, "hashB")

	c, w := newTestContext(userA, http.MethodGet, "/books", nil)
	invoke(c, h.List)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var books []models.Book
	if err := json.Unmarshal(w.Body.Bytes(), &books); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(books) != 1 || books[0].ContentHash != "hashA" {
		t.Fatalf("expected only userA's book, got %+v", books)
	}
	if books[0].Content != "" {
		t.Error("expected List to never include Content")
	}
}

func TestBookHandler_Get_RejectsOtherUsersBook(t *testing.T) {
	db := newTestDB(t)
	h := NewBookHandler(db)
	owner := testUser("owner")
	attacker := testUser("attacker")

	c, w := newTestContext(owner, http.MethodPost, "/books", createBookRequest{
		ContentHash: "private-hash",
		Title:       "Private Book",
		Chapters:    []BookChapter{{ID: "ch1", Paragraphs: []string{"secret"}}},
	})
	invoke(c, h.Create)
	if w.Code != http.StatusNoContent {
		t.Fatalf("seed create failed: %d", w.Code)
	}

	c, w = newTestContext(attacker, http.MethodGet, "/books/private-hash", nil)
	c.Params = gin.Params{{Key: "hash", Value: "private-hash"}}
	invoke(c, h.Get)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for another user's book, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBookHandler_Get_NotFoundForUnknownHash(t *testing.T) {
	db := newTestDB(t)
	h := NewBookHandler(db)

	c, w := newTestContext(testUser("u1"), http.MethodGet, "/books/does-not-exist", nil)
	c.Params = gin.Params{{Key: "hash", Value: "does-not-exist"}}
	invoke(c, h.Get)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestBookHandler_Delete_MarksDeletedAndListReflectsIt(t *testing.T) {
	db := newTestDB(t)
	h := NewBookHandler(db)
	user := testUser("u1")

	c, w := newTestContext(user, http.MethodPost, "/books", createBookRequest{
		ContentHash: "hash1",
		Title:       "A Book",
		Chapters:    []BookChapter{{ID: "ch1", Paragraphs: []string{"text"}}},
	})
	invoke(c, h.Create)
	if w.Code != http.StatusNoContent {
		t.Fatalf("seed create failed: %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/books/hash1/delete", deleteBookRequest{
		UpdatedAt: time.Now(),
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Delete)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting book, got %d: %s", w.Code, w.Body.String())
	}

	c, w = newTestContext(user, http.MethodGet, "/books", nil)
	invoke(c, h.List)
	var books []models.Book
	json.Unmarshal(w.Body.Bytes(), &books)
	if len(books) != 1 || !books[0].Deleted {
		t.Fatalf("expected List to positively confirm the deletion, got %+v", books)
	}

	c, w = newTestContext(user, http.MethodGet, "/books/hash1", nil)
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Get)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected Get to reject a deleted book, got %d", w.Code)
	}
}

func TestBookHandler_Delete_OlderDeleteIsIgnored(t *testing.T) {
	db := newTestDB(t)
	h := NewBookHandler(db)
	user := testUser("u1")
	now := time.Now()

	c, w := newTestContext(user, http.MethodPost, "/books", createBookRequest{
		ContentHash: "hash1",
		Chapters:    []BookChapter{{ID: "ch1", Paragraphs: []string{"text"}}},
	})
	invoke(c, h.Create)
	if w.Code != http.StatusNoContent {
		t.Fatalf("seed create failed: %d", w.Code)
	}

	c, w = newTestContext(user, http.MethodPost, "/books/hash1/delete", deleteBookRequest{
		UpdatedAt: now.Add(-time.Hour),
	})
	c.Params = gin.Params{{Key: "hash", Value: "hash1"}}
	invoke(c, h.Delete)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	var stored models.Book
	db.Where("user_id = ? AND content_hash = ?", user.ID, "hash1").First(&stored)
	if stored.Deleted {
		t.Error("a delete request older than the book's creation should be ignored")
	}
}

func TestBookHandler_Delete_NonexistentBookIsNoOp(t *testing.T) {
	db := newTestDB(t)
	h := NewBookHandler(db)

	c, w := newTestContext(testUser("u1"), http.MethodPost, "/books/does-not-exist/delete", deleteBookRequest{
		UpdatedAt: time.Now(),
	})
	c.Params = gin.Params{{Key: "hash", Value: "does-not-exist"}}
	invoke(c, h.Delete)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 no-op for a nonexistent book, got %d", w.Code)
	}
}
