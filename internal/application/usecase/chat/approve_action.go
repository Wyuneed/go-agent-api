package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/application/port"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
)

type ApproveInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Approved       bool
	Reason         string
}

type ApproveOutput struct {
	Conversation *entity.Conversation `json:"conversation"`
	Status       string               `json:"status"`
	Response     string               `json:"response,omitempty"`
}

type ApproveActionUseCase struct {
	convRepo repository.ConversationRepository
	msgRepo  repository.MessageRepository
	engine   port.WorkflowEngine
}

func NewApproveActionUseCase(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	engine port.WorkflowEngine,
) *ApproveActionUseCase {
	return &ApproveActionUseCase{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		engine:   engine,
	}
}

func (uc *ApproveActionUseCase) Execute(ctx context.Context, input ApproveInput) (*ApproveOutput, error) {
	conv, err := uc.convRepo.FindByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	if conv.UserID != input.UserID {
		return nil, fmt.Errorf("unauthorized access to conversation")
	}

	if !conv.IsPendingApproval() {
		return nil, fmt.Errorf("conversation is not pending approval")
	}

	// Load serialised workflow state from the JSONB column
	stateRaw, ok := conv.WorkflowState["paused_state"]
	if !ok {
		return nil, fmt.Errorf("no paused workflow state found in conversation")
	}
	stateJSON, ok := stateRaw.(string)
	if !ok {
		return nil, fmt.Errorf("invalid paused workflow state format")
	}

	// Resume the workflow engine from the saved checkpoint
	result, err := uc.engine.Resume(ctx, []byte(stateJSON), input.Approved)
	if err != nil {
		return nil, fmt.Errorf("resume workflow: %w", err)
	}

	// Persist the assistant response produced after resuming
	if result.Response != "" {
		assistantMsg := entity.NewAssistantMessage(conv.ID, result.Response, nil, "")
		if err := uc.msgRepo.Create(ctx, assistantMsg); err != nil {
			return nil, fmt.Errorf("save assistant message: %w", err)
		}
	}

	// Update conversation
	conv.AddMessageStats(result.TokensUsed, result.ToolCallsCount)

	if result.RequiresApproval {
		// Workflow paused again for another approval
		if err := uc.convRepo.UpdateWorkflowState(ctx, conv.ID, map[string]any{
			"paused_state": string(result.PausedStateJSON),
		}); err != nil {
			return nil, fmt.Errorf("persist next paused state: %w", err)
		}
		conv.RequestApproval(result.AgentType, map[string]any{
			"reason": result.ApprovalReason,
		})
	} else if input.Approved {
		conv.Approve()
	} else {
		conv.Reject(input.Reason)
	}

	if err := uc.convRepo.Update(ctx, conv); err != nil {
		return nil, fmt.Errorf("update conversation: %w", err)
	}

	status := "rejected"
	if input.Approved {
		status = "approved"
	}
	if result.RequiresApproval {
		status = "pending_approval"
	}

	return &ApproveOutput{
		Conversation: conv,
		Status:       status,
		Response:     result.Response,
	}, nil
}
