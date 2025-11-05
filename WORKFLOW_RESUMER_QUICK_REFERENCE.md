# Workflow Resumer - Quick Reference Guide

## TL;DR

The workflow execution system **does NOT currently support pause/resume**. There's only a placeholder implementation that logs but does nothing. This guide summarizes what needs to be built.

---

## Current Architecture

| Component | Status | Purpose |
|-----------|--------|---------|
| **WorkflowExecutor** | ✅ Complete | Main execution engine (sequential step execution) |
| **ActionExecutor** | ✅ Complete | Executes actions (notify, webhook, etc.) |
| **Evaluator** | ✅ Complete | Evaluates conditions |
| **ContextBuilder** | ✅ Complete | Builds and enriches context |
| **EventRouter** | ✅ Complete | Routes events to workflows |
| **ApprovalService** | ✅ Complete | Creates approval requests |
| **WorkflowResumer** | ❌ **Placeholder** | **THIS NEEDS IMPLEMENTATION** |

---

## Key Files to Know

### Core Execution
- `/internal/engine/executor.go` - Main executor (450+ lines)
- `/internal/engine/action_executor.go` - Action execution (370+ lines)
- `/internal/engine/event_router.go` - Event routing (220+ lines)

### Services & Persistence
- `/internal/services/workflow_resumer.go` - **NEEDS IMPLEMENTATION** (currently 43 lines, just logs)
- `/internal/services/approval_service.go` - Calls Resumer on approval (288 lines)
- `/internal/repository/postgres/execution_repository.go` - Database ops (305 lines)

### Data Models
- `/internal/models/execution.go` - Execution & StepExecution models
- `/internal/models/workflow.go` - Workflow definition models

### Database
- `/migrations/postgres/001_initial_schema.up.sql` - Current schema (has workflow_executions, step_executions, approval_requests tables)

---

## The Problem

### What Happens Today

```
1. Workflow hits approval requirement (block action)
2. ApprovalRequest created in database
3. Workflow completes with status=completed, result=blocked
4. Approver approves in UI
5. ApprovalService.ApproveRequest() calls workflowResumer.ResumeWorkflow()
6. NOTHING HAPPENS - it just logs and returns
7. Workflow stays completed forever
```

### What Should Happen

```
1. Workflow hits approval requirement
2. Workflow PAUSES with status=paused (not completed)
3. ApprovalRequest created
4. Approver approves
5. workflowResumer.ResumeWorkflow() actually resumes execution
6. Workflow continues from where it paused
7. Workflow completes with proper result
```

---

## What Needs to Be Built

### 1. Database Schema Changes

Add to `workflow_executions` table:
- `paused_at` - When workflow was paused
- `paused_reason` - Why (e.g., "approval_required")
- `paused_step_id` - Which step paused it
- `next_step_id` - Where to resume from
- `resume_data` - Context updates from approval/event

Add new state to `ExecutionStatus`:
- `ExecutionStatusPaused = "paused"`

### 2. WorkflowResumer Implementation

Current placeholder:
```go
func (w *WorkflowResumerImpl) ResumeWorkflow(ctx context.Context, 
    executionID uuid.UUID, approved bool) error {
    w.logger.Infof("Resuming workflow execution %s", executionID)
    return nil  // Does nothing!
}
```

Should be:
```go
func (w *WorkflowResumerImpl) ResumeWorkflow(ctx context.Context, 
    executionID uuid.UUID, approved bool, triggerData map[string]interface{}) error {
    // 1. Fetch paused execution from database
    // 2. Fetch workflow definition
    // 3. Call executor.ExecuteWithResumption()
    // 4. Update execution record in database
}

func (w *WorkflowResumerImpl) ResumeFromApproval(ctx context.Context,
    approval *models.ApprovalRequest) error {
    // Called by ApprovalService when approval is decided
}

func (w *WorkflowResumerImpl) ResumeManually(ctx context.Context,
    executionID uuid.UUID, userID uuid.UUID, reason string) error {
    // Manual resume via API
}
```

### 3. WorkflowExecutor Enhancements

Add new methods:
```go
func (we *WorkflowExecutor) ExecuteWithResumption(
    ctx context.Context,
    execution *models.WorkflowExecution,  // Resume this
    workflow *models.Workflow,
    resumeData map[string]interface{},    // From approval/event
) (*models.WorkflowExecution, error) {
    // Skip already-executed steps
    // Continue from next_step_id
    // Merge resumeData into context
}

func (we *WorkflowExecutor) executeFromStep(
    ctx context.Context,
    execution *models.WorkflowExecution,
    workflow *models.Workflow,
    startStepID string,
    execContext map[string]interface{},
) (models.ExecutionResult, error) {
    // Execute from specific step (not from start)
}
```

### 4. API Endpoints

Add to `/executions` route:
- `GET /api/v1/executions/{id}/status` - Get pause status
- `POST /api/v1/executions/{id}/pause` - Manual pause
- `POST /api/v1/executions/{id}/resume` - Manual resume

### 5. Integration Points

**ApprovalService** (already calls resumer):
```go
// In ApproveRequest():
if s.workflowResumer != nil {
    s.workflowResumer.ResumeFromApproval(ctx, approval)  // Currently does nothing
}
```

Change to actually implement `ResumeFromApproval()`.

### 6. State Tracking

When workflow hits approval point:
```go
if step.Action.Type == "block" && step.Action.Requires.Type == "approval" {
    execution.Status = models.ExecutionStatusPaused  // NEW
    execution.PausedAt = time.Now()                   // NEW
    execution.PausedReason = "approval_required"      // NEW
    execution.PausedStepID = step.ID                  // NEW
    execution.NextStepID = step.OnTrue                // NEW
    we.executionRepo.UpdateExecution(ctx, execution)
    
    // Create approval, then RETURN (don't continue)
    return ""  // Signal pause
}
```

---

## Implementation Checklist

### Phase 1: Foundation (Week 1-2)
- [ ] Create migration: add paused_* columns to workflow_executions
- [ ] Create migration: add ExecutionStatusPaused constant
- [ ] Update WorkflowExecution model with new fields
- [ ] Update database schema indices

### Phase 2: Core Logic (Week 3-4)
- [ ] Implement WorkflowResumer.ResumeWorkflow()
- [ ] Implement WorkflowResumer.ResumeFromApproval()
- [ ] Implement WorkflowResumer.ResumeManually()
- [ ] Implement WorkflowExecutor.ExecuteWithResumption()
- [ ] Implement WorkflowExecutor.executeFromStep()
- [ ] Add pause detection in executor (when to pause)
- [ ] Update ExecutionRepository for pause/resume queries
- [ ] Write unit tests

### Phase 3: API (Week 5)
- [ ] Add execution status endpoint
- [ ] Add pause endpoint
- [ ] Add resume endpoint
- [ ] Add permission checks (execution:pause, execution:resume)
- [ ] Write API tests

### Phase 4: Integration (Week 6)
- [ ] Update ApprovalService to call ResumeFromApproval()
- [ ] Update EventRouter for wait step resumption
- [ ] Create WorkflowResumeWorker for automatic resumption
- [ ] Write integration tests

### Phase 5: Polish (Week 7)
- [ ] Load testing
- [ ] Edge case handling
- [ ] Documentation
- [ ] Example workflows

---

## Critical Design Decisions

### Question 1: When to Pause?
- On `block` action with `requires: {type: "approval"}`
- Or also on `wait` steps?
- **Recommendation**: Start with approvals only, add wait later

### Question 2: When to Resume?
- Automatically after approval decision? (recommended)
- Or require manual resume?
- **Recommendation**: Auto-resume on approval

### Question 3: Resume Strategy?
- Start from next_step_id (skip executed steps)
- Or restart entire workflow from beginning?
- **Recommendation**: Skip executed steps, continue from next_step_id

### Question 4: Context Handling?
- Use context from pause point?
- Or rebuild context?
- **Recommendation**: Use paused context + merge approval decision

### Question 5: Idempotency?
- What if approval → resume → fails → retry?
- Need idempotency markers?
- **Recommendation**: Steps should be idempotent, mark step as completed before executing

---

## Testing Strategy

### Minimal Test Suite
```go
// Test pause detection
TestExecutorPauseOnApprovalRequired()

// Test context preservation
TestPausedContextPreserved()

// Test resumption
TestExecutorResumesFromPause()

// End-to-end
TestApprovalTriggersResume()
```

### Full Test Suite
- Unit tests for each resumer method
- Unit tests for pause detection
- Unit tests for step skipping
- Integration tests for approval → resume
- Integration tests for manual resume
- Edge cases: timeout, multiple pauses, concurrent resumes

---

## Key Code Locations to Modify

| File | Change | Why |
|------|--------|-----|
| `executor.go` | Add ExecuteWithResumption() | Execute from pause point |
| `executor.go` | Add pause detection | When to pause |
| `workflow_resumer.go` | Implement all methods | Core resumption logic |
| `approval_service.go` | Call ResumeFromApproval() | Trigger resume on approval |
| `execution_repository.go` | Add pause/resume queries | Fetch paused executions |
| `execution.go` | Add paused_* fields | Persistence |
| `execution.go` | Add ExecutionStatusPaused | State machine |
| `router.go` | Add resume endpoints | API access |
| `execution.go` handler | Add resume handlers | API implementation |

---

## Success Criteria

- [ ] Workflow pauses on approval requirement (status=paused)
- [ ] ApprovalService.ApproveRequest() triggers resume
- [ ] Execution continues from pause point
- [ ] Context preserved across pause/resume
- [ ] No data loss
- [ ] Approval → resume latency < 1 second
- [ ] All existing tests still pass
- [ ] New tests cover pause/resume scenarios

---

## Potential Pitfalls

1. **Context Mutation**: Context changes between pause and resume
   - Solution: Snapshot context at pause point

2. **Side Effects**: Steps that create records can't be undone
   - Solution: Mark steps as completed before executing

3. **Concurrency**: Race condition between pause and resume
   - Solution: Use database transactions

4. **Backward Compatibility**: Old workflows without pause support
   - Solution: Treat paused_at=NULL as completed (old behavior)

5. **Testing**: Hard to test pause/resume without full integration
   - Solution: Mock executor, write integration tests

---

## Related Documentation

- **Full Architecture Analysis**: `/WORKFLOW_EXECUTION_ANALYSIS.md`
- **Detailed Design**: `/WORKFLOW_RESUMER_ARCHITECTURE.md`
- **Existing Approval System**: See `ApprovalService` in `/internal/services/`
- **Database Schema**: `/migrations/postgres/001_initial_schema.up.sql`

