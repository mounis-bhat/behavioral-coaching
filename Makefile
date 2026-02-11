.PHONY: dev dev-logs dev-go dev-web db db-stop db-drop valkey-flush swag types build run clean install help \
       migrate-up migrate-down migrate-status migrate-create sqlc

# Default target
help:
	@echo "Available targets:"
	@echo "  make install        - Install Go and bun dependencies"
	@echo "  make dev            - Start everything (Genkit UI + Go + SvelteKit)"
	@echo "  make dev-go         - Start Go server with hot reload (air)"
	@echo "  make dev-web        - Start SvelteKit dev server only"
	@echo "  make db             - Start Postgres, Valkey, and MinIO"
	@echo "  make db-stop        - Stop Postgres, Valkey, and MinIO"
	@echo "  make db-drop        - Drop the Postgres database"
	@echo "  make valkey-flush   - Flush all Valkey keys"
	@echo "  make migrate-up     - Run all pending migrations"
	@echo "  make migrate-down   - Rollback last migration"
	@echo "  make migrate-status - Show migration status"
	@echo "  make migrate-create - Create new migration (NAME=migration_name)"
	@echo "  make sqlc           - Generate Go code from SQL queries"
	@echo "  make swag           - Generate OpenAPI spec"
	@echo "  make types          - Generate TypeScript types from OpenAPI"
	@echo "  make build          - Full production build (dist)"
	@echo "  make run            - Run production binary (dist)"
	@echo "  make clean          - Clean all artifacts (build)"

# Install dependencies
install:
	curl -sL cli.genkit.dev | bash
	go mod download
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/air-verse/air@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	cd web && bun install

# Load env vars for docker compose
LOAD_ENV = set -a && . ./.env.development && set +a

# Development - runs everything together (Genkit UI + Go + SvelteKit)
dev:
	@echo "Starting Genkit UI + Go server + SvelteKit..."
	@echo "Go API:     http://localhost:3400"
	@echo "SvelteKit:  http://localhost:5173"
	@echo "Genkit UI:  http://localhost:4000"
	@echo "Press Ctrl+C to stop all"
	@$(MAKE) types
	@$(LOAD_ENV) && docker compose up -d postgres valkey minio minio-init
	@bash -c 'trap "$(LOAD_ENV) && docker compose down; kill $$(jobs -p) 2>/dev/null" INT TERM EXIT; genkit start -- air & cd web && bun dev & wait' || [ $$? -eq 130 ]

# Development logs (docker compose)
dev-logs:
	@$(LOAD_ENV) && docker compose logs

# Individual dev servers
dev-go:
	genkit start -- air

dev-web:
	cd web && bun dev

# Database
db:
	@$(LOAD_ENV) && docker compose up -d postgres valkey minio minio-init

db-stop:
	@$(LOAD_ENV) && docker compose down

db-drop:
	@$(LOAD_ENV) && docker compose exec -T postgres sh -c 'PGPASSWORD="$${POSTGRES_PASSWORD}" psql -U "$${POSTGRES_USER}" -d postgres -c "DROP DATABASE IF EXISTS $${POSTGRES_DB};"'

valkey-flush:
	@$(LOAD_ENV) && docker compose exec -T valkey sh -c 'valkey-cli -a "$${VALKEY_PASSWORD}" FLUSHALL'

# Migrations (uses env vars from shell or .env.development)
# URL-encodes the password to handle special characters
define GOOSE_CMD
goose -dir migrations postgres "postgres://$${POSTGRES_USER:-app}:$$(printf '%s' "$${POSTGRES_PASSWORD}" | python3 -c 'import sys,urllib.parse;print(urllib.parse.quote(sys.stdin.read(),safe=""))')@$${POSTGRES_HOST:-localhost}:$${POSTGRES_PORT:-5432}/$${POSTGRES_DB:-app}?sslmode=$${POSTGRES_SSLMODE:-disable}"
endef

migrate-up:
	@set -a && . ./.env.development && set +a && $(GOOSE_CMD) up

migrate-down:
	@set -a && . ./.env.development && set +a && $(GOOSE_CMD) down

migrate-status:
	@set -a && . ./.env.development && set +a && $(GOOSE_CMD) status

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	goose -dir migrations create $(NAME) sql

# Code generation
sqlc:
	sqlc generate

# Code generation
swag:
	swag init -g main.go -o docs -d ./cmd/server,./internal/api,./internal/app,./internal/config,./internal/domain,./internal/storage

types: swag
	cd web && bun run generate:api-types

# Build
build: swag
	cd web && bun run build
	rm -rf assets/static/*
	cp -r web/build/* assets/static/
	rm -rf dist
	mkdir -p dist
	cp .env.production dist/.env.production
	go build -o dist/server ./cmd/server

# Run production
run:
	cd dist && ENV=production ./server

# Clean
clean:
	rm -rf server tmp dist
	rm -rf assets/static/* && touch assets/static/.gitkeep
	rm -rf web/build
