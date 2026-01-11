package model

import (
	"time"

	"github.com/google/uuid"
)

// Channel represents a chat channel.
type Channel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	OwnerID     uuid.UUID `gorm:"type:uuid;not null" json:"owner_id"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations
	Owner   *User           `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members []ChannelMember `gorm:"foreignKey:ChannelID" json:"members,omitempty"`
}

// TableName returns the table name for the Channel model.
func (Channel) TableName() string {
	return "channels"
}

// ChannelMember represents a user's membership in a channel.
type ChannelMember struct {
	ChannelID uuid.UUID `gorm:"type:uuid;primaryKey" json:"channel_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	JoinedAt  time.Time `gorm:"autoCreateTime" json:"joined_at"`

	// Associations
	Channel *Channel `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for the ChannelMember model.
func (ChannelMember) TableName() string {
	return "channel_members"
}
