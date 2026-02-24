# Shipping a Go AI Agent to Production: 18MB Docker Image, 5 Services, Zero Downtime

*Part 8 of the "Building Production-Ready AI Agent APIs in Go" series*

---

The most common advice for deploying Go applications is: "compile a binary and run it." That is true, but production requires more: automated database migrations, a proxy for LLM providers, a cache layer, health checks, graceful shutdown, and resource limits.

This article walks through every aspect of deploying this AI agent API to production. By the end, you will understand the multi-stage Dockerfile that produces an 18MB image, the 5-service Docker Compose stack, the graceful shutdown pattern, and how LiteLLM lets you swap GPT-4o for Claude with a config file change.

---

## The Multi-Stage Dockerfile: Builder → Scratch

The `deployments/Dockerfile` uses two stages:

```dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependencies first (Docker layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Then copy source
COPY . .

# Build a statically-linked binary
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o /app/server ./cmd/api

# Stage 2: Final image
FROM scratch

# Copy TLS certificates (needed for HTTPS calls to LiteLLM, PostgreSQL SSL)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data (needed for correct time.Now() behavior)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the compiled binary
COPY --from=builder /app/server /server

# Copy migrations — the binary and its migrations ship as one unit
COPY --from=builder /app/internal/infrastructure/persistence/postgres/migrations /migrations

EXPOSE 8080
ENTRYPOINT ["/server"]
```

### Why `FROM scratch`?

`FROM scratch` is Docker's empty base image. No shell, no utilities, no package manager, no OS userspace. The final image contains exactly:

1. CA certificates (for TLS)
2. Timezone data
3. The compiled Go binary
4. The SQL migration files

That is it. The result is an 18MB image.

Compare this to a typical Dockerfile using `FROM golang:1.24` as the final stage — that image is 1.2GB because it includes the entire Go toolchain and Alpine Linux. Using `FROM alpine` as the final stage gets you to ~15MB for the OS layer + your binary. `FROM scratch` gets you to just the binary.

### Why `CGO_ENABLED=0`?

Go can link against C libraries (CGO), but that creates a binary that depends on the system's C standard library being present at runtime. `FROM scratch` has no C library.

`CGO_ENABLED=0` compiles a fully static binary — all code, including any normally-CGO parts, is compiled into the binary itself. The resulting binary runs anywhere with no dynamic library dependencies.

The flags on the build command:
- `GOOS=linux GOARCH=amd64` — explicit cross-compilation target (important if you build on macOS/ARM)
- `-w` — strip DWARF debug information (reduces binary size)
- `-s` — strip symbol table (reduces binary size further)
- `-X main.version=${VERSION}` — embed the version string at compile time

### The Migration Trick

```dockerfile
COPY --from=builder /app/internal/infrastructure/persistence/postgres/migrations /migrations
```

The migration SQL files are copied into the final image alongside the binary. This means the binary and its database migrations are always deployed together as a single atomic unit. You cannot accidentally deploy a binary that requires migrations that were not deployed.

In the Docker Compose setup, a separate `migrate` container uses `golang-migrate` to run these files at startup. In a Kubernetes deployment, you would run this as an init container before the API pod starts.

---

## The 5-Service Docker Compose Stack

```yaml
services:
  api:          # The Go agent API
  postgres:     # PostgreSQL 16 (persistent storage)
  redis:        # Redis 7 (rate limiting, caching)
  litellm:      # LLM proxy (GPT-4o, Claude, Gemini)
  migrate:      # golang-migrate (one-shot migration runner)
```

### Startup Order with Health Checks

The most important aspect of the Compose configuration is startup ordering:

```yaml
api:
  depends_on:
    postgres:
      condition: service_healthy
    redis:
      condition: service_healthy

migrate:
  depends_on:
    postgres:
      condition: service_healthy
```

Each `depends_on` with `condition: service_healthy` means Docker waits for the health check to pass before starting the dependent service. Without this, the API might start before PostgreSQL is ready to accept connections — leading to startup failures.

The health checks:

```yaml
postgres:
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U postgres"]
    interval: 5s
    timeout: 5s
    retries: 5

redis:
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 5s
    timeout: 5s
    retries: 5

api:
  healthcheck:
    test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
    interval: 10s
    timeout: 5s
    retries: 3
```

Startup sequence:
1. Postgres and Redis start simultaneously
2. Once both are healthy, `migrate` runs migrations
3. Once Postgres is healthy (migrate could have altered tables), `api` starts
4. `api` becomes healthy once `/health` returns 200

### Memory Limits

```yaml
api:
  deploy:
    resources:
      limits:
        memory: 128M
      reservations:
        memory: 64M
```

The API is limited to 128MB RAM. This is feasible because:
- The Go binary itself is ~18MB on disk, ~30MB in memory with stack/heap
- Connection pools (PostgreSQL, Redis) use minimal memory
- The main variable cost is concurrent request processing — the Eino workflow state per request is small

In practice, the API uses 30-60MB at moderate load. The 128MB limit provides headroom for spikes while preventing a single runaway process from consuming all available memory.

Redis has a memory limit configured in its startup command:

```yaml
redis:
  command: redis-server --appendonly yes --maxmemory 128mb --maxmemory-policy allkeys-lru
```

`allkeys-lru` means when Redis hits 128MB, it evicts the least-recently-used keys. For rate limiting counters that expire after 60 seconds or 24 hours anyway, this is acceptable — a missed eviction just means a slightly less accurate rate limit count, not data corruption.

---

## LiteLLM: Swap LLM Providers With a Config File

LiteLLM is the secret weapon. It is an OpenAI-compatible proxy that translates requests to any LLM provider's API format. The Go code sends OpenAI-format requests to LiteLLM. LiteLLM forwards them to whatever provider you configure.

The config in `deployments/litellm_config.yaml`:

```yaml
model_list:
  - model_name: gpt-4o-mini
    litellm_params:
      model: openai/gpt-4o-mini
      api_key: os.environ/OPENAI_API_KEY

  - model_name: claude-haiku
    litellm_params:
      model: anthropic/claude-3-haiku-20240307
      api_key: os.environ/ANTHROPIC_API_KEY

  - model_name: gpt-4o
    litellm_params:
      model: openai/gpt-4o
      api_key: os.environ/OPENAI_API_KEY
```

To switch the default model from GPT-4o-mini to Claude Haiku: change `LLM_DEFAULT_MODEL=gpt-4o-mini` to `LLM_DEFAULT_MODEL=claude-haiku` in your `.env` file. Restart the API container. Done. Zero code changes.

The Go code sends to `http://litellm:4000/chat/completions` with `model: "claude-haiku"`. LiteLLM looks up `claude-haiku` in its config, translates to Anthropic's API format, forwards the request, and translates the response back to OpenAI format. The Go code never knows the difference.

This also enables:
- **Cost optimization**: use a cheaper model for general tasks, expensive model for complex coding
- **Fallback routing**: if OpenAI is down, route to Anthropic automatically
- **A/B testing**: route a percentage of traffic to different models
- **Local models**: point LiteLLM at an Ollama instance for self-hosted models

---

## Graceful Shutdown: The Complete Pattern

The shutdown logic in `cmd/api/main.go` follows the standard Go pattern:

```go
func main() {
    // ... setup ...

    // Server starts in a goroutine
    go func() {
        slog.Info("starting server", "addr", server.Addr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    // Block until shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit  // Wait here until Ctrl+C or SIGTERM

    slog.Info("shutting down server...")

    // Give in-flight requests up to 30 seconds to complete
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
    defer shutdownCancel()

    if err := server.Shutdown(shutdownCtx); err != nil {
        slog.Error("server shutdown error", "error", err)
    }

    slog.Info("server stopped")
    // defer db.Close() and defer redisClient.Close() run here
}
```

`server.Shutdown(ctx)` does the following:
1. Stops accepting new connections immediately
2. Waits for all in-flight requests to complete
3. If the context times out, forcefully closes remaining connections

This means a user in the middle of a streaming LLM response gets to finish. A 30-second timeout means a user who just started a long generation gets up to 30 seconds of grace. After that, the server forcefully closes.

`signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)` listens for:
- `SIGINT` — Ctrl+C in development
- `SIGTERM` — what Kubernetes and Docker send when stopping a container

The `defer db.Close()` and `defer redisClient.Close()` at the top of `main()` run after `server.Shutdown()` completes, ensuring connections are properly closed in the right order.

---

## Environment Configuration

All configuration is environment-variable driven. The full set:

```bash
# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_SHUTDOWN_TIMEOUT=30s

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=change-in-production
DB_NAME=aiagent
DB_SSL_MODE=disable  # Use "require" in production

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=change-me-in-production-must-be-32-chars
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h  # 7 days

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_PER_MINUTE=60
RATE_LIMIT_PER_DAY=10000

# LLM
LLM_PROVIDER=litellm
LLM_BASE_URL=http://litellm:4000
LLM_API_KEY=sk-1234  # LiteLLM master key
LLM_DEFAULT_MODEL=gpt-4o-mini

# Tools
WEB_SEARCH_API_KEY=your-brave-search-api-key
TOOLS_MAX_CONCURRENT=10
```

Every variable has a default. `config.Load()` reads from environment variables with fallbacks:

```go
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

This means `go run ./cmd/api` works out of the box in development (with sensible defaults), and production is configured purely through environment variables — no config file parsing, no YAML, no TOML.

---

## Production Checklist

Before going to production:

```
Security:
[ ] Set strong JWT_SECRET (32+ random chars: openssl rand -hex 32)
[ ] Set DB_SSL_MODE=require (encrypt DB connections)
[ ] Set Redis password (REDIS_PASSWORD=...)
[ ] Change LiteLLM master key from "sk-1234"
[ ] Set DB_PASSWORD to something strong

Observability:
[ ] Configure log aggregation (the structured JSON logs are ready for Datadog, Loki, etc.)
[ ] Set up /health and /ready monitoring
[ ] Set DB backup schedule

Scale:
[ ] Increase DB_MAX_OPEN_CONNS for higher concurrency (default 25)
[ ] Increase memory limits if needed
[ ] Enable Redis sentinel/cluster for HA

LLM:
[ ] Set real OPENAI_API_KEY and/or ANTHROPIC_API_KEY
[ ] Configure LiteLLM fallback routing
```

The most critical: **`JWT_SECRET`**. Any JWT signed with the default value can be forged by anyone who reads this article. Always override it.

---

## Running the Full Stack

```bash
# 1. Clone and configure
git clone https://github.com/wyuneed/go-agent-api
cd go-agent-api
cp .env.example .env
# Edit .env: add API keys, change JWT_SECRET

# 2. Start all services
make docker-up
# or: docker-compose -f deployments/docker-compose.yml up -d

# 3. Run migrations (if not already run via the migrate service)
make migrate-up

# 4. Verify
curl http://127.0.0.1:8080/health
# {"success":true,"data":{"status":"ok",...}}

# 5. Open Swagger UI
open http://127.0.0.1:8080/swagger/index.html

# 6. Register and test
curl -X POST http://127.0.0.1:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"secret","name":"You"}'

curl -X POST http://127.0.0.1:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"secret"}'
# → returns access_token

curl -X POST http://127.0.0.1:8080/v1/chat/completions \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"What is 15% of 847?"}]}'
```

---

## What We Just Learned

- `FROM scratch` produces an 18MB image — no OS, no shell, just the binary and its assets
- `CGO_ENABLED=0` produces a fully static binary that runs in the scratch container
- Migrations ship inside the image (`COPY --from=builder /migrations`) — binary and schema changes are atomic
- `depends_on: condition: service_healthy` ensures startup order is correct, not just "service started"
- LiteLLM decouples LLM providers from application code — swap GPT-4o for Claude with a config change
- Graceful shutdown uses `signal.Notify(quit, SIGINT, SIGTERM)` + `server.Shutdown(ctx)` with a timeout
- All configuration is environment-variable driven with sensible defaults for local development

---

## Series Complete

You have now read through the entire architecture:

| Article | Topic |
|---------|-------|
| 1 | Architecture Overview: DDD, five pillars, explicit DI |
| 2 | Domain Layer: Entities with business rules, repository interfaces |
| 3 | Tool System: 15-line tools, OpenAI format, per-token ACLs |
| 4 | Eino Workflow: 6-node DAG, ReAct loop, type-safe routing |
| 5 | Dual Auth: JWT + API keys, hashing, fire-and-forget tracking |
| 6 | SSE Streaming: Channel pipeline, Flusher, context cancellation |
| 7 | Human-in-the-Loop: Pause, approve, resume with `*bool` |
| 8 | Deployment: 18MB scratch image, health checks, graceful shutdown |

The codebase is at `github.com/wyuneed/go-agent-api`. It is production-ready and designed to be extended — add tools, add agents, add new LLM providers, replace components. The architecture makes extension easy and regression unlikely.

---

*Previous: [Article 7 — Human-in-the-Loop AI: Pause, Approve, and Resume Agentic Workflows](#)*
