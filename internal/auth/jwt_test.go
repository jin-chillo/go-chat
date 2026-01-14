package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newJWTTestService creates a JWTService for testing
func newJWTTestService() *JWTService {
	return &JWTService{
		secret:        []byte("test-secret-key-for-testing-only"),
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
		redisClient:   nil,
	}
}

func TestNewJWTService(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:        "test-secret",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}

	service := NewJWTService(cfg, nil)

	assert.NotNil(t, service)
	assert.Equal(t, []byte("test-secret"), service.secret)
	assert.Equal(t, 15*time.Minute, service.accessExpiry)
	assert.Equal(t, 7*24*time.Hour, service.refreshExpiry)
}

func TestJWTService_GenerateAccessToken(t *testing.T) {
	service := newJWTTestService()

	userID := uuid.New()
	email := "test@example.com"
	nickname := "testuser"

	token, err := service.GenerateAccessToken(userID, email, nickname)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token structure by parsing it
	parsed, err := jwt.ParseWithClaims(token, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return service.secret, nil
	})
	require.NoError(t, err)
	assert.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(*TokenClaims)
	require.True(t, ok)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, nickname, claims.Nickname)
	assert.Equal(t, "gochat", claims.Issuer)
}

func TestJWTService_GenerateRefreshToken(t *testing.T) {
	service := newJWTTestService()

	token1 := service.GenerateRefreshToken()
	token2 := service.GenerateRefreshToken()

	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token1, token2) // Should generate unique tokens

	// Verify it's a valid UUID
	_, err := uuid.Parse(token1)
	assert.NoError(t, err)
}

func TestJWTService_GetRefreshExpiry(t *testing.T) {
	service := newJWTTestService()

	expiry := service.GetRefreshExpiry()

	assert.Equal(t, 7*24*time.Hour, expiry)
}

func TestJWTService_GetAccessExpiry(t *testing.T) {
	service := newJWTTestService()

	expiry := service.GetAccessExpiry()

	assert.Equal(t, 15*time.Minute, expiry)
}

func TestJWTService_ValidateAccessToken_InvalidToken(t *testing.T) {
	service := newJWTTestService()

	_, err := service.ValidateAccessToken(context.Background(), "invalid-token")

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestJWTService_ValidateAccessToken_WrongSignature(t *testing.T) {
	service1 := newJWTTestService()
	service2 := &JWTService{
		secret:        []byte("different-secret"),
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
		redisClient:   nil,
	}

	userID := uuid.New()
	token, err := service1.GenerateAccessToken(userID, "test@example.com", "testuser")
	require.NoError(t, err)

	// Try to validate with different secret
	_, err = service2.ValidateAccessToken(context.Background(), token)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestJWTService_ValidateAccessToken_ExpiredToken(t *testing.T) {
	// Create service with very short expiry
	service := &JWTService{
		secret:        []byte("test-secret"),
		accessExpiry:  1 * time.Millisecond, // Very short
		refreshExpiry: 7 * 24 * time.Hour,
		redisClient:   nil,
	}

	userID := uuid.New()
	token, err := service.GenerateAccessToken(userID, "test@example.com", "testuser")
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Validate - expired check happens before blacklist check
	_, err = service.ValidateAccessToken(context.Background(), token)
	assert.Error(t, err)
	assert.Equal(t, ErrTokenExpired, err)
}

func TestJWTService_ValidateAccessToken_MalformedToken(t *testing.T) {
	service := newJWTTestService()

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"random string", "not-a-jwt-token"},
		{"invalid segments", "a.b"},
		{"too many segments", "a.b.c.d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateAccessToken(context.Background(), tt.token)
			assert.Error(t, err)
		})
	}
}

func TestTokenClaims_Structure(t *testing.T) {
	service := newJWTTestService()

	userID := uuid.New()
	email := "test@example.com"
	nickname := "testuser"

	token, err := service.GenerateAccessToken(userID, email, nickname)
	require.NoError(t, err)

	// Parse without full validation (bypass blacklist check)
	parsed, err := jwt.ParseWithClaims(token, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return service.secret, nil
	})
	require.NoError(t, err)

	claims, ok := parsed.Claims.(*TokenClaims)
	require.True(t, ok)

	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, nickname, claims.Nickname)
	assert.Equal(t, "gochat", claims.Issuer)
	assert.Equal(t, userID.String(), claims.Subject)
	assert.NotEmpty(t, claims.ID) // JTI
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.ExpiresAt)
}
