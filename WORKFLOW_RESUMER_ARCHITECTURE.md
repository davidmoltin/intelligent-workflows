# Workflow Resumer Feature - Architectural Design

## Overview

This document outlines the architectural design for implementing a complete workflow pause/resume system that will enable workflows to:
1. Pause at approval points
2. Persist execution state
3. Resume based on approval decisions
4. Continue execution from the pause point

---

## 1. Current State Assessment

### What Works
- Workflows execute sequentially with full state persistence
- Step execution history is tracked in `step_executions` table
- Execution context is stored in `workflow_executions.context`
- Approval requests link back to executions via `execution_id`
- Background worker infrastructure exists (ApprovalExpirationWorker)

### What's Missing
- No "paused" execution state
- No resumption point tracking (next_step_id)
- Placeholder `ResumeWorkflow()` that does nothing
- No mechanism to continue from where execution paused
- No API endpoint to trigger resume

---

## 2. Proposed Data Model Changes

### 2.1 Extend `workflow_executions` Table

```sql
ALTER TABLE workflow_executions ADD COLUMN (
    paused_at TIMESTAMP,                    -- When paused
    paused_reason VARCHAR(255),             -- Why paused (e.g., "approval_required")
    paused_step_id VARCHAR(255),            -- Which step caused pause
    next_step_id VARCHAR(255),              -- Which step to resume from
    resume_data JSONB,                      -- Context updates from pause trigger
    resume_attempted_at TIMESTAMP,          -- When resume was attempted
    resume_error_message TEXT               -- Error if resume failed
);

-- Indices for efficient querying
CREATE INDEX idx_executions_paused_at ON workflow_executions(paused_at DESC);
CREATE INDEX idx_executions_paused_reason ON workflow_executions(paused_reason);
CREATE INDEX idx_executions_next_step ON workflow_executions(next_step_id);
```

### 2.2 New `workflow_pausepoints` Table (Optional - for tracking pause logic)

```sql
CREATE TABLE workflow_pausepoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
    step_id VARCHAR(255) NOT NULL,
    pause_trigger VARCHAR(50),              -- approval, wait, manual
    resumption_behavior VARCHAR(50),        -- continue, branch_on_decision, repeat
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(workflow_id, step_id)
);
```

### 2.3 New `execution_resume_requests` Table (For audit trail)

```sql
CREATE TABLE execution_resume_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID REFERENCES workflow_executions(id) ON DELETE CASCADE,
    triggered_by VARCHAR(100),              -- "approval", "manual", "event"
    triggered_by_id UUID,                   -- user_id, approval_id, etc.
    trigger_data JSONB,                     -- approval decision, event, etc.
    requested_at TIMESTAMP DEFAULT NOW(),
    executed_at TIMESTAMP,
    error_message TEXT
);
```

---

## 3. Execution State Machine (Revised)

```
                    ┌─────────────────────────────────┐
                    │         pending                 │
                    │   (initial state, rare)         │
                    └─────────────────────────────────┘
                                 ↓
                    ┌─────────────────────────────────┐
                    │         running                 │
                    │   (actively executing steps)    │
                    └─────────────────────────────────┘
         ┌──────────────┬────────────────┬──────────────┐
         ↓              ↓                ↓              ↓
    completed      failed          blocked       ⭐ PAUSED (NEW)
    (result set)   (error)         (action:     (awaiting decision)
                                    block)      └─→ Waiting for:
                                                 - Approval
                                                 - Event
                                                 - Manual trigger
                                                      ↓
                                                  Resume requested
                                                      ↓
                                               ┌─────────────────┐
                                               │    resuming     │
                                               │ (restart from   │
                                               │  next_step_id)  │
                                               └─────────────────┘
                                                      ↓
                                           (returns to completed/failed)
```

---

## 4. Core Components to Implement

### 4.1 Enhanced WorkflowExecutor

**Changes to `internal/engine/executor.go`**:

```go
// New field to support resumption
type WorkflowExecutor struct {
    evaluator       *Evaluator
    contextBuilder  *ContextBuilder
    actionExecutor  *ActionExecutor
    executionRepo   ExecutionRepository
    logger          *logger.Logger
    maxRetries      int
    defaultTimeout  time.Duration
}

// New method: Execute with resumption support
func (we *WorkflowExecutor) ExecuteWithResumption(
    ctx context.Context,
    execution *models.WorkflowExecution,  // Resume this execution
    workflow *models.Workflow,
    resumeData map[string]interface{},    // Data from approval/event
) (*models.WorkflowExecution, error) {
    // Logic:
    // 1. Fetch execution and step history
    // 2. Identify next_step_id from execution.next_step_id
    // 3. Skip already-executed steps
    // 4. Continue from next_step_id
    // 5. Merge resumeData into context
}

// Helper: Find resume point in workflow
func (we *WorkflowExecutor) getResumePoint(
    workflow *models.Workflow,
    nextStepID string,
) (*models.Step, int, error)

// Helper: Skip to resume point
func (we *WorkflowExecutor) executeFromStep(
    ctx context.Context,
    execution *models.WorkflowExecution,
    workflow *models.Workflow,
    startStepID string,
    execContext map[string]interface{},
) (models.ExecutionResult, error)
```

### 4.2 Implemented WorkflowResumer Service

**Enhance `internal/services/workflow_resumer.go`**:

```go
type WorkflowResumerImpl struct {
    logger           *logger.Logger
    executor         *engine.WorkflowExecutor  // Add reference
    executionRepo    engine.ExecutionRepository
    workflowRepo     engine.WorkflowRepository  // Add reference
}

// Implement: Resume workflow from paused state
func (w *WorkflowResumerImpl) ResumeWorkflow(
    ctx context.Context,
    executionID uuid.UUID,
    approved bool,
    triggerData map[string]interface{},
) error {
    // Logic:
    // 1. Fetch paused execution
    // 2. Validate it's in paused state
    // 3. Get workflow definition
    // 4. Build resume data with approval decision
    // 5. Call ExecuteWithResumption()
    // 6. Update execution record
}

// New method: Resume from approval
func (w *WorkflowResumerImpl) ResumeFromApproval(
    ctx context.Context,
    approval *models.ApprovalRequest,
) error {
    // Called by ApprovalService when approval is decided
    // Includes approval decision in resume data
}

// New method: Resume from event
func (w *WorkflowResumerImpl) ResumeFromEvent(
    ctx context.Context,
    executionID uuid.UUID,
    event *models.Event,
) error {
    // For wait steps: trigger resume when expected event arrives
}

// New method: Manual resume
func (w *WorkflowResumerImpl) ResumeManually(
    ctx context.Context,
    executionID uuid.UUID,
    userID uuid.UUID,
    reason string,
) error {
    // Allow operator to manually resume paused workflow
}
```

### 4.3 Pause Point Handler in Executor

**New logic in `executor.go` `executeStep()` method**:

```go
// When executing an action step with "block" action that has approval:
if step.Action.Type == "block" && step.Action.Requires != nil {
    // Check if approval requirement
    if step.Action.Requires.Type == "approval" {
        // 1. Create approval request
        // 2. Mark execution as paused
        execution.Status = models.ExecutionStatusPaused
        execution.PausedAt = time.Now()
        execution.PausedReason = "approval_required"
        execution.PausedStepID = step.ID
        execution.NextStepID = step.OnTrue  // Resume to on_true if approved
        
        // 3. Save to database
        we.executionRepo.UpdateExecution(ctx, execution)
        
        // 4. Create approval request
        // 5. Return (don't complete execution)
        return ""  // No next step - paused
    }
}
```

### 4.4 REST API Endpoints

**New endpoints in `internal/api/rest/router.go`**:

```go
// Existing executions endpoints
router.Route("/executions", func(router chi.Router) {
    router.With(...).Get("/", ...)              // List
    router.With(...).Get("/{id}", ...)          // Get
    router.With(...).Get("/{id}/trace", ...)    // Trace
    
    // NEW ENDPOINTS
    router.With(...).Get("/{id}/status", ...)   // Get current status
    router.With(...).Post("/{id}/pause", ...)   // Manual pause
    router.With(...).Post("/{id}/resume", ...)  // Manual resume
    router.With(...).Get("/{id}/resume-options", ...) // List resume options
})
```

**Handler implementation in `internal/api/rest/handlers/execution.go`**:

```go
// GET /api/v1/executions/{id}/status
func (h *ExecutionHandler) GetExecutionStatus(w http.ResponseWriter, r *http.Request) {
    // Return: status, paused_at, paused_reason, next_step_id, etc.
}

// POST /api/v1/executions/{id}/pause
func (h *ExecutionHandler) PauseExecution(w http.ResponseWriter, r *http.Request) {
    // Manual pause: update status, set paused_at, paused_reason
    // Requires permission: execution:pause
}

// POST /api/v1/executions/{id}/resume
func (h *ExecutionHandler) ResumeExecution(w http.ResponseWriter, r *http.Request) {
    // Resume: call workflowResumer.ResumeManually()
    // Request body: { "reason": "...", "metadata": {...} }
    // Requires permission: execution:resume
}

// GET /api/v1/executions/{id}/resume-options
func (h *ExecutionHandler) GetResumeOptions(w http.ResponseWriter, r *http.Request) {
    // Return available resume actions based on pause point
    // E.g., { "actions": ["approve", "reject", "escalate"] }
}
```

---

## 5. Integration Points

### 5.1 ApprovalService Integration

**Modify `internal/services/approval_service.go`**:

```go
// In ApproveRequest() method:
if s.workflowResumer != nil {
    // Pass approval data to resumer
    if err := s.workflowResumer.ResumeFromApproval(ctx, approval); err != nil {
        s.logger.Errorf("Failed to resume workflow after approval: %v", err)
        // Don't fail approval, just log
    }
}

// In RejectRequest() method:
if s.workflowResumer != nil {
    // Resume with rejection decision
    if err := s.workflowResumer.ResumeFromApproval(ctx, approval); err != nil {
        s.logger.Errorf("Failed to resume workflow after rejection: %v", err)
    }
}
```

### 5.2 EventRouter Integration

**Modify `internal/engine/event_router.go`**:

```go
// New method: Handle event that resumes waiting execution
func (er *EventRouter) RouteEventToWaitingExecutions(
    ctx context.Context,
    eventType string,
    payload map[string]interface{},
) error {
    // Find executions paused on wait steps expecting this event
    // Call workflowResumer.ResumeFromEvent() for each
}
```

### 5.3 Background Worker

**New worker: `internal/workers/workflow_resume_worker.go`**:

```go
type WorkflowResumeWorker struct {
    logger        *logger.Logger
    resumer       *services.WorkflowResumerImpl
    checkInterval time.Duration
    stopCh        chan struct{}
    doneCh        chan struct{}
}

func (w *WorkflowResumeWorker) Start(ctx context.Context) {
    go w.run(ctx)  // Periodically check for resumable executions
}

// Check for:
// - Paused executions with completed approvals
// - Paused executions with timeout reached
// - Manually resumed executions
```

---

## 6. Implementation Phases

### Phase 1: Foundation (Week 1-2)
- [ ] Add new columns to workflow_executions table
- [ ] Create execution_resume_requests table
- [ ] Add ExecutionStatus = "paused" constant
- [ ] Add paused fields to WorkflowExecution model
- [ ] Write database migrations

### Phase 2: Core Resumer Logic (Week 3-4)
- [ ] Implement WorkflowResumerImpl.ResumeWorkflow()
- [ ] Implement WorkflowExecutor.ExecuteWithResumption()
- [ ] Add pause point detection in executor
- [ ] Implement resume from approval (ApprovalService integration)
- [ ] Add unit tests

### Phase 3: API & Integration (Week 5)
- [ ] Add resume endpoints to ExecutionHandler
- [ ] Integrate with ApprovalService
- [ ] Add REST endpoint tests
- [ ] Update ExecutionRepository with pause/resume queries

### Phase 4: Advanced Features (Week 6-7)
- [ ] Implement wait step resumption (EventRouter integration)
- [ ] Add resume worker for automatic resumption
- [ ] Implement timeout handling for paused executions
- [ ] Add audit trail (execution_resume_requests table)

### Phase 5: Testing & Polish (Week 8)
- [ ] Integration tests: approval → resume flow
- [ ] Integration tests: event → resume flow
- [ ] Load testing for resume operations
- [ ] Documentation and examples

---

## 7. Code Structure

```
internal/
├── engine/
│   ├── executor.go (MODIFIED)
│   │   ├── Execute()           (existing)
│   │   ├── ExecuteWithResumption()  (NEW)
│   │   ├── executeFromStep()   (NEW)
│   │   └── getResumePoint()    (NEW)
│   └── event_router.go (MODIFIED)
│       └── RouteEventToWaitingExecutions() (NEW)
│
├── services/
│   ├── workflow_resumer.go (IMPLEMENTED - was placeholder)
│   │   ├── ResumeWorkflow()        (implement)
│   │   ├── ResumeFromApproval()    (NEW)
│   │   ├── ResumeFromEvent()       (NEW)
│   │   └── ResumeManually()        (NEW)
│   └── approval_service.go (MODIFIED)
│       ├── ApproveRequest()  (add resumer call)
│       └── RejectRequest()   (add resumer call)
│
├── repository/postgres/
│   └── execution_repository.go (ENHANCED)
│       ├── UpdateExecutionPauseState()    (NEW)
│       ├── UpdateExecutionResumeState()   (NEW)
│       ├── GetPausedExecutions()          (NEW)
│       └── GetResumableExecutions()       (NEW)
│
├── workers/
│   ├── approval_expiration_worker.go (existing)
│   └── workflow_resume_worker.go (NEW)
│
├── models/
│   ├── execution.go (MODIFIED)
│   │   └── Add PausedAt, PausedReason, NextStepID fields
│   └── ...
│
└── api/rest/
    ├── handlers/
    │   └── execution.go (ENHANCED)
    │       ├── GetExecutionStatus()       (NEW)
    │       ├── PauseExecution()           (NEW)
    │       ├── ResumeExecution()          (NEW)
    │       └── GetResumeOptions()         (NEW)
    └── router.go (MODIFIED)
        └── Add resume endpoints
```

---

## 8. Configuration & Permissions

### New Permissions (RBAC)

```
- execution:pause      # Manual pause
- execution:resume     # Manual resume
- execution:read       # View execution status
```

### Environment Variables

```
# Resume behavior
WORKFLOW_RESUME_TIMEOUT=24h              # How long to keep paused
WORKFLOW_RESUME_AUTO_ESCALATE=false      # Auto-escalate if no decision
WORKFLOW_AUTO_RESUME_AFTER_APPROVAL=true # Resume immediately after approval
WORKFLOW_RESUME_WORKER_INTERVAL=5m       # Check interval
```

---

## 9. Example Workflow: Approval-Based Pause/Resume

### Workflow Definition

```json
{
  "workflow_id": "order_approval",
  "version": "1.0.0",
  "name": "Order Approval Workflow",
  "steps": [
    {
      "id": "check_order_value",
      "type": "condition",
      "condition": {"field": "order.total", "operator": "gte", "value": 10000},
      "on_true": "require_approval",
      "on_false": "allow_order"
    },
    {
      "id": "require_approval",
      "type": "action",
      "action": "block",
      "reason": "Order requires approval",
      "requires": {
        "type": "approval",
        "role": "sales_manager",
        "timeout": "24h"
      },
      "execute": [
        {"type": "notify", "recipients": ["sales_manager"], "message": "..."}
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

### Execution Flow

```
1. Event: order.created (total: $15,000)
   ↓
2. WorkflowExecutor.Execute() starts
   ├─ Status: running
   ├─ Context: {order: {total: 15000}}
   ↓
3. Step: check_order_value (condition)
   ├─ Result: true (15000 >= 10000)
   ├─ Next: require_approval
   ↓
4. Step: require_approval (action)
   ├─ Type: block with approval requirement
   ├─ Creates ApprovalRequest
   ├─ Pauses execution
   │  ├─ Status: paused
   │  ├─ PausedAt: now
   │  ├─ PausedReason: "approval_required"
   │  ├─ NextStepID: "allow_order" (on_true from require_approval)
   │  └─ PausedStepID: "require_approval"
   ↓
5. (Time passes - approver reviews)
   ↓
6. ApprovalService.ApproveRequest() called
   ├─ Sets approval.status = approved
   ├─ Calls workflowResumer.ResumeFromApproval()
   ↓
7. WorkflowResumer.ResumeFromApproval()
   ├─ Fetches paused execution
   ├─ Calls executor.ExecuteWithResumption()
   │  ├─ Resume from next_step_id = "allow_order"
   │  ├─ Skip already-executed steps
   │  ├─ Execute "allow_order" (action: allow)
   ├─ Status: completed
   ├─ Result: allowed
   ↓
8. Order proceeds
```

---

## 10. Testing Strategy

### Unit Tests

```go
// Test pause point detection
TestExecutorDetectsPausePointOnApprovalRequired()

// Test resumption from pause
TestExecutorResumesFromPausePoint()

// Test context preservation across pause
TestContextPreservedAcrossPause()

// Test idempotency
TestResumeIsIdempotent()
```

### Integration Tests

```go
// End-to-end: approval → resume
TestApprovalTriggersResume()

// End-to-end: manual resume
TestManualResume()

// Timeout handling
TestPausedExecutionTimesOut()

// Multiple pause/resume cycles
TestMultiplePauseCycles()
```

### Load Tests

```
- 1000 concurrent paused executions
- Resume 100 executions simultaneously
- Pause point detection under high throughput
```

---

## 11. Backward Compatibility

### Migration Strategy

1. Add new columns with default values (nullable or default)
2. Existing executions don't have paused_at, so treated as completed
3. New code recognizes both old (completed) and new (paused) states
4. No breaking API changes - only additions

### Rollback

1. Disable resume endpoints (404)
2. WorkflowResumer returns no-op
3. Paused executions remain paused
4. Drop new columns if needed

---

## 12. Success Metrics

- [ ] Workflow pause latency: < 100ms
- [ ] Resume latency: < 500ms
- [ ] Approval → resume time: < 1 second
- [ ] 99.9% execution completion after resume
- [ ] Zero data loss in pause/resume cycle
- [ ] Context integrity: 100% test coverage

