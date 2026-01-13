//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresDB_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer, err := testutil.SetupPostgres(ctx)
	require.NoError(t, err)
	defer pgContainer.Cleanup(ctx)

	t.Run("NewPostgresDB with valid config", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			URL: pgContainer.ConnStr,
		}

		db, err := NewPostgresDB(cfg)
		require.NoError(t, err)
		require.NotNil(t, db)

		// Clean up
		err = Close(db)
		require.NoError(t, err)
	})

	t.Run("Health check", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			URL: pgContainer.ConnStr,
		}

		db, err := NewPostgresDB(cfg)
		require.NoError(t, err)
		defer Close(db)

		err = Health(db)
		require.NoError(t, err)
	})

	t.Run("NewPostgresDB with invalid URL", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			URL: "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable",
		}

		db, err := NewPostgresDB(cfg)
		assert.Error(t, err)
		assert.Nil(t, db)
	})
}

func TestMongoDB_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup MongoDB container
	mongoContainer, err := testutil.SetupMongo(ctx)
	require.NoError(t, err)
	defer mongoContainer.Cleanup(ctx)

	t.Run("NewMongoDB with valid config", func(t *testing.T) {
		cfg := &config.MongoDBConfig{
			URI:      mongoContainer.ConnStr,
			Database: "testdb",
		}

		db, err := NewMongoDB(cfg)
		require.NoError(t, err)
		require.NotNil(t, db)

		// Clean up
		err = db.Close(ctx)
		require.NoError(t, err)
	})

	t.Run("Collection and Database methods", func(t *testing.T) {
		cfg := &config.MongoDBConfig{
			URI:      mongoContainer.ConnStr,
			Database: "testdb",
		}

		db, err := NewMongoDB(cfg)
		require.NoError(t, err)
		defer db.Close(ctx)

		// Test Collection method
		collection := db.Collection("test_collection")
		require.NotNil(t, collection)

		// Test Database method
		database := db.Database()
		require.NotNil(t, database)
	})

	t.Run("Health check", func(t *testing.T) {
		cfg := &config.MongoDBConfig{
			URI:      mongoContainer.ConnStr,
			Database: "testdb",
		}

		db, err := NewMongoDB(cfg)
		require.NoError(t, err)
		defer db.Close(ctx)

		err = db.Health(ctx)
		require.NoError(t, err)
	})

	t.Run("CreateIndexes", func(t *testing.T) {
		cfg := &config.MongoDBConfig{
			URI:      mongoContainer.ConnStr,
			Database: "testdb_indexes",
		}

		db, err := NewMongoDB(cfg)
		require.NoError(t, err)
		defer db.Close(ctx)

		// Create indexes
		err = db.CreateIndexes(ctx)
		require.NoError(t, err)

		// Verify messages indexes
		messagesCollection := db.Collection("messages")
		cursor, err := messagesCollection.Indexes().List(ctx)
		require.NoError(t, err)

		var indexes []map[string]interface{}
		err = cursor.All(ctx, &indexes)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(indexes), 2) // _id index + our index

		// Verify login_history indexes
		loginHistoryCollection := db.Collection("login_history")
		cursor, err = loginHistoryCollection.Indexes().List(ctx)
		require.NoError(t, err)

		indexes = nil
		err = cursor.All(ctx, &indexes)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(indexes), 3) // _id index + 2 our indexes
	})

	t.Run("NewMongoDB with invalid URI", func(t *testing.T) {
		cfg := &config.MongoDBConfig{
			URI:      "mongodb://invalid:27017/test?serverSelectionTimeoutMS=1000",
			Database: "testdb",
		}

		db, err := NewMongoDB(cfg)
		assert.Error(t, err)
		assert.Nil(t, db)
	})
}
