.PHONY: help build run test test-unit test-integration test-e2e test-all test-load clean \
        docker-up docker-down docker-build docker-logs docker-restart docker-stop docker-rebuild docker-clean \
        migrate-up migrate-down mocks \
        monitoring-up monitoring-down monitoring-logs \
        k8s-validate k8s-deploy-dev k8s-deploy-staging k8s-deploy-prod k8s-delete k8s-status k8s-logs \
        health ready db-shell db-console db-backup db-restore redis-cli \
        deps fmt lint dev dev-full ci security-scan tag metrics dashboards

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

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

# Docker commands
docker-restart: ## Restart all Docker containers
	@echo "Restarting Docker containers..."
	@docker-compose restart

docker-stop: docker-down ## Stop Docker containers (alias for docker-down)

docker-rebuild: ## Rebuild and restart Docker containers
	@echo "Rebuilding Docker containers..."
	@docker-compose down
	@docker-compose build --no-cache
	@docker-compose up -d

docker-clean: ## Clean Docker resources (containers, volumes, images)
	@echo "Cleaning Docker resources..."
	@docker-compose down -v
	@docker system prune -f

# Monitoring commands
monitoring-up: ## Start monitoring stack (Prometheus + Grafana)
	@echo "Starting monitoring stack..."
	@docker-compose up -d prometheus grafana postgres-exporter redis-exporter

monitoring-down: ## Stop monitoring stack
	@echo "Stopping monitoring stack..."
	@docker-compose stop prometheus grafana postgres-exporter redis-exporter

monitoring-logs: ## View monitoring logs
	@docker-compose logs -f prometheus grafana

# Kubernetes commands
k8s-validate: ## Validate Kubernetes manifests
	@echo "Validating Kubernetes manifests..."
	@kubectl kustomize k8s/base > /dev/null && echo "✓ Base manifests valid"
	@kubectl kustomize k8s/overlays/dev > /dev/null && echo "✓ Dev manifests valid"
	@kubectl kustomize k8s/overlays/staging > /dev/null && echo "✓ Staging manifests valid"
	@kubectl kustomize k8s/overlays/production > /dev/null && echo "✓ Production manifests valid"

k8s-deploy-dev: ## Deploy to Kubernetes (dev environment)
	@echo "Deploying to dev..."
	@kubectl apply -k k8s/overlays/dev

k8s-deploy-staging: ## Deploy to Kubernetes (staging environment)
	@echo "Deploying to staging..."
	@kubectl apply -k k8s/overlays/staging

k8s-deploy-prod: ## Deploy to Kubernetes (production environment)
	@echo "Deploying to production..."
	@kubectl apply -k k8s/overlays/production

k8s-delete: ## Delete from Kubernetes
	@echo "Deleting from Kubernetes..."
	@kubectl delete -k k8s/base

k8s-status: ## Check Kubernetes deployment status
	@echo "Checking deployment status..."
	@kubectl get pods,svc,ingress -n intelligent-workflows

k8s-logs: ## View API logs in Kubernetes
	@kubectl logs -f -n intelligent-workflows -l app=workflows-api

# Health check commands
health: ## Check API health
	@echo "Checking API health..."
	@curl -s http://localhost:8080/health | jq .

ready: ## Check API readiness
	@echo "Checking API readiness..."
	@curl -s http://localhost:8080/ready | jq .

# Database commands
db-console: db-shell ## Open database console (alias for db-shell)

db-backup: ## Backup database
	@echo "Backing up database..."
	@docker-compose exec -T postgres pg_dump -U postgres workflows > backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "Backup completed!"

db-restore: ## Restore database from backup (specify file with BACKUP_FILE=filename)
	@echo "Restoring database from $(BACKUP_FILE)..."
	@docker-compose exec -T postgres psql -U postgres -d workflows < $(BACKUP_FILE)

# Development commands
dev: ## Start development environment with hot reload
	@echo "Starting development environment..."
	@docker-compose up -d postgres redis
	@sleep 3
	@go run ./cmd/api

dev-full: docker-up monitoring-up ## Start full development environment with monitoring
	@echo "Full development environment started!"
	@echo "API: http://localhost:8080"
	@echo "Prometheus: http://localhost:9091"
	@echo "Grafana: http://localhost:3000 (admin/admin)"

# CI/CD commands
ci: deps lint test build ## Run CI pipeline locally

security-scan: ## Run security scan
	@echo "Running security scan..."
	@gosec ./...

# Release commands
tag: ## Create a new git tag (specify version with VERSION=v1.0.0)
	@echo "Creating tag $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)

# Metrics commands
metrics: ## View Prometheus metrics
	@echo "Opening Prometheus..."
	@open http://localhost:9091 || xdg-open http://localhost:9091 || echo "Open http://localhost:9091"

dashboards: ## View Grafana dashboards
	@echo "Opening Grafana..."
	@open http://localhost:3000 || xdg-open http://localhost:3000 || echo "Open http://localhost:3000"

.DEFAULT_GOAL := help
