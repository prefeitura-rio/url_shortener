package config

import (
	"os"
	"time"
)

type Config struct {
	DatabaseURL      string
	RedisURL         string
	RedisCacheTTL    time.Duration
	OTELExporterURL  string
	Port             string
	TwitterDomain    string
}

func Load() *Config {
	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://url_shortener:password@localhost:5432/url_shortener?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisCacheTTL:   getDurationEnv("REDIS_CACHE_TTL", time.Hour),
		OTELExporterURL: getEnv("OTEL_EXPORTER_URL", ""),
		Port:            getEnv("PORT", "8080"),
		TwitterDomain:   getEnv("TWITTER_DOMAIN", "example.com"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
} 