package tool

import (
	"context"
	"fmt"
	"time"

	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

type ExecuteToolInput struct {
	ToolCall toolspec.ToolCall
	User     *entity.User
	Token    *entity.Token
}

type ExecuteToolOutput struct {
	ToolCallID string        `json:"tool_call_id"`
	ToolName   string        `json:"tool_name"`
	Result     any           `json:"result"`
	Duration   time.Duration `json:"duration"`
}

type ExecuteToolUseCase struct {
	registry      *ToolRegistry
	maxConcurrent int
}

func NewExecuteToolUseCase(registry *ToolRegistry, maxConcurrent int) *ExecuteToolUseCase {
	return &ExecuteToolUseCase{
		registry:      registry,
		maxConcurrent: maxConcurrent,
	}
}

func (uc *ExecuteToolUseCase) Execute(ctx context.Context, input ExecuteToolInput) (*ExecuteToolOutput, error) {
	// Check token permission
	if input.Token != nil && !input.Token.CanUseTool(input.ToolCall.Function.Name) {
		return nil, fmt.Errorf("token does not have access to tool: %s", input.ToolCall.Function.Name)
	}

	// Get tool from registry
	t, ok := uc.registry.Get(input.ToolCall.Function.Name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", input.ToolCall.Function.Name)
	}

	// Parse arguments
	args, err := input.ToolCall.ParseArguments()
	if err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}

	// Validate
	if err := t.Validate(args); err != nil {
		return nil, fmt.Errorf("validate arguments: %w", err)
	}

	// Execute
	start := time.Now()
	result, err := t.Execute(ctx, args)
	duration := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("execute tool %s: %w", input.ToolCall.Function.Name, err)
	}

	return &ExecuteToolOutput{
		ToolCallID: input.ToolCall.ID,
		ToolName:   input.ToolCall.Function.Name,
		Result:     result,
		Duration:   duration,
	}, nil
}
