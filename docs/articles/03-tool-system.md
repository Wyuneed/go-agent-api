# Add a New AI Tool in 15 Lines of Go: The OpenAI-Compatible Tool System Explained

*Part 3 of the "Building Production-Ready AI Agent APIs in Go" series*

---

Every AI agent framework needs a tool system. Python's LangChain has `@tool` decorators. LangGraph has `ToolNode`. In Go, we built ours around a clean interface pattern — and the result is that adding a new AI tool to a production agent takes about 15 lines of code and 2 lines to register it.

This is not an exaggeration. By the end of this article, you will have added a real, working tool that your AI agent can call. Let me show you exactly how it works.

---

## The Tool Interface: 6 Methods, 2 Are Free

Everything starts with this interface in `internal/application/usecase/tool/tool_registry.go`:

```go
// Tool interface that all tools must implement.
type Tool interface {
    Name() string
    Description() string
    Definition() toolspec.Tool
    Execute(ctx context.Context, args map[string]any) (any, error)
    RequiresApproval() bool
    Validate(args map[string]any) error
}
```

Six methods. But here is the trick — you never implement `RequiresApproval()` or `Validate()` yourself unless you need to. There is a `BaseTool` struct you embed that gives you both for free:

```go
// BaseTool provides default implementations.
type BaseTool struct{}

func (BaseTool) RequiresApproval() bool            { return false }
func (BaseTool) Validate(args map[string]any) error { return nil }
```

So in practice, every tool you write needs exactly **four** methods: `Name()`, `Description()`, `Definition()`, and `Execute()`. Three of those are trivial one-liners. The only method with real logic is `Execute()`.

---

## The CalculatorTool: A 15-Line Deep Dive

Here is the entire `CalculatorTool` from `internal/application/usecase/tool/builtin/calculator.go`:

```go
type CalculatorTool struct {
    tool.BaseTool  // Embeds RequiresApproval() and Validate()
}

func NewCalculatorTool() *CalculatorTool {
    return &CalculatorTool{}
}

func (t *CalculatorTool) Name() string { return "calculator" }

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
        return map[string]any{"error": err.Error(), "expression": expr}, nil
    }
    return map[string]any{"expression": expr, "result": result}, nil
}
```

That is the entire public surface of the tool. The 15 lines that matter are the `Execute()` function and the struct definition. Everything else is metadata that tells the LLM what the tool does and what parameters it expects.

### The Secret: Go's AST Parser as a Safe Evaluator

The `evaluate()` method underneath is the genuinely interesting part:

```go
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
        left, _ := t.eval(n.X)
        right, _ := t.eval(n.Y)
        switch n.Op {
        case token.ADD: return left + right, nil
        case token.SUB: return left - right, nil
        case token.MUL: return left * right, nil
        case token.QUO:
            if right == 0 { return 0, fmt.Errorf("division by zero") }
            return left / right, nil
        }
    case *ast.ParenExpr:
        return t.eval(n.X)
    case *ast.UnaryExpr:
        val, _ := t.eval(n.X)
        if n.Op == token.SUB { return -val, nil }
        return val, nil
    }
    return 0, fmt.Errorf("unsupported expression type")
}
```

We use Go's own `go/parser` and `go/ast` packages — the same infrastructure the Go compiler uses — to parse and evaluate mathematical expressions. This gives us:

- **Safety**: No `eval()`, no string execution. Just AST tree walking.
- **Correctness**: Operator precedence handled automatically by the parser.
- **Zero dependencies**: It is all in the Go standard library.

`parser.ParseExpr("(2 + 3) * 4")` returns an `*ast.BinaryExpr` where the left side is another `*ast.BinaryExpr` (the parenthesized `2 + 3`) and the right side is `*ast.BasicLit` (the number 4). We walk the tree recursively and evaluate each node. No external math library needed.

---

## The toolspec Package: Why OpenAI Format Matters

The `Definition()` method returns a `toolspec.Tool` from `pkg/toolspec/openai_format.go`. This is a public package because the format is shared across the entire stack:

```go
// Tool definition compatible with OpenAI, Anthropic (via LiteLLM), and other providers.
type Tool struct {
    Type     string   `json:"type"`     // Always "function"
    Function Function `json:"function"`
}

type Function struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    Parameters  *JSONSchema `json:"parameters,omitempty"`
    Strict      bool        `json:"strict,omitempty"`
}

type JSONSchema struct {
    Type       string                    `json:"type"`
    Properties map[string]PropertySchema `json:"properties,omitempty"`
    Required   []string                  `json:"required,omitempty"`
}
```

When the LLM (GPT-4o, Claude, Gemini via LiteLLM) receives this definition, it knows exactly what `calculator` does, what parameters it accepts, and which are required. When the LLM decides to call `calculator`, it returns a `ToolCall`:

```go
type ToolCall struct {
    ID       string       `json:"id"`       // e.g. "call_abc123"
    Type     string       `json:"type"`     // "function"
    Function FunctionCall `json:"function"`
}

type FunctionCall struct {
    Name      string `json:"name"`       // "calculator"
    Arguments string `json:"arguments"`  // JSON string: {"expression": "(2+3)*4"}
}

// ParseArguments parses the JSON arguments into a map.
func (tc *ToolCall) ParseArguments() (map[string]any, error) {
    var args map[string]any
    return args, json.Unmarshal([]byte(tc.Function.Arguments), &args)
}
```

This is the OpenAI tool calling format. Because we use LiteLLM as a proxy, the exact same JSON works with GPT-4o, Claude 3.5 Sonnet, and Gemini Pro. You change your model name in a config file; the tool calling code does not change at all.

---

## The ToolRegistry: Thread-Safe Tool Management

All tools are stored in a `ToolRegistry` — a thread-safe map protected by `sync.RWMutex`:

```go
type ToolRegistry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

func (r *ToolRegistry) Register(tool Tool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.tools[name], ok
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
```

The write lock is only held during `Register()`. The much more frequent read operations (`Get()`, `List()`) use the read lock, which allows concurrent reads without blocking each other. In a high-throughput agent API, this matters.

### Per-Token Tool Permissions

There is one more method that makes this registry genuinely production-ready:

```go
// ListForToken returns tools filtered by token permissions.
func (r *ToolRegistry) ListForToken(allowedTools []string) []toolspec.Tool {
    if len(allowedTools) == 0 {
        return r.List() // All tools allowed
    }

    r.mu.RLock()
    defer r.mu.RUnlock()

    allowedMap := make(map[string]bool)
    for _, t := range allowedTools {
        if t == "*" {
            return r.List() // Wildcard: all allowed
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
```

Each API token in the database has an `AllowedTools []string` field. When the agent is invoked, the tool list passed to the LLM is filtered to only the tools that token is permitted to use. An API key for a customer who purchased "basic" might only see `calculator`. An admin key sees everything.

The LLM never sees tools it is not allowed to call. No extra validation needed.

---

## Adding Your Own Tool: A Practical Example

Let's add a `WeatherTool` that calls a hypothetical weather API. Here is everything you need to write:

```go
// internal/application/usecase/tool/builtin/weather.go
package builtin

import (
    "context"
    "fmt"
    "net/http"
    "encoding/json"

    "github.com/wyuneed/go-agent-api/internal/application/usecase/tool"
    "github.com/wyuneed/go-agent-api/pkg/toolspec"
)

type WeatherTool struct {
    tool.BaseTool
    apiKey string
}

func NewWeatherTool(apiKey string) *WeatherTool {
    return &WeatherTool{apiKey: apiKey}
}

func (t *WeatherTool) Name() string { return "get_weather" }

func (t *WeatherTool) Description() string {
    return "Get the current weather for a city. Returns temperature, conditions, and humidity."
}

func (t *WeatherTool) Definition() toolspec.Tool {
    return toolspec.NewTool(t.Name(), t.Description(), &toolspec.JSONSchema{
        Type: "object",
        Properties: map[string]toolspec.PropertySchema{
            "city": {
                Type:        "string",
                Description: "The city name, e.g. 'Tokyo' or 'New York'",
            },
        },
        Required: []string{"city"},
    })
}

func (t *WeatherTool) Execute(ctx context.Context, args map[string]any) (any, error) {
    city, _ := args["city"].(string)
    if city == "" {
        return nil, fmt.Errorf("city is required")
    }
    // Call your weather API here...
    return map[string]any{
        "city":        city,
        "temperature": "22°C",
        "conditions":  "Partly cloudy",
    }, nil
}
```

Then register it in `cmd/api/main.go` — **two lines**:

```go
toolRegistry.RegisterAll(
    builtin.NewCalculatorTool(),
    builtin.NewWebSearchTool(cfg.Tools.WebSearchAPIKey),
    builtin.NewWeatherTool(cfg.Tools.WeatherAPIKey), // ← Add this
)
```

That is genuinely it. The next time a user asks "What is the weather in Tokyo?", the LLM will call `get_weather` with `{"city": "Tokyo"}`, your `Execute()` method runs, and the result goes back to the LLM for a final response.

---

## The Execution Pipeline

When the Eino workflow calls your tool, it goes through `execute_tool.go` which handles the full pipeline:

1. **Look up the tool** in the registry by name
2. **Check permissions** — does this token's `AllowedTools` include this tool name?
3. **Parse arguments** — `tc.ParseArguments()` unmarshals the JSON string
4. **Validate** — `tool.Validate(args)` (default: always passes)
5. **Execute** — `tool.Execute(ctx, args)`

```go
func (uc *ExecuteToolUseCase) Execute(ctx context.Context, input ExecuteToolInput) (*ExecuteToolOutput, error) {
    t, ok := uc.registry.Get(input.ToolCall.Function.Name)
    if !ok {
        return nil, apperrors.ErrNotFound.Wrap(fmt.Errorf("tool not found: %s", input.ToolCall.Function.Name))
    }

    // Check token permission
    if input.Token != nil && !input.Token.CanUseTool(t.Name()) {
        return nil, apperrors.ErrToolNotAllowed
    }

    args, err := input.ToolCall.ParseArguments()
    if err != nil {
        return nil, fmt.Errorf("parse arguments: %w", err)
    }

    if err := t.Validate(args); err != nil {
        return nil, fmt.Errorf("validate: %w", err)
    }

    start := time.Now()
    result, err := t.Execute(ctx, args)
    duration := time.Since(start)

    return &ExecuteToolOutput{
        ToolCallID: input.ToolCall.ID,
        ToolName:   t.Name(),
        Result:     result,
        Duration:   duration,
    }, err
}
```

Notice that the Eino graph's `actNode` (which we will cover in Article 4) also checks `t.RequiresApproval()` before executing. If a tool needs human approval, the graph pauses instead of executing — no changes needed to the `ExecuteToolUseCase`.

---

## What Makes This Design Work

The elegance here comes from a few specific decisions:

**Interface over inheritance.** Go does not have class hierarchies. The `Tool` interface, combined with `BaseTool` embedding, gives you optional overrides without forcing you to implement everything.

**Separation of definition from execution.** `Definition()` describes the tool to the LLM in a format the LLM understands. `Execute()` is pure Go logic. These two concerns never mix.

**Registry as the single source of truth.** The LLM, the HTTP endpoint (`GET /v1/tools`), and the workflow graph all use the same registry. There is no duplication of tool definitions.

**OpenAI format means portability.** By targeting the OpenAI tool calling spec, the same tool definitions work with any LLM provider that LiteLLM supports — which today includes GPT-4o, Claude 3.5, Gemini 1.5 Pro, Llama 3, Mistral, and dozens more.

---

## What We Just Learned

- The `Tool` interface has 4 required methods; 2 are free via `BaseTool` embedding
- `CalculatorTool` uses Go's `go/ast` package for safe, dependency-free expression evaluation
- The `toolspec` package defines OpenAI-compatible tool types that work with any LLM via LiteLLM
- `ToolRegistry` is a thread-safe map with `sync.RWMutex` that supports per-token filtering
- Adding a new tool requires: one new file implementing the interface + two lines in `main.go`

---

## Try This Now

```bash
git clone https://github.com/wyuneed/go-agent-api
make docker-up
make migrate-up
make run
curl http://127.0.0.1:8080/v1/tools  # See registered tools
```

---

*Next in the series: [Article 4 — Eino: ByteDance's LangGraph for Go — Building a 6-Node Agentic Workflow](#)*

*Previous: [Article 2 — Domain-Driven Design for Go Developers](#)*
