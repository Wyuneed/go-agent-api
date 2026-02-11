package builtin

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"github.com/wyuneed/go-agent-api/internal/application/usecase/tool"
	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

type CalculatorTool struct {
	tool.BaseTool
}

func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

func (t *CalculatorTool) Name() string {
	return "calculator"
}

func (t *CalculatorTool) Description() string {
	return "Perform mathematical calculations. Supports basic arithmetic (+, -, *, /), parentheses, and common functions."
}

func (t *CalculatorTool) Definition() toolspec.Tool {
	return toolspec.NewTool(t.Name(), t.Description(), &toolspec.JSONSchema{
		Type: "object",
		Properties: map[string]toolspec.PropertySchema{
			"expression": {
				Type:        "string",
				Description: "Mathematical expression to evaluate. Example: '(2 + 3) * 4' or '15 / 3'",
			},
		},
		Required: []string{"expression"},
	})
}

func (t *CalculatorTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	expr, _ := args["expression"].(string)
	if expr == "" {
		return nil, fmt.Errorf("expression is required")
	}

	result, err := t.evaluate(expr)
	if err != nil {
		return map[string]any{
			"error":      err.Error(),
			"expression": expr,
		}, nil
	}

	return map[string]any{
		"expression": expr,
		"result":     result,
	}, nil
}

func (t *CalculatorTool) evaluate(expr string) (float64, error) {
	node, err := parser.ParseExpr(expr)
	if err != nil {
		return 0, fmt.Errorf("invalid expression: %w", err)
	}
	return t.eval(node)
}

func (t *CalculatorTool) eval(node ast.Expr) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		return strconv.ParseFloat(n.Value, 64)
	case *ast.BinaryExpr:
		left, err := t.eval(n.X)
		if err != nil {
			return 0, err
		}
		right, err := t.eval(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		default:
			return 0, fmt.Errorf("unsupported operator: %s", n.Op)
		}
	case *ast.ParenExpr:
		return t.eval(n.X)
	case *ast.UnaryExpr:
		val, err := t.eval(n.X)
		if err != nil {
			return 0, err
		}
		if n.Op == token.SUB {
			return -val, nil
		}
		return val, nil
	default:
		return 0, fmt.Errorf("unsupported expression type")
	}
}
