package builtin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/tool/builtin"
)

func TestCalculatorTool_Name(t *testing.T) {
	calc := builtin.NewCalculatorTool()
	assert.Equal(t, "calculator", calc.Name())
}

func TestCalculatorTool_Execute(t *testing.T) {
	calc := builtin.NewCalculatorTool()
	ctx := context.Background()

	tests := []struct {
		name       string
		expression string
		wantResult float64
		wantErr    bool
	}{
		{"addition", "2 + 3", 5, false},
		{"subtraction", "10 - 4", 6, false},
		{"multiplication", "3 * 4", 12, false},
		{"division", "15 / 3", 5, false},
		{"parentheses", "(2 + 3) * 4", 20, false},
		{"nested parens", "(10 - (2 + 3)) * 2", 10, false},
		{"unary minus", "-5 + 10", 5, false},
		{"float division", "7 / 2", 3.5, false},
		{"division by zero", "10 / 0", 0, false}, // returns error in result map
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Execute(ctx, map[string]any{"expression": tt.expression})
			require.NoError(t, err)

			resultMap, ok := result.(map[string]any)
			require.True(t, ok)

			if tt.name == "division by zero" {
				assert.NotNil(t, resultMap["error"])
			} else {
				assert.Equal(t, tt.expression, resultMap["expression"])
				assert.InDelta(t, tt.wantResult, resultMap["result"], 0.0001)
			}
		})
	}
}

func TestCalculatorTool_Execute_MissingExpression(t *testing.T) {
	calc := builtin.NewCalculatorTool()
	_, err := calc.Execute(context.Background(), map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expression")
}

func TestCalculatorTool_Execute_InvalidExpression(t *testing.T) {
	calc := builtin.NewCalculatorTool()
	result, err := calc.Execute(context.Background(), map[string]any{"expression": "not valid math!"})
	// Calculator returns error in result map, not as Go error
	require.NoError(t, err)
	resultMap, ok := result.(map[string]any)
	require.True(t, ok)
	assert.NotNil(t, resultMap["error"])
}

func TestCalculatorTool_Definition(t *testing.T) {
	calc := builtin.NewCalculatorTool()
	def := calc.Definition()

	assert.Equal(t, "function", def.Type)
	assert.Equal(t, "calculator", def.Function.Name)
	assert.NotEmpty(t, def.Function.Description)
	assert.NotNil(t, def.Function.Parameters)
	assert.Contains(t, def.Function.Parameters.Required, "expression")
}

func TestCalculatorTool_RequiresApproval(t *testing.T) {
	calc := builtin.NewCalculatorTool()
	assert.False(t, calc.RequiresApproval())
}
