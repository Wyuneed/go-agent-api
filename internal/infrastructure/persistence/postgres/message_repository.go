package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

type MessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

func (r *MessageRepository) Create(ctx context.Context, msg *entity.Message) error {
	var toolCallsJSON []byte
	if len(msg.ToolCalls) > 0 {
		toolCallsJSON, _ = json.Marshal(msg.ToolCalls)
	}

	var latencyMs *int
	if msg.Latency > 0 {
		ms := int(msg.Latency.Milliseconds())
		latencyMs = &ms
	}

	query := `
		INSERT INTO messages (id, conversation_id, role, content, name, tool_calls, tool_call_id,
			model, prompt_tokens, completion_tokens, latency_ms, is_streaming, stream_complete, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.pool.Exec(ctx, query,
		msg.ID, msg.ConversationID, string(msg.Role), msg.Content, msg.Name,
		toolCallsJSON, msg.ToolCallID,
		msg.Model, msg.PromptTokens, msg.CompletionTokens, latencyMs,
		msg.IsStreaming, msg.StreamComplete, msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func (r *MessageRepository) CreateBatch(ctx context.Context, msgs []*entity.Message) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, msg := range msgs {
		var toolCallsJSON []byte
		if len(msg.ToolCalls) > 0 {
			toolCallsJSON, _ = json.Marshal(msg.ToolCalls)
		}

		var latencyMs *int
		if msg.Latency > 0 {
			ms := int(msg.Latency.Milliseconds())
			latencyMs = &ms
		}

		query := `
			INSERT INTO messages (id, conversation_id, role, content, name, tool_calls, tool_call_id,
				model, prompt_tokens, completion_tokens, latency_ms, is_streaming, stream_complete, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

		_, err := tx.Exec(ctx, query,
			msg.ID, msg.ConversationID, string(msg.Role), msg.Content, msg.Name,
			toolCallsJSON, msg.ToolCallID,
			msg.Model, msg.PromptTokens, msg.CompletionTokens, latencyMs,
			msg.IsStreaming, msg.StreamComplete, msg.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("create batch message: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *MessageRepository) FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*entity.Message, error) {
	query := `
		SELECT id, conversation_id, role, content, name, tool_calls, tool_call_id,
			model, prompt_tokens, completion_tokens, latency_ms, is_streaming, stream_complete,
			sequence_number, created_at
		FROM messages WHERE conversation_id = $1 ORDER BY sequence_number ASC`

	return r.queryMessages(ctx, query, conversationID)
}

func (r *MessageRepository) FindByConversationIDPaginated(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
	query := `
		SELECT id, conversation_id, role, content, name, tool_calls, tool_call_id,
			model, prompt_tokens, completion_tokens, latency_ms, is_streaming, stream_complete,
			sequence_number, created_at
		FROM messages WHERE conversation_id = $1 ORDER BY sequence_number ASC
		LIMIT $2 OFFSET $3`

	return r.queryMessages(ctx, query, conversationID, limit, offset)
}

func (r *MessageRepository) FindLastN(ctx context.Context, conversationID uuid.UUID, n int) ([]*entity.Message, error) {
	query := `
		SELECT id, conversation_id, role, content, name, tool_calls, tool_call_id,
			model, prompt_tokens, completion_tokens, latency_ms, is_streaming, stream_complete,
			sequence_number, created_at
		FROM (
			SELECT * FROM messages WHERE conversation_id = $1
			ORDER BY sequence_number DESC LIMIT $2
		) sub ORDER BY sequence_number ASC`

	return r.queryMessages(ctx, query, conversationID, n)
}

func (r *MessageRepository) CountByConversationID(ctx context.Context, conversationID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM messages WHERE conversation_id = $1",
		conversationID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count messages: %w", err)
	}
	return count, nil
}

func (r *MessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM messages WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	return nil
}

func (r *MessageRepository) DeleteByConversationID(ctx context.Context, conversationID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM messages WHERE conversation_id = $1", conversationID)
	if err != nil {
		return fmt.Errorf("delete messages by conversation: %w", err)
	}
	return nil
}

func (r *MessageRepository) queryMessages(ctx context.Context, query string, args ...any) ([]*entity.Message, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []*entity.Message
	for rows.Next() {
		msg, err := r.scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *MessageRepository) scanMessage(rows pgx.Rows) (*entity.Message, error) {
	msg := &entity.Message{}
	var role string
	var toolCallsJSON []byte
	var latencyMs *int

	err := rows.Scan(
		&msg.ID, &msg.ConversationID, &role, &msg.Content, &msg.Name,
		&toolCallsJSON, &msg.ToolCallID,
		&msg.Model, &msg.PromptTokens, &msg.CompletionTokens, &latencyMs,
		&msg.IsStreaming, &msg.StreamComplete, &msg.SequenceNumber, &msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan message: %w", err)
	}

	msg.Role = entity.MessageRole(role)

	if toolCallsJSON != nil {
		json.Unmarshal(toolCallsJSON, &msg.ToolCalls)
	}

	if latencyMs != nil {
		msg.Latency = time.Duration(*latencyMs) * time.Millisecond
	}

	return msg, nil
}
