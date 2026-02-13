package chat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wyuneed/go-agent-api/internal/application/port"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/chat"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/tests/mocks"
)

// newMockChatResp returns a minimal LLM response.
func newMockChatResp(content string) *port.ChatResponse {
	return &port.ChatResponse{
		ID:    "chatcmpl-test",
		Model: "gpt-4o-mini",
		Choices: []port.Choice{
			{
				Message:      port.ChatMessage{Role: "assistant", Content: content},
				FinishReason: "stop",
			},
		},
		Usage: port.UsageInfo{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}
}

// ---- SendMessage — new conversation ----

func TestSendMessage_NewConversation_Success(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}
	llm := &mocks.MockLLMProvider{}

	userID := uuid.New()

	convRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)
	msgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Message")).Return(nil)
	msgRepo.On("FindByConversationID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return([]*entity.Message{}, nil)
	llm.On("Chat", mock.Anything, mock.AnythingOfType("port.ChatRequest")).
		Return(newMockChatResp("Hello!"), nil)
	convRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	uc := chat.NewSendMessageUseCase(convRepo, msgRepo, llm, "")
	out, err := uc.Execute(context.Background(), chat.SendMessageInput{
		UserID:    userID,
		Content:   "Hi there",
		AgentType: "general",
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello!", out.Response)
	assert.NotEqual(t, uuid.Nil, out.ConversationID)
	assert.NotNil(t, out.Message)

	convRepo.AssertExpectations(t)
	msgRepo.AssertExpectations(t)
	llm.AssertExpectations(t)
}

func TestSendMessage_NewConversation_DefaultAgent(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}
	llm := &mocks.MockLLMProvider{}

	convRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *entity.Conversation) bool {
		return c.AgentType == "general"
	})).Return(nil)
	msgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Message")).Return(nil)
	msgRepo.On("FindByConversationID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return([]*entity.Message{}, nil)
	llm.On("Chat", mock.Anything, mock.AnythingOfType("port.ChatRequest")).
		Return(newMockChatResp("OK"), nil)
	convRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	uc := chat.NewSendMessageUseCase(convRepo, msgRepo, llm, "")
	// AgentType not set — should default to "general"
	_, err := uc.Execute(context.Background(), chat.SendMessageInput{
		UserID:  uuid.New(),
		Content: "test",
	})

	require.NoError(t, err)
	convRepo.AssertExpectations(t)
}

// ---- SendMessage — existing conversation ----

func TestSendMessage_ExistingConversation_Success(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}
	llm := &mocks.MockLLMProvider{}

	userID := uuid.New()
	conv := entity.NewConversation(userID, "general")
	convID := conv.ID

	convRepo.On("FindByID", mock.Anything, convID).Return(conv, nil)
	msgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Message")).Return(nil)
	msgRepo.On("FindByConversationID", mock.Anything, convID).Return([]*entity.Message{}, nil)
	llm.On("Chat", mock.Anything, mock.AnythingOfType("port.ChatRequest")).
		Return(newMockChatResp("Great!"), nil)
	convRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	uc := chat.NewSendMessageUseCase(convRepo, msgRepo, llm, "")
	out, err := uc.Execute(context.Background(), chat.SendMessageInput{
		ConversationID: &convID,
		UserID:         userID,
		Content:        "follow-up",
	})

	require.NoError(t, err)
	assert.Equal(t, "Great!", out.Response)
	assert.Equal(t, convID, out.ConversationID)
}

func TestSendMessage_ExistingConversation_NotFound(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}
	llm := &mocks.MockLLMProvider{}

	convID := uuid.New()
	convRepo.On("FindByID", mock.Anything, convID).Return(nil, errors.New("not found"))

	uc := chat.NewSendMessageUseCase(convRepo, msgRepo, llm, "")
	_, err := uc.Execute(context.Background(), chat.SendMessageInput{
		ConversationID: &convID,
		UserID:         uuid.New(),
		Content:        "hello",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conversation not found")
}

func TestSendMessage_ExistingConversation_WrongUser(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}
	llm := &mocks.MockLLMProvider{}

	ownerID := uuid.New()
	conv := entity.NewConversation(ownerID, "general")
	convID := conv.ID

	convRepo.On("FindByID", mock.Anything, convID).Return(conv, nil)

	uc := chat.NewSendMessageUseCase(convRepo, msgRepo, llm, "")
	_, err := uc.Execute(context.Background(), chat.SendMessageInput{
		ConversationID: &convID,
		UserID:         uuid.New(), // different user
		Content:        "hack",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestSendMessage_LLMError(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}
	llm := &mocks.MockLLMProvider{}

	userID := uuid.New()

	convRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)
	msgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Message")).Return(nil)
	msgRepo.On("FindByConversationID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return([]*entity.Message{}, nil)
	llm.On("Chat", mock.Anything, mock.AnythingOfType("port.ChatRequest")).
		Return(nil, errors.New("LLM unavailable"))

	uc := chat.NewSendMessageUseCase(convRepo, msgRepo, llm, "")
	_, err := uc.Execute(context.Background(), chat.SendMessageInput{
		UserID:  userID,
		Content: "anything",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM call")
}

func TestSendMessage_DefaultModel(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}
	llm := &mocks.MockLLMProvider{}

	convRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)
	msgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Message")).Return(nil)
	msgRepo.On("FindByConversationID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return([]*entity.Message{}, nil)
	llm.On("Chat", mock.Anything, mock.MatchedBy(func(req port.ChatRequest) bool {
		return req.Model == "gpt-4o-mini"
	})).Return(newMockChatResp("done"), nil)
	convRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	uc := chat.NewSendMessageUseCase(convRepo, msgRepo, llm, "")
	_, err := uc.Execute(context.Background(), chat.SendMessageInput{
		UserID:  uuid.New(),
		Content: "test",
		// Model intentionally empty — should default to gpt-4o-mini
	})

	require.NoError(t, err)
	llm.AssertExpectations(t)
}

// ---- GetConversation ----

func TestGetConversation_Success(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}

	userID := uuid.New()
	conv := entity.NewConversation(userID, "general")
	msgs := []*entity.Message{
		entity.NewUserMessage(conv.ID, "hello"),
		entity.NewAssistantMessage(conv.ID, "hi", nil, "gpt-4o-mini"),
	}

	convRepo.On("FindByID", mock.Anything, conv.ID).Return(conv, nil)
	msgRepo.On("FindByConversationID", mock.Anything, conv.ID).Return(msgs, nil)

	uc := chat.NewGetConversationUseCase(convRepo, msgRepo)
	out, err := uc.Execute(context.Background(), conv.ID, userID)

	require.NoError(t, err)
	assert.Equal(t, conv.ID, out.Conversation.ID)
	assert.Len(t, out.Messages, 2)
}

func TestGetConversation_NotFound(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}

	convID := uuid.New()
	convRepo.On("FindByID", mock.Anything, convID).Return(nil, errors.New("not found"))

	uc := chat.NewGetConversationUseCase(convRepo, msgRepo)
	_, err := uc.Execute(context.Background(), convID, uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conversation not found")
}

func TestGetConversation_WrongUser(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}
	msgRepo := &mocks.MockMessageRepository{}

	ownerID := uuid.New()
	conv := entity.NewConversation(ownerID, "general")
	convRepo.On("FindByID", mock.Anything, conv.ID).Return(conv, nil)

	uc := chat.NewGetConversationUseCase(convRepo, msgRepo)
	_, err := uc.Execute(context.Background(), conv.ID, uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

// ---- ListConversations ----

func TestListConversations_Success(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}

	userID := uuid.New()
	convs := []*entity.Conversation{
		entity.NewConversation(userID, "general"),
		entity.NewConversation(userID, "coder"),
	}

	convRepo.On("FindByUserID", mock.Anything, userID, mock.AnythingOfType("repository.ConversationFilter")).
		Return(convs, nil)
	convRepo.On("CountByUserID", mock.Anything, userID).Return(int64(2), nil)

	uc := chat.NewListConversationsUseCase(convRepo)
	out, err := uc.Execute(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, out.Conversations, 2)
	assert.Equal(t, int64(2), out.Total)
}

func TestListConversations_Empty(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}

	userID := uuid.New()
	convRepo.On("FindByUserID", mock.Anything, userID, mock.AnythingOfType("repository.ConversationFilter")).
		Return([]*entity.Conversation{}, nil)
	convRepo.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	uc := chat.NewListConversationsUseCase(convRepo)
	out, err := uc.Execute(context.Background(), userID)

	require.NoError(t, err)
	assert.Empty(t, out.Conversations)
	assert.Equal(t, int64(0), out.Total)
}

func TestListConversations_RepoError(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}

	userID := uuid.New()
	convRepo.On("FindByUserID", mock.Anything, userID, mock.AnythingOfType("repository.ConversationFilter")).
		Return(nil, errors.New("db error"))

	uc := chat.NewListConversationsUseCase(convRepo)
	_, err := uc.Execute(context.Background(), userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list conversations")
}

// ---- ApproveAction ----

func TestApproveAction_Approve_Success(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}

	userID := uuid.New()
	conv := entity.NewConversation(userID, "general")
	conv.RequestApproval("tool_node", map[string]any{"tool": "web_search"})

	convRepo.On("FindByID", mock.Anything, conv.ID).Return(conv, nil)
	convRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *entity.Conversation) bool {
		return c.Status == entity.ConversationStatusActive
	})).Return(nil)

	uc := chat.NewApproveActionUseCase(convRepo)
	out, err := uc.Execute(context.Background(), chat.ApproveInput{
		ConversationID: conv.ID,
		UserID:         userID,
		Approved:       true,
	})

	require.NoError(t, err)
	assert.Equal(t, "approved", out.Status)
	assert.Equal(t, entity.ConversationStatusActive, out.Conversation.Status)
}

func TestApproveAction_Reject_Success(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}

	userID := uuid.New()
	conv := entity.NewConversation(userID, "general")
	conv.RequestApproval("tool_node", map[string]any{"tool": "dangerous_tool"})

	convRepo.On("FindByID", mock.Anything, conv.ID).Return(conv, nil)
	convRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	uc := chat.NewApproveActionUseCase(convRepo)
	out, err := uc.Execute(context.Background(), chat.ApproveInput{
		ConversationID: conv.ID,
		UserID:         userID,
		Approved:       false,
		Reason:         "Too risky",
	})

	require.NoError(t, err)
	assert.Equal(t, "rejected", out.Status)
}

func TestApproveAction_NotPendingApproval(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}

	userID := uuid.New()
	conv := entity.NewConversation(userID, "general") // status = active, not pending

	convRepo.On("FindByID", mock.Anything, conv.ID).Return(conv, nil)

	uc := chat.NewApproveActionUseCase(convRepo)
	_, err := uc.Execute(context.Background(), chat.ApproveInput{
		ConversationID: conv.ID,
		UserID:         userID,
		Approved:       true,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not pending approval")
}

func TestApproveAction_WrongUser(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}

	ownerID := uuid.New()
	conv := entity.NewConversation(ownerID, "general")
	conv.RequestApproval("node", nil)

	convRepo.On("FindByID", mock.Anything, conv.ID).Return(conv, nil)

	uc := chat.NewApproveActionUseCase(convRepo)
	_, err := uc.Execute(context.Background(), chat.ApproveInput{
		ConversationID: conv.ID,
		UserID:         uuid.New(), // wrong user
		Approved:       true,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestApproveAction_ConversationNotFound(t *testing.T) {
	convRepo := &mocks.MockConversationRepository{}

	convID := uuid.New()
	convRepo.On("FindByID", mock.Anything, convID).Return(nil, errors.New("not found"))

	uc := chat.NewApproveActionUseCase(convRepo)
	_, err := uc.Execute(context.Background(), chat.ApproveInput{
		ConversationID: convID,
		UserID:         uuid.New(),
		Approved:       true,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conversation not found")
}
