.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down \
        docker-restart docker-stop monitoring-up monitoring-down k8s-deploy k8s-delete

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
