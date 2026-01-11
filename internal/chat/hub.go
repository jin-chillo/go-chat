package chat

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ClientDisconnectHandler is called when a client disconnects.
type ClientDisconnectHandler func(client *Client, channelIDs []uuid.UUID)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients by channel
	channels map[uuid.UUID]map[*Client]bool

	// All registered clients
	clients map[*Client]bool

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Inbound messages from the clients
	broadcast chan *BroadcastMessage

	// Channel subscription requests
	subscribe chan *SubscriptionRequest

	// Channel unsubscription requests
	unsubscribe chan *SubscriptionRequest

	// Mutex for thread-safe access
	mu sync.RWMutex

	// Disconnect handler
	onDisconnect ClientDisconnectHandler
}

// BroadcastMessage represents a message to be broadcast to a channel.
type BroadcastMessage struct {
	ChannelID uuid.UUID
	Message   []byte
	Sender    *Client // Optional: exclude sender from broadcast
}

// SubscriptionRequest represents a request to subscribe/unsubscribe to a channel.
type SubscriptionRequest struct {
	Client    *Client
	ChannelID uuid.UUID
}

// WSMessage represents a WebSocket message structure.
type WSMessage struct {
	Type      string          `json:"type"`
	ChannelID uuid.UUID       `json:"channel_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	Message   *WSChatMessage  `json:"message,omitempty"`
	User      *WSUserInfo     `json:"user,omitempty"`
	Status    string          `json:"status,omitempty"`
	Error     string          `json:"error,omitempty"`
	Timestamp time.Time       `json:"timestamp,omitempty"`
}

// WSChatMessage represents a chat message in WebSocket format.
type WSChatMessage struct {
	ID        string    `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Nickname  string    `json:"nickname"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// WSUserInfo represents user information in WebSocket messages.
type WSUserInfo struct {
	ID       uuid.UUID `json:"id"`
	Nickname string    `json:"nickname"`
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		channels:    make(map[uuid.UUID]map[*Client]bool),
		clients:     make(map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *BroadcastMessage, 256),
		subscribe:   make(chan *SubscriptionRequest),
		unsubscribe: make(chan *SubscriptionRequest),
	}
}

// SetOnDisconnect sets the disconnect handler.
func (h *Hub) SetOnDisconnect(handler ClientDisconnectHandler) {
	h.onDisconnect = handler
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case req := <-h.subscribe:
			h.subscribeToChannel(req)

		case req := <-h.unsubscribe:
			h.unsubscribeFromChannel(req)

		case msg := <-h.broadcast:
			h.broadcastToChannel(msg)
		}
	}
}

// registerClient adds a client to the hub.
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
	log.Printf("Client registered: %s", client.userID)
}

// unregisterClient removes a client from the hub and all channels.
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Collect channel IDs for disconnect handler
		var channelIDs []uuid.UUID

		// Remove from all channels
		for channelID, clients := range h.channels {
			if _, ok := clients[client]; ok {
				delete(clients, client)
				channelIDs = append(channelIDs, channelID)
				// Notify others in the channel
				h.notifyPresence(channelID, client, "offline")
			}
		}

		// Call disconnect handler if set
		if h.onDisconnect != nil && len(channelIDs) > 0 {
			go h.onDisconnect(client, channelIDs)
		}

		log.Printf("Client unregistered: %s", client.userID)
	}
}

// subscribeToChannel adds a client to a channel.
func (h *Hub) subscribeToChannel(req *SubscriptionRequest) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[req.ChannelID]; !ok {
		h.channels[req.ChannelID] = make(map[*Client]bool)
	}

	h.channels[req.ChannelID][req.Client] = true
	log.Printf("Client %s subscribed to channel %s", req.Client.userID, req.ChannelID)

	// Notify others in the channel
	h.notifyPresence(req.ChannelID, req.Client, "online")
}

// unsubscribeFromChannel removes a client from a channel.
func (h *Hub) unsubscribeFromChannel(req *SubscriptionRequest) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.channels[req.ChannelID]; ok {
		if _, ok := clients[req.Client]; ok {
			delete(clients, req.Client)
			log.Printf("Client %s unsubscribed from channel %s", req.Client.userID, req.ChannelID)

			// Notify others in the channel
			h.notifyPresence(req.ChannelID, req.Client, "offline")

			// Clean up empty channel
			if len(clients) == 0 {
				delete(h.channels, req.ChannelID)
			}
		}
	}
}

// broadcastToChannel sends a message to all clients in a channel.
func (h *Hub) broadcastToChannel(msg *BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.channels[msg.ChannelID]; ok {
		for client := range clients {
			// Optionally skip the sender
			if msg.Sender != nil && client == msg.Sender {
				continue
			}

			select {
			case client.send <- msg.Message:
			default:
				// Client's send buffer is full, close connection
				close(client.send)
				delete(clients, client)
				delete(h.clients, client)
			}
		}
	}
}

// notifyPresence sends a presence notification to a channel (must be called with lock held).
func (h *Hub) notifyPresence(channelID uuid.UUID, client *Client, status string) {
	msg := WSMessage{
		Type:      "presence",
		ChannelID: channelID,
		User: &WSUserInfo{
			ID:       client.userID,
			Nickname: client.nickname,
		},
		Status:    status,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal presence message: %v", err)
		return
	}

	if clients, ok := h.channels[channelID]; ok {
		for c := range clients {
			if c != client {
				select {
				case c.send <- data:
				default:
					// Skip if buffer is full
				}
			}
		}
	}
}

// Register registers a client with the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Subscribe subscribes a client to a channel.
func (h *Hub) Subscribe(client *Client, channelID uuid.UUID) {
	h.subscribe <- &SubscriptionRequest{
		Client:    client,
		ChannelID: channelID,
	}
}

// Unsubscribe unsubscribes a client from a channel.
func (h *Hub) Unsubscribe(client *Client, channelID uuid.UUID) {
	h.unsubscribe <- &SubscriptionRequest{
		Client:    client,
		ChannelID: channelID,
	}
}

// Broadcast sends a message to all clients in a channel.
func (h *Hub) Broadcast(channelID uuid.UUID, message []byte, excludeSender *Client) {
	h.broadcast <- &BroadcastMessage{
		ChannelID: channelID,
		Message:   message,
		Sender:    excludeSender,
	}
}

// GetOnlineUsers returns the list of online users in a channel.
func (h *Hub) GetOnlineUsers(channelID uuid.UUID) []WSUserInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]WSUserInfo, 0)
	if clients, ok := h.channels[channelID]; ok {
		for client := range clients {
			users = append(users, WSUserInfo{
				ID:       client.userID,
				Nickname: client.nickname,
			})
		}
	}
	return users
}

// IsOnline checks if a user is online in a specific channel.
func (h *Hub) IsOnline(channelID, userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.channels[channelID]; ok {
		for client := range clients {
			if client.userID == userID {
				return true
			}
		}
	}
	return false
}
