# Intelligent Workflows

An event-driven workflow automation system with built-in approval mechanisms and comprehensive API.

## Features

- 🔄 **Event-Driven Workflows** - Trigger workflows automatically based on events
- ✅ **Approval Management** - Built-in approval workflows with role-based routing
- 🔐 **Secure Authentication** - JWT and API Key authentication with RBAC
- 📊 **Execution Tracking** - Monitor and trace workflow executions in real-time
- 🚀 **REST API** - Comprehensive RESTful API with OpenAPI specification
- 📚 **Interactive Documentation** - Built-in Swagger UI for API exploration
- ⚡ **High Performance** - Built with Go for speed and efficiency
- 🔒 **Production Ready** - Rate limiting, monitoring, and security best practices

## Quick Start

### Prerequisites

- Go 1.24.7 or higher
- PostgreSQL 15+
- Redis 7+
- Docker and Docker Compose (optional)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/davidmoltin/intelligent-workflows.git
cd intelligent-workflows
```

2. Install dependencies:
```bash
go mod download
```

3. Start dependencies (PostgreSQL and Redis):
```bash
docker-compose up -d postgres redis
```

4. Run database migrations:
```bash
./bin/migrate up
```

5. Start the API server:
```bash
go run cmd/api/main.go
```

The API will be available at `http://localhost:8080`.

## API Documentation

Comprehensive API documentation is available:

- **[Interactive API Docs (Swagger UI)](http://localhost:8080/api/v1/docs/ui)** - Try the API in your browser
- **[OpenAPI Specification](http://localhost:8080/api/v1/docs/openapi.yaml)** - Complete API spec
- **[Authentication Guide](./docs/api/AUTHENTICATION.md)** - JWT and API Key authentication
- **[API Examples](./docs/api/EXAMPLES.md)** - Practical examples with curl commands
- **[API Documentation Overview](./docs/api/README.md)** - Complete documentation index

### Quick API Example

```bash
# Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "demo",
    "email": "demo@example.com",
    "password": "SecurePass123!"
  }'

# Login to get access token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "demo",
    "password": "SecurePass123!"
  }'

# Create a workflow (use the access_token from login response)
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "order_approval",
    "version": "1.0.0",
    "name": "Order Approval Workflow",
    "definition": {
      "trigger": {
        "type": "event",
        "event": "order.created"
      },
      "steps": [...]
    }
  }'
```

## Architecture

The system consists of several key components:

- **REST API** - HTTP API for workflow management and event emission
- **Workflow Engine** - Event-driven workflow execution engine
- **Approval Service** - Manages approval requests and decisions
- **Authentication Service** - JWT and API Key authentication with RBAC
- **Event Router** - Routes events to matching workflows
- **Execution Tracker** - Tracks and stores workflow execution history

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Project Structure

```
intelligent-workflows/
├── cmd/                    # Application entry points
│   ├── api/               # REST API server
│   └── migrate/           # Database migration tool
├── internal/              # Private application code
│   ├── api/              # API handlers and middleware
│   ├── engine/           # Workflow engine
│   ├── models/           # Domain models
│   ├── repository/       # Data access layer
│   └── services/         # Business logic
├── pkg/                   # Public libraries
│   ├── auth/             # Authentication utilities
│   └── logger/           # Logging utilities
├── docs/                  # Documentation
│   └── api/              # API documentation
│       ├── README.md     # API docs overview
│       ├── openapi.yaml  # OpenAPI 3.0 specification
│       ├── AUTHENTICATION.md
│       └── EXAMPLES.md
├── migrations/            # Database migrations
└── docker-compose.yml     # Docker services
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

### Building

```bash
# Build the API server
go build -o bin/api ./cmd/api

# Build with Docker
docker build -t intelligent-workflows:latest .
```

### Code Quality

```bash
# Run linter
golangci-lint run

# Format code
go fmt ./...

# Run security checks
gosec ./...
```

## Configuration

The application is configured through environment variables. Copy `.env.example` to `.env` and customize as needed:

```bash
cp .env.example .env
```

### Environment Variables

#### Server Configuration
- `SERVER_HOST` - Server host address (default: `0.0.0.0`)
- `SERVER_PORT` - Server port (default: `8080`)
- `SERVER_READ_TIMEOUT` - HTTP read timeout (default: `15s`)
- `SERVER_WRITE_TIMEOUT` - HTTP write timeout (default: `15s`)
- `SERVER_SHUTDOWN_TIMEOUT` - Graceful shutdown timeout (default: `30s`)

#### Database Configuration
- `DB_HOST` - PostgreSQL host (default: `localhost`)
- `DB_PORT` - PostgreSQL port (default: `5432`)
- `DB_USER` - Database user (default: `postgres`)
- `DB_PASSWORD` - Database password (**required in production**)
- `DB_NAME` - Database name (default: `workflows`)
- `DB_SSL_MODE` - SSL mode: `disable`, `require`, `verify-ca`, `verify-full` (default: `require`)
- `DB_MAX_OPEN_CONNS` - Maximum open connections (default: `25`)
- `DB_MAX_IDLE_CONNS` - Maximum idle connections (default: `5`)
- `DB_CONN_MAX_LIFETIME` - Connection max lifetime (default: `5m`)

#### Redis Configuration
- `REDIS_HOST` - Redis host (default: `localhost`)
- `REDIS_PORT` - Redis port (default: `6379`)
- `REDIS_PASSWORD` - Redis password (optional)
- `REDIS_DB` - Redis database number (default: `0`)

#### Application Configuration
- `APP_ENV` - Environment: `development`, `staging`, `production` (default: `development`)
- `APP_VERSION` - Application version (default: `0.1.0`)
- `APP_NAME` - Application name (default: `intelligent-workflows`)
- `DEFAULT_APPROVER_EMAIL` - Default approver email (default: `approver@example.com`)

#### Authentication & Security
- `JWT_SECRET` - JWT signing secret (**required in production, will fail if not set**)
- `JWT_ACCESS_TOKEN_TTL` - Access token TTL (default: `15m`)
- `JWT_REFRESH_TOKEN_TTL` - Refresh token TTL (default: `168h`)
- `ALLOWED_ORIGINS` - Comma-separated CORS origins (default: `http://localhost:3000`)

#### Notification Configuration
- `NOTIFICATION_BASE_URL` - Base URL for notification links (default: `http://localhost:8080`)
- `NOTIFICATION_EMAIL_ENABLED` - Enable email notifications (default: `false`)
- `NOTIFICATION_SMTP_HOST` - SMTP server host (default: `smtp.gmail.com`)
- `NOTIFICATION_SMTP_PORT` - SMTP server port (default: `587`)
- `NOTIFICATION_SMTP_USER` - SMTP username
- `NOTIFICATION_SMTP_PASSWORD` - SMTP password
- `NOTIFICATION_FROM_ADDRESS` - Email from address (default: `noreply@example.com`)
- `NOTIFICATION_SLACK_ENABLED` - Enable Slack notifications (default: `false`)
- `NOTIFICATION_SLACK_WEBHOOK_URL` - Slack webhook URL

#### LLM Provider Configuration
- `LLM_PROVIDER` - LLM provider: `anthropic` or `openai` (default: `anthropic`)
- `LLM_API_KEY` - LLM API key (required for AI features)
- `LLM_DEFAULT_MODEL` - Default model to use (optional)
- `LLM_TIMEOUT` - LLM request timeout (default: `60s`)
- `LLM_MAX_RETRIES` - Maximum retry attempts (default: `3`)
- `LLM_RETRY_DELAY` - Delay between retries (default: `1s`)
- `LLM_BASE_URL` - Custom LLM API base URL (optional)

#### Rate Limiting
- `RATE_LIMIT_REQUESTS_PER_SECOND` - Requests per second (default: `100`)
- `RATE_LIMIT_BURST` - Burst capacity (default: `200`)

#### Background Workers
- `WORKER_APPROVAL_EXPIRATION_INTERVAL` - Approval expiration check interval (default: `5m`)
- `WORKER_WORKFLOW_RESUMER_INTERVAL` - Workflow resumer check interval (default: `1m`)
- `WORKER_TIMEOUT_ENFORCER_INTERVAL` - Timeout enforcer check interval (default: `1m`)
- `WORKER_SCHEDULER_INTERVAL` - Scheduler check interval (default: `1m`)

#### Context Enrichment
- `CONTEXT_ENRICHMENT_ENABLED` - Enable context enrichment from microservices (default: `true`)
- `CONTEXT_ENRICHMENT_BASE_URL` - Base URL for context enrichment services (default: `http://localhost:8081`)
- `CONTEXT_ENRICHMENT_TIMEOUT` - Request timeout (default: `10s`)
- `CONTEXT_ENRICHMENT_MAX_RETRIES` - Maximum retry attempts (default: `3`)
- `CONTEXT_ENRICHMENT_RETRY_DELAY` - Delay between retries with exponential backoff (default: `500ms`)
- `CONTEXT_ENRICHMENT_CACHE_TTL` - Cache TTL for enriched data (default: `5m`)

#### Logging
- `LOG_LEVEL` - Log level: `debug`, `info`, `warn`, `error` (default: `info`)
- `LOG_FORMAT` - Log format: `json` or `text` (default: `json`)

### Security Considerations

⚠️ **Important**: In production environments:
- Always set `JWT_SECRET` to a strong random value (application will fail to start without it)
- Change `DB_PASSWORD` from the default value
- Set `DB_SSL_MODE=require` or higher
- Use strong SMTP credentials if email notifications are enabled
- Limit `ALLOWED_ORIGINS` to your specific frontend domains

See the complete `.env.example` file for a full configuration template.

## Deployment

### Docker

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop services
docker-compose down
```

### Kubernetes

```bash
# Apply Kubernetes manifests
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -n intelligent-workflows

# View logs
kubectl logs -f deployment/intelligent-workflows-api -n intelligent-workflows
```

See [INFRASTRUCTURE.md](./INFRASTRUCTURE.md) for detailed deployment documentation.

## Monitoring and Observability

The system includes comprehensive monitoring and observability:

- **Prometheus Metrics** - Application and infrastructure metrics exposed at `/metrics`
- **Grafana Dashboards** - Pre-configured dashboards for visualization
- **AlertManager** - Alerting based on metric thresholds
- **Structured Logging** - JSON-formatted logs with correlation IDs

### Available Metrics

- HTTP requests (count, latency, size)
- Workflow executions (count, duration, errors, active)
- Database and Redis connection health
- Approvals, notifications, and AI requests
- Background worker performance
- Authentication activity

### Access Monitoring

- **Metrics Endpoint**: http://localhost:8080/metrics
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9091

See [docs/METRICS.md](./docs/METRICS.md) for complete metrics documentation and example queries.

## Security

Security features include:

- 🔐 JWT authentication with 15-minute expiration
- 🔑 API Key authentication for service-to-service
- 🛡️ Bcrypt password hashing (cost factor 12)
- 🚦 Rate limiting (100 req/min per user)
- 🎭 Role-based access control (RBAC)
- 🔒 Permission-based authorization
- 📝 Audit logging for authentication events
- 🔄 Token rotation on refresh

See [AUTHENTICATION.md](./docs/api/AUTHENTICATION.md) for security best practices.

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure:
- All tests pass
- Code is formatted with `go fmt`
- Linter passes without errors
- Documentation is updated

## Documentation

- [Getting Started](./GETTING_STARTED.md) - Setup and basic usage
- [Architecture](./ARCHITECTURE.md) - System design and components
- [API Documentation](./docs/api/README.md) - Complete API reference
- [Metrics and Monitoring](./docs/METRICS.md) - Prometheus metrics and observability
- [Testing Guide](./TESTING.md) - Testing strategy and guidelines
- [Infrastructure](./INFRASTRUCTURE.md) - Deployment and operations
- [Implementation Roadmap](./IMPLEMENTATION_ROADMAP.md) - Development timeline

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/davidmoltin/intelligent-workflows/issues)
- **Discussions**: [GitHub Discussions](https://github.com/davidmoltin/intelligent-workflows/discussions)
- **Documentation**: [docs/api/](./docs/api/)

## Acknowledgments

Built with:
- [Go](https://golang.org/) - Programming language
- [Chi](https://github.com/go-chi/chi) - HTTP router
- [PostgreSQL](https://www.postgresql.org/) - Database
- [Redis](https://redis.io/) - Caching and sessions
- [JWT](https://jwt.io/) - Authentication tokens
- [OpenAPI](https://www.openapis.org/) - API specification
- [Swagger UI](https://swagger.io/tools/swagger-ui/) - API documentation

---

Made with ❤️ by the Intelligent Workflows team
