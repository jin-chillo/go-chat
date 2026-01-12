//go:build integration

package chat

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/cache"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresenceService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup Redis container
	redisContainer, err := testutil.SetupRedis(ctx)
	require.NoError(t, err)
	defer redisContainer.Cleanup(ctx)

	// Create RedisClient wrapper
	redisClient := cache.NewRedisClientForTest(redisContainer.Client)

	// Create PresenceService
	service := NewPresenceService(redisClient)

	channelID := uuid.New()
	userID := uuid.New()
	nickname := "testuser"

	t.Run("SetOnline and IsOnline", func(t *testing.T) {
		// Initially not online
		online, err := service.IsOnline(ctx, channelID, userID)
		require.NoError(t, err)
		assert.False(t, online)

		// Set online
		err = service.SetOnline(ctx, channelID, userID, nickname)
		require.NoError(t, err)

		// Now should be online
		online, err = service.IsOnline(ctx, channelID, userID)
		require.NoError(t, err)
		assert.True(t, online)
	})

	t.Run("GetOnlineUsers", func(t *testing.T) {
		testChannelID := uuid.New()
		user1ID := uuid.New()
		user2ID := uuid.New()

		// Add two users
		err := service.SetOnline(ctx, testChannelID, user1ID, "user1")
		require.NoError(t, err)
		err = service.SetOnline(ctx, testChannelID, user2ID, "user2")
		require.NoError(t, err)

		// Get online users
		users, err := service.GetOnlineUsers(ctx, testChannelID)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		// Verify nicknames
		nicknames := make([]string, 0, 2)
		for _, u := range users {
			nicknames = append(nicknames, u.Nickname)
		}
		assert.Contains(t, nicknames, "user1")
		assert.Contains(t, nicknames, "user2")
	})

	t.Run("GetOnlineUsers empty channel", func(t *testing.T) {
		emptyChannelID := uuid.New()

		users, err := service.GetOnlineUsers(ctx, emptyChannelID)
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("SetOffline", func(t *testing.T) {
		offlineChannelID := uuid.New()
		offlineUserID := uuid.New()

		// Set online first
		err := service.SetOnline(ctx, offlineChannelID, offlineUserID, "offlineuser")
		require.NoError(t, err)

		online, err := service.IsOnline(ctx, offlineChannelID, offlineUserID)
		require.NoError(t, err)
		assert.True(t, online)

		// Set offline
		err = service.SetOffline(ctx, offlineChannelID, offlineUserID)
		require.NoError(t, err)

		// Should no longer be online
		online, err = service.IsOnline(ctx, offlineChannelID, offlineUserID)
		require.NoError(t, err)
		assert.False(t, online)
	})

	t.Run("SetOfflineFromAllChannels", func(t *testing.T) {
		channel1 := uuid.New()
		channel2 := uuid.New()
		channel3 := uuid.New()
		multiUserID := uuid.New()

		// Set online in multiple channels
		err := service.SetOnline(ctx, channel1, multiUserID, "multiuser")
		require.NoError(t, err)
		err = service.SetOnline(ctx, channel2, multiUserID, "multiuser")
		require.NoError(t, err)
		err = service.SetOnline(ctx, channel3, multiUserID, "multiuser")
		require.NoError(t, err)

		// Verify online in all channels
		for _, ch := range []uuid.UUID{channel1, channel2, channel3} {
			online, err := service.IsOnline(ctx, ch, multiUserID)
			require.NoError(t, err)
			assert.True(t, online)
		}

		// Set offline from all channels
		err = service.SetOfflineFromAllChannels(ctx, multiUserID, []uuid.UUID{channel1, channel2, channel3})
		require.NoError(t, err)

		// Verify offline in all channels
		for _, ch := range []uuid.UUID{channel1, channel2, channel3} {
			online, err := service.IsOnline(ctx, ch, multiUserID)
			require.NoError(t, err)
			assert.False(t, online)
		}
	})

	t.Run("GetOnlineCount", func(t *testing.T) {
		countChannelID := uuid.New()

		// Initially 0
		count, err := service.GetOnlineCount(ctx, countChannelID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		// Add users
		for i := 0; i < 5; i++ {
			err := service.SetOnline(ctx, countChannelID, uuid.New(), "user")
			require.NoError(t, err)
		}

		count, err = service.GetOnlineCount(ctx, countChannelID)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("RefreshPresence", func(t *testing.T) {
		refreshChannelID := uuid.New()
		refreshUserID := uuid.New()

		// Set online
		err := service.SetOnline(ctx, refreshChannelID, refreshUserID, "refreshuser")
		require.NoError(t, err)

		// Refresh should not error
		err = service.RefreshPresence(ctx, refreshChannelID, refreshUserID)
		require.NoError(t, err)

		// Should still be online
		online, err := service.IsOnline(ctx, refreshChannelID, refreshUserID)
		require.NoError(t, err)
		assert.True(t, online)
	})
}
