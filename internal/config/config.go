// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// releaseMode mirrors gin.ReleaseMode without importing gin here, since
// config is meant to stay dependency-free.
const releaseMode = "release"

type Config struct {
	Port               string
	DatabaseURL        string
	DictionariesDir    string
	GinMode            string
	FrontendURL        string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

// Load reads a .env file if present (ignored if missing) and returns
// a Config populated from environment variables, falling back to defaults.
func Load() Config {
	_ = godotenv.Load()

	return Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/pamphlet_sync?sslmode=disable"),
		DictionariesDir:    getEnv("DICTIONARIES_DIR", "./dictionaries"),
		GinMode:            getEnv("GIN_MODE", "debug"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
	}
}

// Validate rejects a production configuration that still has dev-only
// defaults or missing secrets, so the server fails fast at boot instead of
// silently rejecting every login attempt at runtime.
func (c Config) Validate() error {
	if c.GinMode != releaseMode {
		return nil
	}

	var problems []string
	if c.GoogleClientID == "" {
		problems = append(problems, "GOOGLE_CLIENT_ID is required")
	}
	if c.GoogleClientSecret == "" {
		problems = append(problems, "GOOGLE_CLIENT_SECRET is required")
	}
	if strings.Contains(c.GoogleRedirectURL, "localhost") {
		problems = append(problems, "GOOGLE_REDIRECT_URL still points at localhost")
	}
	if strings.Contains(c.DatabaseURL, "localhost") {
		problems = append(problems, "DATABASE_URL still points at localhost")
	}
	if strings.Contains(c.FrontendURL, "localhost") {
		problems = append(problems, "FRONTEND_URL still points at localhost")
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid production config: %s", strings.Join(problems, "; "))
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
