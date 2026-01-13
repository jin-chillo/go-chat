//go:build integration

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/auth"
	"github.com/jin-chillo/go-chat/internal/cache"
	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup Redis container
	redisContainer, err := testutil.SetupRedis(ctx)
	require.NoError(t, err)
	defer redisContainer.Cleanup(ctx)

	// Create RedisClient wrapper
	redisClient := cache.NewRedisClientForTest(redisContainer.Client)

	// Create JWTService with Redis
	jwtConfig := &config.JWTConfig{
		Secret:        "test-secret-key-for-middleware-test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	}
	jwtService := auth.NewJWTService(jwtConfig, redisClient)

	gin.SetMode(gin.TestMode)

	t.Run("valid token", func(t *testing.T) {
		userID := uuid.New()
		email := "test@example.com"
		nickname := "testuser"

		// Generate a valid token
		token, err := jwtService.GenerateAccessToken(userID, email, nickname)
		require.NoError(t, err)

		// Setup router with middleware
		router := gin.New()
		router.Use(AuthMiddleware(jwtService))
		router.GET("/protected", func(c *gin.Context) {
			uid, _ := GetUserID(c)
			c.JSON(http.StatusOK, gin.H{"user_id": uid})
		})

		// Make request with valid token
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), userID.String())
	})

	t.Run("invalid token", func(t *testing.T) {
		router := gin.New()
		router.Use(AuthMiddleware(jwtService))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-here")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid token")
	})

	t.Run("expired token", func(t *testing.T) {
		// Create JWT service with very short expiry
		shortExpiryConfig := &config.JWTConfig{
			Secret:        "test-secret-key-for-middleware-test",
			AccessExpiry:  1 * time.Millisecond,
			RefreshExpiry: 24 * time.Hour,
		}
		shortExpiryJwtService := auth.NewJWTService(shortExpiryConfig, redisClient)

		userID := uuid.New()
		email := "expired@example.com"
		nickname := "expireduser"

		// Generate a token that will expire immediately
		token, err := shortExpiryJwtService.GenerateAccessToken(userID, email, nickname)
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(50 * time.Millisecond)

		router := gin.New()
		router.Use(AuthMiddleware(shortExpiryJwtService))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Token expired")
	})

	t.Run("blacklisted token", func(t *testing.T) {
		userID := uuid.New()
		email := "blacklist@example.com"
		nickname := "blacklistuser"

		// Generate a valid token
		token, err := jwtService.GenerateAccessToken(userID, email, nickname)
		require.NoError(t, err)

		// Validate and get claims first
		claims, err := jwtService.ValidateAccessToken(ctx, token)
		require.NoError(t, err)

		// Blacklist the token
		err = jwtService.BlacklistToken(ctx, claims.ID, claims.ExpiresAt.Time)
		require.NoError(t, err)

		router := gin.New()
		router.Use(AuthMiddleware(jwtService))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Token has been revoked")
	})

	t.Run("context values set correctly", func(t *testing.T) {
		userID := uuid.New()
		email := "context@example.com"
		nickname := "contextuser"

		token, err := jwtService.GenerateAccessToken(userID, email, nickname)
		require.NoError(t, err)

		var capturedUserID, capturedEmail, capturedNickname string
		var capturedClaims *auth.TokenClaims

		router := gin.New()
		router.Use(AuthMiddleware(jwtService))
		router.GET("/protected", func(c *gin.Context) {
			capturedUserID, _ = GetUserID(c)
			capturedEmail, _ = GetEmail(c)
			capturedNickname, _ = GetNickname(c)
			capturedClaims, _ = GetClaims(c)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, userID.String(), capturedUserID)
		assert.Equal(t, email, capturedEmail)
		assert.Equal(t, nickname, capturedNickname)
		assert.NotNil(t, capturedClaims)
	})
}
