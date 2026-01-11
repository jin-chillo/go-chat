package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jin-chillo/go-chat/internal/model"
)

// AuthService handles authentication business logic.
type AuthService struct {
	userRepo UserRepository
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(userRepo UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
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
