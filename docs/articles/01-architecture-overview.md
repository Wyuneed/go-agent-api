# I Built a Production-Ready AI Agent API in Go — Here's the Architecture That Makes It Actually Work

*Part 1 of the "Building Production-Ready AI Agent APIs in Go" series*

---

Most AI agent tutorials give you a Python script. Some give you a FastAPI app with a few routes. Almost none give you something you would actually run in production.

I built one in Go. It has JWT authentication, API key management, rate limiting, a full workflow engine, streaming responses, human-in-the-loop approvals, and an 18MB Docker image. It runs on a $6/month VPS with plenty of headroom.

This article is the overview. I will walk you through every architectural decision and why we made it. The rest of the series goes deep on each piece.

---

## Why Go for AI Agents?

Python is dominant in AI. The tooling is excellent. But if you are building an API that serves AI agents in production, Go has concrete advantages:

**Memory footprint.** A typical Go HTTP server uses 10-20MB of RAM at idle. A Python FastAPI equivalent uses 80-150MB. For an API handling concurrent agent sessions, this matters.

**Cold start.** Go binaries start in milliseconds. No interpreter to spin up, no imports to resolve at runtime. If you are running in containers that scale to zero, this is the difference between a 50ms cold start and a 2-second one.

**Single binary.** The entire application, with all dependencies compiled in, is one executable. No `requirements.txt`, no virtual environments, no `pip install` in your Dockerfile. Copy the binary, run it.

**Concurrency model.** Goroutines are cheap (2KB stack initially) and Go's runtime scheduler handles thousands of them efficiently. Running 100 concurrent agent sessions with streaming responses is straightforward.

**Type safety.** When the LLM returns a tool call, you know exactly what fields it has. When you pass state through a workflow graph, the compiler catches shape mismatches at compile time.

None of this means Python is wrong for AI. The model inference, fine-tuning, and research ecosystem is irreplaceable in Python. But the API serving layer? Go is an excellent fit.

---

## The Five Pillars

The codebase is built on five architectural pillars:

```
┌─────────────────────────────────────────────────────────────┐
│ 1. DDD — Clean layer separation keeps code maintainable     │
│ 2. Eino — Workflow engine manages agentic reasoning loops   │
│ 3. OpenAI Format — Portable tool calling across LLM providers│
│ 4. Dual Auth — JWT sessions + API keys with per-key ACLs    │
│ 5. SSE Streaming — Real-time responses without WebSockets   │
└─────────────────────────────────────────────────────────────┘
```

These are not optional features you can swap out. They compose together. The auth system controls which tools a token can use. The tool system feeds into the Eino workflow. The workflow streams its output via SSE. The DDD layers keep each piece independently testable.

---

## Walking the Directory Tree

Here is the project structure with the "why" for each layer:

```
go-agent-api/
├── cmd/api/main.go              ← DI container: wires everything together
│
├── internal/
│   ├── domain/                  ← Pure Go. Zero external dependencies.
│   │   ├── entity/              ← User, Token, Conversation, Message
│   │   ├── repository/          ← Interfaces only (no implementations here)
│   │   ├── service/             ← Domain services (password hashing logic)
│   │   └── event/               ← Domain events (MessageCreated, etc.)
│   │
│   ├── application/             ← Orchestration. Depends only on domain.
│   │   ├── usecase/             ← auth/, chat/, tool/, user/
│   │   ├── dto/                 ← Request/response shapes
│   │   └── port/                ← Interfaces for external services
│   │
│   └── infrastructure/          ← The outside world. Depends on application.
│       ├── eino/                ← Workflow graph and state
│       ├── http/                ← Handlers, middleware, router
│       ├── llm/                 ← LiteLLM client
│       ├── persistence/         ← PostgreSQL repos + Redis
│       └── config/              ← Environment variable loading
│
├── pkg/toolspec/                ← Public: OpenAI-compatible tool types
├── deployments/                 ← Dockerfile + docker-compose.yml
└── docs/                        ← Swagger spec + UI
```

The most important thing to understand is the direction of dependencies:

```
Domain ← Application ← Infrastructure
```

The domain layer does not know about PostgreSQL, Redis, HTTP, or LiteLLM. The application layer does not know about chi, pgx, or docker-compose. The infrastructure layer implements the interfaces that the inner layers define.

This is not theoretical purity — it has practical consequences. When you want to write a unit test for the `SendMessage` use case, you mock the repository interface and the LLM provider interface. No database, no HTTP server, no running LiteLLM instance needed. The test runs in milliseconds.

---

## The Dependency Rule in 30 Seconds

The dependency rule states that source code dependencies must point inward. Nothing in the inner circles knows anything about the outer circles.

```go
// DOMAIN: knows nothing about PostgreSQL
// internal/domain/repository/user_repository.go
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*entity.User, error)
}

// INFRASTRUCTURE: implements the domain interface
// internal/infrastructure/persistence/postgres/user_repository.go
type userRepository struct {
    pool *pgxpool.Pool
}
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
    // pgx query here
}
```

The domain defines `UserRepository` as an interface. The infrastructure provides `userRepository` as a concrete PostgreSQL implementation. The domain has zero knowledge that PostgreSQL exists.

This pattern is applied everywhere. `port.LLMProvider` in the application layer is an interface. `litellm.Client` in the infrastructure layer implements it. Swap LiteLLM for a direct OpenAI client by writing a new struct that satisfies the interface.

---

## main.go: The Entire DI Container, Explicit and Readable

The entry point in `cmd/api/main.go` is 200 lines of explicit dependency injection. No magic, no framework, no annotations. Here is the core of it:

```go
// Database
db, err := postgres.NewConnection(ctx, cfg.Database)

// Redis
redisClient, err := redis.NewConnection(ctx, cfg.Redis)

// Repositories (infrastructure layer)
userRepo := postgres.NewUserRepository(db)
tokenRepo := postgres.NewTokenRepository(db)
convRepo := postgres.NewConversationRepository(db)
msgRepo := postgres.NewMessageRepository(db)

// JWT Manager
jwtMgr := jwt.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)

// LLM Provider (infrastructure layer)
llmProvider := litellm.NewClient(cfg.LLM.BaseURL, cfg.LLM.APIKey)

// Tool Registry (application layer)
toolRegistry := tool.NewToolRegistry()
toolRegistry.RegisterAll(
    builtin.NewCalculatorTool(),
    builtin.NewWebSearchTool(cfg.Tools.WebSearchAPIKey),
)

// Use Cases (application layer)
validateTokenUC := auth.NewValidateTokenUseCase(tokenRepo, userRepo, jwtMgr)
loginUC := auth.NewLoginUseCase(userRepo, tokenRepo, jwtMgr)
sendMessageUC := chat.NewSendMessageUseCase(convRepo, msgRepo, llmProvider, cfg.LLM.DefaultModel)
approveActionUC := chat.NewApproveActionUseCase(convRepo)

// Middleware + Handlers (infrastructure layer)
authMiddleware := middleware.NewAuthMiddleware(validateTokenUC)
chatHandler := handler.NewChatHandler(sendMessageUC, getConversationUC, listConversationsUC, approveActionUC)
```

Every dependency is explicit. You can read from top to bottom and understand exactly what depends on what. There are no hidden singletons, no global variables, no container magic.

This explicitness makes debugging trivial. If `chatHandler` misbehaves, you can see exactly what `sendMessageUC` depends on, what `llmProvider` it uses, and which `convRepo` implementation backs it. The entire dependency graph is right there in one file.

---

## docker-compose up: 5 Services, One Command

The `deployments/docker-compose.yml` gives you a complete development and production stack:

```yaml
services:
  api:         # Your Go binary (128MB memory limit)
  postgres:    # PostgreSQL 16 (primary database)
  redis:       # Redis 7 (rate limiting, caching)
  litellm:     # LLM proxy (GPT-4o, Claude, Gemini — config-driven)
  migrate:     # golang-migrate (runs DB migrations on startup)
```

The services are wired with proper health checks:

```yaml
depends_on:
  postgres:
    condition: service_healthy
  redis:
    condition: service_healthy
```

The API container will not start until both Postgres and Redis are accepting connections. The migrate container runs migrations before the API processes any traffic. This is a production-grade startup sequence, not just "hope the database is ready."

---

## What You Can Build With 10 New Lines of Code

Once the infrastructure is running, extending it is intentionally easy. Here is what adding a new capability looks like:

**Add a new tool (file + 2 lines in main.go):**
```go
// New file: internal/application/usecase/tool/builtin/weather.go
type WeatherTool struct{ tool.BaseTool }
func (t *WeatherTool) Name() string { return "get_weather" }
// ... implement Execute()

// In main.go:
toolRegistry.RegisterAll(builtin.NewWeatherTool(cfg.Tools.WeatherAPIKey))
```

**Add a new API key with restricted tools:**
```bash
# Create a token that can ONLY use calculator
POST /v1/auth/tokens
{"name": "limited-key", "allowed_tools": ["calculator"]}
```

**Switch from GPT-4o to Claude:**
```yaml
# deployments/litellm_config.yaml — change one line
- model_name: gpt-4o-mini
  litellm_params:
    model: anthropic/claude-3-haiku-20240307
    api_key: os.environ/ANTHROPIC_API_KEY
```

No code changes. LiteLLM handles the API format translation.

---

## The Full API Surface

The server exposes 18 endpoints across 4 domains:

```
Health:
  GET  /health                          # Liveness check
  GET  /ready                           # Readiness check

Auth (public):
  POST /v1/auth/register
  POST /v1/auth/login
  POST /v1/auth/refresh

Chat (protected):
  POST /v1/chat/completions             # OpenAI-compatible, streaming supported
  POST /v1/conversations
  GET  /v1/conversations
  GET  /v1/conversations/{id}
  POST /v1/conversations/{id}/messages
  POST /v1/conversations/{id}/approve   # Human-in-the-loop

Tools (protected):
  GET  /v1/tools
  POST /v1/tools/execute
  POST /v1/tools/batch
```

There is also a Swagger UI at `http://127.0.0.1:8080/swagger/index.html` with interactive documentation for every endpoint.

---

## The Dependency Stack

The full `go.mod` has 16 direct dependencies:

| Package | Purpose |
|---------|---------|
| `github.com/cloudwego/eino` | Workflow engine (ByteDance) |
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/golang-jwt/jwt/v5` | JWT tokens |
| `github.com/jackc/pgx/v5` | PostgreSQL driver |
| `github.com/redis/go-redis/v9` | Redis client |
| `github.com/google/uuid` | UUID generation |
| `golang.org/x/crypto` | bcrypt password hashing |
| `github.com/stretchr/testify` | Test assertions |
| `github.com/swaggo/swag` | Swagger generation |

No ORMs. No dependency injection frameworks. No "enterprise" middleware. Every dependency does one specific thing.

---

## Series Roadmap

Here is what each article in this series covers:

| # | Title | What You Learn |
|---|-------|----------------|
| 1 | Architecture Overview (this article) | Why these choices, how the pieces fit |
| 2 | DDD Domain Layer | Entities with business logic, repository interfaces |
| 3 | The Tool System | Add a tool in 15 lines, OpenAI format, per-token ACLs |
| 4 | Eino Workflow Engine | 6-node DAG, ReAct loop, conditional routing |
| 5 | Dual Auth | JWT + API keys, token-as-permission-carrier |
| 6 | SSE Streaming | Real-time responses, channel pipeline, context cancellation |
| 7 | Human-in-the-Loop | Pause/approve/resume agentic workflows |
| 8 | Deployment | 18MB Docker image, graceful shutdown, LiteLLM |

Each article stands alone but is richer in context when read in sequence.

---

## What We Just Learned

- Go's memory efficiency, cold start speed, and type safety make it well-suited for AI agent APIs
- The project uses DDD with strict inward dependency: Domain ← Application ← Infrastructure
- `cmd/api/main.go` is an explicit DI container — every dependency is visible and readable
- One `docker-compose up` brings up 5 services with health checks and automatic migrations
- Adding new tools, switching LLM providers, and creating restricted API keys all take single-digit lines of code

---

## Try This Now

```bash
git clone https://github.com/wyuneed/go-agent-api
cd go-agent-api
cp .env.example .env
make docker-up
make migrate-up
make run
# Open http://127.0.0.1:8080/swagger/index.html
```

---

*Next in the series: [Article 2 — Domain-Driven Design for Go Developers](#)*
