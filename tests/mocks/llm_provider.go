package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/wyuneed/go-agent-api/internal/application/port"
)

// MockLLMProvider is a mock of port.LLMProvider.
type MockLLMProvider struct {
	mock.Mock
}

func (m *MockLLMProvider) Chat(ctx context.Context, req port.ChatRequest) (*port.ChatResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.ChatResponse), args.Error(1)
}

func (m *MockLLMProvider) ChatStream(ctx context.Context, req port.ChatRequest) (<-chan port.StreamEvent, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan port.StreamEvent), args.Error(1)
}

// NewMockChatResponse creates a simple chat response for testing.
func NewMockChatResponse(content string) *port.ChatResponse {
	return &port.ChatResponse{
		ID:     "chatcmpl-test",
		Object: "chat.completion",
		Model:  "gpt-4o-mini",
		Choices: []port.Choice{
			{
				Index: 0,
				Message: port.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: port.UsageInfo{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}
}
