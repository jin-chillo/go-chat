//go:build integration

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jin-chillo/go-chat/internal/cache"
	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_Logout_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup containers
	redisContainer, err := testutil.SetupRedis(ctx)
	require.NoError(t, err)
	defer redisContainer.Cleanup(ctx)

	pgContainer, err := testutil.SetupPostgres(ctx)
	require.NoError(t, err)
	defer pgContainer.Cleanup(ctx)

	// Auto-migrate schemas
	err = pgContainer.DB.AutoMigrate(&model.User{}, &model.RefreshToken{})
	require.NoError(t, err)

	// Create services
	redisClient := cache.NewRedisClientForTest(redisContainer.Client)
	jwtConfig := &config.JWTConfig{
		Secret:        "test-secret-key-for-handler-logout",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	}
	jwtService := NewJWTService(jwtConfig, redisClient)

	userRepo := NewUserRepository(pgContainer.DB)
	tokenRepo := NewTokenRepository(pgContainer.DB)
	authService := NewAuthService(userRepo, tokenRepo, jwtService)
	handler := NewHandler(authService, jwtService)

	gin.SetMode(gin.TestMode)

	t.Run("successful logout", func(t *testing.T) {
		// Create a router
		router := gin.New()
		rg := router.Group("/api/v1")
		handler.RegisterRoutes(rg, nil)

		// First, register a user
		registerBody := `{"email":"logouttest@example.com","password":"Password123!","nickname":"logoutuser"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(registerBody)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Login to get tokens
		loginBody := `{"email":"logouttest@example.com","password":"Password123!"}`
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(loginBody)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var loginResp struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &loginResp)
		require.NoError(t, err)
		require.NotEmpty(t, loginResp.AccessToken)

		// Logout with valid token
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Successfully logged out")

		// Try to logout again with the same token - should fail (token blacklisted)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("logout with service error", func(t *testing.T) {
		// Create a new user for this test
		registerBody := `{"email":"logouterror@example.com","password":"Password123!","nickname":"erroruser"}`
		router := gin.New()
		rg := router.Group("/api/v1")
		handler.RegisterRoutes(rg, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(registerBody)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Login
		loginBody := `{"email":"logouterror@example.com","password":"Password123!"}`
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(loginBody)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var loginResp struct {
			AccessToken string `json:"access_token"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &loginResp)
		require.NoError(t, err)

		// Logout succeeds
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_FullAuthFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup containers
	redisContainer, err := testutil.SetupRedis(ctx)
	require.NoError(t, err)
	defer redisContainer.Cleanup(ctx)

	pgContainer, err := testutil.SetupPostgres(ctx)
	require.NoError(t, err)
	defer pgContainer.Cleanup(ctx)

	// Auto-migrate schemas
	err = pgContainer.DB.AutoMigrate(&model.User{}, &model.RefreshToken{})
	require.NoError(t, err)

	// Create services
	redisClient := cache.NewRedisClientForTest(redisContainer.Client)
	jwtConfig := &config.JWTConfig{
		Secret:        "test-secret-key-full-auth-flow",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	}
	jwtService := NewJWTService(jwtConfig, redisClient)

	userRepo := NewUserRepository(pgContainer.DB)
	tokenRepo := NewTokenRepository(pgContainer.DB)
	authService := NewAuthService(userRepo, tokenRepo, jwtService)
	handler := NewHandler(authService, jwtService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg, nil)

	t.Run("full auth flow", func(t *testing.T) {
		// 1. Register
		registerBody := `{"email":"fullflow@example.com","password":"SecurePass123!","nickname":"fullflowuser"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(registerBody)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// 2. Login
		loginBody := `{"email":"fullflow@example.com","password":"SecurePass123!"}`
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(loginBody)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var loginResp struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &loginResp)
		require.NoError(t, err)

		// 3. Refresh token
		refreshBody, _ := json.Marshal(map[string]string{"refresh_token": loginResp.RefreshToken})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var refreshResp struct {
			AccessToken string `json:"access_token"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &refreshResp)
		require.NoError(t, err)
		assert.NotEmpty(t, refreshResp.AccessToken)

		// 4. Logout with new access token
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+refreshResp.AccessToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 5. Verify old refresh token is revoked
		refreshBody, _ = json.Marshal(map[string]string{"refresh_token": loginResp.RefreshToken})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Refresh token should be revoked after logout
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
