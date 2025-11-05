# Package A: Backend Foundation - Setup Guide

This document provides instructions for setting up and running Package A of the Intelligent Workflows Service.

## ✅ Package A Deliverables (Completed)

- [x] Go project structure initialized
- [x] PostgreSQL schema and migrations
- [x] Redis connection setup
- [x] Basic REST API framework (Chi router)
- [x] Data models (Workflow, Execution, Rule, Event)
- [x] Basic CRUD endpoints for workflows
- [x] Repository layer
- [x] Health and readiness endpoints
- [x] Configuration management
- [x] Logging infrastructure
- [x] Docker Compose for local development
- [x] Makefile for common tasks

## 📋 Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Make (optional, for using Makefile commands)

## 🚀 Quick Start

### 1. Install Dependencies

```bash
go mod download
```

### 2. Start Infrastructure (PostgreSQL + Redis)

```bash
docker-compose up -d postgres redis
```

Wait for the services to be healthy:

```bash
docker-compose ps
```

### 3. Run Database Migrations

The migrations will run automatically when PostgreSQL starts. To manually run them:

```bash
make migrate-up
```

### 4. Run the API Server

```bash
# Using Go directly
go run ./cmd/api

# Or using Make
make run
```

The API server will start on `http://localhost:8080`

### 5. Verify the Service is Running

Check health endpoint:

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

Check readiness endpoint:

```bash
curl http://localhost:8080/ready
```

Expected response:
```json
{
  "status": "ready",
  "version": "0.1.0",
  "checks": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

## 📡 API Endpoints

### Health Endpoints

- `GET /health` - Simple health check
- `GET /ready` - Readiness check (includes database and Redis health)

### Workflow Endpoints

- `POST /api/v1/workflows` - Create a new workflow
- `GET /api/v1/workflows` - List workflows
- `GET /api/v1/workflows/{id}` - Get a specific workflow
- `PUT /api/v1/workflows/{id}` - Update a workflow
- `DELETE /api/v1/workflows/{id}` - Delete a workflow
- `POST /api/v1/workflows/{id}/enable` - Enable a workflow
- `POST /api/v1/workflows/{id}/disable` - Disable a workflow

## 🧪 Testing the API

### Create a Workflow

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "high_value_order_approval",
    "version": "1.0.0",
    "name": "High Value Order Approval",
    "description": "Require approval for orders over $10,000",
    "definition": {
      "trigger": {
        "type": "event",
        "event": "order.checkout.initiated"
      },
      "context": {
        "load": ["order.details", "customer.history"]
      },
      "steps": [
        {
          "id": "check_order_value",
          "type": "condition",
          "condition": {
            "field": "order.total",
            "operator": "gte",
            "value": 10000
          },
          "on_true": "require_approval",
          "on_false": "allow_order"
        },
        {
          "id": "require_approval",
          "type": "action",
          "action": {
            "action": "block",
            "reason": "Order requires approval for amounts over $10,000"
          }
        },
        {
          "id": "allow_order",
          "type": "action",
          "action": {
            "action": "allow"
          }
        }
      ]
    },
    "tags": ["approval", "orders", "high-value"]
  }'
```

### List Workflows

```bash
# List all workflows
curl http://localhost:8080/api/v1/workflows

# List enabled workflows only
curl http://localhost:8080/api/v1/workflows?enabled=true

# Paginate results
curl http://localhost:8080/api/v1/workflows?limit=10&offset=0
```

### Get a Workflow

```bash
curl http://localhost:8080/api/v1/workflows/{workflow-id}
```

### Update a Workflow

```bash
curl -X PUT http://localhost:8080/api/v1/workflows/{workflow-id} \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Workflow Name",
    "description": "Updated description"
  }'
```

### Delete a Workflow

```bash
curl -X DELETE http://localhost:8080/api/v1/workflows/{workflow-id}
```

### Enable/Disable a Workflow

```bash
# Enable
curl -X POST http://localhost:8080/api/v1/workflows/{workflow-id}/enable

# Disable
curl -X POST http://localhost:8080/api/v1/workflows/{workflow-id}/disable
```

## 🐳 Docker Development

### Start All Services (including API)

```bash
make docker-up
```

### View Logs

```bash
make docker-logs
```

### Stop All Services

```bash
make docker-down
```

### Rebuild Docker Image

```bash
make docker-build
```

## 🗄️ Database Management

### Access PostgreSQL Shell

```bash
make db-shell
```

### Access Redis CLI

```bash
make redis-cli
```

### Run Migrations

```bash
# Up
make migrate-up

# Down
make migrate-down
```

## 🛠️ Development Commands

```bash
# Build the API server
make build

# Run the API server
make run

# Run tests
make test

# View test coverage
make test-coverage

# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Clean build artifacts
make clean

# Download dependencies
make deps
```

## 📁 Project Structure

```
.
├── cmd/
│   ├── api/          # API server entry point
│   ├── cli/          # CLI tool (Package D)
│   └── worker/       # Background worker
├── internal/
│   ├── api/
│   │   └── rest/
│   │       ├── handlers/    # HTTP handlers
│   │       ├── middleware/  # HTTP middleware
│   │       └── router.go    # Route configuration
│   ├── engine/              # Workflow execution engine (Package B)
│   ├── models/              # Data models
│   ├── repository/
│   │   └── postgres/        # PostgreSQL repositories
│   └── services/            # Business logic services
├── pkg/
│   ├── config/              # Configuration management
│   ├── database/            # Database connections
│   └── logger/              # Logging utilities
├── migrations/
│   └── postgres/            # Database migrations
├── docker-compose.yml       # Local development environment
├── Dockerfile              # API server container
├── Makefile               # Development commands
└── go.mod                 # Go dependencies
```

## 🔧 Configuration

Configuration is managed via environment variables. See `.env.example` for all available options.

Key configuration:

```bash
# Server
SERVER_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=workflows
DB_USER=postgres
DB_PASSWORD=postgres

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

## 🚦 Next Steps

Package A provides the foundation for the Intelligent Workflows Service. Next packages to implement:

- **Package B**: Core Workflow Engine (Weeks 3-6)
- **Package C**: Frontend Development (Weeks 3-11)
- **Package D**: CLI Tool (Weeks 3-5)
- **Package E**: AI Integration (Weeks 5-8)
- **Package F**: Infrastructure & DevOps (Ongoing)
- **Package G**: Testing Infrastructure (Ongoing)

## 📚 Documentation

- [Architecture](./ARCHITECTURE.md)
- [Implementation Roadmap](./IMPLEMENTATION_ROADMAP.md)
- [Parallel Work Packages](./PARALLEL_WORK_PACKAGES.md)
- [Getting Started](./GETTING_STARTED.md)

## 🐛 Troubleshooting

### Port Already in Use

If port 8080 is already in use, change the `SERVER_PORT` environment variable:

```bash
SERVER_PORT=8081 go run ./cmd/api
```

### Database Connection Failed

Ensure PostgreSQL is running:

```bash
docker-compose ps postgres
```

Check logs:

```bash
docker-compose logs postgres
```

### Redis Connection Failed

Ensure Redis is running:

```bash
docker-compose ps redis
```

Check logs:

```bash
docker-compose logs redis
```

## ✅ Package A Status

**Status**: 🟢 Complete

All deliverables for Package A have been implemented and tested. The backend foundation is ready for Package B (Core Workflow Engine) development.
