# Workflow Resumer - Quick Start Guide

## What is it?

The Workflow Resumer allows workflows to **pause** at any point and **resume** later, preserving complete execution state. Perfect for workflows requiring human approval, external callbacks, or manual intervention.

## Quick Example

```bash
# Pause a running workflow
curl -X POST http://localhost:8080/api/v1/executions/{id}/pause \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"reason": "Awaiting approval"}'

# Resume it later
curl -X POST http://localhost:8080/api/v1/executions/{id}/resume \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"resume_data": {"approved": true}}'
```

## Key Features

- 🎯 **Manual Control** - Pause/resume any workflow via API
- ✅ **Approval Integration** - Auto-resume after approval decisions
- 💾 **State Preservation** - Full context saved across pause/resume cycles
- 🤖 **Background Worker** - Automatic processing every 1 minute
- 🔒 **Secure** - Permission-based access control
- 📊 **Monitoring** - Track paused executions and metrics

## How It Works

### Approval-Driven Flow (Most Common)

```
1. Workflow executes → reaches approval step
2. Executor pauses execution (status: paused)
3. Approval request created
4. User approves via API
5. Approval service updates resume_data
6. Background worker auto-resumes
7. Workflow continues to completion
```

### Manual Control Flow

```
1. User pauses workflow via API
2. Execution saved with full state
3. User resumes later with custom data
4. Workflow continues from saved point
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/executions/{id}/pause` | POST | Pause running execution |
| `/api/v1/executions/{id}/resume` | POST | Resume paused execution |
| `/api/v1/executions/paused` | GET | List all paused executions |

## Common Use Cases

### 1. Approval Workflows

```json
{
  "steps": [
    {
      "id": "check-amount",
      "type": "condition",
      "condition": {"field": "amount", "operator": ">", "value": 10000},
      "on_true": "require-approval",
      "on_false": "auto-approve"
    },
    {
      "id": "require-approval",
      "type": "action",
      "action": {"type": "block", "reason": "Requires manager approval"}
    }
  ]
}
```

**Result:** High-value orders pause automatically, resume after approval.

### 2. Manual Intervention

```bash
# Pause for review
curl -X POST .../executions/{id}/pause \
  -d '{"reason": "Manual review required"}'

# Resume after review
curl -X POST .../executions/{id}/resume \
  -d '{"resume_data": {"reviewed_by": "alice@example.com"}}'
```

### 3. External System Callbacks

```bash
# Pause waiting for external response
curl -X POST .../executions/{id}/pause \
  -d '{"reason": "Awaiting payment gateway callback"}'

# Resume when callback received
curl -X POST .../executions/{id}/resume \
  -d '{"resume_data": {"payment_status": "completed"}}'
```

## Monitoring

### List Paused Executions

```bash
curl http://localhost:8080/api/v1/executions/paused?limit=20 \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "executions": [
    {
      "id": "...",
      "paused_at": "2025-11-05T10:00:00Z",
      "paused_reason": "Awaiting approval",
      "resume_count": 0
    }
  ],
  "count": 3
}
```

### Check Worker Status

```bash
# Check application logs
docker logs workflow-api | grep "workflow resumer worker"

# Expected output:
# INFO: Starting workflow resumer worker interval=1m batch_size=50
# INFO: Paused executions processed: resumed=1, skipped=0, errors=0
```

## Configuration

### Background Worker

Default configuration in `cmd/api/main.go`:

```go
resumerWorker := workers.NewWorkflowResumerWorker(
    workflowResumer,
    log,
    1*time.Minute,  // Check interval
)
```

**Adjust as needed:**
- Check interval: 30 seconds to 5 minutes
- Batch size: Default 50 (hardcoded, see worker code)

### Pause Duration Limits

Default: **7 days maximum** pause duration

Configured in `internal/services/workflow_resumer.go`:
```go
maxPauseDuration := 7 * 24 * time.Hour
```

## Permissions Required

| Operation | Permission |
|-----------|------------|
| Pause execution | `execution:cancel` |
| Resume execution | `execution:cancel` |
| List paused executions | `execution:read` |

## Database Schema

Migration: `migrations/postgres/003_workflow_resumer.up.sql`

**New columns on `workflow_executions`:**
- `paused_at` - Pause timestamp
- `paused_reason` - Why paused
- `paused_step_id` - Which step
- `next_step_id` - Resume point
- `resume_data` - Custom data (JSONB)
- `resume_count` - Resume counter
- `last_resumed_at` - Last resume timestamp

## Troubleshooting

### Execution Won't Resume

**Check 1:** Is it actually paused?
```sql
SELECT status, paused_at, paused_reason
FROM workflow_executions
WHERE id = '...';
```

**Check 2:** Has it been paused too long?
```sql
-- If paused > 7 days, manual intervention needed
SELECT id, paused_at,
       NOW() - paused_at AS pause_duration
FROM workflow_executions
WHERE status = 'paused';
```

**Check 3:** Check worker logs
```bash
docker logs workflow-api | grep "Failed to resume"
```

### Worker Not Auto-Resuming

**Verify approval decision format:**
```sql
SELECT resume_data
FROM workflow_executions
WHERE id = '...';

-- Should contain: {"approved": true} or {"approved": false}
-- NOT: {"approved": "true"} (string)
```

**Check worker is running:**
```bash
docker logs workflow-api | grep "Starting workflow resumer worker"
```

### High Pause/Resume Latency

**Check database performance:**
```sql
EXPLAIN ANALYZE
SELECT * FROM workflow_executions
WHERE status = 'paused'
ORDER BY paused_at ASC
LIMIT 50;

-- Should use: idx_executions_paused_at
```

**Monitor worker cycle duration:**
```bash
# Should complete in < 5 seconds
docker logs workflow-api | grep "Paused executions processed"
```

## Best Practices

✅ **DO:**
- Provide meaningful pause reasons
- Monitor paused executions daily
- Set up alerts for executions paused > 24 hours
- Use consistent resume_data structure
- Test pause/resume in staging first

❌ **DON'T:**
- Leave executions paused indefinitely
- Pause without checking current status
- Store sensitive data in resume_data
- Ignore worker error logs
- Set worker interval < 30 seconds

## Performance

**Expected Performance:**
- Pause operation: < 100ms
- Resume operation: < 500ms (includes workflow restart)
- List paused (50 results): < 50ms
- Worker cycle (50 executions): < 5 seconds

**Scaling Considerations:**
- Worker processes 50 executions per minute
- Can handle 3,000+ paused executions efficiently
- Database indexes ensure query performance
- No impact on non-paused executions

## Full Documentation

For complete details, see:
- **Full Guide:** [docs/WORKFLOW_RESUMER.md](WORKFLOW_RESUMER.md)
- **API Reference:** [docs/WORKFLOW_RESUMER.md#api-reference](WORKFLOW_RESUMER.md#api-reference)
- **Architecture:** [docs/WORKFLOW_RESUMER.md#architecture](WORKFLOW_RESUMER.md#architecture)

## Support

Need help?
- Check logs: `docker logs workflow-api | grep -A 10 "resumer"`
- Review [troubleshooting guide](WORKFLOW_RESUMER.md#troubleshooting)
- Open issue: https://github.com/davidmoltin/intelligent-workflows/issues

## Summary

The Workflow Resumer is a production-ready feature that enables:

✅ Pause any workflow at any point
✅ Resume with preserved state
✅ Automatic approval-driven flows
✅ Complete API control
✅ Background processing
✅ Full monitoring and observability

**Status:** ✅ Production Ready
