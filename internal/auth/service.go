package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
)

// ErrInvalidCredentials is returned when email or password is incorrect.
var ErrInvalidCredentials = errors.New("invalid credentials")

// AuthService handles authentication business logic.
type AuthService struct {
	userRepo         UserRepository
	tokenRepo        TokenRepository
	loginHistoryRepo LoginHistoryRepository
	jwtSvc           *JWTService
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(userRepo UserRepository, tokenRepo TokenRepository, jwtSvc *JWTService) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		jwtSvc:    jwtSvc,
	}
}

// SetLoginHistoryRepo sets the login history repository (optional dependency).
func (s *AuthService) SetLoginHistoryRepo(repo LoginHistoryRepository) {
	s.loginHistoryRepo = repo
}

// RecordLoginHistory records a login event in the login history.
func (s *AuthService) RecordLoginHistory(ctx context.Context, userID uuid.UUID, ip, userAgent string) error {
	if s.loginHistoryRepo == nil {
		return nil // Silently skip if repository not configured
	}

	history := model.NewLoginHistory(userID, ip, userAgent)
	if err := s.loginHistoryRepo.Create(ctx, history); err != nil {
		return fmt.Errorf("failed to record login history: %w", err)
	}

	return nil
}

// Register creates a new user account.
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
	// Normalize email to lowercase
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check if email already exists
	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &model.User{
		Email:        email,
		PasswordHash: passwordHash,
		Nickname:     strings.TrimSpace(req.Nickname),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// LoginResult contains the tokens generated after successful login.
type LoginResult struct {
	UserID       uuid.UUID
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// Login authenticates a user and returns access and refresh tokens.
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResult, error) {
	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := CheckPassword(req.Password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate access token
	accessToken, err := s.jwtSvc.GenerateAccessToken(user.ID, user.Email, user.Nickname)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken := s.jwtSvc.GenerateRefreshToken()
	refreshTokenHash := HashToken(refreshToken)

	// Store refresh token in database
	tokenRecord := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(s.jwtSvc.GetRefreshExpiry()),
	}

	if err := s.tokenRepo.Create(ctx, tokenRecord); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &LoginResult{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtSvc.GetAccessExpiry().Seconds()),
	}, nil
}

// Logout invalidates the current access token and revokes all refresh tokens for the user.
func (s *AuthService) Logout(ctx context.Context, claims *TokenClaims) error {
	// Blacklist the access token
	if err := s.jwtSvc.BlacklistToken(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	// Revoke all refresh tokens for the user
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID in claims: %w", err)
	}

	if err := s.tokenRepo.RevokeByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %w", err)
	}

	return nil
}

// RefreshResult contains the new access token after refresh.
type RefreshResult struct {
	AccessToken string
	ExpiresIn   int
}

// Refresh validates a refresh token and issues a new access token.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	// Hash the provided refresh token
	tokenHash := HashToken(refreshToken)

	// Find the refresh token in database
	tokenRecord, err := s.tokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	// Check if token is revoked
	if tokenRecord.IsRevoked() {
		return nil, ErrRefreshTokenRevoked
	}

	// Check if token is expired
	if tokenRecord.IsExpired() {
		return nil, ErrRefreshTokenExpired
	}

	// Get user information
	user, err := s.userRepo.FindByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate new access token
	accessToken, err := s.jwtSvc.GenerateAccessToken(user.ID, user.Email, user.Nickname)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &RefreshResult{
		AccessToken: accessToken,
		ExpiresIn:   int(s.jwtSvc.GetAccessExpiry().Seconds()),
	}, nil
}
