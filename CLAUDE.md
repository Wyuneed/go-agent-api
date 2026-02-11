# Go Production-Ready DDD Boilerplate - Complete Guide
# AI Agentic Framework with Eino Workflow Engine

## Overview

Build a **production-ready Go AI Agent API** following **Domain-Driven Design (DDD)** principles, optimized for:

- ✅ AI agentic workflows (OpenAI tool calling format, LiteLLM compatible)
- ✅ Complex workflows with Eino (LangGraph equivalent for Go)
- ✅ Human-in-the-loop approval system
- ✅ Cost efficiency (low memory, fast cold start)
- ✅ Single binary deployment
- ✅ JWT authentication with DB-managed token expiration

**Tech Stack:**
- Go 1.22+
- PostgreSQL (primary database)
- Redis (caching, rate limiting, workflow state, conversation memory)
- Eino (ByteDance) - LangGraph-equivalent for Go
- LiteLLM compatible (OpenAI tool calling format)
- Docker & Docker Compose

---

## Table of Contents

1. [Project Structure](#project-structure)
2. [Core Design Principles](#core-design-principles)
3. [Domain Layer](#domain-layer)
4. [Database Migrations](#database-migrations)
5. [JWT + Token Authentication](#jwt--token-authentication)
6. [OpenAI-Compatible Tool System](#openai-compatible-tool-system)
7. [Eino Workflow Engine](#eino-workflow-engine)
8. [LLM Provider Integration](#llm-provider-integration)
9. [HTTP Handlers & Streaming](#http-handlers--streaming)
10. [Infrastructure Setup](#infrastructure-setup)
11. [Docker Configuration](#docker-configuration)
12. [Testing Strategy](#testing-strategy)
13. [Makefile & Commands](#makefile--commands)
14. [API Endpoints](#api-endpoints)
15. [Performance Optimization](#performance-optimization)

---

## Project Structure

```
project-root/
├── cmd/
│   └── api/
│       └── main.go                     # Entry point
│
├── internal/
│   ├── domain/                         # Enterprise Business Rules (innermost)
│   │   ├── entity/
│   │   │   ├── user.go
│   │   │   ├── token.go
│   │   │   ├── conversation.go         # Chat conversation aggregate
│   │   │   ├── message.go              # Chat message entity
│   │   │   └── agent.go                # Agent configuration
│   │   ├── valueobject/
│   │   │   ├── email.go
│   │   │   ├── message_role.go         # user, assistant, tool, system
│   │   │   └── tool_call.go
│   │   ├── repository/                 # Interfaces only
│   │   │   ├── user_repository.go
│   │   │   ├── token_repository.go
│   │   │   ├── conversation_repository.go
│   │   │   └── message_repository.go
│   │   ├── service/
│   │   │   ├── auth_domain_service.go
│   │   │   └── conversation_domain_service.go
│   │   └── event/
│   │       ├── message_created.go
│   │       ├── tool_executed.go
│   │       └── conversation_completed.go
│   │
│   ├── application/                    # Application Business Rules
│   │   ├── usecase/
│   │   │   ├── auth/
│   │   │   │   ├── login.go
│   │   │   │   ├── validate_token.go
│   │   │   │   └── refresh_token.go
│   │   │   ├── chat/
│   │   │   │   ├── send_message.go     # Main chat entry point
│   │   │   │   ├── get_conversation.go
│   │   │   │   ├── list_conversations.go
│   │   │   │   ├── approve_action.go   # Human-in-the-loop
│   │   │   │   └── stream_response.go
│   │   │   ├── tool/
│   │   │   │   ├── execute_tool.go
│   │   │   │   ├── tool_registry.go
│   │   │   │   └── builtin/
│   │   │   │       ├── web_search.go
│   │   │   │       ├── calculator.go
│   │   │   │       ├── code_executor.go
│   │   │   │       └── database_query.go
│   │   │   ├── workflow/
│   │   │   │   ├── agent_workflow.go
│   │   │   │   └── nodes/
│   │   │   │       ├── router_node.go
│   │   │   │       ├── thinking_node.go
│   │   │   │       ├── tool_node.go
│   │   │   │       ├── response_node.go
│   │   │   │       └── human_approval_node.go
│   │   │   └── user/
│   │   │       ├── create_user.go
│   │   │       └── get_user.go
│   │   ├── dto/
│   │   │   ├── request/
│   │   │   │   ├── chat_request.go
│   │   │   │   ├── auth_request.go
│   │   │   │   └── tool_request.go
│   │   │   └── response/
│   │   │       ├── chat_response.go
│   │   │       ├── auth_response.go
│   │   │       └── stream_event.go
│   │   └── port/                       # External service interfaces
│   │       ├── llm_provider.go
│   │       ├── cache.go
│   │       ├── event_publisher.go
│   │       └── vector_store.go         # For RAG (optional)
│   │
│   ├── infrastructure/                 # Frameworks & Drivers (outermost)
│   │   ├── llm/
│   │   │   ├── provider.go
│   │   │   ├── litellm/
│   │   │   │   └── client.go
│   │   │   └── openai/
│   │   │       └── client.go
│   │   ├── eino/
│   │   │   ├── graphs/
│   │   │   │   ├── chatbot_graph.go
│   │   │   │   ├── rag_graph.go
│   │   │   │   └── multi_agent_graph.go
│   │   │   ├── nodes/
│   │   │   │   └── node_factory.go
│   │   │   └── state/
│   │   │       └── agent_state.go
│   │   ├── persistence/
│   │   │   ├── postgres/
│   │   │   │   ├── connection.go
│   │   │   │   ├── user_repository.go
│   │   │   │   ├── token_repository.go
│   │   │   │   ├── conversation_repository.go
│   │   │   │   ├── message_repository.go
│   │   │   │   └── migrations/
│   │   │   │       ├── 000001_create_users_table.up.sql
│   │   │   │       ├── 000001_create_users_table.down.sql
│   │   │   │       ├── 000002_create_tokens_table.up.sql
│   │   │   │       ├── 000002_create_tokens_table.down.sql
│   │   │   │       ├── 000003_create_conversations_table.up.sql
│   │   │   │       ├── 000003_create_conversations_table.down.sql
│   │   │   │       ├── 000004_create_messages_table.up.sql
│   │   │   │       ├── 000004_create_messages_table.down.sql
│   │   │   │       ├── 000005_create_tool_executions_table.up.sql
│   │   │   │       └── 000005_create_tool_executions_table.down.sql
│   │   │   └── redis/
│   │   │       ├── connection.go
│   │   │       ├── cache.go
│   │   │       ├── rate_limiter.go
│   │   │       ├── conversation_cache.go
│   │   │       └── workflow_state.go
│   │   ├── http/
│   │   │   ├── server.go
│   │   │   ├── router.go
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go
│   │   │   │   ├── ratelimit.go
│   │   │   │   ├── logging.go
│   │   │   │   ├── recovery.go
│   │   │   │   ├── cors.go
│   │   │   │   └── requestid.go
│   │   │   └── handler/
│   │   │       ├── health.go
│   │   │       ├── auth_handler.go
│   │   │       ├── user_handler.go
│   │   │       ├── chat_handler.go
│   │   │       ├── tool_handler.go
│   │   │       └── approval_handler.go
│   │   ├── worker/
│   │   │   └── pool.go
│   │   └── config/
│   │       └── config.go
│   │
│   └── pkg/                            # Internal shared utilities
│       ├── errors/
│       │   └── errors.go
│       ├── logger/
│       │   └── logger.go
│       ├── jwt/
│       │   └── jwt.go
│       ├── validator/
│       │   └── validator.go
│       ├── response/
│       │   └── response.go
│       └── streaming/
│           └── sse.go
│
├── pkg/                                # Public shared packages
│   └── toolspec/
│       └── openai_format.go            # OpenAI-compatible tool types
│
├── migrations/                         # Alternative location
├── scripts/
│   ├── migrate.sh
│   └── seed.sh
├── deployments/
│   ├── Dockerfile
│   ├── Dockerfile.multistage
│   └── docker-compose.yml
├── configs/
│   ├── config.yaml
│   └── config.example.yaml
├── tests/
│   ├── integration/
│   │   ├── chat_test.go
│   │   └── tool_test.go
│   ├── e2e/
│   └── mocks/
├── Makefile
├── go.mod
├── go.sum
├── .env.example
├── .air.toml                           # Hot reload config
├── .gitignore
└── README.md
```

---

## Core Design Principles

### 1. DDD Layer Rules (STRICT)

```
┌─────────────────────────────────────────────────────────────┐
│                    Infrastructure Layer                      │
│  (HTTP, DB repos, Redis, LLM clients, Eino graphs)          │
├─────────────────────────────────────────────────────────────┤
│                    Application Layer                         │
│  (Use cases, DTOs, orchestration, workflow nodes)           │
├─────────────────────────────────────────────────────────────┤
│                      Domain Layer                            │
│  (Entities, Value Objects, Domain Services, Repo Interfaces)│
└─────────────────────────────────────────────────────────────┘

DEPENDENCY RULE: Dependencies point INWARD only!
- Domain layer has ZERO external dependencies (pure Go)
- Application layer depends only on Domain
- Infrastructure depends on Application and Domain
```

### 2. Dependency Injection Pattern

```go
// GOOD: Constructor injection
func NewChatUseCase(
    convRepo repository.ConversationRepository,
    msgRepo repository.MessageRepository,
    llm port.LLMProvider,
    cache port.Cache,
) *ChatUseCase {
    return &ChatUseCase{
        convRepo: convRepo,
        msgRepo:  msgRepo,
        llm:      llm,
        cache:    cache,
    }
}

// BAD: Global variables
var convRepo = postgres.NewConversationRepository() // DON'T DO THIS
```

### 3. Error Handling Strategy

```go
// internal/pkg/errors/errors.go
package errors

import "fmt"

type AppError struct {
    Code       string `json:"code"`
    Message    string `json:"message"`
    StatusCode int    `json:"-"`
    Err        error  `json:"-"`
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

func (e *AppError) Wrap(err error) *AppError {
    return &AppError{
        Code:       e.Code,
        Message:    e.Message,
        StatusCode: e.StatusCode,
        Err:        err,
    }
}

// Predefined errors
var (
    ErrNotFound            = &AppError{Code: "NOT_FOUND", Message: "Resource not found", StatusCode: 404}
    ErrUnauthorized        = &AppError{Code: "UNAUTHORIZED", Message: "Unauthorized", StatusCode: 401}
    ErrForbidden           = &AppError{Code: "FORBIDDEN", Message: "Forbidden", StatusCode: 403}
    ErrTokenExpired        = &AppError{Code: "TOKEN_EXPIRED", Message: "Token has expired", StatusCode: 401}
    ErrInvalidToken        = &AppError{Code: "INVALID_TOKEN", Message: "Invalid token", StatusCode: 401}
    ErrRateLimitExceeded   = &AppError{Code: "RATE_LIMIT", Message: "Rate limit exceeded", StatusCode: 429}
    ErrToolExecutionFailed = &AppError{Code: "TOOL_EXEC_FAILED", Message: "Tool execution failed", StatusCode: 500}
    ErrToolNotAllowed      = &AppError{Code: "TOOL_NOT_ALLOWED", Message: "Tool not allowed for this token", StatusCode: 403}
    ErrPendingApproval     = &AppError{Code: "PENDING_APPROVAL", Message: "Action pending approval", StatusCode: 202}
    ErrValidation          = &AppError{Code: "VALIDATION_ERROR", Message: "Validation failed", StatusCode: 400}
    ErrInternal            = &AppError{Code: "INTERNAL_ERROR", Message: "Internal server error", StatusCode: 500}
)
```

---

## Domain Layer

### User Entity

```go
// internal/domain/entity/user.go
package entity

import (
    "time"
    "github.com/google/uuid"
)

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

func NewUser(email, passwordHash, name string) *User {
    return &User{
        ID:           uuid.New(),
        Email:        email,
        PasswordHash: passwordHash,
        Name:         name,
        Role:         UserRoleUser,
        IsActive:     true,
        Metadata:     make(map[string]any),
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
}

func (u *User) CanUseTools() bool {
    return u.IsActive && (u.Role == UserRoleAdmin || u.Role == UserRoleUser)
}

func (u *User) IsAdmin() bool {
    return u.Role == UserRoleAdmin
}
```

### Token Entity

```go
// internal/domain/entity/token.go
package entity

import (
    "time"
    "github.com/google/uuid"
)

type TokenType string

const (
    TokenTypeAPIKey  TokenType = "api_key"
    TokenTypeAccess  TokenType = "access"
    TokenTypeRefresh TokenType = "refresh"
)

type Token struct {
    ID                 uuid.UUID
    UserID             uuid.UUID
    TokenHash          string      // Never store raw token
    TokenType          TokenType
    Name               string      // Friendly name
    ExpiresAt          time.Time
    LastUsedAt         *time.Time
    RateLimitPerMinute int
    RateLimitPerDay    int
    AllowedTools       []string    // nil = all tools allowed
    AllowedModels      []string    // nil = all models allowed
    IsRevoked          bool
    Metadata           map[string]any
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

func NewAPIKey(userID uuid.UUID, tokenHash, name string, expiresAt time.Time) *Token {
    return &Token{
        ID:                 uuid.New(),
        UserID:             userID,
        TokenHash:          tokenHash,
        TokenType:          TokenTypeAPIKey,
        Name:               name,
        ExpiresAt:          expiresAt,
        RateLimitPerMinute: 60,
        RateLimitPerDay:    10000,
        IsRevoked:          false,
        Metadata:           make(map[string]any),
        CreatedAt:          time.Now(),
        UpdatedAt:          time.Now(),
    }
}

func (t *Token) IsExpired() bool {
    return time.Now().After(t.ExpiresAt)
}

func (t *Token) IsValid() bool {
    return !t.IsRevoked && !t.IsExpired()
}

func (t *Token) CanUseTool(toolName string) bool {
    if t.AllowedTools == nil || len(t.AllowedTools) == 0 {
        return true
    }
    for _, allowed := range t.AllowedTools {
        if allowed == toolName || allowed == "*" {
            return true
        }
    }
    return false
}

func (t *Token) CanUseModel(modelName string) bool {
    if t.AllowedModels == nil || len(t.AllowedModels) == 0 {
        return true
    }
    for _, allowed := range t.AllowedModels {
        if allowed == modelName || allowed == "*" {
            return true
        }
    }
    return false
}

func (t *Token) UpdateLastUsed() {
    now := time.Now()
    t.LastUsedAt = &now
}
```

### Conversation Entity (Aggregate Root)

```go
// internal/domain/entity/conversation.go
package entity

import (
    "time"
    "github.com/google/uuid"
)

type ConversationStatus string

const (
    ConversationStatusActive    ConversationStatus = "active"
    ConversationStatusPending   ConversationStatus = "pending_approval"
    ConversationStatusCompleted ConversationStatus = "completed"
    ConversationStatusFailed    ConversationStatus = "failed"
    ConversationStatusArchived  ConversationStatus = "archived"
)

// Conversation is the aggregate root for chat sessions
type Conversation struct {
    ID              uuid.UUID
    UserID          uuid.UUID
    Title           string
    Status          ConversationStatus
    
    // Agent configuration
    AgentType       string              // "general", "coder", "researcher", etc.
    SystemPrompt    string
    Model           string              // Default model for this conversation
    Temperature     float64
    
    // Workflow state (for Eino)
    CurrentNode     string
    WorkflowState   map[string]any
    
    // Counters
    MessageCount    int
    ToolCallCount   int
    TotalTokens     int
    TotalCost       float64             // Estimated cost in USD
    
    // Metadata
    Metadata        map[string]any
    Tags            []string
    
    // Timestamps
    CreatedAt       time.Time
    UpdatedAt       time.Time
    LastMessageAt   *time.Time
    CompletedAt     *time.Time
}

func NewConversation(userID uuid.UUID, agentType string) *Conversation {
    return &Conversation{
        ID:            uuid.New(),
        UserID:        userID,
        Status:        ConversationStatusActive,
        AgentType:     agentType,
        Temperature:   0.7,
        Metadata:      make(map[string]any),
        WorkflowState: make(map[string]any),
        Tags:          []string{},
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }
}

func (c *Conversation) IsActive() bool {
    return c.Status == ConversationStatusActive
}

func (c *Conversation) IsPendingApproval() bool {
    return c.Status == ConversationStatusPending
}

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

func (c *Conversation) AddMessageStats(tokens int, toolCalls int) {
    c.MessageCount++
    c.TotalTokens += tokens
    c.ToolCallCount += toolCalls
    now := time.Now()
    c.LastMessageAt = &now
    c.UpdatedAt = now
}

func (c *Conversation) SetTitle(title string) {
    if c.Title == "" && title != "" {
        c.Title = title
        c.UpdatedAt = time.Now()
    }
}
```

### Message Entity

```go
// internal/domain/entity/message.go
package entity

import (
    "encoding/json"
    "time"
    "github.com/google/uuid"
)

type MessageRole string

const (
    RoleSystem    MessageRole = "system"
    RoleUser      MessageRole = "user"
    RoleAssistant MessageRole = "assistant"
    RoleTool      MessageRole = "tool"
)

// ToolCall represents a tool call from the assistant
type ToolCall struct {
    ID       string `json:"id"`
    Type     string `json:"type"`      // "function"
    Function struct {
        Name      string `json:"name"`
        Arguments string `json:"arguments"`  // JSON string
    } `json:"function"`
}

type Message struct {
    ID             uuid.UUID
    ConversationID uuid.UUID
    Role           MessageRole
    Content        string
    Name           string           // For tool messages
    
    // Tool-related
    ToolCalls      []ToolCall       // Assistant requesting tools
    ToolCallID     string           // For tool response messages
    
    // Metadata
    Model          string
    PromptTokens   int
    CompletionTokens int
    Latency        time.Duration
    
    // For streaming
    IsStreaming    bool
    StreamComplete bool
    
    // Ordering
    SequenceNumber int
    
    CreatedAt      time.Time
}

func NewUserMessage(conversationID uuid.UUID, content string) *Message {
    return &Message{
        ID:             uuid.New(),
        ConversationID: conversationID,
        Role:           RoleUser,
        Content:        content,
        CreatedAt:      time.Now(),
    }
}

func NewAssistantMessage(conversationID uuid.UUID, content string, toolCalls []ToolCall, model string) *Message {
    return &Message{
        ID:             uuid.New(),
        ConversationID: conversationID,
        Role:           RoleAssistant,
        Content:        content,
        ToolCalls:      toolCalls,
        Model:          model,
        CreatedAt:      time.Now(),
    }
}

func NewSystemMessage(conversationID uuid.UUID, content string) *Message {
    return &Message{
        ID:             uuid.New(),
        ConversationID: conversationID,
        Role:           RoleSystem,
        Content:        content,
        CreatedAt:      time.Now(),
    }
}

func NewToolMessage(conversationID uuid.UUID, toolCallID string, name string, result any) *Message {
    content, _ := json.Marshal(result)
    return &Message{
        ID:             uuid.New(),
        ConversationID: conversationID,
        Role:           RoleTool,
        Name:           name,
        Content:        string(content),
        ToolCallID:     toolCallID,
        CreatedAt:      time.Now(),
    }
}

// ToOpenAIFormat converts to OpenAI-compatible message format
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

### Repository Interfaces

```go
// internal/domain/repository/user_repository.go
package repository

import (
    "context"
    "github.com/google/uuid"
    "yourproject/internal/domain/entity"
)

type UserRepository interface {
    Create(ctx context.Context, user *entity.User) error
    FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
    FindByEmail(ctx context.Context, email string) (*entity.User, error)
    Update(ctx context.Context, user *entity.User) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

```go
// internal/domain/repository/token_repository.go
package repository

import (
    "context"
    "github.com/google/uuid"
    "yourproject/internal/domain/entity"
)

type TokenRepository interface {
    Create(ctx context.Context, token *entity.Token) error
    FindByHash(ctx context.Context, tokenHash string) (*entity.Token, error)
    FindByID(ctx context.Context, id uuid.UUID) (*entity.Token, error)
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Token, error)
    UpdateLastUsed(ctx context.Context, tokenID uuid.UUID) error
    Revoke(ctx context.Context, tokenID uuid.UUID) error
    RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
    DeleteExpired(ctx context.Context) (int64, error)
}
```

```go
// internal/domain/repository/conversation_repository.go
package repository

import (
    "context"
    "github.com/google/uuid"
    "yourproject/internal/domain/entity"
)

type ConversationFilter struct {
    UserID  *uuid.UUID
    Status  *entity.ConversationStatus
    Limit   int
    Offset  int
    OrderBy string  // "created_at", "updated_at", "last_message_at"
    Order   string  // "asc", "desc"
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

```go
// internal/domain/repository/message_repository.go
package repository

import (
    "context"
    "github.com/google/uuid"
    "yourproject/internal/domain/entity"
)

type MessageRepository interface {
    Create(ctx context.Context, msg *entity.Message) error
    CreateBatch(ctx context.Context, msgs []*entity.Message) error
    FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*entity.Message, error)
    FindByConversationIDPaginated(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*entity.Message, error)
    FindLastN(ctx context.Context, conversationID uuid.UUID, n int) ([]*entity.Message, error)
    CountByConversationID(ctx context.Context, conversationID uuid.UUID) (int64, error)
    Delete(ctx context.Context, id uuid.UUID) error
    DeleteByConversationID(ctx context.Context, conversationID uuid.UUID) error
}
```

---

## Database Migrations

### Install Migration Tool

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 000001_create_users_table.up.sql

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'user' CHECK (role IN ('admin', 'user', 'agent')),
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_is_active ON users(is_active) WHERE is_active = true;

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### 000001_create_users_table.down.sql

```sql
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS update_updated_at_column();
```

### 000002_create_tokens_table.up.sql

```sql
CREATE TABLE user_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    token_type VARCHAR(50) NOT NULL DEFAULT 'api_key' CHECK (token_type IN ('api_key', 'access', 'refresh')),
    name VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    rate_limit_per_minute INT DEFAULT 60,
    rate_limit_per_day INT DEFAULT 10000,
    allowed_tools TEXT[],
    allowed_models TEXT[],
    is_revoked BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_tokens_user_id ON user_tokens(user_id);
CREATE INDEX idx_tokens_token_hash ON user_tokens(token_hash);
CREATE INDEX idx_tokens_expires_at ON user_tokens(expires_at);
CREATE INDEX idx_tokens_valid ON user_tokens(token_hash) 
    WHERE is_revoked = false AND expires_at > NOW();

CREATE TRIGGER update_tokens_updated_at BEFORE UPDATE ON user_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### 000002_create_tokens_table.down.sql

```sql
DROP TRIGGER IF EXISTS update_tokens_updated_at ON user_tokens;
DROP TABLE IF EXISTS user_tokens;
```

### 000003_create_conversations_table.up.sql

```sql
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'active' 
        CHECK (status IN ('active', 'pending_approval', 'completed', 'failed', 'archived')),
    
    -- Agent configuration
    agent_type VARCHAR(100) DEFAULT 'general',
    system_prompt TEXT,
    model VARCHAR(100),
    temperature DECIMAL(3,2) DEFAULT 0.7,
    
    -- Workflow state (Eino)
    current_node VARCHAR(100),
    workflow_state JSONB DEFAULT '{}',
    
    -- Counters
    message_count INT DEFAULT 0,
    tool_call_count INT DEFAULT 0,
    total_tokens INT DEFAULT 0,
    total_cost DECIMAL(10,6) DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_message_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE  -- Soft delete
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_status ON conversations(status);
CREATE INDEX idx_conversations_user_status ON conversations(user_id, status);
CREATE INDEX idx_conversations_updated ON conversations(updated_at DESC);
CREATE INDEX idx_conversations_last_message ON conversations(last_message_at DESC NULLS LAST);
CREATE INDEX idx_conversations_pending ON conversations(status) WHERE status = 'pending_approval';
CREATE INDEX idx_conversations_active ON conversations(user_id, updated_at DESC) 
    WHERE deleted_at IS NULL AND status IN ('active', 'pending_approval');
CREATE INDEX idx_conversations_tags ON conversations USING GIN(tags);

CREATE TRIGGER update_conversations_updated_at BEFORE UPDATE ON conversations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### 000003_create_conversations_table.down.sql

```sql
DROP TRIGGER IF EXISTS update_conversations_updated_at ON conversations;
DROP TABLE IF EXISTS conversations;
```

### 000004_create_messages_table.up.sql

```sql
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content TEXT,
    name VARCHAR(100),
    
    -- Tool-related
    tool_calls JSONB,
    tool_call_id VARCHAR(100),
    
    -- Metadata
    model VARCHAR(100),
    prompt_tokens INT DEFAULT 0,
    completion_tokens INT DEFAULT 0,
    latency_ms INT,
    
    -- Streaming
    is_streaming BOOLEAN DEFAULT false,
    stream_complete BOOLEAN DEFAULT true,
    
    -- Ordering
    sequence_number SERIAL,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id, sequence_number);
CREATE INDEX idx_messages_conversation_recent ON messages(conversation_id, created_at DESC);
CREATE INDEX idx_messages_role ON messages(conversation_id, role);
CREATE INDEX idx_messages_tool_calls ON messages(conversation_id) WHERE tool_calls IS NOT NULL;
```

### 000004_create_messages_table.down.sql

```sql
DROP TABLE IF EXISTS messages;
```

### 000005_create_tool_executions_table.up.sql

```sql
CREATE TABLE tool_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_id UUID REFERENCES user_tokens(id) ON DELETE SET NULL,
    
    tool_name VARCHAR(255) NOT NULL,
    tool_call_id VARCHAR(100),
    input JSONB NOT NULL,
    output JSONB,
    
    status VARCHAR(50) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled', 'requires_approval')),
    error_message TEXT,
    
    -- Timing
    duration_ms INT,
    queued_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Approval
    requires_approval BOOLEAN DEFAULT false,
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_tool_exec_conversation ON tool_executions(conversation_id);
CREATE INDEX idx_tool_exec_user ON tool_executions(user_id);
CREATE INDEX idx_tool_exec_status ON tool_executions(status);
CREATE INDEX idx_tool_exec_tool_name ON tool_executions(tool_name);
CREATE INDEX idx_tool_exec_pending_approval ON tool_executions(status) 
    WHERE status = 'requires_approval';
CREATE INDEX idx_tool_exec_created ON tool_executions(created_at DESC);
```

### 000005_create_tool_executions_table.down.sql

```sql
DROP TABLE IF EXISTS tool_executions;
```

---

## JWT + Token Authentication

### JWT Utility

```go
// internal/pkg/jwt/jwt.go
package jwt

import (
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type Claims struct {
    UserID    uuid.UUID `json:"uid"`
    TokenID   uuid.UUID `json:"tid"`
    TokenType string    `json:"typ"`
    jwt.RegisteredClaims
}

type JWTManager struct {
    secret          []byte
    accessTokenTTL  time.Duration
    refreshTokenTTL time.Duration
    issuer          string
}

func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
    return &JWTManager{
        secret:          []byte(secret),
        accessTokenTTL:  accessTTL,
        refreshTokenTTL: refreshTTL,
        issuer:          "yourapp",
    }
}

func (m *JWTManager) GenerateAccessToken(userID, tokenID uuid.UUID) (string, error) {
    claims := Claims{
        UserID:    userID,
        TokenID:   tokenID,
        TokenType: "access",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenTTL)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    m.issuer,
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secret)
}

func (m *JWTManager) GenerateRefreshToken(userID, tokenID uuid.UUID) (string, error) {
    claims := Claims{
        UserID:    userID,
        TokenID:   tokenID,
        TokenType: "refresh",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshTokenTTL)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    m.issuer,
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secret)
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return m.secret, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }
    
    return claims, nil
}

// HashToken hashes a raw token for storage
func HashToken(rawToken string) string {
    hash := sha256.Sum256([]byte(rawToken))
    return hex.EncodeToString(hash[:])
}

// GenerateAPIKey generates a random API key
func GenerateAPIKey() string {
    return "sk-" + uuid.New().String()
}
```

### Auth Middleware

```go
// internal/infrastructure/http/middleware/auth.go
package middleware

import (
    "context"
    "net/http"
    "strings"
    
    "yourproject/internal/application/usecase/auth"
    "yourproject/internal/domain/entity"
    "yourproject/internal/pkg/jwt"
    "yourproject/internal/pkg/response"
)

type contextKey string

const (
    UserContextKey  contextKey = "user"
    TokenContextKey contextKey = "token"
)

type AuthMiddleware struct {
    validateToken *auth.ValidateTokenUseCase
}

func NewAuthMiddleware(validateToken *auth.ValidateTokenUseCase) *AuthMiddleware {
    return &AuthMiddleware{validateToken: validateToken}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
            return
        }
        
        // Support both "Bearer <token>" and raw token (for API keys)
        token := strings.TrimPrefix(authHeader, "Bearer ")
        token = strings.TrimSpace(token)
        
        // Validate token
        result, err := m.validateToken.Execute(r.Context(), token)
        if err != nil {
            response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", err.Error())
            return
        }
        
        // Add user and token to context
        ctx := context.WithValue(r.Context(), UserContextKey, result.User)
        ctx = context.WithValue(ctx, TokenContextKey, result.Token)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Optional: For endpoints that work with or without auth
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            next.ServeHTTP(w, r)
            return
        }
        
        token := strings.TrimPrefix(authHeader, "Bearer ")
        token = strings.TrimSpace(token)
        
        result, err := m.validateToken.Execute(r.Context(), token)
        if err == nil {
            ctx := context.WithValue(r.Context(), UserContextKey, result.User)
            ctx = context.WithValue(ctx, TokenContextKey, result.Token)
            r = r.WithContext(ctx)
        }
        
        next.ServeHTTP(w, r)
    })
}

// Helper functions to get user/token from context
func GetUserFromContext(ctx context.Context) *entity.User {
    user, ok := ctx.Value(UserContextKey).(*entity.User)
    if !ok {
        return nil
    }
    return user
}

func GetTokenFromContext(ctx context.Context) *entity.Token {
    token, ok := ctx.Value(TokenContextKey).(*entity.Token)
    if !ok {
        return nil
    }
    return token
}
```

### Rate Limiter Middleware

```go
// internal/infrastructure/http/middleware/ratelimit.go
package middleware

import (
    "context"
    "fmt"
    "net/http"
    "time"
    
    "github.com/redis/go-redis/v9"
    "yourproject/internal/pkg/response"
)

type RateLimiter struct {
    redis *redis.Client
}

func NewRateLimiter(redis *redis.Client) *RateLimiter {
    return &RateLimiter{redis: redis}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := GetTokenFromContext(r.Context())
        if token == nil {
            next.ServeHTTP(w, r)
            return
        }
        
        ctx := r.Context()
        
        // Check per-minute limit
        minuteKey := fmt.Sprintf("ratelimit:%s:minute:%d", token.ID, time.Now().Unix()/60)
        count, err := rl.redis.Incr(ctx, minuteKey).Result()
        if err == nil && count == 1 {
            rl.redis.Expire(ctx, minuteKey, time.Minute)
        }
        
        if count > int64(token.RateLimitPerMinute) {
            w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", token.RateLimitPerMinute))
            w.Header().Set("X-RateLimit-Remaining", "0")
            w.Header().Set("Retry-After", "60")
            response.Error(w, http.StatusTooManyRequests, "RATE_LIMIT", "Rate limit exceeded")
            return
        }
        
        // Check per-day limit
        dayKey := fmt.Sprintf("ratelimit:%s:day:%s", token.ID, time.Now().Format("2006-01-02"))
        dayCount, err := rl.redis.Incr(ctx, dayKey).Result()
        if err == nil && dayCount == 1 {
            rl.redis.Expire(ctx, dayKey, 24*time.Hour)
        }
        
        if dayCount > int64(token.RateLimitPerDay) {
            response.Error(w, http.StatusTooManyRequests, "RATE_LIMIT", "Daily rate limit exceeded")
            return
        }
        
        w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", token.RateLimitPerMinute))
        w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", token.RateLimitPerMinute-int(count)))
        
        next.ServeHTTP(w, r)
    })
}
```

---

## OpenAI-Compatible Tool System

### Tool Types (LiteLLM Compatible)

```go
// pkg/toolspec/openai_format.go
package toolspec

import "encoding/json"

// Tool definition compatible with OpenAI, Anthropic (via LiteLLM), and other providers
type Tool struct {
    Type     string   `json:"type"`     // Always "function"
    Function Function `json:"function"`
}

type Function struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    Parameters  *JSONSchema `json:"parameters,omitempty"`
    Strict      bool        `json:"strict,omitempty"` // OpenAI strict mode
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
    Items       *PropertySchema `json:"items,omitempty"`   // For arrays
    Default     any             `json:"default,omitempty"`
    Minimum     *float64        `json:"minimum,omitempty"`
    Maximum     *float64        `json:"maximum,omitempty"`
}

// ToolCall from LLM response
type ToolCall struct {
    ID       string       `json:"id"`
    Type     string       `json:"type"`     // "function"
    Function FunctionCall `json:"function"`
}

type FunctionCall struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"` // JSON string
}

// ParseArguments parses the JSON arguments into a map
func (tc *ToolCall) ParseArguments() (map[string]any, error) {
    var args map[string]any
    if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
        return nil, err
    }
    return args, nil
}

// ToolMessage for sending results back
type ToolMessage struct {
    Role       string `json:"role"`        // "tool"
    Content    string `json:"content"`     // JSON result
    ToolCallID string `json:"tool_call_id"`
    Name       string `json:"name,omitempty"`
}

// Helper to create a new tool definition
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

// Helper for strict mode tools
func NewStrictTool(name, description string, params *JSONSchema) Tool {
    // Strict mode requires additionalProperties: false
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
```

### Tool Interface & Registry

```go
// internal/application/usecase/tool/tool_registry.go
package tool

import (
    "context"
    "sync"
    
    "yourproject/pkg/toolspec"
)

// Tool interface that all tools must implement
type Tool interface {
    Name() string
    Description() string
    Definition() toolspec.Tool
    Execute(ctx context.Context, args map[string]any) (any, error)
    
    // Optional methods
    RequiresApproval() bool  // Default: false
    Validate(args map[string]any) error  // Default: nil
}

// BaseTool provides default implementations
type BaseTool struct{}

func (BaseTool) RequiresApproval() bool { return false }
func (BaseTool) Validate(args map[string]any) error { return nil }

// ToolRegistry manages all available tools
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

// ListForToken returns tools filtered by token permissions
func (r *ToolRegistry) ListForToken(allowedTools []string) []toolspec.Tool {
    if allowedTools == nil || len(allowedTools) == 0 {
        return r.List()
    }
    
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    allowedMap := make(map[string]bool)
    allowAll := false
    for _, t := range allowedTools {
        if t == "*" {
            allowAll = true
            break
        }
        allowedMap[t] = true
    }
    
    if allowAll {
        return r.List()
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

### Example Tool: Web Search

```go
// internal/application/usecase/tool/builtin/web_search.go
package builtin

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"
    
    "yourproject/internal/application/usecase/tool"
    "yourproject/pkg/toolspec"
)

type WebSearchTool struct {
    tool.BaseTool
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

func NewWebSearchTool(apiKey string) *WebSearchTool {
    return &WebSearchTool{
        apiKey:  apiKey,
        baseURL: "https://api.search.brave.com/res/v1/web/search",
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (t *WebSearchTool) Name() string {
    return "web_search"
}

func (t *WebSearchTool) Description() string {
    return "Search the web for current information. Use this when you need up-to-date information, facts you're unsure about, or to verify claims."
}

func (t *WebSearchTool) Definition() toolspec.Tool {
    return toolspec.NewTool(t.Name(), t.Description(), &toolspec.JSONSchema{
        Type: "object",
        Properties: map[string]toolspec.PropertySchema{
            "query": {
                Type:        "string",
                Description: "The search query. Be specific and use relevant keywords.",
            },
            "num_results": {
                Type:        "integer",
                Description: "Number of results to return (1-10). Default is 5.",
                Default:     5,
            },
        },
        Required: []string{"query"},
    })
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (any, error) {
    query, _ := args["query"].(string)
    if query == "" {
        return nil, fmt.Errorf("query is required")
    }
    
    numResults := 5
    if n, ok := args["num_results"].(float64); ok && n >= 1 && n <= 10 {
        numResults = int(n)
    }
    
    // Build request
    u, _ := url.Parse(t.baseURL)
    q := u.Query()
    q.Set("q", query)
    q.Set("count", fmt.Sprintf("%d", numResults))
    u.RawQuery = q.Encode()
    
    req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    req.Header.Set("X-Subscription-Token", t.apiKey)
    req.Header.Set("Accept", "application/json")
    
    resp, err := t.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("search request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
    }
    
    var result map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }
    
    return t.formatResults(result), nil
}

func (t *WebSearchTool) formatResults(raw map[string]any) map[string]any {
    results := []map[string]string{}
    
    if web, ok := raw["web"].(map[string]any); ok {
        if items, ok := web["results"].([]any); ok {
            for _, item := range items {
                if r, ok := item.(map[string]any); ok {
                    results = append(results, map[string]string{
                        "title":       getString(r, "title"),
                        "url":         getString(r, "url"),
                        "description": getString(r, "description"),
                    })
                }
            }
        }
    }
    
    return map[string]any{
        "results": results,
        "count":   len(results),
    }
}

func getString(m map[string]any, key string) string {
    if v, ok := m[key].(string); ok {
        return v
    }
    return ""
}
```

### Example Tool: Calculator

```go
// internal/application/usecase/tool/builtin/calculator.go
package builtin

import (
    "context"
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "strconv"
    
    "yourproject/internal/application/usecase/tool"
    "yourproject/pkg/toolspec"
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
    // Parse expression
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
```

---

## Eino Workflow Engine

### Agent State

```go
// internal/infrastructure/eino/state/agent_state.go
package state

import (
    "encoding/json"
    "time"
    
    "github.com/google/uuid"
    "yourproject/internal/application/port"
    "yourproject/pkg/toolspec"
)

// AgentState is passed through the Eino workflow graph
type AgentState struct {
    // Identifiers
    ConversationID uuid.UUID `json:"conversation_id"`
    UserID         uuid.UUID `json:"user_id"`
    RequestID      string    `json:"request_id"`
    
    // Messages
    Messages       []port.ChatMessage `json:"messages"`
    SystemPrompt   string             `json:"system_prompt"`
    
    // Current turn
    UserInput      string `json:"user_input"`
    CurrentOutput  string `json:"current_output"`
    
    // Model configuration
    Model          string  `json:"model"`
    Temperature    float64 `json:"temperature"`
    MaxTokens      int     `json:"max_tokens"`
    
    // Tool execution
    AvailableTools []toolspec.Tool     `json:"available_tools"`
    PendingTools   []toolspec.ToolCall `json:"pending_tools"`
    ToolResults    []port.ChatMessage  `json:"tool_results"`
    
    // Multi-agent routing
    CurrentAgent   string   `json:"current_agent"`
    NextAgent      string   `json:"next_agent"`
    AgentHistory   []string `json:"agent_history"`
    
    // Human-in-the-loop
    RequiresApproval bool           `json:"requires_approval"`
    ApprovalReason   string         `json:"approval_reason"`
    ApprovalData     map[string]any `json:"approval_data"`
    IsApproved       *bool          `json:"is_approved,omitempty"`
    
    // Control flow
    Iteration      int  `json:"iteration"`
    MaxIterations  int  `json:"max_iterations"`
    ShouldStop     bool `json:"should_stop"`
    Error          string `json:"error,omitempty"`
    
    // Metrics
    StartTime      time.Time `json:"start_time"`
    TokensUsed     int       `json:"tokens_used"`
    ToolCallsCount int       `json:"tool_calls_count"`
}

func NewAgentState(convID, userID uuid.UUID, input string) *AgentState {
    return &AgentState{
        ConversationID: convID,
        UserID:         userID,
        RequestID:      uuid.New().String(),
        UserInput:      input,
        Messages:       []port.ChatMessage{},
        AvailableTools: []toolspec.Tool{},
        PendingTools:   []toolspec.ToolCall{},
        ToolResults:    []port.ChatMessage{},
        AgentHistory:   []string{},
        ApprovalData:   make(map[string]any),
        Model:          "gpt-4o-mini",
        Temperature:    0.7,
        MaxTokens:      4096,
        MaxIterations:  10,
        Iteration:      0,
        StartTime:      time.Now(),
    }
}

func (s *AgentState) AddMessage(msg port.ChatMessage) {
    s.Messages = append(s.Messages, msg)
}

func (s *AgentState) AddUserMessage(content string) {
    s.Messages = append(s.Messages, port.ChatMessage{
        Role:    "user",
        Content: content,
    })
}

func (s *AgentState) AddAssistantMessage(content string, toolCalls []toolspec.ToolCall) {
    msg := port.ChatMessage{
        Role:    "assistant",
        Content: content,
    }
    if len(toolCalls) > 0 {
        msg.ToolCalls = toolCalls
    }
    s.Messages = append(s.Messages, msg)
}

func (s *AgentState) AddToolResult(toolCallID, name string, result any) {
    content, _ := json.Marshal(result)
    s.ToolResults = append(s.ToolResults, port.ChatMessage{
        Role:       "tool",
        Content:    string(content),
        ToolCallID: toolCallID,
        Name:       name,
    })
}

func (s *AgentState) FlushToolResults() {
    s.Messages = append(s.Messages, s.ToolResults...)
    s.ToolResults = []port.ChatMessage{}
}

func (s *AgentState) Clone() *AgentState {
    clone := *s
    clone.Messages = make([]port.ChatMessage, len(s.Messages))
    copy(clone.Messages, s.Messages)
    clone.PendingTools = make([]toolspec.ToolCall, len(s.PendingTools))
    copy(clone.PendingTools, s.PendingTools)
    clone.AgentHistory = make([]string, len(s.AgentHistory))
    copy(clone.AgentHistory, s.AgentHistory)
    return &clone
}

// ToJSON serializes state for persistence
func (s *AgentState) ToJSON() ([]byte, error) {
    return json.Marshal(s)
}

// FromJSON deserializes state
func FromJSON(data []byte) (*AgentState, error) {
    var s AgentState
    if err := json.Unmarshal(data, &s); err != nil {
        return nil, err
    }
    return &s, nil
}
```

### Chatbot Graph (Eino Workflow)

```go
// internal/infrastructure/eino/graphs/chatbot_graph.go
package graphs

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/cloudwego/eino/compose"
    
    "yourproject/internal/application/port"
    "yourproject/internal/application/usecase/tool"
    "yourproject/internal/infrastructure/eino/state"
)

/*
Workflow Graph:

    ┌─────────┐
    │  START  │
    └────┬────┘
         │
    ┌────▼────┐
    │  Router │ ─────────────────┐
    └────┬────┘                  │
         │                       │
    ┌────▼────┐            ┌─────▼─────┐
    │  Think  │            │   (other  │
    └────┬────┘            │   agents) │
         │                 └───────────┘
    ┌────▼────┐
    │ Decide  │
    └────┬────┘
    ┌────┴─────┬──────────┐
    │          │          │
┌───▼───┐ ┌────▼────┐ ┌───▼────┐
│ Tools │ │ Approve │ │Response│
└───┬───┘ └────┬────┘ └───┬────┘
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
*/

type ChatbotGraph struct {
    llm          port.LLMProvider
    toolRegistry *tool.ToolRegistry
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

func (g *ChatbotGraph) Build() (*compose.Graph[*state.AgentState, *state.AgentState], error) {
    graph := compose.NewGraph[*state.AgentState, *state.AgentState]()
    
    // Add nodes
    graph.AddLambdaNode("router", g.routerNode)
    graph.AddLambdaNode("think", g.thinkNode)
    graph.AddLambdaNode("act", g.actNode)
    graph.AddLambdaNode("observe", g.observeNode)
    graph.AddLambdaNode("human_approval", g.humanApprovalNode)
    graph.AddLambdaNode("response", g.responseNode)
    
    // Define edges
    graph.AddEdge(compose.START, "router")
    
    // Router → agent type
    graph.AddConditionalEdges("router", g.routeAfterRouter, map[string]string{
        "general":    "think",
        "coder":      "think",
        "researcher": "think",
    })
    
    // Think → next step
    graph.AddConditionalEdges("think", g.routeAfterThink, map[string]string{
        "use_tools": "act",
        "respond":   "response",
        "approve":   "human_approval",
    })
    
    // Act → Observe
    graph.AddEdge("act", "observe")
    
    // Observe → continue or finish
    graph.AddConditionalEdges("observe", g.routeAfterObserve, map[string]string{
        "continue": "think",
        "respond":  "response",
        "approve":  "human_approval",
    })
    
    // Human approval → next step
    graph.AddConditionalEdges("human_approval", g.routeAfterApproval, map[string]string{
        "approved": "act",
        "rejected": "response",
        "waiting":  compose.END,
    })
    
    // Response → END
    graph.AddEdge("response", compose.END)
    
    return graph.Compile()
}

// ============ NODE IMPLEMENTATIONS ============

func (g *ChatbotGraph) routerNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
    // Quick routing based on keywords (can be replaced with LLM routing)
    input := s.UserInput
    
    // Simple keyword-based routing
    s.CurrentAgent = "general"
    
    codeKeywords := []string{"code", "function", "program", "debug", "fix", "implement", "class", "api"}
    for _, kw := range codeKeywords {
        if contains(input, kw) {
            s.CurrentAgent = "coder"
            break
        }
    }
    
    researchKeywords := []string{"search", "find", "research", "what is", "who is", "latest", "news", "current"}
    for _, kw := range researchKeywords {
        if contains(input, kw) {
            s.CurrentAgent = "researcher"
            break
        }
    }
    
    s.AgentHistory = append(s.AgentHistory, s.CurrentAgent)
    
    // Set model and system prompt based on agent
    s.Model = g.modelConfig[s.CurrentAgent]
    s.SystemPrompt = g.systemPrompts[s.CurrentAgent]
    
    return s, nil
}

func (g *ChatbotGraph) thinkNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
    // Build messages
    messages := g.buildMessages(s)
    
    // Get available tools
    tools := g.toolRegistry.List()
    s.AvailableTools = tools
    
    // Call LLM
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
    
    // Process response
    choice := resp.Choices[0]
    s.TokensUsed += resp.Usage.TotalTokens
    
    // Store response
    s.CurrentOutput = choice.Message.Content
    s.PendingTools = choice.Message.ToolCalls
    
    // Add to message history
    s.AddAssistantMessage(choice.Message.Content, choice.Message.ToolCalls)
    
    return s, nil
}

func (g *ChatbotGraph) actNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
    s.ToolResults = []port.ChatMessage{}
    
    for _, tc := range s.PendingTools {
        // Parse arguments
        args, err := tc.ParseArguments()
        if err != nil {
            s.AddToolResult(tc.ID, tc.Function.Name, map[string]string{
                "error": fmt.Sprintf("failed to parse arguments: %v", err),
            })
            continue
        }
        
        // Get tool
        t, ok := g.toolRegistry.Get(tc.Function.Name)
        if !ok {
            s.AddToolResult(tc.ID, tc.Function.Name, map[string]string{
                "error": fmt.Sprintf("tool not found: %s", tc.Function.Name),
            })
            continue
        }
        
        // Check if approval required
        if t.RequiresApproval() {
            s.RequiresApproval = true
            s.ApprovalReason = fmt.Sprintf("Tool '%s' requires approval", tc.Function.Name)
            s.ApprovalData = map[string]any{
                "tool":       tc.Function.Name,
                "arguments":  args,
                "tool_call":  tc,
            }
            return s, nil
        }
        
        // Execute tool
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

func (g *ChatbotGraph) observeNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
    // Add tool results to messages
    s.FlushToolResults()
    
    // Increment iteration
    s.Iteration++
    
    // Check if we should stop
    if s.Iteration >= s.MaxIterations {
        s.ShouldStop = true
    }
    
    return s, nil
}

func (g *ChatbotGraph) humanApprovalNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
    // This node pauses the workflow
    // State will be persisted and workflow resumed via API
    return s, nil
}

func (g *ChatbotGraph) responseNode(ctx context.Context, s *state.AgentState) (*state.AgentState, error) {
    // If we have no output, get the last assistant message
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

func (g *ChatbotGraph) routeAfterRouter(s *state.AgentState) string {
    if s.CurrentAgent == "" {
        return "general"
    }
    return s.CurrentAgent
}

func (g *ChatbotGraph) routeAfterThink(s *state.AgentState) string {
    if len(s.PendingTools) > 0 {
        return "use_tools"
    }
    if s.RequiresApproval {
        return "approve"
    }
    return "respond"
}

func (g *ChatbotGraph) routeAfterObserve(s *state.AgentState) string {
    if s.ShouldStop {
        return "respond"
    }
    if s.RequiresApproval {
        return "approve"
    }
    return "continue"
}

func (g *ChatbotGraph) routeAfterApproval(s *state.AgentState) string {
    if s.IsApproved == nil {
        return "waiting"
    }
    if *s.IsApproved {
        return "approved"
    }
    return "rejected"
}

// ============ HELPERS ============

func (g *ChatbotGraph) buildMessages(s *state.AgentState) []port.ChatMessage {
    messages := []port.ChatMessage{}
    
    // Add system prompt
    if s.SystemPrompt != "" {
        messages = append(messages, port.ChatMessage{
            Role:    "system",
            Content: s.SystemPrompt,
        })
    }
    
    // Add conversation history
    messages = append(messages, s.Messages...)
    
    // Add current user input if not already added
    if s.UserInput != "" && (len(s.Messages) == 0 || s.Messages[len(s.Messages)-1].Content != s.UserInput) {
        messages = append(messages, port.ChatMessage{
            Role:    "user",
            Content: s.UserInput,
        })
    }
    
    return messages
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(strings.ToLower(s), strings.ToLower(substr)))
}

func containsLower(s, substr string) bool {
    return strings.Contains(s, substr)
}
```

---

## LLM Provider Integration

### Port Interface

```go
// internal/application/port/llm_provider.go
package port

import (
    "context"
    "yourproject/pkg/toolspec"
)

// ChatMessage represents a message in the conversation
type ChatMessage struct {
    Role       string              `json:"role"`
    Content    string              `json:"content,omitempty"`
    Name       string              `json:"name,omitempty"`
    ToolCalls  []toolspec.ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string              `json:"tool_call_id,omitempty"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
    Model       string            `json:"model"`
    Messages    []ChatMessage     `json:"messages"`
    Tools       []toolspec.Tool   `json:"tools,omitempty"`
    ToolChoice  any               `json:"tool_choice,omitempty"`
    Temperature float64           `json:"temperature,omitempty"`
    MaxTokens   int               `json:"max_tokens,omitempty"`
    Stream      bool              `json:"stream,omitempty"`
    User        string            `json:"user,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
    ID      string     `json:"id"`
    Object  string     `json:"object"`
    Created int64      `json:"created"`
    Model   string     `json:"model"`
    Choices []Choice   `json:"choices"`
    Usage   UsageInfo  `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
    Index        int         `json:"index"`
    Message      ChatMessage `json:"message"`
    FinishReason string      `json:"finish_reason"`
}

// UsageInfo contains token usage information
type UsageInfo struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

// StreamEvent represents a streaming event
type StreamEvent struct {
    Type      string             `json:"type"`
    Delta     string             `json:"delta,omitempty"`
    ToolCall  *toolspec.ToolCall `json:"tool_call,omitempty"`
    Usage     *UsageInfo         `json:"usage,omitempty"`
    Error     error              `json:"-"`
    Done      bool               `json:"done,omitempty"`
}

// LLMProvider is the interface for LLM providers
type LLMProvider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
}
```

### LiteLLM Client

```go
// internal/infrastructure/llm/litellm/client.go
package litellm

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
    
    "yourproject/internal/application/port"
)

type Client struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
    return &Client{
        baseURL: strings.TrimSuffix(baseURL, "/"),
        apiKey:  apiKey,
        httpClient: &http.Client{
            Timeout: 120 * time.Second,
        },
    }
}

func (c *Client) Chat(ctx context.Context, req port.ChatRequest) (*port.ChatResponse, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("do request: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
    }
    
    var chatResp port.ChatResponse
    if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }
    
    return &chatResp, nil
}

func (c *Client) ChatStream(ctx context.Context, req port.ChatRequest) (<-chan port.StreamEvent, error) {
    req.Stream = true
    
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    httpReq.Header.Set("Accept", "text/event-stream")
    
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("do request: %w", err)
    }
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
    }
    
    events := make(chan port.StreamEvent)
    go c.readSSE(ctx, resp, events)
    
    return events, nil
}

func (c *Client) readSSE(ctx context.Context, resp *http.Response, events chan<- port.StreamEvent) {
    defer close(events)
    defer resp.Body.Close()
    
    reader := bufio.NewReader(resp.Body)
    
    for {
        select {
        case <-ctx.Done():
            events <- port.StreamEvent{Error: ctx.Err()}
            return
        default:
        }
        
        line, err := reader.ReadString('\n')
        if err != nil {
            if err != io.EOF {
                events <- port.StreamEvent{Error: err}
            }
            events <- port.StreamEvent{Done: true}
            return
        }
        
        line = strings.TrimSpace(line)
        
        if line == "" {
            continue
        }
        
        if !strings.HasPrefix(line, "data: ") {
            continue
        }
        
        data := strings.TrimPrefix(line, "data: ")
        
        if data == "[DONE]" {
            events <- port.StreamEvent{Done: true}
            return
        }
        
        var chunk struct {
            Choices []struct {
                Delta struct {
                    Content   string              `json:"content"`
                    ToolCalls []toolspec.ToolCall `json:"tool_calls"`
                } `json:"delta"`
                FinishReason string `json:"finish_reason"`
            } `json:"choices"`
            Usage *port.UsageInfo `json:"usage"`
        }
        
        if err := json.Unmarshal([]byte(data), &chunk); err != nil {
            continue
        }
        
        if len(chunk.Choices) > 0 {
            choice := chunk.Choices[0]
            
            if choice.Delta.Content != "" {
                events <- port.StreamEvent{
                    Type:  "content",
                    Delta: choice.Delta.Content,
                }
            }
            
            for _, tc := range choice.Delta.ToolCalls {
                events <- port.StreamEvent{
                    Type:     "tool_call",
                    ToolCall: &tc,
                }
            }
            
            if choice.FinishReason != "" {
                events <- port.StreamEvent{
                    Type:  "finish",
                    Delta: choice.FinishReason,
                    Usage: chunk.Usage,
                }
            }
        }
    }
}
```

---

## HTTP Handlers & Streaming

### Response Helpers

```go
// internal/pkg/response/response.go
package response

import (
    "encoding/json"
    "net/http"
)

type APIResponse struct {
    Success bool   `json:"success"`
    Data    any    `json:"data,omitempty"`
    Error   *APIError `json:"error,omitempty"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(APIResponse{
        Success: true,
        Data:    data,
    })
}

func Error(w http.ResponseWriter, status int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(APIResponse{
        Success: false,
        Error: &APIError{
            Code:    code,
            Message: message,
        },
    })
}

func Stream(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")
}

func SSEEvent(w http.ResponseWriter, event, data string) {
    if event != "" {
        fmt.Fprintf(w, "event: %s\n", event)
    }
    fmt.Fprintf(w, "data: %s\n\n", data)
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
}

func SSEData(w http.ResponseWriter, data any) {
    b, _ := json.Marshal(data)
    fmt.Fprintf(w, "data: %s\n\n", string(b))
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
}
```

### Chat Handler

```go
// internal/infrastructure/http/handler/chat_handler.go
package handler

import (
    "encoding/json"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    
    "yourproject/internal/application/dto/request"
    "yourproject/internal/application/usecase/chat"
    "yourproject/internal/domain/entity"
    "yourproject/internal/infrastructure/http/middleware"
    "yourproject/internal/pkg/response"
)

type ChatHandler struct {
    sendMessage      *chat.SendMessageUseCase
    getConversation  *chat.GetConversationUseCase
    listConversations *chat.ListConversationsUseCase
    approveAction    *chat.ApproveActionUseCase
}

func NewChatHandler(
    sendMessage *chat.SendMessageUseCase,
    getConversation *chat.GetConversationUseCase,
    listConversations *chat.ListConversationsUseCase,
    approveAction *chat.ApproveActionUseCase,
) *ChatHandler {
    return &ChatHandler{
        sendMessage:       sendMessage,
        getConversation:   getConversation,
        listConversations: listConversations,
        approveAction:     approveAction,
    }
}

// POST /v1/chat/completions - OpenAI-compatible endpoint
func (h *ChatHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())
    token := middleware.GetTokenFromContext(r.Context())
    
    var req request.ChatCompletionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
        return
    }
    
    // Check if streaming
    if req.Stream {
        h.handleStreamingChat(w, r, user, token, req)
        return
    }
    
    // Non-streaming response
    result, err := h.sendMessage.Execute(r.Context(), chat.SendMessageInput{
        UserID:   user.ID,
        Messages: req.Messages,
        Model:    req.Model,
        Tools:    req.Tools,
    })
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "CHAT_ERROR", err.Error())
        return
    }
    
    // Return OpenAI-compatible response
    response.JSON(w, http.StatusOK, result.ToOpenAIFormat())
}

func (h *ChatHandler) handleStreamingChat(w http.ResponseWriter, r *http.Request, user *entity.User, token *entity.Token, req request.ChatCompletionRequest) {
    response.Stream(w)
    
    events, err := h.sendMessage.ExecuteStream(r.Context(), chat.SendMessageInput{
        UserID:   user.ID,
        Messages: req.Messages,
        Model:    req.Model,
        Tools:    req.Tools,
    })
    if err != nil {
        response.SSEData(w, map[string]string{"error": err.Error()})
        return
    }
    
    for event := range events {
        if event.Error != nil {
            response.SSEData(w, map[string]string{"error": event.Error.Error()})
            break
        }
        response.SSEData(w, event)
        if event.Done {
            break
        }
    }
    
    response.SSEEvent(w, "", "[DONE]")
}

// POST /v1/conversations - Create conversation and send first message
func (h *ChatHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())
    
    var req request.CreateConversationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
        return
    }
    
    result, err := h.sendMessage.Execute(r.Context(), chat.SendMessageInput{
        UserID:    user.ID,
        Content:   req.Message,
        AgentType: req.AgentType,
    })
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "CHAT_ERROR", err.Error())
        return
    }
    
    response.JSON(w, http.StatusCreated, result)
}

// POST /v1/conversations/:id/messages - Send message to existing conversation
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())
    
    convID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
        return
    }
    
    var req request.SendMessageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
        return
    }
    
    result, err := h.sendMessage.Execute(r.Context(), chat.SendMessageInput{
        ConversationID: &convID,
        UserID:         user.ID,
        Content:        req.Message,
    })
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "CHAT_ERROR", err.Error())
        return
    }
    
    response.JSON(w, http.StatusOK, result)
}

// GET /v1/conversations/:id - Get conversation with messages
func (h *ChatHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())
    
    convID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
        return
    }
    
    result, err := h.getConversation.Execute(r.Context(), convID, user.ID)
    if err != nil {
        response.Error(w, http.StatusNotFound, "NOT_FOUND", "Conversation not found")
        return
    }
    
    response.JSON(w, http.StatusOK, result)
}

// GET /v1/conversations - List user's conversations
func (h *ChatHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())
    
    result, err := h.listConversations.Execute(r.Context(), user.ID)
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
        return
    }
    
    response.JSON(w, http.StatusOK, result)
}

// POST /v1/conversations/:id/approve - Approve or reject pending action
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

### Tool Handler

```go
// internal/infrastructure/http/handler/tool_handler.go
package handler

import (
    "encoding/json"
    "net/http"
    
    "yourproject/internal/application/usecase/tool"
    "yourproject/internal/domain/entity"
    "yourproject/internal/infrastructure/http/middleware"
    "yourproject/internal/pkg/response"
    "yourproject/pkg/toolspec"
)

type ToolHandler struct {
    registry    *tool.ToolRegistry
    executeTool *tool.ExecuteToolUseCase
}

func NewToolHandler(registry *tool.ToolRegistry, executeTool *tool.ExecuteToolUseCase) *ToolHandler {
    return &ToolHandler{
        registry:    registry,
        executeTool: executeTool,
    }
}

// GET /v1/tools - List available tools
func (h *ToolHandler) ListTools(w http.ResponseWriter, r *http.Request) {
    token := middleware.GetTokenFromContext(r.Context())
    
    var tools []toolspec.Tool
    if token != nil {
        tools = h.registry.ListForToken(token.AllowedTools)
    } else {
        tools = h.registry.List()
    }
    
    response.JSON(w, http.StatusOK, map[string]any{
        "tools": tools,
    })
}

// POST /v1/tools/execute - Execute a single tool
func (h *ToolHandler) ExecuteTool(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())
    token := middleware.GetTokenFromContext(r.Context())
    
    var req toolspec.ToolCall
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid tool call format")
        return
    }
    
    result, err := h.executeTool.Execute(r.Context(), tool.ExecuteToolInput{
        ToolCall: req,
        User:     user,
        Token:    token,
    })
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "TOOL_ERROR", err.Error())
        return
    }
    
    response.JSON(w, http.StatusOK, result.Result)
}

// POST /v1/tools/batch - Execute multiple tools in parallel
func (h *ToolHandler) ExecuteToolBatch(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())
    token := middleware.GetTokenFromContext(r.Context())
    
    var calls []toolspec.ToolCall
    if err := json.NewDecoder(r.Body).Decode(&calls); err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid tool calls format")
        return
    }
    
    results := make([]any, len(calls))
    errChan := make(chan error, len(calls))
    
    for i, tc := range calls {
        go func(idx int, call toolspec.ToolCall) {
            result, err := h.executeTool.Execute(r.Context(), tool.ExecuteToolInput{
                ToolCall: call,
                User:     user,
                Token:    token,
            })
            if err != nil {
                errChan <- err
                return
            }
            results[idx] = result.Result
            errChan <- nil
        }(i, tc)
    }
    
    for range calls {
        <-errChan
    }
    
    response.JSON(w, http.StatusOK, map[string]any{"results": results})
}
```

---

## Infrastructure Setup

### Configuration

```go
// internal/infrastructure/config/config.go
package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    Server    ServerConfig
    Database  DatabaseConfig
    Redis     RedisConfig
    JWT       JWTConfig
    RateLimit RateLimitConfig
    LLM       LLMConfig
    Tools     ToolsConfig
}

type ServerConfig struct {
    Port            string
    Host            string
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    IdleTimeout     time.Duration
    ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
    Host            string
    Port            string
    User            string
    Password        string
    Name            string
    SSLMode         string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
    ConnMaxIdleTime time.Duration
}

func (c DatabaseConfig) DSN() string {
    return fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
    )
}

type RedisConfig struct {
    Host         string
    Port         string
    Password     string
    DB           int
    PoolSize     int
    MinIdleConns int
}

func (c RedisConfig) Addr() string {
    return c.Host + ":" + c.Port
}

type JWTConfig struct {
    Secret          string
    AccessTokenTTL  time.Duration
    RefreshTokenTTL time.Duration
}

type RateLimitConfig struct {
    RequestsPerMinute int
    RequestsPerDay    int
    Enabled           bool
}

type LLMConfig struct {
    Provider string  // "litellm", "openai"
    BaseURL  string
    APIKey   string
    DefaultModel string
}

type ToolsConfig struct {
    WebSearchAPIKey string
    MaxConcurrent   int
}

func Load() *Config {
    return &Config{
        Server: ServerConfig{
            Port:            getEnv("SERVER_PORT", "8080"),
            Host:            getEnv("SERVER_HOST", "0.0.0.0"),
            ReadTimeout:     getDuration("SERVER_READ_TIMEOUT", 30*time.Second),
            WriteTimeout:    getDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
            IdleTimeout:     getDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
            ShutdownTimeout: getDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
        },
        Database: DatabaseConfig{
            Host:            getEnv("DB_HOST", "localhost"),
            Port:            getEnv("DB_PORT", "5432"),
            User:            getEnv("DB_USER", "postgres"),
            Password:        getEnv("DB_PASSWORD", "postgres"),
            Name:            getEnv("DB_NAME", "aiagent"),
            SSLMode:         getEnv("DB_SSL_MODE", "disable"),
            MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 25),
            MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 5),
            ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
            ConnMaxIdleTime: getDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
        },
        Redis: RedisConfig{
            Host:         getEnv("REDIS_HOST", "localhost"),
            Port:         getEnv("REDIS_PORT", "6379"),
            Password:     getEnv("REDIS_PASSWORD", ""),
            DB:           getInt("REDIS_DB", 0),
            PoolSize:     getInt("REDIS_POOL_SIZE", 10),
            MinIdleConns: getInt("REDIS_MIN_IDLE_CONNS", 5),
        },
        JWT: JWTConfig{
            Secret:          getEnv("JWT_SECRET", "change-me-in-production-please"),
            AccessTokenTTL:  getDuration("JWT_ACCESS_TTL", 15*time.Minute),
            RefreshTokenTTL: getDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
        },
        RateLimit: RateLimitConfig{
            RequestsPerMinute: getInt("RATE_LIMIT_PER_MINUTE", 60),
            RequestsPerDay:    getInt("RATE_LIMIT_PER_DAY", 10000),
            Enabled:           getBool("RATE_LIMIT_ENABLED", true),
        },
        LLM: LLMConfig{
            Provider:     getEnv("LLM_PROVIDER", "litellm"),
            BaseURL:      getEnv("LLM_BASE_URL", "http://localhost:4000"),
            APIKey:       getEnv("LLM_API_KEY", ""),
            DefaultModel: getEnv("LLM_DEFAULT_MODEL", "gpt-4o-mini"),
        },
        Tools: ToolsConfig{
            WebSearchAPIKey: getEnv("WEB_SEARCH_API_KEY", ""),
            MaxConcurrent:   getInt("TOOLS_MAX_CONCURRENT", 10),
        },
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if i, err := strconv.Atoi(value); err == nil {
            return i
        }
    }
    return defaultValue
}

func getBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if b, err := strconv.ParseBool(value); err == nil {
            return b
        }
    }
    return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if d, err := time.ParseDuration(value); err == nil {
            return d
        }
    }
    return defaultValue
}
```

### PostgreSQL Connection

```go
// internal/infrastructure/persistence/postgres/connection.go
package postgres

import (
    "context"
    "fmt"
    
    "github.com/jackc/pgx/v5/pgxpool"
    "yourproject/internal/infrastructure/config"
)

func NewConnection(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
    poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
    if err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    
    poolConfig.MaxConns = int32(cfg.MaxOpenConns)
    poolConfig.MinConns = int32(cfg.MaxIdleConns)
    poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
    poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime
    
    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    if err != nil {
        return nil, fmt.Errorf("create pool: %w", err)
    }
    
    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("ping: %w", err)
    }
    
    return pool, nil
}
```

### Redis Connection

```go
// internal/infrastructure/persistence/redis/connection.go
package redis

import (
    "context"
    "fmt"
    
    "github.com/redis/go-redis/v9"
    "yourproject/internal/infrastructure/config"
)

func NewConnection(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:         cfg.Addr(),
        Password:     cfg.Password,
        DB:           cfg.DB,
        PoolSize:     cfg.PoolSize,
        MinIdleConns: cfg.MinIdleConns,
    })
    
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("ping: %w", err)
    }
    
    return client, nil
}
```

### Worker Pool

```go
// internal/infrastructure/worker/pool.go
package worker

import (
    "context"
    "sync"
)

type Job func(ctx context.Context) error

type Pool struct {
    workers int
    jobs    chan Job
    wg      sync.WaitGroup
    ctx     context.Context
    cancel  context.CancelFunc
}

func NewPool(ctx context.Context, workers, queueSize int) *Pool {
    ctx, cancel := context.WithCancel(ctx)
    p := &Pool{
        workers: workers,
        jobs:    make(chan Job, queueSize),
        ctx:     ctx,
        cancel:  cancel,
    }
    
    for i := 0; i < workers; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
    
    return p
}

func (p *Pool) worker(id int) {
    defer p.wg.Done()
    
    for {
        select {
        case <-p.ctx.Done():
            return
        case job, ok := <-p.jobs:
            if !ok {
                return
            }
            job(p.ctx)
        }
    }
}

func (p *Pool) Submit(job Job) bool {
    select {
    case p.jobs <- job:
        return true
    case <-p.ctx.Done():
        return false
    default:
        return false // Queue full
    }
}

func (p *Pool) SubmitWait(job Job) bool {
    select {
    case p.jobs <- job:
        return true
    case <-p.ctx.Done():
        return false
    }
}

func (p *Pool) Shutdown() {
    p.cancel()
    close(p.jobs)
    p.wg.Wait()
}
```

### Main Entry Point

```go
// cmd/api/main.go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/go-chi/chi/v5"
    chimiddleware "github.com/go-chi/chi/v5/middleware"
    
    "yourproject/internal/application/usecase/auth"
    "yourproject/internal/application/usecase/chat"
    "yourproject/internal/application/usecase/tool"
    "yourproject/internal/application/usecase/tool/builtin"
    "yourproject/internal/infrastructure/config"
    "yourproject/internal/infrastructure/eino/graphs"
    "yourproject/internal/infrastructure/http/handler"
    "yourproject/internal/infrastructure/http/middleware"
    "yourproject/internal/infrastructure/llm/litellm"
    "yourproject/internal/infrastructure/persistence/postgres"
    "yourproject/internal/infrastructure/persistence/redis"
)

func main() {
    // Load config
    cfg := config.Load()
    
    // Setup logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)
    
    // Context for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Database
    db, err := postgres.NewConnection(ctx, cfg.Database)
    if err != nil {
        slog.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer db.Close()
    
    // Redis
    redisClient, err := redis.NewConnection(ctx, cfg.Redis)
    if err != nil {
        slog.Error("failed to connect to redis", "error", err)
        os.Exit(1)
    }
    defer redisClient.Close()
    
    // Repositories
    userRepo := postgres.NewUserRepository(db)
    tokenRepo := postgres.NewTokenRepository(db)
    convRepo := postgres.NewConversationRepository(db)
    msgRepo := postgres.NewMessageRepository(db)
    
    // LLM Provider
    llmProvider := litellm.NewClient(cfg.LLM.BaseURL, cfg.LLM.APIKey)
    
    // Tool Registry
    toolRegistry := tool.NewToolRegistry()
    toolRegistry.RegisterAll(
        builtin.NewCalculatorTool(),
        builtin.NewWebSearchTool(cfg.Tools.WebSearchAPIKey),
    )
    
    // Eino Graph
    chatbotGraph := graphs.NewChatbotGraph(llmProvider, toolRegistry)
    
    // Use Cases
    validateTokenUC := auth.NewValidateTokenUseCase(tokenRepo, userRepo)
    sendMessageUC := chat.NewSendMessageUseCase(convRepo, msgRepo, chatbotGraph, llmProvider)
    getConversationUC := chat.NewGetConversationUseCase(convRepo, msgRepo)
    listConversationsUC := chat.NewListConversationsUseCase(convRepo)
    approveActionUC := chat.NewApproveActionUseCase(convRepo)
    executeToolUC := tool.NewExecuteToolUseCase(toolRegistry, cfg.Tools.MaxConcurrent)
    
    // Middleware
    authMiddleware := middleware.NewAuthMiddleware(validateTokenUC)
    rateLimiter := middleware.NewRateLimiter(redisClient)
    
    // Handlers
    healthHandler := handler.NewHealthHandler(db, redisClient)
    chatHandler := handler.NewChatHandler(sendMessageUC, getConversationUC, listConversationsUC, approveActionUC)
    toolHandler := handler.NewToolHandler(toolRegistry, executeToolUC)
    
    // Router
    r := chi.NewRouter()
    
    // Global middleware
    r.Use(chimiddleware.RequestID)
    r.Use(chimiddleware.RealIP)
    r.Use(middleware.Logger(logger))
    r.Use(middleware.Recoverer)
    r.Use(middleware.CORS)
    
    // Health routes
    r.Get("/health", healthHandler.Health)
    r.Get("/ready", healthHandler.Ready)
    
    // API routes
    r.Route("/v1", func(r chi.Router) {
        // Public routes
        r.Post("/auth/register", authHandler.Register)
        r.Post("/auth/login", authHandler.Login)
        
        // Protected routes
        r.Group(func(r chi.Router) {
            r.Use(authMiddleware.Authenticate)
            if cfg.RateLimit.Enabled {
                r.Use(rateLimiter.Limit)
            }
            
            // Chat
            r.Post("/chat/completions", chatHandler.ChatCompletions)
            r.Post("/conversations", chatHandler.CreateConversation)
            r.Get("/conversations", chatHandler.ListConversations)
            r.Get("/conversations/{id}", chatHandler.GetConversation)
            r.Post("/conversations/{id}/messages", chatHandler.SendMessage)
            r.Post("/conversations/{id}/approve", chatHandler.ApproveAction)
            
            // Tools
            r.Get("/tools", toolHandler.ListTools)
            r.Post("/tools/execute", toolHandler.ExecuteTool)
            r.Post("/tools/batch", toolHandler.ExecuteToolBatch)
        })
    })
    
    // Server
    server := &http.Server{
        Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
        Handler:      r,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
        IdleTimeout:  cfg.Server.IdleTimeout,
    }
    
    // Start server
    go func() {
        slog.Info("starting server", "addr", server.Addr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    slog.Info("shutting down server...")
    
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
    defer shutdownCancel()
    
    if err := server.Shutdown(shutdownCtx); err != nil {
        slog.Error("server shutdown error", "error", err)
    }
    
    slog.Info("server stopped")
}
```

---

## Docker Configuration

### Multi-stage Dockerfile

```dockerfile
# deployments/Dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o /app/server ./cmd/api

# Final stage
FROM scratch

# Copy certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /app/server /server

# Copy migrations
COPY --from=builder /app/internal/infrastructure/persistence/postgres/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/server"]
```

### Docker Compose

```yaml
# deployments/docker-compose.yml
version: '3.8'

services:
  api:
    build:
      context: ..
      dockerfile: deployments/Dockerfile
      args:
        VERSION: ${VERSION:-dev}
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=8080
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=postgres
      - DB_PASSWORD=postgres
      - DB_NAME=aiagent
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - JWT_SECRET=${JWT_SECRET:-change-me-in-production}
      - LLM_PROVIDER=litellm
      - LLM_BASE_URL=http://litellm:4000
      - LLM_API_KEY=${LLM_API_KEY:-}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 128M
        reservations:
          memory: 64M
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: aiagent
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes --maxmemory 128mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  litellm:
    image: ghcr.io/berriai/litellm:main-latest
    ports:
      - "4000:4000"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY:-}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY:-}
    volumes:
      - ./litellm_config.yaml:/app/config.yaml
    command: ["--config", "/app/config.yaml", "--port", "4000"]

  migrate:
    image: migrate/migrate
    volumes:
      - ../internal/infrastructure/persistence/postgres/migrations:/migrations
    command: [
      "-path", "/migrations",
      "-database", "postgres://postgres:postgres@postgres:5432/aiagent?sslmode=disable",
      "up"
    ]
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  postgres_data:
  redis_data:
```

### LiteLLM Config

```yaml
# deployments/litellm_config.yaml
model_list:
  - model_name: gpt-4o
    litellm_params:
      model: openai/gpt-4o
      api_key: os.environ/OPENAI_API_KEY
  - model_name: gpt-4o-mini
    litellm_params:
      model: openai/gpt-4o-mini
      api_key: os.environ/OPENAI_API_KEY
  - model_name: claude-3-5-sonnet
    litellm_params:
      model: anthropic/claude-3-5-sonnet-20241022
      api_key: os.environ/ANTHROPIC_API_KEY

general_settings:
  master_key: sk-1234  # Change in production
```

---

## Testing Strategy

### Unit Test Example

```go
// internal/application/usecase/tool/execute_tool_test.go
package tool_test

import (
    "context"
    "testing"
    
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    
    "yourproject/internal/application/usecase/tool"
    "yourproject/internal/domain/entity"
    "yourproject/pkg/toolspec"
)

// MockTool for testing
type MockTool struct {
    mock.Mock
}

func (m *MockTool) Name() string                     { return "mock_tool" }
func (m *MockTool) Description() string              { return "A mock tool for testing" }
func (m *MockTool) Definition() toolspec.Tool        { return toolspec.NewTool(m.Name(), m.Description(), nil) }
func (m *MockTool) RequiresApproval() bool           { return false }
func (m *MockTool) Validate(args map[string]any) error { return nil }

func (m *MockTool) Execute(ctx context.Context, args map[string]any) (any, error) {
    called := m.Called(ctx, args)
    return called.Get(0), called.Error(1)
}

func TestExecuteToolUseCase_Success(t *testing.T) {
    // Setup
    registry := tool.NewToolRegistry()
    mockTool := new(MockTool)
    registry.Register(mockTool)
    
    uc := tool.NewExecuteToolUseCase(registry, 10)
    
    expectedResult := map[string]string{"result": "success"}
    mockTool.On("Execute", mock.Anything, map[string]any{"input": "test"}).
        Return(expectedResult, nil)
    
    // Execute
    result, err := uc.Execute(context.Background(), tool.ExecuteToolInput{
        ToolCall: toolspec.ToolCall{
            ID:   "call_123",
            Type: "function",
            Function: toolspec.FunctionCall{
                Name:      "mock_tool",
                Arguments: `{"input": "test"}`,
            },
        },
        User:  &entity.User{ID: uuid.New()},
        Token: &entity.Token{AllowedTools: nil}, // All allowed
    })
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, "call_123", result.ToolCallID)
    assert.Equal(t, expectedResult, result.Content)
    assert.False(t, result.IsError)
    
    mockTool.AssertExpectations(t)
}

func TestExecuteToolUseCase_ToolNotAllowed(t *testing.T) {
    registry := tool.NewToolRegistry()
    mockTool := new(MockTool)
    registry.Register(mockTool)
    
    uc := tool.NewExecuteToolUseCase(registry, 10)
    
    _, err := uc.Execute(context.Background(), tool.ExecuteToolInput{
        ToolCall: toolspec.ToolCall{
            Function: toolspec.FunctionCall{
                Name:      "mock_tool",
                Arguments: "{}",
            },
        },
        User:  &entity.User{ID: uuid.New()},
        Token: &entity.Token{AllowedTools: []string{"other_tool"}}, // mock_tool not allowed
    })
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not allowed")
}
```

### Integration Test Example

```go
// tests/integration/chat_test.go
//go:build integration

package integration

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

func TestChatAPI_SendMessage(t *testing.T) {
    ctx := context.Background()
    
    // Start PostgreSQL container
    postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "postgres:16-alpine",
            ExposedPorts: []string{"5432/tcp"},
            Env: map[string]string{
                "POSTGRES_USER":     "test",
                "POSTGRES_PASSWORD": "test",
                "POSTGRES_DB":       "test",
            },
            WaitingFor: wait.ForListeningPort("5432/tcp"),
        },
        Started: true,
    })
    require.NoError(t, err)
    defer postgres.Terminate(ctx)
    
    // Setup test server
    server := setupTestServer(t, postgres)
    defer server.Close()
    
    // Create test user and token
    token := createTestToken(t, server)
    
    // Send message
    reqBody := map[string]any{
        "model": "gpt-4o-mini",
        "messages": []map[string]string{
            {"role": "user", "content": "Hello, how are you?"},
        },
    }
    body, _ := json.Marshal(reqBody)
    
    req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    
    rec := httptest.NewRecorder()
    server.Handler.ServeHTTP(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
    
    var resp map[string]any
    json.NewDecoder(rec.Body).Decode(&resp)
    
    assert.True(t, resp["success"].(bool))
    assert.NotEmpty(t, resp["data"])
}
```

---

## Makefile

```makefile
.PHONY: all build run test lint migrate docker help

# Variables
BINARY_NAME=server
DOCKER_IMAGE=ai-agent-api
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOFLAGS=-ldflags="-w -s -X main.version=$(VERSION)"

# Database
DATABASE_URL?=postgres://postgres:postgres@localhost:5432/aiagent?sslmode=disable

# Default target
all: lint test build

## Build
build:
	@echo "Building..."
	CGO_ENABLED=0 go build $(GOFLAGS) -o bin/$(BINARY_NAME) ./cmd/api

build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o bin/$(BINARY_NAME)-linux ./cmd/api

## Run
run:
	@echo "Running..."
	go run ./cmd/api

run-watch:
	@echo "Running with hot reload..."
	air -c .air.toml

## Test
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./tests/integration/...

## Lint
lint:
	@echo "Linting..."
	golangci-lint run ./...

## Database
migrate-up:
	@echo "Running migrations up..."
	migrate -path internal/infrastructure/persistence/postgres/migrations -database "$(DATABASE_URL)" up

migrate-down:
	@echo "Running migrations down..."
	migrate -path internal/infrastructure/persistence/postgres/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir internal/infrastructure/persistence/postgres/migrations -seq $$name

migrate-force:
	@read -p "Version to force: " version; \
	migrate -path internal/infrastructure/persistence/postgres/migrations -database "$(DATABASE_URL)" force $$version

## Docker
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -f deployments/Dockerfile .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-up:
	@echo "Starting Docker Compose..."
	docker-compose -f deployments/docker-compose.yml up -d

docker-down:
	@echo "Stopping Docker Compose..."
	docker-compose -f deployments/docker-compose.yml down

docker-logs:
	docker-compose -f deployments/docker-compose.yml logs -f api

docker-ps:
	docker-compose -f deployments/docker-compose.yml ps

## Development
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

generate:
	@echo "Running go generate..."
	go generate ./...

clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

## Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-linux    - Build for Linux"
	@echo "  run            - Run the application"
	@echo "  run-watch      - Run with hot reload"
	@echo "  test           - Run unit tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  lint           - Run linter"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback last migration"
	@echo "  migrate-create - Create new migration"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-up      - Start Docker Compose"
	@echo "  docker-down    - Stop Docker Compose"
	@echo "  docker-logs    - View Docker logs"
	@echo "  dev-deps       - Install development dependencies"
	@echo "  clean          - Clean build artifacts"
```

---

## API Endpoints

```
Health:
  GET  /health                          - Liveness check
  GET  /ready                           - Readiness check

Auth:
  POST /v1/auth/register                - Register new user
  POST /v1/auth/login                   - Login, get JWT tokens
  POST /v1/auth/refresh                 - Refresh access token
  POST /v1/auth/tokens                  - Create API key
  GET  /v1/auth/tokens                  - List user's API keys
  DELETE /v1/auth/tokens/:id            - Revoke API key

Chat (OpenAI Compatible):
  POST /v1/chat/completions             - Chat completion (streaming supported)

Conversations:
  POST /v1/conversations                - Create new conversation
  GET  /v1/conversations                - List conversations
  GET  /v1/conversations/:id            - Get conversation with messages
  POST /v1/conversations/:id/messages   - Send message to conversation
  POST /v1/conversations/:id/approve    - Approve/reject pending action
  DELETE /v1/conversations/:id          - Delete conversation

Tools:
  GET  /v1/tools                        - List available tools
  POST /v1/tools/execute                - Execute single tool
  POST /v1/tools/batch                  - Execute multiple tools

Users:
  GET  /v1/users/me                     - Get current user
  PUT  /v1/users/me                     - Update current user
```

---

## Performance Optimization Checklist

- [ ] Use `sync.Pool` for frequently allocated buffers
- [ ] Pre-allocate slices with `make([]T, 0, expectedSize)`
- [ ] Connection pooling configured for DB and Redis
- [ ] HTTP client reused (not created per request)
- [ ] Context timeout on all external calls
- [ ] Worker pool for tool execution
- [ ] Response streaming for long operations
- [ ] Gzip compression enabled for responses > 1KB
- [ ] Redis caching for hot data
- [ ] Database indexes on frequently queried columns
- [ ] Prepared statements for repeated queries
- [ ] Profile with `pprof` before optimizing
- [ ] Set `GOMAXPROCS` appropriately for containers

---

## Recommended Dependencies

```go
// go.mod
module yourproject

go 1.22

require (
    // Web framework
    github.com/go-chi/chi/v5 v5.0.12
    
    // Database
    github.com/jackc/pgx/v5 v5.5.5
    github.com/redis/go-redis/v9 v9.5.1
    
    // Eino (Workflow)
    github.com/cloudwego/eino v0.3.0
    
    // Auth
    github.com/golang-jwt/jwt/v5 v5.2.1
    golang.org/x/crypto v0.21.0
    
    // Utils
    github.com/google/uuid v1.6.0
    github.com/go-playground/validator/v10 v10.19.0
    
    // Testing
    github.com/stretchr/testify v1.9.0
    github.com/testcontainers/testcontainers-go v0.29.1
)
```

---

## Quick Start Commands

```bash
# 1. Install dependencies
make dev-deps
go mod tidy

# 2. Start infrastructure
make docker-up

# 3. Run migrations
make migrate-up

# 4. Run the server
make run

# Or with hot reload
make run-watch

# 5. Test the API
curl http://localhost:8080/health
```

---

## Claude Code Instructions

When implementing this project, follow this order:

1. **Domain Layer First**
   - Create all entities
   - Define repository interfaces
   - No external dependencies

2. **Infrastructure Layer**
   - Database connections
   - Repository implementations
   - LLM client

3. **Tool System**
   - Tool interface and registry
   - Built-in tools (calculator, web_search)
   - Execute use case

4. **Eino Workflow**
   - Agent state
   - Graph builder
   - Node implementations

5. **Application Layer**
   - Use cases for chat
   - Use cases for auth

6. **HTTP Layer**
   - Middleware
   - Handlers
   - Router

7. **Testing & Docker**
   - Unit tests
   - Integration tests
   - Docker configuration

Start with: **"Begin with the domain entities: User, Token, Conversation, Message"**
