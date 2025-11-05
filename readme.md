# Intelligent Workflows for E-commerce

A state-of-the-art, API-first workflow orchestration platform designed for e-commerce microservices. Built to be both human-friendly and AI-agent-ready, removing the complexity typically found in traditional workflow platforms.

## 🎯 Overview

Intelligent Workflows allows you to:
- **Control actions** across multiple microservices (orders, quotes, products, carts, etc.)
- **Allow or block** operations based on sophisticated business rules
- **Automate processes** with visual workflows or code-based definitions
- **Integrate AI agents** to dynamically create and manage workflows
- **Simplify complexity** with an intuitive React UI for non-technical users

## ✨ Key Features

### 🤖 AI-Native Design
- Natural language workflow creation
- AI agents can read, create, and execute workflows
- Structured API for programmatic workflow management
- Real-time execution monitoring for agents

### 🚀 Developer-First
- **API-First**: Every feature accessible via REST/GraphQL
- **Code as Configuration**: Define workflows in JSON/YAML
- **Type-Safe**: Go with PostgreSQL for robust, scalable backend
- **CLI Tool**: Manage workflows from the command line
- **Version Control**: Workflows are code - commit, diff, review

### 👤 User-Friendly
- **Visual Workflow Builder**: Drag-and-drop interface
- **Pre-built Templates**: Start with proven workflow patterns
- **No-Code Conditions**: Build rules without coding
- **Real-Time Monitoring**: Watch workflows execute live
- **Approval Dashboard**: Manage pending approvals easily

### 🏗️ Enterprise-Grade
- **Scalable**: Handle thousands of workflows concurrently
- **Reliable**: ACID transactions, automatic retries
- **Observable**: Built-in monitoring, tracing, and audit logs
- **Secure**: Role-based access control, encrypted secrets
- **Performant**: Sub-second execution for most workflows

## 🏛️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│              React UI • AI Agents • CLI                      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│              REST API • GraphQL • WebSockets                 │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│    Workflow Engine • Rule Evaluator • Action Executor        │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│         PostgreSQL • Redis • (Optional: MongoDB)             │
└─────────────────────────────────────────────────────────────┘
```

**Technology Stack:**
- **Backend**: Go 1.21+, Chi/Fiber, PostgreSQL 15+, Redis 7+
- **Frontend**: React 18+, TypeScript, Tailwind CSS, shadcn/ui
- **Deployment**: Docker, Kubernetes
- **Monitoring**: Prometheus, Grafana, Jaeger

## 📖 Documentation

- **[Architecture](./ARCHITECTURE.md)** - Comprehensive system design and technical decisions
- **[Getting Started](./GETTING_STARTED.md)** - Set up your development environment
- **[Implementation Roadmap](./IMPLEMENTATION_ROADMAP.md)** - 16-week plan to MVP
- **[Database Decision](./DATABASE_DECISION.md)** - PostgreSQL vs MongoDB analysis
- **[AI Agent Examples](./examples/ai-agent-examples.md)** - How AI agents interact with the system

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (for UI)
- Docker (recommended)

### Using Docker (Fastest)

```bash
# Clone the repository
git clone https://github.com/yourorg/intelligent-workflows.git
cd intelligent-workflows

# Start all services
docker-compose up -d

# Run migrations
make migrate-up

# Verify
curl http://localhost:8080/health
```

### Manual Setup

```bash
# Install dependencies
go mod download

# Set up database
createdb workflows
migrate -database "postgresql://localhost/workflows?sslmode=disable" \
        -path migrations/postgres up

# Start Redis
redis-server

# Run the API server
go run cmd/api/main.go
```

See [Getting Started](./GETTING_STARTED.md) for detailed instructions.

## 💡 Example Workflow

Here's a simple workflow that requires approval for high-value orders:

```json
{
  "workflow_id": "high_value_order_approval",
  "version": "1.0.0",
  "name": "High Value Order Approval",
  "trigger": {
    "type": "event",
    "event": "order.checkout.initiated"
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
      "action": "block",
      "reason": "Order requires approval for amounts over $10,000",
      "execute": [
        {
          "type": "notify",
          "channel": "email",
          "recipients": ["role:sales_manager"],
          "template": "order_approval_required"
        }
      ]
    },
    {
      "id": "allow_order",
      "type": "action",
      "action": "allow"
    }
  ]
}
```

### Deploy and Test

```bash
# Deploy workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @high-value-order-approval.json

# Trigger with an event
curl -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "order.checkout.initiated",
    "payload": {"order": {"id": "ord_123", "total": 15000}}
  }'

# View execution
curl http://localhost:8080/api/v1/executions
```

More examples in [examples/workflows/](./examples/workflows/)

## 🤖 AI Agent Integration

AI agents can create workflows using natural language:

```json
POST /api/v1/ai/interpret
{
  "prompt": "Block all orders over $5000 from customers who signed up in the last 30 days"
}
```

The service responds with a suggested workflow definition that the agent can review and deploy.

See [AI Agent Examples](./examples/ai-agent-examples.md) for comprehensive integration patterns.

## 🛠️ Development

### Project Structure

```
intelligent-workflows/
├── cmd/                    # Application entry points
│   ├── api/               # API server
│   ├── worker/            # Background worker
│   └── cli/               # CLI tool
├── internal/              # Private application code
│   ├── api/              # API handlers and routes
│   ├── engine/           # Workflow execution engine
│   ├── models/           # Data models
│   ├── repository/       # Data access layer
│   └── services/         # Business logic
├── pkg/                  # Public libraries
├── migrations/           # Database migrations
├── web/                  # React frontend
├── examples/             # Example workflows
├── docs/                 # Documentation
└── tests/               # Integration tests
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./tests/...
```

### Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## 📊 Use Cases

### E-commerce Operations
- **Order Approval**: Require approval for high-value or risky orders
- **Fraud Detection**: Block suspicious transactions automatically
- **Inventory Management**: Sync inventory across warehouses, prevent overselling
- **Pricing Rules**: Dynamic pricing based on customer tier, quantity, time
- **Quote Management**: Automate quote lifecycle with reminders and expiration

### Customer Experience
- **Cart Abandonment**: Send recovery emails with discount codes
- **Order Updates**: Notify customers at each order stage
- **Personalization**: Trigger personalized recommendations
- **Loyalty Programs**: Automate tier upgrades and rewards

### Compliance & Risk
- **Audit Logging**: Track all decisions and actions
- **Approval Chains**: Multi-level approvals for sensitive operations
- **Data Retention**: Automatically archive old data
- **Regulatory Compliance**: Enforce business rules consistently

## 🎯 Roadmap

### Phase 1: MVP (Weeks 1-8) ✅ Planning Complete
- Core workflow engine
- CRUD API
- Event routing
- Basic UI

### Phase 2: Advanced Features (Weeks 9-13)
- Parallel execution
- Approval workflows
- AI integration
- Visual workflow builder

### Phase 3: Production Ready (Weeks 14-16)
- Performance optimization
- Security hardening
- Monitoring & alerting
- Deployment automation

### Phase 4: Enhanced Features (Post-MVP)
- GraphQL API
- Workflow versioning
- A/B testing
- Workflow marketplace
- Mobile app

See [Implementation Roadmap](./IMPLEMENTATION_ROADMAP.md) for details.

## 🤝 Support

- **Documentation**: [docs/](./docs/)
- **Examples**: [examples/](./examples/)
- **Issues**: [GitHub Issues](https://github.com/yourorg/intelligent-workflows/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourorg/intelligent-workflows/discussions)
- **Enterprise Support**: support@yourcompany.com

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

## 🙏 Acknowledgments

Built with:
- [Go](https://golang.org/)
- [PostgreSQL](https://www.postgresql.org/)
- [Redis](https://redis.io/)
- [React](https://react.dev/)
- [Chi Router](https://github.com/go-chi/chi)
- [sqlc](https://sqlc.dev/)

## 🌟 Why Intelligent Workflows?

Traditional workflow platforms are often:
- ❌ Complex to learn and use
- ❌ Require expensive enterprise licenses
- ❌ Not designed for AI agent interaction
- ❌ Vendor lock-in with proprietary formats
- ❌ Difficult to version control and test

Intelligent Workflows is:
- ✅ Simple JSON/YAML definitions
- ✅ Open-source and extensible
- ✅ AI-native from day one
- ✅ Git-friendly (workflows as code)
- ✅ Easy to test and debug

---

**Ready to get started?** Check out the [Getting Started Guide](./GETTING_STARTED.md)
