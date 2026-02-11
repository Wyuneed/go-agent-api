package tool

import (
	"context"
	"sync"

	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

// Tool interface that all tools must implement.
type Tool interface {
	Name() string
	Description() string
	Definition() toolspec.Tool
	Execute(ctx context.Context, args map[string]any) (any, error)
	RequiresApproval() bool
	Validate(args map[string]any) error
}

// BaseTool provides default implementations.
type BaseTool struct{}

func (BaseTool) RequiresApproval() bool            { return false }
func (BaseTool) Validate(args map[string]any) error { return nil }

// ToolRegistry manages all available tools.
type ToolRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) RegisterAll(tools ...Tool) {
	for _, tool := range tools {
		r.Register(tool)
	}
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *ToolRegistry) List() []toolspec.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]toolspec.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, tool.Definition())
	}
	return defs
}

func (r *ToolRegistry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ListForToken returns tools filtered by token permissions.
func (r *ToolRegistry) ListForToken(allowedTools []string) []toolspec.Tool {
	if len(allowedTools) == 0 {
		return r.List()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	allowedMap := make(map[string]bool)
	for _, t := range allowedTools {
		if t == "*" {
			return r.List()
		}
		allowedMap[t] = true
	}

	defs := make([]toolspec.Tool, 0)
	for name, tool := range r.tools {
		if allowedMap[name] {
			defs = append(defs, tool.Definition())
		}
	}
	return defs
}
