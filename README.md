<div align="center">

# Go Agent API

**Production-ready AI Agent API framework built with Go and Domain-Driven Design**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Architecture](https://img.shields.io/badge/architecture-DDD-orange)](#architecture)

[Features](#features) · [Quick Start](#quick-start) · [Architecture](#architecture) · [API Reference](#api-reference) · [Configuration](#configuration) · [Contributing](#contributing)

</div>

---

## Overview

Go Agent API is a production-grade backend framework for building AI-powered agents. It combines **Domain-Driven Design (DDD)** with the **Eino workflow engine** (ByteDance's LangGraph equivalent for Go) to deliver a clean, extensible, and high-performance agentic system.

It ships with JWT authentication, an OpenAI-compatible tool calling system, human-in-the-loop approvals, streaming responses via SSE, and full Docker support — everything you need to go from idea to deployed agent.

**Why Go?** Single binary deployment, ~18MB Docker image from scratch, sub-millisecond cold starts, and a memory footprint orders of magnitude smaller than Python equivalents.

---

## Features

| Category | Capability |
|---|---|
| **Workflows** | Eino DAG-based agentic loops with router → think → act → observe nodes |
| **LLM Compatibility** | OpenAI tool calling format; LiteLLM proxy for multi-provider support (OpenAI, Anthropic, etc.) |
| **Authentication** | JWT access + refresh tokens, API keys, DB-managed expiration, per-token tool/model ACLs |
| **Human-in-the-Loop** | Pause workflow for human approval of sensitive tool calls; resume via API |
| **Tool System** | Extensible registry with built-in calculator & web search; per-token tool restrictions |
| **Streaming** | Server-Sent Events (SSE) for real-time streaming chat responses |
| **Rate Limiting** | Per-token per-minute and per-day limits enforced in Redis |
| **Persistence** | PostgreSQL with pgx/v5 connection pooling; 5 managed migration tables |
| **Caching** | Redis for token validation, rate limiting, and conversation state |
| **Observability** | Structured JSON logging (slog), request IDs, panic recovery |
| **Deployment** | Multi-stage Docker build from scratch (~18MB image), Docker Compose full stack |
| **Testing** | 39 unit tests with race detector; testify mocks; integration test harness |

---

## Tech Stack

```
Go 1.24          — Language
Chi v5           — HTTP router & middleware
Eino v0.7        — Workflow engine (ByteDance, LangGraph equivalent)
pgx/v5           — PostgreSQL driver with connection pooling
go-redis/v9      — Redis client
golang-jwt/v5    — JWT implementation
golang-migrate   — Database migrations
LiteLLM          — Multi-provider LLM proxy (OpenAI, Anthropic, etc.)
Docker / Compose — Containerization
```

---

## Quick Start

### Option A — Docker Compose (Recommended)

The fastest way to run the full stack locally.

**1. Clone**
```bash
git clone https://github.com/wyuneed/go-agent-api.git
cd go-agent-api
```

**2. Configure environment**
```bash
cp .env.example .env
```

Edit `.env` and set at minimum:
```env
JWT_SECRET=your-super-secret-key-min-32-chars

# Add your LLM provider key (pick one):
OPENAI_API_KEY=sk-...
# or
ANTHROPIC_API_KEY=sk-ant-...
```

**3. Start all services**
```bash
make docker-up
# Starts: API · PostgreSQL 16 · Redis 7 · LiteLLM proxy
```

**4. Run database migrations**
```bash
make migrate-up
```

**5. Verify**
```bash
curl -4 http://localhost:8080/health
# {"success":true,"data":{"status":"ok"}}

curl -4 http://localhost:8080/ready
# {"success":true,"data":{"status":"ready"}}
```

> **Note:** Use `curl -4` to force IPv4 if you get "Connection reset by peer".

---

### Option B — Local Development (with hot reload)

**Prerequisites:** Go 1.24+, PostgreSQL 14+, Redis 7+

```bash
# Install dev tools (golangci-lint, air, golang-migrate)
make dev-deps

# Start only infrastructure
docker-compose -f deployments/docker-compose.yml up postgres redis -d

# Copy and edit environment
cp .env.example .env

# Run migrations
make migrate-up

# Start with hot reload
make run-watch
# Server on http://localhost:8080
```

---

## First API Call

**Register:**
```bash
curl -4 -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"SecurePass123","name":"Your Name"}'
```

**Login:**
```bash
curl -4 -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"SecurePass123"}'
# Returns access_token + refresh_token
```

**Chat:**
```bash
TOKEN="eyJ..."

curl -4 -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "What is 1337 * 42?"}]
  }'
```

**Execute a tool directly:**
```bash
cat > /tmp/req.json << 'EOF'
{
  "id": "call_1",
  "type": "function",
  "function": {
    "name": "calculator",
    "arguments": "{\"expression\": \"1337 * 42\"}"
  }
}
EOF

curl -4 -X POST http://localhost:8080/v1/tools/execute \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @/tmp/req.json
# {"success":true,"data":{"expression":"1337 * 42","result":56154}}
```

---

## Architecture

This project follows **Clean Architecture** with Domain-Driven Design. Dependencies point strictly inward — the domain layer has zero external dependencies.

```
┌──────────────────────────────────────────────────────┐
│                  Infrastructure Layer                 │
│   HTTP handlers · DB repos · Redis · LLM · Eino      │
├──────────────────────────────────────────────────────┤
│                  Application Layer                    │
│       Use cases · DTOs · Ports (interfaces)           │
├──────────────────────────────────────────────────────┤
│                    Domain Layer                       │
│   Entities · Value Objects · Domain Services         │
│          Repository Interfaces · Events               │
└──────────────────────────────────────────────────────┘
            Dependencies point INWARD only ↑
```

### Eino Workflow Graph

Every chat request flows through a 6-node directed acyclic graph:

```
START
  │
  ▼
[Router] ── keyword routing → general / coder / researcher agent
  │
  ▼
[Think] ── LLM call; decides: use tools | respond | request approval
  │
  ├─ use_tools ──► [Act] ── execute tools (parallel)
  │                  │
  │                  ▼
  │             [Observe] ── add results to messages; check iteration limit
  │                  │
  │                  ├─ continue ──► [Think]  (agentic loop)
  │                  └─ respond  ──► [Response]
  │
  ├─ approve ──► [HumanApproval] ── pause; resume via POST /approve
  │
  └─ respond ──► [Response] ──► END
```

### Project Structure

```
go-agent-api/
├── cmd/api/main.go                        # Entry point & full DI wiring
│
├── internal/
│   ├── domain/                            # ← Pure Go, zero external deps
│   │   ├── entity/                        # User, Token, Conversation, Message
│   │   ├── valueobject/                   # Email, MessageRole, ToolCall
│   │   ├── repository/                    # Interface definitions only
│   │   ├── service/                       # Auth & Conversation domain services
│   │   └── event/                         # MessageCreated, ToolExecuted, etc.
│   │
│   ├── application/
│   │   ├── usecase/
│   │   │   ├── auth/                      # Login, ValidateToken, RefreshToken
│   │   │   ├── chat/                      # SendMessage, GetConversation, Approve
│   │   │   ├── tool/                      # Registry, ExecuteTool + built-ins
│   │   │   └── user/                      # CreateUser
│   │   ├── dto/request/                   # Request DTOs
│   │   └── port/                          # LLMProvider, Cache, EventPublisher
│   │
│   └── infrastructure/
│       ├── eino/                          # Workflow graphs & agent state
│       ├── http/                          # Chi router, middleware, handlers
│       ├── llm/litellm/                   # HTTP client + SSE streaming
│       ├── persistence/
│       │   ├── postgres/                  # pgx/v5 repo implementations
│       │   │   └── migrations/            # 5 SQL migration files (up + down)
│       │   └── redis/                     # Cache implementation
│       └── config/                        # Env-based config loader
│
├── pkg/toolspec/                          # Public: OpenAI-compatible tool types
│
├── tests/
│   ├── mocks/                             # testify/mock for all repo interfaces
│   └── integration/                       # HTTP handler integration tests
│
├── deployments/
│   ├── Dockerfile                         # Multi-stage → scratch (~18MB)
│   ├── docker-compose.yml                 # Full dev stack (5 services)
│   └── litellm_config.yaml                # LiteLLM model routing
│
├── Makefile                               # 20+ targets
└── .env.example                           # All env variables documented
```

---

## API Reference

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/auth/register` | Create account |
| `POST` | `/v1/auth/login` | Login, receive JWT pair |
| `POST` | `/v1/auth/refresh` | Refresh access token |

**Login response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

---

### Chat (OpenAI-Compatible)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/chat/completions` | Chat completion (streaming supported) |
| `POST` | `/v1/conversations` | Create conversation + first message |
| `GET`  | `/v1/conversations` | List user conversations |
| `GET`  | `/v1/conversations/:id` | Get conversation with all messages |
| `POST` | `/v1/conversations/:id/messages` | Continue a conversation |
| `POST` | `/v1/conversations/:id/approve` | Approve or reject pending action |

All protected routes require `Authorization: Bearer <access_token>`.

**Streaming (SSE):**
```bash
curl -4 -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Tell me a story"}],"stream":true}'

# data: {"type":"content","delta":"Once"}
# data: {"type":"content","delta":" upon"}
# data: [DONE]
```

**Human-in-the-loop approval:**
```bash
# When conversation status is "pending_approval":
curl -4 -X POST http://localhost:8080/v1/conversations/$CONV_ID/approve \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"approved": true, "reason": "Looks safe"}'
```

---

### Tools

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET`  | `/v1/tools` | List tools available to this token |
| `POST` | `/v1/tools/execute` | Execute a single tool |
| `POST` | `/v1/tools/batch` | Execute multiple tools in parallel |

---

### Health

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Liveness (always 200 if process is up) |
| `GET` | `/ready` | Readiness (checks DB + Redis connectivity) |

---

## Configuration

All config is via environment variables. See `.env.example` for the full list.

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP listen port |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `aiagent` | Database name |
| `DB_SSL_MODE` | `disable` | Use `require` in production |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PASSWORD` | — | Redis password (set in production) |
| `JWT_SECRET` | — | **Required.** Min 32 random chars |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime (7 days) |
| `LLM_BASE_URL` | `http://localhost:4000` | LiteLLM or OpenAI base URL |
| `LLM_API_KEY` | — | API key for LLM provider |
| `LLM_DEFAULT_MODEL` | `gpt-4o-mini` | Default model for new conversations |
| `OPENAI_API_KEY` | — | OpenAI key (passed to LiteLLM) |
| `ANTHROPIC_API_KEY` | — | Anthropic key (passed to LiteLLM) |
| `WEB_SEARCH_API_KEY` | — | Brave Search API key |
| `RATE_LIMIT_ENABLED` | `true` | Enable rate limiting |
| `RATE_LIMIT_PER_MINUTE` | `60` | Max requests/minute per token |
| `RATE_LIMIT_PER_DAY` | `10000` | Max requests/day per token |

---

## Adding a Custom Tool

Implement the `Tool` interface and register it in `main.go`:

```go
// internal/application/usecase/tool/builtin/my_tool.go
package builtin

import (
    "context"
    "github.com/wyuneed/go-agent-api/internal/application/usecase/tool"
    "github.com/wyuneed/go-agent-api/pkg/toolspec"
)

type MyTool struct{ tool.BaseTool }

func NewMyTool() *MyTool { return &MyTool{} }

func (t *MyTool) Name() string        { return "my_tool" }
func (t *MyTool) Description() string { return "Does something useful." }

func (t *MyTool) Definition() toolspec.Tool {
    return toolspec.NewTool(t.Name(), t.Description(), &toolspec.JSONSchema{
        Type: "object",
        Properties: map[string]toolspec.PropertySchema{
            "input": {Type: "string", Description: "Input value"},
        },
        Required: []string{"input"},
    })
}

func (t *MyTool) Execute(ctx context.Context, args map[string]any) (any, error) {
    input, _ := args["input"].(string)
    return map[string]string{"result": "processed: " + input}, nil
}
```

Then in `cmd/api/main.go`:
```go
toolRegistry.RegisterAll(
    builtin.NewCalculatorTool(),
    builtin.NewWebSearchTool(cfg.Tools.WebSearchAPIKey),
    builtin.NewMyTool(), // ← add here
)
```

To require human approval before a tool runs, override `RequiresApproval`:
```go
func (t *MyTool) RequiresApproval() bool { return true }
```

---

## Database Migrations

```bash
make migrate-up      # Apply all pending migrations
make migrate-down    # Roll back last migration
make migrate-create  # Scaffold new up/down files
```

Migration files are in [internal/infrastructure/persistence/postgres/migrations/](internal/infrastructure/persistence/postgres/migrations/):

| Version | Table |
|---------|-------|
| 1 | `users` |
| 2 | `user_tokens` |
| 3 | `conversations` |
| 4 | `messages` |
| 5 | `tool_executions` |

---

## Testing

```bash
# Unit tests with race detector
make test

# Generate HTML coverage report
make test-coverage

# Integration tests (requires running DB + Redis)
make test-integration

# Specific package
go test -v ./internal/domain/entity/...
```

**Coverage summary:**
- `internal/domain/entity/` — User, Token, Conversation, Message
- `internal/domain/service/` — AuthDomainService (7 cases)
- `internal/application/usecase/tool/` — Registry, ExecuteTool
- `internal/application/usecase/tool/builtin/` — Calculator (9 expressions)
- `internal/pkg/jwt/` — Generate, validate, expire, hash

---

## Deployment

### Docker

```bash
# Build ~18MB production image
make docker-build

# Run standalone
docker run -d \
  -e DB_HOST=your-db-host \
  -e REDIS_HOST=your-redis-host \
  -e JWT_SECRET=your-secret \
  -e OPENAI_API_KEY=sk-... \
  -p 8080:8080 \
  ai-agent-api:latest
```

### Production Checklist

- [ ] Set a strong `JWT_SECRET` (32+ random characters)
- [ ] Set `DB_SSL_MODE=require` for managed PostgreSQL
- [ ] Set `REDIS_PASSWORD` for protected Redis
- [ ] Set real `OPENAI_API_KEY` / `ANTHROPIC_API_KEY`
- [ ] Change LiteLLM `master_key` in `deployments/litellm_config.yaml`
- [ ] Set up PostgreSQL automated backups
- [ ] Add uptime monitoring on the `/ready` endpoint
- [ ] Never commit `.env` — use a secrets manager in production

---

## Makefile Reference

```
make build            Build binary → bin/server
make build-linux      Cross-compile for Linux/amd64
make run              Run locally
make run-watch        Run with air hot reload
make test             Unit tests (race detector)
make test-coverage    Tests + HTML coverage report
make test-integration Integration tests
make lint             golangci-lint
make migrate-up       Apply pending migrations
make migrate-down     Roll back last migration
make migrate-create   Scaffold new migration
make docker-build     Build Docker image
make docker-up        docker-compose up -d
make docker-down      docker-compose down
make docker-logs      Follow API container logs
make docker-ps        List running containers
make dev-deps         Install golangci-lint, air, migrate
make clean            Remove build artifacts
```

---

## Troubleshooting

<details>
<summary><strong>curl: Connection reset by peer</strong></summary>

curl may prefer IPv6. Force IPv4 with `-4`:
```bash
curl -4 http://localhost:8080/health
```
</details>

<details>
<summary><strong>Dirty database version</strong></summary>

A migration failed halfway. Force to the last clean version, then re-run:
```bash
# Check current state
migrate -path internal/infrastructure/persistence/postgres/migrations \
  -database "postgres://postgres:postgres@localhost:5432/aiagent?sslmode=disable" version

# Force to last clean version (e.g., 2)
migrate ... force 2

make migrate-up
```
</details>

<details>
<summary><strong>migrate: command not found</strong></summary>

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```
</details>

<details>
<summary><strong>LLM returns 401 Authentication Error</strong></summary>

Set your API key in `.env`, then restart:
```env
OPENAI_API_KEY=sk-...
```
```bash
make docker-down && make docker-up
```
</details>

<details>
<summary><strong>bytedance/sonic build error on Go 1.24</strong></summary>

```bash
go get github.com/bytedance/sonic@latest
go mod tidy
```
</details>

---

## Contributing

1. **Respect DDD boundaries** — business logic stays in domain/application layers
2. **Write tests** — use `tests/mocks/` for repository mocks; add unit tests alongside source
3. **Lint** — `make lint` must pass
4. **Migrations** — always add both `up` and `down` files; never edit existing migrations
5. **Commits** — use conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`

---

## License

[MIT](LICENSE)

---

<div align="center">
Built with Go · Eino · PostgreSQL · Redis
</div>
