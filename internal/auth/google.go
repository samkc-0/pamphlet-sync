// Package auth handles Google OAuth login and session tokens.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/samkc-0/pamphlet-sync/internal/config"
)

const googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

// GoogleUser is the subset of Google's userinfo response we care about.
type GoogleUser struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// NewGoogleOAuthConfig builds an oauth2.Config for the Google login flow.
func NewGoogleOAuthConfig(cfg config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// FetchGoogleUser exchanges an authorization code for a token and fetches
// the authenticated user's Google profile.
func FetchGoogleUser(ctx context.Context, oauthConfig *oauth2.Config, code string) (*GoogleUser, error) {
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	client := oauthConfig.Client(ctx, token)
	resp, err := client.Get(googleUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo status %d: %s", resp.StatusCode, body)
	}

	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}

	if user.Sub == "" {
		return nil, fmt.Errorf("userinfo response missing sub")
	}

	return &user, nil
}
