package chat

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestToChatMessageResponse(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()
	createdAt := time.Now()
	objID := bson.NewObjectID()

	msg := &model.Message{
		ID:        objID,
		ChannelID: channelID,
		UserID:    userID,
		Nickname:  "testuser",
		Content:   "Hello, world!",
		CreatedAt: createdAt,
	}

	response := ToChatMessageResponse(msg)

	assert.Equal(t, objID.Hex(), response.ID)
	assert.Equal(t, channelID, response.ChannelID)
	assert.Equal(t, userID, response.UserID)
	assert.Equal(t, "testuser", response.Nickname)
	assert.Equal(t, "Hello, world!", response.Content)
	assert.Equal(t, createdAt, response.CreatedAt)
}

func TestToChannelResponse(t *testing.T) {
	channelID := uuid.New()
	ownerID := uuid.New()
	createdAt := time.Now()
	updatedAt := time.Now()

	tests := []struct {
		name        string
		channel     *model.Channel
		memberCount int64
		hasOwner    bool
	}{
		{
			name: "with owner",
			channel: &model.Channel{
				ID:          channelID,
				Name:        "Test Channel",
				Description: "A test channel",
				OwnerID:     ownerID,
				Owner: &model.User{
					ID:       ownerID,
					Nickname: "owner",
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			memberCount: 5,
			hasOwner:    true,
		},
		{
			name: "without owner",
			channel: &model.Channel{
				ID:          channelID,
				Name:        "Test Channel",
				Description: "A test channel",
				OwnerID:     ownerID,
				Owner:       nil,
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
			},
			memberCount: 3,
			hasOwner:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ToChannelResponse(tt.channel, tt.memberCount)

			assert.Equal(t, channelID, response.ID)
			assert.Equal(t, "Test Channel", response.Name)
			assert.Equal(t, "A test channel", response.Description)
			assert.Equal(t, ownerID, response.OwnerID)
			assert.Equal(t, tt.memberCount, response.MemberCount)

			if tt.hasOwner {
				assert.NotNil(t, response.Owner)
				assert.Equal(t, ownerID, response.Owner.ID)
				assert.Equal(t, "owner", response.Owner.Nickname)
			} else {
				assert.Nil(t, response.Owner)
			}
		})
	}
}

func TestToChannelDetailResponse(t *testing.T) {
	channelID := uuid.New()
	ownerID := uuid.New()
	member1ID := uuid.New()
	member2ID := uuid.New()
	joinedAt := time.Now()

	tests := []struct {
		name     string
		channel  *model.Channel
		wantLen  int
	}{
		{
			name: "with members",
			channel: &model.Channel{
				ID:          channelID,
				Name:        "Test Channel",
				Description: "A test channel",
				OwnerID:     ownerID,
				Owner: &model.User{
					ID:       ownerID,
					Nickname: "owner",
				},
				Members: []model.ChannelMember{
					{
						ChannelID: channelID,
						UserID:    member1ID,
						JoinedAt:  joinedAt,
						User: &model.User{
							ID:       member1ID,
							Nickname: "member1",
						},
					},
					{
						ChannelID: channelID,
						UserID:    member2ID,
						JoinedAt:  joinedAt,
						User: &model.User{
							ID:       member2ID,
							Nickname: "member2",
						},
					},
				},
			},
			wantLen: 2,
		},
		{
			name: "with member having nil user",
			channel: &model.Channel{
				ID:          channelID,
				Name:        "Test Channel",
				OwnerID:     ownerID,
				Members: []model.ChannelMember{
					{
						ChannelID: channelID,
						UserID:    member1ID,
						JoinedAt:  joinedAt,
						User:      nil, // User not loaded
					},
				},
			},
			wantLen: 0,
		},
		{
			name: "without members",
			channel: &model.Channel{
				ID:          channelID,
				Name:        "Empty Channel",
				OwnerID:     ownerID,
				Members:     []model.ChannelMember{},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := ToChannelDetailResponse(tt.channel)

			assert.Equal(t, channelID, response.ID)
			assert.Equal(t, tt.channel.Name, response.Name)
			assert.Len(t, response.Members, tt.wantLen)
		})
	}
}
