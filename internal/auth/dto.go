package auth

import "github.com/jin-chillo/go-chat/internal/model"

// RegisterRequest represents the request body for user registration.
// @Description 회원가입 요청 데이터
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"password123"`
	Nickname string `json:"nickname" binding:"required,min=2,max=50" example:"홍길동"`
}

// RegisterResponse represents the response body for user registration.
// @Description 회원가입 응답 데이터
type RegisterResponse struct {
	User *UserResponse `json:"user"`
}

// UserResponse represents a user in API responses.
// @Description 사용자 정보
type UserResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string `json:"email" example:"user@example.com"`
	Nickname  string `json:"nickname" example:"홍길동"`
	CreatedAt string `json:"created_at" example:"2024-01-15T09:30:00Z"`
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
// @Description 에러 응답 데이터
type ErrorResponse struct {
	Error   string `json:"error" example:"Validation failed"`
	Code    string `json:"code,omitempty" example:"AUTH001"`
	Details string `json:"details,omitempty" example:"email is required"`
}

// LoginRequest represents the request body for user login.
// @Description 로그인 요청 데이터
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// LoginResponse represents the response body for user login.
// @Description 로그인 응답 데이터
type LoginResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"550e8400-e29b-41d4-a716-446655440000"`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int    `json:"expires_in" example:"3600"`
}

// LogoutResponse represents the response body for user logout.
// @Description 로그아웃 응답 데이터
type LogoutResponse struct {
	Message string `json:"message" example:"Successfully logged out"`
}

// RefreshRequest represents the request body for token refresh.
// @Description 토큰 갱신 요청 데이터
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// RefreshResponse represents the response body for token refresh.
// @Description 토큰 갱신 응답 데이터
type RefreshResponse struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int    `json:"expires_in" example:"3600"`
}
