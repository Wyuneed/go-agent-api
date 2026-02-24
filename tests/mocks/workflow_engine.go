package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/wyuneed/go-agent-api/internal/application/port"
)

// MockWorkflowEngine is a mock of port.WorkflowEngine.
type MockWorkflowEngine struct {
	mock.Mock
}

func (m *MockWorkflowEngine) Run(ctx context.Context, input port.WorkflowInput) (*port.WorkflowOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.WorkflowOutput), args.Error(1)
}

func (m *MockWorkflowEngine) Resume(ctx context.Context, pausedStateJSON []byte, approved bool) (*port.WorkflowOutput, error) {
	args := m.Called(ctx, pausedStateJSON, approved)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.WorkflowOutput), args.Error(1)
}

// NewMockWorkflowOutput creates a simple workflow output for testing.
func NewMockWorkflowOutput(response string) *port.WorkflowOutput {
	return &port.WorkflowOutput{
		Response:       response,
		TokensUsed:     15,
		ToolCallsCount: 0,
		AgentType:      "general",
	}
}
