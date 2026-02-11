package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

type TokenRepository struct {
	pool *pgxpool.Pool
}

func NewTokenRepository(pool *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{pool: pool}
}

func (r *TokenRepository) Create(ctx context.Context, token *entity.Token) error {
	query := `
		INSERT INTO user_tokens (id, user_id, token_hash, token_type, name, expires_at,
			rate_limit_per_minute, rate_limit_per_day, allowed_tools, allowed_models,
			is_revoked, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.pool.Exec(ctx, query,
		token.ID, token.UserID, token.TokenHash, string(token.TokenType),
		token.Name, token.ExpiresAt, token.RateLimitPerMinute, token.RateLimitPerDay,
		token.AllowedTools, token.AllowedModels, token.IsRevoked,
		token.Metadata, token.CreatedAt, token.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create token: %w", err)
	}
	return nil
}

func (r *TokenRepository) FindByHash(ctx context.Context, tokenHash string) (*entity.Token, error) {
	query := `
		SELECT id, user_id, token_hash, token_type, name, expires_at, last_used_at,
			rate_limit_per_minute, rate_limit_per_day, allowed_tools, allowed_models,
			is_revoked, metadata, created_at, updated_at
		FROM user_tokens WHERE token_hash = $1`

	return r.scanToken(ctx, query, tokenHash)
}

func (r *TokenRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Token, error) {
	query := `
		SELECT id, user_id, token_hash, token_type, name, expires_at, last_used_at,
			rate_limit_per_minute, rate_limit_per_day, allowed_tools, allowed_models,
			is_revoked, metadata, created_at, updated_at
		FROM user_tokens WHERE id = $1`

	return r.scanToken(ctx, query, id)
}

func (r *TokenRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Token, error) {
	query := `
		SELECT id, user_id, token_hash, token_type, name, expires_at, last_used_at,
			rate_limit_per_minute, rate_limit_per_day, allowed_tools, allowed_models,
			is_revoked, metadata, created_at, updated_at
		FROM user_tokens WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("find tokens by user id: %w", err)
	}
	defer rows.Close()

	var tokens []*entity.Token
	for rows.Next() {
		token, err := r.scanTokenFromRow(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (r *TokenRepository) UpdateLastUsed(ctx context.Context, tokenID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE user_tokens SET last_used_at = $2 WHERE id = $1",
		tokenID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}
	return nil
}

func (r *TokenRepository) Revoke(ctx context.Context, tokenID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE user_tokens SET is_revoked = true WHERE id = $1",
		tokenID,
	)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}

func (r *TokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE user_tokens SET is_revoked = true WHERE user_id = $1",
		userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all tokens for user: %w", err)
	}
	return nil
}

func (r *TokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.pool.Exec(ctx,
		"DELETE FROM user_tokens WHERE expires_at < NOW()",
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired tokens: %w", err)
	}
	return result.RowsAffected(), nil
}

func (r *TokenRepository) scanToken(ctx context.Context, query string, arg any) (*entity.Token, error) {
	token := &entity.Token{}
	var tokenType string
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &tokenType,
		&token.Name, &token.ExpiresAt, &token.LastUsedAt,
		&token.RateLimitPerMinute, &token.RateLimitPerDay,
		&token.AllowedTools, &token.AllowedModels,
		&token.IsRevoked, &token.Metadata, &token.CreatedAt, &token.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("scan token: %w", err)
	}
	token.TokenType = entity.TokenType(tokenType)
	return token, nil
}

func (r *TokenRepository) scanTokenFromRow(rows pgx.Rows) (*entity.Token, error) {
	token := &entity.Token{}
	var tokenType string
	err := rows.Scan(
		&token.ID, &token.UserID, &token.TokenHash, &tokenType,
		&token.Name, &token.ExpiresAt, &token.LastUsedAt,
		&token.RateLimitPerMinute, &token.RateLimitPerDay,
		&token.AllowedTools, &token.AllowedModels,
		&token.IsRevoked, &token.Metadata, &token.CreatedAt, &token.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan token row: %w", err)
	}
	token.TokenType = entity.TokenType(tokenType)
	return token, nil
}
