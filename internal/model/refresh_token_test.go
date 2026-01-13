package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRefreshToken_TableName(t *testing.T) {
	token := RefreshToken{}
	assert.Equal(t, "refresh_tokens", token.TableName())
}

func TestRefreshToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "expired token",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "valid token",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-1 * time.Second),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &RefreshToken{
				ID:        uuid.New(),
				ExpiresAt: tt.expiresAt,
			}
			assert.Equal(t, tt.want, token.IsExpired())
		})
	}
}

func TestRefreshToken_IsRevoked(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		revokedAt *time.Time
		want      bool
	}{
		{
			name:      "revoked token",
			revokedAt: &now,
			want:      true,
		},
		{
			name:      "not revoked token",
			revokedAt: nil,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &RefreshToken{
				ID:        uuid.New(),
				RevokedAt: tt.revokedAt,
			}
			assert.Equal(t, tt.want, token.IsRevoked())
		})
	}
}

func TestRefreshToken_IsValid(t *testing.T) {
	now := time.Now()
	future := time.Now().Add(1 * time.Hour)
	past := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name      string
		expiresAt time.Time
		revokedAt *time.Time
		want      bool
	}{
		{
			name:      "valid token",
			expiresAt: future,
			revokedAt: nil,
			want:      true,
		},
		{
			name:      "expired token",
			expiresAt: past,
			revokedAt: nil,
			want:      false,
		},
		{
			name:      "revoked token",
			expiresAt: future,
			revokedAt: &now,
			want:      false,
		},
		{
			name:      "expired and revoked token",
			expiresAt: past,
			revokedAt: &now,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &RefreshToken{
				ID:        uuid.New(),
				ExpiresAt: tt.expiresAt,
				RevokedAt: tt.revokedAt,
			}
			assert.Equal(t, tt.want, token.IsValid())
		})
	}
}
