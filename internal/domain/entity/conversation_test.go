package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

func TestNewConversation(t *testing.T) {
	userID := uuid.New()
	conv := entity.NewConversation(userID, "general")

	assert.NotNil(t, conv)
	assert.Equal(t, userID, conv.UserID)
	assert.Equal(t, entity.ConversationStatusActive, conv.Status)
	assert.Equal(t, "general", conv.AgentType)
	assert.Equal(t, 0.7, conv.Temperature)
	assert.NotNil(t, conv.Metadata)
	assert.NotNil(t, conv.WorkflowState)
	assert.False(t, conv.CreatedAt.IsZero())
}

func TestConversation_IsActive(t *testing.T) {
	conv := entity.NewConversation(uuid.New(), "general")
	assert.True(t, conv.IsActive())

	conv.Status = entity.ConversationStatusCompleted
	assert.False(t, conv.IsActive())
}

func TestConversation_IsPendingApproval(t *testing.T) {
	conv := entity.NewConversation(uuid.New(), "general")
	assert.False(t, conv.IsPendingApproval())

	conv.Status = entity.ConversationStatusPending
	assert.True(t, conv.IsPendingApproval())
}

func TestConversation_RequestApproval(t *testing.T) {
	conv := entity.NewConversation(uuid.New(), "general")
	data := map[string]any{"tool": "dangerous_tool"}

	conv.RequestApproval("act", data)

	assert.Equal(t, entity.ConversationStatusPending, conv.Status)
	assert.Equal(t, "act", conv.CurrentNode)
	assert.NotNil(t, conv.Metadata["pending_approval"])
}

func TestConversation_Approve(t *testing.T) {
	conv := entity.NewConversation(uuid.New(), "general")
	conv.RequestApproval("act", map[string]any{"tool": "test"})
	conv.Approve()

	assert.Equal(t, entity.ConversationStatusActive, conv.Status)
	assert.Nil(t, conv.Metadata["pending_approval"])
}

func TestConversation_Reject(t *testing.T) {
	conv := entity.NewConversation(uuid.New(), "general")
	conv.RequestApproval("act", map[string]any{"tool": "test"})
	conv.Reject("too risky")

	assert.Equal(t, entity.ConversationStatusActive, conv.Status)
	assert.NotNil(t, conv.Metadata["last_rejection"])
}

func TestConversation_Complete(t *testing.T) {
	conv := entity.NewConversation(uuid.New(), "general")
	conv.Complete()

	assert.Equal(t, entity.ConversationStatusCompleted, conv.Status)
	assert.NotNil(t, conv.CompletedAt)
}

func TestConversation_AddMessageStats(t *testing.T) {
	conv := entity.NewConversation(uuid.New(), "general")
	conv.AddMessageStats(100, 1)
	conv.AddMessageStats(200, 2)

	assert.Equal(t, 2, conv.MessageCount)
	assert.Equal(t, 300, conv.TotalTokens)
	assert.Equal(t, 3, conv.ToolCallCount)
	assert.NotNil(t, conv.LastMessageAt)
}

func TestConversation_SetTitle(t *testing.T) {
	conv := entity.NewConversation(uuid.New(), "general")
	assert.Empty(t, conv.Title)

	conv.SetTitle("My First Chat")
	assert.Equal(t, "My First Chat", conv.Title)

	// Should not overwrite existing title
	conv.SetTitle("New Title")
	assert.Equal(t, "My First Chat", conv.Title)
}
