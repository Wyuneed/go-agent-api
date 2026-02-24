package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

type ToolExecutionRepository struct {
	pool *pgxpool.Pool
}

func NewToolExecutionRepository(pool *pgxpool.Pool) *ToolExecutionRepository {
	return &ToolExecutionRepository{pool: pool}
}

func (r *ToolExecutionRepository) Create(ctx context.Context, exec *entity.ToolExecution) error {
	query := `
		INSERT INTO tool_executions (
			id, conversation_id, message_id, user_id, token_id,
			tool_name, tool_call_id, input, output,
			status, error_message, duration_ms,
			queued_at, started_at, completed_at,
			requires_approval, approved_by, approved_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15,
			$16, $17, $18, $19
		)`

	_, err := r.pool.Exec(ctx, query,
		exec.ID, exec.ConversationID, exec.MessageID, exec.UserID, exec.TokenID,
		exec.ToolName, exec.ToolCallID, exec.Input, exec.Output,
		string(exec.Status), exec.ErrorMessage, exec.DurationMs,
		exec.QueuedAt, exec.StartedAt, exec.CompletedAt,
		exec.RequiresApproval, exec.ApprovedBy, exec.ApprovedAt, exec.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create tool execution: %w", err)
	}
	return nil
}

func (r *ToolExecutionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ToolExecution, error) {
	query := `
		SELECT id, conversation_id, message_id, user_id, token_id,
			tool_name, tool_call_id, input, output,
			status, error_message, duration_ms,
			queued_at, started_at, completed_at,
			requires_approval, approved_by, approved_at, created_at
		FROM tool_executions WHERE id = $1`

	exec := &entity.ToolExecution{}
	var status string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&exec.ID, &exec.ConversationID, &exec.MessageID, &exec.UserID, &exec.TokenID,
		&exec.ToolName, &exec.ToolCallID, &exec.Input, &exec.Output,
		&status, &exec.ErrorMessage, &exec.DurationMs,
		&exec.QueuedAt, &exec.StartedAt, &exec.CompletedAt,
		&exec.RequiresApproval, &exec.ApprovedBy, &exec.ApprovedAt, &exec.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tool execution not found")
		}
		return nil, fmt.Errorf("find tool execution by id: %w", err)
	}
	exec.Status = entity.ToolExecutionStatus(status)
	return exec, nil
}

func (r *ToolExecutionRepository) FindByConversationID(ctx context.Context, convID uuid.UUID) ([]*entity.ToolExecution, error) {
	query := `
		SELECT id, conversation_id, message_id, user_id, token_id,
			tool_name, tool_call_id, input, output,
			status, error_message, duration_ms,
			queued_at, started_at, completed_at,
			requires_approval, approved_by, approved_at, created_at
		FROM tool_executions WHERE conversation_id = $1 ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, convID)
	if err != nil {
		return nil, fmt.Errorf("find tool executions by conversation id: %w", err)
	}
	defer rows.Close()

	var executions []*entity.ToolExecution
	for rows.Next() {
		exec := &entity.ToolExecution{}
		var status string
		if err := rows.Scan(
			&exec.ID, &exec.ConversationID, &exec.MessageID, &exec.UserID, &exec.TokenID,
			&exec.ToolName, &exec.ToolCallID, &exec.Input, &exec.Output,
			&status, &exec.ErrorMessage, &exec.DurationMs,
			&exec.QueuedAt, &exec.StartedAt, &exec.CompletedAt,
			&exec.RequiresApproval, &exec.ApprovedBy, &exec.ApprovedAt, &exec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tool execution row: %w", err)
		}
		exec.Status = entity.ToolExecutionStatus(status)
		executions = append(executions, exec)
	}
	return executions, rows.Err()
}

func (r *ToolExecutionRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status entity.ToolExecutionStatus,
	output map[string]any,
	errMsg string,
	durationMs *int,
) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tool_executions
		SET status = $2, output = $3, error_message = $4, duration_ms = $5,
		    completed_at = NOW()
		WHERE id = $1`,
		id, string(status), output, errMsg, durationMs,
	)
	if err != nil {
		return fmt.Errorf("update tool execution status: %w", err)
	}
	return nil
}
