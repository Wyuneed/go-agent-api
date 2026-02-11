package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// ToolCall represents a tool call from the assistant (OpenAI-compatible format).
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	Role           MessageRole
	Content        string
	Name           string

	// Tool-related
	ToolCalls  []ToolCall
	ToolCallID string

	// Metadata
	Model            string
	PromptTokens     int
	CompletionTokens int
	Latency          time.Duration

	// Streaming
	IsStreaming     bool
	StreamComplete  bool

	// Ordering
	SequenceNumber int

	CreatedAt time.Time
}

func NewUserMessage(conversationID uuid.UUID, content string) *Message {
	return &Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           RoleUser,
		Content:        content,
		CreatedAt:      time.Now(),
	}
}

func NewAssistantMessage(conversationID uuid.UUID, content string, toolCalls []ToolCall, model string) *Message {
	return &Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           RoleAssistant,
		Content:        content,
		ToolCalls:      toolCalls,
		Model:          model,
		CreatedAt:      time.Now(),
	}
}

func NewSystemMessage(conversationID uuid.UUID, content string) *Message {
	return &Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           RoleSystem,
		Content:        content,
		CreatedAt:      time.Now(),
	}
}

func NewToolMessage(conversationID uuid.UUID, toolCallID string, name string, result any) *Message {
	content, _ := json.Marshal(result)
	return &Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		Role:           RoleTool,
		Name:           name,
		Content:        string(content),
		ToolCallID:     toolCallID,
		CreatedAt:      time.Now(),
	}
}

// ToOpenAIFormat converts to OpenAI-compatible message format.
func (m *Message) ToOpenAIFormat() map[string]any {
	msg := map[string]any{
		"role":    string(m.Role),
		"content": m.Content,
	}

	if len(m.ToolCalls) > 0 {
		msg["tool_calls"] = m.ToolCalls
	}

	if m.ToolCallID != "" {
		msg["tool_call_id"] = m.ToolCallID
	}

	if m.Name != "" {
		msg["name"] = m.Name
	}

	return msg
}
