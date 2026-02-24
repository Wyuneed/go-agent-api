package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

type ToolExecutionRepository interface {
	Create(ctx context.Context, exec *entity.ToolExecution) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.ToolExecution, error)
	FindByConversationID(ctx context.Context, convID uuid.UUID) ([]*entity.ToolExecution, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ToolExecutionStatus, output map[string]any, errMsg string, durationMs *int) error
}
