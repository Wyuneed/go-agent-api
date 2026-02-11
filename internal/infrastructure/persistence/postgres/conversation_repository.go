package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
)

type ConversationRepository struct {
	pool *pgxpool.Pool
}

func NewConversationRepository(pool *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{pool: pool}
}

func (r *ConversationRepository) Create(ctx context.Context, conv *entity.Conversation) error {
	query := `
		INSERT INTO conversations (id, user_id, title, status, agent_type, system_prompt,
			model, temperature, current_node, workflow_state, message_count, tool_call_count,
			total_tokens, total_cost, metadata, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`

	_, err := r.pool.Exec(ctx, query,
		conv.ID, conv.UserID, conv.Title, string(conv.Status),
		conv.AgentType, conv.SystemPrompt, conv.Model, conv.Temperature,
		conv.CurrentNode, conv.WorkflowState,
		conv.MessageCount, conv.ToolCallCount, conv.TotalTokens, conv.TotalCost,
		conv.Metadata, conv.Tags, conv.CreatedAt, conv.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Conversation, error) {
	query := `
		SELECT id, user_id, title, status, agent_type, system_prompt, model, temperature,
			current_node, workflow_state, message_count, tool_call_count, total_tokens, total_cost,
			metadata, tags, created_at, updated_at, last_message_at, completed_at
		FROM conversations WHERE id = $1 AND deleted_at IS NULL`

	conv := &entity.Conversation{}
	var status string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.UserID, &conv.Title, &status,
		&conv.AgentType, &conv.SystemPrompt, &conv.Model, &conv.Temperature,
		&conv.CurrentNode, &conv.WorkflowState,
		&conv.MessageCount, &conv.ToolCallCount, &conv.TotalTokens, &conv.TotalCost,
		&conv.Metadata, &conv.Tags, &conv.CreatedAt, &conv.UpdatedAt,
		&conv.LastMessageAt, &conv.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, fmt.Errorf("find conversation by id: %w", err)
	}
	conv.Status = entity.ConversationStatus(status)
	return conv, nil
}

func (r *ConversationRepository) FindByUserID(ctx context.Context, userID uuid.UUID, filter repository.ConversationFilter) ([]*entity.Conversation, error) {
	query := `
		SELECT id, user_id, title, status, agent_type, system_prompt, model, temperature,
			current_node, workflow_state, message_count, tool_call_count, total_tokens, total_cost,
			metadata, tags, created_at, updated_at, last_message_at, completed_at
		FROM conversations
		WHERE user_id = $1 AND deleted_at IS NULL`

	args := []any{userID}
	argIdx := 2

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}

	orderBy := "updated_at"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	order := "DESC"
	if filter.Order != "" {
		order = filter.Order
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, order)

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find conversations by user id: %w", err)
	}
	defer rows.Close()

	var conversations []*entity.Conversation
	for rows.Next() {
		conv := &entity.Conversation{}
		var status string
		err := rows.Scan(
			&conv.ID, &conv.UserID, &conv.Title, &status,
			&conv.AgentType, &conv.SystemPrompt, &conv.Model, &conv.Temperature,
			&conv.CurrentNode, &conv.WorkflowState,
			&conv.MessageCount, &conv.ToolCallCount, &conv.TotalTokens, &conv.TotalCost,
			&conv.Metadata, &conv.Tags, &conv.CreatedAt, &conv.UpdatedAt,
			&conv.LastMessageAt, &conv.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan conversation row: %w", err)
		}
		conv.Status = entity.ConversationStatus(status)
		conversations = append(conversations, conv)
	}
	return conversations, rows.Err()
}

func (r *ConversationRepository) Update(ctx context.Context, conv *entity.Conversation) error {
	query := `
		UPDATE conversations
		SET title = $2, status = $3, agent_type = $4, system_prompt = $5,
			model = $6, temperature = $7, current_node = $8, workflow_state = $9,
			message_count = $10, tool_call_count = $11, total_tokens = $12, total_cost = $13,
			metadata = $14, tags = $15, last_message_at = $16, completed_at = $17
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query,
		conv.ID, conv.Title, string(conv.Status),
		conv.AgentType, conv.SystemPrompt, conv.Model, conv.Temperature,
		conv.CurrentNode, conv.WorkflowState,
		conv.MessageCount, conv.ToolCallCount, conv.TotalTokens, conv.TotalCost,
		conv.Metadata, conv.Tags, conv.LastMessageAt, conv.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("update conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepository) UpdateWorkflowState(ctx context.Context, id uuid.UUID, state map[string]any) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE conversations SET workflow_state = $2 WHERE id = $1",
		id, state,
	)
	if err != nil {
		return fmt.Errorf("update workflow state: %w", err)
	}
	return nil
}

func (r *ConversationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Soft delete
	_, err := r.pool.Exec(ctx,
		"UPDATE conversations SET deleted_at = NOW() WHERE id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM conversations WHERE user_id = $1 AND deleted_at IS NULL",
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count conversations: %w", err)
	}
	return count, nil
}
