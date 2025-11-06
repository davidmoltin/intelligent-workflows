# Prometheus Instrumentation Analysis
## Intelligent Workflows Service

---

## EXECUTIVE SUMMARY

This is a **Go-based workflow orchestration API** with a **React frontend**, built with enterprise-grade monitoring infrastructure already in place. The project has:
- Prometheus configured and ready for metrics collection
- Grafana dashboards for visualization
- Alert rules defined
- Full monitoring stack with PostgreSQL and Redis exporters
- BUT: **No application-level metrics instrumentation yet** - the dashboards reference metrics that don't exist

---

## 1. PROMETHEUS CONFIGURATION FILES

### Location and Structure

**Primary Prometheus Config:**
- `/home/user/intelligent-workflows/monitoring/prometheus/prometheus.yml`

**Alert Rules:**
- `/home/user/intelligent-workflows/monitoring/alerts/api_alerts.yml`

**Grafana Datasource Config:**
- `/home/user/intelligent-workflows/monitoring/grafana/provisioning/datasources/prometheus.yml`

**Docker Compose Integration:**
- `/home/user/intelligent-workflows/docker-compose.yml`

### Prometheus Configuration Details

```yaml
Global Settings:
  - Scrape interval: 15s
  - Scrape timeout: 10s
  - Evaluation interval: 15s
  - External labels: cluster='intelligent-workflows', environment='production'

Scrape Configs Defined:
  1. Prometheus self-monitoring (localhost:9090)
  2. workflows-api (workflows-api:9090) - WHERE YOUR METRICS SHOULD EXPOSE
  3. Kubernetes service discovery (for k8s deployments)
  4. PostgreSQL exporter (postgres-exporter:9187)
  5. Redis exporter (redis-exporter:9121)
  6. Node exporter (node-exporter:9100)

Alert Manager: Configured at alertmanager:9093

Ports:
  - Prometheus UI: localhost:9091 (mapped from 9090)
  - Metrics endpoint expected at: /metrics
```

### Alert Rules Currently Defined

The `api_alerts.yml` defines alerts that **reference metrics not yet instrumented**:
- `up{job="workflows-api"}` - Service availability
- `http_requests_total` - HTTP request counts
- `http_request_duration_seconds` - Request latency
- `process_resident_memory_bytes` - Memory usage
- `process_cpu_seconds_total` - CPU usage
- `db_connections_failed_total` - Database connection failures
- `redis_connections_failed_total` - Redis connection failures

---

## 2. APPLICATION STRUCTURE & TECH STACK

### Technology Stack

**Backend:**
- **Language:** Go 1.24.0
- **Framework:** chi/v5 (lightweight HTTP router)
- **Database:** PostgreSQL 15 + migrations
- **Cache:** Redis 7
- **Logging:** go.uber.org/zap (structured JSON logging)
- **Authentication:** JWT + API Keys
- **LLM Integration:** Anthropic & OpenAI support

**Frontend:**
- **Framework:** React 19.1.1 + TypeScript
- **Build Tool:** Vite 7.1.7
- **HTTP Client:** Fetch API (custom wrapper)
- **UI Components:** Radix UI
- **Visualizations:** ReCharts, React Flow
- **State Management:** Zustand

### Main Application Components

#### Backend Structure (`/cmd/api/main.go`)

```
API Server (Port 8080)
├── Health Endpoints
├── REST API (v1)
├── Authentication Service
├── Workflow Engine
├── Event Router
├── Notification Service
└── Background Workers
    ├── Approval Expiration Worker
    ├── Workflow Resumer Worker
    └── Timeout Enforcer Worker
```

#### Core Services (`/internal/`)

1. **Workflow Engine** (`/internal/engine/`)
   - `executor.go` - Executes workflow steps
   - `event_router.go` - Routes events to workflows
   - `action_executor.go` - Executes workflow actions
   - `context.go` - Builds execution context
   - `evaluator.go` - Evaluates conditions

2. **API Handlers** (`/internal/api/rest/handlers/`)
   - `workflow.go` - Workflow CRUD operations
   - `execution.go` - Execution lifecycle management
   - `event.go` - Event ingestion
   - `approval.go` - Approval workflows
   - `analytics.go` - Analytics & reporting
   - `auth.go` - Authentication endpoints
   - `ai.go` - AI capabilities (if configured)

3. **Business Services** (`/internal/services/`)
   - `approval_service.go` - Approval workflow logic
   - `auth_service.go` - JWT & API key management
   - `notification_service.go` - Email/notification delivery
   - `workflow_resumer.go` - Resume paused workflows
   - `ai_service.go` - AI chat & interpretation

4. **Background Workers** (`/internal/workers/`)
   - `approval_expiration_worker.go` - Expires pending approvals
   - `workflow_resumer_worker.go` - Resumes paused executions
   - `timeout_enforcer_worker.go` - Enforces execution timeouts

5. **Data Layer** (`/internal/repository/postgres/`)
   - Workflow repository
   - Execution repository
   - Analytics repository
   - Event repository
   - Approval repository
   - User & Auth repositories

#### Frontend Structure (`/web/src/`)

```
API Client (`/api/`)
├── client.ts - Centralized fetch wrapper
└── hooks.ts - React Query hooks

Pages:
├── WorkflowsPage - Workflow list/management
├── WorkflowDetailPage - Single workflow view
├── EditWorkflowPage - Workflow editor
├── ExecutionDetailPage - Execution trace
├── AnalyticsPage - Metrics & dashboards
└── ApprovalsPage - Approval management

Components: Workflow graph, execution trace, approval dialogs
```

### API Endpoints (From `/internal/api/rest/router.go`)

```
Public:
  GET  /health
  GET  /ready
  POST /api/v1/auth/register
  POST /api/v1/auth/login
  POST /api/v1/auth/refresh
  POST /api/v1/auth/logout

Protected (JWT/API Key):
  GET    /api/v1/workflows           (list all)
  POST   /api/v1/workflows           (create)
  GET    /api/v1/workflows/{id}      (detail)
  PUT    /api/v1/workflows/{id}      (update)
  DELETE /api/v1/workflows/{id}      (delete)
  POST   /api/v1/workflows/{id}/enable
  POST   /api/v1/workflows/{id}/disable

  POST   /api/v1/events              (ingest events)

  GET    /api/v1/executions          (list)
  GET    /api/v1/executions/paused   (paused only)
  GET    /api/v1/executions/{id}     (detail)
  GET    /api/v1/executions/{id}/trace
  POST   /api/v1/executions/{id}/pause
  POST   /api/v1/executions/{id}/resume

  GET    /api/v1/approvals           (list)
  GET    /api/v1/approvals/{id}      (detail)
  POST   /api/v1/approvals/{id}/approve
  POST   /api/v1/approvals/{id}/reject

  GET    /api/v1/analytics/dashboard
  GET    /api/v1/analytics/stats
  GET    /api/v1/analytics/trends
  GET    /api/v1/analytics/workflows
  GET    /api/v1/analytics/errors
  GET    /api/v1/analytics/steps

  POST   /api/v1/ai/chat
  GET    /api/v1/ai/capabilities
  POST   /api/v1/ai/interpret
```

---

## 3. EXISTING METRICS & INSTRUMENTATION

### Current State

**Status:** ❌ **NO APPLICATION METRICS INSTRUMENTATION**

The project has:
- ✅ Prometheus configured and running
- ✅ Alert rules defined (but referencing non-existent metrics)
- ✅ Grafana dashboards (expecting specific metrics)
- ✅ Structured logging with Zap (JSON format)
- ✅ Middleware for logging, auth, rate limiting
- ❌ **NO Prometheus client library imported**
- ❌ **NO metric collectors instantiated**
- ❌ **NO metrics endpoint implementation**
- ❌ **NO instrumentation in critical paths**

### What Exists: Logging Middleware

File: `/internal/api/rest/middleware/logger.go`

```go
// Current middleware captures:
- HTTP method
- Request path
- Remote address
- User agent
- HTTP status code
- Response bytes
- Request duration
- Request ID
```

This provides **request-level visibility** but not the **Prometheus metrics format** that the alerts and dashboards expect.

---

## 4. PACKAGE.JSON & DEPENDENCIES

### Go Dependencies

**No Prometheus client library is currently imported.**

```go
Key dependencies:
- github.com/go-chi/chi/v5 v5.2.3 (HTTP router)
- github.com/jackc/pgx/v5 v5.7.6 (PostgreSQL driver)
- github.com/redis/go-redis/v9 v9.16.0 (Redis client)
- go.uber.org/zap v1.27.0 (Structured logging)
- github.com/google/uuid v1.6.0 (UUID generation)
- github.com/golang-jwt/jwt/v5 v5.3.0 (JWT)
```

### React Dependencies

```json
No metrics/monitoring libraries in web app.

Key dependencies:
- react 19.1.1
- react-router-dom 7.9.5
- @tanstack/react-query 5.90.7 (API state management)
- recharts 3.3.0 (charting)
- reactflow 11.11.4 (workflow visualization)
- zustand 5.0.8 (state management)
- @radix-ui/react-* (component library)
```

---

## 5. ENTRY POINTS & INITIALIZATION

### Backend Entry Point

**File:** `/cmd/api/main.go`

**Startup Sequence:**
1. Load configuration
2. Initialize logger
3. Initialize PostgreSQL
4. Initialize Redis
5. Initialize repositories
6. Initialize workflow engine (executor + event router)
7. Initialize services (notification, approval, auth, AI)
8. Start background workers
   - Approval expiration worker (5 min interval)
   - Workflow resumer worker (1 min interval)
   - Timeout enforcer worker (1 min interval)
9. Initialize HTTP router with middleware
10. Start HTTP server (Port 8080)
11. Wait for SIGTERM/SIGINT for graceful shutdown

**Critical Missing:** No Prometheus metrics initialization or HTTP handler for `/metrics`

### Frontend Entry Point

**File:** `/web/src/main.tsx`

Vite development server, builds with React + TypeScript.

---

## 6. RECOMMENDED PROMETHEUS INSTRUMENTATION POINTS

### A. Core HTTP Metrics (Highest Priority)

**Metric Type:** Counter + Histogram

**Where:** Router middleware (`/internal/api/rest/router.go`)

**Metrics to Collect:**
```
http_requests_total{method, path, status, service}
  - Count of HTTP requests by method, endpoint, and status

http_request_duration_seconds{method, path, status, service}
  - Request latency histogram

http_request_size_bytes{method, path, service}
  - Request body size

http_response_size_bytes{method, path, status, service}
  - Response body size
```

**Alert thresholds already expecting these:**
- High error rate > 5% for 5m
- P95 latency > 1 second

### B. Workflow Execution Metrics

**Where:** `/internal/engine/executor.go`

**Metrics:**
```
workflow_executions_total{workflow_id, status, trigger_type}
  - Total executions started/completed

workflow_execution_duration_seconds{workflow_id}
  - Histogram of execution times

workflow_step_duration_seconds{workflow_id, step_name, status}
  - Step-level execution times

workflow_execution_errors_total{workflow_id, error_type}
  - Failed executions by type

active_workflow_executions{workflow_id}
  - Gauge of currently running executions
```

### C. Database Metrics

**Where:** Database connection pool (initialized in `/cmd/api/main.go`)

**Metrics:**
```
db_connections_active{database}
  - Current active connections

db_connections_failed_total{database}
  - Failed connection attempts (referenced in alerts)

db_query_duration_seconds{query_type, table}
  - Query execution time

db_query_errors_total{query_type, error_type}
  - Failed queries

db_pool_size{database}
  - Connection pool size
```

### D. Redis Metrics

**Where:** Redis client initialization (in `/cmd/api/main.go`)

**Metrics:**
```
redis_connections_active
  - Current active Redis connections

redis_connections_failed_total
  - Failed connection attempts (referenced in alerts)

redis_operations_duration_seconds{operation}
  - Redis command execution time

redis_operations_errors_total{operation}
  - Failed Redis operations
```

### E. Business Logic Metrics

**Where:** Services (`/internal/services/`)

**Metrics:**
```
approvals_total{status, workflow_id}
  - Approvals created/processed

approvals_duration_seconds{workflow_id}
  - Time to approval

notifications_sent_total{type, status}
  - Email/notification delivery

ai_requests_total{model, status}
  - AI API calls

ai_request_duration_seconds{model}
  - AI API latency
```

### F. Background Worker Metrics

**Where:** `/internal/workers/`

**Metrics:**
```
worker_jobs_processed_total{worker_type, status}
  - Jobs processed by each worker

worker_job_duration_seconds{worker_type}
  - Time per job

worker_errors_total{worker_type}
  - Worker errors
```

### G. Authentication & Security Metrics

**Where:** `/internal/services/auth_service.go`

**Metrics:**
```
auth_requests_total{method, status}
  - Login/registration attempts

auth_failures_total{reason}
  - Auth failures (invalid token, expired, etc)

auth_token_validations_total{valid}
  - JWT validation attempts
```

---

## 7. RECOMMENDED INSTRUMENTATION STRATEGY

### Phase 1: Critical Path (Week 1)
1. **Add Prometheus client library** to `go.mod`
   - `github.com/prometheus/client_golang v1.x`
2. **Implement HTTP metrics middleware**
   - Counter: http_requests_total
   - Histogram: http_request_duration_seconds
   - Expose at `/metrics` endpoint
3. **Update router** to register metrics middleware
4. **Update docker-compose.yml** - already configured, just needs working metrics

### Phase 2: Business Logic (Week 2)
5. **Workflow execution metrics**
   - Instrument executor.Execute()
   - Track workflow_executions_total, duration
6. **Database metrics**
   - Instrument repository layer
7. **Authentication metrics**
   - Track login attempts and failures

### Phase 3: Operational (Week 3)
8. **Background worker metrics**
9. **Service-level metrics** (approvals, notifications)
10. **Custom Grafana dashboards**

### Phase 4: Fine-tuning
11. **Application metrics** (business KPIs)
12. **Frontend RUM metrics** (if needed)

---

## 8. GRAFANA DASHBOARDS

### Existing Dashboards

**Location:** `/monitoring/grafana/dashboards/`

1. **api-overview.json**
   - Request rate (by method, status)
   - Error rate %
   - P95 latency
   - Memory/CPU usage
   - Panels expecting: `http_requests_total`, `http_request_duration_seconds`

2. **workflow-metrics.json**
   - (Content not fully reviewed but likely expects workflow metrics)

### Provisioning Config

**Location:** `/monitoring/grafana/provisioning/`

- Datasources: Prometheus on localhost:9090
- Dashboards auto-load from `/etc/grafana/provisioning/dashboards/`

---

## 9. NEXT STEPS

1. **Install prometheus/client_golang**: Add to `go.mod`
   ```bash
   go get github.com/prometheus/client_golang/prometheus
   ```

2. **Create metrics package** at `/pkg/metrics/`
   - Registry initialization
   - Metric definitions
   - Middleware factory

3. **Implement HTTP metrics middleware** in router

4. **Expose `/metrics` endpoint** in router

5. **Add metrics collection** to critical paths:
   - HTTP handlers
   - Workflow executor
   - Database operations
   - Background workers

6. **Test with Docker Compose**:
   ```bash
   docker-compose up
   # Prometheus UI: http://localhost:9091
   # Grafana UI: http://localhost:3000
   # API: http://localhost:8080
   ```

7. **Verify metrics** appear in Prometheus
   - Check target status
   - Query metrics in Prometheus UI
   - Verify dashboards populate

---

## FILE LOCATIONS SUMMARY

| Component | Location |
|-----------|----------|
| Prometheus Config | `/monitoring/prometheus/prometheus.yml` |
| Alert Rules | `/monitoring/alerts/api_alerts.yml` |
| Grafana Dashboards | `/monitoring/grafana/dashboards/` |
| Grafana Provisioning | `/monitoring/grafana/provisioning/` |
| API Entry Point | `/cmd/api/main.go` |
| HTTP Router | `/internal/api/rest/router.go` |
| Middleware | `/internal/api/rest/middleware/` |
| Workflow Executor | `/internal/engine/executor.go` |
| Services | `/internal/services/` |
| Workers | `/internal/workers/` |
| Docker Compose | `/docker-compose.yml` |
| Go Dependencies | `/go.mod` |

