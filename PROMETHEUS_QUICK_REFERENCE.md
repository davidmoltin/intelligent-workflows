# Prometheus Instrumentation - Quick Reference

## Current Status
- **Prometheus:** ✅ Configured & running (port 9091)
- **Grafana:** ✅ Configured & running (port 3000)
- **Alert Rules:** ✅ Defined but expect metrics that don't exist yet
- **Application Metrics:** ❌ **NOT IMPLEMENTED**

---

## What Needs to Be Done

### 1. Add Prometheus Client to Go
```go
// go.mod - Add dependency
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/promhttp
```

### 2. Create Metrics Package
**Location:** `/pkg/metrics/metrics.go`

Define all metrics once:
```go
type Metrics struct {
    // HTTP
    HTTPRequestsTotal   prometheus.CounterVec
    HTTPDuration        prometheus.HistogramVec
    
    // Workflows
    WorkflowExecutionsTotal prometheus.CounterVec
    WorkflowDuration    prometheus.HistogramVec
    
    // Database
    DBConnectionsFailed prometheus.CounterVec
    
    // Redis
    RedisConnectionsFailed prometheus.CounterVec
}

func New() (*Metrics, error) {
    // Initialize all metrics here
    // Register with prometheus.DefaultRegisterer
}
```

### 3. Update Router (`/internal/api/rest/router.go`)
```go
// In NewRouter():
router.Handle("/metrics", promhttp.HandlerFor(
    prometheus.DefaultGatherer,
    promhttp.HandlerOpts{},
))

// Add metrics middleware
router.Use(metricsMiddleware(metrics))
```

### 4. Add Middleware (`/internal/api/rest/middleware/metrics.go`)
```go
func Metrics(m *metrics.Metrics) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
            
            defer func() {
                m.HTTPRequestsTotal.WithLabelValues(
                    r.Method,
                    r.URL.Path,
                    fmt.Sprintf("%d", ww.Status()),
                ).Inc()
                
                m.HTTPDuration.WithLabelValues(
                    r.Method,
                    r.URL.Path,
                ).Observe(time.Since(start).Seconds())
            }()
            
            next.ServeHTTP(ww, r)
        })
    }
}
```

### 5. Instrument Executor (`/internal/engine/executor.go`)
```go
// At start of Execute():
start := time.Now()

// At end or error:
metrics.WorkflowExecutionsTotal.WithLabelValues(
    workflow.ID,
    status,
).Inc()

metrics.WorkflowDuration.WithLabelValues(
    workflow.ID,
).Observe(time.Since(start).Seconds())
```

---

## Critical Metrics to Implement

| Metric | Type | Where | Priority |
|--------|------|-------|----------|
| `http_requests_total` | Counter | Router middleware | 🔴 CRITICAL |
| `http_request_duration_seconds` | Histogram | Router middleware | 🔴 CRITICAL |
| `workflow_executions_total` | Counter | Executor | 🟠 HIGH |
| `workflow_execution_duration_seconds` | Histogram | Executor | 🟠 HIGH |
| `db_connections_failed_total` | Counter | Database init | 🟠 HIGH |
| `redis_connections_failed_total` | Counter | Redis init | 🟠 HIGH |
| `approvals_total` | Counter | Approval service | 🟡 MEDIUM |
| `notifications_sent_total` | Counter | Notification service | 🟡 MEDIUM |

---

## Docker Compose Testing

```bash
# Start stack
docker-compose up -d

# Check services
curl http://localhost:8080/health     # API
curl http://localhost:9091/targets    # Prometheus targets
http://localhost:3000                 # Grafana (admin/admin)

# Query metrics
curl http://localhost:8080/metrics    # Your app metrics
```

---

## Verification Checklist

After implementing:

- [ ] `/metrics` endpoint returns Prometheus format
- [ ] `http_requests_total` counter increments with each request
- [ ] `http_request_duration_seconds` histogram records latencies
- [ ] Prometheus targets shows "UP" for workflows-api
- [ ] Grafana dashboard "api-overview" shows data
- [ ] Alerts no longer show "no data"
- [ ] Database/Redis connection metrics working

---

## Key Files to Modify

1. **go.mod** - Add prometheus/client_golang
2. **/cmd/api/main.go** - Initialize metrics
3. **/internal/api/rest/router.go** - Add /metrics endpoint, middleware
4. **/internal/api/rest/middleware/metrics.go** - NEW FILE
5. **/pkg/metrics/metrics.go** - NEW FILE (metric definitions)
6. **/internal/engine/executor.go** - Add instrumentation
7. **/internal/services/approval_service.go** - Add instrumentation
8. **/internal/services/notification_service.go** - Add instrumentation

---

## Prometheus Alert Dependencies

These alerts **already exist** and expect these metrics:

```yaml
HighErrorRate:
  Requires: http_requests_total{status=~"5.."}

HighLatency:
  Requires: http_request_duration_seconds_bucket

DatabaseConnectionFailure:
  Requires: db_connections_failed_total

RedisConnectionFailure:
  Requires: redis_connections_failed_total
```

---

## References

- Prometheus Go Client: https://github.com/prometheus/client_golang
- Best Practices: https://prometheus.io/docs/practices/instrumenting/
- Metric Types: https://prometheus.io/docs/concepts/metric_types/
- Histogram Config: https://prometheus.io/docs/concepts/metric_types/#histogram

---

## Implementation Timeline

- **Phase 1 (Critical):** HTTP metrics + /metrics endpoint
  - Estimated: 1-2 hours
  - Impact: Alerts + Dashboard visibility

- **Phase 2 (High Priority):** Workflow + DB metrics
  - Estimated: 2-3 hours
  - Impact: Business KPI tracking

- **Phase 3 (Nice to Have):** Service metrics
  - Estimated: 2-3 hours
  - Impact: Operational visibility

**Total Time:** ~5-8 hours of development

---

## Questions?

Refer to: `/PROMETHEUS_INSTRUMENTATION_ANALYSIS.md` for detailed analysis
