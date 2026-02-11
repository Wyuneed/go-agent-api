package state

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/application/port"
	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

// AgentState is passed through the Eino workflow graph.
type AgentState struct {
	// Identifiers
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	RequestID      string    `json:"request_id"`

	// Messages
	Messages     []port.ChatMessage `json:"messages"`
	SystemPrompt string             `json:"system_prompt"`

	// Current turn
	UserInput     string `json:"user_input"`
	CurrentOutput string `json:"current_output"`

	// Model configuration
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`

	// Tool execution
	AvailableTools []toolspec.Tool     `json:"available_tools"`
	PendingTools   []toolspec.ToolCall `json:"pending_tools"`
	ToolResults    []port.ChatMessage  `json:"tool_results"`

	// Multi-agent routing
	CurrentAgent string   `json:"current_agent"`
	NextAgent    string   `json:"next_agent"`
	AgentHistory []string `json:"agent_history"`

	// Human-in-the-loop
	RequiresApproval bool           `json:"requires_approval"`
	ApprovalReason   string         `json:"approval_reason"`
	ApprovalData     map[string]any `json:"approval_data"`
	IsApproved       *bool          `json:"is_approved,omitempty"`

	// Control flow
	Iteration     int    `json:"iteration"`
	MaxIterations int    `json:"max_iterations"`
	ShouldStop    bool   `json:"should_stop"`
	Error         string `json:"error,omitempty"`

	// Metrics
	StartTime      time.Time `json:"start_time"`
	TokensUsed     int       `json:"tokens_used"`
	ToolCallsCount int       `json:"tool_calls_count"`
}

func NewAgentState(convID, userID uuid.UUID, input string) *AgentState {
	return &AgentState{
		ConversationID: convID,
		UserID:         userID,
		RequestID:      uuid.New().String(),
		UserInput:      input,
		Messages:       []port.ChatMessage{},
		AvailableTools: []toolspec.Tool{},
		PendingTools:   []toolspec.ToolCall{},
		ToolResults:    []port.ChatMessage{},
		AgentHistory:   []string{},
		ApprovalData:   make(map[string]any),
		Model:          "gpt-4o-mini",
		Temperature:    0.7,
		MaxTokens:      4096,
		MaxIterations:  10,
		Iteration:      0,
		StartTime:      time.Now(),
	}
}

func (s *AgentState) AddMessage(msg port.ChatMessage) {
	s.Messages = append(s.Messages, msg)
}

func (s *AgentState) AddUserMessage(content string) {
	s.Messages = append(s.Messages, port.ChatMessage{
		Role:    "user",
		Content: content,
	})
}

func (s *AgentState) AddAssistantMessage(content string, toolCalls []toolspec.ToolCall) {
	msg := port.ChatMessage{
		Role:    "assistant",
		Content: content,
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}
	s.Messages = append(s.Messages, msg)
}

func (s *AgentState) AddToolResult(toolCallID, name string, result any) {
	content, _ := json.Marshal(result)
	s.ToolResults = append(s.ToolResults, port.ChatMessage{
		Role:       "tool",
		Content:    string(content),
		ToolCallID: toolCallID,
		Name:       name,
	})
}

func (s *AgentState) FlushToolResults() {
	s.Messages = append(s.Messages, s.ToolResults...)
	s.ToolResults = []port.ChatMessage{}
}

func (s *AgentState) Clone() *AgentState {
	clone := *s
	clone.Messages = make([]port.ChatMessage, len(s.Messages))
	copy(clone.Messages, s.Messages)
	clone.PendingTools = make([]toolspec.ToolCall, len(s.PendingTools))
	copy(clone.PendingTools, s.PendingTools)
	clone.AgentHistory = make([]string, len(s.AgentHistory))
	copy(clone.AgentHistory, s.AgentHistory)
	return &clone
}

// ToJSON serializes state for persistence.
func (s *AgentState) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON deserializes state.
func FromJSON(data []byte) (*AgentState, error) {
	var s AgentState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
