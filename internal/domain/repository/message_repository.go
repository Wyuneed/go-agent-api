package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *entity.Message) error
	CreateBatch(ctx context.Context, msgs []*entity.Message) error
	FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*entity.Message, error)
	FindByConversationIDPaginated(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*entity.Message, error)
	FindLastN(ctx context.Context, conversationID uuid.UUID, n int) ([]*entity.Message, error)
	CountByConversationID(ctx context.Context, conversationID uuid.UUID) (int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByConversationID(ctx context.Context, conversationID uuid.UUID) error
}
