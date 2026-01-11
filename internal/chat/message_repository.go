package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MessageRepository defines the interface for message data access.
type MessageRepository interface {
	Create(ctx context.Context, message *model.Message) error
	FindByChannelID(ctx context.Context, channelID uuid.UUID, before *time.Time, limit int) ([]model.Message, error)
	CountByChannelID(ctx context.Context, channelID uuid.UUID) (int64, error)
}

// messageRepository implements MessageRepository using MongoDB.
type messageRepository struct {
	collection *mongo.Collection
}

// NewMessageRepository creates a new MessageRepository instance.
func NewMessageRepository(db *mongo.Database) MessageRepository {
	return &messageRepository{
		collection: db.Collection("messages"),
	}
}

// Create inserts a new message into MongoDB.
func (r *messageRepository) Create(ctx context.Context, message *model.Message) error {
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	result, err := r.collection.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		message.ID = oid
	}

	return nil
}

// FindByChannelID retrieves messages for a channel with cursor-based pagination.
func (r *messageRepository) FindByChannelID(ctx context.Context, channelID uuid.UUID, before *time.Time, limit int) ([]model.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	filter := bson.M{"channel_id": channelID}

	// Cursor-based pagination: get messages before the given time
	if before != nil {
		filter["created_at"] = bson.M{"$lt": *before}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}). // Newest first
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []model.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// CountByChannelID counts messages in a channel.
func (r *messageRepository) CountByChannelID(ctx context.Context, channelID uuid.UUID) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"channel_id": channelID})
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}
	return count, nil
}
