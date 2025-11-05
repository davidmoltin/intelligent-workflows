# Getting Started with Intelligent Workflows

This guide will help you set up your development environment and start building the Intelligent Workflows service.

## Prerequisites

### Required Software

- **Go** 1.21 or higher ([install](https://golang.org/doc/install))
- **PostgreSQL** 15 or higher ([install](https://www.postgresql.org/download/))
- **Redis** 7 or higher ([install](https://redis.io/download))
- **Node.js** 18 or higher (for UI development) ([install](https://nodejs.org/))
- **Git** ([install](https://git-scm.com/downloads))

### Development Tools

- **Docker & Docker Compose** (recommended for local development)
- **Make** (optional, for convenience commands)
- **sqlc** ([install](https://docs.sqlc.dev/en/stable/overview/install.html))
- **golang-migrate** ([install](https://github.com/golang-migrate/migrate))
- **VS Code** or **GoLand** (recommended IDEs)

## Quick Start with Docker

The fastest way to get started is using Docker Compose:

```bash
# Clone the repository
git clone https://github.com/yourorg/intelligent-workflows.git
cd intelligent-workflows

# Start all services
docker-compose up -d

# Run migrations
make migrate-up

# Verify services are running
curl http://localhost:8080/health
```

## Manual Setup

### 1. Clone the Repository

```bash
git clone https://github.com/yourorg/intelligent-workflows.git
cd intelligent-workflows
```

### 2. Set Up PostgreSQL

```bash
# Create database
createdb workflows

# Or using psql
psql -U postgres
CREATE DATABASE workflows;
CREATE USER workflows_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE workflows TO workflows_user;
\q
```

### 3. Set Up Redis

```bash
# Start Redis server
redis-server

# Or with Docker
docker run -d -p 6379:6379 redis:7
```

### 4. Configure Environment

Create a `.env` file in the project root:

```bash
# Server
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
ENV=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_DATABASE=workflows
DB_USER=workflows_user
DB_PASSWORD=your_password
DB_MAX_CONNECTIONS=50
DB_MIN_CONNECTIONS=10
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
REDIS_PASSWORD=

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json

# JWT
JWT_SECRET=your-secret-key-change-this-in-production
JWT_EXPIRY=24h

# External Services (optional)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=
SMTP_PASSWORD=
SLACK_WEBHOOK_URL=
```

### 5. Install Go Dependencies

```bash
go mod download
go mod tidy
```

### 6. Run Database Migrations

```bash
# Install golang-migrate if not already installed
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -database "postgresql://workflows_user:your_password@localhost:5432/workflows?sslmode=disable" \
        -path migrations/postgres up
```

### 7. Generate Type-Safe SQL Code

```bash
# Install sqlc if not already installed
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Generate code
sqlc generate
```

### 8. Build and Run

```bash
# Build the API server
go build -o bin/api cmd/api/main.go

# Run the API server
./bin/api

# Or run directly
go run cmd/api/main.go
```

The API server will start on `http://localhost:8080`

### 9. Verify Installation

```bash
# Health check
curl http://localhost:8080/health

# Response:
# {"status": "ok", "timestamp": "2025-11-05T10:00:00Z"}

# Ready check
curl http://localhost:8080/ready

# Response:
# {"status": "ready", "database": "ok", "redis": "ok"}
```

## Development Workflow

### Running in Development Mode

```bash
# Install air for hot reloading
go install github.com/cosmtrek/air@latest

# Run with hot reload
air

# Or use the Makefile
make dev
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Database Operations

```bash
# Create a new migration
migrate create -ext sql -dir migrations/postgres -seq add_users_table

# Migrate up
make migrate-up

# Migrate down
make migrate-down

# Reset database (danger!)
make db-reset
```

### Code Generation

```bash
# Generate SQL code with sqlc
sqlc generate

# Or use the Makefile
make generate
```

## Project Structure

```
intelligent-workflows/
├── cmd/
│   ├── api/          # API server entry point
│   ├── worker/       # Background worker
│   └── cli/          # CLI tool
├── internal/
│   ├── api/          # API handlers and routes
│   ├── engine/       # Workflow execution engine
│   ├── models/       # Data models
│   ├── repository/   # Database repositories
│   └── services/     # Business logic services
├── pkg/
│   ├── logger/       # Logging utilities
│   ├── config/       # Configuration management
│   └── database/     # Database utilities
├── migrations/       # Database migrations
├── web/              # React frontend
├── examples/         # Example workflows
├── docs/             # Documentation
└── tests/            # Integration tests
```

## Creating Your First Workflow

### 1. Create a Workflow Definition

Create `my-first-workflow.json`:

```json
{
  "workflow_id": "my_first_workflow",
  "version": "1.0.0",
  "name": "My First Workflow",
  "description": "A simple workflow that logs events",
  "enabled": true,
  "trigger": {
    "type": "event",
    "event": "test.event"
  },
  "steps": [
    {
      "id": "log_event",
      "type": "action",
      "action": "allow",
      "metadata": {
        "message": "Event received successfully"
      }
    }
  ]
}
```

### 2. Deploy the Workflow

```bash
# Using curl
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @my-first-workflow.json

# Or using the CLI (once implemented)
workflow deploy my-first-workflow.json
```

### 3. Trigger the Workflow

```bash
# Send a test event
curl -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "test.event",
    "source": "manual",
    "payload": {
      "message": "Hello, World!"
    }
  }'
```

### 4. View Execution Results

```bash
# List executions
curl http://localhost:8080/api/v1/executions

# Get specific execution
curl http://localhost:8080/api/v1/executions/{execution_id}
```

## Frontend Development

### Set Up React App

```bash
cd web

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

The React app will run on `http://localhost:5173` (Vite default) and proxy API requests to `http://localhost:8080`.

### Available Scripts

- `npm run dev` - Start development server
- `npm test` - Run tests
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Lint code

## Docker Development

### Using Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: workflows
      POSTGRES_USER: workflows_user
      POSTGRES_PASSWORD: workflows_pass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    ports:
      - "6379:6379"

  web:
    build:
      context: ./web
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    depends_on:
      - api

volumes:
  postgres_data:
```

### Docker Commands

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Rebuild and restart
docker-compose up -d --build

# Run migrations
docker-compose exec api migrate -database "$DATABASE_URL" -path migrations/postgres up
```

## Debugging

### VS Code Launch Configuration

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch API Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/api",
      "env": {
        "ENV": "development",
        "DB_HOST": "localhost",
        "DB_PORT": "5432",
        "DB_DATABASE": "workflows",
        "REDIS_HOST": "localhost",
        "REDIS_PORT": "6379"
      }
    },
    {
      "name": "Debug Test",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}"
    }
  ]
}
```

### Logging

```go
// Enable debug logging
logger.Debug("workflow execution started",
    zap.String("workflow_id", workflowID),
    zap.String("execution_id", executionID))

// Log errors
logger.Error("workflow execution failed",
    zap.Error(err),
    zap.String("workflow_id", workflowID))
```

### Database Debugging

```bash
# Connect to database
psql -U workflows_user -d workflows

# View workflows
SELECT workflow_id, name, enabled FROM workflows;

# View recent executions
SELECT execution_id, workflow_id, status, started_at
FROM workflow_executions
ORDER BY started_at DESC
LIMIT 10;

# Check step executions
SELECT step_id, status, duration_ms
FROM step_executions
WHERE execution_id = 'exec_xxx';
```

## Common Tasks

### Adding a New API Endpoint

1. Define the handler in `internal/api/rest/handlers/`
2. Add the route in `internal/api/rest/router.go`
3. Add tests in `internal/api/rest/handlers/*_test.go`
4. Update API documentation

### Adding a New Database Table

1. Create migration: `migrate create -ext sql -dir migrations/postgres -seq add_table_name`
2. Write migration SQL
3. Add queries in `internal/repository/postgres/queries/`
4. Generate code: `sqlc generate`
5. Create repository methods
6. Add service layer methods

### Adding a New Workflow Step Type

1. Update `internal/models/step.go`
2. Add executor logic in `internal/engine/executor.go`
3. Add validation in `internal/engine/validator.go`
4. Update documentation
5. Add tests

## Troubleshooting

### Database Connection Issues

```bash
# Check PostgreSQL is running
pg_isready -h localhost -p 5432

# Check connection
psql -U workflows_user -d workflows -h localhost

# View PostgreSQL logs
tail -f /usr/local/var/log/postgresql.log
```

### Redis Connection Issues

```bash
# Check Redis is running
redis-cli ping

# Should return: PONG

# Connect to Redis
redis-cli

# Test commands
> SET test "hello"
> GET test
```

### Port Already in Use

```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>
```

### Migration Errors

```bash
# Check current migration version
migrate -database "$DATABASE_URL" -path migrations/postgres version

# Force to specific version (danger!)
migrate -database "$DATABASE_URL" -path migrations/postgres force <version>
```

## Best Practices

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use `golint` for linting
- Write tests for all new code
- Document public functions and types

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/workflow-templates

# Make changes and commit
git add .
git commit -m "feat: add workflow templates"

# Push and create PR
git push origin feature/workflow-templates
```

### Commit Message Format

```
type(scope): subject

body

footer
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:
```
feat(engine): add parallel step execution

Implement parallel execution strategy for workflow steps
with configurable concurrency limits.

Closes #123
```

## Resources

### Documentation

- [Architecture](./ARCHITECTURE.md)
- [Database Decision](./DATABASE_DECISION.md)
- [Implementation Roadmap](./IMPLEMENTATION_ROADMAP.md)
- [AI Agent Examples](./examples/ai-agent-examples.md)

### Example Workflows

- [High Value Order Approval](./examples/workflows/high-value-order-approval.json)
- [Cart Fraud Detection](./examples/workflows/cart-fraud-detection.json)
- [Product Inventory Sync](./examples/workflows/product-inventory-sync.json)
- [Quote Expiration Management](./examples/workflows/quote-expiration-management.json)

### External Resources

- [Go Documentation](https://golang.org/doc/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Redis Documentation](https://redis.io/documentation)
- [React Documentation](https://react.dev/)
- [Chi Router](https://github.com/go-chi/chi)
- [sqlc Documentation](https://docs.sqlc.dev/)

## Getting Help

### Community

- GitHub Issues: [Report bugs or request features](https://github.com/yourorg/intelligent-workflows/issues)
- Discussions: [Ask questions and share ideas](https://github.com/yourorg/intelligent-workflows/discussions)
- Discord: [Join our community](https://discord.gg/workflows)

### Support

For enterprise support, contact: support@yourcompany.com

## Next Steps

1. ✅ Complete local setup
2. ✅ Create your first workflow
3. ✅ Explore example workflows
4. ✅ Read the [Architecture](./ARCHITECTURE.md) document
5. ✅ Review the [Implementation Roadmap](./IMPLEMENTATION_ROADMAP.md)
6. ✅ Start implementing Phase 1 features
7. ✅ Join our community

Happy coding! 🚀
