# Workflow Execution System Architecture Analysis

## Executive Summary

The Intelligent Workflows system implements a **synchronous, step-by-step workflow executor** with built-in support for blocking operations (approvals) but lacks explicit pause/resume infrastructure. The system is designed to execute workflows sequentially with conditional branching, parallel execution, and action steps, with all execution state persisted to PostgreSQL.

---

## 1. How Workflows Are Currently Executed

### 1.1 Execution Flow

```
Event/Manual Trigger
       ↓
EventRouter (matches event to workflows)
       ↓
WorkflowExecutor.Execute()
       ↓
Build Context (merge trigger payload + enriched data)
       ↓
Sequential Step Execution
       ├─ Condition Steps (evaluate, branch to on_true/on_false)
       ├─ Action Steps (allow/block/execute)
       ├─ Parallel Steps (execute multiple in parallel)
       └─ Wait Steps (placeholder for future pause mechanism)
       ↓
Complete Execution (mark as completed/failed/blocked)
       ↓
Store in Database (workflow_executions, step_executions tables)
```

### 1.2 Execution Modes

1. **Event-Triggered**: Via EventRouter when events match trigger definitions
2. **Manually-Triggered**: Via `EventRouter.TriggerWorkflowManually()`
3. **Scheduled**: Placeholder infrastructure exists (no cron implementation yet)

### 1.3 Execution Lifecycle

```go
// Location: internal/engine/executor.go - WorkflowExecutor.Execute()

1. Create WorkflowExecution record with status="running"
2. Build execution context (trigger payload + enriched data)
3. Execute steps sequentially:
   - Create StepExecution record
   - Execute step (condition/action/parallel/wait)
   - Update StepExecution with result
4. Mark execution as completed/failed/blocked
5. Return execution result
```

### 1.4 Key Characteristics

- **Synchronous execution**: Steps execute sequentially in a single goroutine
- **Single thread per workflow**: No concurrent step execution within a workflow
- **Immediate completion**: Workflows complete in a single API call (blocking for caller)
- **Async event handling**: Workflows triggered by events run in background goroutines
- **No timeout mechanism**: Workflows can run indefinitely (no default timeout enforced)

---

## 2. Where Workflow Execution Logic Is Located

### 2.1 Core Components Directory Structure

```
internal/
├── engine/
│   ├── executor.go              # Main WorkflowExecutor - sequential step execution
│   ├── action_executor.go       # Executes action steps (allow/block/execute)
│   ├── evaluator.go             # Evaluates conditions
│   ├── context.go               # Builds and enriches execution context
│   └── event_router.go          # Routes events to workflows, triggers execution
├── models/
│   ├── execution.go             # WorkflowExecution, StepExecution models
│   ├── workflow.go              # Workflow, Step, Condition, Action models
│   └── event.go                 # Event, ApprovalRequest models
├── repository/postgres/
│   └── execution_repository.go  # Persistence: CreateExecution, UpdateExecution, etc.
├── services/
│   ├── workflow_resumer.go      # **PLACEHOLDER** - where resume logic should go
│   ├── approval_service.go      # Creates approvals, calls WorkflowResumer on decision
│   └── notification_service.go  # Sends notifications (approval requests, etc.)
└── workers/
    └── approval_expiration_worker.go  # Background worker for approval expiration
```

### 2.2 Execution Entry Points

| Location | Function | Trigger |
|----------|----------|---------|
| `EventRouter.RouteEvent()` | Routes event to matching workflows | External event via REST API |
| `EventRouter.TriggerWorkflowManually()` | Manually trigger a workflow | REST API call |
| `EventRouter.ProcessScheduledWorkflows()` | Process scheduled workflows | **Not implemented** |
| `WorkflowExecutor.Execute()` | Execute a specific workflow | Called by EventRouter |

### 2.3 File-by-File Breakdown

#### **internal/engine/executor.go** (450+ lines)
- `WorkflowExecutor` struct with dependencies:
  - `Evaluator`: condition evaluation
  - `ContextBuilder`: context building
  - `ActionExecutor`: action execution
  - `ExecutionRepository`: persistence
- Key methods:
  - `Execute()`: Main entry point for workflow execution
  - `executeSteps()`: Sequential step loop
  - `executeStep()`: Single step execution
  - `executeStepWithRetry()`: Retry logic (exponential/linear backoff)
  - `executeConditionStep()`: Branch logic
  - `executeActionStep()`: Allow/block/execute actions
  - `executeParallelStep()`: Parallel execution with strategies
  - `executeWaitStep()`: Placeholder for pause mechanism (TODO in comments)
  - `completeExecution()`: Finalizes execution in database

#### **internal/engine/action_executor.go** (370+ lines)
- `ActionExecutor` struct manages action execution
- Supported action types:
  - `allow`: Permit operation
  - `block`: Deny operation with reason
  - `execute`: Execute sub-actions (notify, webhook, create_record, etc.)
- Sub-action types:
  - `notify`: Send notification
  - `webhook`/`http_request`: Call external APIs
  - `create_record`: Create entities
  - `update_record`: Update entities
  - `log`: Log messages
- Features:
  - Variable interpolation using `${context.path}` syntax
  - Dot-notation for nested context access
  - HTTP request execution with custom headers

#### **internal/engine/evaluator.go** (250+ lines)
- Condition evaluation logic
- Supported operators:
  - Comparison: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`
  - Membership: `in`, `contains`
  - Pattern: `regex`
- Supports nested AND/OR conditions
- Field resolution via dot notation

#### **internal/engine/context.go** (260+ lines)
- Builds execution context from trigger payload
- Context enrichment:
  - Loads additional resources (e.g., `order.details`, `customer.history`)
  - Caches in Redis for performance
  - Computes derived fields (order_is_high_value, customer_is_new, etc.)
- Merges data at appropriate paths

#### **internal/engine/event_router.go** (220+ lines)
- Routes events to matching workflows
- Event matching via:
  - Exact match: `trigger.event == eventType`
  - Wildcard match: `"order.*"` matches `"order.created"`, `"order.updated"`
- Spawns goroutine per workflow: `go executor.Execute()`
- No queue or persistence between event and execution

#### **internal/repository/postgres/execution_repository.go** (280+ lines)
- Database operations for executions and step executions
- Methods:
  - `CreateExecution()`: Insert workflow execution
  - `UpdateExecution()`: Update execution status/result
  - `GetExecutionByID()`: Fetch execution
  - `ListExecutions()`: List with pagination and filters
  - `CreateStepExecution()`: Insert step execution
  - `UpdateStepExecution()`: Update step result
  - `GetStepExecutions()`: Fetch all steps for execution
  - `GetExecutionTrace()`: Fetch execution + all steps

---

## 3. Infrastructure for Pausing/Resuming Workflows

### 3.1 Current State: Minimal Infrastructure

**Status**: ❌ **Not implemented** - only placeholder exists

### 3.2 The Placeholder: `workflow_resumer.go`

```go
// Location: internal/services/workflow_resumer.go
type WorkflowResumerImpl struct {
    logger *logger.Logger
    // TODO: Add workflow engine reference when implementing actual resume logic
}

func (w *WorkflowResumerImpl) ResumeWorkflow(ctx context.Context, 
    executionID uuid.UUID, approved bool) error {
    w.logger.Infof("Resuming workflow execution %s with approval status: %v", 
        executionID, approved)
    // TODO: Implement actual workflow resumption logic
    return nil
}
```

**What it does**: Logs that resumption was called - that's it. Nothing actually happens.

### 3.3 Where Pause/Resume Is Referenced

1. **ApprovalService** calls `WorkflowResumer.ResumeWorkflow()` after approval decision:
   ```go
   // internal/services/approval_service.go:144
   if s.workflowResumer != nil {
       if err := s.workflowResumer.ResumeWorkflow(ctx, approval.ExecutionID, true); err != nil {
           s.logger.Errorf("Failed to resume workflow after approval: %v", err)
       }
   }
   ```

2. **ExecutionHandler** provides endpoints but no resume endpoint:
   - ✅ GET `/api/v1/executions` - List executions
   - ✅ GET `/api/v1/executions/{id}` - Get execution
   - ✅ GET `/api/v1/executions/{id}/trace` - Get execution trace
   - ❌ POST `/api/v1/executions/{id}/resume` - **Not implemented**

3. **Wait Steps** have TODO comments:
   ```go
   // internal/engine/executor.go:373-384
   func (we *WorkflowExecutor) executeWaitStep(ctx context.Context,
       step *models.Step, execContext map[string]interface{}) error {
       we.logger.Infof("Wait step: waiting for event %s", step.Wait.Event)
       // TODO: Implement actual wait/pause mechanism
       return nil
   }
   ```

### 3.4 Blocking Points (Implicit Pauses)

Workflows currently "pause" implicitly when they hit a blocking action:

1. **Block Action**: Workflow completes with result="blocked"
2. **Approval Creation**: 
   - ApprovalService creates an `approval_request` record
   - Workflow completes (not paused, just finished)
   - System expects external trigger (approval decision) to continue

**Problem**: Workflow is marked COMPLETED even though it's awaiting approval. No mechanism to resume from where it left off.

---

## 4. Persistence and State Management

### 4.1 Database Schema

#### **workflow_executions** Table
```sql
CREATE TABLE workflow_executions (
    id UUID PRIMARY KEY,
    workflow_id UUID REFERENCES workflows(id),
    execution_id VARCHAR(255) UNIQUE NOT NULL,
    trigger_event VARCHAR(255),
    trigger_payload JSONB,          -- Original event data
    context JSONB,                  -- Full execution context
    status VARCHAR(50),             -- pending, running, completed, failed, blocked, cancelled
    result VARCHAR(50),             -- allowed, blocked, executed, failed
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error_message TEXT,
    metadata JSONB
);
```

**Indices**:
- `idx_executions_workflow`: For listing by workflow
- `idx_executions_status`: For filtering by status
- `idx_executions_trigger`: For audit/analysis
- `idx_executions_started_at`: For ordering

#### **step_executions** Table
```sql
CREATE TABLE step_executions (
    id UUID PRIMARY KEY,
    execution_id UUID REFERENCES workflow_executions(id) ON DELETE CASCADE,
    step_id VARCHAR(255) NOT NULL,
    step_type VARCHAR(50),          -- condition, action, parallel, wait
    status VARCHAR(50) NOT NULL,    -- pending, running, completed, failed, skipped
    input JSONB,                    -- Context at step execution
    output JSONB,                   -- Step result (action, success, data, etc.)
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error_message TEXT
);
```

**Indices**:
- `idx_step_executions`: For lookups
- `idx_step_executions_status`: For filtering

#### **approval_requests** Table
```sql
CREATE TABLE approval_requests (
    id UUID PRIMARY KEY,
    request_id VARCHAR(255) UNIQUE NOT NULL,
    execution_id UUID REFERENCES workflow_executions(id),  -- Link to workflow
    entity_type VARCHAR(100),       -- order, quote, product
    entity_id VARCHAR(255),
    requester_id UUID,
    approver_role VARCHAR(100),
    approver_id UUID,
    status VARCHAR(50),             -- pending, approved, rejected, expired
    reason TEXT,
    decision_reason TEXT,
    requested_at TIMESTAMP DEFAULT NOW(),
    decided_at TIMESTAMP,
    expires_at TIMESTAMP
);
```

### 4.2 Execution State Machine

```
State Transitions:
┌─────────────────────────────────────────┐
│ pending (initial, rarely used)          │
└─────────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────┐
│ running (during execution)              │
└─────────────────────────────────────────┘
         ↓           ↓          ↓
      completed  failed    blocked
        ✅         ❌        ⏸️

Other states:
- cancelled (not used in code currently)
```

**Current Limitation**: No "paused" state. Workflows either:
- Complete successfully (status=completed, result=allowed/executed)
- Complete with block (status=completed, result=blocked)
- Fail (status=failed, result=failed)

### 4.3 Data Persistence Strategy

| What | Where | When | How |
|------|-------|------|-----|
| Workflow execution started | PostgreSQL | Immediately (line 73 of executor.go) | INSERT |
| Execution context | PostgreSQL | After building (line 91) | UPDATE |
| Step execution | PostgreSQL | Before step runs (line 217) | INSERT |
| Step result | PostgreSQL | After step completes (line 268) | UPDATE |
| Execution completed | PostgreSQL | After all steps (line 406) | UPDATE |

### 4.4 Context Caching

- **Redis**: Used for enrichment data cache
- **Cache key format**: `context:{resource}:{identifier}`
- **Example**: `context:order.details:ord_123`
- **TTL**: Not set in current code (keys persist indefinitely)

### 4.5 What's NOT Persisted

- ❌ Current step pointer (no "resumption point")
- ❌ Execution state snapshots
- ❌ Intermediate variable state
- ❌ Pause reason or metadata
- ❌ Queue of pending steps

---

## 5. Workflow Engine, Executor, Runner Components

### 5.1 Component Overview

```
┌─────────────────────────────────────────────────┐
│              WorkflowExecutor                    │ Main orchestrator
├─────────────────────────────────────────────────┤
├─ Evaluator: Condition evaluation                │
├─ ContextBuilder: Context building & enrichment  │
├─ ActionExecutor: Action execution               │
├─ ExecutionRepository: Persistence               │
└─────────────────────────────────────────────────┘
                        ↑
                        │ created by
┌─────────────────────────────────────────────────┐
│              EventRouter                        │ Event dispatcher
├─────────────────────────────────────────────────┤
├─ Routes events to matching workflows            │
├─ Spawns goroutine per workflow                  │
├─ Handles scheduled workflows (TODO)             │
├─ Manual trigger support                         │
└─────────────────────────────────────────────────┘
```

### 5.2 WorkflowExecutor (lines 25-50 of executor.go)

**Struct**:
```go
type WorkflowExecutor struct {
    evaluator       *Evaluator
    contextBuilder  *ContextBuilder
    actionExecutor  *ActionExecutor
    executionRepo   ExecutionRepository
    logger          *logger.Logger
    maxRetries      int              // default: 3
    defaultTimeout  time.Duration    // default: 30 seconds (unused)
}
```

**Dependencies Injected**:
- Redis client (for context caching)
- ExecutionRepository (for persistence)
- Logger

**Key Methods**:
- `Execute()` - Main entry point
- `executeSteps()` - Step loop
- `executeStep()` - Single step dispatch
- `executeStepWithRetry()` - Retry coordination
- `executeConditionStep()` - Condition branching
- `executeActionStep()` - Action dispatch
- `executeParallelStep()` - Parallel coordination
- `executeWaitStep()` - **TODO** placeholder
- `completeExecution()` - Finalization

### 5.3 ActionExecutor (lines 27-40 of action_executor.go)

**Scope**: Executes individual "execute" type actions within steps

**Supported Types**:
- `notify` - Send notifications
- `webhook`/`http_request` - HTTP calls
- `create_record` - Entity creation
- `update_record` - Entity updates
- `log` - Logging

**Features**:
- HTTP client with 30s timeout
- Variable interpolation
- Nested context navigation

### 5.4 Evaluator (lines 12-17 of evaluator.go)

**Scope**: Evaluates conditions for conditional branching

**Operators**:
- Comparison: eq, neq, gt, gte, lt, lte
- Membership: in, contains
- Pattern: regex
- Logical: AND, OR

### 5.5 ContextBuilder (lines 16-19 of context.go)

**Scope**: Builds and enriches execution context

**Functions**:
- Merge trigger payload into context
- Load additional resources (from config, placeholder for microservices)
- Enrich with computed fields
- Cache in Redis

### 5.6 EventRouter (lines 27-32 of event_router.go)

**Scope**: Routes events to workflows and triggers execution

**Functions**:
- Match events to workflows (exact + wildcard)
- Spawn goroutine per matching workflow
- Record event in database
- Update event with triggered workflows
- Support manual and scheduled triggers (scheduled is TODO)

### 5.7 ApprovalExpirationWorker (lines 12-17 of approval_expiration_worker.go)

**Scope**: Background worker for periodic approval maintenance

**Features**:
- Runs on configurable interval (default: 5 minutes)
- Marks expired approvals as expired
- Sends notifications on expiration
- Graceful start/stop

---

## 6. Key Architectural Gaps for Workflow Resumption

### 6.1 Missing Components

| Component | Status | Why Needed | Impact |
|-----------|--------|-----------|--------|
| Paused state | ❌ Missing | To distinguish paused from completed | Can't identify which workflows are waiting |
| Pause/Resume API endpoint | ❌ Missing | To allow external resume requests | No API to resume |
| Resumption point tracking | ❌ Missing | To know which step to resume from | Would need to re-execute from start |
| Execution checkpoints | ❌ Missing | To capture state at pause points | Can't resume with context |
| Job queue | ❌ Missing | To queue resumed executions | No way to schedule resumption |
| Resume trigger handler | ❌ Missing | To handle approval/event-driven resumption | No coordination |

### 6.2 Current Limitation Scenario

**Today's Flow**:
```
1. Workflow executes: order.created event
2. Hits approval requirement (block action)
3. Creates approval_request in DB
4. Execution completes with status=completed, result=blocked
5. Approver approves in UI
6. ApprovalService.ApproveRequest() called
7. Calls workflowResumer.ResumeWorkflow() (does nothing)
8. Workflow stays completed - no continuation

What should happen:
8. Resume from "blocked" state
9. Continue to next step after approval
10. Or execute conditional branch based on approval result
```

### 6.3 Challenges for Implementation

1. **Stateless Steps**: Current steps don't track resumption points
2. **Synchronous Model**: Steps run to completion immediately
3. **No Queuing**: No job queue to handle deferred execution
4. **Context Mutation**: Context changes between pause and resume need handling
5. **Idempotency**: Steps may have side effects (webhooks, record creation)

---

## 7. Summary: Current vs. Required for Resumer Feature

### Current Capabilities
- ✅ Sequential workflow execution
- ✅ Step execution tracking (step_executions table)
- ✅ Context building and enrichment
- ✅ Retry logic with backoff
- ✅ Parallel execution
- ✅ Conditional branching
- ✅ Full execution persistence
- ✅ Approval request framework

### Missing for Full Resumption
- ❌ Paused execution state
- ❌ Resume checkpoint tracking
- ❌ Asynchronous execution model
- ❌ Job queue for deferred execution
- ❌ Resume API endpoints
- ❌ Step idempotency markers
- ❌ Approval decision feedback into workflow

### Recommended Implementation Approach

1. **Add `paused` status** to execution states
2. **Extend `workflow_executions` table** with:
   - `paused_at` TIMESTAMP
   - `paused_reason` TEXT
   - `resume_data` JSONB (approval decision, etc.)
   - `next_step_id` VARCHAR (resumption point)
3. **Implement `ResumeWorkflow()` service** that:
   - Fetches paused execution
   - Updates next step pointer
   - Queues execution for resumption
   - Or immediately resumes in-process
4. **Add REST endpoint** `POST /api/v1/executions/{id}/resume`
5. **Modify step execution** to support resumption:
   - Skip already-executed steps
   - Continue from next step

---

## Files Summary

| File | Lines | Purpose |
|------|-------|---------|
| `executor.go` | 448 | Main workflow execution orchestrator |
| `action_executor.go` | 373 | Action execution (notify, webhook, etc.) |
| `evaluator.go` | 247 | Condition evaluation |
| `context.go` | 263 | Context building and enrichment |
| `event_router.go` | 224 | Event routing and workflow triggering |
| `execution_repository.go` | 305 | Database persistence |
| `approval_service.go` | 288 | Approval request management |
| `workflow_resumer.go` | 43 | **PLACEHOLDER** - Resume logic (unimplemented) |
| `execution.go` | 123 | Models: WorkflowExecution, StepExecution |
| `workflow.go` | 146 | Models: Workflow, Step, Condition, Action |

