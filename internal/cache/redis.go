package cache

import (
	"context"
	"time"

	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the go-redis client with convenience methods.
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client from the provided configuration.
func NewRedisClient(cfg *config.RedisConfig) (*RedisClient, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisClient{client: client}, nil
}

// Get retrieves the value for the given key.
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set stores a value with the given key and TTL.
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Del removes the specified keys.
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if the given key exists.
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Health checks the Redis connection health.
func (r *RedisClient) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection.
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Client returns the underlying redis.Client for advanced operations.
func (r *RedisClient) Client() *redis.Client {
	return r.client
}
