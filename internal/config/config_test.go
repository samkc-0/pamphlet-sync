package config

import "testing"

func TestValidate_DebugModeAlwaysPasses(t *testing.T) {
	cfg := Config{GinMode: "debug"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("debug mode should never fail validation, got: %v", err)
	}
}

func TestValidate_ReleaseModeRejectsDevDefaults(t *testing.T) {
	cfg := Config{
		GinMode:            "release",
		DatabaseURL:        "postgres://user:pass@localhost:5433/db",
		FrontendURL:        "http://localhost:5173",
		GoogleClientID:     "",
		GoogleClientSecret: "",
		GoogleRedirectURL:  "http://localhost:8080/auth/google/callback",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected release mode with missing secrets and localhost URLs to fail validation")
	}
}

func TestValidate_ReleaseModeAcceptsRealConfig(t *testing.T) {
	cfg := Config{
		GinMode:            "release",
		DatabaseURL:        "postgres://user:pass@postgres:5432/db",
		FrontendURL:        "https://pamphlet-sandbox.netlify.app",
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRedirectURL:  "https://pamphlet-sync.pug.homes/auth/google/callback",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected fully-configured release config to pass validation, got: %v", err)
	}
}
