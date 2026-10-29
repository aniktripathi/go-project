// Package config reads application settings from environment variables
// (optionally loaded from a .env file during development).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds every setting the server needs.
type Config struct {
	Port                string
	GinMode             string
	DatabaseURL         string
	JWTSecret           string
	JWTExpiryHours      int
	AllowedOrigins      []string
	AuthRateLimitPerMin int
	MLServiceURL        string
}

// Load reads the environment and validates required values.
func Load() (*Config, error) {
	// A missing .env file is fine (e.g. in Docker where real env vars are set).
	_ = godotenv.Load()

	cfg := &Config{
		Port:         getEnv("APP_PORT", "8080"),
		GinMode:      getEnv("GIN_MODE", "debug"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		MLServiceURL: strings.TrimRight(os.Getenv("ML_SERVICE_URL"), "/"),
	}

	var err error
	if cfg.JWTExpiryHours, err = getInt("JWT_EXPIRY_HOURS", 24); err != nil {
		return nil, err
	}
	if cfg.AuthRateLimitPerMin, err = getInt("AUTH_RATE_LIMIT_PER_MIN", 10); err != nil {
		return nil, err
	}
	for _, origin := range strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, origin)
		}
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set (see backend/.env.example)")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be set and at least 32 characters long")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative whole number", key)
	}
	return n, nil
}
