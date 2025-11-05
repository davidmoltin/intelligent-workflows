.PHONY: help build run test test-unit test-integration test-e2e test-all test-load clean docker-up docker-down migrate-up migrate-down mocks

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

test: test-unit ## Run unit tests

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.out ./internal/... ./pkg/...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@echo "Ensure PostgreSQL and Redis are running (make docker-up)"
	@INTEGRATION_TESTS=1 go test -v ./tests/integration/...

test-e2e: ## Run E2E tests
	@echo "Running E2E tests..."
	@echo "Ensure PostgreSQL and Redis are running (make docker-up)"
	@E2E_TESTS=1 go test -v ./tests/e2e/...

test-all: ## Run all tests
	@echo "Running all tests..."
	@bash ./scripts/test-all.sh

test-coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	@bash ./scripts/test-coverage.sh

test-load: ## Run load tests
	@echo "Running load tests with k6..."
	@command -v k6 >/dev/null 2>&1 || { echo "k6 is not installed. Install from https://k6.io/docs/getting-started/installation/"; exit 1; }
	@k6 run tests/load/k6-workflow-load.js

mocks: ## Generate mocks
	@echo "Generating mocks..."
	@command -v mockery >/dev/null 2>&1 || { echo "mockery is not installed. Install with: go install github.com/vektra/mockery/v2@latest"; exit 1; }
	@mockery

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
