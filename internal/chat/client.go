package chat

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// User information
	userID   uuid.UUID
	nickname string

	// Subscribed channels
	channels map[uuid.UUID]bool

	// Message handler for processing incoming messages
	messageHandler MessageHandler
}

// MessageHandler is called when a client sends a message.
type MessageHandler func(client *Client, msg *WSMessage) error

// NewClient creates a new Client instance.
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, nickname string, handler MessageHandler) *Client {
	return &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan []byte, 256),
		userID:         userID,
		nickname:       nickname,
		channels:       make(map[uuid.UUID]bool),
		messageHandler: handler,
	}
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse the message
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			c.sendError("invalid message format")
			continue
		}

		// Handle the message
		if c.messageHandler != nil {
			if err := c.messageHandler(c, &wsMsg); err != nil {
				log.Printf("Message handler error: %v", err)
				c.sendError(err.Error())
			}
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Failed to set write deadline: %v", err)
				return
			}
			if !ok {
				// The hub closed the channel.
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Failed to write close message: %v", err)
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				log.Printf("Failed to write message: %v", err)
				return
			}

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				if _, err := w.Write([]byte{'\n'}); err != nil {
					log.Printf("Failed to write newline: %v", err)
					return
				}
				if _, err := w.Write(<-c.send); err != nil {
					log.Printf("Failed to write queued message: %v", err)
					return
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Failed to set write deadline: %v", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendError sends an error message to the client.
func (c *Client) sendError(errMsg string) {
	msg := WSMessage{
		Type:      "error",
		Error:     errMsg,
		Timestamp: time.Now(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		// Buffer is full, skip
	}
}

// Send sends a message to the client.
func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
	default:
		// Buffer is full
	}
}

// UserID returns the client's user ID.
func (c *Client) UserID() uuid.UUID {
	return c.userID
}

// Nickname returns the client's nickname.
func (c *Client) Nickname() string {
	return c.nickname
}

// SubscribedChannels returns the channels the client is subscribed to.
func (c *Client) SubscribedChannels() map[uuid.UUID]bool {
	return c.channels
}

// AddChannel marks a channel as subscribed.
func (c *Client) AddChannel(channelID uuid.UUID) {
	c.channels[channelID] = true
}

// RemoveChannel removes a channel subscription.
func (c *Client) RemoveChannel(channelID uuid.UUID) {
	delete(c.channels, channelID)
}

// IsSubscribed checks if the client is subscribed to a channel.
func (c *Client) IsSubscribed(channelID uuid.UUID) bool {
	return c.channels[channelID]
}
