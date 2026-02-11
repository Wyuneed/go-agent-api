package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
)

type GetConversationOutput struct {
	Conversation *entity.Conversation `json:"conversation"`
	Messages     []*entity.Message    `json:"messages"`
}

type GetConversationUseCase struct {
	convRepo repository.ConversationRepository
	msgRepo  repository.MessageRepository
}

func NewGetConversationUseCase(convRepo repository.ConversationRepository, msgRepo repository.MessageRepository) *GetConversationUseCase {
	return &GetConversationUseCase{
		convRepo: convRepo,
		msgRepo:  msgRepo,
	}
}

func (uc *GetConversationUseCase) Execute(ctx context.Context, convID uuid.UUID, userID uuid.UUID) (*GetConversationOutput, error) {
	conv, err := uc.convRepo.FindByID(ctx, convID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	if conv.UserID != userID {
		return nil, fmt.Errorf("unauthorized access to conversation")
	}

	messages, err := uc.msgRepo.FindByConversationID(ctx, convID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	return &GetConversationOutput{
		Conversation: conv,
		Messages:     messages,
	}, nil
}
