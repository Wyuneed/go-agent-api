#!/bin/bash
set -e

DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/aiagent?sslmode=disable}"
MIGRATIONS_PATH="internal/infrastructure/persistence/postgres/migrations"

case "$1" in
  up)
    echo "Running migrations up..."
    migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" up
    ;;
  down)
    echo "Running migrations down..."
    migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" down "${2:-1}"
    ;;
  create)
    if [ -z "$2" ]; then
      echo "Usage: $0 create <migration_name>"
      exit 1
    fi
    echo "Creating migration: $2"
    migrate create -ext sql -dir "$MIGRATIONS_PATH" -seq "$2"
    ;;
  version)
    migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" version
    ;;
  force)
    if [ -z "$2" ]; then
      echo "Usage: $0 force <version>"
      exit 1
    fi
    migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" force "$2"
    ;;
  *)
    echo "Usage: $0 {up|down|create|version|force}"
    exit 1
    ;;
esac
