# Go AI Agent API

A **production-ready Go AI Agent API** framework built with Domain-Driven Design (DDD) principles, featuring an agentic workflow engine, human-in-the-loop approvals, and OpenAI-compatible tool integration.

## Features

- ✅ **Agentic AI Workflows** - OpenAI tool calling format with Eino workflow engine (LangGraph equivalent for Go)
- ✅ **Domain-Driven Design** - Clean architecture with clear separation of concerns
- ✅ **Human-in-the-Loop** - Approval system for sensitive actions
- ✅ **JWT Authentication** - DB-managed token expiration and refresh
- ✅ **Multi-Tool Support** - Built-in tools: calculator, web search, and extensible architecture
- ✅ **Production-Ready** - Single binary deployment, Redis caching, rate limiting
- ✅ **High Performance** - Low memory footprint, fast cold start
- ✅ **PostgreSQL + Redis** - Persistent storage with distributed caching

## Tech Stack

- **Language**: Go 1.24+
- **Web Framework**: Chi v5
- **Workflow Engine**: Eino (ByteDance)
- **Database**: PostgreSQL with migrations
- **Cache**: Redis
- **Authentication**: JWT (golang-jwt)
- **Containerization**: Docker & Docker Compose

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go                    # Application entry point
├── internal/
│   ├── application/                   # Application Business Rules
│   │   ├── dto/                       # Data Transfer Objects
│   │   │   ├── request/
│   │   │   └── response/
│   │   ├── port/                      # Interfaces (cache, events, LLM)
│   │   └── usecase/                   # Use cases
│   │       ├── auth/                  # Authentication (login, token refresh)
│   │       ├── chat/                  # Chat conversations and messaging
│   │       ├── tool/                  # Tool execution and registry
│   │       └── user/                  # User management
│   ├── domain/                        # Enterprise Business Rules
│   │   ├── entity/                    # Domain entities
│   │   ├── valueobject/               # Value objects
│   │   ├── repository/                # Repository interfaces
│   │   ├── service/                   # Domain services
│   │   └── event/                     # Domain events
│   ├── infrastructure/                # External tools & frameworks
│   │   ├── config/                    # Configuration management
│   │   ├── eino/                      # Workflow engine setup
│   │   │   ├── graphs/                # Workflow definitions
│   │   │   └── state/                 # Workflow state
│   │   ├── http/                      # HTTP layer
│   │   │   ├── handler/               # HTTP handlers
│   │   │   └── middleware/            # Middleware (auth, logging, rate limit)
│   │   ├── llm/                       # LLM provider integration
│   │   ├── persistence/               # Data persistence
│   │   │   ├── postgres/              # PostgreSQL implementations
│   │   │   └── redis/                 # Redis implementations
│   │   └── worker/                    # Background workers
│   └── pkg/                           # Shared packages
│       ├── errors/                    # Error handling
│       ├── jwt/                       # JWT utilities
│       └── response/                  # Response formatting
├── configs/
│   └── config.example.yaml            # Configuration template
├── deployments/
│   ├── Dockerfile                     # Production Docker image
│   ├── docker-compose.yml             # Local development stack
│   └── litellm_config.yaml            # LiteLLM configuration
├── scripts/
│   └── migrate.sh                     # Database migration runner
├── Makefile                           # Build and development tasks
├── go.mod & go.sum                    # Go dependencies
└── README.md                          # This file
```

## Getting Started

### Prerequisites

- Go 1.24+
- PostgreSQL 14+
- Redis 7+
- Docker & Docker Compose (optional, for containerized setup)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/wyuneed/go-agent-api.git
   cd go-agent-api
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   ```bash
   cp configs/config.example.yaml configs/config.yaml
   ```
   Edit `configs/config.yaml` with your database, Redis, and LLM provider credentials.

4. **Run database migrations**
   ```bash
   make migrate
   ```
   Or manually:
   ```bash
   DATABASE_URL="postgres://user:pass@localhost:5432/aiagent" scripts/migrate.sh
   ```

### Development

**Using Makefile:**
```bash
# Build the application
make build

# Run the application
make run

# Run with hot reload
make run-watch

# Run tests
make test

# Run tests with coverage
make test-coverage

# Lint code
make lint
```

**Using Docker Compose (recommended for full stack):**
```bash
# Start all services (API, PostgreSQL, Redis, LiteLLM)
docker-compose -f deployments/docker-compose.yml up -d

# Stop services
docker-compose -f deployments/docker-compose.yml down
```

## API Endpoints

### Authentication

- `POST /api/v1/auth/login` - User login (returns access & refresh tokens)
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/validate` - Validate JWT token

### Chat

- `POST /api/v1/chat/send` - Send a message to the agent (supports streaming)
- `GET /api/v1/chat/conversation/{id}` - Get conversation details
- `GET /api/v1/chat/conversations` - List user's conversations
- `POST /api/v1/chat/approve-action` - Human-in-the-loop approval for tool execution

### Tools

- `POST /api/v1/tools/execute` - Execute a specific tool
- `GET /api/v1/tools/registry` - List available tools

### Health

- `GET /health` - Health check endpoint

## Configuration

Edit `configs/config.yaml` to configure:

```yaml
server:
  port: 8080
  timeout: 30s

database:
  url: "postgres://user:pass@localhost:5432/aiagent?sslmode=disable"
  max_connections: 25
  max_idle_connections: 5

redis:
  url: "redis://localhost:6379"
  db: 0

jwt:
  secret_key: "your-secret-key"
  access_token_ttl: 15m
  refresh_token_ttl: 7d

llm:
  provider: "litellm"
  api_key: "your-api-key"
  model: "gpt-4"
```

## Core Concepts

### Domain-Driven Design (DDD)

The project follows DDD principles:

- **Entity** (`domain/entity/`) - Objects with unique identity (User, Conversation, Message)
- **Value Object** (`domain/valueobject/`) - Immutable objects without identity (Email, MessageRole)
- **Repository** (`domain/repository/`) - Interfaces for data access
- **Domain Service** (`domain/service/`) - Business logic spanning multiple entities
- **Domain Event** (`domain/event/`) - Events triggered by domain state changes

### Workflow Engine (Eino)

The Eino workflow engine orchestrates multi-step agent interactions:

- **Nodes** - Individual processing units (LLM call, tool execution, approval)
- **Graphs** - DAG combining nodes for complex workflows
- **State** - Persistent state across workflow steps

Example workflow: User Message → LLM Analysis → Tool Selection → Tool Execution → Approval (if needed) → Response

### Tool System

Tools follow OpenAI's tool calling format for LLM compatibility:

```go
type Tool struct {
    Name        string
    Description string
    InputSchema json.RawMessage // JSON Schema
    Handler     func(args map[string]interface{}) (interface{}, error)
}
```

**Built-in Tools:**
- `calculator` - Basic math operations
- `web_search` - Search the web

**Extensible:** Add custom tools in `internal/application/usecase/tool/builtin/`

### Authentication

JWT-based authentication with DB-managed token expiration:

1. User logs in → Server generates `access_token` (short-lived) + `refresh_token` (long-lived)
2. Client includes access token in `Authorization: Bearer <token>` header
3. Tokens stored in PostgreSQL with expiration timestamps
4. Refreshing creates new access token from valid refresh token

## Database Migrations

Migrations use the standard up/down pattern:

```
migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_tokens_table.up.sql
├── ...
```

Run migrations:
```bash
make migrate
```

## Deployment

### Docker

Build and run with Docker:

```bash
# Build image
docker build -t ai-agent-api:latest -f deployments/Dockerfile .

# Run container
docker run -e DATABASE_URL="..." -e REDIS_URL="..." -p 8080:8080 ai-agent-api:latest
```

### Production Checklist

- [ ] Set strong JWT secret key
- [ ] Configure CORS origins appropriately
- [ ] Enable rate limiting
- [ ] Set up database backups
- [ ] Monitor Redis memory usage
- [ ] Use environment variables for secrets (never commit `.env`)
- [ ] Enable request logging
- [ ] Set up health checks
- [ ] Configure graceful shutdown

## Testing

Run all tests:
```bash
make test
```

Run with coverage report:
```bash
make test-coverage
```

Run integration tests:
```bash
make test-integration
```

## Performance Optimization

- **Connection Pooling** - PostgreSQL and Redis connections are pooled
- **Redis Caching** - Conversation and token data cached in Redis
- **Rate Limiting** - Built-in rate limiting middleware
- **Request IDs** - Correlation IDs for request tracing
- **Streaming** - SSE support for real-time chat responses
- **Worker Pool** - Background job processing

## Middleware Stack

- **Authentication** - JWT validation on protected routes
- **CORS** - Cross-Origin Resource Sharing configuration
- **Logging** - Structured request/response logging
- **Rate Limiting** - Per-user/IP rate limiting
- **Recovery** - Panic recovery and error responses
- **Request ID** - Unique ID injection for tracing

## Troubleshooting

### Database Connection Issues
```bash
# Check PostgreSQL is running
psql -U postgres -h localhost -p 5432

# Verify DATABASE_URL
echo $DATABASE_URL
```

### Redis Connection Issues
```bash
# Check Redis is running
redis-cli ping
```

### Port Already in Use
```bash
# Change port in config.yaml or use environment variable
SERVER_PORT=8081 make run
```

### JWT Token Errors
- Ensure JWT secret is consistent across restarts
- Check token hasn't expired
- Verify Authorization header format: `Bearer <token>`

## Contributing

1. Follow DDD principles - keep business logic in the domain layer
2. Write tests for new features
3. Ensure `make lint` passes
4. Update relevant documentation
5. Use meaningful commit messages

## License

See [LICENSE](LICENSE) file for details.

## Support

For issues, questions, or contributions, please open an issue on GitHub.

---

**Built with Go, Domain-Driven Design, and ❤️**
