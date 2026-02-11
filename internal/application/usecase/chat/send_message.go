package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/application/port"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
)

type SendMessageInput struct {
	ConversationID *uuid.UUID
	UserID         uuid.UUID
	Content        string
	AgentType      string
	Messages       []port.ChatMessage
	Model          string
	Tools          []any
}

type SendMessageOutput struct {
	ConversationID uuid.UUID        `json:"conversation_id"`
	Message        *entity.Message  `json:"message"`
	Response       string           `json:"response"`
}

type SendMessageUseCase struct {
	convRepo repository.ConversationRepository
	msgRepo  repository.MessageRepository
	llm      port.LLMProvider
}

func NewSendMessageUseCase(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	llm port.LLMProvider,
) *SendMessageUseCase {
	return &SendMessageUseCase{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		llm:      llm,
	}
}

func (uc *SendMessageUseCase) Execute(ctx context.Context, input SendMessageInput) (*SendMessageOutput, error) {
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

	// Get conversation history
	messages, err := uc.msgRepo.FindByConversationID(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	// Build chat messages for LLM
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
		model = "gpt-4o-mini"
	}

	// Call LLM
	resp, err := uc.llm.Chat(ctx, port.ChatRequest{
		Model:       model,
		Messages:    chatMessages,
		Temperature: conv.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	choice := resp.Choices[0]

	// Store assistant message
	assistantMsg := entity.NewAssistantMessage(conv.ID, choice.Message.Content, nil, resp.Model)
	assistantMsg.PromptTokens = resp.Usage.PromptTokens
	assistantMsg.CompletionTokens = resp.Usage.CompletionTokens
	if err := uc.msgRepo.Create(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("save assistant message: %w", err)
	}

	// Update conversation stats
	conv.AddMessageStats(resp.Usage.TotalTokens, 0)
	conv.SetTitle(input.Content)
	if err := uc.convRepo.Update(ctx, conv); err != nil {
		return nil, fmt.Errorf("update conversation: %w", err)
	}

	return &SendMessageOutput{
		ConversationID: conv.ID,
		Message:        assistantMsg,
		Response:       choice.Message.Content,
	}, nil
}

func (uc *SendMessageUseCase) ExecuteStream(ctx context.Context, input SendMessageInput) (<-chan port.StreamEvent, error) {
	model := input.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	return uc.llm.ChatStream(ctx, port.ChatRequest{
		Model:    model,
		Messages: input.Messages,
		Stream:   true,
	})
}
