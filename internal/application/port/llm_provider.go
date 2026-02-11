package port

import (
	"context"

	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

// ChatMessage represents a message in the conversation.
type ChatMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	Name       string            `json:"name,omitempty"`
	ToolCalls  []toolspec.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model       string          `json:"model"`
	Messages    []ChatMessage   `json:"messages"`
	Tools       []toolspec.Tool `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	User        string          `json:"user,omitempty"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   UsageInfo `json:"usage"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// UsageInfo contains token usage information.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamEvent represents a streaming event.
type StreamEvent struct {
	Type     string            `json:"type"`
	Delta    string            `json:"delta,omitempty"`
	ToolCall *toolspec.ToolCall `json:"tool_call,omitempty"`
	Usage    *UsageInfo        `json:"usage,omitempty"`
	Error    error             `json:"-"`
	Done     bool              `json:"done,omitempty"`
}

// LLMProvider is the interface for LLM providers.
type LLMProvider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
}
