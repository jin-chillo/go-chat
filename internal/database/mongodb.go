package database

import (
	"context"
	"fmt"

	"github.com/jin-chillo/go-chat/internal/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// MongoDB wraps the MongoDB client and database handle.
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewMongoDB creates a new MongoDB connection using the provided configuration.
func NewMongoDB(cfg *config.MongoDBConfig) (*MongoDB, error) {
	clientOpts := options.Client().ApplyURI(cfg.URI)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection with a ping
	if err := client.Ping(context.Background(), readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoDB{
		client:   client,
		database: client.Database(cfg.Database),
	}, nil
}

// Collection returns a handle to the specified collection.
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.database.Collection(name)
}

// Health checks the MongoDB connection by pinging the server.
func (m *MongoDB) Health(ctx context.Context) error {
	if err := m.client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("MongoDB health check failed: %w", err)
	}
	return nil
}

// Close disconnects from MongoDB.
func (m *MongoDB) Close(ctx context.Context) error {
	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}
	return nil
}

// CreateIndexes creates necessary indexes for all collections.
func (m *MongoDB) CreateIndexes(ctx context.Context) error {
	// Create indexes for messages collection
	if err := m.createMessagesIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create messages indexes: %w", err)
	}

	// Create indexes for login_history collection
	if err := m.createLoginHistoryIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create login_history indexes: %w", err)
	}

	return nil
}

// createMessagesIndexes creates indexes for the messages collection.
func (m *MongoDB) createMessagesIndexes(ctx context.Context) error {
	collection := m.Collection("messages")

	// Compound index on channel_id and created_at for efficient message retrieval
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "channel_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
		Options: options.Index().SetName("idx_channel_created"),
	}

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create channel_id + created_at index: %w", err)
	}

	return nil
}

// createLoginHistoryIndexes creates indexes for the login_history collection.
func (m *MongoDB) createLoginHistoryIndexes(ctx context.Context) error {
	collection := m.Collection("login_history")

	// Compound index on user_id and created_at for efficient login history retrieval
	userCreatedIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
		Options: options.Index().SetName("idx_user_created"),
	}

	// TTL index for automatic cleanup after 90 days (7776000 seconds)
	ttlIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "created_at", Value: 1},
		},
		Options: options.Index().
			SetName("idx_ttl_created_at").
			SetExpireAfterSeconds(7776000),
	}

	indexes := []mongo.IndexModel{userCreatedIndex, ttlIndex}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create login_history indexes: %w", err)
	}

	return nil
}
