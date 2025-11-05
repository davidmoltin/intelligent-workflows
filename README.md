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

Configuration can be provided via environment variables or a configuration file:

```bash
# Database
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=password
export DB_NAME=intelligent_workflows

# Redis
export REDIS_HOST=localhost
export REDIS_PORT=6379

# JWT
export JWT_SECRET=your-secret-key
export JWT_EXPIRATION=15m

# API
export API_PORT=8080
export API_HOST=0.0.0.0
```

See [GETTING_STARTED.md](./GETTING_STARTED.md) for detailed configuration options.

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

## Monitoring

The system includes built-in monitoring with:

- **Prometheus** - Metrics collection
- **Grafana** - Visualization dashboards
- **Jaeger** - Distributed tracing
- **Structured Logging** - JSON-formatted logs with correlation IDs

Access monitoring dashboards:
- Grafana: http://localhost:3000
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686

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
