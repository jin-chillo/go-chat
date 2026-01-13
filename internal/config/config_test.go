package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	originalEnv := map[string]string{
		"DATABASE_URL": os.Getenv("DATABASE_URL"),
		"MONGODB_URI":  os.Getenv("MONGODB_URI"),
		"REDIS_URL":    os.Getenv("REDIS_URL"),
		"JWT_SECRET":   os.Getenv("JWT_SECRET"),
	}

	// Restore env vars after test
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	t.Run("successful load with all required vars", func(t *testing.T) {
		// Set required env vars
		os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
		os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Setenv("JWT_SECRET", "test-secret-key")

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify values
		assert.Equal(t, "postgres://test:test@localhost:5432/test", cfg.Database.URL)
		assert.Equal(t, "mongodb://localhost:27017", cfg.MongoDB.URI)
		assert.Equal(t, "redis://localhost:6379", cfg.Redis.URL)
		assert.Equal(t, "test-secret-key", cfg.JWT.Secret)

		// Verify defaults
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, "debug", cfg.Server.GinMode)
		assert.Equal(t, "gochat", cfg.MongoDB.Database)
		assert.Equal(t, 15*time.Minute, cfg.JWT.AccessExpiry)
		assert.Equal(t, 168*time.Hour, cfg.JWT.RefreshExpiry)
		assert.Equal(t, 12, cfg.Bcrypt.Cost)
	})

	t.Run("custom values override defaults", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
		os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
		os.Setenv("MONGODB_DATABASE", "customdb")
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Setenv("JWT_SECRET", "test-secret-key")
		os.Setenv("PORT", "3000")
		os.Setenv("GIN_MODE", "release")
		os.Setenv("JWT_ACCESS_EXPIRY", "30m")
		os.Setenv("JWT_REFRESH_EXPIRY", "24h")
		os.Setenv("BCRYPT_COST", "10")

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, 3000, cfg.Server.Port)
		assert.Equal(t, "release", cfg.Server.GinMode)
		assert.Equal(t, "customdb", cfg.MongoDB.Database)
		assert.Equal(t, 30*time.Minute, cfg.JWT.AccessExpiry)
		assert.Equal(t, 24*time.Hour, cfg.JWT.RefreshExpiry)
		assert.Equal(t, 10, cfg.Bcrypt.Cost)

		// Cleanup custom env vars
		os.Unsetenv("MONGODB_DATABASE")
		os.Unsetenv("PORT")
		os.Unsetenv("GIN_MODE")
		os.Unsetenv("JWT_ACCESS_EXPIRY")
		os.Unsetenv("JWT_REFRESH_EXPIRY")
		os.Unsetenv("BCRYPT_COST")
	})

	t.Run("missing required DATABASE_URL", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Setenv("JWT_SECRET", "test-secret-key")

		cfg, err := Load()
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("missing required MONGODB_URI", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
		os.Unsetenv("MONGODB_URI")
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Setenv("JWT_SECRET", "test-secret-key")

		cfg, err := Load()
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("missing required REDIS_URL", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
		os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
		os.Unsetenv("REDIS_URL")
		os.Setenv("JWT_SECRET", "test-secret-key")

		cfg, err := Load()
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("missing required JWT_SECRET", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
		os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Unsetenv("JWT_SECRET")

		cfg, err := Load()
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})
}
