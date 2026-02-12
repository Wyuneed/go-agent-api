// Package mocks provides testify mock implementations of domain repository interfaces.
package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
)

// MockUserRepository is a mock of repository.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *entity.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *entity.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// MockTokenRepository is a mock of repository.TokenRepository.
type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) Create(ctx context.Context, token *entity.Token) error {
	return m.Called(ctx, token).Error(0)
}

func (m *MockTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*entity.Token, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Token), args.Error(1)
}

func (m *MockTokenRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Token, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Token), args.Error(1)
}

func (m *MockTokenRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Token, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Token), args.Error(1)
}

func (m *MockTokenRepository) UpdateLastUsed(ctx context.Context, tokenID uuid.UUID) error {
	return m.Called(ctx, tokenID).Error(0)
}

func (m *MockTokenRepository) Revoke(ctx context.Context, tokenID uuid.UUID) error {
	return m.Called(ctx, tokenID).Error(0)
}

func (m *MockTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// MockConversationRepository is a mock of repository.ConversationRepository.
type MockConversationRepository struct {
	mock.Mock
}

func (m *MockConversationRepository) Create(ctx context.Context, conv *entity.Conversation) error {
	return m.Called(ctx, conv).Error(0)
}

func (m *MockConversationRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Conversation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Conversation), args.Error(1)
}

func (m *MockConversationRepository) FindByUserID(ctx context.Context, userID uuid.UUID, filter repository.ConversationFilter) ([]*entity.Conversation, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Conversation), args.Error(1)
}

func (m *MockConversationRepository) Update(ctx context.Context, conv *entity.Conversation) error {
	return m.Called(ctx, conv).Error(0)
}

func (m *MockConversationRepository) UpdateWorkflowState(ctx context.Context, id uuid.UUID, state map[string]any) error {
	return m.Called(ctx, id, state).Error(0)
}

func (m *MockConversationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockConversationRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// MockMessageRepository is a mock of repository.MessageRepository.
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) Create(ctx context.Context, msg *entity.Message) error {
	return m.Called(ctx, msg).Error(0)
}

func (m *MockMessageRepository) CreateBatch(ctx context.Context, msgs []*entity.Message) error {
	return m.Called(ctx, msgs).Error(0)
}

func (m *MockMessageRepository) FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*entity.Message, error) {
	args := m.Called(ctx, conversationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Message), args.Error(1)
}

func (m *MockMessageRepository) FindByConversationIDPaginated(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
	args := m.Called(ctx, conversationID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Message), args.Error(1)
}

func (m *MockMessageRepository) FindLastN(ctx context.Context, conversationID uuid.UUID, n int) ([]*entity.Message, error) {
	args := m.Called(ctx, conversationID, n)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Message), args.Error(1)
}

func (m *MockMessageRepository) CountByConversationID(ctx context.Context, conversationID uuid.UUID) (int64, error) {
	args := m.Called(ctx, conversationID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockMessageRepository) DeleteByConversationID(ctx context.Context, conversationID uuid.UUID) error {
	return m.Called(ctx, conversationID).Error(0)
}
