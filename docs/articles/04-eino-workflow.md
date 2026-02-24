# Eino: ByteDance's LangGraph for Go — Building a 6-Node Agentic Workflow

*Part 4 of the "Building Production-Ready AI Agent APIs in Go" series*

---

If you build AI agents in Python, you have probably used LangGraph. It lets you define stateful, cyclical workflows as a directed graph — nodes that process state, edges that connect them, and conditional branches that let the agent decide where to go next. It is the foundation of the "ReAct" (Reason + Act) agent loop.

**Eino** is ByteDance's Go equivalent. Same concept, same graph-based architecture — but compiled, type-safe, and running as a single binary at ~5MB. Almost nothing has been written about it in English.

This article is the deep dive. We will build the 6-node chatbot graph that powers the agent in this project, step by step. By the end, you will understand how a Go LLM call can loop through tool execution and return a final response.

---

## What Eino Is (and Is Not)

Eino is a workflow orchestration library from ByteDance (`github.com/cloudwego/eino`). It lets you:

- Define **nodes** — functions that take a state object and return a modified version
- Connect nodes with **edges** — simple A→B connections
- Add **branches** — conditional routing functions that decide which node to visit next
- **Compile** the graph into a `Runnable` that can be invoked with a state object

What it is not: a replacement for an LLM provider, a tool framework, or a database. It is purely the orchestration layer.

The key difference from LangGraph: Eino is **fully type-parameterized**. `compose.NewGraph[I, O]()` takes generic type parameters for input and output. The compiler enforces that your state type flows correctly through every node in the graph. In LangGraph, state shape errors only surface at runtime.

---

## The AgentState: One Struct That Carries Everything

Before building the graph, we need the state that flows through it. Every node receives a pointer to `AgentState` and returns a (potentially modified) pointer. Here is the complete struct from `internal/infrastructure/eino/state/agent_state.go`:

```go
type AgentState struct {
    // Identifiers
    ConversationID uuid.UUID `json:"conversation_id"`
    UserID         uuid.UUID `json:"user_id"`
    RequestID      string    `json:"request_id"`

    // Messages (the conversation history)
    Messages     []port.ChatMessage `json:"messages"`
    SystemPrompt string             `json:"system_prompt"`

    // Current turn
    UserInput     string `json:"user_input"`
    CurrentOutput string `json:"current_output"`

    // Model configuration
    Model       string  `json:"model"`
    Temperature float64 `json:"temperature"`
    MaxTokens   int     `json:"max_tokens"`

    // Tool execution
    AvailableTools []toolspec.Tool     `json:"available_tools"`
    PendingTools   []toolspec.ToolCall `json:"pending_tools"`
    ToolResults    []port.ChatMessage  `json:"tool_results"`

    // Multi-agent routing
    CurrentAgent string   `json:"current_agent"`
    AgentHistory []string `json:"agent_history"`

    // Human-in-the-loop
    RequiresApproval bool           `json:"requires_approval"`
    ApprovalReason   string         `json:"approval_reason"`
    ApprovalData     map[string]any `json:"approval_data"`
    IsApproved       *bool          `json:"is_approved,omitempty"`

    // Control flow
    Iteration     int    `json:"iteration"`
    MaxIterations int    `json:"max_iterations"`
    ShouldStop    bool   `json:"should_stop"`
    Error         string `json:"error,omitempty"`

    // Metrics
    StartTime      time.Time `json:"start_time"`
    TokensUsed     int       `json:"tokens_used"`
    ToolCallsCount int       `json:"tool_calls_count"`
}
```

Key design choices here:

**`PendingTools []toolspec.ToolCall`** — when the LLM responds with tool calls, they land here. The `act` node processes them. After processing, `PendingTools` is cleared.

**`ToolResults []port.ChatMessage`** — tool execution results are temporarily staged here, then moved to `Messages` by `FlushToolResults()`. This two-phase approach means the act→observe nodes can be tested independently.

**`IsApproved *bool`** — a pointer, not a bool. This is a deliberate three-state design:
- `nil` = waiting for approval (workflow paused)
- `true` = approved (workflow resumes to execute tools)
- `false` = rejected (workflow routes to response node)

A regular `bool` cannot represent "not yet decided." The pointer makes the third state explicit and compiler-enforced.

**`ShouldStop bool`** — set by either the `observe` node (when `Iteration >= MaxIterations`) or the `response` node. The routing functions check this to decide when to terminate the loop.

The state also implements helpful mutation methods:

```go
func (s *AgentState) AddAssistantMessage(content string, toolCalls []toolspec.ToolCall) {
    msg := port.ChatMessage{Role: "assistant", Content: content}
    if len(toolCalls) > 0 {
        msg.ToolCalls = toolCalls
    }
    s.Messages = append(s.Messages, msg)
}

func (s *AgentState) AddToolResult(toolCallID, name string, result any) {
    content, _ := json.Marshal(result)
    s.ToolResults = append(s.ToolResults, port.ChatMessage{
        Role: "tool", Content: string(content),
        ToolCallID: toolCallID, Name: name,
    })
}

func (s *AgentState) FlushToolResults() {
    s.Messages = append(s.Messages, s.ToolResults...)
    s.ToolResults = []port.ChatMessage{}
}
```

These methods enforce consistent state transitions. Any node that needs to add messages goes through these methods — there is no ad-hoc `append` scattered across nodes.

---

## The Graph Structure

Here is the full workflow we are building, exactly as it appears in the source code's ASCII diagram:

```
    ┌─────────┐
    │  START  │
    └────┬────┘
         │
    ┌────▼────┐
    │  Router │   (selects agent type + system prompt + model)
    └────┬────┘
         │
    ┌────▼────┐
    │  Think  │   (calls the LLM, gets response + tool calls)
    └────┬────┘
    ┌────┴─────┬──────────┐
    │          │          │
┌───▼───┐ ┌───▼────┐ ┌───▼────┐
│  Act  │ │Approve │ │Response│
└───┬───┘ └────┬───┘ └───┬────┘
    │          │          │
┌───▼───┐      │          │
│Observe│      │          │
└───┬───┘      │          │
    │          │          │
    └──────────┴──────────┘
                │
           ┌────▼────┐
           │   END   │
           └─────────┘
```

There are 6 nodes: `router`, `think`, `act`, `observe`, `human_approval`, `response`.

The key insight: **`think` → `act` → `observe` is a loop**. After `observe`, the routing function checks whether to loop back to `think` (more tool calls needed) or go to `response` (done). This is the ReAct loop — the core of any tool-using agent.

---

## Building the Graph: AddLambdaNode + compose.InvokableLambda

The graph is built in `Build(ctx context.Context)` in `internal/infrastructure/eino/graphs/chatbot_graph.go`. Here is how you add nodes:

```go
func (g *ChatbotGraph) Build(ctx context.Context) (compose.Runnable[*state.AgentState, *state.AgentState], error) {
    graph := compose.NewGraph[*state.AgentState, *state.AgentState]()

    // Add nodes using AddLambdaNode + compose.InvokableLambda
    if err := graph.AddLambdaNode("router", compose.InvokableLambda(g.routerNode)); err != nil {
        return nil, fmt.Errorf("add router node: %w", err)
    }
    if err := graph.AddLambdaNode("think", compose.InvokableLambda(g.thinkNode)); err != nil {
        return nil, fmt.Errorf("add think node: %w", err)
    }
    if err := graph.AddLambdaNode("act", compose.InvokableLambda(g.actNode)); err != nil {
        return nil, fmt.Errorf("add act node: %w", err)
    }
    if err := graph.AddLambdaNode("observe", compose.InvokableLambda(g.observeNode)); err != nil {
        return nil, fmt.Errorf("add observe node: %w", err)
    }
    if err := graph.AddLambdaNode("human_approval", compose.InvokableLambda(g.humanApprovalNode)); err != nil {
        return nil, fmt.Errorf("add human_approval node: %w", err)
    }
    if err := graph.AddLambdaNode("response", compose.InvokableLambda(g.responseNode)); err != nil {
        return nil, fmt.Errorf("add response node: %w", err)
    }
```

`compose.InvokableLambda()` wraps a Go function with the signature `func(ctx context.Context, in I) (O, error)` into an Eino-compatible node. Because our graph is typed as `*state.AgentState → *state.AgentState`, every node function has the signature:

```go
func (g *ChatbotGraph) routerNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error)
```

The compiler enforces this. If you accidentally return a different type, it fails at compile time.

---

## Edges and Branches

Simple edges connect nodes unconditionally:

```go
// START → router (always)
graph.AddEdge(compose.START, "router")

// router → think (always, after routing decision is stored in state)
graph.AddEdge("router", "think")

// act → observe (always)
graph.AddEdge("act", "observe")

// response → END (always)
graph.AddEdge("response", compose.END)
```

Conditional branches use `compose.NewGraphBranch()`:

```go
// think → one of: act, response, human_approval
thinkBranch := compose.NewGraphBranch(g.routeAfterThink, map[string]bool{
    "act":            true,
    "response":       true,
    "human_approval": true,
})
graph.AddBranch("think", thinkBranch)
```

The routing function receives the current state and returns the name of the next node:

```go
func (g *ChatbotGraph) routeAfterThink(_ context.Context, s *state.AgentState) (string, error) {
    if len(s.PendingTools) > 0 {
        return "act", nil       // LLM wants to call tools
    }
    if s.RequiresApproval {
        return "human_approval", nil  // A tool needs human sign-off
    }
    return "response", nil      // LLM gave a final answer
}
```

The map `map[string]bool{"act": true, "response": true, "human_approval": true}` tells Eino which node names are valid return values. This is a static declaration that enables the graph compiler to validate routes at compile time rather than at runtime.

---

## Node Implementations

### Router Node: Keyword-Based Agent Selection

```go
func (g *ChatbotGraph) routerNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
    input := s.UserInput
    s.CurrentAgent = "general"

    codeKeywords := []string{"code", "function", "program", "debug", "fix", "implement", "class", "api"}
    for _, kw := range codeKeywords {
        if containsIgnoreCase(input, kw) {
            s.CurrentAgent = "coder"
            break
        }
    }

    researchKeywords := []string{"search", "find", "research", "what is", "who is", "latest", "news", "current"}
    for _, kw := range researchKeywords {
        if containsIgnoreCase(input, kw) {
            s.CurrentAgent = "researcher"
            break
        }
    }

    s.AgentHistory = append(s.AgentHistory, s.CurrentAgent)

    if model, ok := g.modelConfig[s.CurrentAgent]; ok {
        s.Model = model   // "coder" → "gpt-4o", others → "gpt-4o-mini"
    }
    if prompt, ok := g.systemPrompts[s.CurrentAgent]; ok {
        s.SystemPrompt = prompt
    }

    return s, nil
}
```

The router selects between three agent personas: `general`, `coder`, `researcher`. Each gets a different system prompt and model. "coder" gets `gpt-4o` because code generation benefits from the more capable model. "general" and "researcher" get `gpt-4o-mini` for cost efficiency.

This is a simple keyword router. In a production system, you might replace this with a small LLM call that classifies intent more accurately. The graph structure does not change — just the router node implementation.

### Think Node: The LLM Call

```go
func (g *ChatbotGraph) thinkNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
    messages := g.buildMessages(s)  // system prompt + conversation history + user input
    tools := g.toolRegistry.List()  // all available tools in OpenAI format
    s.AvailableTools = tools

    resp, err := g.llm.Chat(ctx, port.ChatRequest{
        Model:       s.Model,
        Messages:    messages,
        Tools:       tools,
        Temperature: s.Temperature,
        MaxTokens:   s.MaxTokens,
    })
    if err != nil {
        s.Error = err.Error()
        return s, err
    }

    choice := resp.Choices[0]
    s.TokensUsed += resp.Usage.TotalTokens
    s.CurrentOutput = choice.Message.Content
    s.PendingTools = choice.Message.ToolCalls  // If LLM called tools, they land here

    s.AddAssistantMessage(choice.Message.Content, choice.Message.ToolCalls)

    return s, nil
}
```

The think node calls the LLM with the full conversation history and tool definitions. The LLM responds with either:
- A text answer (no tool calls) → `PendingTools` is empty → routing goes to `response`
- Tool call requests → `PendingTools` is populated → routing goes to `act`

### Act Node: Tool Execution

```go
func (g *ChatbotGraph) actNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
    s.ToolResults = []port.ChatMessage{}

    for _, tc := range s.PendingTools {
        args, err := tc.ParseArguments()
        if err != nil {
            s.AddToolResult(tc.ID, tc.Function.Name, map[string]string{"error": "failed to parse arguments"})
            continue
        }

        t, ok := g.toolRegistry.Get(tc.Function.Name)
        if !ok {
            s.AddToolResult(tc.ID, tc.Function.Name, map[string]string{"error": "tool not found"})
            continue
        }

        // Check if this tool needs human approval before running
        if t.RequiresApproval() {
            s.RequiresApproval = true
            s.ApprovalReason = fmt.Sprintf("Tool '%s' requires approval", tc.Function.Name)
            s.ApprovalData = map[string]any{
                "tool": tc.Function.Name, "arguments": args, "tool_call": tc,
            }
            return s, nil  // Pause here
        }

        result, err := t.Execute(ctx, args)
        if err != nil {
            s.AddToolResult(tc.ID, tc.Function.Name, map[string]string{"error": err.Error()})
        } else {
            s.AddToolResult(tc.ID, tc.Function.Name, result)
        }
        s.ToolCallsCount++
    }

    s.PendingTools = nil
    return s, nil
}
```

The act node is where tools run. For each pending tool call from the LLM:
1. Parse the JSON arguments
2. Look up the tool in the registry
3. Check if it requires human approval (if yes, pause)
4. Execute and store the result

### Observe Node: Loop Control

```go
func (g *ChatbotGraph) observeNode(_ context.Context, s *state.AgentState) (*state.AgentState, error) {
    s.FlushToolResults()  // Move tool results into Messages for the next LLM call
    s.Iteration++

    if s.Iteration >= s.MaxIterations {
        s.ShouldStop = true  // Prevent infinite loops
    }

    return s, nil
}
```

The observe node does two things: moves tool results into the message history (so the LLM can see them next iteration), and increments the iteration counter. If we hit `MaxIterations` (default: 10), we force a stop. This prevents runaway agent loops that would rack up LLM costs.

### The Observe Branch: Loop or Stop?

```go
func (g *ChatbotGraph) routeAfterObserve(_ context.Context, s *state.AgentState) (string, error) {
    if s.ShouldStop {
        return "response", nil    // Force a final response
    }
    if s.RequiresApproval {
        return "human_approval", nil  // Pause for approval
    }
    return "think", nil           // Loop back for another LLM call with tool results
}
```

When the graph returns to `think` after `observe`, the LLM now sees the tool results in its message history. It can either call more tools or give a final answer with the information it gathered.

---

## Compiling and Running the Graph

After defining all nodes and edges, compile the graph:

```go
return graph.Compile(ctx, compose.WithGraphName("chatbot"))
```

`Compile()` validates the graph structure — every branch target must be a registered node, there must be a path from START to END, etc. It returns a `compose.Runnable[*state.AgentState, *state.AgentState]`:

```go
type Runnable[I, O any] interface {
    Invoke(ctx context.Context, input I, opts ...Option) (O, error)
    Stream(ctx context.Context, input I, opts ...Option) (*StreamReader[O], error)
}
```

To run the agent:

```go
runnable, _ := chatbotGraph.Build(ctx)

initialState := state.NewAgentState(convID, userID, "What is the weather in Tokyo?")
finalState, err := runnable.Invoke(ctx, initialState)

fmt.Println(finalState.CurrentOutput) // The agent's final response
fmt.Println(finalState.TokensUsed)    // Total tokens consumed
fmt.Println(finalState.ToolCallsCount) // How many tools were called
```

The `Invoke()` call runs the full graph synchronously. For streaming, you would use `Stream()` to get a `StreamReader` that emits state updates as they occur.

---

## A Complete ReAct Loop Traced

Let us trace what happens when a user asks "What is 15% of 847?":

1. **START → Router**: Input contains no code or research keywords → `s.CurrentAgent = "general"`, model = `"gpt-4o-mini"`

2. **Router → Think**: LLM receives the question with `calculator` tool available. Responds with a tool call: `{"name": "calculator", "arguments": "{\"expression\": \"847 * 0.15\"}"}`

3. **Think → Act** (routeAfterThink: `PendingTools` is not empty): Act node parses arguments, calls `calculator.Execute(ctx, {"expression": "847 * 0.15"})`, gets `{"expression": "847 * 0.15", "result": 127.05}`, stores in `ToolResults`

4. **Act → Observe**: Observe moves result from `ToolResults` into `Messages`. Iteration = 1. `ShouldStop = false`.

5. **Observe → Think** (routeAfterObserve: not stopped, no approval needed): LLM now sees the tool result in its message history. Responds: "15% of 847 is 127.05"

6. **Think → Response** (routeAfterThink: `PendingTools` is empty): Response node stores `s.CurrentOutput = "15% of 847 is 127.05"`, sets `ShouldStop = true`

7. **Response → END**: Final state returned. `finalState.CurrentOutput` contains the answer. Total: 2 LLM calls, 1 tool execution, 2 graph iterations.

---

## Why This Architecture Works

**Testability.** Every node is a pure function: `(*AgentState) → (*AgentState, error)`. You can unit test `actNode` by constructing an `AgentState` with specific `PendingTools` and asserting the state after execution — no mocking of the graph itself needed.

**Observability.** The complete state after each node is available. You can log `s.TokensUsed`, `s.ToolCallsCount`, `s.Iteration`, and `s.AgentHistory` at the end of every request.

**Safety.** The `MaxIterations` counter prevents infinite loops. The `RequiresApproval` check prevents unauthorized tool execution. The compile-time graph validation prevents invalid routing.

**Extensibility.** Add a new node by implementing a function with the right signature and calling `graph.AddLambdaNode()`. Add a new agent persona by adding an entry to `systemPrompts` and `modelConfig`. The graph structure handles the routing.

---

## What We Just Learned

- Eino is ByteDance's Go LangGraph equivalent — type-safe, compiled, single-binary
- `AgentState` is a single struct carrying all workflow state, passed by pointer through every node
- `IsApproved *bool` uses pointer semantics to represent three states (nil/true/false)
- `compose.InvokableLambda()` wraps any `func(ctx, in) (out, error)` into an Eino node
- `compose.NewGraphBranch()` with a routing function handles conditional edges
- The think→act→observe loop is the ReAct pattern: reason, act, observe, repeat
- `MaxIterations` prevents runaway loops; `ShouldStop` signals the graph to terminate

---

*Next in the series: [Article 5 — JWT Meets API Keys: Building Dual-Auth in Go](#)*

*Previous: [Article 3 — Add a New AI Tool in 15 Lines of Go](#)*
