package request

import "github.com/wyuneed/go-agent-api/internal/application/port"

type ChatCompletionRequest struct {
	Model       string            `json:"model"`
	Messages    []port.ChatMessage `json:"messages"`
	Tools       []any             `json:"tools,omitempty"`
	ToolChoice  any               `json:"tool_choice,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
	MaxTokens   *int              `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
}

type CreateConversationRequest struct {
	Message   string `json:"message"`
	AgentType string `json:"agent_type,omitempty"`
}

type SendMessageRequest struct {
	Message string `json:"message"`
}

type ApprovalRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}
