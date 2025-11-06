# Metrics and Monitoring

This document describes the Prometheus metrics exposed by the Intelligent Workflows API and how to use them for monitoring and observability.

## Accessing Metrics

The application exposes Prometheus-compatible metrics at the `/metrics` endpoint:

```bash
curl http://localhost:8080/metrics
```

## Available Metrics

### HTTP Metrics

Track HTTP request performance and status codes.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `http_requests_total` | Counter | method, path, status | Total number of HTTP requests |
| `http_request_duration_seconds` | Histogram | method, path | HTTP request latency distribution |
| `http_request_size_bytes` | Histogram | method, path | HTTP request body size |
| `http_response_size_bytes` | Histogram | method, path, status | HTTP response body size |

**Example Queries:**
```promql
# Request rate by endpoint
rate(http_requests_total[5m])

# P95 latency by endpoint
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate percentage
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100
```

### Workflow Execution Metrics

Monitor workflow execution performance and status.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `workflow_executions_total` | Counter | workflow_id, status | Total workflow executions by status |
| `workflow_execution_duration_seconds` | Histogram | workflow_id | Workflow execution time |
| `workflow_step_duration_seconds` | Histogram | workflow_id, step_name, status | Step execution time |
| `workflow_execution_errors_total` | Counter | workflow_id, error_type | Workflow execution errors |
| `active_workflow_executions` | Gauge | workflow_id | Currently running workflows |

**Status Values:** `executed`, `allowed`, `blocked`, `failed`, `paused`, `timeout`

**Error Types:** `context_build_error`, `execution_error`, `timeout`

**Example Queries:**
```promql
# Active workflows by ID
active_workflow_executions

# Workflow success rate
sum(rate(workflow_executions_total{status!="failed"}[5m])) / sum(rate(workflow_executions_total[5m]))

# Average workflow duration
rate(workflow_execution_duration_seconds_sum[5m]) / rate(workflow_execution_duration_seconds_count[5m])

# Workflows by status
sum by (status) (rate(workflow_executions_total[5m]))
```

### Database Metrics

Monitor PostgreSQL connection health and performance.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `db_connections_active` | Gauge | - | Active database connections |
| `db_connections_failed_total` | Counter | database | Failed connection attempts |
| `db_query_duration_seconds` | Histogram | query_type, table | Query execution time |
| `db_query_errors_total` | Counter | query_type, error_type | Database query errors |

**Example Queries:**
```promql
# Database connection failures
rate(db_connections_failed_total[5m])

# Slow queries (P95 > 100ms)
histogram_quantile(0.95, rate(db_query_duration_seconds_bucket[5m])) > 0.1
```

### Redis Metrics

Monitor Redis connection health and operation performance.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `redis_connections_active` | Gauge | - | Active Redis connections |
| `redis_connections_failed_total` | Counter | operation | Failed Redis operations |
| `redis_operations_duration_seconds` | Histogram | operation | Redis operation latency |
| `redis_operations_errors_total` | Counter | operation | Redis operation errors |

**Example Queries:**
```promql
# Redis connection health
redis_connections_active

# Redis error rate
rate(redis_operations_errors_total[5m])
```

### Business Logic Metrics

Track business-level operations.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `approvals_total` | Counter | status, workflow_id | Approval requests |
| `approvals_duration_seconds` | Histogram | workflow_id | Time to approval |
| `notifications_sent_total` | Counter | type, status | Notifications sent |
| `ai_requests_total` | Counter | model, status | AI API requests |
| `ai_request_duration_seconds` | Histogram | model | AI API latency |

**Example Queries:**
```promql
# Approval rate
rate(approvals_total{status="approved"}[5m])

# Average time to approval
rate(approvals_duration_seconds_sum[5m]) / rate(approvals_duration_seconds_count[5m])

# AI request success rate
sum(rate(ai_requests_total{status="success"}[5m])) / sum(rate(ai_requests_total[5m]))
```

### Worker Metrics

Monitor background worker performance.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `worker_jobs_processed_total` | Counter | worker_type, status | Jobs processed by workers |
| `worker_job_duration_seconds` | Histogram | worker_type | Worker job duration |
| `worker_errors_total` | Counter | worker_type | Worker errors |

**Worker Types:** `approval_expiration`, `workflow_resumer`, `timeout_enforcer`

**Example Queries:**
```promql
# Worker job rate
rate(worker_jobs_processed_total[5m])

# Worker error rate by type
sum by (worker_type) (rate(worker_errors_total[5m]))
```

### Authentication Metrics

Track authentication and authorization activity.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `auth_requests_total` | Counter | method, status | Authentication requests |
| `auth_failures_total` | Counter | reason | Authentication failures |
| `auth_token_validations_total` | Counter | valid | Token validation attempts |

**Auth Methods:** `login`, `register`, `refresh`, `api_key`

**Example Queries:**
```promql
# Login success rate
sum(rate(auth_requests_total{method="login",status="success"}[5m])) / sum(rate(auth_requests_total{method="login"}[5m]))

# Failed auth attempts by reason
sum by (reason) (rate(auth_failures_total[5m]))
```

## Pre-configured Alerts

The following alerts are configured in `/monitoring/alerts/api_alerts.yml`:

### HighErrorRate
Triggers when HTTP 5xx error rate exceeds 5% for 5 minutes.

### HighLatency
Triggers when P95 request latency exceeds 1 second for 5 minutes.

### DatabaseConnectionFailure
Triggers when database connection failures are detected.

### RedisConnectionFailure
Triggers when Redis connection failures are detected.

## Grafana Dashboards

Pre-configured dashboards are available at `http://localhost:3000` (default credentials: admin/admin):

- **API Overview**: HTTP metrics, error rates, latency
- **Workflow Metrics**: Execution stats, duration, success rates

## Prometheus Configuration

Metrics are scraped by Prometheus every 15 seconds. Configuration is in `/monitoring/prometheus/prometheus.yml`.

**Targets:**
- `workflows-api:9090` → Application metrics (exposed at `:8080/metrics`)
- `postgres-exporter:9187` → PostgreSQL database metrics
- `redis-exporter:9121` → Redis cache metrics

## Integration with Your Stack

### Docker Compose

```bash
# Start monitoring stack
docker-compose up -d

# View Prometheus targets
curl http://localhost:9091/targets

# Query metrics directly
curl http://localhost:9091/api/v1/query?query=http_requests_total
```

### Kubernetes

If deploying to Kubernetes, ensure your Service and ServiceMonitor are configured:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: workflows-api
  labels:
    app: workflows-api
spec:
  ports:
  - name: metrics
    port: 8080
    targetPort: 8080
  selector:
    app: workflows-api
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: workflows-api
spec:
  selector:
    matchLabels:
      app: workflows-api
  endpoints:
  - port: metrics
    path: /metrics
    interval: 15s
```

## Best Practices

1. **Alert on SLIs**: Focus alerts on Service Level Indicators (error rate, latency)
2. **Use rate() for counters**: Always use `rate()` when querying counter metrics
3. **Histogram quantiles**: Use `histogram_quantile()` for latency percentiles
4. **Cardinality**: Be mindful of label cardinality (avoid user IDs, timestamps in labels)
5. **Recording rules**: Create recording rules for frequently-used queries

## Troubleshooting

### Metrics not appearing

1. Check if `/metrics` endpoint is accessible:
   ```bash
   curl http://localhost:8080/metrics
   ```

2. Verify Prometheus can scrape the target:
   ```bash
   curl http://localhost:9091/targets
   ```

3. Check application logs for metric initialization errors

### High cardinality warnings

If you see high cardinality warnings, review metric labels. Avoid using:
- User IDs
- Execution IDs
- Timestamps
- High-variability values

Use aggregated or bucketed values instead.

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/instrumentation/)
- [Grafana Documentation](https://grafana.com/docs/)
