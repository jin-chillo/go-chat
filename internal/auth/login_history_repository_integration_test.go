//go:build integration

package auth

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

func TestLoginHistoryRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup MongoDB container
	mongoContainer, err := testutil.SetupMongo(ctx)
	require.NoError(t, err)
	defer mongoContainer.Cleanup(ctx)

	repo := NewLoginHistoryRepository(mongoContainer.Database)
	userID := uuid.New()

	t.Run("Create", func(t *testing.T) {
		history := model.NewLoginHistory(userID, "192.168.1.1", "Mozilla/5.0")

		err := repo.Create(ctx, history)
		require.NoError(t, err)
	})

	t.Run("FindByUserID", func(t *testing.T) {
		testUserID := uuid.New()

		// Create multiple login history records
		for i := 0; i < 5; i++ {
			history := &model.LoginHistory{
				UserID:    testUserID,
				IP:        "192.168.1.1",
				UserAgent: "Mozilla/5.0",
				CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			}
			err := repo.Create(ctx, history)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}

		histories, err := repo.FindByUserID(ctx, testUserID, 10)
		require.NoError(t, err)
		assert.Len(t, histories, 5)

		// Should be ordered by created_at descending (newest first)
		for i := 1; i < len(histories); i++ {
			assert.True(t, histories[i].CreatedAt.Before(histories[i-1].CreatedAt) ||
				histories[i].CreatedAt.Equal(histories[i-1].CreatedAt))
		}
	})

	t.Run("FindByUserID with limit", func(t *testing.T) {
		limitUserID := uuid.New()

		for i := 0; i < 10; i++ {
			history := &model.LoginHistory{
				UserID:    limitUserID,
				IP:        "192.168.1.1",
				UserAgent: "Mozilla/5.0",
				CreatedAt: time.Now(),
			}
			err := repo.Create(ctx, history)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}

		histories, err := repo.FindByUserID(ctx, limitUserID, 5)
		require.NoError(t, err)
		assert.Len(t, histories, 5)
	})

	t.Run("FindByUserID empty result", func(t *testing.T) {
		histories, err := repo.FindByUserID(ctx, uuid.New(), 10)
		require.NoError(t, err)
		assert.Empty(t, histories)
	})

	t.Run("FindByUserID different users", func(t *testing.T) {
		user1ID := uuid.New()
		user2ID := uuid.New()

		// Create history for user1
		for i := 0; i < 3; i++ {
			history := &model.LoginHistory{
				UserID:    user1ID,
				IP:        "192.168.1.1",
				UserAgent: "User1 Agent",
				CreatedAt: time.Now(),
			}
			err := repo.Create(ctx, history)
			require.NoError(t, err)
		}

		// Create history for user2
		for i := 0; i < 2; i++ {
			history := &model.LoginHistory{
				UserID:    user2ID,
				IP:        "192.168.1.2",
				UserAgent: "User2 Agent",
				CreatedAt: time.Now(),
			}
			err := repo.Create(ctx, history)
			require.NoError(t, err)
		}

		// Verify user1 histories
		user1Histories, err := repo.FindByUserID(ctx, user1ID, 10)
		require.NoError(t, err)
		assert.Len(t, user1Histories, 3)
		for _, h := range user1Histories {
			assert.Equal(t, user1ID, h.UserID)
		}

		// Verify user2 histories
		user2Histories, err := repo.FindByUserID(ctx, user2ID, 10)
		require.NoError(t, err)
		assert.Len(t, user2Histories, 2)
		for _, h := range user2Histories {
			assert.Equal(t, user2ID, h.UserID)
		}
	})
}
