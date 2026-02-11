package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
)

type ApproveInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Approved       bool
	Reason         string
}

type ApproveOutput struct {
	Conversation *entity.Conversation `json:"conversation"`
	Status       string               `json:"status"`
}

type ApproveActionUseCase struct {
	convRepo repository.ConversationRepository
}

func NewApproveActionUseCase(convRepo repository.ConversationRepository) *ApproveActionUseCase {
	return &ApproveActionUseCase{convRepo: convRepo}
}

func (uc *ApproveActionUseCase) Execute(ctx context.Context, input ApproveInput) (*ApproveOutput, error) {
	conv, err := uc.convRepo.FindByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	if conv.UserID != input.UserID {
		return nil, fmt.Errorf("unauthorized access to conversation")
	}

	if !conv.IsPendingApproval() {
		return nil, fmt.Errorf("conversation is not pending approval")
	}

	if input.Approved {
		conv.Approve()
	} else {
		conv.Reject(input.Reason)
	}

	if err := uc.convRepo.Update(ctx, conv); err != nil {
		return nil, fmt.Errorf("update conversation: %w", err)
	}

	status := "rejected"
	if input.Approved {
		status = "approved"
	}

	return &ApproveOutput{
		Conversation: conv,
		Status:       status,
	}, nil
}
