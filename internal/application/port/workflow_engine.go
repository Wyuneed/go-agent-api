package port

import (
	"context"

	"github.com/google/uuid"
)

// WorkflowInput contains everything needed to start a workflow run.
type WorkflowInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Content        string
	Model          string
	Messages       []ChatMessage
	Temperature    float64
}

// WorkflowOutput is the result of a completed (or paused) workflow run.
type WorkflowOutput struct {
	Response         string
	TokensUsed       int
	ToolCallsCount   int
	AgentType        string
	RequiresApproval bool
	ApprovalReason   string
	PausedStateJSON  []byte // serialised AgentState; non-nil when RequiresApproval is true
}

// WorkflowEngine is the application-layer interface for the Eino workflow engine.
// It is implemented by infrastructure/eino/graphs.ChatbotGraph.
type WorkflowEngine interface {
	// Run executes the workflow for a new user turn.
	Run(ctx context.Context, input WorkflowInput) (*WorkflowOutput, error)
	// Resume continues a workflow that was paused for human approval.
	Resume(ctx context.Context, pausedStateJSON []byte, approved bool) (*WorkflowOutput, error)
}
