package model

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Message represents a chat message stored in MongoDB.
type Message struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID uuid.UUID     `bson:"channel_id" json:"channel_id"`
	UserID    uuid.UUID     `bson:"user_id" json:"user_id"`
	Nickname  string        `bson:"nickname" json:"nickname"`
	Content   string        `bson:"content" json:"content"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}

// CollectionName returns the MongoDB collection name for messages.
func (Message) CollectionName() string {
	return "messages"
}
