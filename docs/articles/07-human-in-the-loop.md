# Human-in-the-Loop AI: How to Pause, Approve, and Resume an Agentic Workflow

*Part 7 of the "Building Production-Ready AI Agent APIs in Go" series*

---

The hardest problem in deploying AI agents is not intelligence — it is control.

When an AI agent can take real actions in the world (send an email, modify a database record, make a purchase, delete a file), you often need a human to review and approve before the action runs. The agent is not wrong to propose the action — but the consequences are irreversible, and you want a human in the approval chain.

This is called **human-in-the-loop** (HITL). Implementing it correctly requires:
1. The workflow pausing mid-execution without losing state
2. The paused state being persisted (so a restart does not lose it)
3. An API endpoint for the human to approve or reject
4. The workflow resuming from exactly where it stopped

All four are implemented in this project. Let me show you how.

---

## The Problem: Irreversible Tool Actions

Consider a hypothetical `DeleteFileTool`. If the AI agent calls this tool and it runs without review, files are gone. Even if the AI was right, users need to feel in control.

The standard approach in this codebase: any tool can mark itself as requiring approval by overriding one method.

```go
// Default: tool runs immediately
type BaseTool struct{}
func (BaseTool) RequiresApproval() bool { return false }

// Override to require approval
type DeleteFileTool struct {
    tool.BaseTool
}

func (t *DeleteFileTool) RequiresApproval() bool {
    return true  // This is the ENTIRE change needed to pause the workflow
}
```

That single method override triggers the entire approval flow. Everything else — pausing the workflow, persisting state, resuming on approval — is handled automatically.

---

## Step 1: The actNode Detects RequiresApproval

In `internal/infrastructure/eino/graphs/chatbot_graph.go`, the `actNode` processes each tool call. Before executing, it checks `RequiresApproval()`:

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

        // THE APPROVAL CHECK
        if t.RequiresApproval() {
            s.RequiresApproval = true
            s.ApprovalReason = fmt.Sprintf("Tool '%s' requires approval", tc.Function.Name)
            s.ApprovalData = map[string]any{
                "tool":      tc.Function.Name,
                "arguments": args,
                "tool_call": tc,
            }
            return s, nil  // Return immediately — do not execute this tool
        }

        // Normal execution
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

When `RequiresApproval()` returns true:
1. `s.RequiresApproval = true` — signals the routing function
2. `s.ApprovalData` — stores exactly what was going to be executed (for display to the human)
3. `return s, nil` immediately — the loop stops; pending tools are NOT cleared

The routing function after `actNode` sends the graph to `human_approval`:

```go
// Actually, the routing happens after think, not act.
// The actNode sets RequiresApproval, then returns.
// The routing after observe checks it:
func (g *ChatbotGraph) routeAfterObserve(_ context.Context, s *state.AgentState) (string, error) {
    if s.ShouldStop {
        return "response", nil
    }
    if s.RequiresApproval {
        return "human_approval", nil  // ← Goes here
    }
    return "think", nil
}
```

---

## Step 2: The humanApprovalNode Pauses the Workflow

```go
func (g *ChatbotGraph) humanApprovalNode(_ context.Context, s *state.AgentState) (*state.AgentState, error) {
    // This node pauses the workflow. State will be persisted and
    // workflow resumed via the approve API endpoint.
    return s, nil
}
```

This function does nothing. It returns immediately. But its routing function determines what happens next:

```go
func (g *ChatbotGraph) routeAfterApproval(_ context.Context, s *state.AgentState) (string, error) {
    if s.IsApproved == nil {
        return compose.END, nil  // No decision yet → end this graph run
    }
    if *s.IsApproved {
        return "act", nil     // Approved → resume tool execution
    }
    return "response", nil    // Rejected → generate rejection response
}
```

**`IsApproved *bool`** — a pointer to bool. This is the core of the three-state design:

| Value | Meaning |
|-------|---------|
| `nil` | No decision yet (first time through) |
| `true` | Human approved |
| `false` | Human rejected |

When the workflow first reaches `humanApprovalNode`, `IsApproved` is `nil`. The routing returns `compose.END` — the graph run terminates here.

The entire `AgentState` at this point (including the pending tool calls that weren't executed) is serialized and stored in the database. The HTTP response to the client contains the conversation's `pending_approval` status, telling the UI to ask for user input.

---

## Step 3: Conversation.RequestApproval() — Storing the Pause State

In the `SendMessage` use case, after the graph run ends with `RequiresApproval == true`, the conversation status is updated:

```go
// After graph.Invoke() returns with s.RequiresApproval == true:
conv.RequestApproval(s.CurrentNode, s.ApprovalData)
```

From `internal/domain/entity/conversation.go`:

```go
func (c *Conversation) RequestApproval(node string, data map[string]any) {
    c.Status = ConversationStatusPending  // "pending_approval"
    c.CurrentNode = node                 // Which Eino node triggered this
    c.Metadata["pending_approval"] = data // What data the human needs to review
    c.UpdatedAt = time.Now()
}
```

After calling `RequestApproval()` and saving the conversation to the database, the response to the client includes:

```json
{
    "conversation": {
        "id": "...",
        "status": "pending_approval",
        "metadata": {
            "pending_approval": {
                "tool": "delete_file",
                "arguments": {"path": "/important/data.csv"},
                "tool_call": {...}
            }
        }
    }
}
```

The UI reads `status: "pending_approval"` and shows the human a dialog: "The AI wants to delete /important/data.csv. Approve?"

---

## Step 4: The ApproveActionUseCase — Validating and Transitioning

When the human clicks Approve or Reject, they hit `POST /v1/conversations/{id}/approve`. The use case in `internal/application/usecase/chat/approve_action.go`:

```go
func (uc *ApproveActionUseCase) Execute(ctx context.Context, input ApproveInput) (*ApproveOutput, error) {
    conv, err := uc.convRepo.FindByID(ctx, input.ConversationID)
    if err != nil {
        return nil, fmt.Errorf("conversation not found: %w", err)
    }

    // Authorization: only the conversation owner can approve
    if conv.UserID != input.UserID {
        return nil, fmt.Errorf("unauthorized access to conversation")
    }

    // Guard: must actually be waiting for approval
    if !conv.IsPendingApproval() {
        return nil, fmt.Errorf("conversation is not pending approval")
    }

    // Domain method handles the state transition
    if input.Approved {
        conv.Approve()   // Status → "active", clears pending_approval metadata
    } else {
        conv.Reject(input.Reason)  // Status → "active", records rejection reason
    }

    // Persist the updated conversation
    if err := uc.convRepo.Update(ctx, conv); err != nil {
        return nil, fmt.Errorf("update conversation: %w", err)
    }

    return &ApproveOutput{Conversation: conv, Status: "approved"}, nil
}
```

Three validations happen in sequence:
1. **Does the conversation exist?** If not, 404
2. **Does the user own it?** If not, 403 (prevents user A from approving user B's conversation)
3. **Is it actually waiting for approval?** If not, 400 (prevents double-approval)

Then the domain entity handles the transition: `conv.Approve()` or `conv.Reject(reason)`. These methods clean up the `pending_approval` metadata and update the status.

---

## Step 5: Resuming the Workflow

After the human approves, the next message from the client (or the approval response itself) needs to resume the workflow. The workflow is resumed by loading the saved `AgentState` from the conversation's `WorkflowState` field, setting `IsApproved = &true`, and running the graph again.

The key routing logic that makes resumption work:

```go
func (g *ChatbotGraph) routeAfterApproval(_ context.Context, s *state.AgentState) (string, error) {
    if s.IsApproved == nil {
        return compose.END, nil   // First run: no decision → stop
    }
    if *s.IsApproved {
        return "act", nil         // Resumed with approval → execute tools
    }
    return "response", nil        // Resumed with rejection → respond to user
}
```

When the graph is resumed with `IsApproved = &true`:
1. START → router → think → act (but wait — we are resuming)
2. Actually: the saved state is loaded with `PendingTools` still populated (they were not cleared on the first run)
3. The graph is re-entered, `IsApproved` is already set to `&true`
4. `humanApprovalNode` runs, `routeAfterApproval` returns `"act"`
5. `actNode` executes the pending tools
6. Workflow continues normally

The fact that `PendingTools` were not cleared when `RequiresApproval` was detected is intentional. The pending tool calls are preserved in the serialized state, ready to be executed when the human approves.

---

## The Approval Handler

The HTTP handler in `internal/infrastructure/http/handler/chat_handler.go`:

```go
// ApproveAction godoc
// @Summary      Approve or reject a pending action
// @Description  Resumes a conversation that is paused waiting for human-in-the-loop approval
// @Tags         conversations
// @Param        id    path  string               true "Conversation ID (UUID)"
// @Param        body  body  request.ApprovalRequest  true "Approval decision"
// @Router       /v1/conversations/{id}/approve [post]
func (h *ChatHandler) ApproveAction(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())

    convID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
        return
    }

    var req request.ApprovalRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
        return
    }

    result, err := h.approveAction.Execute(r.Context(), chat.ApproveInput{
        ConversationID: convID,
        UserID:         user.ID,
        Approved:       req.Approved,
        Reason:         req.Reason,
    })
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "APPROVAL_ERROR", err.Error())
        return
    }

    response.JSON(w, http.StatusOK, result)
}
```

The request body:

```json
{
    "approved": true,
    "reason": ""
}
```

Or to reject:

```json
{
    "approved": false,
    "reason": "I don't want to delete that file"
}
```

---

## The Complete Flow, End to End

Let us trace the complete human-in-the-loop cycle:

```
1. User: "Delete the old backup file"
   POST /v1/conversations/{id}/messages
   {"message": "Delete the old backup file"}

2. Workflow runs: router → think → act
   LLM calls delete_file tool
   actNode: t.RequiresApproval() == true
   AgentState: RequiresApproval = true, ApprovalData = {tool: "delete_file", args: {path: "/backup.csv"}}
   humanApprovalNode: IsApproved == nil → routeAfterApproval returns END
   Graph terminates

3. Server: conv.RequestApproval("act", s.ApprovalData)
   conv.Status = "pending_approval"
   Saves conversation + AgentState to DB
   Response: {conversation: {status: "pending_approval", metadata: {pending_approval: {...}}}}

4. UI: shows "The AI wants to delete /backup.csv — Approve?"
   User clicks Approve
   POST /v1/conversations/{id}/approve
   {"approved": true}

5. ApproveActionUseCase:
   - Verifies user owns conversation
   - Verifies status is "pending_approval"
   - conv.Approve() → status = "active"
   - Saves conversation
   Response: {status: "approved"}

6. Workflow resumes with IsApproved = &true
   humanApprovalNode: IsApproved = &true → routeAfterApproval returns "act"
   actNode: executes delete_file
   File deleted, result added to messages
   observe → think → response
   Final answer: "I've deleted /backup.csv"

7. User sees: "I've deleted /backup.csv"
```

The user was in control throughout. The AI proposed, the human decided, the AI executed.

---

## Why *bool Instead of bool

The `IsApproved *bool` field is worth dwelling on. Why not just `IsApproved bool`?

With `bool`:
- `false` means "not approved" OR "not yet decided" — ambiguous
- You need a second field like `ApprovalDecisionMade bool` to distinguish "no decision" from "rejected"

With `*bool`:
- `nil` = "no decision yet" — the workflow just reached this node for the first time
- `true` = "approved" — human said yes
- `false` = "rejected" — human said no

This is a fundamental Go idiom: **use pointer types to add a null/absent state**. It appears repeatedly in the codebase wherever three-value semantics are needed (`LastUsedAt *time.Time`, `CompletedAt *time.Time` on conversations).

The routing function explicitly checks for `nil`:
```go
if s.IsApproved == nil {
    return compose.END, nil
}
if *s.IsApproved {
    return "act", nil
}
return "response", nil
```

This is safer than `if !s.IsApproved` — you cannot accidentally dereference a nil pointer because you check for nil first.

---

## What We Just Learned

- Any tool marks itself as requiring approval with a single method: `func (t *YourTool) RequiresApproval() bool { return true }`
- The Eino `actNode` checks `RequiresApproval()` before executing — if true, it sets `RequiresApproval = true` on `AgentState` and returns without executing
- `IsApproved *bool` uses pointer semantics for three states: nil (waiting), true (approved), false (rejected)
- `Conversation.RequestApproval()` persists the pause state to the DB
- `ApproveActionUseCase` validates ownership and uses domain methods (`conv.Approve()` / `conv.Reject()`) for state transitions
- The workflow resumes by loading saved state and re-running the graph with `IsApproved` set

---

*Next in the series: [Article 8 — Shipping to Production: 18MB Docker Image, 5 Services, Zero Downtime](#)*

*Previous: [Article 6 — Server-Sent Events in Go](#)*
