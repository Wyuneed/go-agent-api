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
	@echo "  build            - Build the binary"
	@echo "  build-linux      - Build for Linux"
	@echo "  run              - Run the application"
	@echo "  run-watch        - Run with hot reload"
	@echo "  test             - Run unit tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  lint             - Run linter"
	@echo "  migrate-up       - Run database migrations"
	@echo "  migrate-down     - Rollback last migration"
	@echo "  migrate-create   - Create new migration"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-up        - Start Docker Compose"
	@echo "  docker-down      - Stop Docker Compose"
	@echo "  docker-logs      - View Docker logs"
	@echo "  dev-deps         - Install development dependencies"
	@echo "  clean            - Clean build artifacts"
