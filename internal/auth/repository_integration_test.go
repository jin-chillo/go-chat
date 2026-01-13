//go:build integration

package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/jin-chillo/go-chat/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer, err := testutil.SetupPostgres(ctx)
	require.NoError(t, err)
	defer pgContainer.Cleanup(ctx)

	// Auto-migrate the schema
	err = pgContainer.DB.AutoMigrate(&model.User{}, &model.RefreshToken{})
	require.NoError(t, err)

	repo := NewUserRepository(pgContainer.DB)

	t.Run("Create and FindByID", func(t *testing.T) {
		user := &model.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "testuser",
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
		assert.Equal(t, user.Nickname, found.Nickname)
	})

	t.Run("FindByEmail", func(t *testing.T) {
		user := &model.User{
			ID:           uuid.New(),
			Email:        "findbyemail@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "emailuser",
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.FindByEmail(ctx, user.Email)
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Nickname, found.Nickname)
	})

	t.Run("FindByEmail not found", func(t *testing.T) {
		_, err := repo.FindByEmail(ctx, "notfound@example.com")
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("FindByID not found", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.New())
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("Update", func(t *testing.T) {
		user := &model.User{
			ID:           uuid.New(),
			Email:        "update@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "updateuser",
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		user.Nickname = "updatednickname"
		err = repo.Update(ctx, user)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "updatednickname", found.Nickname)
	})

	t.Run("Create duplicate email", func(t *testing.T) {
		user1 := &model.User{
			ID:           uuid.New(),
			Email:        "duplicate@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "user1",
		}

		err := repo.Create(ctx, user1)
		require.NoError(t, err)

		user2 := &model.User{
			ID:           uuid.New(),
			Email:        "duplicate@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "user2",
		}

		err = repo.Create(ctx, user2)
		assert.ErrorIs(t, err, ErrEmailAlreadyExists)
	})

	t.Run("Update with duplicate email", func(t *testing.T) {
		// Create first user
		user1 := &model.User{
			ID:           uuid.New(),
			Email:        "updatedup1@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "updatedup1",
		}
		err := repo.Create(ctx, user1)
		require.NoError(t, err)

		// Create second user
		user2 := &model.User{
			ID:           uuid.New(),
			Email:        "updatedup2@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "updatedup2",
		}
		err = repo.Create(ctx, user2)
		require.NoError(t, err)

		// Try to update user2's email to user1's email
		user2.Email = "updatedup1@example.com"
		err = repo.Update(ctx, user2)
		assert.ErrorIs(t, err, ErrEmailAlreadyExists)
	})
}

func TestTokenRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer, err := testutil.SetupPostgres(ctx)
	require.NoError(t, err)
	defer pgContainer.Cleanup(ctx)

	// Auto-migrate the schema
	err = pgContainer.DB.AutoMigrate(&model.User{}, &model.RefreshToken{})
	require.NoError(t, err)

	// Create a test user first
	userRepo := NewUserRepository(pgContainer.DB)
	testUser := &model.User{
		ID:           uuid.New(),
		Email:        "tokentest@example.com",
		PasswordHash: "hashedpassword",
		Nickname:     "tokenuser",
	}
	err = userRepo.Create(ctx, testUser)
	require.NoError(t, err)

	repo := NewTokenRepository(pgContainer.DB)

	t.Run("Create and FindByTokenHash", func(t *testing.T) {
		tokenHash := HashToken("test-refresh-token")
		token := &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			TokenHash: tokenHash,
			ExpiresAt: model.RefreshToken{}.ExpiresAt, // Use default
		}

		err := repo.Create(ctx, token)
		require.NoError(t, err)

		found, err := repo.FindByTokenHash(ctx, tokenHash)
		require.NoError(t, err)
		assert.Equal(t, token.ID, found.ID)
		assert.Equal(t, token.UserID, found.UserID)
	})

	t.Run("FindByTokenHash not found", func(t *testing.T) {
		_, err := repo.FindByTokenHash(ctx, "nonexistent-hash")
		assert.ErrorIs(t, err, ErrRefreshTokenNotFound)
	})

	t.Run("RevokeByUserID", func(t *testing.T) {
		// Create another user for this test
		user := &model.User{
			ID:           uuid.New(),
			Email:        "revoke@example.com",
			PasswordHash: "hashedpassword",
			Nickname:     "revokeuser",
		}
		err := userRepo.Create(ctx, user)
		require.NoError(t, err)

		// Create tokens for this user
		tokenHash1 := HashToken("revoke-token-1")
		token1 := &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    user.ID,
			TokenHash: tokenHash1,
		}
		err = repo.Create(ctx, token1)
		require.NoError(t, err)

		tokenHash2 := HashToken("revoke-token-2")
		token2 := &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    user.ID,
			TokenHash: tokenHash2,
		}
		err = repo.Create(ctx, token2)
		require.NoError(t, err)

		// Revoke all tokens for this user
		err = repo.RevokeByUserID(ctx, user.ID)
		require.NoError(t, err)

		// Verify tokens are revoked
		found1, err := repo.FindByTokenHash(ctx, tokenHash1)
		require.NoError(t, err)
		assert.True(t, found1.IsRevoked())

		found2, err := repo.FindByTokenHash(ctx, tokenHash2)
		require.NoError(t, err)
		assert.True(t, found2.IsRevoked())
	})

	t.Run("RevokeByTokenHash", func(t *testing.T) {
		tokenHash := HashToken("single-revoke-token")
		token := &model.RefreshToken{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			TokenHash: tokenHash,
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		err = repo.RevokeByTokenHash(ctx, tokenHash)
		require.NoError(t, err)

		found, err := repo.FindByTokenHash(ctx, tokenHash)
		require.NoError(t, err)
		assert.True(t, found.IsRevoked())
	})
}
