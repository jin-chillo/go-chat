package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"gorm.io/gorm"
)

var (
	// ErrRefreshTokenNotFound is returned when a refresh token is not found.
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	// ErrRefreshTokenExpired is returned when a refresh token has expired.
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	// ErrRefreshTokenRevoked is returned when a refresh token has been revoked.
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
)

// TokenRepository defines the interface for refresh token data access.
type TokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
}

// tokenRepository implements TokenRepository using GORM.
type tokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository creates a new TokenRepository instance.
func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

// Create inserts a new refresh token into the database.
func (r *tokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	result := r.db.WithContext(ctx).Create(token)
	if result.Error != nil {
		return fmt.Errorf("failed to create refresh token: %w", result.Error)
	}
	return nil
}

// FindByTokenHash retrieves a refresh token by its hash.
func (r *tokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	result := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", result.Error)
	}
	return &token, nil
}

// RevokeByUserID revokes all refresh tokens for a user.
func (r *tokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %w", result.Error)
	}
	return nil
}

// RevokeByTokenHash revokes a specific refresh token.
func (r *tokenRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", now)
	if result.Error != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", result.Error)
	}
	return nil
}

// HashToken creates a SHA-256 hash of the token.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
