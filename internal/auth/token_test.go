package auth

import "testing"

func TestNewRandomToken(t *testing.T) {
	a, err := NewRandomToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a) != 64 {
		t.Errorf("expected a 64-character hex string (32 random bytes), got %d chars", len(a))
	}

	b, err := NewRandomToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == b {
		t.Error("expected two calls to produce different tokens")
	}
}
