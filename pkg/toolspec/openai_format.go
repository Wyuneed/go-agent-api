package toolspec

import "encoding/json"

// Tool definition compatible with OpenAI, Anthropic (via LiteLLM), and other providers.
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  *JSONSchema `json:"parameters,omitempty"`
	Strict      bool        `json:"strict,omitempty"`
}

type JSONSchema struct {
	Type                 string                    `json:"type"`
	Description          string                    `json:"description,omitempty"`
	Properties           map[string]PropertySchema `json:"properties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AdditionalProperties *bool                     `json:"additionalProperties,omitempty"`
}

type PropertySchema struct {
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Enum        []string        `json:"enum,omitempty"`
	Items       *PropertySchema `json:"items,omitempty"`
	Default     any             `json:"default,omitempty"`
	Minimum     *float64        `json:"minimum,omitempty"`
	Maximum     *float64        `json:"maximum,omitempty"`
}

// ToolCall from LLM response.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ParseArguments parses the JSON arguments into a map.
func (tc *ToolCall) ParseArguments() (map[string]any, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// ToolMessage for sending results back.
type ToolMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name,omitempty"`
}

// NewTool creates a new tool definition.
func NewTool(name, description string, params *JSONSchema) Tool {
	return Tool{
		Type: "function",
		Function: Function{
			Name:        name,
			Description: description,
			Parameters:  params,
		},
	}
}

// NewStrictTool creates a tool with OpenAI strict mode.
func NewStrictTool(name, description string, params *JSONSchema) Tool {
	if params != nil {
		falseVal := false
		params.AdditionalProperties = &falseVal
	}
	return Tool{
		Type: "function",
		Function: Function{
			Name:        name,
			Description: description,
			Parameters:  params,
			Strict:      true,
		},
	}
}
