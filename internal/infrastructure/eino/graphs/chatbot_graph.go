package graphs

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"

	"github.com/wyuneed/go-agent-api/internal/application/port"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/tool"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/eino/state"
)

/*
Workflow Graph:

    ┌─────────┐
    │  START  │
    └────┬────┘
         │
    ┌────▼────┐
    │  Router │
    └────┬────┘
         │
    ┌────▼────┐
    │  Think  │
    └────┬────┘
    ┌────┴─────┬──────────┐
    │          │          │
┌───▼───┐ ┌────▼────┐ ┌───▼────┐
│  Act  │ │ Approve │ │Response│
└───┬───┘ └────┬────┘ └───┬────┘
    │          │          │
┌───▼───┐     │          │
│Observe│     │          │
└───┬───┘     │          │
    │          │          │
    └──────────┴──────────┘
                │
           ┌────▼────┐
           │   END   │
           └─────────┘
*/

type ChatbotGraph struct {
	llm           port.LLMProvider
	toolRegistry  *tool.ToolRegistry
	runnable      compose.Runnable[*state.AgentState, *state.AgentState]
	systemPrompts map[string]string
	modelConfig   map[string]string
}

func NewChatbotGraph(llm port.LLMProvider, registry *tool.ToolRegistry) *ChatbotGraph {
	return &ChatbotGraph{
		llm:          llm,
		toolRegistry: registry,
		systemPrompts: map[string]string{
			"general": `You are a helpful AI assistant. Be concise, accurate, and helpful.
When you need current information, use the web_search tool.
When asked to calculate something, use the calculator tool.`,
			"coder": `You are an expert programmer. Write clean, efficient, well-documented code.
Always explain your code and consider edge cases.
Use appropriate tools to validate or test code when available.`,
			"researcher": `You are a research assistant. Search for accurate, up-to-date information.
Always cite your sources and verify claims using the web_search tool.
Present balanced perspectives on controversial topics.`,
		},
		modelConfig: map[string]string{
			"general":    "gpt-4o-mini",
			"coder":      "gpt-4o",
			"researcher": "gpt-4o-mini",
		},
	}
}

// BuildRunnable compiles the Eino graph and stores the runnable for repeated use.
// Must be called once at startup before Run or Resume.
func (g *ChatbotGraph) BuildRunnable(ctx context.Context) error {
	runnable, err := g.build(ctx)
	if err != nil {
		return err
	}
	g.runnable = runnable
	return nil
}

// build constructs and compiles the Eino workflow graph.
func (g *ChatbotGraph) build(ctx context.Context) (compose.Runnable[*state.AgentState, *state.AgentState], error) {
	graph := compose.NewGraph[*state.AgentState, *state.AgentState]()

	// Add nodes
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

	// START → router
	if err := graph.AddEdge(compose.START, "router"); err != nil {
		return nil, fmt.Errorf("add start edge: %w", err)
	}

	// router → think
	if err := graph.AddEdge("router", "think"); err != nil {
		return nil, fmt.Errorf("add router->think edge: %w", err)
	}

	// think → branch (act | response | human_approval)
	thinkBranch := compose.NewGraphBranch(g.routeAfterThink, map[string]bool{
		"act":            true,
		"response":       true,
		"human_approval": true,
	})
	if err := graph.AddBranch("think", thinkBranch); err != nil {
		return nil, fmt.Errorf("add think branch: %w", err)
	}

	// act → observe
	if err := graph.AddEdge("act", "observe"); err != nil {
		return nil, fmt.Errorf("add act->observe edge: %w", err)
	}

	// observe → branch (think | response | human_approval)
	observeBranch := compose.NewGraphBranch(g.routeAfterObserve, map[string]bool{
		"think":          true,
		"response":       true,
		"human_approval": true,
	})
	if err := graph.AddBranch("observe", observeBranch); err != nil {
		return nil, fmt.Errorf("add observe branch: %w", err)
	}

	// human_approval → branch (act | response | END)
	approvalBranch := compose.NewGraphBranch(g.routeAfterApproval, map[string]bool{
		"act":         true,
		"response":    true,
		compose.END:   true,
	})
	if err := graph.AddBranch("human_approval", approvalBranch); err != nil {
		return nil, fmt.Errorf("add approval branch: %w", err)
	}

	// response → END
	if err := graph.AddEdge("response", compose.END); err != nil {
		return nil, fmt.Errorf("add response->end edge: %w", err)
	}

	return graph.Compile(ctx, compose.WithGraphName("chatbot"))
}

// ============ WorkflowEngine INTERFACE IMPLEMENTATION ============

// Run executes the full workflow for a new user turn.
func (g *ChatbotGraph) Run(ctx context.Context, input port.WorkflowInput) (*port.WorkflowOutput, error) {
	if g.runnable == nil {
		return nil, fmt.Errorf("graph not built: call BuildRunnable first")
	}

	s := state.NewAgentState(input.ConversationID, input.UserID, input.Content)
	s.Messages = input.Messages
	if input.Model != "" {
		s.Model = input.Model
	}
	if input.Temperature > 0 {
		s.Temperature = input.Temperature
	}

	finalState, err := g.runnable.Invoke(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("graph invoke: %w", err)
	}

	out := &port.WorkflowOutput{
		Response:       finalState.CurrentOutput,
		TokensUsed:     finalState.TokensUsed,
		ToolCallsCount: finalState.ToolCallsCount,
		AgentType:      finalState.CurrentAgent,
	}

	if finalState.RequiresApproval {
		stateJSON, err := finalState.ToJSON()
		if err != nil {
			return nil, fmt.Errorf("serialise paused state: %w", err)
		}
		out.RequiresApproval = true
		out.ApprovalReason = finalState.ApprovalReason
		out.PausedStateJSON = stateJSON
	}

	return out, nil
}

// Resume continues a workflow paused at human_approval.
// It manually executes actNode → observeNode and then re-invokes the graph
// so the remaining think→response turns complete normally.
func (g *ChatbotGraph) Resume(ctx context.Context, pausedStateJSON []byte, approved bool) (*port.WorkflowOutput, error) {
	if g.runnable == nil {
		return nil, fmt.Errorf("graph not built: call BuildRunnable first")
	}

	s, err := state.FromJSON(pausedStateJSON)
	if err != nil {
		return nil, fmt.Errorf("deserialise paused state: %w", err)
	}

	s.IsApproved = &approved
	s.RequiresApproval = false

	if !approved {
		// Rejected: synthesise a response message and stop.
		if s.CurrentOutput == "" {
			s.CurrentOutput = "The requested action was rejected."
		}
		return &port.WorkflowOutput{
			Response:       s.CurrentOutput,
			TokensUsed:     s.TokensUsed,
			ToolCallsCount: s.ToolCallsCount,
			AgentType:      s.CurrentAgent,
		}, nil
	}

	// Approved: execute the pending tools directly, then continue via the graph.
	s, err = g.actNode(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("resume act: %w", err)
	}
	s, err = g.observeNode(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("resume observe: %w", err)
	}

	// Clear approval flags so the graph continues normally from think/response.
	s.IsApproved = nil
	s.RequiresApproval = false

	finalState, err := g.runnable.Invoke(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("resume graph invoke: %w", err)
	}

	out := &port.WorkflowOutput{
		Response:       finalState.CurrentOutput,
		TokensUsed:     finalState.TokensUsed,
		ToolCallsCount: finalState.ToolCallsCount,
		AgentType:      finalState.CurrentAgent,
	}

	if finalState.RequiresApproval {
		stateJSON, err := finalState.ToJSON()
		if err != nil {
			return nil, fmt.Errorf("serialise paused state: %w", err)
		}
		out.RequiresApproval = true
		out.ApprovalReason = finalState.ApprovalReason
		out.PausedStateJSON = stateJSON
	}

	return out, nil
}

// ============ NODE IMPLEMENTATIONS ============

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

	if model, ok := g.modelConfig[s.CurrentAgent]; ok && s.Model == "" {
		s.Model = model
	}
	if prompt, ok := g.systemPrompts[s.CurrentAgent]; ok {
		s.SystemPrompt = prompt
	}

	return s, nil
}

func (g *ChatbotGraph) thinkNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
	messages := g.buildMessages(s)
	tools := g.toolRegistry.List()
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
	s.PendingTools = choice.Message.ToolCalls
	s.AddAssistantMessage(choice.Message.Content, choice.Message.ToolCalls)

	return s, nil
}

func (g *ChatbotGraph) actNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
	s.ToolResults = []port.ChatMessage{}

	for _, tc := range s.PendingTools {
		args, err := tc.ParseArguments()
		if err != nil {
			s.AddToolResult(tc.ID, tc.Function.Name, map[string]string{
				"error": fmt.Sprintf("failed to parse arguments: %v", err),
			})
			continue
		}

		t, ok := g.toolRegistry.Get(tc.Function.Name)
		if !ok {
			s.AddToolResult(tc.ID, tc.Function.Name, map[string]string{
				"error": fmt.Sprintf("tool not found: %s", tc.Function.Name),
			})
			continue
		}

		if t.RequiresApproval() {
			s.RequiresApproval = true
			s.ApprovalReason = fmt.Sprintf("Tool '%s' requires approval", tc.Function.Name)
			s.ApprovalData = map[string]any{
				"tool":      tc.Function.Name,
				"arguments": args,
				"tool_call": tc,
			}
			return s, nil
		}

		result, err := t.Execute(ctx, args)
		if err != nil {
			s.AddToolResult(tc.ID, tc.Function.Name, map[string]string{
				"error": err.Error(),
			})
		} else {
			s.AddToolResult(tc.ID, tc.Function.Name, result)
		}

		s.ToolCallsCount++
	}

	s.PendingTools = nil
	return s, nil
}

func (g *ChatbotGraph) observeNode(_ context.Context, s *state.AgentState) (*state.AgentState, error) {
	s.FlushToolResults()
	s.Iteration++

	if s.Iteration >= s.MaxIterations {
		s.ShouldStop = true
	}

	return s, nil
}

func (g *ChatbotGraph) humanApprovalNode(_ context.Context, s *state.AgentState) (*state.AgentState, error) {
	// This node pauses the workflow. State will be persisted and
	// workflow resumed via the approve API endpoint.
	return s, nil
}

func (g *ChatbotGraph) responseNode(_ context.Context, s *state.AgentState) (*state.AgentState, error) {
	if s.CurrentOutput == "" && len(s.Messages) > 0 {
		for i := len(s.Messages) - 1; i >= 0; i-- {
			if s.Messages[i].Role == "assistant" && s.Messages[i].Content != "" {
				s.CurrentOutput = s.Messages[i].Content
				break
			}
		}
	}

	s.ShouldStop = true
	return s, nil
}

// ============ ROUTING FUNCTIONS ============

func (g *ChatbotGraph) routeAfterThink(_ context.Context, s *state.AgentState) (string, error) {
	if len(s.PendingTools) > 0 {
		return "act", nil
	}
	if s.RequiresApproval {
		return "human_approval", nil
	}
	return "response", nil
}

func (g *ChatbotGraph) routeAfterObserve(_ context.Context, s *state.AgentState) (string, error) {
	if s.ShouldStop {
		return "response", nil
	}
	if s.RequiresApproval {
		return "human_approval", nil
	}
	return "think", nil
}

func (g *ChatbotGraph) routeAfterApproval(_ context.Context, s *state.AgentState) (string, error) {
	if s.IsApproved == nil {
		return compose.END, nil
	}
	if *s.IsApproved {
		return "act", nil
	}
	return "response", nil
}

// ============ HELPERS ============

func (g *ChatbotGraph) buildMessages(s *state.AgentState) []port.ChatMessage {
	messages := []port.ChatMessage{}

	if s.SystemPrompt != "" {
		messages = append(messages, port.ChatMessage{
			Role:    "system",
			Content: s.SystemPrompt,
		})
	}

	messages = append(messages, s.Messages...)

	if s.UserInput != "" && (len(s.Messages) == 0 || s.Messages[len(s.Messages)-1].Content != s.UserInput) {
		messages = append(messages, port.ChatMessage{
			Role:    "user",
			Content: s.UserInput,
		})
	}

	return messages
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
