# Database Decision: PostgreSQL vs MongoDB

## Executive Summary

**Recommendation: PostgreSQL as Primary Database with Redis for Caching**

For an intelligent workflows service, PostgreSQL is the superior choice as the primary database, with optional MongoDB for specific use cases (detailed execution traces, unstructured logs).

## Requirements Analysis

### Core Data Characteristics

1. **Workflows**: Structured, versioned, require ACID transactions
2. **Executions**: Time-series data, require querying and aggregation
3. **Rules**: Structured, frequently referenced, need transactional consistency
4. **Events**: High write volume, time-series, need efficient querying
5. **Approvals**: Structured, require strong consistency and transactions
6. **Context Data**: Mix of structured and semi-structured
7. **Audit Logs**: Time-series, write-heavy, compliance requirements

### System Requirements

- **Transactions**: Multi-step workflow operations need ACID guarantees
- **Relationships**: Complex relationships between workflows, executions, approvals
- **Querying**: Complex queries with joins, aggregations, filtering
- **Indexing**: Fast lookups on various fields
- **Consistency**: Strong consistency for approval workflows
- **Scalability**: Horizontal read scaling, vertical write scaling
- **JSON Support**: Dynamic workflow definitions and context data

## PostgreSQL Advantages

### 1. JSONB Support
PostgreSQL's JSONB type provides the best of both worlds:
- Store flexible workflow definitions as JSON
- Query JSON fields with SQL
- Index JSON properties for fast lookups
- Validate JSON schema constraints

```sql
-- Example: Query workflows by condition in JSON
SELECT * FROM workflows
WHERE definition @> '{"trigger": {"event": "order.created"}}'::jsonb;

-- Index on JSON field
CREATE INDEX idx_workflow_trigger ON workflows
USING GIN ((definition -> 'trigger'));

-- Complex JSON queries
SELECT
  workflow_id,
  definition->>'name' as name,
  jsonb_array_length(definition->'steps') as step_count
FROM workflows
WHERE definition->'trigger'->>'event' LIKE 'order.%';
```

### 2. ACID Transactions
Critical for workflow integrity:
```sql
BEGIN;
  -- Create workflow
  INSERT INTO workflows (workflow_id, definition, ...) VALUES (...);

  -- Create associated rules
  INSERT INTO rules (workflow_id, ...) VALUES (...);

  -- Create initial version
  INSERT INTO workflow_versions (...) VALUES (...);
COMMIT;
```

### 3. Referential Integrity
Foreign keys ensure data consistency:
```sql
CREATE TABLE workflow_executions (
    id UUID PRIMARY KEY,
    workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
    ...
);
```

### 4. Advanced Querying
Complex analytics queries:
```sql
-- Workflow success rate by trigger event
SELECT
    w.name,
    we.trigger_event,
    COUNT(*) as total_executions,
    COUNT(*) FILTER (WHERE we.status = 'completed') as successful,
    ROUND(100.0 * COUNT(*) FILTER (WHERE we.status = 'completed') / COUNT(*), 2) as success_rate,
    AVG(we.duration_ms) as avg_duration_ms
FROM workflows w
JOIN workflow_executions we ON w.id = we.workflow_id
WHERE we.started_at > NOW() - INTERVAL '7 days'
GROUP BY w.name, we.trigger_event
ORDER BY total_executions DESC;
```

### 5. Time-Series Optimization
Native partitioning for execution data:
```sql
-- Partition executions by month
CREATE TABLE workflow_executions (
    id UUID NOT NULL,
    started_at TIMESTAMP NOT NULL,
    ...
) PARTITION BY RANGE (started_at);

CREATE TABLE executions_2025_01 PARTITION OF workflow_executions
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE executions_2025_02 PARTITION OF workflow_executions
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
```

### 6. Full-Text Search
Built-in search capabilities:
```sql
-- Add search index
ALTER TABLE workflows ADD COLUMN search_vector tsvector;
CREATE INDEX idx_workflow_search ON workflows USING GIN(search_vector);

-- Search workflows
SELECT * FROM workflows
WHERE search_vector @@ to_tsquery('approval & order');
```

### 7. Row-Level Security
Multi-tenant isolation:
```sql
ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON workflows
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

### 8. Materialized Views
Pre-computed analytics:
```sql
CREATE MATERIALIZED VIEW workflow_performance AS
SELECT
    workflow_id,
    DATE_TRUNC('hour', started_at) as hour,
    COUNT(*) as execution_count,
    AVG(duration_ms) as avg_duration,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration
FROM workflow_executions
WHERE started_at > NOW() - INTERVAL '30 days'
GROUP BY workflow_id, DATE_TRUNC('hour', started_at);

CREATE UNIQUE INDEX ON workflow_performance (workflow_id, hour);
REFRESH MATERIALIZED VIEW CONCURRENTLY workflow_performance;
```

### 9. Go Ecosystem Support
Excellent Go libraries:
- **sqlc**: Type-safe SQL code generation
- **pgx**: High-performance PostgreSQL driver
- **golang-migrate**: Database migrations
- **pgxpool**: Connection pooling

```go
// Example with sqlc
// queries.sql
-- name: GetWorkflow :one
SELECT * FROM workflows WHERE id = $1;

// Generated Go code
workflow, err := queries.GetWorkflow(ctx, workflowID)
```

### 10. Extensions
Rich ecosystem:
- **pg_trgm**: Fuzzy string matching
- **uuid-ossp**: UUID generation
- **pgcrypto**: Encryption functions
- **pg_stat_statements**: Query performance tracking

## MongoDB Advantages (Limited for This Use Case)

### Where MongoDB Makes Sense

1. **Execution Traces** (Optional)
   - Deeply nested, variable structure
   - Large documents with array of trace events
   - Rarely queried for aggregation

2. **AI Agent Interactions** (Optional)
   - Unstructured conversation logs
   - Variable schema based on agent type
   - Primarily append-only

3. **Unstructured Logs** (Optional)
   - Debug information
   - Error stack traces
   - Performance profiling data

### MongoDB Disadvantages for This Use Case

1. **No Joins**: Complex relationships require application-level joins
2. **Weak Transactions**: Multi-document transactions have overhead
3. **Consistency**: Eventual consistency can cause issues for approvals
4. **Aggregation Complexity**: Complex analytics are harder than SQL
5. **Schema Validation**: Less robust than PostgreSQL constraints
6. **Index Management**: More manual tuning required

## Hybrid Approach (Recommended)

### PostgreSQL for Core Data
- Workflows
- Executions (summary data)
- Rules
- Events (recent + aggregated)
- Approvals
- Context cache
- Audit logs (compliance-critical)

### MongoDB for Ancillary Data (Optional)
- Detailed execution traces (debugging)
- AI agent interaction logs
- Unstructured performance profiling

### Redis for Operational Data
- Active workflow executions state
- Distributed locks
- Rate limiting
- Session data
- Message queue (Redis Streams)
- Real-time event routing

## Performance Considerations

### PostgreSQL Optimization

```sql
-- Indexes for common queries
CREATE INDEX idx_executions_workflow_status ON workflow_executions(workflow_id, status);
CREATE INDEX idx_executions_started ON workflow_executions(started_at DESC);
CREATE INDEX idx_events_type_received ON events(event_type, received_at DESC);
CREATE INDEX idx_approvals_status_expires ON approval_requests(status, expires_at);

-- Partial indexes for efficiency
CREATE INDEX idx_pending_approvals ON approval_requests(requested_at)
WHERE status = 'pending';

-- Expression indexes
CREATE INDEX idx_workflow_enabled ON workflows(id) WHERE enabled = true;

-- GIN indexes for JSONB
CREATE INDEX idx_workflow_definition ON workflows USING GIN(definition);
CREATE INDEX idx_execution_context ON workflow_executions USING GIN(context);
```

### Connection Pooling

```go
// PostgreSQL connection pooling with pgxpool
config, err := pgxpool.ParseConfig(databaseURL)
config.MaxConns = 50
config.MinConns = 10
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute

pool, err := pgxpool.ConnectConfig(context.Background(), config)
```

### Read Replicas

```
┌─────────────┐
│   Primary   │ ◄─── Writes
└─────────────┘
       │
       ├──────► Read Replica 1 (Analytics)
       ├──────► Read Replica 2 (API reads)
       └──────► Read Replica 3 (Monitoring)
```

## Data Volume Projections

### Assumptions
- 10,000 workflow executions per day
- 50 active workflows
- 30-day retention for detailed execution data
- 1-year retention for aggregated data

### Storage Estimates (PostgreSQL)

```
Workflows:           50 workflows × 10KB ≈ 500KB
Executions (30d):    300,000 × 5KB ≈ 1.5GB
Step Executions:     300,000 × 10 steps × 2KB ≈ 6GB
Events (30d):        1,000,000 × 3KB ≈ 3GB
Approvals (30d):     10,000 × 2KB ≈ 20MB
Context Cache:       ≈ 500MB
Audit Logs (365d):   ≈ 5GB
─────────────────────────────────────────────
Total (active data): ≈ 16GB

With indexes:        ≈ 25-30GB
```

### Scaling Thresholds

| Volume | Strategy |
|--------|----------|
| < 50GB | Single PostgreSQL instance |
| 50-500GB | Add read replicas, partition tables |
| 500GB-5TB | Sharding by tenant/date, archive old data |
| > 5TB | Consider time-series DB for executions |

## Backup and Recovery

### PostgreSQL
```bash
# Continuous archiving (PITR)
# postgresql.conf
archive_mode = on
archive_command = 'aws s3 cp %p s3://backups/wal/%f'

# Daily base backups
pg_basebackup -D /backup -Ft -z -P

# Point-in-time recovery
restore_command = 'aws s3 cp s3://backups/wal/%f %p'
recovery_target_time = '2025-11-05 10:00:00'
```

### Backup Strategy
- **Continuous WAL archiving**: Real-time
- **Base backups**: Daily at 2 AM
- **Retention**: 30 days point-in-time recovery
- **Cross-region replication**: For disaster recovery

## Migration Path

### Phase 1: Start with PostgreSQL
- Implement core system with PostgreSQL
- Use JSONB for flexible fields
- Monitor performance and query patterns

### Phase 2: Add Redis
- Implement caching layer
- Add message queue for events
- Use for distributed locks

### Phase 3: Evaluate MongoDB (Optional)
- Only if execution traces become problematic
- Only if AI agent logs require flexibility
- Measure actual benefit vs. complexity

## Cost Analysis (AWS Example)

### PostgreSQL (RDS)
```
db.m6g.xlarge (4 vCPU, 16GB RAM):    $290/month
Storage (100GB SSD):                  $11.50/month
Backup storage (100GB):               $9.50/month
Read Replica (optional):              $290/month
───────────────────────────────────────────────
Total (single instance):              ~$311/month
Total (with replica):                 ~$601/month
```

### MongoDB Atlas (For Comparison)
```
M30 (8GB RAM, 2 vCPU):               $315/month
Storage (100GB):                      Included
Backup:                               $12/month
Additional nodes (HA):                $315/month each
───────────────────────────────────────────────
Total (3-node replica set):           ~$957/month
```

### Redis (ElastiCache)
```
cache.m6g.large (2 vCPU, 6.38GB):    $108/month
───────────────────────────────────────────────
Recommended Stack Total:              $419-709/month
```

## Code Examples

### PostgreSQL with sqlc

```sql
-- queries/workflows.sql
-- name: CreateWorkflow :one
INSERT INTO workflows (
    workflow_id, version, name, definition, enabled
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetWorkflow :one
SELECT * FROM workflows WHERE id = $1;

-- name: ListActiveWorkflows :many
SELECT * FROM workflows
WHERE enabled = true
ORDER BY created_at DESC;

-- name: GetWorkflowExecutionStats :one
SELECT
    COUNT(*) as total_executions,
    COUNT(*) FILTER (WHERE status = 'completed') as successful,
    AVG(duration_ms) as avg_duration
FROM workflow_executions
WHERE workflow_id = $1
    AND started_at > NOW() - INTERVAL '7 days';
```

```go
// Go code using generated types
type WorkflowRepository struct {
    queries *db.Queries
}

func (r *WorkflowRepository) CreateWorkflow(ctx context.Context, w *models.Workflow) error {
    params := db.CreateWorkflowParams{
        WorkflowID: w.WorkflowID,
        Version:    w.Version,
        Name:       w.Name,
        Definition: w.Definition, // JSONB type
        Enabled:    w.Enabled,
    }

    workflow, err := r.queries.CreateWorkflow(ctx, params)
    if err != nil {
        return err
    }

    w.ID = workflow.ID
    w.CreatedAt = workflow.CreatedAt
    return nil
}

func (r *WorkflowRepository) GetExecutionStats(ctx context.Context, workflowID uuid.UUID) (*models.ExecutionStats, error) {
    stats, err := r.queries.GetWorkflowExecutionStats(ctx, workflowID)
    if err != nil {
        return nil, err
    }

    return &models.ExecutionStats{
        TotalExecutions: stats.TotalExecutions,
        Successful:      stats.Successful,
        AvgDuration:     stats.AvgDuration,
    }, nil
}
```

## Monitoring and Observability

### PostgreSQL Metrics to Track

```sql
-- Connection stats
SELECT count(*), state FROM pg_stat_activity GROUP BY state;

-- Slow queries
SELECT query, calls, mean_exec_time, max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC;

-- Table bloat
SELECT schemaname, tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### Prometheus Metrics

```go
var (
    queryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "workflow_db_query_duration_seconds",
            Help: "Database query duration in seconds",
        },
        []string{"query_name"},
    )

    connectionPoolSize = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "workflow_db_pool_size",
            Help: "Current size of database connection pool",
        },
    )
)

func (r *Repository) executeQuery(ctx context.Context, queryName string, fn func() error) error {
    start := time.Now()
    err := fn()
    duration := time.Since(start).Seconds()

    queryDuration.WithLabelValues(queryName).Observe(duration)
    return err
}
```

## Final Recommendation

### ✅ Use PostgreSQL

**Primary Reasons:**
1. **ACID transactions** essential for workflow integrity
2. **JSONB support** provides flexibility without sacrificing structure
3. **Complex querying** capabilities for analytics and monitoring
4. **Referential integrity** ensures data consistency
5. **Mature ecosystem** in Go
6. **Better cost/performance** ratio for this use case
7. **Single source of truth** simplifies architecture
8. **Time-series optimizations** via partitioning
9. **Full-text search** built-in
10. **Row-level security** for multi-tenancy

### 📦 Add Redis

**For:**
- Caching frequently accessed workflows
- Distributed locking during execution
- Message queue (Redis Streams) for events
- Rate limiting
- Session storage

### ❓ MongoDB Optional

**Only Consider If:**
- Execution trace documents become too large (>100KB each)
- AI agent interaction patterns truly need document structure
- You have MongoDB expertise on the team

**But first try:**
- PostgreSQL JSONB for these use cases
- Separate trace data into related tables
- Archive old traces to S3/object storage

## Implementation Checklist

- [ ] Set up PostgreSQL with connection pooling
- [ ] Implement database migrations with golang-migrate
- [ ] Generate type-safe code with sqlc
- [ ] Create core schema (workflows, executions, rules)
- [ ] Add JSONB indexes for workflow definitions
- [ ] Set up Redis for caching and message queue
- [ ] Implement partition strategy for time-series data
- [ ] Configure backup and recovery procedures
- [ ] Set up monitoring and alerting
- [ ] Load test with realistic workload
- [ ] Document query performance patterns
- [ ] Create runbook for common operations
