package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Nickname     string    `gorm:"type:varchar(50);not null" json:"nickname"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for the User model.
func (User) TableName() string {
	return "users"
}
