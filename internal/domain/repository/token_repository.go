package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

type TokenRepository interface {
	Create(ctx context.Context, token *entity.Token) error
	FindByHash(ctx context.Context, tokenHash string) (*entity.Token, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Token, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Token, error)
	UpdateLastUsed(ctx context.Context, tokenID uuid.UUID) error
	Revoke(ctx context.Context, tokenID uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}
