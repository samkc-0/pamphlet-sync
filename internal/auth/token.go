package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// NewRandomToken returns a cryptographically random hex string, used for
// both the OAuth CSRF state value and session bearer tokens.
func NewRandomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
