//go:build integration

package chat

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer, err := testutil.SetupPostgres(ctx)
	require.NoError(t, err)
	defer pgContainer.Cleanup(ctx)

	// Auto-migrate the schema
	err = pgContainer.DB.AutoMigrate(&model.User{}, &model.Channel{}, &model.ChannelMember{})
	require.NoError(t, err)

	// Create a test user first
	testUser := &model.User{
		ID:           uuid.New(),
		Email:        "channeltest@example.com",
		PasswordHash: "hashedpassword",
		Nickname:     "channeluser",
	}
	err = pgContainer.DB.Create(testUser).Error
	require.NoError(t, err)

	repo := NewChannelRepository(pgContainer.DB)

	t.Run("Create and FindByID", func(t *testing.T) {
		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Test Channel",
			Description: "A test channel",
			OwnerID:     testUser.ID,
		}

		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, channel.Name, found.Name)
		assert.Equal(t, channel.Description, found.Description)
		assert.Equal(t, channel.OwnerID, found.OwnerID)
	})

	t.Run("FindByID not found", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.New())
		assert.ErrorIs(t, err, ErrChannelNotFound)
	})

	t.Run("FindByIDWithMembers", func(t *testing.T) {
		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Members Channel",
			Description: "Channel with members",
			OwnerID:     testUser.ID,
		}
		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		found, err := repo.FindByIDWithMembers(ctx, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, channel.Name, found.Name)
		// Owner should be loaded
		assert.NotNil(t, found.Owner)
		// Owner is auto-added as member
		assert.Len(t, found.Members, 1)
	})

	t.Run("FindAll", func(t *testing.T) {
		// Create more channels
		for i := 0; i < 3; i++ {
			channel := &model.Channel{
				ID:          uuid.New(),
				Name:        "FindAll Channel",
				Description: "Findall test",
				OwnerID:     testUser.ID,
			}
			err := repo.Create(ctx, channel)
			require.NoError(t, err)
		}

		channels, total, err := repo.FindAll(ctx, 0, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, int(total), 3)
		assert.NotEmpty(t, channels)
	})

	t.Run("Update", func(t *testing.T) {
		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Update Channel",
			Description: "Before update",
			OwnerID:     testUser.ID,
		}
		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		channel.Name = "Updated Channel"
		channel.Description = "After update"
		err = repo.Update(ctx, channel)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Channel", found.Name)
		assert.Equal(t, "After update", found.Description)
	})

	t.Run("Delete", func(t *testing.T) {
		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Delete Channel",
			Description: "To be deleted",
			OwnerID:     testUser.ID,
		}
		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		// Remove member first (owner is auto-added as member)
		err = repo.RemoveMember(ctx, channel.ID, testUser.ID)
		require.NoError(t, err)

		err = repo.Delete(ctx, channel.ID)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, channel.ID)
		assert.ErrorIs(t, err, ErrChannelNotFound)
	})

	t.Run("AddMember and IsMember", func(t *testing.T) {
		// Create another user
		anotherUser := &model.User{
			ID:           uuid.New(),
			Email:        "member@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "memberuser",
		}
		err := pgContainer.DB.Create(anotherUser).Error
		require.NoError(t, err)

		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Member Channel",
			Description: "For membership test",
			OwnerID:     testUser.ID,
		}
		err = repo.Create(ctx, channel)
		require.NoError(t, err)

		// Check owner is member
		isMember, err := repo.IsMember(ctx, channel.ID, testUser.ID)
		require.NoError(t, err)
		assert.True(t, isMember)

		// Another user is not member yet
		isMember, err = repo.IsMember(ctx, channel.ID, anotherUser.ID)
		require.NoError(t, err)
		assert.False(t, isMember)

		// Add member
		err = repo.AddMember(ctx, channel.ID, anotherUser.ID)
		require.NoError(t, err)

		// Now should be member
		isMember, err = repo.IsMember(ctx, channel.ID, anotherUser.ID)
		require.NoError(t, err)
		assert.True(t, isMember)
	})

	t.Run("AddMember duplicate", func(t *testing.T) {
		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Duplicate Member Channel",
			Description: "For duplicate test",
			OwnerID:     testUser.ID,
		}
		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		// Try to add owner again (already a member)
		err = repo.AddMember(ctx, channel.ID, testUser.ID)
		assert.ErrorIs(t, err, ErrAlreadyMember)
	})

	t.Run("RemoveMember", func(t *testing.T) {
		// Create another user
		removeUser := &model.User{
			ID:           uuid.New(),
			Email:        "remove@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "removeuser",
		}
		err := pgContainer.DB.Create(removeUser).Error
		require.NoError(t, err)

		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Remove Member Channel",
			Description: "For remove test",
			OwnerID:     testUser.ID,
		}
		err = repo.Create(ctx, channel)
		require.NoError(t, err)

		// Add and then remove
		err = repo.AddMember(ctx, channel.ID, removeUser.ID)
		require.NoError(t, err)

		err = repo.RemoveMember(ctx, channel.ID, removeUser.ID)
		require.NoError(t, err)

		// Should no longer be member
		isMember, err := repo.IsMember(ctx, channel.ID, removeUser.ID)
		require.NoError(t, err)
		assert.False(t, isMember)
	})

	t.Run("GetMembers", func(t *testing.T) {
		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Get Members Channel",
			Description: "For get members test",
			OwnerID:     testUser.ID,
		}
		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		members, err := repo.GetMembers(ctx, channel.ID)
		require.NoError(t, err)
		assert.Len(t, members, 1) // Owner only
	})

	t.Run("CountMembers", func(t *testing.T) {
		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Count Members Channel",
			Description: "For count test",
			OwnerID:     testUser.ID,
		}
		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		count, err := repo.CountMembers(ctx, channel.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("FindByUserID", func(t *testing.T) {
		// Create user-specific channel
		userChannel := &model.Channel{
			ID:          uuid.New(),
			Name:        "User Channel",
			Description: "For FindByUserID test",
			OwnerID:     testUser.ID,
		}
		err := repo.Create(ctx, userChannel)
		require.NoError(t, err)

		channels, total, err := repo.FindByUserID(ctx, testUser.ID, 0, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, int(total), 1)
		assert.NotEmpty(t, channels)
	})

	t.Run("FindByIDWithMembers not found", func(t *testing.T) {
		_, err := repo.FindByIDWithMembers(ctx, uuid.New())
		assert.ErrorIs(t, err, ErrChannelNotFound)
	})

	t.Run("Update not found", func(t *testing.T) {
		nonExistentChannel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Non-existent",
			Description: "Does not exist",
		}
		err := repo.Update(ctx, nonExistentChannel)
		assert.ErrorIs(t, err, ErrChannelNotFound)
	})

	t.Run("Delete not found", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		assert.ErrorIs(t, err, ErrChannelNotFound)
	})

	t.Run("RemoveMember not member", func(t *testing.T) {
		channel := &model.Channel{
			ID:          uuid.New(),
			Name:        "Not Member Channel",
			Description: "For not member test",
			OwnerID:     testUser.ID,
		}
		err := repo.Create(ctx, channel)
		require.NoError(t, err)

		// Try to remove a user who is not a member
		err = repo.RemoveMember(ctx, channel.ID, uuid.New())
		assert.ErrorIs(t, err, ErrNotMember)
	})
}
