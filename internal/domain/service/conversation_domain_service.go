package service

import (
	"fmt"

	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

// ConversationDomainService encapsulates business rules
// that span across Conversation and Message entities.
type ConversationDomainService struct{}

func NewConversationDomainService() *ConversationDomainService {
	return &ConversationDomainService{}
}

// CanSendMessage checks if a new message can be sent to the conversation.
func (s *ConversationDomainService) CanSendMessage(conv *entity.Conversation, user *entity.User) error {
	if !user.IsActive {
		return fmt.Errorf("user account is inactive")
	}

	if conv.UserID != user.ID && !user.IsAdmin() {
		return fmt.Errorf("user does not own this conversation")
	}

	if !conv.IsActive() {
		if conv.IsPendingApproval() {
			return fmt.Errorf("conversation is pending approval, cannot send new messages")
		}
		return fmt.Errorf("conversation is not active (status: %s)", conv.Status)
	}

	return nil
}

// CanApprove checks whether a user can approve a pending action.
func (s *ConversationDomainService) CanApprove(conv *entity.Conversation, user *entity.User) error {
	if !conv.IsPendingApproval() {
		return fmt.Errorf("conversation is not pending approval")
	}

	if conv.UserID != user.ID && !user.IsAdmin() {
		return fmt.Errorf("user is not authorized to approve this action")
	}

	return nil
}

// GenerateTitle produces a default title from the first user message content.
func (s *ConversationDomainService) GenerateTitle(content string) string {
	const maxLen = 100
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}
