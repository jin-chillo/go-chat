//go:build integration

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/cache"
	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_Blacklist_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup Redis container
	redisContainer, err := testutil.SetupRedis(ctx)
	require.NoError(t, err)
	defer redisContainer.Cleanup(ctx)

	// Create RedisClient wrapper using test helper
	redisClient := cache.NewRedisClientForTest(redisContainer.Client)

	// Create JWTService with test config
	jwtConfig := &config.JWTConfig{
		Secret:        "test-secret-key-for-integration-test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	}
	jwtService := NewJWTService(jwtConfig, redisClient)

	t.Run("BlacklistToken and IsBlacklisted", func(t *testing.T) {
		jti := uuid.New().String()
		expiresAt := time.Now().Add(time.Hour)

		// Initially not blacklisted
		blacklisted, err := jwtService.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.False(t, blacklisted)

		// Blacklist the token
		err = jwtService.BlacklistToken(ctx, jti, expiresAt)
		require.NoError(t, err)

		// Now should be blacklisted
		blacklisted, err = jwtService.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("BlacklistToken with expired token", func(t *testing.T) {
		jti := uuid.New().String()
		expiresAt := time.Now().Add(-time.Hour) // Already expired

		// Should not error, just skip blacklisting
		err := jwtService.BlacklistToken(ctx, jti, expiresAt)
		require.NoError(t, err)

		// Should not be blacklisted since it was already expired
		blacklisted, err := jwtService.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("ValidateAccessToken with blacklisted token", func(t *testing.T) {
		userID := uuid.New()
		email := "test@example.com"
		nickname := "testuser"

		// Generate a token
		token, err := jwtService.GenerateAccessToken(userID, email, nickname)
		require.NoError(t, err)

		// Validate the token - should work initially
		claims, err := jwtService.ValidateAccessToken(ctx, token)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, nickname, claims.Nickname)

		// Blacklist the token using its JTI
		err = jwtService.BlacklistToken(ctx, claims.ID, claims.ExpiresAt.Time)
		require.NoError(t, err)

		// Now validation should fail
		_, err = jwtService.ValidateAccessToken(ctx, token)
		assert.ErrorIs(t, err, ErrTokenBlacklisted)
	})

	t.Run("Blacklist TTL expiration", func(t *testing.T) {
		jti := uuid.New().String()
		expiresAt := time.Now().Add(200 * time.Millisecond)

		err := jwtService.BlacklistToken(ctx, jti, expiresAt)
		require.NoError(t, err)

		// Should be blacklisted
		blacklisted, err := jwtService.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.True(t, blacklisted)

		// Wait for TTL to expire
		time.Sleep(300 * time.Millisecond)

		// Should no longer be blacklisted
		blacklisted, err = jwtService.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.False(t, blacklisted)
	})
}
