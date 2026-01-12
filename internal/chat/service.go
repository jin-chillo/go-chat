package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
)

// Service defines the interface for chat business logic.
type Service interface {
	CreateChannel(ctx context.Context, ownerID uuid.UUID, req *CreateChannelRequest) (*model.Channel, error)
	GetChannel(ctx context.Context, id uuid.UUID) (*model.Channel, error)
	GetChannelDetail(ctx context.Context, id uuid.UUID) (*model.Channel, error)
	ListChannels(ctx context.Context, page, limit int) ([]model.Channel, int64, error)
	ListUserChannels(ctx context.Context, userID uuid.UUID, page, limit int) ([]model.Channel, int64, error)
	UpdateChannel(ctx context.Context, id, userID uuid.UUID, req *UpdateChannelRequest) (*model.Channel, error)
	DeleteChannel(ctx context.Context, id, userID uuid.UUID) error

	JoinChannel(ctx context.Context, channelID, userID uuid.UUID) error
	LeaveChannel(ctx context.Context, channelID, userID uuid.UUID) error
	GetChannelMembers(ctx context.Context, channelID uuid.UUID) ([]model.ChannelMember, error)
	CountMembers(ctx context.Context, channelID uuid.UUID) (int64, error)
	IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error)

	// Message operations
	CreateMessage(ctx context.Context, channelID, userID uuid.UUID, nickname, content string) (*model.Message, error)
	GetMessages(ctx context.Context, channelID uuid.UUID, before *time.Time, limit int) ([]model.Message, bool, error)
}

// service implements Service interface.
type service struct {
	channelRepo ChannelRepository
	messageRepo MessageRepository
}

// NewService creates a new chat service.
func NewService(channelRepo ChannelRepository, messageRepo MessageRepository) Service {
	return &service{
		channelRepo: channelRepo,
		messageRepo: messageRepo,
	}
}

// CreateChannel creates a new channel.
func (s *service) CreateChannel(ctx context.Context, ownerID uuid.UUID, req *CreateChannelRequest) (*model.Channel, error) {
	channel := &model.Channel{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     ownerID,
	}

	if err := s.channelRepo.Create(ctx, channel); err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	return channel, nil
}

// GetChannel retrieves a channel by ID.
func (s *service) GetChannel(ctx context.Context, id uuid.UUID) (*model.Channel, error) {
	channel, err := s.channelRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return channel, nil
}

// GetChannelDetail retrieves a channel with members.
func (s *service) GetChannelDetail(ctx context.Context, id uuid.UUID) (*model.Channel, error) {
	channel, err := s.channelRepo.FindByIDWithMembers(ctx, id)
	if err != nil {
		return nil, err
	}
	return channel, nil
}

// ListChannels retrieves all channels with pagination.
func (s *service) ListChannels(ctx context.Context, page, limit int) ([]model.Channel, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	channels, total, err := s.channelRepo.FindAll(ctx, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list channels: %w", err)
	}

	return channels, total, nil
}

// ListUserChannels retrieves channels that a user is a member of.
func (s *service) ListUserChannels(ctx context.Context, userID uuid.UUID, page, limit int) ([]model.Channel, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	channels, total, err := s.channelRepo.FindByUserID(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list user channels: %w", err)
	}

	return channels, total, nil
}

// UpdateChannel updates a channel.
func (s *service) UpdateChannel(ctx context.Context, id, userID uuid.UUID, req *UpdateChannelRequest) (*model.Channel, error) {
	// Get existing channel
	channel, err := s.channelRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if channel.OwnerID != userID {
		return nil, ErrNotChannelOwner
	}

	// Update fields
	if req.Name != "" {
		channel.Name = req.Name
	}
	if req.Description != "" {
		channel.Description = req.Description
	}

	if err := s.channelRepo.Update(ctx, channel); err != nil {
		return nil, fmt.Errorf("failed to update channel: %w", err)
	}

	return channel, nil
}

// DeleteChannel deletes a channel.
func (s *service) DeleteChannel(ctx context.Context, id, userID uuid.UUID) error {
	// Get existing channel
	channel, err := s.channelRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Check ownership
	if channel.OwnerID != userID {
		return ErrNotChannelOwner
	}

	if err := s.channelRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	return nil
}

// JoinChannel adds a user to a channel.
func (s *service) JoinChannel(ctx context.Context, channelID, userID uuid.UUID) error {
	// Check if channel exists
	_, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return err
	}

	// Add member
	if err := s.channelRepo.AddMember(ctx, channelID, userID); err != nil {
		if errors.Is(err, ErrAlreadyMember) {
			return err
		}
		return fmt.Errorf("failed to join channel: %w", err)
	}

	return nil
}

// LeaveChannel removes a user from a channel.
func (s *service) LeaveChannel(ctx context.Context, channelID, userID uuid.UUID) error {
	// Check if channel exists and get owner info
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return err
	}

	// Owner cannot leave the channel
	if channel.OwnerID == userID {
		return ErrOwnerCannotLeave
	}

	// Remove member
	if err := s.channelRepo.RemoveMember(ctx, channelID, userID); err != nil {
		return err
	}

	return nil
}

// GetChannelMembers retrieves all members of a channel.
func (s *service) GetChannelMembers(ctx context.Context, channelID uuid.UUID) ([]model.ChannelMember, error) {
	// Check if channel exists
	_, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	members, err := s.channelRepo.GetMembers(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel members: %w", err)
	}

	return members, nil
}

// CountMembers counts members in a channel.
func (s *service) CountMembers(ctx context.Context, channelID uuid.UUID) (int64, error) {
	return s.channelRepo.CountMembers(ctx, channelID)
}

// IsMember checks if a user is a member of a channel.
func (s *service) IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	return s.channelRepo.IsMember(ctx, channelID, userID)
}

// CreateMessage creates a new message in a channel.
func (s *service) CreateMessage(ctx context.Context, channelID, userID uuid.UUID, nickname, content string) (*model.Message, error) {
	if s.messageRepo == nil {
		return nil, errors.New("message repository not configured")
	}

	// Check if channel exists
	_, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	// Check if user is a member
	isMember, err := s.channelRepo.IsMember(ctx, channelID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	message := &model.Message{
		ChannelID: channelID,
		UserID:    userID,
		Nickname:  nickname,
		Content:   content,
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return message, nil
}

// GetMessages retrieves messages from a channel with cursor-based pagination.
func (s *service) GetMessages(ctx context.Context, channelID uuid.UUID, before *time.Time, limit int) ([]model.Message, bool, error) {
	if s.messageRepo == nil {
		return nil, false, errors.New("message repository not configured")
	}

	// Check if channel exists
	_, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, false, err
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Request one more to check if there are more messages
	messages, err := s.messageRepo.FindByChannelID(ctx, channelID, before, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get messages: %w", err)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[1:] // Remove the oldest message (first after reverse)
	}

	return messages, hasMore, nil
}
