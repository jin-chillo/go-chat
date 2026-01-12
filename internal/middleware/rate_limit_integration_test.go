//go:build integration

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jin-chillo/go-chat/internal/cache"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_Integration(t *testing.T) {
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

	gin.SetMode(gin.TestMode)

	t.Run("NewRateLimiter with default config", func(t *testing.T) {
		rl, err := NewRateLimiter(redisClient, nil)
		require.NoError(t, err)
		require.NotNil(t, rl)
	})

	t.Run("NewRateLimiter with custom config", func(t *testing.T) {
		cfg := &RateLimitConfig{
			LoginLimit:    10,
			RegisterLimit: 5,
			GeneralLimit:  200,
		}
		rl, err := NewRateLimiter(redisClient, cfg)
		require.NoError(t, err)
		require.NotNil(t, rl)
	})

	t.Run("LoginRateLimit middleware", func(t *testing.T) {
		cfg := &RateLimitConfig{
			LoginLimit:    2, // Low limit for testing
			RegisterLimit: 3,
			GeneralLimit:  100,
		}
		rl, err := NewRateLimiter(redisClient, cfg)
		require.NoError(t, err)

		router := gin.New()
		router.POST("/login", rl.LoginRateLimit(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// First request - should succeed
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Second request - should succeed
		w = httptest.NewRecorder()
		req = httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Third request - should be rate limited
		w = httptest.NewRecorder()
		req = httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("RegisterRateLimit middleware", func(t *testing.T) {
		cfg := &RateLimitConfig{
			LoginLimit:    5,
			RegisterLimit: 1, // Very low limit for testing
			GeneralLimit:  100,
		}
		rl, err := NewRateLimiter(redisClient, cfg)
		require.NoError(t, err)

		router := gin.New()
		router.POST("/register", rl.RegisterRateLimit(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// First request - should succeed
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/register", nil)
		req.RemoteAddr = "192.168.1.101:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Second request - should be rate limited
		w = httptest.NewRecorder()
		req = httptest.NewRequest("POST", "/register", nil)
		req.RemoteAddr = "192.168.1.101:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("GeneralRateLimit middleware", func(t *testing.T) {
		cfg := &RateLimitConfig{
			LoginLimit:    5,
			RegisterLimit: 3,
			GeneralLimit:  2, // Low limit for testing
		}
		rl, err := NewRateLimiter(redisClient, cfg)
		require.NoError(t, err)

		router := gin.New()
		router.GET("/api", rl.GeneralRateLimit(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// First two requests - should succeed
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api", nil)
			req.RemoteAddr = "192.168.1.102:12345"
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// Third request - should be rate limited
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api", nil)
		req.RemoteAddr = "192.168.1.102:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("Different IPs have separate limits", func(t *testing.T) {
		cfg := &RateLimitConfig{
			LoginLimit:    1,
			RegisterLimit: 1,
			GeneralLimit:  1,
		}
		rl, err := NewRateLimiter(redisClient, cfg)
		require.NoError(t, err)

		router := gin.New()
		router.GET("/test", rl.GeneralRateLimit(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Request from IP 1 - should succeed
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Request from IP 2 - should also succeed (different IP)
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Second request from IP 1 - should be limited
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})
}
