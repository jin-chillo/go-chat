package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/middleware"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// MockService is a mock implementation of Service interface
type MockService struct {
	mock.Mock
}

func (m *MockService) CreateChannel(ctx context.Context, ownerID uuid.UUID, req *CreateChannelRequest) (*model.Channel, error) {
	args := m.Called(ctx, ownerID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Channel), args.Error(1)
}

func (m *MockService) GetChannel(ctx context.Context, id uuid.UUID) (*model.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Channel), args.Error(1)
}

func (m *MockService) GetChannelDetail(ctx context.Context, id uuid.UUID) (*model.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Channel), args.Error(1)
}

func (m *MockService) ListChannels(ctx context.Context, page, limit int) ([]model.Channel, int64, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]model.Channel), args.Get(1).(int64), args.Error(2)
}

func (m *MockService) ListUserChannels(ctx context.Context, userID uuid.UUID, page, limit int) ([]model.Channel, int64, error) {
	args := m.Called(ctx, userID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]model.Channel), args.Get(1).(int64), args.Error(2)
}

func (m *MockService) UpdateChannel(ctx context.Context, id, userID uuid.UUID, req *UpdateChannelRequest) (*model.Channel, error) {
	args := m.Called(ctx, id, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Channel), args.Error(1)
}

func (m *MockService) DeleteChannel(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockService) JoinChannel(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockService) LeaveChannel(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockService) GetChannelMembers(ctx context.Context, channelID uuid.UUID) ([]model.ChannelMember, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ChannelMember), args.Error(1)
}

func (m *MockService) CountMembers(ctx context.Context, channelID uuid.UUID) (int64, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockService) IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, channelID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockService) CreateMessage(ctx context.Context, channelID, userID uuid.UUID, nickname, content string) (*model.Message, error) {
	args := m.Called(ctx, channelID, userID, nickname, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Message), args.Error(1)
}

func (m *MockService) GetMessages(ctx context.Context, channelID uuid.UUID, before *time.Time, limit int) ([]model.Message, bool, error) {
	args := m.Called(ctx, channelID, before, limit)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]model.Message), args.Bool(1), args.Error(2)
}

func setupHandlerTestRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api/v1")

	// Mock auth middleware that sets user claims
	authMiddleware := func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID != "" {
			c.Set(middleware.ContextKeyUserID, userID)
			c.Set(middleware.ContextKeyEmail, "test@example.com")
			c.Set(middleware.ContextKeyNickname, "testuser")
		}
		c.Next()
	}

	handler.RegisterRoutes(rg, authMiddleware)
	return router
}

func TestNewHandler(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService, nil)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.chatService)
	assert.Nil(t, handler.presenceService)
}

func TestHandler_ListChannels(t *testing.T) {
	channelID := uuid.New()
	ownerID := uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*MockService)
		expectedStatus int
	}{
		{
			name:        "successful list channels",
			queryParams: "?page=1&limit=10",
			setupMock: func(m *MockService) {
				channels := []model.Channel{
					{ID: channelID, Name: "Test Channel", OwnerID: ownerID},
				}
				m.On("ListChannels", mock.Anything, 1, 10).Return(channels, int64(1), nil)
				m.On("CountMembers", mock.Anything, channelID).Return(int64(5), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "default pagination",
			queryParams: "",
			setupMock: func(m *MockService) {
				m.On("ListChannels", mock.Anything, 1, 20).Return([]model.Channel{}, int64(0), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "service error",
			queryParams: "",
			setupMock: func(m *MockService) {
				m.On("ListChannels", mock.Anything, 1, 20).Return(nil, int64(0), ErrChannelNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil)
			router := setupHandlerTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/channels"+tt.queryParams, nil)
			req.Header.Set("X-User-ID", uuid.New().String())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_ListMyChannels(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		setupMock      func(*MockService)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:   "successful list my channels",
			userID: userID.String(),
			setupMock: func(m *MockService) {
				channels := []model.Channel{
					{ID: channelID, Name: "My Channel", OwnerID: userID},
				}
				m.On("ListUserChannels", mock.Anything, userID, 1, 20).Return(channels, int64(1), nil)
				m.On("CountMembers", mock.Anything, channelID).Return(int64(3), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing user ID",
			userID:         "",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH006",
		},
		{
			name:           "invalid user ID",
			userID:         "invalid-uuid",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "AUTH006",
		},
		{
			name:   "service error",
			userID: userID.String(),
			setupMock: func(m *MockService) {
				m.On("ListUserChannels", mock.Anything, userID, 1, 20).Return(nil, int64(0), ErrChannelNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil)
			router := setupHandlerTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/my", nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp["code"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_CreateChannel(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		body           interface{}
		setupMock      func(*MockService)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:   "successful create channel",
			userID: userID.String(),
			body: CreateChannelRequest{
				Name:        "New Channel",
				Description: "A new channel",
			},
			setupMock: func(m *MockService) {
				channel := &model.Channel{
					ID:      channelID,
					Name:    "New Channel",
					OwnerID: userID,
				}
				m.On("CreateChannel", mock.Anything, userID, mock.AnythingOfType("*chat.CreateChannelRequest")).Return(channel, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing user ID",
			userID:         "",
			body:           CreateChannelRequest{Name: "Test"},
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH006",
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			body:           CreateChannelRequest{Name: "Test"},
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "AUTH006",
		},
		{
			name:           "invalid request body",
			userID:         userID.String(),
			body:           "invalid",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "CHAT001",
		},
		{
			name:   "service error",
			userID: userID.String(),
			body:   CreateChannelRequest{Name: "Test"},
			setupMock: func(m *MockService) {
				m.On("CreateChannel", mock.Anything, userID, mock.Anything).Return(nil, ErrChannelNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil)
			router := setupHandlerTestRouter(handler)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/channels", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp["code"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_GetChannel(t *testing.T) {
	channelID := uuid.New()
	ownerID := uuid.New()

	tests := []struct {
		name           string
		channelID      string
		setupMock      func(*MockService)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "successful get channel",
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				channel := &model.Channel{
					ID:      channelID,
					Name:    "Test Channel",
					OwnerID: ownerID,
				}
				m.On("GetChannelDetail", mock.Anything, channelID).Return(channel, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid channel ID",
			channelID:      "invalid",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "CHAT001",
		},
		{
			name:      "channel not found",
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("GetChannelDetail", mock.Anything, channelID).Return(nil, ErrChannelNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "CHAT001",
		},
		{
			name:      "service error",
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("GetChannelDetail", mock.Anything, channelID).Return(nil, ErrAlreadyMember)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil)
			router := setupHandlerTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+tt.channelID, nil)
			req.Header.Set("X-User-ID", uuid.New().String())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp["code"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_UpdateChannel(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		channelID      string
		body           interface{}
		setupMock      func(*MockService)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "successful update",
			userID:    userID.String(),
			channelID: channelID.String(),
			body:      UpdateChannelRequest{Name: "Updated Name"},
			setupMock: func(m *MockService) {
				channel := &model.Channel{ID: channelID, Name: "Updated Name", OwnerID: userID}
				m.On("UpdateChannel", mock.Anything, channelID, userID, mock.Anything).Return(channel, nil)
				m.On("CountMembers", mock.Anything, channelID).Return(int64(5), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing user ID",
			userID:         "",
			channelID:      channelID.String(),
			body:           UpdateChannelRequest{Name: "Test"},
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			channelID:      channelID.String(),
			body:           UpdateChannelRequest{Name: "Test"},
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			userID:         userID.String(),
			channelID:      "invalid",
			body:           UpdateChannelRequest{Name: "Test"},
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid body",
			userID:         userID.String(),
			channelID:      channelID.String(),
			body:           "invalid",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "channel not found",
			userID:    userID.String(),
			channelID: channelID.String(),
			body:      UpdateChannelRequest{Name: "Test"},
			setupMock: func(m *MockService) {
				m.On("UpdateChannel", mock.Anything, channelID, userID, mock.Anything).Return(nil, ErrChannelNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "not channel owner",
			userID:    userID.String(),
			channelID: channelID.String(),
			body:      UpdateChannelRequest{Name: "Test"},
			setupMock: func(m *MockService) {
				m.On("UpdateChannel", mock.Anything, channelID, userID, mock.Anything).Return(nil, ErrNotChannelOwner)
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "CHAT002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil)
			router := setupHandlerTestRouter(handler)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/channels/"+tt.channelID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp["code"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_DeleteChannel(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		channelID      string
		setupMock      func(*MockService)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "successful delete",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("DeleteChannel", mock.Anything, channelID, userID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing user ID",
			userID:         "",
			channelID:      channelID.String(),
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			channelID:      channelID.String(),
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			userID:         userID.String(),
			channelID:      "invalid",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "channel not found",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("DeleteChannel", mock.Anything, channelID, userID).Return(ErrChannelNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "not channel owner",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("DeleteChannel", mock.Anything, channelID, userID).Return(ErrNotChannelOwner)
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "CHAT002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil)
			router := setupHandlerTestRouter(handler)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/channels/"+tt.channelID, nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp["code"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_JoinChannel(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		channelID      string
		setupMock      func(*MockService)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "successful join",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("JoinChannel", mock.Anything, channelID, userID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing user ID",
			userID:         "",
			channelID:      channelID.String(),
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			channelID:      channelID.String(),
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			userID:         userID.String(),
			channelID:      "invalid",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "channel not found",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("JoinChannel", mock.Anything, channelID, userID).Return(ErrChannelNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "already member",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("JoinChannel", mock.Anything, channelID, userID).Return(ErrAlreadyMember)
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "CHAT002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil)
			router := setupHandlerTestRouter(handler)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/"+tt.channelID+"/join", nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp["code"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_LeaveChannel(t *testing.T) {
	userID := uuid.New()
	channelID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		channelID      string
		setupMock      func(*MockService)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "successful leave",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("LeaveChannel", mock.Anything, channelID, userID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing user ID",
			userID:         "",
			channelID:      channelID.String(),
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			channelID:      channelID.String(),
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid channel ID",
			userID:         userID.String(),
			channelID:      "invalid",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "channel not found",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("LeaveChannel", mock.Anything, channelID, userID).Return(ErrChannelNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "not member",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("LeaveChannel", mock.Anything, channelID, userID).Return(ErrNotMember)
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "CHAT002",
		},
		{
			name:      "owner cannot leave",
			userID:    userID.String(),
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("LeaveChannel", mock.Anything, channelID, userID).Return(ErrOwnerCannotLeave)
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "CHAT002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil)
			router := setupHandlerTestRouter(handler)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/"+tt.channelID+"/leave", nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp["code"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_GetMessages(t *testing.T) {
	channelID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name           string
		channelID      string
		queryParams    string
		setupMock      func(*MockService)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:        "successful get messages",
			channelID:   channelID.String(),
			queryParams: "",
			setupMock: func(m *MockService) {
				messages := []model.Message{
					{ChannelID: channelID, UserID: userID, Content: "Hello"},
				}
				m.On("GetMessages", mock.Anything, channelID, (*time.Time)(nil), 50).Return(messages, false, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "with limit",
			channelID:   channelID.String(),
			queryParams: "?limit=10",
			setupMock: func(m *MockService) {
				m.On("GetMessages", mock.Anything, channelID, (*time.Time)(nil), 10).Return([]model.Message{}, false, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "invalid limit defaults to 50",
			channelID:   channelID.String(),
			queryParams: "?limit=200",
			setupMock: func(m *MockService) {
				m.On("GetMessages", mock.Anything, channelID, (*time.Time)(nil), 50).Return([]model.Message{}, false, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid channel ID",
			channelID:      "invalid",
			queryParams:    "",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "CHAT001",
		},
		{
			name:           "invalid before parameter",
			channelID:      channelID.String(),
			queryParams:    "?before=invalid-time",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "CHAT001",
		},
		{
			name:      "channel not found",
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("GetMessages", mock.Anything, channelID, (*time.Time)(nil), 50).Return(nil, false, ErrChannelNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "CHAT001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil)
			router := setupHandlerTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+tt.channelID+"/messages"+tt.queryParams, nil)
			req.Header.Set("X-User-ID", uuid.New().String())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp["code"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_GetOnlineMembers(t *testing.T) {
	channelID := uuid.New()
	ownerID := uuid.New()

	tests := []struct {
		name           string
		channelID      string
		setupMock      func(*MockService)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:      "no presence service returns empty",
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				channel := &model.Channel{ID: channelID, OwnerID: ownerID}
				m.On("GetChannel", mock.Anything, channelID).Return(channel, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid channel ID",
			channelID:      "invalid",
			setupMock:      func(m *MockService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "CHAT001",
		},
		{
			name:      "channel not found",
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("GetChannel", mock.Anything, channelID).Return(nil, ErrChannelNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "CHAT001",
		},
		{
			name:      "get channel error",
			channelID: channelID.String(),
			setupMock: func(m *MockService) {
				m.On("GetChannel", mock.Anything, channelID).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "CHAT001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			tt.setupMock(mockService)

			handler := NewHandler(mockService, nil) // nil presence service
			router := setupHandlerTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+tt.channelID+"/members/online", nil)
			req.Header.Set("X-User-ID", uuid.New().String())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp["code"])
			}

			mockService.AssertExpectations(t)
		})
	}
}
