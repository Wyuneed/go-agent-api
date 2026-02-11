package event

import (
	"github.com/google/uuid"
)

const ConversationCompletedType = "conversation.completed"

type ConversationCompleted struct {
	baseEvent
	ConversationID uuid.UUID
	UserID         uuid.UUID
	MessageCount   int
	ToolCallCount  int
	TotalTokens    int
	TotalCost      float64
}

func NewConversationCompleted(conversationID, userID uuid.UUID, messageCount, toolCallCount, totalTokens int, totalCost float64) ConversationCompleted {
	return ConversationCompleted{
		baseEvent:      newBaseEvent(ConversationCompletedType),
		ConversationID: conversationID,
		UserID:         userID,
		MessageCount:   messageCount,
		ToolCallCount:  toolCallCount,
		TotalTokens:    totalTokens,
		TotalCost:      totalCost,
	}
}
