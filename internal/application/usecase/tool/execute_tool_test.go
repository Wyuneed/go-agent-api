package tool_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/tool"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

func TestExecuteToolUseCase_Success(t *testing.T) {
	registry := tool.NewToolRegistry()
	registry.Register(&mockTool{name: "mock_tool"})

	uc := tool.NewExecuteToolUseCase(registry, 10)
	userID := uuid.New()

	result, err := uc.Execute(context.Background(), tool.ExecuteToolInput{
		ToolCall: toolspec.ToolCall{
			ID:   "call_123",
			Type: "function",
			Function: toolspec.FunctionCall{
				Name:      "mock_tool",
				Arguments: `{"input": "test"}`,
			},
		},
		User:  &entity.User{ID: userID},
		Token: &entity.Token{AllowedTools: nil},
	})

	require.NoError(t, err)
	assert.Equal(t, "call_123", result.ToolCallID)
	assert.Equal(t, "mock_tool", result.ToolName)
	assert.NotNil(t, result.Result)
}

func TestExecuteToolUseCase_ToolNotFound(t *testing.T) {
	registry := tool.NewToolRegistry()
	uc := tool.NewExecuteToolUseCase(registry, 10)

	_, err := uc.Execute(context.Background(), tool.ExecuteToolInput{
		ToolCall: toolspec.ToolCall{
			Function: toolspec.FunctionCall{
				Name:      "nonexistent_tool",
				Arguments: "{}",
			},
		},
		User:  &entity.User{ID: uuid.New()},
		Token: &entity.Token{AllowedTools: nil},
	})

	assert.Error(t, err)
}

func TestExecuteToolUseCase_ToolNotAllowed(t *testing.T) {
	registry := tool.NewToolRegistry()
	registry.Register(&mockTool{name: "restricted_tool"})

	uc := tool.NewExecuteToolUseCase(registry, 10)

	_, err := uc.Execute(context.Background(), tool.ExecuteToolInput{
		ToolCall: toolspec.ToolCall{
			Function: toolspec.FunctionCall{
				Name:      "restricted_tool",
				Arguments: "{}",
			},
		},
		User: &entity.User{ID: uuid.New()},
		Token: &entity.Token{
			AllowedTools: []string{"other_tool"}, // restricted_tool NOT in list
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		},
	})

	assert.Error(t, err)
}

// Note: RequiresApproval is checked in the Eino workflow graph (actNode),
// not in ExecuteToolUseCase directly. ExecuteToolUseCase always executes the tool.

func TestExecuteToolUseCase_InvalidArguments(t *testing.T) {
	registry := tool.NewToolRegistry()
	registry.Register(&mockTool{name: "my_tool"})

	uc := tool.NewExecuteToolUseCase(registry, 10)

	_, err := uc.Execute(context.Background(), tool.ExecuteToolInput{
		ToolCall: toolspec.ToolCall{
			Function: toolspec.FunctionCall{
				Name:      "my_tool",
				Arguments: "not valid json",
			},
		},
		User:  &entity.User{ID: uuid.New()},
		Token: &entity.Token{AllowedTools: nil},
	})

	assert.Error(t, err)
}
