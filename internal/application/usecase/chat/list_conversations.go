package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
)

type ListConversationsOutput struct {
	Conversations []*entity.Conversation `json:"conversations"`
	Total         int64                  `json:"total"`
}

type ListConversationsUseCase struct {
	convRepo repository.ConversationRepository
}

func NewListConversationsUseCase(convRepo repository.ConversationRepository) *ListConversationsUseCase {
	return &ListConversationsUseCase{convRepo: convRepo}
}

func (uc *ListConversationsUseCase) Execute(ctx context.Context, userID uuid.UUID) (*ListConversationsOutput, error) {
	conversations, err := uc.convRepo.FindByUserID(ctx, userID, repository.ConversationFilter{
		Limit:   50,
		OrderBy: "updated_at",
		Order:   "DESC",
	})
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	total, err := uc.convRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count conversations: %w", err)
	}

	return &ListConversationsOutput{
		Conversations: conversations,
		Total:         total,
	}, nil
}
