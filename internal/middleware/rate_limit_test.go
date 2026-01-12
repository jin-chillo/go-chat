package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimitExceededHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	rateLimitExceededHandler(c)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "RATE001", response["code"])
	assert.Contains(t, response["error"], "Rate limit exceeded")
}

func TestRateLimitConfig_Values(t *testing.T) {
	tests := []struct {
		name          string
		config        *RateLimitConfig
		wantLogin     int
		wantRegister  int
		wantGeneral   int
	}{
		{
			name: "custom config",
			config: &RateLimitConfig{
				LoginLimit:    10,
				RegisterLimit: 5,
				GeneralLimit:  200,
			},
			wantLogin:    10,
			wantRegister: 5,
			wantGeneral:  200,
		},
		{
			name: "zero values",
			config: &RateLimitConfig{
				LoginLimit:    0,
				RegisterLimit: 0,
				GeneralLimit:  0,
			},
			wantLogin:    0,
			wantRegister: 0,
			wantGeneral:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantLogin, tt.config.LoginLimit)
			assert.Equal(t, tt.wantRegister, tt.config.RegisterLimit)
			assert.Equal(t, tt.wantGeneral, tt.config.GeneralLimit)
		})
	}
}
