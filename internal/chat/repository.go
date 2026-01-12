package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"gorm.io/gorm"
)

var (
	// ErrChannelNotFound is returned when a channel is not found.
	ErrChannelNotFound = errors.New("channel not found")
	// ErrNotChannelOwner is returned when user is not the channel owner.
	ErrNotChannelOwner = errors.New("not channel owner")
	// ErrAlreadyMember is returned when user is already a member of the channel.
	ErrAlreadyMember = errors.New("already a member of this channel")
	// ErrNotMember is returned when user is not a member of the channel.
	ErrNotMember = errors.New("not a member of this channel")
	// ErrOwnerCannotLeave is returned when channel owner tries to leave.
	ErrOwnerCannotLeave = errors.New("channel owner cannot leave the channel")
)

// ChannelRepository defines the interface for channel data access.
type ChannelRepository interface {
	Create(ctx context.Context, channel *model.Channel) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Channel, error)
	FindByIDWithMembers(ctx context.Context, id uuid.UUID) (*model.Channel, error)
	FindAll(ctx context.Context, offset, limit int) ([]model.Channel, int64, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.Channel, int64, error)
	Update(ctx context.Context, channel *model.Channel) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Member operations
	AddMember(ctx context.Context, channelID, userID uuid.UUID) error
	RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error
	IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	GetMembers(ctx context.Context, channelID uuid.UUID) ([]model.ChannelMember, error)
	CountMembers(ctx context.Context, channelID uuid.UUID) (int64, error)
}

// channelRepository implements ChannelRepository using GORM.
type channelRepository struct {
	db *gorm.DB
}

// NewChannelRepository creates a new ChannelRepository instance.
func NewChannelRepository(db *gorm.DB) ChannelRepository {
	return &channelRepository{db: db}
}

// Create inserts a new channel into the database.
func (r *channelRepository) Create(ctx context.Context, channel *model.Channel) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create the channel
		if err := tx.Create(channel).Error; err != nil {
			return fmt.Errorf("failed to create channel: %w", err)
		}

		// Add owner as a member
		member := &model.ChannelMember{
			ChannelID: channel.ID,
			UserID:    channel.OwnerID,
		}
		if err := tx.Create(member).Error; err != nil {
			return fmt.Errorf("failed to add owner as member: %w", err)
		}

		return nil
	})
}

// FindByID retrieves a channel by its ID.
func (r *channelRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Channel, error) {
	var channel model.Channel
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&channel)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrChannelNotFound
		}
		return nil, fmt.Errorf("failed to find channel: %w", result.Error)
	}
	return &channel, nil
}

// FindByIDWithMembers retrieves a channel by its ID with members preloaded.
func (r *channelRepository) FindByIDWithMembers(ctx context.Context, id uuid.UUID) (*model.Channel, error) {
	var channel model.Channel
	result := r.db.WithContext(ctx).
		Preload("Owner").
		Preload("Members").
		Preload("Members.User").
		Where("id = ?", id).
		First(&channel)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrChannelNotFound
		}
		return nil, fmt.Errorf("failed to find channel: %w", result.Error)
	}
	return &channel, nil
}

// FindAll retrieves all channels with pagination.
func (r *channelRepository) FindAll(ctx context.Context, offset, limit int) ([]model.Channel, int64, error) {
	var channels []model.Channel
	var total int64

	// Count total
	if err := r.db.WithContext(ctx).Model(&model.Channel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count channels: %w", err)
	}

	// Fetch channels
	result := r.db.WithContext(ctx).
		Preload("Owner").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&channels)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("failed to find channels: %w", result.Error)
	}

	return channels, total, nil
}

// FindByUserID retrieves channels that a user is a member of.
func (r *channelRepository) FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.Channel, int64, error) {
	var channels []model.Channel
	var total int64

	// Subquery for user's channel IDs
	subquery := r.db.Model(&model.ChannelMember{}).
		Select("channel_id").
		Where("user_id = ?", userID)

	// Count total
	if err := r.db.WithContext(ctx).
		Model(&model.Channel{}).
		Where("id IN (?)", subquery).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count user channels: %w", err)
	}

	// Fetch channels
	result := r.db.WithContext(ctx).
		Preload("Owner").
		Where("id IN (?)", subquery).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&channels)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("failed to find user channels: %w", result.Error)
	}

	return channels, total, nil
}

// Update updates an existing channel.
func (r *channelRepository) Update(ctx context.Context, channel *model.Channel) error {
	result := r.db.WithContext(ctx).
		Model(channel).
		Select("name", "description", "updated_at").
		Updates(channel)
	if result.Error != nil {
		return fmt.Errorf("failed to update channel: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrChannelNotFound
	}
	return nil
}

// Delete removes a channel from the database.
func (r *channelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Channel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete channel: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrChannelNotFound
	}
	return nil
}

// AddMember adds a user to a channel.
func (r *channelRepository) AddMember(ctx context.Context, channelID, userID uuid.UUID) error {
	member := &model.ChannelMember{
		ChannelID: channelID,
		UserID:    userID,
	}

	result := r.db.WithContext(ctx).Create(member)
	if result.Error != nil {
		// Check for duplicate key error
		if isDuplicateKeyError(result.Error) {
			return ErrAlreadyMember
		}
		return fmt.Errorf("failed to add member: %w", result.Error)
	}
	return nil
}

// RemoveMember removes a user from a channel.
func (r *channelRepository) RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Delete(&model.ChannelMember{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotMember
	}
	return nil
}

// IsMember checks if a user is a member of a channel.
func (r *channelRepository) IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.ChannelMember{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("failed to check membership: %w", result.Error)
	}
	return count > 0, nil
}

// GetMembers retrieves all members of a channel.
func (r *channelRepository) GetMembers(ctx context.Context, channelID uuid.UUID) ([]model.ChannelMember, error) {
	var members []model.ChannelMember
	result := r.db.WithContext(ctx).
		Preload("User").
		Where("channel_id = ?", channelID).
		Order("joined_at ASC").
		Find(&members)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get members: %w", result.Error)
	}
	return members, nil
}

// CountMembers counts the number of members in a channel.
func (r *channelRepository) CountMembers(ctx context.Context, channelID uuid.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.ChannelMember{}).
		Where("channel_id = ?", channelID).
		Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count members: %w", result.Error)
	}
	return count, nil
}

// isDuplicateKeyError checks if the error is a duplicate key violation.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint")
}
