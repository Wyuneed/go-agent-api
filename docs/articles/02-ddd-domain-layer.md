# Domain-Driven Design for Go Developers: Build Entities That Actually Enforce Business Rules

*Part 2 of the "Building Production-Ready AI Agent APIs in Go" series*

---

Here is a question to consider: where does the rule "an expired token cannot be used" live in your codebase?

In many Go applications, that check exists in three or four different places — a middleware function, a repository method, a handler guard. When the rule changes, you have to find all four. When you write a test, you have to test all four.

In Domain-Driven Design, that rule lives in exactly one place: the `Token` entity. The method is called `IsValid()`, it is 5 lines, and every piece of code that needs to check token validity calls it. There is one test for the rule, and it is a plain Go test with no database, no HTTP server, no mocks.

This article shows you how the four domain entities in this project — `User`, `Token`, `Conversation`, and `Message` — encode their business rules as methods, and how the repository interface pattern makes the domain layer completely independent of PostgreSQL.

---

## What the Domain Layer Is (and What It Cannot Import)

The domain layer lives in `internal/domain/`. Open the `go.mod` imports in any file there and you will find exactly two external packages:

```go
import (
    "time"
    "github.com/google/uuid"
)
```

That is it. No `pgx`, no `redis`, no `chi`, no `eino`. The domain layer is pure Go logic. It can be compiled, tested, and reasoned about without any external infrastructure.

This is not an accident. It is the **Dependency Rule**: source code dependencies point inward. Domain ← Application ← Infrastructure. The inner layers cannot import the outer layers.

The practical consequence: you can run every domain test with `go test ./internal/domain/...` and it completes in milliseconds. No database connection needed. No containers. No ports. Just Go.

---

## The User Entity: Roles and Business Methods

```go
// internal/domain/entity/user.go
type UserRole string

const (
    UserRoleAdmin UserRole = "admin"
    UserRoleUser  UserRole = "user"
    UserRoleAgent UserRole = "agent"  // For AI agent service accounts
)

type User struct {
    ID           uuid.UUID
    Email        string
    PasswordHash string
    Name         string
    Role         UserRole
    IsActive     bool
    Metadata     map[string]any
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

The `User` struct is plain data. No ORM tags, no JSON annotations (those belong in the infrastructure layer). But notice the constructor:

```go
func NewUser(email, passwordHash, name string) *User {
    return &User{
        ID:        uuid.New(),
        Role:      UserRoleUser,    // New users are regular users by default
        IsActive:  true,            // Active by default
        Metadata:  make(map[string]any),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}
```

The constructor enforces invariants. You cannot create a `User` with no ID, no role, or uninitialized metadata. The zero value of `User{}` is invalid — the constructor is the only correct way to create one.

Then the business methods:

```go
func (u *User) CanUseTools() bool {
    return u.IsActive && (u.Role == UserRoleAdmin || u.Role == UserRoleUser)
}

func (u *User) IsAdmin() bool {
    return u.Role == UserRoleAdmin
}
```

`CanUseTools()` encodes a business rule: only active admin and user accounts can execute tools. Agent accounts cannot. This rule lives in the entity because it is a business concern, not a technical one. The HTTP handler asks `if !user.CanUseTools()` — it does not duplicate the role logic.

---

## The Token Entity: A Permission Carrier

The `Token` entity is where this project gets interesting. A token is not just an authentication credential — it is a **permission carrier** with rate limits and access control lists built in:

```go
type Token struct {
    ID                 uuid.UUID
    UserID             uuid.UUID
    TokenHash          string      // Never store raw tokens
    TokenType          TokenType   // api_key, access, refresh
    Name               string      // "Production API Key", "Dev Testing"
    ExpiresAt          time.Time
    LastUsedAt         *time.Time  // Pointer: nil until first use
    RateLimitPerMinute int         // Per-token rate limits
    RateLimitPerDay    int
    AllowedTools       []string    // nil = all tools; ["calculator"] = calculator only
    AllowedModels      []string    // nil = all models
    IsRevoked          bool
    Metadata           map[string]any
    CreatedAt          time.Time
    UpdatedAt          time.Time
}
```

The business methods on `Token` are where the real value is:

```go
func (t *Token) IsExpired() bool {
    return time.Now().After(t.ExpiresAt)
}

func (t *Token) IsValid() bool {
    return !t.IsRevoked && !t.IsExpired()
}

func (t *Token) CanUseTool(toolName string) bool {
    if len(t.AllowedTools) == 0 {
        return true  // nil = all tools allowed
    }
    for _, allowed := range t.AllowedTools {
        if allowed == toolName || allowed == "*" {
            return true
        }
    }
    return false
}

func (t *Token) CanUseModel(modelName string) bool {
    if len(t.AllowedModels) == 0 {
        return true  // nil = all models allowed
    }
    for _, allowed := range t.AllowedModels {
        if allowed == modelName || allowed == "*" {
            return true
        }
    }
    return false
}
```

**`IsValid()`** — one line that combines two conditions. Every piece of code that needs to check token validity calls this method. The rule is defined once.

**`CanUseTool()`** — the access control logic for tools. If `AllowedTools` is empty, all tools are allowed (open access). Otherwise, it checks for an exact match or a wildcard `"*"`. This means you can issue API keys that are scoped to a specific subset of tools.

For example, a customer on a "Basic" plan gets an API key with `AllowedTools: ["calculator"]`. A "Pro" customer gets `AllowedTools: nil` (all tools). An enterprise customer gets `AllowedTools: ["calculator", "web_search", "database_query"]`.

The rate limits per token mean different keys can have different throttling — your internal admin key has no limit, third-party integrations have conservative limits.

---

## The Conversation Entity: A Go State Machine

The `Conversation` entity is the aggregate root for chat sessions. It has five possible statuses:

```go
const (
    ConversationStatusActive    ConversationStatus = "active"
    ConversationStatusPending   ConversationStatus = "pending_approval"
    ConversationStatusCompleted ConversationStatus = "completed"
    ConversationStatusFailed    ConversationStatus = "failed"
    ConversationStatusArchived  ConversationStatus = "archived"
)
```

And the transition methods enforce the state machine:

```go
func (c *Conversation) RequestApproval(node string, data map[string]any) {
    c.Status = ConversationStatusPending
    c.CurrentNode = node
    c.Metadata["pending_approval"] = data
    c.UpdatedAt = time.Now()
}

func (c *Conversation) Approve() {
    c.Status = ConversationStatusActive
    delete(c.Metadata, "pending_approval")
    c.UpdatedAt = time.Now()
}

func (c *Conversation) Reject(reason string) {
    c.Status = ConversationStatusActive
    c.Metadata["last_rejection"] = map[string]any{
        "reason": reason,
        "at":     time.Now(),
    }
    c.UpdatedAt = time.Now()
}

func (c *Conversation) Complete() {
    c.Status = ConversationStatusCompleted
    now := time.Now()
    c.CompletedAt = &now
    c.UpdatedAt = now
}
```

What makes this a proper state machine:

1. **Transitions have names** — `RequestApproval()`, not `c.Status = "pending_approval"`
2. **Transitions carry data** — `RequestApproval(node, data)` stores what was pending and where
3. **Transitions have side effects** — `Approve()` clears the pending approval metadata; `Reject()` records the rejection reason
4. **`UpdatedAt` is always maintained** — every mutation updates the timestamp

If you wrote this as raw field assignments scattered across use cases and handlers:

```go
// BAD: business logic leaking into application layer
conv.Status = "pending_approval"
conv.CurrentNode = node
conv.Metadata["pending_approval"] = data
conv.UpdatedAt = time.Now()
```

...you would have to remember those four lines every time. With the domain method, it is one call.

The Conversation entity also carries Eino workflow state:

```go
type Conversation struct {
    // ...
    CurrentNode   string         // Which Eino node is waiting
    WorkflowState map[string]any // Serialized AgentState for resumption
    // ...
}
```

When a workflow pauses for human approval, the entire `AgentState` is serialized to JSON and stored in `WorkflowState`. When the user approves, the state is deserialized and the workflow resumes from where it left off. The entity is the persistence boundary.

---

## The Message Entity: Protocol Translation at Domain Level

The `Message` entity stores chat messages with OpenAI-compatible fields:

```go
type Message struct {
    ID             uuid.UUID
    ConversationID uuid.UUID
    Role           MessageRole    // system, user, assistant, tool
    Content        string
    Name           string         // For tool messages
    ToolCalls      []ToolCall     // Assistant → tool calls
    ToolCallID     string         // Tool response → back-reference
    Model          string
    PromptTokens   int
    CompletionTokens int
    Latency        time.Duration
    SequenceNumber int
    CreatedAt      time.Time
}
```

Notice `ToolCalls []ToolCall` and `ToolCallID string`. These are the two sides of the tool calling protocol:

- When the assistant wants to call a tool, it produces a message with `Role = "assistant"` and `ToolCalls` populated
- When the tool returns a result, it produces a message with `Role = "tool"`, a matching `ToolCallID`, and the result in `Content`

The factory constructors enforce correct construction:

```go
func NewToolMessage(conversationID uuid.UUID, toolCallID string, name string, result any) *Message {
    content, _ := json.Marshal(result)
    return &Message{
        ID:             uuid.New(),
        ConversationID: conversationID,
        Role:           RoleTool,
        Name:           name,
        Content:        string(content),  // result serialized to JSON
        ToolCallID:     toolCallID,
        CreatedAt:      time.Now(),
    }
}
```

The `result any` parameter gets JSON-serialized into `Content`. The domain entity knows that tool results are always JSON strings. The caller passes any Go value; the entity handles the serialization.

The `ToOpenAIFormat()` method handles protocol translation:

```go
func (m *Message) ToOpenAIFormat() map[string]any {
    msg := map[string]any{
        "role":    string(m.Role),
        "content": m.Content,
    }
    if len(m.ToolCalls) > 0 {
        msg["tool_calls"] = m.ToolCalls
    }
    if m.ToolCallID != "" {
        msg["tool_call_id"] = m.ToolCallID
    }
    if m.Name != "" {
        msg["name"] = m.Name
    }
    return msg
}
```

This method converts the domain entity into the map format expected by the OpenAI chat completions API. The domain entity knows about OpenAI's format — not as a framework dependency, but as a protocol specification that the entity is responsible for producing correctly.

---

## Repository Interfaces: Why They Live in the Domain Layer

The repository interfaces are defined in `internal/domain/repository/`, not in `internal/infrastructure/`. This is the critical architectural decision.

```go
// internal/domain/repository/conversation_repository.go
type ConversationFilter struct {
    UserID  *uuid.UUID
    Status  *entity.ConversationStatus
    Limit   int
    Offset  int
    OrderBy string
    Order   string
}

type ConversationRepository interface {
    Create(ctx context.Context, conv *entity.Conversation) error
    FindByID(ctx context.Context, id uuid.UUID) (*entity.Conversation, error)
    FindByUserID(ctx context.Context, userID uuid.UUID, filter ConversationFilter) ([]*entity.Conversation, error)
    Update(ctx context.Context, conv *entity.Conversation) error
    UpdateWorkflowState(ctx context.Context, id uuid.UUID, state map[string]any) error
    Delete(ctx context.Context, id uuid.UUID) error
    CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}
```

The interface lives in the domain because it expresses what the domain needs from its persistence mechanism — not what PostgreSQL can provide. The domain layer defines the contract; the infrastructure layer fulfills it.

This means:
- The `SendMessage` use case depends on `ConversationRepository` (an interface) — not `postgres.ConversationRepository` (a concrete type)
- To test `SendMessage`, you pass a mock implementation of `ConversationRepository` — no PostgreSQL needed
- To swap PostgreSQL for MySQL or DynamoDB, you implement the interface with a new concrete type — the domain and application layers do not change

---

## Testing Pure Domain Logic

The lack of external dependencies in the domain layer means tests are instant and have no setup:

```go
// internal/domain/entity/token_test.go
func TestToken_IsValid(t *testing.T) {
    t.Run("valid token", func(t *testing.T) {
        token := entity.NewAPIKey(uuid.New(), "hash123", "test", time.Now().Add(time.Hour))
        assert.True(t, token.IsValid())
    })

    t.Run("expired token", func(t *testing.T) {
        token := entity.NewAPIKey(uuid.New(), "hash123", "test", time.Now().Add(-time.Hour))
        assert.False(t, token.IsValid())
        assert.True(t, token.IsExpired())
    })

    t.Run("revoked token", func(t *testing.T) {
        token := entity.NewAPIKey(uuid.New(), "hash123", "test", time.Now().Add(time.Hour))
        token.IsRevoked = true
        assert.False(t, token.IsValid())
    })
}
```

No database. No mock setup. No context. Just: construct an entity, call a method, assert the result. These tests run in under a millisecond.

The same pattern applies to the conversation state machine:

```go
func TestConversation_ApprovalFlow(t *testing.T) {
    conv := entity.NewConversation(uuid.New(), "general")
    assert.True(t, conv.IsActive())
    assert.False(t, conv.IsPendingApproval())

    conv.RequestApproval("act", map[string]any{"tool": "dangerous_tool"})
    assert.True(t, conv.IsPendingApproval())
    assert.Equal(t, "act", conv.CurrentNode)

    conv.Approve()
    assert.True(t, conv.IsActive())
    assert.NotContains(t, conv.Metadata, "pending_approval")
}
```

Every business rule in the entity has a corresponding test. The tests are exhaustive because the entities are small and focused.

---

## What Makes This Design Work

**Business rules live with the data they protect.** The rule "a revoked token cannot be used" lives on the `Token` struct, not in a validator, not in a middleware, not in a handler. The method `IsValid()` is the single source of truth.

**Constructors are factories, not just `{}`**.** `NewConversation()`, `NewAPIKey()`, `NewUserMessage()` — every entity has a constructor that sets invariants and defaults. The zero value of any entity struct is intentionally incomplete.

**State transitions are named methods.** `RequestApproval()`, `Approve()`, `Reject()`, `Complete()` — you read the code and understand what is happening at the business level, not just at the field level.

**Protocol translation at the boundary.** `Message.ToOpenAIFormat()` is the only place in the domain layer that knows about the OpenAI message structure. This method is the boundary between "what a message means to our domain" and "what a message looks like to an LLM API."

---

## What We Just Learned

- The domain layer imports only stdlib and `uuid` — zero framework dependencies
- `NewUser()`, `NewToken()`, `NewConversation()` constructors enforce invariants that the zero value cannot
- `Token` carries rate limits, tool allowlists, and model allowlists as first-class business data
- `Conversation` is a proper state machine with named transition methods
- `Message.ToOpenAIFormat()` handles protocol translation at the domain boundary
- Repository interfaces live in the domain layer, not the infrastructure layer — they define what the domain needs, not what the database provides
- Domain tests run in milliseconds with no external dependencies

---

*Next in the series: [Article 3 — Add a New AI Tool in 15 Lines of Go](#)*

*Previous: [Article 1 — Architecture Overview](#)*
