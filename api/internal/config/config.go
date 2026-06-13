package config

import (
	"log/slog"
	"os"
)

type Config struct {
	Port               string
	DatabasePoolerURL  string // PgBouncer — used by API handlers
	DatabaseDirectURL  string // Direct — used by River workers
	Environment        string
}

func Load() *Config {
	c := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabasePoolerURL: mustEnv("DATABASE_POOLER_URL"),
		DatabaseDirectURL: mustEnv("DATABASE_DIRECT_URL"),
		Environment:       getEnv("ENVIRONMENT", "development"),
	}
	return c
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}
