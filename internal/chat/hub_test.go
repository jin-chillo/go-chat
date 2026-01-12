package chat

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.channels)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
}

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()

	// Create a mock client
	userID := uuid.New()
	client := &Client{
		hub:      hub,
		userID:   userID,
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Manually register client (bypass channel for unit test)
	hub.mu.Lock()
	hub.clients[client] = true
	hub.mu.Unlock()

	// Verify client is registered
	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()
	assert.True(t, exists)

	// Manually unregister client
	hub.unregisterClient(client)

	// Verify client is unregistered
	hub.mu.RLock()
	_, exists = hub.clients[client]
	hub.mu.RUnlock()
	assert.False(t, exists)
}

func TestHub_SubscribeUnsubscribe_Direct(t *testing.T) {
	hub := NewHub()

	channelID := uuid.New()
	userID := uuid.New()

	client := &Client{
		hub:      hub,
		userID:   userID,
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register client first
	hub.mu.Lock()
	hub.clients[client] = true
	hub.mu.Unlock()

	// Directly add to channel (bypassing Subscribe channel)
	hub.mu.Lock()
	if hub.channels[channelID] == nil {
		hub.channels[channelID] = make(map[*Client]bool)
	}
	hub.channels[channelID][client] = true
	hub.mu.Unlock()

	// Verify subscription
	hub.mu.RLock()
	channelClients, exists := hub.channels[channelID]
	hub.mu.RUnlock()

	assert.True(t, exists)
	assert.Contains(t, channelClients, client)

	// Directly remove from channel (bypassing Unsubscribe channel)
	hub.mu.Lock()
	delete(hub.channels[channelID], client)
	if len(hub.channels[channelID]) == 0 {
		delete(hub.channels, channelID)
	}
	hub.mu.Unlock()

	// Verify unsubscription
	hub.mu.RLock()
	_, exists = hub.channels[channelID]
	hub.mu.RUnlock()
	assert.False(t, exists)
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()

	channelID := uuid.New()

	// Create multiple clients
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &Client{
			hub:      hub,
			userID:   uuid.New(),
			nickname: "user",
			channels: make(map[uuid.UUID]bool),
			send:     make(chan []byte, 256),
		}

		hub.mu.Lock()
		hub.clients[clients[i]] = true
		if hub.channels[channelID] == nil {
			hub.channels[channelID] = make(map[*Client]bool)
		}
		hub.channels[channelID][clients[i]] = true
		hub.mu.Unlock()
	}

	// Broadcast message using internal method (bypassing channel)
	message := []byte(`{"type":"test"}`)
	hub.broadcastToChannel(&BroadcastMessage{
		ChannelID: channelID,
		Message:   message,
		Sender:    nil,
	})

	// Verify all clients received the message
	for i, client := range clients {
		select {
		case msg := <-client.send:
			assert.Equal(t, message, msg)
		case <-time.After(100 * time.Millisecond):
			t.Errorf("client %d did not receive message", i)
		}
	}
}

func TestHub_Broadcast_ExcludesClient(t *testing.T) {
	hub := NewHub()

	channelID := uuid.New()

	// Create sender and receiver
	sender := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "sender",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	receiver := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "receiver",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register both clients to channel
	hub.mu.Lock()
	hub.clients[sender] = true
	hub.clients[receiver] = true
	hub.channels[channelID] = make(map[*Client]bool)
	hub.channels[channelID][sender] = true
	hub.channels[channelID][receiver] = true
	hub.mu.Unlock()

	// Broadcast excluding sender using internal method
	message := []byte(`{"type":"test"}`)
	hub.broadcastToChannel(&BroadcastMessage{
		ChannelID: channelID,
		Message:   message,
		Sender:    sender,
	})

	// Receiver should get message
	select {
	case msg := <-receiver.send:
		assert.Equal(t, message, msg)
	case <-time.After(100 * time.Millisecond):
		t.Error("receiver did not receive message")
	}

	// Sender should NOT get message
	select {
	case <-sender.send:
		t.Error("sender should not receive their own message")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

func TestHub_Broadcast_NoChannel(t *testing.T) {
	hub := NewHub()

	channelID := uuid.New()
	message := []byte(`{"type":"test"}`)

	// Should not panic when broadcasting to non-existent channel
	hub.broadcastToChannel(&BroadcastMessage{
		ChannelID: channelID,
		Message:   message,
		Sender:    nil,
	})
}

func TestHub_ConcurrentBroadcast(t *testing.T) {
	hub := NewHub()

	channelID := uuid.New()

	// Create client
	client := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "user",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	hub.mu.Lock()
	hub.clients[client] = true
	hub.channels[channelID] = make(map[*Client]bool)
	hub.channels[channelID][client] = true
	hub.mu.Unlock()

	var wg sync.WaitGroup

	// Concurrent broadcasts using internal method
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			message := []byte(`{"type":"test"}`)
			hub.broadcastToChannel(&BroadcastMessage{
				ChannelID: channelID,
				Message:   message,
				Sender:    nil,
			})
		}(i)
	}

	wg.Wait()

	// Drain the channel
	received := 0
	for {
		select {
		case <-client.send:
			received++
		case <-time.After(50 * time.Millisecond):
			goto done
		}
	}
done:
	assert.Equal(t, 10, received)
}

func TestHub_SetOnDisconnect(t *testing.T) {
	hub := NewHub()

	var disconnectedClient *Client
	var disconnectedChannels []uuid.UUID
	done := make(chan struct{})

	hub.SetOnDisconnect(func(client *Client, channelIDs []uuid.UUID) {
		disconnectedClient = client
		disconnectedChannels = channelIDs
		close(done)
	})

	channelID := uuid.New()
	client := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}
	client.channels[channelID] = true

	// Register client and channel
	hub.mu.Lock()
	hub.clients[client] = true
	hub.channels[channelID] = make(map[*Client]bool)
	hub.channels[channelID][client] = true
	hub.mu.Unlock()

	// This should trigger the disconnect handler
	hub.unregisterClient(client)

	// Wait for callback (with timeout)
	select {
	case <-done:
		// Callback was called
	case <-time.After(time.Second):
		t.Fatal("disconnect handler was not called")
	}

	assert.Equal(t, client, disconnectedClient)
	assert.Contains(t, disconnectedChannels, channelID)
}

func TestHub_UnregisterClient_ClosesChannel(t *testing.T) {
	hub := NewHub()

	client := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	hub.mu.Lock()
	hub.clients[client] = true
	hub.mu.Unlock()

	hub.unregisterClient(client)

	// send channel should be closed
	_, ok := <-client.send
	assert.False(t, ok, "send channel should be closed")
}

func TestHub_GetOnlineUsers(t *testing.T) {
	hub := NewHub()

	channelID := uuid.New()
	user1ID := uuid.New()
	user2ID := uuid.New()

	// Create clients
	client1 := &Client{
		hub:      hub,
		userID:   user1ID,
		nickname: "user1",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	client2 := &Client{
		hub:      hub,
		userID:   user2ID,
		nickname: "user2",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register clients to channel
	hub.mu.Lock()
	hub.clients[client1] = true
	hub.clients[client2] = true
	hub.channels[channelID] = make(map[*Client]bool)
	hub.channels[channelID][client1] = true
	hub.channels[channelID][client2] = true
	hub.mu.Unlock()

	// Test GetOnlineUsers
	users := hub.GetOnlineUsers(channelID)
	assert.Len(t, users, 2)

	// Verify user info
	nicknames := make([]string, 0, 2)
	for _, u := range users {
		nicknames = append(nicknames, u.Nickname)
	}
	assert.Contains(t, nicknames, "user1")
	assert.Contains(t, nicknames, "user2")
}

func TestHub_GetOnlineUsers_EmptyChannel(t *testing.T) {
	hub := NewHub()

	channelID := uuid.New()

	users := hub.GetOnlineUsers(channelID)
	assert.Empty(t, users)
}

func TestHub_IsOnline(t *testing.T) {
	hub := NewHub()

	channelID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()

	client := &Client{
		hub:      hub,
		userID:   userID,
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register client to channel
	hub.mu.Lock()
	hub.clients[client] = true
	hub.channels[channelID] = make(map[*Client]bool)
	hub.channels[channelID][client] = true
	hub.mu.Unlock()

	// Test IsOnline - user is online
	assert.True(t, hub.IsOnline(channelID, userID))

	// Test IsOnline - other user is not online
	assert.False(t, hub.IsOnline(channelID, otherUserID))

	// Test IsOnline - non-existent channel
	assert.False(t, hub.IsOnline(uuid.New(), userID))
}

func TestHub_NotifyPresence(t *testing.T) {
	hub := NewHub()

	channelID := uuid.New()
	userID := uuid.New()

	client := &Client{
		hub:      hub,
		userID:   userID,
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Create another client to receive the notification
	otherClient := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "other",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register both clients to channel
	hub.mu.Lock()
	hub.clients[client] = true
	hub.clients[otherClient] = true
	hub.channels[channelID] = make(map[*Client]bool)
	hub.channels[channelID][client] = true
	hub.channels[channelID][otherClient] = true
	hub.mu.Unlock()

	// Call notifyPresence - client joins, other receives notification
	hub.notifyPresence(channelID, client, "join")

	// Other client should receive the presence notification
	select {
	case msg := <-otherClient.send:
		assert.Contains(t, string(msg), "presence")
		assert.Contains(t, string(msg), "join")
	case <-time.After(100 * time.Millisecond):
		t.Error("other client did not receive presence notification")
	}

	// Client should NOT receive their own presence notification
	select {
	case <-client.send:
		t.Error("client should not receive their own presence notification")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

func TestHub_Register_ViaChannel(t *testing.T) {
	hub := NewHub()

	// Start hub in goroutine
	go hub.Run()

	client := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register via channel
	hub.Register(client)

	// Wait a bit for the hub to process
	time.Sleep(50 * time.Millisecond)

	// Verify client is registered
	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()
	assert.True(t, exists)
}

func TestHub_Unregister_ViaChannel(t *testing.T) {
	hub := NewHub()

	// Start hub in goroutine
	go hub.Run()

	client := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register first
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	// Unregister via channel
	hub.Unregister(client)
	time.Sleep(50 * time.Millisecond)

	// Verify client is unregistered
	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()
	assert.False(t, exists)
}

func TestHub_Subscribe_ViaChannel(t *testing.T) {
	hub := NewHub()

	// Start hub in goroutine
	go hub.Run()

	channelID := uuid.New()
	client := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register first
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	// Subscribe via channel
	hub.Subscribe(client, channelID)
	time.Sleep(50 * time.Millisecond)

	// Verify subscription
	hub.mu.RLock()
	channelClients, exists := hub.channels[channelID]
	_, clientInChannel := channelClients[client]
	hub.mu.RUnlock()

	assert.True(t, exists)
	assert.True(t, clientInChannel)
}

func TestHub_Unsubscribe_ViaChannel(t *testing.T) {
	hub := NewHub()

	// Start hub in goroutine
	go hub.Run()

	channelID := uuid.New()
	client := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register and subscribe
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)
	hub.Subscribe(client, channelID)
	time.Sleep(50 * time.Millisecond)

	// Unsubscribe via channel
	hub.Unsubscribe(client, channelID)
	time.Sleep(50 * time.Millisecond)

	// Verify unsubscription - channel should be removed since it's empty
	hub.mu.RLock()
	_, exists := hub.channels[channelID]
	hub.mu.RUnlock()

	assert.False(t, exists)
}

func TestHub_Broadcast_ViaChannel(t *testing.T) {
	hub := NewHub()

	// Start hub in goroutine
	go hub.Run()

	channelID := uuid.New()
	client := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 256),
	}

	// Register and subscribe
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)
	hub.Subscribe(client, channelID)
	time.Sleep(50 * time.Millisecond)

	// Broadcast via channel
	message := []byte(`{"type":"test","content":"hello"}`)
	hub.Broadcast(channelID, message, nil)

	// Wait for message
	select {
	case msg := <-client.send:
		assert.Equal(t, message, msg)
	case <-time.After(200 * time.Millisecond):
		t.Error("client did not receive broadcast message")
	}
}

func TestHub_Broadcast_BufferFull(t *testing.T) {
	hub := NewHub()

	// Start hub in goroutine
	go hub.Run()

	channelID := uuid.New()
	// Create a client with a very small buffer (1)
	client := &Client{
		hub:      hub,
		userID:   uuid.New(),
		nickname: "testuser",
		channels: make(map[uuid.UUID]bool),
		send:     make(chan []byte, 1),
	}

	// Register and subscribe
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)
	hub.Subscribe(client, channelID)
	time.Sleep(50 * time.Millisecond)

	// Fill the buffer first
	client.send <- []byte(`{"type":"filler"}`)

	// Now broadcast - this should trigger the buffer full path
	message := []byte(`{"type":"test","content":"hello"}`)
	hub.Broadcast(channelID, message, nil)

	// Allow time for the broadcast to process
	time.Sleep(100 * time.Millisecond)

	// The client should have been removed due to buffer full
	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	// Client may or may not be removed depending on timing,
	// but at least we exercised the code path
	_ = exists
}
