package entity

import (
	"time"

	"github.com/google/uuid"
)

type ToolExecutionStatus string

const (
	ToolExecStatusPending          ToolExecutionStatus = "pending"
	ToolExecStatusRunning          ToolExecutionStatus = "running"
	ToolExecStatusSuccess          ToolExecutionStatus = "success"
	ToolExecStatusFailed           ToolExecutionStatus = "failed"
	ToolExecStatusCancelled        ToolExecutionStatus = "cancelled"
	ToolExecStatusRequiresApproval ToolExecutionStatus = "requires_approval"
)

type ToolExecution struct {
	ID             uuid.UUID
	ConversationID *uuid.UUID
	MessageID      *uuid.UUID
	UserID         uuid.UUID
	TokenID        *uuid.UUID

	ToolName   string
	ToolCallID string
	Input      map[string]any
	Output     map[string]any

	Status       ToolExecutionStatus
	ErrorMessage string

	DurationMs  *int
	QueuedAt    time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time

	RequiresApproval bool
	ApprovedBy       *uuid.UUID
	ApprovedAt       *time.Time

	CreatedAt time.Time
}

func NewToolExecution(userID uuid.UUID, toolName, toolCallID string, input map[string]any) *ToolExecution {
	now := time.Now()
	return &ToolExecution{
		ID:         uuid.New(),
		UserID:     userID,
		ToolName:   toolName,
		ToolCallID: toolCallID,
		Input:      input,
		Status:     ToolExecStatusPending,
		QueuedAt:   now,
		CreatedAt:  now,
	}
}

func (e *ToolExecution) MarkRunning() {
	now := time.Now()
	e.Status = ToolExecStatusRunning
	e.StartedAt = &now
}

func (e *ToolExecution) MarkSuccess(output map[string]any, durationMs int) {
	now := time.Now()
	e.Status = ToolExecStatusSuccess
	e.Output = output
	e.DurationMs = &durationMs
	e.CompletedAt = &now
}

func (e *ToolExecution) MarkFailed(errMsg string, durationMs int) {
	now := time.Now()
	e.Status = ToolExecStatusFailed
	e.ErrorMessage = errMsg
	e.DurationMs = &durationMs
	e.CompletedAt = &now
}
