// Package config loads application configuration from environment variables.
package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DatabaseURL        string
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
		GinMode:            getEnv("GIN_MODE", "debug"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
