package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const loginHistoryCollection = "login_history"

// LoginHistoryRepository defines the interface for login history data access.
type LoginHistoryRepository interface {
	Create(ctx context.Context, history *model.LoginHistory) error
	FindByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*model.LoginHistory, error)
}

// loginHistoryRepository implements LoginHistoryRepository using MongoDB.
type loginHistoryRepository struct {
	collection *mongo.Collection
}

// NewLoginHistoryRepository creates a new LoginHistoryRepository instance.
func NewLoginHistoryRepository(db *mongo.Database) LoginHistoryRepository {
	return &loginHistoryRepository{
		collection: db.Collection(loginHistoryCollection),
	}
}

// Create inserts a new login history record.
func (r *loginHistoryRepository) Create(ctx context.Context, history *model.LoginHistory) error {
	_, err := r.collection.InsertOne(ctx, history)
	if err != nil {
		return fmt.Errorf("failed to insert login history: %w", err)
	}
	return nil
}

// FindByUserID retrieves login history for a user, ordered by created_at descending.
func (r *loginHistoryRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*model.LoginHistory, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find login history: %w", err)
	}
	defer cursor.Close(ctx)

	var histories []*model.LoginHistory
	if err := cursor.All(ctx, &histories); err != nil {
		return nil, fmt.Errorf("failed to decode login history: %w", err)
	}

	return histories, nil
}
