package chat

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jin-chillo/go-chat/internal/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: In production, validate origin
		return true
	},
}

// WebSocketHandler handles WebSocket connections.
type WebSocketHandler struct {
	hub             *Hub
	chatService     Service
	jwtService      *auth.JWTService
	presenceService *PresenceService
}

// NewWebSocketHandler creates a new WebSocket handler.
func NewWebSocketHandler(hub *Hub, chatService Service, jwtService *auth.JWTService, presenceService *PresenceService) *WebSocketHandler {
	return &WebSocketHandler{
		hub:             hub,
		chatService:     chatService,
		jwtService:      jwtService,
		presenceService: presenceService,
	}
}

// RegisterRoutes registers WebSocket routes.
func (h *WebSocketHandler) RegisterRoutes(router *gin.Engine) {
	router.GET("/ws/chat", h.HandleWebSocket)
}

// HandleWebSocket handles WebSocket connection requests.
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "missing token",
			"code":  "AUTH006",
		})
		return
	}

	// Validate token
	claims, err := h.jwtService.ValidateAccessToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token",
			"code":  "AUTH006",
		})
		return
	}

	// Check if token is blacklisted
	isBlacklisted, err := h.jwtService.IsBlacklisted(c.Request.Context(), claims.ID)
	if err != nil || isBlacklisted {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "token is invalid",
			"code":  "AUTH006",
		})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		log.Printf("Invalid user ID in token: %v", err)
		conn.Close()
		return
	}

	// Create client
	client := NewClient(h.hub, conn, userID, claims.Nickname, h.handleMessage)

	// Register with hub
	h.hub.Register(client)

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()
}

// handleMessage processes incoming WebSocket messages.
func (h *WebSocketHandler) handleMessage(client *Client, msg *WSMessage) error {
	switch msg.Type {
	case "subscribe":
		return h.handleSubscribe(client, msg)
	case "unsubscribe":
		return h.handleUnsubscribe(client, msg)
	case "message":
		return h.handleChatMessage(client, msg)
	case "typing":
		return h.handleTyping(client, msg)
	default:
		return errors.New("unknown message type")
	}
}

// handleSubscribe handles channel subscription requests.
func (h *WebSocketHandler) handleSubscribe(client *Client, msg *WSMessage) error {
	if msg.ChannelID == uuid.Nil {
		return errors.New("channel_id is required")
	}

	// Check if user is a member of the channel
	ctx := context.Background()
	isMember, err := h.chatService.IsMember(ctx, msg.ChannelID, client.UserID())
	if err != nil {
		return errors.New("failed to check membership")
	}
	if !isMember {
		return errors.New("not a member of this channel")
	}

	// Subscribe to channel
	h.hub.Subscribe(client, msg.ChannelID)
	client.AddChannel(msg.ChannelID)

	// Set online status in Redis
	if h.presenceService != nil {
		if err := h.presenceService.SetOnline(ctx, msg.ChannelID, client.UserID(), client.Nickname()); err != nil {
			log.Printf("Failed to set online status: %v", err)
		}
	}

	// Send confirmation
	response := WSMessage{
		Type:      "subscribed",
		ChannelID: msg.ChannelID,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(response)
	client.Send(data)

	return nil
}

// handleUnsubscribe handles channel unsubscription requests.
func (h *WebSocketHandler) handleUnsubscribe(client *Client, msg *WSMessage) error {
	if msg.ChannelID == uuid.Nil {
		return errors.New("channel_id is required")
	}

	// Unsubscribe from channel
	h.hub.Unsubscribe(client, msg.ChannelID)
	client.RemoveChannel(msg.ChannelID)

	// Set offline status in Redis
	ctx := context.Background()
	if h.presenceService != nil {
		if err := h.presenceService.SetOffline(ctx, msg.ChannelID, client.UserID()); err != nil {
			log.Printf("Failed to set offline status: %v", err)
		}
	}

	// Send confirmation
	response := WSMessage{
		Type:      "unsubscribed",
		ChannelID: msg.ChannelID,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(response)
	client.Send(data)

	return nil
}

// handleChatMessage handles chat messages.
func (h *WebSocketHandler) handleChatMessage(client *Client, msg *WSMessage) error {
	if msg.ChannelID == uuid.Nil {
		return errors.New("channel_id is required")
	}
	if msg.Content == "" {
		return errors.New("content is required")
	}
	if len(msg.Content) > 2000 {
		return errors.New("message too long")
	}

	// Check if client is subscribed to the channel
	if !client.IsSubscribed(msg.ChannelID) {
		return errors.New("not subscribed to this channel")
	}

	// Save message to database
	savedMsg, err := h.chatService.CreateMessage(
		context.Background(),
		msg.ChannelID,
		client.UserID(),
		client.Nickname(),
		msg.Content,
	)
	if err != nil {
		log.Printf("Failed to save message: %v", err)
		return errors.New("failed to save message")
	}

	// Broadcast message to channel
	broadcastMsg := WSMessage{
		Type:      "message",
		ChannelID: msg.ChannelID,
		Message: &WSChatMessage{
			ID:        savedMsg.ID.Hex(),
			UserID:    client.UserID(),
			Nickname:  client.Nickname(),
			Content:   msg.Content,
			CreatedAt: savedMsg.CreatedAt,
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(broadcastMsg)
	if err != nil {
		return errors.New("failed to marshal message")
	}

	// Broadcast to all clients in the channel (including sender)
	h.hub.Broadcast(msg.ChannelID, data, nil)

	return nil
}

// handleTyping handles typing indicator messages.
func (h *WebSocketHandler) handleTyping(client *Client, msg *WSMessage) error {
	if msg.ChannelID == uuid.Nil {
		return errors.New("channel_id is required")
	}

	// Check if client is subscribed to the channel
	if !client.IsSubscribed(msg.ChannelID) {
		return errors.New("not subscribed to this channel")
	}

	// Broadcast typing indicator to channel (excluding sender)
	typingMsg := WSMessage{
		Type:      "typing",
		ChannelID: msg.ChannelID,
		User: &WSUserInfo{
			ID:       client.UserID(),
			Nickname: client.Nickname(),
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(typingMsg)
	if err != nil {
		return errors.New("failed to marshal typing message")
	}

	// Broadcast to all clients except sender
	h.hub.Broadcast(msg.ChannelID, data, client)

	return nil
}

// SetupDisconnectHandler configures the hub to clean up presence on disconnect.
func (h *WebSocketHandler) SetupDisconnectHandler() {
	h.hub.SetOnDisconnect(func(client *Client, channelIDs []uuid.UUID) {
		if h.presenceService == nil {
			return
		}

		ctx := context.Background()
		if err := h.presenceService.SetOfflineFromAllChannels(ctx, client.UserID(), channelIDs); err != nil {
			log.Printf("Failed to clean up presence for user %s: %v", client.UserID(), err)
		}
	})
}

// GetPresenceService returns the presence service for use by handlers.
func (h *WebSocketHandler) GetPresenceService() *PresenceService {
	return h.presenceService
}
