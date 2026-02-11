package event

import (
	"time"

	"github.com/google/uuid"
)

const ToolExecutedType = "tool.executed"

type ToolExecuted struct {
	baseEvent
	ConversationID uuid.UUID
	UserID         uuid.UUID
	ToolName       string
	ToolCallID     string
	Success        bool
	Duration       time.Duration
	ErrorMessage   string
}

func NewToolExecuted(conversationID, userID uuid.UUID, toolName, toolCallID string, success bool, duration time.Duration, errMsg string) ToolExecuted {
	return ToolExecuted{
		baseEvent:      newBaseEvent(ToolExecutedType),
		ConversationID: conversationID,
		UserID:         userID,
		ToolName:       toolName,
		ToolCallID:     toolCallID,
		Success:        success,
		Duration:       duration,
		ErrorMessage:   errMsg,
	}
}
