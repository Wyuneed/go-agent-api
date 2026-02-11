package entity

import (
	"time"

	"github.com/google/uuid"
)

type ConversationStatus string

const (
	ConversationStatusActive    ConversationStatus = "active"
	ConversationStatusPending   ConversationStatus = "pending_approval"
	ConversationStatusCompleted ConversationStatus = "completed"
	ConversationStatusFailed    ConversationStatus = "failed"
	ConversationStatusArchived  ConversationStatus = "archived"
)

// Conversation is the aggregate root for chat sessions.
type Conversation struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Status      ConversationStatus

	// Agent configuration
	AgentType    string
	SystemPrompt string
	Model        string
	Temperature  float64

	// Workflow state (for Eino)
	CurrentNode   string
	WorkflowState map[string]any

	// Counters
	MessageCount  int
	ToolCallCount int
	TotalTokens   int
	TotalCost     float64

	// Metadata
	Metadata map[string]any
	Tags     []string

	// Timestamps
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastMessageAt *time.Time
	CompletedAt   *time.Time
}

func NewConversation(userID uuid.UUID, agentType string) *Conversation {
	return &Conversation{
		ID:            uuid.New(),
		UserID:        userID,
		Status:        ConversationStatusActive,
		AgentType:     agentType,
		Temperature:   0.7,
		Metadata:      make(map[string]any),
		WorkflowState: make(map[string]any),
		Tags:          []string{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func (c *Conversation) IsActive() bool {
	return c.Status == ConversationStatusActive
}

func (c *Conversation) IsPendingApproval() bool {
	return c.Status == ConversationStatusPending
}

func (c *Conversation) RequestApproval(node string, data map[string]any) {
	c.Status = ConversationStatusPending
	c.CurrentNode = node
	c.Metadata["pending_approval"] = data
	c.UpdatedAt = time.Now()
}

func (c *Conversation) Approve() {
	c.Status = ConversationStatusActive
	delete(c.Metadata, "pending_approval")
	c.UpdatedAt = time.Now()
}

func (c *Conversation) Reject(reason string) {
	c.Status = ConversationStatusActive
	c.Metadata["last_rejection"] = map[string]any{
		"reason": reason,
		"at":     time.Now(),
	}
	c.UpdatedAt = time.Now()
}

func (c *Conversation) Complete() {
	c.Status = ConversationStatusCompleted
	now := time.Now()
	c.CompletedAt = &now
	c.UpdatedAt = now
}

func (c *Conversation) AddMessageStats(tokens int, toolCalls int) {
	c.MessageCount++
	c.TotalTokens += tokens
	c.ToolCallCount += toolCalls
	now := time.Now()
	c.LastMessageAt = &now
	c.UpdatedAt = now
}

func (c *Conversation) SetTitle(title string) {
	if c.Title == "" && title != "" {
		c.Title = title
		c.UpdatedAt = time.Now()
	}
}
