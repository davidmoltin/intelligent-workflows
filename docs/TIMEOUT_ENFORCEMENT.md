# Timeout Enforcement

## Overview

The Timeout Enforcement feature prevents workflows and individual steps from running indefinitely by automatically failing executions that exceed their configured timeout duration.

### Key Capabilities

- **Workflow-level timeouts**: Set a maximum execution time for the entire workflow
- **Step-level timeouts**: Configure individual step timeouts for fine-grained control
- **Automatic enforcement**: Background worker monitors and fails timed-out executions
- **Context-based enforcement**: Uses Go's context cancellation for immediate timeout detection
- **Flexible configuration**: Support for human-readable duration strings (e.g., "5m", "30s", "1h")

### Use Cases

1. **Prevent runaway workflows**: Stop workflows that hang due to external service failures
2. **SLA enforcement**: Ensure workflows complete within required time windows
3. **Resource management**: Free up resources from stuck executions
4. **Cost control**: Limit execution time for expensive operations
5. **User experience**: Provide predictable response times

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                  Workflow Executor                           │
│  - Applies workflow-level timeout via context               │
│  - Applies step-level timeout per step                      │
│  - Detects context.DeadlineExceeded errors                  │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│              WorkflowExecution Record                        │
│  - timeout_at: Absolute timestamp when execution times out  │
│  - timeout_duration: Duration in seconds (for reference)    │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│           TimeoutEnforcer Worker                             │
│  - Polls for timed-out executions every 1 minute            │
│  - Marks executions as failed when timeout_at < NOW()       │
│  - Updates execution with timeout error message             │
└─────────────────────────────────────────────────────────────┘
```

### Database Schema

Migration `004_timeout_enforcement.up.sql` adds:

```sql
ALTER TABLE workflow_executions
ADD COLUMN timeout_at TIMESTAMP,
ADD COLUMN timeout_duration INTEGER;

CREATE INDEX idx_executions_timeout ON workflow_executions(timeout_at)
WHERE timeout_at IS NOT NULL AND status IN ('running', 'waiting');
```

### Timeout Levels

**1. Workflow-level timeout (global)**
```json
{
  "definition": {
    "timeout": "5m",
    "steps": [...]
  }
}
```

**2. Step-level timeout (per step)**
```json
{
  "steps": [
    {
      "id": "fetch-data",
      "timeout": "30s",
      "type": "action"
    }
  ]
}
```

## Configuration

### Workflow Definition

Add a `timeout` field to the workflow definition:

```json
{
  "workflow_id": "order_processing",
  "version": "1.0.0",
  "definition": {
    "timeout": "5m",
    "trigger": {
      "type": "event",
      "event": "order.created"
    },
    "steps": [
      {
        "id": "validate",
        "timeout": "30s",
        "type": "condition",
        ...
      }
    ]
  }
}
```

### Supported Duration Formats

Timeouts use Go's `time.ParseDuration` format:

- `30s` - 30 seconds
- `2m` - 2 minutes
- `1h` - 1 hour
- `1h30m` - 1 hour 30 minutes
- `500ms` - 500 milliseconds

### Default Timeout

If no timeout is specified:
- **Workflow default**: 30 seconds (configurable in `WorkflowExecutor.defaultTimeout`)
- **Step default**: No timeout (inherits workflow timeout via context)

### Background Worker Configuration

In `cmd/api/main.go`:

```go
// Check interval: how often to poll for timed-out executions
timeoutEnforcerWorker := workers.NewTimeoutEnforcerWorker(
    executionRepo,
    log,
    1*time.Minute, // Check every 1 minute
)
timeoutEnforcerWorker.Start(workerCtx)
```

**Recommended settings:**
- **Check interval**: 1 minute (default)
- Lower for stricter enforcement, higher for reduced database load

## Implementation Details

### Context-Based Enforcement

The executor uses Go's context cancellation:

```go
timeout := parseTimeout(workflow.Definition.Timeout, we.defaultTimeout)
ctx, cancel := context.WithTimeout(ctx, timeout)
defer cancel()

// Execute workflow steps
result, err := we.executeSteps(ctx, execution, workflow, execContext)
if ctx.Err() == context.DeadlineExceeded {
    // Workflow timed out
}
```

**Benefits:**
- Immediate cancellation when timeout is reached
- Propagates to all nested operations
- Standard Go pattern for timeout handling

### Step-Level Enforcement

Each step can have its own timeout:

```go
stepCtx := ctx
if step.Timeout != "" {
    timeout := parseTimeout(step.Timeout, 0)
    stepCtx, cancel = context.WithTimeout(ctx, timeout)
    defer cancel()
}

nextStepID, result, err := executeStep(stepCtx, execution, step, execContext)
if stepCtx.Err() == context.DeadlineExceeded {
    return fmt.Errorf("step %s timed out", step.ID)
}
```

### Background Worker Logic

**Polling Cycle:**
```
1. Timer triggers (every 1 minute)
                ↓
2. Query: SELECT ... WHERE timeout_at < NOW() AND status IN ('running', 'waiting')
                ↓
3. For each timed-out execution:
   - Update status to 'failed'
   - Set error_message: "Workflow execution timed out after 5m"
   - Record completed_at and duration_ms
                ↓
4. Log metrics: failed=X, errors=Y
```

**SQL Query:**
```sql
SELECT id, workflow_id, execution_id, ...
FROM workflow_executions
WHERE timeout_at IS NOT NULL
  AND timeout_at < NOW()
  AND status IN ('running', 'waiting')
ORDER BY timeout_at ASC
LIMIT 100;
```

## Usage Examples

### Example 1: Simple Workflow Timeout

```json
{
  "definition": {
    "timeout": "2m",
    "trigger": {
      "type": "event",
      "event": "order.created"
    },
    "steps": [
      {
        "id": "process-order",
        "type": "action",
        "action": {"action": "execute"}
      }
    ]
  }
}
```

**Result**: Entire workflow must complete within 2 minutes or it will be failed.

### Example 2: Step-Specific Timeouts

```json
{
  "definition": {
    "timeout": "10m",
    "steps": [
      {
        "id": "fetch-data",
        "timeout": "1m",
        "type": "action",
        "action": {"action": "execute"},
        "execute": [
          {
            "type": "http_request",
            "url": "https://api.example.com/data"
          }
        ]
      },
      {
        "id": "process-data",
        "timeout": "5m",
        "type": "action",
        ...
      }
    ]
  }
}
```

**Result**:
- `fetch-data` must complete within 1 minute
- `process-data` must complete within 5 minutes
- Overall workflow must complete within 10 minutes

### Example 3: Timeout with Retry

```json
{
  "steps": [
    {
      "id": "external-api-call",
      "timeout": "30s",
      "type": "action",
      "action": {"action": "execute"},
      "execute": [
        {
          "type": "http_request",
          "url": "https://external-api.example.com/process"
        }
      ],
      "retry": {
        "max_attempts": 3,
        "backoff": "exponential",
        "retry_on": ["*"]
      }
    }
  ]
}
```

**Result**:
- Each retry attempt has a 30-second timeout
- If all 3 attempts timeout, the step fails
- Total time: up to 90 seconds (3 × 30s)

## Error Handling

### Timeout Errors

When a timeout occurs:

**1. Context-based detection (immediate):**
```
Error: step fetch-data timed out: context deadline exceeded
Status: failed
Result: failed
ErrorMessage: "step fetch-data timed out: context deadline exceeded"
```

**2. Background worker detection:**
```
Status: failed
Result: failed
ErrorMessage: "Workflow execution timed out after 5m0s"
CompletedAt: <timestamp>
DurationMs: 300000
```

### Timeout vs Other Failures

| Scenario | Status | Error Message |
|----------|--------|---------------|
| Step times out | `failed` | `"step {id} timed out: context deadline exceeded"` |
| Workflow times out | `failed` | `"Workflow execution timed out after {duration}"` |
| Background worker catches timeout | `failed` | `"Workflow execution timed out after {duration}"` |
| Step fails (non-timeout) | `failed` | Original error message |

## Monitoring and Observability

### Logs

**Executor Logs:**
```
INFO: Workflow timeout set to: 5m0s
INFO: Step fetch-data timeout set to: 30s
ERROR: Workflow execution timed out: exec_abc123
```

**Worker Logs:**
```
INFO: Starting timeout enforcer worker interval=1m0s
DEBUG: Checking for timed-out executions
INFO: Found 3 timed-out executions to process
INFO: Failing timed-out execution: exec_abc123 (timeout: 2024-11-05T12:00:00Z)
INFO: Successfully failed timed-out execution: exec_abc123
INFO: Timeout enforcement completed: failed=3, errors=0
```

### Metrics to Track

**Operational Metrics:**
- Timed-out executions count per hour
- Average timeout enforcement latency (time between timeout_at and worker detection)
- Timeout failure rate (timed-out / total executions)

**Performance Metrics:**
- Worker cycle duration
- GetTimedOutExecutions query latency
- Average workflow execution duration

**Example Prometheus Metrics:**
```
workflow_executions_timed_out_total
workflow_timeout_enforcer_cycle_duration_seconds
workflow_timeout_enforcement_latency_seconds
workflow_executions_timeout_rate
```

## Troubleshooting

### Problem: Workflow timing out too early

**Symptoms:**
- Workflows failing with timeout errors
- timeout_duration seems too short

**Solutions:**
1. Increase workflow-level timeout in definition
2. Check for slow external API calls
3. Add step-level timeouts to identify bottlenecks
4. Review execution traces to see which steps are slow

### Problem: Worker not detecting timeouts

**Symptoms:**
- Executions remain in `running` state past timeout_at
- Worker logs show "No timed-out executions found"

**Diagnosis:**
1. Check worker is running: `grep "Starting timeout enforcer worker" logs`
2. Verify timeout_at is set: `SELECT timeout_at FROM workflow_executions WHERE id=...`
3. Check execution status: `SELECT status FROM workflow_executions WHERE id=...`
4. Verify worker check interval

**Solutions:**
- Ensure worker started: Check main.go initialization
- Verify timeout is set in workflow definition
- Check database indexes exist
- Review worker logs for errors

### Problem: Context timeout not working

**Symptoms:**
- Steps run longer than configured timeout
- No "context deadline exceeded" errors

**Diagnosis:**
1. Verify step has `timeout` field in definition
2. Check parseTimeout is receiving the value
3. Ensure HTTP client respects context (for http_request actions)

**Solutions:**
- Add timeout to step definition
- Verify timeout format is valid (e.g., "30s", not "30")
- Ensure all operations respect the context

## Best Practices

### 1. Set Realistic Timeouts

✅ **DO:**
- Measure actual execution times first
- Add 50-100% buffer to average execution time
- Consider worst-case scenarios (network latency, API slowness)
- Use step-level timeouts for long workflows

❌ **DON'T:**
- Set arbitrary timeouts without measurement
- Use same timeout for all workflows
- Set timeouts shorter than external API SLAs

### 2. Combine with Retry Logic

✅ **DO:**
```json
{
  "timeout": "30s",
  "retry": {
    "max_attempts": 3,
    "backoff": "exponential"
  }
}
```

### 3. Monitor Timeout Rates

✅ **DO:**
- Track percentage of workflows timing out
- Alert if timeout rate > 5%
- Investigate root causes of timeouts
- Adjust timeouts based on metrics

### 4. Use Step Timeouts for Long Workflows

✅ **DO:**
```json
{
  "definition": {
    "timeout": "10m",
    "steps": [
      {"id": "fetch", "timeout": "1m"},
      {"id": "process", "timeout": "5m"},
      {"id": "store", "timeout": "2m"}
    ]
  }
}
```

❌ **DON'T:**
```json
{
  "definition": {
    "timeout": "10m",
    "steps": [
      {"id": "fetch"},
      {"id": "process"},
      {"id": "store"}
    ]
  }
}
```

### 5. Handle Timeouts Gracefully

✅ **DO:**
- Log timeout errors for debugging
- Consider adding compensation logic
- Notify relevant teams when timeouts occur
- Review and adjust timeouts regularly

## Performance Considerations

### Database

- **Indexed queries** on `timeout_at` and `status`
- **Batch processing** limits query load (100 executions per cycle)
- **Efficient updates** (single UPDATE per execution)

**Expected Query Performance:**
- `GetTimedOutExecutions`: < 100ms (for 100 results)
- `UpdateExecution`: < 20ms

### Worker

- **1-minute intervals** prevent database overload
- **Batch size of 100** balances throughput and latency
- **Graceful shutdown** ensures no data loss

**Expected Worker Performance:**
- Cycle duration: < 5 seconds (for 100 executions)
- Memory usage: < 50MB
- CPU usage: < 2% during processing

### Executor

- **No performance impact** when no timeout is set
- **Minimal overhead** for timeout enforcement (context creation)
- **Immediate cancellation** when timeout is reached

## Migration Guide

### Applying the Migration

```bash
psql -U dbuser -d workflows -f migrations/postgres/004_timeout_enforcement.up.sql
```

### Rollback (if needed)

```bash
psql -U dbuser -d workflows -f migrations/postgres/004_timeout_enforcement.down.sql
```

### Zero Downtime Deployment

1. Apply database migration (adds columns with defaults)
2. Deploy application with timeout enforcement
3. Verify worker is running
4. Monitor logs and metrics
5. Gradually enable timeouts on workflows

**No breaking changes** - existing workflows continue to work without timeouts.

## Security Considerations

### Timeout Values

- **Validate timeout values** to prevent extremely long timeouts
- **Maximum timeout**: Consider setting a global max (e.g., 1 hour)
- **Minimum timeout**: Prevent too-short timeouts (e.g., minimum 1 second)

### Denial of Service Prevention

- **Rate limiting**: Standard rate limits apply to API requests
- **Worker protection**: Batch size limit prevents worker overload
- **Database indexes**: Ensure queries are efficient

## Summary

The Timeout Enforcement feature provides:

✅ **Workflow-level timeouts** - Global execution limits
✅ **Step-level timeouts** - Fine-grained control
✅ **Context-based enforcement** - Immediate cancellation
✅ **Background worker** - Catches missed timeouts
✅ **Flexible configuration** - Human-readable durations
✅ **Production ready** - Tested, monitored, efficient

The feature is designed for reliability, performance, and ease of use in production environments.
