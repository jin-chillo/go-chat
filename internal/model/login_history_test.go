package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewLoginHistory(t *testing.T) {
	userID := uuid.New()
	ip := "192.168.1.1"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

	before := time.Now()
	history := NewLoginHistory(userID, ip, userAgent)
	after := time.Now()

	assert.Equal(t, userID, history.UserID)
	assert.Equal(t, ip, history.IP)
	assert.Equal(t, userAgent, history.UserAgent)
	assert.True(t, history.CreatedAt.After(before) || history.CreatedAt.Equal(before))
	assert.True(t, history.CreatedAt.Before(after) || history.CreatedAt.Equal(after))
}
