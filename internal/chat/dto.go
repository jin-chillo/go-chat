package chat

import (
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
)

// CreateChannelRequest represents a request to create a new channel.
type CreateChannelRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
}

// UpdateChannelRequest represents a request to update a channel.
type UpdateChannelRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
}

// ChannelResponse represents a channel in API responses.
type ChannelResponse struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	OwnerID     uuid.UUID    `json:"owner_id"`
	Owner       *UserBrief   `json:"owner,omitempty"`
	MemberCount int64        `json:"member_count"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// ChannelDetailResponse represents detailed channel information.
type ChannelDetailResponse struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	OwnerID     uuid.UUID     `json:"owner_id"`
	Owner       *UserBrief    `json:"owner,omitempty"`
	MemberCount int64         `json:"member_count"`
	Members     []MemberBrief `json:"members"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// UserBrief represents minimal user information.
type UserBrief struct {
	ID       uuid.UUID `json:"id"`
	Nickname string    `json:"nickname"`
}

// MemberBrief represents a channel member with basic info.
type MemberBrief struct {
	ID       uuid.UUID `json:"id"`
	Nickname string    `json:"nickname"`
	JoinedAt time.Time `json:"joined_at"`
}

// ChannelListResponse represents a paginated list of channels.
type ChannelListResponse struct {
	Channels []ChannelResponse `json:"channels"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	Limit    int               `json:"limit"`
}

// MessageResponse represents a generic message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// ChatMessageResponse represents a chat message in API responses.
type ChatMessageResponse struct {
	ID        string    `json:"id"`
	ChannelID uuid.UUID `json:"channel_id"`
	UserID    uuid.UUID `json:"user_id"`
	Nickname  string    `json:"nickname"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// MessageListResponse represents a paginated list of messages.
type MessageListResponse struct {
	Messages []ChatMessageResponse `json:"messages"`
	HasMore  bool                  `json:"has_more"`
}

// ToChatMessageResponse converts a Message model to ChatMessageResponse.
func ToChatMessageResponse(msg *model.Message) ChatMessageResponse {
	return ChatMessageResponse{
		ID:        msg.ID.Hex(),
		ChannelID: msg.ChannelID,
		UserID:    msg.UserID,
		Nickname:  msg.Nickname,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
}

// ToChannelResponse converts a Channel model to ChannelResponse.
func ToChannelResponse(channel *model.Channel, memberCount int64) ChannelResponse {
	response := ChannelResponse{
		ID:          channel.ID,
		Name:        channel.Name,
		Description: channel.Description,
		OwnerID:     channel.OwnerID,
		MemberCount: memberCount,
		CreatedAt:   channel.CreatedAt,
		UpdatedAt:   channel.UpdatedAt,
	}

	if channel.Owner != nil {
		response.Owner = &UserBrief{
			ID:       channel.Owner.ID,
			Nickname: channel.Owner.Nickname,
		}
	}

	return response
}

// ToChannelDetailResponse converts a Channel model to ChannelDetailResponse.
func ToChannelDetailResponse(channel *model.Channel) ChannelDetailResponse {
	response := ChannelDetailResponse{
		ID:          channel.ID,
		Name:        channel.Name,
		Description: channel.Description,
		OwnerID:     channel.OwnerID,
		MemberCount: int64(len(channel.Members)),
		Members:     make([]MemberBrief, 0, len(channel.Members)),
		CreatedAt:   channel.CreatedAt,
		UpdatedAt:   channel.UpdatedAt,
	}

	if channel.Owner != nil {
		response.Owner = &UserBrief{
			ID:       channel.Owner.ID,
			Nickname: channel.Owner.Nickname,
		}
	}

	for _, member := range channel.Members {
		if member.User != nil {
			response.Members = append(response.Members, MemberBrief{
				ID:       member.User.ID,
				Nickname: member.User.Nickname,
				JoinedAt: member.JoinedAt,
			})
		}
	}

	return response
}
