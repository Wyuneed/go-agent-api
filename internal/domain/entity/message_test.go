package entity_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

func TestNewUserMessage(t *testing.T) {
	convID := uuid.New()
	msg := entity.NewUserMessage(convID, "Hello, world!")

	assert.Equal(t, convID, msg.ConversationID)
	assert.Equal(t, entity.RoleUser, msg.Role)
	assert.Equal(t, "Hello, world!", msg.Content)
	assert.False(t, msg.CreatedAt.IsZero())
}

func TestNewAssistantMessage(t *testing.T) {
	convID := uuid.New()
	toolCalls := []entity.ToolCall{
		{ID: "call_1", Type: "function"},
	}
	msg := entity.NewAssistantMessage(convID, "I'll help you.", toolCalls, "gpt-4o-mini")

	assert.Equal(t, entity.RoleAssistant, msg.Role)
	assert.Equal(t, "I'll help you.", msg.Content)
	assert.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "gpt-4o-mini", msg.Model)
}

func TestNewSystemMessage(t *testing.T) {
	msg := entity.NewSystemMessage(uuid.New(), "You are a helpful assistant.")

	assert.Equal(t, entity.RoleSystem, msg.Role)
	assert.Equal(t, "You are a helpful assistant.", msg.Content)
}

func TestNewToolMessage(t *testing.T) {
	result := map[string]string{"answer": "42"}
	msg := entity.NewToolMessage(uuid.New(), "call_123", "calculator", result)

	assert.Equal(t, entity.RoleTool, msg.Role)
	assert.Equal(t, "calculator", msg.Name)
	assert.Equal(t, "call_123", msg.ToolCallID)

	// Content should be JSON
	var decoded map[string]string
	require.NoError(t, json.Unmarshal([]byte(msg.Content), &decoded))
	assert.Equal(t, "42", decoded["answer"])
}

func TestMessage_ToOpenAIFormat(t *testing.T) {
	convID := uuid.New()

	t.Run("user message", func(t *testing.T) {
		msg := entity.NewUserMessage(convID, "Hello")
		format := msg.ToOpenAIFormat()
		assert.Equal(t, "user", format["role"])
		assert.Equal(t, "Hello", format["content"])
		assert.Nil(t, format["tool_calls"])
	})

	t.Run("assistant with tool calls", func(t *testing.T) {
		toolCalls := []entity.ToolCall{{ID: "tc_1", Type: "function"}}
		msg := entity.NewAssistantMessage(convID, "", toolCalls, "gpt-4o")
		format := msg.ToOpenAIFormat()
		assert.Equal(t, "assistant", format["role"])
		assert.NotNil(t, format["tool_calls"])
	})

	t.Run("tool message", func(t *testing.T) {
		msg := entity.NewToolMessage(convID, "call_1", "calc", "result")
		format := msg.ToOpenAIFormat()
		assert.Equal(t, "tool", format["role"])
		assert.Equal(t, "call_1", format["tool_call_id"])
		assert.Equal(t, "calc", format["name"])
	})
}
