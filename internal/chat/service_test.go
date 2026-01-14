package chat

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
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MockChannelRepository is a mock implementation of ChannelRepository.
type MockChannelRepository struct {
	mock.Mock
}

func (m *MockChannelRepository) Create(ctx context.Context, channel *model.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Channel), args.Error(1)
}

func (m *MockChannelRepository) FindByIDWithMembers(ctx context.Context, id uuid.UUID) (*model.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Channel), args.Error(1)
}

func (m *MockChannelRepository) FindAll(ctx context.Context, offset, limit int) ([]model.Channel, int64, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]model.Channel), args.Get(1).(int64), args.Error(2)
}

func (m *MockChannelRepository) FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.Channel, int64, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]model.Channel), args.Get(1).(int64), args.Error(2)
}

func (m *MockChannelRepository) Update(ctx context.Context, channel *model.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChannelRepository) AddMember(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepository) RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepository) IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, channelID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockChannelRepository) GetMembers(ctx context.Context, channelID uuid.UUID) ([]model.ChannelMember, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).([]model.ChannelMember), args.Error(1)
}

func (m *MockChannelRepository) CountMembers(ctx context.Context, channelID uuid.UUID) (int64, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).(int64), args.Error(1)
}

// MockMessageRepository is a mock implementation of MessageRepository.
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) Create(ctx context.Context, message *model.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageRepository) FindByChannelID(ctx context.Context, channelID uuid.UUID, before *time.Time, limit int) ([]model.Message, error) {
	args := m.Called(ctx, channelID, before, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Message), args.Error(1)
}

func (m *MockMessageRepository) CountByChannelID(ctx context.Context, channelID uuid.UUID) (int64, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).(int64), args.Error(1)
}

func TestService_CreateChannel(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()

	tests := []struct {
		name      string
		request   *CreateChannelRequest
		setupMock func(*MockChannelRepository)
		wantErr   bool
	}{
		{
			name: "successful creation",
			request: &CreateChannelRequest{
				Name:        "Test Channel",
				Description: "A test channel",
			},
			setupMock: func(m *MockChannelRepository) {
				m.On("Create", ctx, mock.AnythingOfType("*model.Channel")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "repository error",
			request: &CreateChannelRequest{
				Name: "Test Channel",
			},
			setupMock: func(m *MockChannelRepository) {
				m.On("Create", ctx, mock.AnythingOfType("*model.Channel")).Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			channel, err := svc.CreateChannel(ctx, ownerID, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, channel)
				assert.Equal(t, tt.request.Name, channel.Name)
				assert.Equal(t, ownerID, channel.OwnerID)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_UpdateChannel(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	existingChannel := &model.Channel{
		ID:          channelID,
		Name:        "Original Name",
		Description: "Original Description",
		OwnerID:     ownerID,
	}

	tests := []struct {
		name      string
		userID    uuid.UUID
		request   *UpdateChannelRequest
		setupMock func(*MockChannelRepository)
		wantErr   error
	}{
		{
			name:   "successful update by owner",
			userID: ownerID,
			request: &UpdateChannelRequest{
				Name:        "Updated Name",
				Description: "Updated Description",
			},
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				m.On("Update", ctx, mock.AnythingOfType("*model.Channel")).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:   "not owner",
			userID: otherUserID,
			request: &UpdateChannelRequest{
				Name: "Updated Name",
			},
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
			},
			wantErr: ErrNotChannelOwner,
		},
		{
			name:   "channel not found",
			userID: ownerID,
			request: &UpdateChannelRequest{
				Name: "Updated Name",
			},
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(nil, ErrChannelNotFound)
			},
			wantErr: ErrChannelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			channel, err := svc.UpdateChannel(ctx, channelID, tt.userID, tt.request)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, channel)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_DeleteChannel(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	existingChannel := &model.Channel{
		ID:      channelID,
		OwnerID: ownerID,
	}

	tests := []struct {
		name       string
		userID     uuid.UUID
		setupMock  func(*MockChannelRepository)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:   "successful delete by owner",
			userID: ownerID,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				m.On("Delete", ctx, channelID).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:   "not owner",
			userID: otherUserID,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
			},
			wantErr: ErrNotChannelOwner,
		},
		{
			name:   "channel not found",
			userID: ownerID,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(nil, ErrChannelNotFound)
			},
			wantErr: ErrChannelNotFound,
		},
		{
			name:   "delete error",
			userID: ownerID,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				m.On("Delete", ctx, channelID).Return(errors.New("database error"))
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			err := svc.DeleteChannel(ctx, channelID, tt.userID)

			if tt.wantAnyErr {
				assert.Error(t, err)
			} else if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_JoinChannel(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	userID := uuid.New()

	existingChannel := &model.Channel{ID: channelID}

	tests := []struct {
		name       string
		setupMock  func(*MockChannelRepository)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "successful join",
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				m.On("AddMember", ctx, channelID, userID).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "channel not found",
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(nil, ErrChannelNotFound)
			},
			wantErr: ErrChannelNotFound,
		},
		{
			name: "already member",
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				m.On("AddMember", ctx, channelID, userID).Return(ErrAlreadyMember)
			},
			wantErr: ErrAlreadyMember,
		},
		{
			name: "add member database error",
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				m.On("AddMember", ctx, channelID, userID).Return(errors.New("database error"))
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			err := svc.JoinChannel(ctx, channelID, userID)

			if tt.wantAnyErr {
				assert.Error(t, err)
			} else if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_LeaveChannel(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	existingChannel := &model.Channel{
		ID:      channelID,
		OwnerID: ownerID,
	}

	tests := []struct {
		name      string
		userID    uuid.UUID
		setupMock func(*MockChannelRepository)
		wantErr   error
	}{
		{
			name:   "successful leave",
			userID: memberID,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				m.On("RemoveMember", ctx, channelID, memberID).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:   "owner cannot leave",
			userID: ownerID,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
			},
			wantErr: ErrOwnerCannotLeave,
		},
		{
			name:   "not a member",
			userID: memberID,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				m.On("RemoveMember", ctx, channelID, memberID).Return(ErrNotMember)
			},
			wantErr: ErrNotMember,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			err := svc.LeaveChannel(ctx, channelID, tt.userID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_ListChannels(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		page          int
		limit         int
		expectedPage  int
		expectedLimit int
	}{
		{
			name:          "valid pagination",
			page:          1,
			limit:         20,
			expectedPage:  1,
			expectedLimit: 20,
		},
		{
			name:          "page below 1 defaults to 1",
			page:          0,
			limit:         20,
			expectedPage:  1,
			expectedLimit: 20,
		},
		{
			name:          "limit below 1 defaults to 20",
			page:          1,
			limit:         0,
			expectedPage:  1,
			expectedLimit: 20,
		},
		{
			name:          "limit above 100 defaults to 20",
			page:          1,
			limit:         150,
			expectedPage:  1,
			expectedLimit: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			expectedOffset := (tt.expectedPage - 1) * tt.expectedLimit

			mockChannelRepo.On("FindAll", ctx, expectedOffset, tt.expectedLimit).
				Return([]model.Channel{}, int64(0), nil)

			svc := NewService(mockChannelRepo, nil)
			_, _, err := svc.ListChannels(ctx, tt.page, tt.limit)

			assert.NoError(t, err)
			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_CreateMessage(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	userID := uuid.New()

	existingChannel := &model.Channel{ID: channelID}

	tests := []struct {
		name      string
		setupMock func(*MockChannelRepository, *MockMessageRepository)
		wantErr   bool
	}{
		{
			name: "successful message creation",
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				cr.On("IsMember", ctx, channelID, userID).Return(true, nil)
				mr.On("Create", ctx, mock.AnythingOfType("*model.Message")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "channel not found",
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(nil, ErrChannelNotFound)
			},
			wantErr: true,
		},
		{
			name: "not a member",
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				cr.On("IsMember", ctx, channelID, userID).Return(false, nil)
			},
			wantErr: true,
		},
		{
			name: "membership check error",
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				cr.On("IsMember", ctx, channelID, userID).Return(false, ErrChannelNotFound)
			},
			wantErr: true,
		},
		{
			name: "message creation error",
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				cr.On("IsMember", ctx, channelID, userID).Return(true, nil)
				mr.On("Create", ctx, mock.AnythingOfType("*model.Message")).Return(ErrChannelNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			mockMessageRepo := new(MockMessageRepository)
			tt.setupMock(mockChannelRepo, mockMessageRepo)

			svc := NewService(mockChannelRepo, mockMessageRepo)
			msg, err := svc.CreateMessage(ctx, channelID, userID, "testuser", "Hello!")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, msg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, msg)
				assert.Equal(t, "Hello!", msg.Content)
			}

			mockChannelRepo.AssertExpectations(t)
			mockMessageRepo.AssertExpectations(t)
		})
	}
}

func TestService_CreateMessage_NoMessageRepo(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	userID := uuid.New()

	mockChannelRepo := new(MockChannelRepository)
	svc := NewService(mockChannelRepo, nil)

	message, err := svc.CreateMessage(ctx, channelID, userID, "testuser", "Hello")

	assert.Error(t, err)
	assert.Nil(t, message)
	assert.Contains(t, err.Error(), "message repository not configured")
}

func TestService_GetMessages(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	userID := uuid.New()

	existingChannel := &model.Channel{ID: channelID}

	tests := []struct {
		name        string
		limit       int
		setupMock   func(*MockChannelRepository, *MockMessageRepository)
		wantHasMore bool
		wantCount   int
		wantErr     bool
	}{
		{
			name:  "returns messages without has more",
			limit: 50,
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				messages := []model.Message{
					{ID: bson.NewObjectID(), ChannelID: channelID, UserID: userID, Content: "msg1"},
					{ID: bson.NewObjectID(), ChannelID: channelID, UserID: userID, Content: "msg2"},
				}
				mr.On("FindByChannelID", ctx, channelID, (*time.Time)(nil), 51).Return(messages, nil)
			},
			wantHasMore: false,
			wantCount:   2,
			wantErr:     false,
		},
		{
			name:  "returns messages with has more",
			limit: 2,
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				messages := []model.Message{
					{ID: bson.NewObjectID(), ChannelID: channelID, UserID: userID, Content: "msg1"},
					{ID: bson.NewObjectID(), ChannelID: channelID, UserID: userID, Content: "msg2"},
					{ID: bson.NewObjectID(), ChannelID: channelID, UserID: userID, Content: "msg3"},
				}
				mr.On("FindByChannelID", ctx, channelID, (*time.Time)(nil), 3).Return(messages, nil)
			},
			wantHasMore: true,
			wantCount:   2,
			wantErr:     false,
		},
		{
			name:  "invalid limit defaults to 50",
			limit: 0,
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				mr.On("FindByChannelID", ctx, channelID, (*time.Time)(nil), 51).Return([]model.Message{}, nil)
			},
			wantHasMore: false,
			wantCount:   0,
			wantErr:     false,
		},
		{
			name:  "channel not found",
			limit: 50,
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(nil, ErrChannelNotFound)
			},
			wantErr: true,
		},
		{
			name:  "message repository error",
			limit: 50,
			setupMock: func(cr *MockChannelRepository, mr *MockMessageRepository) {
				cr.On("FindByID", ctx, channelID).Return(existingChannel, nil)
				mr.On("FindByChannelID", ctx, channelID, (*time.Time)(nil), 51).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			mockMessageRepo := new(MockMessageRepository)
			tt.setupMock(mockChannelRepo, mockMessageRepo)

			svc := NewService(mockChannelRepo, mockMessageRepo)
			messages, hasMore, err := svc.GetMessages(ctx, channelID, nil, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantHasMore, hasMore)
				assert.Len(t, messages, tt.wantCount)
			}

			mockChannelRepo.AssertExpectations(t)
			mockMessageRepo.AssertExpectations(t)
		})
	}
}

func TestService_GetChannel(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	ownerID := uuid.New()

	tests := []struct {
		name      string
		setupMock func(*MockChannelRepository)
		wantErr   bool
	}{
		{
			name: "successful get",
			setupMock: func(m *MockChannelRepository) {
				channel := &model.Channel{
					ID:      channelID,
					OwnerID: ownerID,
					Name:    "Test Channel",
				}
				m.On("FindByID", ctx, channelID).Return(channel, nil)
			},
			wantErr: false,
		},
		{
			name: "channel not found",
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(nil, ErrChannelNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			channel, err := svc.GetChannel(ctx, channelID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, channel)
				assert.Equal(t, channelID, channel.ID)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_GetChannelDetail(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	ownerID := uuid.New()

	tests := []struct {
		name      string
		setupMock func(*MockChannelRepository)
		wantErr   bool
	}{
		{
			name: "successful get with members",
			setupMock: func(m *MockChannelRepository) {
				channel := &model.Channel{
					ID:      channelID,
					OwnerID: ownerID,
					Name:    "Test Channel",
					Members: []model.ChannelMember{
						{ChannelID: channelID, UserID: ownerID},
					},
				}
				m.On("FindByIDWithMembers", ctx, channelID).Return(channel, nil)
			},
			wantErr: false,
		},
		{
			name: "channel not found",
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByIDWithMembers", ctx, channelID).Return(nil, ErrChannelNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			channel, err := svc.GetChannelDetail(ctx, channelID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, channel)
				assert.Equal(t, channelID, channel.ID)
				assert.NotEmpty(t, channel.Members)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_ListUserChannels(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	tests := []struct {
		name      string
		page      int
		limit     int
		setupMock func(*MockChannelRepository)
		wantCount int
		wantTotal int64
		wantErr   bool
	}{
		{
			name:  "successful list",
			page:  1,
			limit: 10,
			setupMock: func(m *MockChannelRepository) {
				channels := []model.Channel{
					{ID: uuid.New(), Name: "Channel 1"},
					{ID: uuid.New(), Name: "Channel 2"},
				}
				m.On("FindByUserID", ctx, userID, 0, 10).Return(channels, int64(2), nil)
			},
			wantCount: 2,
			wantTotal: 2,
			wantErr:   false,
		},
		{
			name:  "page normalization",
			page:  0,
			limit: 10,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByUserID", ctx, userID, 0, 10).Return([]model.Channel{}, int64(0), nil)
			},
			wantCount: 0,
			wantTotal: 0,
			wantErr:   false,
		},
		{
			name:  "limit normalization - too high",
			page:  1,
			limit: 200,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByUserID", ctx, userID, 0, 20).Return([]model.Channel{}, int64(0), nil)
			},
			wantCount: 0,
			wantTotal: 0,
			wantErr:   false,
		},
		{
			name:  "limit normalization - zero",
			page:  1,
			limit: 0,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByUserID", ctx, userID, 0, 20).Return([]model.Channel{}, int64(0), nil)
			},
			wantCount: 0,
			wantTotal: 0,
			wantErr:   false,
		},
		{
			name:  "repository error",
			page:  1,
			limit: 10,
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByUserID", ctx, userID, 0, 10).Return([]model.Channel{}, int64(0), ErrChannelNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			channels, total, err := svc.ListUserChannels(ctx, userID, tt.page, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, channels, tt.wantCount)
				assert.Equal(t, tt.wantTotal, total)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_CountMembers(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()

	tests := []struct {
		name      string
		setupMock func(*MockChannelRepository)
		wantCount int64
		wantErr   bool
	}{
		{
			name: "successful count",
			setupMock: func(m *MockChannelRepository) {
				m.On("CountMembers", ctx, channelID).Return(int64(5), nil)
			},
			wantCount: 5,
			wantErr:   false,
		},
		{
			name: "repository error",
			setupMock: func(m *MockChannelRepository) {
				m.On("CountMembers", ctx, channelID).Return(int64(0), ErrChannelNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			count, err := svc.CountMembers(ctx, channelID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_IsMember(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		setupMock  func(*MockChannelRepository)
		wantMember bool
		wantErr    bool
	}{
		{
			name: "user is member",
			setupMock: func(m *MockChannelRepository) {
				m.On("IsMember", ctx, channelID, userID).Return(true, nil)
			},
			wantMember: true,
			wantErr:    false,
		},
		{
			name: "user is not member",
			setupMock: func(m *MockChannelRepository) {
				m.On("IsMember", ctx, channelID, userID).Return(false, nil)
			},
			wantMember: false,
			wantErr:    false,
		},
		{
			name: "repository error",
			setupMock: func(m *MockChannelRepository) {
				m.On("IsMember", ctx, channelID, userID).Return(false, ErrChannelNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			isMember, err := svc.IsMember(ctx, channelID, userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantMember, isMember)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}

func TestService_GetChannelMembers(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	ownerID := uuid.New()

	tests := []struct {
		name      string
		setupMock func(*MockChannelRepository)
		wantCount int
		wantErr   bool
	}{
		{
			name: "successful get members",
			setupMock: func(m *MockChannelRepository) {
				channel := &model.Channel{ID: channelID, OwnerID: ownerID}
				m.On("FindByID", ctx, channelID).Return(channel, nil)
				members := []model.ChannelMember{
					{ChannelID: channelID, UserID: ownerID},
					{ChannelID: channelID, UserID: uuid.New()},
				}
				m.On("GetMembers", ctx, channelID).Return(members, nil)
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "channel not found",
			setupMock: func(m *MockChannelRepository) {
				m.On("FindByID", ctx, channelID).Return(nil, ErrChannelNotFound)
			},
			wantErr: true,
		},
		{
			name: "get members error",
			setupMock: func(m *MockChannelRepository) {
				channel := &model.Channel{ID: channelID, OwnerID: ownerID}
				m.On("FindByID", ctx, channelID).Return(channel, nil)
				m.On("GetMembers", ctx, channelID).Return([]model.ChannelMember{}, ErrChannelNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockChannelRepo := new(MockChannelRepository)
			tt.setupMock(mockChannelRepo)

			svc := NewService(mockChannelRepo, nil)
			members, err := svc.GetChannelMembers(ctx, channelID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, members, tt.wantCount)
			}

			mockChannelRepo.AssertExpectations(t)
		})
	}
}
