//go:build integration

package chat

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup MongoDB container
	mongoContainer, err := testutil.SetupMongo(ctx)
	require.NoError(t, err)
	defer mongoContainer.Cleanup(ctx)

	repo := NewMessageRepository(mongoContainer.Database)
	channelID := uuid.New()
	userID := uuid.New()

	t.Run("Create", func(t *testing.T) {
		message := &model.Message{
			ChannelID: channelID,
			UserID:    userID,
			Nickname:  "testuser",
			Content:   "Hello, World!",
		}

		err := repo.Create(ctx, message)
		require.NoError(t, err)
		assert.NotEmpty(t, message.ID)
		assert.False(t, message.CreatedAt.IsZero())
	})

	t.Run("FindByChannelID", func(t *testing.T) {
		// Create multiple messages
		testChannelID := uuid.New()
		for i := 0; i < 5; i++ {
			message := &model.Message{
				ChannelID: testChannelID,
				UserID:    userID,
				Nickname:  "testuser",
				Content:   "Message content",
				CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			}
			err := repo.Create(ctx, message)
			require.NoError(t, err)
			// Small delay to ensure ordering
			time.Sleep(10 * time.Millisecond)
		}

		messages, err := repo.FindByChannelID(ctx, testChannelID, nil, 10)
		require.NoError(t, err)
		assert.Len(t, messages, 5)

		// Messages should be in chronological order (oldest first)
		for i := 1; i < len(messages); i++ {
			assert.True(t, messages[i].CreatedAt.After(messages[i-1].CreatedAt) ||
				messages[i].CreatedAt.Equal(messages[i-1].CreatedAt))
		}
	})

	t.Run("FindByChannelID with limit", func(t *testing.T) {
		limitChannelID := uuid.New()
		for i := 0; i < 10; i++ {
			message := &model.Message{
				ChannelID: limitChannelID,
				UserID:    userID,
				Nickname:  "testuser",
				Content:   "Message content",
			}
			err := repo.Create(ctx, message)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}

		messages, err := repo.FindByChannelID(ctx, limitChannelID, nil, 5)
		require.NoError(t, err)
		assert.Len(t, messages, 5)
	})

	t.Run("FindByChannelID with cursor pagination", func(t *testing.T) {
		cursorChannelID := uuid.New()
		baseTime := time.Now()

		// Create messages with specific timestamps
		for i := 0; i < 10; i++ {
			message := &model.Message{
				ChannelID: cursorChannelID,
				UserID:    userID,
				Nickname:  "testuser",
				Content:   "Message content",
				CreatedAt: baseTime.Add(time.Duration(i) * time.Minute),
			}
			err := repo.Create(ctx, message)
			require.NoError(t, err)
		}

		// Get first page
		messages, err := repo.FindByChannelID(ctx, cursorChannelID, nil, 5)
		require.NoError(t, err)
		assert.Len(t, messages, 5)

		// Get second page using cursor (before the oldest message of first page)
		cursor := messages[0].CreatedAt
		olderMessages, err := repo.FindByChannelID(ctx, cursorChannelID, &cursor, 5)
		require.NoError(t, err)
		assert.Len(t, olderMessages, 5)

		// All messages in second page should be older than cursor
		for _, msg := range olderMessages {
			assert.True(t, msg.CreatedAt.Before(cursor))
		}
	})

	t.Run("FindByChannelID empty result", func(t *testing.T) {
		messages, err := repo.FindByChannelID(ctx, uuid.New(), nil, 10)
		require.NoError(t, err)
		assert.Empty(t, messages)
	})

	t.Run("FindByChannelID default limit", func(t *testing.T) {
		// Test with invalid limit (should default to 50)
		messages, err := repo.FindByChannelID(ctx, channelID, nil, 0)
		require.NoError(t, err)
		assert.NotNil(t, messages)

		messages, err = repo.FindByChannelID(ctx, channelID, nil, -1)
		require.NoError(t, err)
		assert.NotNil(t, messages)
	})

	t.Run("CountByChannelID", func(t *testing.T) {
		countChannelID := uuid.New()

		// Initially zero
		count, err := repo.CountByChannelID(ctx, countChannelID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		// Add messages
		for i := 0; i < 3; i++ {
			message := &model.Message{
				ChannelID: countChannelID,
				UserID:    userID,
				Nickname:  "testuser",
				Content:   "Message content",
			}
			err := repo.Create(ctx, message)
			require.NoError(t, err)
		}

		count, err = repo.CountByChannelID(ctx, countChannelID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})
}
