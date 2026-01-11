package auth

import "github.com/jin-chillo/go-chat/internal/model"

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Nickname string `json:"nickname" binding:"required,min=2,max=50"`
}

// RegisterResponse represents the response body for user registration.
type RegisterResponse struct {
	User *UserResponse `json:"user"`
}

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	CreatedAt string `json:"created_at"`
}

// NewUserResponse creates a UserResponse from a User model.
func NewUserResponse(user *model.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Nickname:  user.Nickname,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the response body for user login.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// LogoutResponse represents the response body for user logout.
type LogoutResponse struct {
	Message string `json:"message"`
}

// RefreshRequest represents the request body for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshResponse represents the response body for token refresh.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
