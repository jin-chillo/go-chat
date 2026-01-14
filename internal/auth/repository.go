package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jin-chillo/go-chat/internal/model"
	"gorm.io/gorm"
)

var (
	// ErrUserNotFound is returned when a user is not found.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailAlreadyExists is returned when email is already registered.
	ErrEmailAlreadyExists = errors.New("email already exists")
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
}

// userRepository implements UserRepository using GORM.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create inserts a new user into the database.
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", result.Error)
	}
	return nil
}

// FindByID retrieves a user by their ID.
func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by id: %w", result.Error)
	}
	return &user, nil
}

// FindByEmail retrieves a user by their email address.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", result.Error)
	}
	return &user, nil
}

// Update updates an existing user in the database.
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to update user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// isUniqueViolation checks if the error is a unique constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	// Check PostgreSQL specific error type
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// PostgreSQL unique violation error code: 23505
		return pgErr.Code == "23505"
	}

	// Fallback: check error message for unique constraint keywords
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "unique") || strings.Contains(errMsg, "duplicate")
}
