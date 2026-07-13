// Package config loads runtime configuration from the environment. Secrets
// (database credentials, cookie security) never live in source control.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config carries everything the API needs at startup.
type Config struct {
	Port        int
	DatabaseURL string
	Env         string        // "development" or "production"
	SessionTTL  time.Duration // how long a login session lasts
	BcryptCost  int           // work factor for password hashing
	StaticDir   string        // optional: serve a built frontend from this dir
}

// Load reads configuration from environment variables, applying development
// defaults so `go run ./cmd/api` works against a local compose database.
func Load() (Config, error) {
	cfg := Config{
		Port:        envInt("PORT", 8080),
		DatabaseURL: envStr("DATABASE_URL", "postgres://devsignal:devsignal@localhost:5432/devsignal?sslmode=disable"),
		Env:         envStr("APP_ENV", "development"),
		SessionTTL:  time.Duration(envInt("SESSION_TTL_HOURS", 72)) * time.Hour,
		BcryptCost:  envInt("BCRYPT_COST", 12),
		StaticDir:   envStr("STATIC_DIR", ""),
	}

	if cfg.Env != "development" && cfg.Env != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development or production, got %q", cfg.Env)
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("PORT out of range: %d", cfg.Port)
	}
	if cfg.BcryptCost < 10 || cfg.BcryptCost > 16 {
		return Config{}, fmt.Errorf("BCRYPT_COST must be between 10 and 16, got %d", cfg.BcryptCost)
	}
	return cfg, nil
}

// Production reports whether the app runs with production hardening
// (Secure cookies, HSTS).
func (c Config) Production() bool { return c.Env == "production" }

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
