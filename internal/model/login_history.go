package model

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// LoginHistory represents a login history record stored in MongoDB.
type LoginHistory struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    uuid.UUID     `bson:"user_id" json:"user_id"`
	IP        string        `bson:"ip" json:"ip"`
	UserAgent string        `bson:"user_agent" json:"user_agent"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}

// NewLoginHistory creates a new LoginHistory instance.
func NewLoginHistory(userID uuid.UUID, ip, userAgent string) *LoginHistory {
	return &LoginHistory{
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
	}
}
