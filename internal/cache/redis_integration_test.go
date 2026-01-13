//go:build integration

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup Redis container
	redisContainer, err := testutil.SetupRedis(ctx)
	require.NoError(t, err)
	defer redisContainer.Cleanup(ctx)

	// Create RedisClient wrapper using test helper
	client := NewRedisClientForTest(redisContainer.Client)

	t.Run("Set and Get", func(t *testing.T) {
		err := client.Set(ctx, "test:key1", "value1", time.Hour)
		require.NoError(t, err)

		value, err := client.Get(ctx, "test:key1")
		require.NoError(t, err)
		assert.Equal(t, "value1", value)
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		_, err := client.Get(ctx, "test:nonexistent")
		assert.Error(t, err)
	})

	t.Run("Exists", func(t *testing.T) {
		err := client.Set(ctx, "test:exists", "value", time.Hour)
		require.NoError(t, err)

		exists, err := client.Exists(ctx, "test:exists")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = client.Exists(ctx, "test:notexists")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Del", func(t *testing.T) {
		err := client.Set(ctx, "test:todelete", "value", time.Hour)
		require.NoError(t, err)

		err = client.Del(ctx, "test:todelete")
		require.NoError(t, err)

		exists, err := client.Exists(ctx, "test:todelete")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Del multiple keys", func(t *testing.T) {
		err := client.Set(ctx, "test:multi1", "v1", time.Hour)
		require.NoError(t, err)
		err = client.Set(ctx, "test:multi2", "v2", time.Hour)
		require.NoError(t, err)

		err = client.Del(ctx, "test:multi1", "test:multi2")
		require.NoError(t, err)

		exists1, _ := client.Exists(ctx, "test:multi1")
		exists2, _ := client.Exists(ctx, "test:multi2")
		assert.False(t, exists1)
		assert.False(t, exists2)
	})

	t.Run("Set with TTL expiration", func(t *testing.T) {
		err := client.Set(ctx, "test:ttl", "value", 100*time.Millisecond)
		require.NoError(t, err)

		// Key should exist initially
		exists, err := client.Exists(ctx, "test:ttl")
		require.NoError(t, err)
		assert.True(t, exists)

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Key should be gone
		exists, err = client.Exists(ctx, "test:ttl")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Health", func(t *testing.T) {
		err := client.Health(ctx)
		require.NoError(t, err)
	})

	t.Run("Client returns underlying client", func(t *testing.T) {
		underlyingClient := client.Client()
		require.NotNil(t, underlyingClient)

		// Verify it works by using it directly
		err := underlyingClient.Set(ctx, "test:direct", "value", time.Hour).Err()
		require.NoError(t, err)

		val, err := underlyingClient.Get(ctx, "test:direct").Result()
		require.NoError(t, err)
		assert.Equal(t, "value", val)
	})

	t.Run("Close", func(t *testing.T) {
		// Create a separate client for this test
		separateClient := NewRedisClientForTest(redisContainer.Client)

		err := separateClient.Close()
		require.NoError(t, err)
	})
}

func TestNewRedisClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup Redis container
	redisContainer, err := testutil.SetupRedis(ctx)
	require.NoError(t, err)
	defer redisContainer.Cleanup(ctx)

	t.Run("successful connection with valid URL", func(t *testing.T) {
		cfg := &config.RedisConfig{
			URL: redisContainer.ConnStr,
		}

		client, err := NewRedisClient(cfg)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Verify the client works
		err = client.Set(ctx, "newclient:test", "value", time.Minute)
		require.NoError(t, err)

		val, err := client.Get(ctx, "newclient:test")
		require.NoError(t, err)
		assert.Equal(t, "value", val)

		// Clean up
		err = client.Close()
		require.NoError(t, err)
	})

	t.Run("invalid URL format", func(t *testing.T) {
		cfg := &config.RedisConfig{
			URL: "not-a-valid-url",
		}

		client, err := NewRedisClient(cfg)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to parse redis URL")
	})

	t.Run("connection failure with invalid host", func(t *testing.T) {
		cfg := &config.RedisConfig{
			URL: "redis://nonexistent-host:6379",
		}

		client, err := NewRedisClient(cfg)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to ping redis")
	})
}
