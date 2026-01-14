//go:build integration

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/cache"
	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Logout_Integration(t *testing.T) {
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
		Secret:        "test-secret-key-for-logout-test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	}
	jwtService := NewJWTService(jwtConfig, redisClient)

	t.Run("successful logout", func(t *testing.T) {
		userID := uuid.New()
		email := "logout@example.com"
		nickname := "logoutuser"

		// Generate access token
		accessToken, err := jwtService.GenerateAccessToken(userID, email, nickname)
		require.NoError(t, err)

		// Validate token to get claims
		claims, err := jwtService.ValidateAccessToken(ctx, accessToken)
		require.NoError(t, err)

		// Create mock repositories
		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)
		mockTokenRepo.On("RevokeByUserID", ctx, userID).Return(nil)

		// Create service and call Logout
		service := NewAuthService(mockUserRepo, mockTokenRepo, jwtService)
		err = service.Logout(ctx, claims)
		require.NoError(t, err)

		// Verify token is blacklisted
		blacklisted, err := jwtService.IsBlacklisted(ctx, claims.ID)
		require.NoError(t, err)
		assert.True(t, blacklisted)

		// Verify token validation fails
		_, err = jwtService.ValidateAccessToken(ctx, accessToken)
		assert.ErrorIs(t, err, ErrTokenBlacklisted)

		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("logout with revoke error", func(t *testing.T) {
		userID := uuid.New()
		email := "revokeerror@example.com"
		nickname := "revokeerroruser"

		// Generate access token
		accessToken, err := jwtService.GenerateAccessToken(userID, email, nickname)
		require.NoError(t, err)

		// Validate token to get claims
		claims, err := jwtService.ValidateAccessToken(ctx, accessToken)
		require.NoError(t, err)

		// Create mock repositories - RevokeByUserID returns error
		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)
		mockTokenRepo.On("RevokeByUserID", ctx, userID).Return(ErrRefreshTokenNotFound)

		// Create service and call Logout
		service := NewAuthService(mockUserRepo, mockTokenRepo, jwtService)
		err = service.Logout(ctx, claims)

		// Should return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to revoke refresh tokens")

		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("logout with invalid user ID in claims", func(t *testing.T) {
		// Create claims with invalid UserID
		claims := &TokenClaims{
			UserID:   "invalid-uuid",
			Email:    "invalid@example.com",
			Nickname: "invaliduser",
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        uuid.New().String(),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}

		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)

		service := NewAuthService(mockUserRepo, mockTokenRepo, jwtService)
		err := service.Logout(ctx, claims)

		// Should return error due to invalid UUID
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID in claims")
	})
}

func TestAuthService_FullLoginLogoutFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup Redis and PostgreSQL containers
	redisContainer, err := testutil.SetupRedis(ctx)
	require.NoError(t, err)
	defer redisContainer.Cleanup(ctx)

	pgContainer, err := testutil.SetupPostgres(ctx)
	require.NoError(t, err)
	defer pgContainer.Cleanup(ctx)

	// Auto-migrate schemas
	err = pgContainer.DB.AutoMigrate(&model.User{}, &model.RefreshToken{})
	require.NoError(t, err)

	// Create RedisClient
	redisClient := cache.NewRedisClientForTest(redisContainer.Client)

	// Create JWTService
	jwtConfig := &config.JWTConfig{
		Secret:        "test-secret-key-full-flow",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	}
	jwtService := NewJWTService(jwtConfig, redisClient)

	// Create repositories
	userRepo := NewUserRepository(pgContainer.DB)
	tokenRepo := NewTokenRepository(pgContainer.DB)

	// Create service
	service := NewAuthService(userRepo, tokenRepo, jwtService)

	t.Run("register login and logout flow", func(t *testing.T) {
		// 1. Register
		registerReq := &RegisterRequest{
			Email:    "flowtest@example.com",
			Password: "SecurePassword123!",
			Nickname: "flowuser",
		}
		user, err := service.Register(ctx, registerReq)
		require.NoError(t, err)
		require.NotNil(t, user)

		// 2. Login
		loginReq := &LoginRequest{
			Email:    "flowtest@example.com",
			Password: "SecurePassword123!",
		}
		loginResult, err := service.Login(ctx, loginReq)
		require.NoError(t, err)
		require.NotEmpty(t, loginResult.AccessToken)
		require.NotEmpty(t, loginResult.RefreshToken)

		// 3. Validate access token
		claims, err := jwtService.ValidateAccessToken(ctx, loginResult.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, user.ID.String(), claims.UserID)

		// 4. Logout
		err = service.Logout(ctx, claims)
		require.NoError(t, err)

		// 5. Verify access token is now blacklisted
		_, err = jwtService.ValidateAccessToken(ctx, loginResult.AccessToken)
		assert.ErrorIs(t, err, ErrTokenBlacklisted)

		// 6. Verify refresh token is revoked
		_, err = tokenRepo.FindByTokenHash(ctx, HashToken(loginResult.RefreshToken))
		// Token should be revoked (not deleted, but marked as revoked)
		tokenRecord, err := tokenRepo.FindByTokenHash(ctx, HashToken(loginResult.RefreshToken))
		require.NoError(t, err)
		assert.True(t, tokenRecord.IsRevoked())
	})
}

