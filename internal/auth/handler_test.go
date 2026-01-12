package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg, nil)
	return router
}

func TestHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockUserRepository, *MockTokenRepository)
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "successful registration",
			body: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Nickname: "testuser",
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, ErrUserNotFound)
				userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "email already exists",
			body: RegisterRequest{
				Email:    "existing@example.com",
				Password: "password123",
				Nickname: "testuser",
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				existingUser := &model.User{ID: uuid.New(), Email: "existing@example.com"}
				userRepo.On("FindByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "AUTH003",
		},
		{
			name: "validation error - missing email",
			body: map[string]string{
				"password": "password123",
				"nickname": "testuser",
			},
			setupMock:      func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "AUTH001",
		},
		{
			name: "validation error - short password",
			body: RegisterRequest{
				Email:    "test@example.com",
				Password: "short",
				Nickname: "testuser",
			},
			setupMock:      func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "AUTH001",
		},
		{
			name: "internal server error - create fails",
			body: RegisterRequest{
				Email:    "newerror@example.com",
				Password: "password123",
				Nickname: "erroruser",
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", mock.Anything, "newerror@example.com").Return(nil, ErrUserNotFound)
				userRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockTokenRepo := new(MockTokenRepository)
			jwtSvc := newTestJWTService()

			tt.setupMock(mockUserRepo, mockTokenRepo)

			authService := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
			handler := NewHandler(authService, jwtSvc)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp.Code)
			}

			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestHandler_Login(t *testing.T) {
	password := "correctPassword123"
	passwordHash, _ := HashPassword(password)
	testUser := &model.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: passwordHash,
		Nickname:     "testuser",
	}

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockUserRepository, *MockTokenRepository)
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "successful login",
			body: LoginRequest{
				Email:    "user@example.com",
				Password: password,
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", mock.Anything, "user@example.com").Return(testUser, nil)
				tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user not found",
			body: LoginRequest{
				Email:    "notfound@example.com",
				Password: "anypassword",
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", mock.Anything, "notfound@example.com").Return(nil, ErrUserNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH004",
		},
		{
			name: "wrong password",
			body: LoginRequest{
				Email:    "user@example.com",
				Password: "wrongPassword",
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", mock.Anything, "user@example.com").Return(testUser, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH004",
		},
		{
			name: "validation error",
			body: map[string]string{
				"email": "invalid-email",
			},
			setupMock:      func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "AUTH001",
		},
		{
			name: "internal server error - token create fails",
			body: LoginRequest{
				Email:    "user@example.com",
				Password: password,
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", mock.Anything, "user@example.com").Return(testUser, nil)
				tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockTokenRepo := new(MockTokenRepository)
			jwtSvc := newTestJWTService()

			tt.setupMock(mockUserRepo, mockTokenRepo)

			authService := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
			handler := NewHandler(authService, jwtSvc)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp.Code)
			}

			mockUserRepo.AssertExpectations(t)
			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestHandler_Refresh(t *testing.T) {
	testUser := &model.User{
		ID:       uuid.New(),
		Email:    "user@example.com",
		Nickname: "testuser",
	}

	refreshToken := "valid-refresh-token"
	tokenHash := HashToken(refreshToken)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockUserRepository, *MockTokenRepository)
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "successful refresh",
			body: RefreshRequest{
				RefreshToken: refreshToken,
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				tokenRecord := &model.RefreshToken{
					ID:        uuid.New(),
					UserID:    testUser.ID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				tokenRepo.On("FindByTokenHash", mock.Anything, tokenHash).Return(tokenRecord, nil)
				userRepo.On("FindByID", mock.Anything, testUser.ID).Return(testUser, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid refresh token",
			body: RefreshRequest{
				RefreshToken: "invalid-token",
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				tokenRepo.On("FindByTokenHash", mock.Anything, mock.Anything).Return(nil, ErrRefreshTokenNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH006",
		},
		{
			name: "expired refresh token",
			body: RefreshRequest{
				RefreshToken: refreshToken,
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				expiredToken := &model.RefreshToken{
					ID:        uuid.New(),
					UserID:    testUser.ID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(-1 * time.Hour),
				}
				tokenRepo.On("FindByTokenHash", mock.Anything, tokenHash).Return(expiredToken, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH005",
		},
		{
			name: "validation error - missing refresh token",
			body: map[string]string{},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "AUTH001",
		},
		{
			name: "revoked refresh token",
			body: RefreshRequest{
				RefreshToken: refreshToken,
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				revokedToken := &model.RefreshToken{
					ID:        uuid.New(),
					UserID:    testUser.ID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(24 * time.Hour),
					RevokedAt: func() *time.Time { t := time.Now(); return &t }(),
				}
				tokenRepo.On("FindByTokenHash", mock.Anything, tokenHash).Return(revokedToken, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH006",
		},
		{
			name: "user not found during refresh",
			body: RefreshRequest{
				RefreshToken: refreshToken,
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				tokenRecord := &model.RefreshToken{
					ID:        uuid.New(),
					UserID:    testUser.ID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				tokenRepo.On("FindByTokenHash", mock.Anything, tokenHash).Return(tokenRecord, nil)
				userRepo.On("FindByID", mock.Anything, testUser.ID).Return(nil, ErrUserNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockTokenRepo := new(MockTokenRepository)
			jwtSvc := newTestJWTService()

			tt.setupMock(mockUserRepo, mockTokenRepo)

			authService := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
			handler := NewHandler(authService, jwtSvc)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp.Code)
			}

			mockUserRepo.AssertExpectations(t)
			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestHandler_Logout(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH006",
		},
		{
			name:           "invalid header format - no bearer",
			authHeader:     "token123",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH006",
		},
		{
			name:           "invalid header format - wrong prefix",
			authHeader:     "Basic token123",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH006",
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH006",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockTokenRepo := new(MockTokenRepository)
			jwtSvc := newTestJWTService()

			authService := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
			handler := NewHandler(authService, jwtSvc)
			router := setupTestRouter(handler)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				var resp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, resp.Code)
			}
		})
	}
}

func TestNewHandler(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSvc := newTestJWTService()

	authService := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
	handler := NewHandler(authService, jwtSvc)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.authService)
	assert.NotNil(t, handler.jwtService)
}

func TestHandler_RegisterRoutes_WithRateLimit(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSvc := newTestJWTService()

	authService := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
	handler := NewHandler(authService, jwtSvc)

	router := gin.New()
	rg := router.Group("/api/v1")

	rateLimitCalled := false
	mockRateLimit := func(c *gin.Context) {
		rateLimitCalled = true
		c.Next()
	}

	handler.RegisterRoutes(rg, &RouteConfig{
		RegisterRateLimit: mockRateLimit,
		LoginRateLimit:    mockRateLimit,
	})

	// Test that rate limit middleware is registered
	mockUserRepo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, ErrUserNotFound)
	mockUserRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	body := `{"email":"test@example.com","password":"password123","nickname":"testuser"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.True(t, rateLimitCalled)
}
