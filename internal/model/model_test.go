package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_TableName(t *testing.T) {
	user := User{}
	assert.Equal(t, "users", user.TableName())
}

func TestChannel_TableName(t *testing.T) {
	channel := Channel{}
	assert.Equal(t, "channels", channel.TableName())
}

func TestChannelMember_TableName(t *testing.T) {
	member := ChannelMember{}
	assert.Equal(t, "channel_members", member.TableName())
}

func TestMessage_CollectionName(t *testing.T) {
	message := Message{}
	assert.Equal(t, "messages", message.CollectionName())
}
