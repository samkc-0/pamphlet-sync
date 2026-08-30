package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

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

func newTestContext(bearerToken string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/me", nil)
	if bearerToken != "" {
		c.Request.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	return c, w
}

func TestRequireSession_MissingToken(t *testing.T) {
	db := newTestDB(t)
	c, w := newTestContext("")

	RequireSession(db)(c)

	if !c.IsAborted() {
		t.Fatal("expected request to be aborted")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireSession_UnknownToken(t *testing.T) {
	db := newTestDB(t)
	c, w := newTestContext("does-not-exist")

	RequireSession(db)(c)

	if !c.IsAborted() {
		t.Fatal("expected request to be aborted")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireSession_ExpiredSessionIsRejectedAndDeleted(t *testing.T) {
	db := newTestDB(t)
	user := models.User{ID: "u1", GoogleSub: "sub1", Email: "a@example.com", CreatedAt: time.Now()}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	session := models.Session{
		Token:     "expired-token",
		UserID:    user.ID,
		CreatedAt: time.Now().Add(-time.Hour),
		ExpiresAt: time.Now().Add(-time.Minute),
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	c, w := newTestContext("expired-token")
	RequireSession(db)(c)

	if !c.IsAborted() {
		t.Fatal("expected request to be aborted")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	var remaining int64
	db.Model(&models.Session{}).Where("token = ?", "expired-token").Count(&remaining)
	if remaining != 0 {
		t.Error("expected the expired session row to be deleted")
	}
}

func TestRequireSession_ValidSessionSetsContext(t *testing.T) {
	db := newTestDB(t)
	user := models.User{ID: "u2", GoogleSub: "sub2", Email: "b@example.com", CreatedAt: time.Now()}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	session := models.Session{
		Token:     "valid-token",
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	c, w := newTestContext("valid-token")
	RequireSession(db)(c)

	if c.IsAborted() {
		t.Fatalf("expected request not to be aborted, got status %d", w.Code)
	}

	gotUser, ok := c.Get(CurrentUserKey)
	if !ok {
		t.Fatal("expected current user to be set in context")
	}
	if gotUser.(models.User).ID != user.ID {
		t.Errorf("expected user %s, got %s", user.ID, gotUser.(models.User).ID)
	}

	gotToken, ok := c.Get(CurrentTokenKey)
	if !ok || gotToken.(string) != "valid-token" {
		t.Errorf("expected token %q in context, got %v", "valid-token", gotToken)
	}
}
