package valueobject

import (
	"encoding/json"
	"fmt"
)

// ToolCall represents an OpenAI-compatible tool call value object.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func NewToolCall(id, name, arguments string) (ToolCall, error) {
	if id == "" {
		return ToolCall{}, fmt.Errorf("tool call id cannot be empty")
	}
	if name == "" {
		return ToolCall{}, fmt.Errorf("tool call function name cannot be empty")
	}
	if !json.Valid([]byte(arguments)) {
		return ToolCall{}, fmt.Errorf("tool call arguments must be valid JSON")
	}
	return ToolCall{
		ID:   id,
		Type: "function",
		Function: ToolFunction{
			Name:      name,
			Arguments: arguments,
		},
	}, nil
}

func (tc ToolCall) ParseArguments(v any) error {
	return json.Unmarshal([]byte(tc.Function.Arguments), v)
}
