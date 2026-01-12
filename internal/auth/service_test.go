package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jin-chillo/go-chat/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var ErrDatabaseError = errors.New("database error")

// MockUserRepository is a mock implementation of UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// MockTokenRepository is a mock implementation of TokenRepository.
type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RefreshToken), args.Error(1)
}

func (m *MockTokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockTokenRepository) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	args := m.Called(ctx, tokenHash)
	return args.Error(0)
}

// MockJWTService wraps JWTService for testing without Redis.
type testJWTService struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func newTestJWTService() *JWTService {
	return &JWTService{
		secret:        []byte("test-secret-key-for-testing-only"),
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
		redisClient:   nil, // No Redis for unit tests
	}
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		request   *RegisterRequest
		setupMock func(*MockUserRepository)
		wantErr   error
	}{
		{
			name: "successful registration",
			request: &RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Nickname: "testuser",
			},
			setupMock: func(m *MockUserRepository) {
				m.On("FindByEmail", ctx, "test@example.com").Return(nil, ErrUserNotFound)
				m.On("Create", ctx, mock.AnythingOfType("*model.User")).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "email already exists",
			request: &RegisterRequest{
				Email:    "existing@example.com",
				Password: "password123",
				Nickname: "testuser",
			},
			setupMock: func(m *MockUserRepository) {
				existingUser := &model.User{
					ID:    uuid.New(),
					Email: "existing@example.com",
				}
				m.On("FindByEmail", ctx, "existing@example.com").Return(existingUser, nil)
			},
			wantErr: ErrEmailAlreadyExists,
		},
		{
			name: "email normalized to lowercase",
			request: &RegisterRequest{
				Email:    "TEST@EXAMPLE.COM",
				Password: "password123",
				Nickname: "testuser",
			},
			setupMock: func(m *MockUserRepository) {
				m.On("FindByEmail", ctx, "test@example.com").Return(nil, ErrUserNotFound)
				m.On("Create", ctx, mock.MatchedBy(func(u *model.User) bool {
					return u.Email == "test@example.com"
				})).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "create user error",
			request: &RegisterRequest{
				Email:    "error@example.com",
				Password: "password123",
				Nickname: "erroruser",
			},
			setupMock: func(m *MockUserRepository) {
				m.On("FindByEmail", ctx, "error@example.com").Return(nil, ErrUserNotFound)
				m.On("Create", ctx, mock.AnythingOfType("*model.User")).Return(ErrUserNotFound)
			},
			wantErr: ErrUserNotFound,
		},
		{
			name: "find by email database error",
			request: &RegisterRequest{
				Email:    "dberror@example.com",
				Password: "password123",
				Nickname: "dberroruser",
			},
			setupMock: func(m *MockUserRepository) {
				m.On("FindByEmail", ctx, "dberror@example.com").Return(nil, ErrDatabaseError)
			},
			wantErr: ErrDatabaseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockTokenRepo := new(MockTokenRepository)
			jwtSvc := newTestJWTService()

			tt.setupMock(mockUserRepo)

			service := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
			user, err := service.Register(ctx, tt.request)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.request.Nickname, user.Nickname)
			}

			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	// Create a user with known password
	password := "correctPassword123"
	passwordHash, _ := HashPassword(password)
	testUser := &model.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: passwordHash,
		Nickname:     "testuser",
	}

	tests := []struct {
		name      string
		request   *LoginRequest
		setupMock func(*MockUserRepository, *MockTokenRepository)
		wantErr   error
	}{
		{
			name: "successful login",
			request: &LoginRequest{
				Email:    "user@example.com",
				Password: password,
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", ctx, "user@example.com").Return(testUser, nil)
				tokenRepo.On("Create", ctx, mock.AnythingOfType("*model.RefreshToken")).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "user not found",
			request: &LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "anypassword",
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", ctx, "nonexistent@example.com").Return(nil, ErrUserNotFound)
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "wrong password",
			request: &LoginRequest{
				Email:    "user@example.com",
				Password: "wrongPassword",
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", ctx, "user@example.com").Return(testUser, nil)
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "token create error",
			request: &LoginRequest{
				Email:    "user@example.com",
				Password: password,
			},
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userRepo.On("FindByEmail", ctx, "user@example.com").Return(testUser, nil)
				tokenRepo.On("Create", ctx, mock.AnythingOfType("*model.RefreshToken")).Return(ErrUserNotFound)
			},
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockTokenRepo := new(MockTokenRepository)
			jwtSvc := newTestJWTService()

			tt.setupMock(mockUserRepo, mockTokenRepo)

			service := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
			result, err := service.Login(ctx, tt.request)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
				assert.Equal(t, testUser.ID, result.UserID)
			}

			mockUserRepo.AssertExpectations(t)
			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	ctx := context.Background()

	testUser := &model.User{
		ID:       uuid.New(),
		Email:    "user@example.com",
		Nickname: "testuser",
	}

	refreshToken := "valid-refresh-token"
	tokenHash := HashToken(refreshToken)

	tests := []struct {
		name      string
		token     string
		setupMock func(*MockUserRepository, *MockTokenRepository)
		wantErr   error
	}{
		{
			name:  "successful refresh",
			token: refreshToken,
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				tokenRecord := &model.RefreshToken{
					ID:        uuid.New(),
					UserID:    testUser.ID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				tokenRepo.On("FindByTokenHash", ctx, tokenHash).Return(tokenRecord, nil)
				userRepo.On("FindByID", ctx, testUser.ID).Return(testUser, nil)
			},
			wantErr: nil,
		},
		{
			name:  "token not found",
			token: "invalid-token",
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				tokenRepo.On("FindByTokenHash", ctx, mock.Anything).Return(nil, ErrRefreshTokenNotFound)
			},
			wantErr: ErrRefreshTokenNotFound,
		},
		{
			name:  "expired token",
			token: refreshToken,
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				expiredToken := &model.RefreshToken{
					ID:        uuid.New(),
					UserID:    testUser.ID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
				}
				tokenRepo.On("FindByTokenHash", ctx, tokenHash).Return(expiredToken, nil)
			},
			wantErr: ErrRefreshTokenExpired,
		},
		{
			name:  "revoked token",
			token: refreshToken,
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				revokedAt := time.Now().Add(-1 * time.Hour)
				revokedToken := &model.RefreshToken{
					ID:        uuid.New(),
					UserID:    testUser.ID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(24 * time.Hour),
					RevokedAt: &revokedAt,
				}
				tokenRepo.On("FindByTokenHash", ctx, tokenHash).Return(revokedToken, nil)
			},
			wantErr: ErrRefreshTokenRevoked,
		},
		{
			name:  "user not found",
			token: refreshToken,
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				tokenRecord := &model.RefreshToken{
					ID:        uuid.New(),
					UserID:    testUser.ID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				tokenRepo.On("FindByTokenHash", ctx, tokenHash).Return(tokenRecord, nil)
				userRepo.On("FindByID", ctx, testUser.ID).Return(nil, ErrUserNotFound)
			},
			wantErr: ErrUserNotFound,
		},
		{
			name:  "database error finding token",
			token: refreshToken,
			setupMock: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				tokenRepo.On("FindByTokenHash", ctx, tokenHash).Return(nil, ErrDatabaseError)
			},
			wantErr: ErrDatabaseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockTokenRepo := new(MockTokenRepository)
			jwtSvc := newTestJWTService()

			tt.setupMock(mockUserRepo, mockTokenRepo)

			service := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
			result, err := service.Refresh(ctx, tt.token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.AccessToken)
			}

			mockUserRepo.AssertExpectations(t)
			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestHashToken(t *testing.T) {
	token := "test-refresh-token"

	hash1 := HashToken(token)
	hash2 := HashToken(token)

	// Same input should produce same output (deterministic)
	assert.Equal(t, hash1, hash2)

	// Different input should produce different output
	hash3 := HashToken("different-token")
	assert.NotEqual(t, hash1, hash3)

	// Hash should be hex-encoded SHA-256 (64 characters)
	assert.Len(t, hash1, 64)
}

func TestAuthService_Login_DatabaseError(t *testing.T) {
	ctx := context.Background()

	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSvc := newTestJWTService()

	// Simulate database error when finding user
	mockUserRepo.On("FindByEmail", ctx, "error@example.com").Return(nil, ErrDatabaseError)

	service := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
	result, err := service.Login(ctx, &LoginRequest{
		Email:    "error@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrDatabaseError)
	assert.Nil(t, result)

	mockUserRepo.AssertExpectations(t)
}

// MockLoginHistoryRepository is a mock implementation of LoginHistoryRepository.
type MockLoginHistoryRepository struct {
	mock.Mock
}

func (m *MockLoginHistoryRepository) Create(ctx context.Context, history *model.LoginHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockLoginHistoryRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*model.LoginHistory, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.LoginHistory), args.Error(1)
}

func TestAuthService_SetLoginHistoryRepo(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	jwtSvc := newTestJWTService()

	service := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
	assert.Nil(t, service.loginHistoryRepo)

	mockLoginHistoryRepo := new(MockLoginHistoryRepository)
	service.SetLoginHistoryRepo(mockLoginHistoryRepo)

	assert.NotNil(t, service.loginHistoryRepo)
}

func TestAuthService_RecordLoginHistory(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	ip := "192.168.1.1"
	userAgent := "Mozilla/5.0"

	tests := []struct {
		name      string
		setupRepo func(*AuthService)
		setupMock func(*MockLoginHistoryRepository)
		wantErr   bool
	}{
		{
			name: "successful record",
			setupRepo: func(s *AuthService) {
				mockRepo := new(MockLoginHistoryRepository)
				mockRepo.On("Create", ctx, mock.AnythingOfType("*model.LoginHistory")).Return(nil)
				s.SetLoginHistoryRepo(mockRepo)
			},
			wantErr: false,
		},
		{
			name: "repo not configured - silently skips",
			setupRepo: func(s *AuthService) {
				// Don't set the repo
			},
			wantErr: false,
		},
		{
			name: "repo error",
			setupRepo: func(s *AuthService) {
				mockRepo := new(MockLoginHistoryRepository)
				mockRepo.On("Create", ctx, mock.AnythingOfType("*model.LoginHistory")).Return(ErrUserNotFound)
				s.SetLoginHistoryRepo(mockRepo)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockTokenRepo := new(MockTokenRepository)
			jwtSvc := newTestJWTService()

			service := NewAuthService(mockUserRepo, mockTokenRepo, jwtSvc)
			tt.setupRepo(service)

			err := service.RecordLoginHistory(ctx, userID, ip, userAgent)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
