package event

import (
	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

const MessageCreatedType = "message.created"

type MessageCreated struct {
	baseEvent
	ConversationID uuid.UUID
	MessageID      uuid.UUID
	Role           entity.MessageRole
	HasToolCalls   bool
}

func NewMessageCreated(conversationID, messageID uuid.UUID, role entity.MessageRole, hasToolCalls bool) MessageCreated {
	return MessageCreated{
		baseEvent:      newBaseEvent(MessageCreatedType),
		ConversationID: conversationID,
		MessageID:      messageID,
		Role:           role,
		HasToolCalls:   hasToolCalls,
	}
}
