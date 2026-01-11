package chat

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/middleware"
)

// Handler handles HTTP requests for chat operations.
type Handler struct {
	chatService     Service
	presenceService *PresenceService
}

// NewHandler creates a new chat handler.
func NewHandler(chatService Service, presenceService *PresenceService) *Handler {
	return &Handler{
		chatService:     chatService,
		presenceService: presenceService,
	}
}

// RegisterRoutes registers chat routes.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	channels := rg.Group("/channels")
	channels.Use(authMiddleware)
	{
		channels.GET("", h.ListChannels)
		channels.POST("", h.CreateChannel)
		channels.GET("/my", h.ListMyChannels)
		channels.GET("/:id", h.GetChannel)
		channels.PUT("/:id", h.UpdateChannel)
		channels.DELETE("/:id", h.DeleteChannel)
		channels.POST("/:id/join", h.JoinChannel)
		channels.POST("/:id/leave", h.LeaveChannel)
		channels.GET("/:id/messages", h.GetMessages)
		channels.GET("/:id/members/online", h.GetOnlineMembers)
	}
}

// ListChannels handles GET /api/v1/channels
func (h *Handler) ListChannels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	channels, total, err := h.chatService.ListChannels(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list channels",
			"code":  "CHAT001",
		})
		return
	}

	// Get member counts for each channel
	responses := make([]ChannelResponse, 0, len(channels))
	for i := range channels {
		count, _ := h.chatService.CountMembers(c.Request.Context(), channels[i].ID)
		responses = append(responses, ToChannelResponse(&channels[i], count))
	}

	c.JSON(http.StatusOK, ChannelListResponse{
		Channels: responses,
		Total:    total,
		Page:     page,
		Limit:    limit,
	})
}

// ListMyChannels handles GET /api/v1/channels/my
func (h *Handler) ListMyChannels(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "AUTH006",
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
			"code":  "AUTH006",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	channels, total, err := h.chatService.ListUserChannels(c.Request.Context(), userUUID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list channels",
			"code":  "CHAT001",
		})
		return
	}

	// Get member counts for each channel
	responses := make([]ChannelResponse, 0, len(channels))
	for i := range channels {
		count, _ := h.chatService.CountMembers(c.Request.Context(), channels[i].ID)
		responses = append(responses, ToChannelResponse(&channels[i], count))
	}

	c.JSON(http.StatusOK, ChannelListResponse{
		Channels: responses,
		Total:    total,
		Page:     page,
		Limit:    limit,
	})
}

// CreateChannel handles POST /api/v1/channels
func (h *Handler) CreateChannel(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "AUTH006",
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
			"code":  "AUTH006",
		})
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "CHAT001",
		})
		return
	}

	channel, err := h.chatService.CreateChannel(c.Request.Context(), userUUID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create channel",
			"code":  "CHAT001",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"channel": ToChannelResponse(channel, 1), // Owner is the first member
	})
}

// GetChannel handles GET /api/v1/channels/:id
func (h *Handler) GetChannel(c *gin.Context) {
	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid channel id",
			"code":  "CHAT001",
		})
		return
	}

	channel, err := h.chatService.GetChannelDetail(c.Request.Context(), channelID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "channel not found",
				"code":  "CHAT001",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get channel",
			"code":  "CHAT001",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channel": ToChannelDetailResponse(channel),
	})
}

// UpdateChannel handles PUT /api/v1/channels/:id
func (h *Handler) UpdateChannel(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "AUTH006",
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
			"code":  "AUTH006",
		})
		return
	}

	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid channel id",
			"code":  "CHAT001",
		})
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "CHAT001",
		})
		return
	}

	channel, err := h.chatService.UpdateChannel(c.Request.Context(), channelID, userUUID, &req)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "channel not found",
				"code":  "CHAT001",
			})
			return
		}
		if errors.Is(err, ErrNotChannelOwner) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "not channel owner",
				"code":  "CHAT002",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update channel",
			"code":  "CHAT001",
		})
		return
	}

	count, _ := h.chatService.CountMembers(c.Request.Context(), channel.ID)
	c.JSON(http.StatusOK, gin.H{
		"channel": ToChannelResponse(channel, count),
	})
}

// DeleteChannel handles DELETE /api/v1/channels/:id
func (h *Handler) DeleteChannel(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "AUTH006",
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
			"code":  "AUTH006",
		})
		return
	}

	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid channel id",
			"code":  "CHAT001",
		})
		return
	}

	err = h.chatService.DeleteChannel(c.Request.Context(), channelID, userUUID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "channel not found",
				"code":  "CHAT001",
			})
			return
		}
		if errors.Is(err, ErrNotChannelOwner) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "not channel owner",
				"code":  "CHAT002",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete channel",
			"code":  "CHAT001",
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "channel deleted successfully",
	})
}

// JoinChannel handles POST /api/v1/channels/:id/join
func (h *Handler) JoinChannel(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "AUTH006",
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
			"code":  "AUTH006",
		})
		return
	}

	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid channel id",
			"code":  "CHAT001",
		})
		return
	}

	err = h.chatService.JoinChannel(c.Request.Context(), channelID, userUUID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "channel not found",
				"code":  "CHAT001",
			})
			return
		}
		if errors.Is(err, ErrAlreadyMember) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "already a member of this channel",
				"code":  "CHAT002",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to join channel",
			"code":  "CHAT001",
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "채널에 참여했습니다",
	})
}

// LeaveChannel handles POST /api/v1/channels/:id/leave
func (h *Handler) LeaveChannel(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"code":  "AUTH006",
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
			"code":  "AUTH006",
		})
		return
	}

	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid channel id",
			"code":  "CHAT001",
		})
		return
	}

	err = h.chatService.LeaveChannel(c.Request.Context(), channelID, userUUID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "channel not found",
				"code":  "CHAT001",
			})
			return
		}
		if errors.Is(err, ErrNotMember) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "not a member of this channel",
				"code":  "CHAT002",
			})
			return
		}
		if errors.Is(err, ErrOwnerCannotLeave) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "channel owner cannot leave the channel",
				"code":  "CHAT002",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to leave channel",
			"code":  "CHAT001",
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "채널에서 퇴장했습니다",
	})
}

// GetMessages handles GET /api/v1/channels/:id/messages
func (h *Handler) GetMessages(c *gin.Context) {
	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid channel id",
			"code":  "CHAT001",
		})
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var before *time.Time
	if beforeStr := c.Query("before"); beforeStr != "" {
		t, err := time.Parse(time.RFC3339, beforeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid before parameter, use RFC3339 format",
				"code":  "CHAT001",
			})
			return
		}
		before = &t
	}

	messages, hasMore, err := h.chatService.GetMessages(c.Request.Context(), channelID, before, limit)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "channel not found",
				"code":  "CHAT001",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get messages",
			"code":  "CHAT001",
		})
		return
	}

	// Convert to response format
	responses := make([]ChatMessageResponse, 0, len(messages))
	for i := range messages {
		responses = append(responses, ToChatMessageResponse(&messages[i]))
	}

	c.JSON(http.StatusOK, MessageListResponse{
		Messages: responses,
		HasMore:  hasMore,
	})
}

// GetOnlineMembers handles GET /api/v1/channels/:id/members/online
func (h *Handler) GetOnlineMembers(c *gin.Context) {
	channelID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid channel id",
			"code":  "CHAT001",
		})
		return
	}

	// Check if channel exists
	_, err = h.chatService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "channel not found",
				"code":  "CHAT001",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get channel",
			"code":  "CHAT001",
		})
		return
	}

	// Get online users from Redis
	if h.presenceService == nil {
		c.JSON(http.StatusOK, gin.H{
			"users": []WSUserInfo{},
			"count": 0,
		})
		return
	}

	users, err := h.presenceService.GetOnlineUsers(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get online users",
			"code":  "CHAT001",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}
