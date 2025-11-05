.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the API server
	@echo "Building API server..."
	@go build -o bin/api ./cmd/api

run: ## Run the API server locally
	@echo "Running API server..."
	@go run ./cmd/api

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	@go tool cover -html=coverage.out

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker-compose build

docker-logs: ## View Docker logs
	@docker-compose logs -f api

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@docker-compose exec -T postgres psql -U postgres -d workflows -f /docker-entrypoint-initdb.d/001_initial_schema.up.sql

migrate-down: ## Run database migrations down
	@echo "Rolling back migrations..."
	@docker-compose exec -T postgres psql -U postgres -d workflows -f /docker-entrypoint-initdb.d/001_initial_schema.down.sql

db-shell: ## Open PostgreSQL shell
	@docker-compose exec postgres psql -U postgres -d workflows

redis-cli: ## Open Redis CLI
	@docker-compose exec redis redis-cli

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

.DEFAULT_GOAL := help
