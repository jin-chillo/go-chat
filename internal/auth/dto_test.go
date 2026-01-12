package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewUserResponse(t *testing.T) {
	userID := uuid.New()
	createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	user := &model.User{
		ID:        userID,
		Email:     "test@example.com",
		Nickname:  "testuser",
		CreatedAt: createdAt,
	}

	response := NewUserResponse(user)

	assert.Equal(t, userID.String(), response.ID)
	assert.Equal(t, "test@example.com", response.Email)
	assert.Equal(t, "testuser", response.Nickname)
	assert.Equal(t, "2024-01-15T10:30:00Z", response.CreatedAt)
}
