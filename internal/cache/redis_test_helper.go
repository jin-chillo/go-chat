//go:build integration

package cache

import "github.com/redis/go-redis/v9"

// NewRedisClientForTest creates a RedisClient from an existing redis.Client.
// This is intended for use in integration tests.
func NewRedisClientForTest(client *redis.Client) *RedisClient {
	return &RedisClient{client: client}
}
