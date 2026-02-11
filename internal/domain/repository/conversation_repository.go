package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

type ConversationFilter struct {
	UserID  *uuid.UUID
	Status  *entity.ConversationStatus
	Limit   int
	Offset  int
	OrderBy string // "created_at", "updated_at", "last_message_at"
	Order   string // "asc", "desc"
}

type ConversationRepository interface {
	Create(ctx context.Context, conv *entity.Conversation) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Conversation, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, filter ConversationFilter) ([]*entity.Conversation, error)
	Update(ctx context.Context, conv *entity.Conversation) error
	UpdateWorkflowState(ctx context.Context, id uuid.UUID, state map[string]any) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}
