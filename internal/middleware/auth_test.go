package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jin-chillo/go-chat/internal/auth"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*gin.Context)
		wantValue string
		wantOK    bool
	}{
		{
			name: "valid user ID",
			setup: func(c *gin.Context) {
				c.Set(ContextKeyUserID, "user-123")
			},
			wantValue: "user-123",
			wantOK:    true,
		},
		{
			name:      "missing user ID",
			setup:     func(c *gin.Context) {},
			wantValue: "",
			wantOK:    false,
		},
		{
			name: "wrong type",
			setup: func(c *gin.Context) {
				c.Set(ContextKeyUserID, 123)
			},
			wantValue: "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)

			value, ok := GetUserID(c)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestGetEmail(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*gin.Context)
		wantValue string
		wantOK    bool
	}{
		{
			name: "valid email",
			setup: func(c *gin.Context) {
				c.Set(ContextKeyEmail, "test@example.com")
			},
			wantValue: "test@example.com",
			wantOK:    true,
		},
		{
			name:      "missing email",
			setup:     func(c *gin.Context) {},
			wantValue: "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)

			value, ok := GetEmail(c)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestGetNickname(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*gin.Context)
		wantValue string
		wantOK    bool
	}{
		{
			name: "valid nickname",
			setup: func(c *gin.Context) {
				c.Set(ContextKeyNickname, "testuser")
			},
			wantValue: "testuser",
			wantOK:    true,
		},
		{
			name:      "missing nickname",
			setup:     func(c *gin.Context) {},
			wantValue: "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)

			value, ok := GetNickname(c)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestGetClaims(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*gin.Context)
		wantOK bool
	}{
		{
			name: "valid claims",
			setup: func(c *gin.Context) {
				claims := &auth.TokenClaims{
					UserID:   "user-123",
					Email:    "test@example.com",
					Nickname: "testuser",
				}
				c.Set(ContextKeyClaims, claims)
			},
			wantOK: true,
		},
		{
			name:   "missing claims",
			setup:  func(c *gin.Context) {},
			wantOK: false,
		},
		{
			name: "wrong type",
			setup: func(c *gin.Context) {
				c.Set(ContextKeyClaims, "invalid")
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)

			claims, ok := GetClaims(c)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.NotNil(t, claims)
			}
		})
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// Create a mock JWT service (won't be used since header is missing)
	middleware := AuthMiddleware(nil)
	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization header required")
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "no bearer prefix",
			header: "token-without-bearer",
		},
		{
			name:   "wrong prefix",
			header: "Basic token123",
		},
		{
			name:   "only bearer",
			header: "Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.Header.Set("Authorization", tt.header)

			middleware := AuthMiddleware(nil)
			middleware(c)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid authorization header format")
		})
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()

	assert.Equal(t, 5, cfg.LoginLimit)
	assert.Equal(t, 3, cfg.RegisterLimit)
	assert.Equal(t, 100, cfg.GeneralLimit)
}
