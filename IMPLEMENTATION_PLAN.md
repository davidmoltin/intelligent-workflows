# Implementation Plan - Codebase Gap Remediation

**Created:** 2025-11-05
**Status:** Ready for Execution
**Estimated Total Effort:** 6-8 weeks (1 developer)

---

## Executive Summary

This plan addresses the critical gaps identified in the codebase review. Tasks are organized into three phases: **Critical (Blocks Production)**, **High Priority (Pre-GA)**, and **Post-MVP Enhancements**. Each task includes effort estimates, dependencies, and detailed implementation steps.

---

## Phase 1: Critical - Production Blockers (2-3 weeks)

### 1.1 Add Workflow Resumer Tests
**Priority:** P0 - Critical
**Effort:** 3-4 days
**Dependencies:** None
**Risk:** High - Recently implemented code (Phase 5) has no test coverage

#### Scope
- Service-layer unit tests for `WorkflowResumerImpl`
- Handler-level integration tests for pause/resume endpoints
- Background worker integration tests
- Edge case coverage (concurrent resume, invalid states)

#### Implementation Steps

**Step 1: Service Layer Tests** (1.5 days)
```
File: internal/services/workflow_resumer_service_test.go
```

Test cases to implement:
- `TestPauseExecution_Success` - Happy path pause
- `TestPauseExecution_AlreadyPaused` - Idempotent pause
- `TestPauseExecution_CompletedExecution` - Cannot pause completed
- `TestPauseExecution_InvalidExecutionID` - Not found error
- `TestResumeExecution_Success` - Happy path resume
- `TestResumeExecution_NotPaused` - Cannot resume non-paused
- `TestResumeExecution_WithResumeData` - Resume data restoration
- `TestResumeWorkflow_FromApproval` - Auto-resume after approval
- `TestGetPausedExecutions_Filtering` - List paused with filters
- `TestGetPausedExecutions_Pagination` - Pagination logic
- `TestCanResume_ValidationRules` - Resume validation logic

**Step 2: Handler Tests** (1 day)
```
File: internal/api/rest/handlers/workflow_resumer_test.go
```

Test cases to implement:
- `TestHandlePauseExecution_Success` - 200 response
- `TestHandlePauseExecution_Unauthorized` - 401 response
- `TestHandlePauseExecution_Forbidden` - 403 response (no permission)
- `TestHandlePauseExecution_InvalidID` - 400 response
- `TestHandleResumeExecution_Success` - 200 response
- `TestHandleResumeExecution_WithBody` - Resume data in request
- `TestHandleListPausedExecutions_Success` - 200 with pagination
- `TestHandleListPausedExecutions_EmptyResult` - 200 with empty array

**Step 3: Worker Integration Tests** (0.5 days)
```
File: internal/workers/workflow_resumer_worker_test.go
```

Test cases to implement:
- `TestWorker_AutoResumeWithApproval` - Approved approval triggers resume
- `TestWorker_SkipRejectedApproval` - Rejected approval doesn't resume
- `TestWorker_BatchProcessing` - Processes multiple executions
- `TestWorker_LongPausedWarning` - Warns on executions paused > 24h
- `TestWorker_ErrorHandling` - Graceful error handling

**Step 4: Edge Cases** (1 day)
- Concurrent resume attempts (use locks/transactions)
- Resume with modified workflow definition (validation)
- Nested parallel steps pause/resume
- Race condition between manual and auto-resume

#### Acceptance Criteria
- [ ] All service methods have unit tests
- [ ] All API endpoints have handler tests
- [ ] Worker has integration tests
- [ ] Test coverage > 80% for resumer code
- [ ] All edge cases documented and tested

---

### 1.2 Add Panic Recovery to Event Routing
**Priority:** P0 - Critical
**Effort:** 0.5 days
**Dependencies:** None
**Risk:** Medium - Goroutines could crash silently

#### Current Issue
```go
// internal/engine/event_router.go:87
go func() {
    // No panic recovery! If this crashes, worker dies silently
    r.executeWorkflow(ctx, workflow, event)
}()
```

#### Implementation Steps

**Step 1: Add Recovery Middleware** (2 hours)
```go
File: internal/engine/event_router.go

// Add helper function
func (r *EventRouter) safeExecuteWorkflow(ctx context.Context, wf *models.Workflow, event models.Event) {
    defer func() {
        if rec := recover(); rec != nil {
            r.logger.Error("panic in workflow execution goroutine",
                "workflow_id", wf.ID,
                "event_type", event.Type,
                "panic", rec,
                "stack", string(debug.Stack()))

            // Optionally: record execution as failed
            r.recordPanicExecution(ctx, wf.ID, event, rec)
        }
    }()

    r.executeWorkflow(ctx, wf, event)
}

// Update goroutine call
go r.safeExecuteWorkflow(ctx, workflow, event)
```

**Step 2: Add Panic Recording** (2 hours)
- Create execution record with status "failed"
- Store panic message in execution trace
- Send alert/notification to monitoring system

**Step 3: Add Tests** (1 hour)
```
File: internal/engine/event_router_test.go
```

Test cases:
- `TestSafeExecuteWorkflow_PanicRecovery` - Recovers from panic
- `TestSafeExecuteWorkflow_PanicLogging` - Logs panic with stack
- `TestSafeExecuteWorkflow_PanicRecording` - Creates failed execution

#### Acceptance Criteria
- [ ] All goroutines have panic recovery
- [ ] Panics are logged with stack traces
- [ ] Failed executions are recorded
- [ ] Tests verify recovery behavior

---

### 1.3 Implement Timeout Enforcement
**Priority:** P0 - Critical
**Effort:** 1 day
**Dependencies:** None
**Risk:** Medium - Workflows could run indefinitely

#### Current Issue
```go
// internal/engine/executor.go:35
const defaultTimeout = 5 * time.Minute  // Defined but never used!
```

#### Implementation Steps

**Step 1: Add Context Timeout** (3 hours)
```go
File: internal/engine/executor.go

func (e *WorkflowExecutor) ExecuteWorkflow(ctx context.Context, execution *models.WorkflowExecution) error {
    // Get timeout from workflow or use default
    timeout := e.getWorkflowTimeout(execution.WorkflowID)
    if timeout == 0 {
        timeout = defaultTimeout
    }

    // Create timeout context
    timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Pass timeout context to execution
    return e.executeWithContext(timeoutCtx, execution)
}
```

**Step 2: Add Timeout to Workflow Model** (2 hours)
- Add `execution_timeout_seconds` to workflow definition
- Update migration to add column
- Update workflow creation/update handlers

**Step 3: Handle Timeout Gracefully** (2 hours)
```go
// Check for timeout in step loop
select {
case <-ctx.Done():
    if ctx.Err() == context.DeadlineExceeded {
        return e.handleTimeout(execution)
    }
    return ctx.Err()
default:
    // Continue execution
}
```

**Step 4: Add Tests** (1 hour)
```
File: internal/engine/executor_test.go
```

Test cases:
- `TestExecuteWorkflow_Timeout` - Workflow times out
- `TestExecuteWorkflow_CustomTimeout` - Uses workflow-specific timeout
- `TestExecuteWorkflow_TimeoutRecording` - Records timeout status
- `TestExecuteWorkflow_NoTimeout` - Completes before timeout

#### Acceptance Criteria
- [ ] All workflow executions have timeout enforcement
- [ ] Timeout is configurable per workflow
- [ ] Timeout failures are properly recorded
- [ ] Tests verify timeout behavior

---

### 1.4 Implement Cron-Based Scheduling
**Priority:** P0 - Critical (if scheduled workflows required)
**Effort:** 5 days
**Dependencies:** None
**Risk:** High - Core feature for scheduled automation

#### Current Issue
```go
// internal/engine/event_router.go:172
func (r *EventRouter) ProcessScheduledWorkflows(ctx context.Context) {
    // TODO: For now, this is a placeholder
}
```

#### Implementation Steps

**Step 1: Add Cron Library** (0.5 days)
```bash
go get github.com/robfig/cron/v3
```

**Step 2: Add Schedule Model** (1 day)
```sql
File: migrations/009_workflow_schedules.up.sql

CREATE TABLE workflow_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    enabled BOOLEAN DEFAULT true,
    last_triggered_at TIMESTAMP,
    next_trigger_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT valid_cron CHECK (cron_expression ~ '^[@0-9*/,-]+\s+[@0-9*/,-]+\s+[@0-9*/,-]+\s+[@0-9*/,-]+\s+[@0-9*/,-]+$')
);

CREATE INDEX idx_schedules_next_trigger ON workflow_schedules(next_trigger_at) WHERE enabled = true;
CREATE INDEX idx_schedules_workflow ON workflow_schedules(workflow_id);
```

**Step 3: Implement Schedule Service** (1.5 days)
```go
File: internal/services/schedule_service.go

type ScheduleService interface {
    CreateSchedule(ctx context.Context, schedule *models.WorkflowSchedule) error
    UpdateSchedule(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
    DeleteSchedule(ctx context.Context, id uuid.UUID) error
    GetDueSchedules(ctx context.Context) ([]*models.WorkflowSchedule, error)
    MarkTriggered(ctx context.Context, id uuid.UUID) error
}
```

**Step 4: Implement Scheduler Worker** (1.5 days)
```go
File: internal/workers/scheduler_worker.go

type SchedulerWorker struct {
    scheduleService ScheduleService
    eventRouter     *EventRouter
    interval        time.Duration
}

func (w *SchedulerWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            w.processDueSchedules(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (w *SchedulerWorker) processDueSchedules(ctx context.Context) {
    schedules, err := w.scheduleService.GetDueSchedules(ctx)
    if err != nil {
        // log error
        return
    }

    for _, schedule := range schedules {
        // Create synthetic event
        event := models.Event{
            Type: "schedule.triggered",
            Payload: map[string]interface{}{
                "schedule_id": schedule.ID,
                "workflow_id": schedule.WorkflowID,
            },
        }

        // Route event to workflow
        w.eventRouter.RouteEvent(ctx, event)

        // Mark as triggered and calculate next run
        w.scheduleService.MarkTriggered(ctx, schedule.ID)
    }
}
```

**Step 5: Add REST API Endpoints** (0.5 days)
```
POST   /api/v1/workflows/{id}/schedules
GET    /api/v1/workflows/{id}/schedules
PUT    /api/v1/schedules/{id}
DELETE /api/v1/schedules/{id}
GET    /api/v1/schedules/{id}/next-runs (preview next 10 runs)
```

**Step 6: Add Tests** (1 day)
- Schedule service tests
- Scheduler worker tests
- Cron expression validation tests
- Timezone handling tests

#### Acceptance Criteria
- [ ] Workflows can be scheduled with cron expressions
- [ ] Scheduler worker processes schedules every minute
- [ ] Missed schedules are handled (catchup or skip)
- [ ] Timezone support works correctly
- [ ] REST API endpoints for schedule management
- [ ] Tests cover cron parsing and schedule execution

---

## Phase 2: High Priority - Pre-GA (2-3 weeks)

### 2.1 Replace Microservice Integration Stubs
**Priority:** P1 - High
**Effort:** 1-2 weeks
**Dependencies:** Microservice specifications
**Risk:** High - Core feature for real-world use

#### Current Issue
```go
// internal/engine/action_executor.go:155
func (e *ActionExecutor) executeCreateRecord(ctx context.Context, action models.Action) error {
    // TODO: This is a placeholder for actual microservice integration
    return nil
}
```

#### Implementation Steps

**Step 1: Design Service Client Interface** (1 day)
```go
File: pkg/clients/service_client.go

type ServiceClient interface {
    CreateRecord(ctx context.Context, service, recordType string, data map[string]interface{}) (string, error)
    UpdateRecord(ctx context.Context, service, recordType, recordID string, data map[string]interface{}) error
    DeleteRecord(ctx context.Context, service, recordType, recordID string) error
    GetRecord(ctx context.Context, service, recordType, recordID string) (map[string]interface{}, error)
}

type HTTPServiceClient struct {
    baseURLs   map[string]string  // service name -> base URL
    httpClient *http.Client
    auth       AuthProvider
}
```

**Step 2: Implement HTTP Client** (2 days)
- JSON serialization/deserialization
- Authentication (API key, JWT)
- Retry logic with exponential backoff
- Circuit breaker for failing services
- Request/response logging

**Step 3: Add Service Registry** (1 day)
```go
File: pkg/clients/service_registry.go

type ServiceRegistry struct {
    services map[string]ServiceConfig
}

type ServiceConfig struct {
    BaseURL     string
    AuthType    string  // "api_key", "jwt", "none"
    Credentials map[string]string
    Timeout     time.Duration
}

// Load from config/database
func LoadServiceRegistry(ctx context.Context) (*ServiceRegistry, error)
```

**Step 4: Update Action Executor** (1 day)
```go
File: internal/engine/action_executor.go

func (e *ActionExecutor) executeCreateRecord(ctx context.Context, action models.Action) error {
    service := action.Config["service"].(string)
    recordType := action.Config["record_type"].(string)
    data := action.Config["data"].(map[string]interface{})

    recordID, err := e.serviceClient.CreateRecord(ctx, service, recordType, data)
    if err != nil {
        return fmt.Errorf("failed to create record: %w", err)
    }

    // Store record ID in execution context
    e.storeActionResult(action.ID, recordID)
    return nil
}
```

**Step 5: Add Configuration** (0.5 days)
```yaml
File: config/services.yaml

services:
  user-service:
    base_url: "https://api.example.com/users"
    auth_type: "api_key"
    api_key: "${USER_SERVICE_API_KEY}"
    timeout: 30s

  order-service:
    base_url: "https://api.example.com/orders"
    auth_type: "jwt"
    jwt_secret: "${ORDER_SERVICE_JWT_SECRET}"
    timeout: 60s
```

**Step 6: Add Tests** (2 days)
- HTTP client unit tests with mocks
- Service registry tests
- Action executor integration tests
- Error handling tests (timeout, 5xx, auth failure)

#### Acceptance Criteria
- [ ] HTTP service client with retry and circuit breaker
- [ ] Service registry for multi-service support
- [ ] Authentication support (API key, JWT)
- [ ] Action executor uses real client
- [ ] Configuration externalized
- [ ] Tests with mocked HTTP responses

---

### 2.2 Implement External Resource Loading
**Priority:** P1 - High
**Effort:** 3 days
**Dependencies:** Service client from 2.1
**Risk:** Medium - Needed for context enrichment

#### Current Issue
```go
// internal/engine/context.go:78
func (b *ContextBuilder) loadData(ctx context.Context, resource models.ContextResource) (interface{}, error) {
    // TODO: Implement actual resource loading from external APIs/services
    // For now, return placeholder data
    return map[string]interface{}{"placeholder": "data"}, nil
}
```

#### Implementation Steps

**Step 1: Define Resource Loaders** (1 day)
```go
File: internal/engine/resource_loaders.go

type ResourceLoader interface {
    Load(ctx context.Context, resource models.ContextResource) (interface{}, error)
    CanLoad(resource models.ContextResource) bool
}

// HTTP API loader
type HTTPResourceLoader struct {
    serviceClient ServiceClient
}

// Database query loader
type DatabaseResourceLoader struct {
    db *sql.DB
}

// Redis cache loader
type RedisResourceLoader struct {
    redis *redis.Client
}
```

**Step 2: Implement Loaders** (1 day)
- HTTP loader: GET/POST to external APIs
- Database loader: Execute SQL queries
- Redis loader: Get keys/hashes
- GraphQL loader: Execute GraphQL queries

**Step 3: Update Context Builder** (0.5 days)
```go
File: internal/engine/context.go

func (b *ContextBuilder) loadData(ctx context.Context, resource models.ContextResource) (interface{}, error) {
    for _, loader := range b.loaders {
        if loader.CanLoad(resource) {
            data, err := loader.Load(ctx, resource)
            if err != nil {
                if resource.Optional {
                    b.logger.Warn("failed to load optional resource", "error", err)
                    return nil, nil
                }
                return nil, err
            }
            return data, nil
        }
    }

    return nil, fmt.Errorf("no loader found for resource type: %s", resource.Type)
}
```

**Step 4: Add Caching** (0.5 days)
- Cache loaded resources in Redis
- TTL-based expiration
- Cache key: workflow_id + resource_id + hash(params)

**Step 5: Add Tests** (1 day)
- HTTP loader tests with mocked responses
- Database loader tests with test DB
- Caching tests
- Error handling tests

#### Acceptance Criteria
- [ ] Multiple resource loaders implemented
- [ ] Context builder uses loaders
- [ ] Caching for expensive loads
- [ ] Optional resources don't fail workflow
- [ ] Tests cover all loader types

---

### 2.3 Add Database Connection Resilience
**Priority:** P1 - High
**Effort:** 2 days
**Dependencies:** None
**Risk:** Medium - Connection failures could crash app

#### Implementation Steps

**Step 1: Add Retry Logic** (1 day)
```go
File: pkg/database/connection.go

func ConnectWithRetry(config DatabaseConfig) (*sql.DB, error) {
    var db *sql.DB
    var err error

    backoff := []time.Duration{1*time.Second, 2*time.Second, 5*time.Second, 10*time.Second}

    for i := 0; i < len(backoff); i++ {
        db, err = sql.Open("postgres", config.DSN())
        if err == nil {
            err = db.Ping()
            if err == nil {
                return db, nil
            }
        }

        if i < len(backoff)-1 {
            time.Sleep(backoff[i])
        }
    }

    return nil, fmt.Errorf("failed to connect after %d attempts: %w", len(backoff), err)
}
```

**Step 2: Add Circuit Breaker** (0.5 days)
```bash
go get github.com/sony/gobreaker
```

```go
File: pkg/database/circuit_breaker.go

var dbCircuitBreaker *gobreaker.CircuitBreaker

func InitCircuitBreaker() {
    settings := gobreaker.Settings{
        Name:        "database",
        MaxRequests: 3,
        Interval:    10 * time.Second,
        Timeout:     60 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= 3 && failureRatio >= 0.6
        },
    }

    dbCircuitBreaker = gobreaker.NewCircuitBreaker(settings)
}
```

**Step 3: Wrap Database Calls** (0.5 day)
```go
// Wrap critical queries
result, err := dbCircuitBreaker.Execute(func() (interface{}, error) {
    return db.Query("SELECT ...")
})
```

#### Acceptance Criteria
- [ ] Database connection has retry logic
- [ ] Circuit breaker prevents cascade failures
- [ ] Monitoring alerts on circuit open
- [ ] Tests verify retry behavior

---

### 2.4 Add Workflow Definition Validation
**Priority:** P2 - Medium
**Effort:** 2 days
**Dependencies:** None
**Risk:** Low - Invalid workflows could be created

#### Implementation Steps

**Step 1: Create Validator** (1 day)
```go
File: internal/validators/workflow_validator.go

type WorkflowValidator struct {
    db *sql.DB
}

func (v *WorkflowValidator) Validate(workflow *models.Workflow) error {
    var errors []string

    // Check all step references are valid
    if err := v.validateStepReferences(workflow); err != nil {
        errors = append(errors, err.Error())
    }

    // Check no circular dependencies
    if err := v.validateNoCycles(workflow); err != nil {
        errors = append(errors, err.Error())
    }

    // Check action configs are valid
    if err := v.validateActionConfigs(workflow); err != nil {
        errors = append(errors, err.Error())
    }

    // Check condition syntax
    if err := v.validateConditions(workflow); err != nil {
        errors = append(errors, err.Error())
    }

    if len(errors) > 0 {
        return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
    }

    return nil
}
```

**Step 2: Add to Handlers** (0.5 days)
```go
// In create/update handlers
if err := validator.Validate(workflow); err != nil {
    return response.BadRequest(w, err.Error())
}
```

**Step 3: Add Tests** (0.5 days)
- Invalid step reference test
- Circular dependency test
- Invalid action config test
- Invalid condition syntax test

#### Acceptance Criteria
- [ ] Workflows validated on create/update
- [ ] Clear error messages for invalid workflows
- [ ] Tests cover validation rules

---

### 2.5 Make Configuration Externalized
**Priority:** P2 - Low
**Effort:** 0.5 days
**Dependencies:** None
**Risk:** Low - Hardcoded values

#### Implementation Steps

**Step 1: Add Config Fields** (0.25 days)
```go
File: internal/config/config.go

type Config struct {
    // ... existing fields ...

    Version              string `env:"APP_VERSION" envDefault:"0.1.0"`
    DefaultApproverEmail string `env:"DEFAULT_APPROVER_EMAIL" envDefault:"admin@example.com"`
}
```

**Step 2: Update Usage** (0.25 days)
```go
// internal/api/rest/handlers/health.go
Version: cfg.Version,

// internal/services/approval_service.go
defaultEmail := s.config.DefaultApproverEmail
```

#### Acceptance Criteria
- [ ] Version from config
- [ ] Approver email from config
- [ ] Environment variables documented in .env.example

---

## Phase 3: Post-MVP - Enhancements (4+ weeks)

### 3.1 Complete Frontend Workflow Builder
**Priority:** P3 - Post-MVP
**Effort:** 3-4 weeks
**Dependencies:** None
**Risk:** Low - Nice-to-have feature

#### Implementation Steps
- Visual drag-and-drop workflow editor
- Step configuration panels
- Condition builder UI
- Workflow testing interface
- Execution visualization dashboard

*(Detailed plan omitted for brevity - can be expanded if needed)*

---

### 3.2 Advanced Workflow Resumer Features
**Priority:** P3 - Post-MVP
**Effort:** 1-2 weeks
**Dependencies:** Phase 1 tests complete
**Risk:** Low - Advanced features

#### Features
- Scheduled resume (resume at specific time)
- Conditional resume rules
- Webhook callbacks on pause/resume events
- Bulk pause/resume operations
- Resume with modified workflow definitions

---

### 3.3 LLM Integration Examples
**Priority:** P3 - Post-MVP
**Effort:** 1 week
**Dependencies:** None
**Risk:** Low - Documentation/examples

#### Deliverables
- Example workflows using LLM actions
- Prompt engineering guide
- Template library
- Cost estimation guide

---

## Execution Strategy

### Recommended Approach

**Week 1-2: Critical Tests & Safety**
1. Workflow Resumer tests (1.1)
2. Panic recovery (1.2)
3. Timeout enforcement (1.3)

**Week 3-4: Scheduling (if required)**
4. Cron scheduling implementation (1.4)

**Week 5-6: Service Integration**
5. Microservice client (2.1)
6. Resource loading (2.2)

**Week 7-8: Reliability & Polish**
7. Database resilience (2.3)
8. Validation (2.4)
9. Configuration (2.5)

### Parallelization Opportunities

Can be done in parallel:
- 1.1 (tests) + 1.2 (panic recovery)
- 1.3 (timeout) + 2.5 (config)
- 2.1 (service client) + 2.3 (DB resilience)

### Risk Mitigation

**High-Risk Items:**
- Cron scheduling (1.4) - Consider using battle-tested library
- Service integration (2.1) - Start with one service, then expand
- Workflow Resumer tests (1.1) - Critical for production confidence

**Mitigation:**
- Feature flags for new functionality
- Incremental rollout (canary deployments)
- Comprehensive integration tests before merge

---

## Success Criteria

### Phase 1 Complete
- [ ] All critical tests passing (>80% coverage)
- [ ] No goroutines can crash silently
- [ ] All workflows have timeout enforcement
- [ ] Scheduled workflows functional (if required)

### Phase 2 Complete
- [ ] Real microservice integration working
- [ ] External resource loading functional
- [ ] Database connection resilient to failures
- [ ] Invalid workflows rejected on create

### Ready for Production
- [ ] All Phase 1 tasks complete
- [ ] All Phase 2 P1 tasks complete
- [ ] Documentation updated
- [ ] Security review passed
- [ ] Load testing completed

---

## Resource Requirements

**Developer Time:**
- Phase 1: 2-3 weeks (1 developer) or 1-1.5 weeks (2 developers)
- Phase 2: 2-3 weeks (1 developer)
- Phase 3: 4+ weeks (optional enhancements)

**Infrastructure:**
- Test database instances
- Test microservices (for integration testing)
- CI/CD pipeline capacity

**Dependencies:**
- Access to microservice specifications
- Test environment setup
- Security review scheduling

---

## Next Steps

1. **Review and prioritize** - Confirm which tasks are must-have for your GA
2. **Resource allocation** - Assign developers to tasks
3. **Create feature branches** - Set up git workflow
4. **Start with tests** - Task 1.1 provides immediate safety improvement
5. **Daily standups** - Track progress and blockers

---

## Questions for Decision

1. **Scheduled workflows:** Are cron-based workflows a requirement for GA?
2. **Service integration:** Which microservices need integration first?
3. **Frontend:** Is workflow builder needed for GA, or can it wait?
4. **Timeline:** What is the target GA date?
5. **Resources:** How many developers can work on this?

---

**Document Status:** Ready for execution
**Last Updated:** 2025-11-05
**Owner:** TBD
**Reviewer:** TBD
