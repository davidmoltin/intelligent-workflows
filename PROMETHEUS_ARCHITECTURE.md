# Prometheus Instrumentation Architecture

## System Architecture Overview

```
┌────────────────────────────────────────────────────────────────────────┐
│                         CLIENT LAYER                                   │
├────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  React Frontend (Port 3000)    |    Curl/Scripts    |    External API  │
│  - Dashboard                   |    - Health check   |    - Integrations
│  - Metrics View                |    - Monitoring     |                  │
│                                                                         │
└─────────────────────────┬──────────────────────────────────────────────┘
                          │
                          │ HTTP Requests
                          ▼
┌────────────────────────────────────────────────────────────────────────┐
│                    INTELLIGENT WORKFLOWS API                            │
│                      (Go, Port 8080/9090)                              │
├────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  HTTP Router (chi/v5)                                                  │
│  ├─ /health                                                            │
│  ├─ /ready                                                             │
│  ├─ /metrics ◄─────────────── [PROMETHEUS METRICS ENDPOINT]           │
│  │                                                                     │
│  └─ /api/v1/...                                                       │
│     ├─ Auth (JWT, API Keys)                                           │
│     ├─ Workflows (CRUD)                                               │
│     ├─ Executions (Run, Pause, Resume)                                │
│     ├─ Events (Ingest)                                                │
│     ├─ Approvals                                                      │
│     ├─ Analytics                                                      │
│     └─ AI (Chat, Interpret)                                           │
│                                                                         │
│  Middleware Stack:                                                     │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │ 1. Request ID                                                │    │
│  │ 2. Real IP                                                   │    │
│  │ 3. Logging (Zap) ◄─── Already implemented                   │    │
│  │ 4. Error Recovery                                            │    │
│  │ 5. Compression                                               │    │
│  │ 6. Security Headers                                          │    │
│  │ 7. Rate Limiting                                             │    │
│  │ 8. CORS                                                      │    │
│  │ 9. JWT/Auth Validation                                       │    │
│  │ 10. [METRICS] ◄─────── TO BE IMPLEMENTED                    │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  Core Services:                                                        │
│  ├─ Workflow Engine                                                   │
│  │  ├─ Executor (execute steps) ◄─── [Needs instrumentation]        │
│  │  ├─ Event Router (route events)                                   │
│  │  ├─ Action Executor (execute actions)                             │
│  │  ├─ Context Builder (enrich data)                                 │
│  │  └─ Evaluator (evaluate conditions)                               │
│  │                                                                    │
│  ├─ Approval Service ◄─── [Needs instrumentation]                   │
│  ├─ Notification Service ◄─── [Needs instrumentation]               │
│  ├─ Auth Service ◄─── [Needs instrumentation]                       │
│  ├─ AI Service ◄─── [Needs instrumentation]                         │
│  └─ Workflow Resumer ◄─── [Needs instrumentation]                   │
│                                                                         │
│  Background Workers:                                                   │
│  ├─ Approval Expiration Worker ◄─── [Needs instrumentation]         │
│  ├─ Workflow Resumer Worker ◄─── [Needs instrumentation]            │
│  └─ Timeout Enforcer Worker ◄─── [Needs instrumentation]            │
│                                                                         │
└─────────────────────────┬──────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│  PostgreSQL  │   │    Redis     │   │ Notification │
│   (Port      │   │   (Port      │   │   Service    │
│    5432)     │   │    6379)     │   │ (Email, etc) │
│              │   │              │   │              │
│ - Workflows  │   │ - Cache      │   │ - Email      │
│ - Executions │   │ - Pub/Sub    │   │ - Webhooks   │
│ - Events     │   │ - Locks      │   │              │
│ - Users      │   │              │   │              │
└──────────────┘   └──────────────┘   └──────────────┘
        ▲                 ▲
        │                 │
   [Exporter]        [Exporter]
        │                 │
        └─────────────────┘
                 │
                 ▼
      ┌────────────────────┐
      │   Node Exporter    │
      │  (Host Metrics)    │
      │   (Port 9100)      │
      └────────────────────┘
```

---

## Metrics Flow

```
Application Events
    │
    ├─ HTTP Request → Router Middleware ──────┐
    │                                          │
    ├─ Workflow Execution ─────────────────┐  │
    │                                      │  │
    ├─ Database Query ──────────────────┐ │  │
    │                                   │ │  │
    ├─ Redis Operation ───────────────┐ │ │  │
    │                                 │ │ │  │
    ├─ Background Worker Job ───────┐ │ │ │  │
    │                               │ │ │ │  │
    └─ Service Operation ────────┐  │ │ │ │  │
                                │  │ │ │ │  │
                                ▼  ▼ ▼ ▼ ▼  ▼
                           ┌─────────────────┐
                           │   Prometheus    │
                           │   Registry      │
                           │                 │
                           │ Counter         │
                           │ Gauge           │
                           │ Histogram       │
                           │ Summary         │
                           └────────┬────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    │                               │
                    ▼                               ▼
            ┌─────────────────┐          ┌──────────────────┐
            │  Prometheus     │          │  Application     │
            │  (Port 9091)    │          │  Scrapes Every   │
            │                 │          │  15 Seconds      │
            │ - TSDB Storage  │          │  (/metrics)      │
            │ - Alerting      │          │                  │
            │ - Querying      │          │  10.0.0.2:8080   │
            │ - Rules Engine  │          │  /metrics        │
            └────────┬────────┘          └──────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
    ┌─────────────┐         ┌──────────────┐
    │ Grafana     │         │ AlertManager │
    │ (Port 3000) │         │ (Port 9093)  │
    │             │         │              │
    │ Dashboards: │         │ - Routes     │
    │ - API Perf  │         │ - Enriches   │
    │ - Workflows │         │ - Notifies   │
    │ - Database  │         │              │
    └─────────────┘         └──────────────┘
```

---

## Instrumentation Points Priority Map

```
HIGH PRIORITY (Impact: ALERTS + DASHBOARDS)
═══════════════════════════════════════════════════════════
│
├─ HTTP Metrics (Router Middleware)
│  ├─ http_requests_total (Counter)
│  ├─ http_request_duration_seconds (Histogram)
│  ├─ http_request_size_bytes (Counter)
│  └─ http_response_size_bytes (Counter)
│
└─ Connection Metrics (Data Layer)
   ├─ db_connections_failed_total (Counter)
   └─ redis_connections_failed_total (Counter)
      └─ ALERT READY: DatabaseConnectionFailure
      └─ ALERT READY: RedisConnectionFailure


MEDIUM PRIORITY (Impact: BUSINESS KPIs)
═══════════════════════════════════════════════════════════
│
├─ Workflow Metrics (Executor)
│  ├─ workflow_executions_total (Counter)
│  ├─ workflow_execution_duration_seconds (Histogram)
│  ├─ workflow_step_duration_seconds (Histogram)
│  └─ workflow_execution_errors_total (Counter)
│
└─ Service Metrics
   ├─ approvals_total (Counter)
   ├─ approvals_duration_seconds (Histogram)
   ├─ notifications_sent_total (Counter)
   └─ ai_requests_total (Counter)


NICE-TO-HAVE (Impact: OPERATIONAL VISIBILITY)
═══════════════════════════════════════════════════════════
│
├─ Worker Metrics
│  ├─ worker_jobs_processed_total
│  ├─ worker_job_duration_seconds
│  └─ worker_errors_total
│
├─ Auth Metrics
│  ├─ auth_requests_total
│  ├─ auth_failures_total
│  └─ auth_token_validations_total
│
└─ Query Performance
   ├─ db_query_duration_seconds
   ├─ db_query_errors_total
   └─ redis_operation_duration_seconds
```

---

## Alert Chain

```
Metric Generated
    │
    ▼
Prometheus Scrapes (/metrics endpoint)
    │
    ▼
Rules Engine Evaluates (15s interval)
    │
    ├─ HighErrorRate: error_rate > 5% for 5m
    │  └─ Metric: http_requests_total{status=~"5.."}
    │     └─ Status: 🔴 WILL FAIL until instrumented
    │
    ├─ HighLatency: P95 > 1s for 5m
    │  └─ Metric: http_request_duration_seconds_bucket
    │     └─ Status: 🔴 WILL FAIL until instrumented
    │
    ├─ DatabaseConnectionFailure: rate > 0 for 2m
    │  └─ Metric: db_connections_failed_total
    │     └─ Status: 🔴 WILL FAIL until instrumented
    │
    ├─ RedisConnectionFailure: rate > 0 for 2m
    │  └─ Metric: redis_connections_failed_total
    │     └─ Status: 🔴 WILL FAIL until instrumented
    │
    └─ [Other alerts...]
        └─ Status: 🔴 WILL FAIL until instrumented
    │
    ▼
Alert Triggered (if condition met)
    │
    ▼
AlertManager Processes
    │
    ├─ Routes to correct receiver
    ├─ Enriches alert data
    ├─ Handles silencing
    └─ Sends notifications
        │
        ├─ Email
        ├─ Slack
        ├─ PagerDuty
        └─ Webhooks
```

---

## Data Flow Example: HTTP Request

```
1. Client sends request
   │
   ▼
2. Router receives request
   │
   ├─ Request ID middleware
   ├─ Security headers middleware
   ├─ Auth middleware
   │
   ├─ [NEW] METRICS MIDDLEWARE ◄── Records start_time
   │                              Records labels: method, path
   │
   ▼
3. Handler processes request
   │
   ├─ Queries database
   ├─ Checks Redis cache
   ├─ Calls external service
   │
   ▼
4. Handler returns response
   │
   ├─ [NEW] METRICS MIDDLEWARE ◄── Records:
   │                              - status_code
   │                              - response_time
   │                              - Increments http_requests_total
   │                              - Observes http_request_duration_seconds
   │
   ▼
5. Response sent to client
   │
   ▼
6. Prometheus scrapes /metrics
   │
   ├─ Reads all metrics
   ├─ Stores in TSDB
   ├─ Evaluates rules
   │
   ▼
7. Grafana queries metrics
   │
   ├─ Displays in dashboard
   ├─ Renders graphs
   │
   ▼
8. User sees metrics in UI
```

---

## Key Implementation Files

```
/home/user/intelligent-workflows/

PROMETHEUS CONFIG (Already exists)
├── monitoring/
│   ├── prometheus/
│   │   └── prometheus.yml ✅ (Configured, expects /metrics)
│   ├── alerts/
│   │   └── api_alerts.yml ✅ (Defined, missing metrics)
│   └── grafana/
│       ├── dashboards/
│       │   ├── api-overview.json ✅ (Expects metrics)
│       │   └── workflow-metrics.json ✅ (Expects metrics)
│       └── provisioning/ ✅ (Auto-loads dashboards)

APPLICATION (Needs instrumentation)
├── cmd/
│   └── api/
│       └── main.go (Initialize metrics here)
│
├── internal/
│   ├── api/
│   │   └── rest/
│   │       ├── router.go (Add /metrics endpoint)
│   │       ├── middleware/
│   │       │   ├── logger.go (Exists: logs)
│   │       │   └── metrics.go (NEW: prometheus metrics)
│   │       └── handlers/
│   │           ├── execution.go (Needs instrumentation)
│   │           ├── workflow.go (Needs instrumentation)
│   │           ├── auth.go (Needs instrumentation)
│   │           └── analytics.go (Uses metrics)
│   │
│   ├── engine/
│   │   ├── executor.go (Instrument Execute())
│   │   ├── event_router.go (Instrument routing)
│   │   └── action_executor.go (Instrument actions)
│   │
│   ├── services/
│   │   ├── approval_service.go (Instrument approvals)
│   │   ├── notification_service.go (Instrument notifications)
│   │   ├── auth_service.go (Instrument auth)
│   │   └── ai_service.go (Instrument AI calls)
│   │
│   ├── workers/
│   │   ├── approval_expiration_worker.go
│   │   ├── workflow_resumer_worker.go
│   │   └── timeout_enforcer_worker.go
│   │
│   └── repository/
│       └── postgres/ (Instrument query methods)
│
├── pkg/
│   ├── metrics/ (NEW PACKAGE)
│   │   └── metrics.go (Define all metrics here)
│   ├── database/ (Instrument pool)
│   └── logger/ (Already exists)
│
└── go.mod (Add prometheus/client_golang)
```

---

## Dependencies to Add

```go
// go.mod additions
require (
    github.com/prometheus/client_golang v1.19.0  // Prometheus client
    github.com/prometheus/client_model v0.5.0     // Data models
)
```

---

## Success Criteria

✅ All tasks complete when:

1. Prometheus targets shows "UP" for workflows-api
2. `/metrics` endpoint returns valid Prometheus format
3. Grafana dashboards display actual data
4. Alerts can be triggered (test with high load)
5. All 8+ critical metrics are collecting
6. No "no data" messages in Grafana
7. P95 latency visible in dashboard
8. Error rate trackable

---

## Timeline

```
Week 1:
├─ Day 1-2: HTTP metrics middleware + /metrics endpoint (2 hours)
└─ Day 3-5: Database/Redis/Workflow metrics (6 hours)

Week 2:
├─ Day 1-3: Service metrics (approval, notification, auth)
├─ Day 4: Worker metrics
└─ Day 5: Testing + Grafana tuning

Total: ~10-12 hours development
```

