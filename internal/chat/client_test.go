package chat

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestClient_UserID(t *testing.T) {
	userID := uuid.New()
	client := &Client{
		userID:   userID,
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	assert.Equal(t, userID, client.UserID())
}

func TestClient_Nickname(t *testing.T) {
	client := &Client{
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	assert.Equal(t, "testuser", client.Nickname())
}

func TestClient_AddChannel(t *testing.T) {
	channelID := uuid.New()
	client := &Client{
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	client.AddChannel(channelID)

	assert.True(t, client.channels[channelID])
}

func TestClient_RemoveChannel(t *testing.T) {
	channelID := uuid.New()
	client := &Client{
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Add then remove
	client.channels[channelID] = true
	client.RemoveChannel(channelID)

	assert.False(t, client.channels[channelID])
}

func TestClient_IsSubscribed(t *testing.T) {
	channelID := uuid.New()
	otherChannelID := uuid.New()
	client := &Client{
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	client.channels[channelID] = true

	assert.True(t, client.IsSubscribed(channelID))
	assert.False(t, client.IsSubscribed(otherChannelID))
}

func TestClient_SubscribedChannels(t *testing.T) {
	channel1 := uuid.New()
	channel2 := uuid.New()
	client := &Client{
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	client.channels[channel1] = true
	client.channels[channel2] = true

	channels := client.SubscribedChannels()

	assert.Len(t, channels, 2)
	assert.Contains(t, channels, channel1)
	assert.Contains(t, channels, channel2)
}

func TestClient_Send(t *testing.T) {
	client := &Client{
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	message := []byte(`{"type":"test"}`)
	client.Send(message)

	// Message should be in the send channel
	received := <-client.send
	assert.Equal(t, message, received)
}

func TestClient_Send_BufferFull(t *testing.T) {
	// Create client with small buffer
	client := &Client{
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 1), // Small buffer
	}

	// Fill the buffer
	client.send <- []byte(`{"type":"first"}`)

	// This should not block (non-blocking send when buffer is full)
	client.Send([]byte(`{"type":"second"}`))

	// Only first message should be in channel
	received := <-client.send
	assert.Equal(t, []byte(`{"type":"first"}`), received)
}
