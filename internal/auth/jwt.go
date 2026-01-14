package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/cache"
	"github.com/jin-chillo/go-chat/internal/config"
)

var (
	// ErrInvalidToken is returned when the token is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired is returned when the token has expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrTokenBlacklisted is returned when the token is blacklisted.
	ErrTokenBlacklisted = errors.New("token blacklisted")
)

// TokenClaims represents the JWT claims.
type TokenClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations.
type JWTService struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	redisClient   *cache.RedisClient
}

// NewJWTService creates a new JWTService instance.
func NewJWTService(cfg *config.JWTConfig, redisClient *cache.RedisClient) *JWTService {
	return &JWTService{
		secret:        []byte(cfg.Secret),
		accessExpiry:  cfg.AccessExpiry,
		refreshExpiry: cfg.RefreshExpiry,
		redisClient:   redisClient,
	}
}

// GenerateAccessToken generates a new access token for the given user.
func (s *JWTService) GenerateAccessToken(userID uuid.UUID, email, nickname string) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID:   userID.String(),
		Email:    email,
		Nickname: nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
			Issuer:    "gochat",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GenerateRefreshToken generates a new refresh token string.
func (s *JWTService) GenerateRefreshToken() string {
	return uuid.New().String()
}

// GetRefreshExpiry returns the refresh token expiry duration.
func (s *JWTService) GetRefreshExpiry() time.Duration {
	return s.refreshExpiry
}

// GetAccessExpiry returns the access token expiry duration.
func (s *JWTService) GetAccessExpiry() time.Duration {
	return s.accessExpiry
}

// ValidateAccessToken validates the access token and returns the claims.
func (s *JWTService) ValidateAccessToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check if token is blacklisted
	blacklisted, err := s.IsBlacklisted(ctx, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check blacklist: %w", err)
	}
	if blacklisted {
		return nil, ErrTokenBlacklisted
	}

	return claims, nil
}

// BlacklistToken adds a token to the blacklist.
func (s *JWTService) BlacklistToken(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // Token already expired, no need to blacklist
	}

	key := fmt.Sprintf("blacklist:%s", jti)
	return s.redisClient.Set(ctx, key, "1", ttl)
}

// IsBlacklisted checks if a token is blacklisted.
func (s *JWTService) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", jti)
	return s.redisClient.Exists(ctx, key)
}
