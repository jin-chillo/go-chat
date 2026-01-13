package chat

import (
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
)

// CreateChannelRequest represents a request to create a new channel.
// @Description 채널 생성 요청 데이터
type CreateChannelRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100" example:"일반 채팅방"`
	Description string `json:"description" binding:"max=500" example:"자유롭게 대화하는 채널입니다"`
}

// UpdateChannelRequest represents a request to update a channel.
// @Description 채널 수정 요청 데이터
type UpdateChannelRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100" example:"수정된 채널명"`
	Description string `json:"description" binding:"max=500" example:"수정된 설명"`
}

// ChannelResponse represents a channel in API responses.
// @Description 채널 응답 데이터
type ChannelResponse struct {
	ID          uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string     `json:"name" example:"일반 채팅방"`
	Description string     `json:"description,omitempty" example:"자유롭게 대화하는 채널입니다"`
	OwnerID     uuid.UUID  `json:"owner_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Owner       *UserBrief `json:"owner,omitempty"`
	MemberCount int64      `json:"member_count" example:"5"`
	CreatedAt   time.Time  `json:"created_at" example:"2024-01-15T09:30:00Z"`
	UpdatedAt   time.Time  `json:"updated_at" example:"2024-01-15T10:00:00Z"`
}

// ChannelDetailResponse represents detailed channel information.
// @Description 채널 상세 응답 데이터
type ChannelDetailResponse struct {
	ID          uuid.UUID     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string        `json:"name" example:"일반 채팅방"`
	Description string        `json:"description,omitempty" example:"자유롭게 대화하는 채널입니다"`
	OwnerID     uuid.UUID     `json:"owner_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Owner       *UserBrief    `json:"owner,omitempty"`
	MemberCount int64         `json:"member_count" example:"5"`
	Members     []MemberBrief `json:"members"`
	CreatedAt   time.Time     `json:"created_at" example:"2024-01-15T09:30:00Z"`
	UpdatedAt   time.Time     `json:"updated_at" example:"2024-01-15T10:00:00Z"`
}

// UserBrief represents minimal user information.
// @Description 간략한 사용자 정보
type UserBrief struct {
	ID       uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Nickname string    `json:"nickname" example:"홍길동"`
}

// MemberBrief represents a channel member with basic info.
// @Description 채널 멤버 정보
type MemberBrief struct {
	ID       uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Nickname string    `json:"nickname" example:"홍길동"`
	JoinedAt time.Time `json:"joined_at" example:"2024-01-15T09:30:00Z"`
}

// ChannelListResponse represents a paginated list of channels.
// @Description 채널 목록 응답 데이터
type ChannelListResponse struct {
	Channels []ChannelResponse `json:"channels"`
	Total    int64             `json:"total" example:"100"`
	Page     int               `json:"page" example:"1"`
	Limit    int               `json:"limit" example:"20"`
}

// MessageResponse represents a generic message response.
// @Description 일반 메시지 응답
type MessageResponse struct {
	Message string `json:"message" example:"작업이 완료되었습니다"`
}

// ChatMessageResponse represents a chat message in API responses.
// @Description 채팅 메시지 응답 데이터
type ChatMessageResponse struct {
	ID        string    `json:"id" example:"507f1f77bcf86cd799439011"`
	ChannelID uuid.UUID `json:"channel_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    uuid.UUID `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Nickname  string    `json:"nickname" example:"홍길동"`
	Content   string    `json:"content" example:"안녕하세요!"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T09:30:00Z"`
}

// MessageListResponse represents a paginated list of messages.
// @Description 메시지 목록 응답 데이터
type MessageListResponse struct {
	Messages []ChatMessageResponse `json:"messages"`
	HasMore  bool                  `json:"has_more" example:"true"`
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
