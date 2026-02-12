package tool_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/tool"
	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

// mockTool is a test double for the Tool interface.
type mockTool struct {
	tool.BaseTool
	name string
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "Mock tool: " + m.name }
func (m *mockTool) Definition() toolspec.Tool {
	return toolspec.NewTool(m.name, m.Description(), nil)
}
func (m *mockTool) Execute(_ context.Context, _ map[string]any) (any, error) {
	return map[string]string{"result": "ok"}, nil
}

func TestToolRegistry_Register(t *testing.T) {
	registry := tool.NewToolRegistry()
	registry.Register(&mockTool{name: "test_tool"})

	tools := registry.List()
	assert.Len(t, tools, 1)
	assert.Equal(t, "test_tool", tools[0].Function.Name)
}

func TestToolRegistry_RegisterAll(t *testing.T) {
	registry := tool.NewToolRegistry()
	registry.RegisterAll(
		&mockTool{name: "tool_a"},
		&mockTool{name: "tool_b"},
		&mockTool{name: "tool_c"},
	)

	assert.Len(t, registry.List(), 3)
}

func TestToolRegistry_Get(t *testing.T) {
	registry := tool.NewToolRegistry()
	registry.Register(&mockTool{name: "my_tool"})

	t.Run("found", func(t *testing.T) {
		tool, ok := registry.Get("my_tool")
		assert.True(t, ok)
		assert.NotNil(t, tool)
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := registry.Get("nonexistent")
		assert.False(t, ok)
	})
}

func TestToolRegistry_ListForToken(t *testing.T) {
	registry := tool.NewToolRegistry()
	registry.RegisterAll(
		&mockTool{name: "tool_a"},
		&mockTool{name: "tool_b"},
		&mockTool{name: "tool_c"},
	)

	t.Run("nil allowed = all tools", func(t *testing.T) {
		tools := registry.ListForToken(nil)
		assert.Len(t, tools, 3)
	})

	t.Run("empty allowed = all tools", func(t *testing.T) {
		tools := registry.ListForToken([]string{})
		assert.Len(t, tools, 3)
	})

	t.Run("specific tools", func(t *testing.T) {
		tools := registry.ListForToken([]string{"tool_a", "tool_c"})
		assert.Len(t, tools, 2)
	})

	t.Run("wildcard", func(t *testing.T) {
		tools := registry.ListForToken([]string{"*"})
		assert.Len(t, tools, 3)
	})

	t.Run("nonexistent tool filtered", func(t *testing.T) {
		tools := registry.ListForToken([]string{"tool_a", "nonexistent"})
		assert.Len(t, tools, 1)
	})
}

func TestToolRegistry_ListNames(t *testing.T) {
	registry := tool.NewToolRegistry()
	registry.RegisterAll(
		&mockTool{name: "alpha"},
		&mockTool{name: "beta"},
	)

	names := registry.ListNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "beta")
}
