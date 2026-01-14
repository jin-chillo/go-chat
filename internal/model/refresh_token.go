package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a refresh token in the database.
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"type:varchar(255);not null;index"`
	ExpiresAt time.Time  `gorm:"not null"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	RevokedAt *time.Time `gorm:"default:null"`
}

// TableName returns the table name for the RefreshToken model.
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired checks if the refresh token has expired.
func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsRevoked checks if the refresh token has been revoked.
func (r *RefreshToken) IsRevoked() bool {
	return r.RevokedAt != nil
}

// IsValid checks if the refresh token is valid (not expired and not revoked).
func (r *RefreshToken) IsValid() bool {
	return !r.IsExpired() && !r.IsRevoked()
}
