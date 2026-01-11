package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/cache"
)

const (
	// presenceKeyPrefix is the Redis key prefix for channel presence.
	presenceKeyPrefix = "presence:channel:"

	// userPresenceKeyPrefix is the Redis key prefix for user presence info.
	userPresenceKeyPrefix = "presence:user:"

	// presenceTTL is how long presence data stays valid.
	presenceTTL = 5 * time.Minute
)

// PresenceService manages online status using Redis.
type PresenceService struct {
	redis *cache.RedisClient
}

// NewPresenceService creates a new presence service.
func NewPresenceService(redis *cache.RedisClient) *PresenceService {
	return &PresenceService{redis: redis}
}

// SetOnline marks a user as online in a channel.
func (p *PresenceService) SetOnline(ctx context.Context, channelID, userID uuid.UUID, nickname string) error {
	client := p.redis.Client()

	// Add user to channel's online set
	channelKey := fmt.Sprintf("%s%s", presenceKeyPrefix, channelID.String())
	if err := client.SAdd(ctx, channelKey, userID.String()).Err(); err != nil {
		return fmt.Errorf("failed to add user to channel presence: %w", err)
	}

	// Set TTL on the channel key
	if err := client.Expire(ctx, channelKey, presenceTTL).Err(); err != nil {
		return fmt.Errorf("failed to set presence TTL: %w", err)
	}

	// Store user info (nickname) for lookup
	userKey := fmt.Sprintf("%s%s", userPresenceKeyPrefix, userID.String())
	if err := client.HSet(ctx, userKey, "nickname", nickname).Err(); err != nil {
		return fmt.Errorf("failed to store user info: %w", err)
	}
	if err := client.Expire(ctx, userKey, presenceTTL).Err(); err != nil {
		return fmt.Errorf("failed to set user info TTL: %w", err)
	}

	return nil
}

// SetOffline marks a user as offline in a channel.
func (p *PresenceService) SetOffline(ctx context.Context, channelID, userID uuid.UUID) error {
	client := p.redis.Client()

	channelKey := fmt.Sprintf("%s%s", presenceKeyPrefix, channelID.String())
	if err := client.SRem(ctx, channelKey, userID.String()).Err(); err != nil {
		return fmt.Errorf("failed to remove user from channel presence: %w", err)
	}

	return nil
}

// SetOfflineFromAllChannels removes a user from all channel presence sets.
func (p *PresenceService) SetOfflineFromAllChannels(ctx context.Context, userID uuid.UUID, channelIDs []uuid.UUID) error {
	client := p.redis.Client()

	for _, channelID := range channelIDs {
		channelKey := fmt.Sprintf("%s%s", presenceKeyPrefix, channelID.String())
		if err := client.SRem(ctx, channelKey, userID.String()).Err(); err != nil {
			return fmt.Errorf("failed to remove user from channel %s: %w", channelID, err)
		}
	}

	// Clean up user info
	userKey := fmt.Sprintf("%s%s", userPresenceKeyPrefix, userID.String())
	if err := client.Del(ctx, userKey).Err(); err != nil {
		return fmt.Errorf("failed to delete user info: %w", err)
	}

	return nil
}

// GetOnlineUsers returns online users in a channel.
func (p *PresenceService) GetOnlineUsers(ctx context.Context, channelID uuid.UUID) ([]WSUserInfo, error) {
	client := p.redis.Client()

	channelKey := fmt.Sprintf("%s%s", presenceKeyPrefix, channelID.String())
	userIDs, err := client.SMembers(ctx, channelKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel members: %w", err)
	}

	users := make([]WSUserInfo, 0, len(userIDs))
	for _, userIDStr := range userIDs {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}

		userKey := fmt.Sprintf("%s%s", userPresenceKeyPrefix, userIDStr)
		nickname, err := client.HGet(ctx, userKey, "nickname").Result()
		if err != nil {
			nickname = "Unknown"
		}

		users = append(users, WSUserInfo{
			ID:       userID,
			Nickname: nickname,
		})
	}

	return users, nil
}

// IsOnline checks if a user is online in a channel.
func (p *PresenceService) IsOnline(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	client := p.redis.Client()

	channelKey := fmt.Sprintf("%s%s", presenceKeyPrefix, channelID.String())
	return client.SIsMember(ctx, channelKey, userID.String()).Result()
}

// RefreshPresence refreshes the TTL for a user's presence.
func (p *PresenceService) RefreshPresence(ctx context.Context, channelID, userID uuid.UUID) error {
	client := p.redis.Client()

	channelKey := fmt.Sprintf("%s%s", presenceKeyPrefix, channelID.String())
	if err := client.Expire(ctx, channelKey, presenceTTL).Err(); err != nil {
		return fmt.Errorf("failed to refresh channel presence: %w", err)
	}

	userKey := fmt.Sprintf("%s%s", userPresenceKeyPrefix, userID.String())
	if err := client.Expire(ctx, userKey, presenceTTL).Err(); err != nil {
		return fmt.Errorf("failed to refresh user presence: %w", err)
	}

	return nil
}

// GetOnlineCount returns the count of online users in a channel.
func (p *PresenceService) GetOnlineCount(ctx context.Context, channelID uuid.UUID) (int64, error) {
	client := p.redis.Client()

	channelKey := fmt.Sprintf("%s%s", presenceKeyPrefix, channelID.String())
	return client.SCard(ctx, channelKey).Result()
}
