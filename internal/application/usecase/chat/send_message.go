package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/application/port"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

type SendMessageInput struct {
	ConversationID *uuid.UUID
	UserID         uuid.UUID
	Content        string
	AgentType      string
	Messages       []port.ChatMessage
	Model          string
	Tools          []toolspec.Tool
}

type SendMessageOutput struct {
	ConversationID uuid.UUID        `json:"conversation_id"`
	Message        *entity.Message  `json:"message"`
	Response       string           `json:"response"`
}

type SendMessageUseCase struct {
	convRepo     repository.ConversationRepository
	msgRepo      repository.MessageRepository
	llm          port.LLMProvider
	engine       port.WorkflowEngine
	defaultModel string
}

func NewSendMessageUseCase(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	llm port.LLMProvider,
	engine port.WorkflowEngine,
	defaultModel string,
) *SendMessageUseCase {
	if defaultModel == "" {
		defaultModel = "gpt-4o-mini"
	}
	return &SendMessageUseCase{
		convRepo:     convRepo,
		msgRepo:      msgRepo,
		llm:          llm,
		engine:       engine,
		defaultModel: defaultModel,
	}
}

func (uc *SendMessageUseCase) Execute(ctx context.Context, input SendMessageInput) (*SendMessageOutput, error) {
	// Stateless path: caller supplied a full message array (e.g. POST /v1/chat/completions).
	// Pass them directly to the LLM without touching the database.
	if len(input.Messages) > 0 {
		model := input.Model
		if model == "" {
			model = uc.defaultModel
		}
		resp, err := uc.llm.Chat(ctx, port.ChatRequest{
			Model:    model,
			Messages: input.Messages,
			Tools:    input.Tools,
		})
		if err != nil {
			return nil, fmt.Errorf("LLM call: %w", err)
		}
		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("no response from LLM")
		}
		return &SendMessageOutput{
			Response: resp.Choices[0].Message.Content,
		}, nil
	}

	// Stateful path: manage conversation + message storage in DB via the workflow engine.
	var conv *entity.Conversation
	var err error

	if input.ConversationID != nil {
		conv, err = uc.convRepo.FindByID(ctx, *input.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("conversation not found: %w", err)
		}
		if conv.UserID != input.UserID {
			return nil, fmt.Errorf("unauthorized access to conversation")
		}
	} else {
		agentType := input.AgentType
		if agentType == "" {
			agentType = "general"
		}
		conv = entity.NewConversation(input.UserID, agentType)
		if err := uc.convRepo.Create(ctx, conv); err != nil {
			return nil, fmt.Errorf("create conversation: %w", err)
		}
	}

	// Store user message
	userMsg := entity.NewUserMessage(conv.ID, input.Content)
	if err := uc.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	// Get conversation history for context
	messages, err := uc.msgRepo.FindByConversationID(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	chatMessages := make([]port.ChatMessage, 0, len(messages))
	for _, msg := range messages {
		chatMessages = append(chatMessages, port.ChatMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		})
	}

	model := input.Model
	if model == "" {
		model = uc.defaultModel
	}

	// Run the Eino workflow engine
	result, err := uc.engine.Run(ctx, port.WorkflowInput{
		ConversationID: conv.ID,
		UserID:         input.UserID,
		Content:        input.Content,
		Model:          model,
		Messages:       chatMessages,
		Temperature:    conv.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("workflow run: %w", err)
	}

	// Workflow paused for human approval — persist state and return early
	if result.RequiresApproval {
		if err := uc.convRepo.UpdateWorkflowState(ctx, conv.ID, map[string]any{
			"paused_state": string(result.PausedStateJSON),
		}); err != nil {
			return nil, fmt.Errorf("persist paused state: %w", err)
		}
		conv.RequestApproval(result.AgentType, map[string]any{
			"reason": result.ApprovalReason,
		})
		if err := uc.convRepo.Update(ctx, conv); err != nil {
			return nil, fmt.Errorf("update conversation status: %w", err)
		}
		return &SendMessageOutput{
			ConversationID: conv.ID,
			Response:       "pending_approval",
		}, nil
	}

	// Store assistant message
	assistantMsg := entity.NewAssistantMessage(conv.ID, result.Response, nil, model)
	if err := uc.msgRepo.Create(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("save assistant message: %w", err)
	}

	// Update conversation stats
	conv.AddMessageStats(result.TokensUsed, result.ToolCallsCount)
	conv.SetTitle(input.Content)
	if err := uc.convRepo.Update(ctx, conv); err != nil {
		return nil, fmt.Errorf("update conversation: %w", err)
	}

	return &SendMessageOutput{
		ConversationID: conv.ID,
		Message:        assistantMsg,
		Response:       result.Response,
	}, nil
}

func (uc *SendMessageUseCase) ExecuteStream(ctx context.Context, input SendMessageInput) (<-chan port.StreamEvent, error) {
	model := input.Model
	if model == "" {
		model = uc.defaultModel
	}

	return uc.llm.ChatStream(ctx, port.ChatRequest{
		Model:    model,
		Messages: input.Messages,
		Tools:    input.Tools,
		Stream:   true,
	})
}
