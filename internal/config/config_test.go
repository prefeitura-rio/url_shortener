package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		// Clear environment variables
		os.Clearenv()

		cfg := Load()

		assert.Equal(t, "postgres://url_shortener:password@localhost:5432/url_shortener?sslmode=disable", cfg.DatabaseURL)
		assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
		assert.Equal(t, time.Hour, cfg.RedisCacheTTL)
		assert.Equal(t, "", cfg.OTELExporterURL)
		assert.Equal(t, "8080", cfg.Port)
		assert.Equal(t, "example.com", cfg.TwitterDomain)
	})

	t.Run("EnvironmentOverrides", func(t *testing.T) {
		// Set environment variables
		os.Setenv("DATABASE_URL", "postgres://custom@host:5433/db")
		os.Setenv("REDIS_URL", "redis://custom:6380")
		os.Setenv("REDIS_CACHE_TTL", "30m")
		os.Setenv("OTEL_EXPORTER_URL", "http://jaeger:14268/api/traces")
		os.Setenv("PORT", "9090")
		os.Setenv("TWITTER_DOMAIN", "custom.com")

		defer func() {
			os.Clearenv()
		}()

		cfg := Load()

		assert.Equal(t, "postgres://custom@host:5433/db", cfg.DatabaseURL)
		assert.Equal(t, "redis://custom:6380", cfg.RedisURL)
		assert.Equal(t, 30*time.Minute, cfg.RedisCacheTTL)
		assert.Equal(t, "http://jaeger:14268/api/traces", cfg.OTELExporterURL)
		assert.Equal(t, "9090", cfg.Port)
		assert.Equal(t, "custom.com", cfg.TwitterDomain)
	})

	t.Run("InvalidDurationFallback", func(t *testing.T) {
		os.Setenv("REDIS_CACHE_TTL", "invalid-duration")
		defer os.Clearenv()

		cfg := Load()

		// Should fall back to default
		assert.Equal(t, time.Hour, cfg.RedisCacheTTL)
	})
}

func TestGetEnv(t *testing.T) {
	t.Run("ExistingValue", func(t *testing.T) {
		os.Setenv("TEST_KEY", "test_value")
		defer os.Unsetenv("TEST_KEY")

		value := getEnv("TEST_KEY", "default")
		assert.Equal(t, "test_value", value)
	})

	t.Run("DefaultValue", func(t *testing.T) {
		// Ensure the key doesn't exist
		os.Unsetenv("NON_EXISTENT_KEY")

		value := getEnv("NON_EXISTENT_KEY", "default")
		assert.Equal(t, "default", value)
	})

	t.Run("EmptyValue", func(t *testing.T) {
		os.Setenv("EMPTY_KEY", "")
		defer os.Unsetenv("EMPTY_KEY")

		value := getEnv("EMPTY_KEY", "default")
		assert.Equal(t, "default", value)
	})
}

func TestGetDurationEnv(t *testing.T) {
	t.Run("ValidDuration", func(t *testing.T) {
		os.Setenv("DURATION_KEY", "45m")
		defer os.Unsetenv("DURATION_KEY")

		duration := getDurationEnv("DURATION_KEY", time.Hour)
		assert.Equal(t, 45*time.Minute, duration)
	})

	t.Run("InvalidDuration", func(t *testing.T) {
		os.Setenv("INVALID_DURATION_KEY", "not-a-duration")
		defer os.Unsetenv("INVALID_DURATION_KEY")

		duration := getDurationEnv("INVALID_DURATION_KEY", time.Hour)
		assert.Equal(t, time.Hour, duration)
	})

	t.Run("MissingKey", func(t *testing.T) {
		// Ensure the key doesn't exist
		os.Unsetenv("MISSING_DURATION_KEY")

		duration := getDurationEnv("MISSING_DURATION_KEY", 30*time.Minute)
		assert.Equal(t, 30*time.Minute, duration)
	})
}