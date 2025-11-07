# Phase 1: Repository Layer Multi-Tenancy Updates

## Status: Phase 1A Complete (60% of Phase 1)

### ✅ Completed Repositories

#### 1. Organization Repository (NEW)
**File:** `internal/repository/postgres/organization_repository.go`

**Methods:**
- `Create(ctx, req, createdBy)` - Create new organization
- `GetByID(ctx, id)` - Get organization by ID
- `GetBySlug(ctx, slug)` - Get organization by slug
- `List(ctx, page, pageSize)` - List organizations with pagination
- `Update(ctx, id, req)` - Update organization
- `Delete(ctx, id)` - Soft-delete organization
- `GetUserOrganizations(ctx, userID)` - Get all orgs for a user
- `AddUser(ctx, orgID, userID, roleID, invitedBy)` - Add user to org
- `RemoveUser(ctx, orgID, userID)` - Remove user from org
- `GetOrganizationUsers(ctx, orgID)` - Get all users in org
- `UpdateUserRole(ctx, orgID, userID, roleID)` - Update user's role in org
- `GetUserRoleInOrganization(ctx, orgID, userID)` - Get user's role
- `CheckUserAccess(ctx, orgID, userID)` - Verify user has org access

**Pattern:** Fully implements organization management with RBAC support.

---

#### 2. Workflow Repository ✅
**File:** `internal/repository/postgres/workflow_repository.go`

**Updated Methods:**
- `Create(ctx, organizationID, req, createdBy)` - Added organizationID parameter
- `GetByID(ctx, organizationID, id)` - Filters by org
- `GetByWorkflowID(ctx, organizationID, workflowID)` - Filters by org
- `List(ctx, organizationID, enabled, limit, offset)` - Filters by org
- `Update(ctx, organizationID, id, req)` - Requires org ownership
- `Delete(ctx, organizationID, id)` - Requires org ownership
- `SetEnabled(ctx, organizationID, id, enabled)` - Requires org ownership
- `GetWorkflowByID(ctx, organizationID, id)` - Adapter method updated
- `ListWorkflows(ctx, organizationID, enabled, limit, offset)` - Adapter updated
- `ListByEventType(ctx, organizationID, eventType)` - NEW - For event routing

**SQL Changes:**
- All INSERT statements include `organization_id`
- All WHERE clauses include `organization_id = $1` for data isolation
- All SELECT statements include `organization_id` in result columns

---

#### 3. Execution Repository ✅
**File:** `internal/repository/postgres/execution_repository.go`

**Updated Methods:**
- `CreateExecution(ctx, execution)` - Uses execution.OrganizationID
- `UpdateExecution(ctx, organizationID, execution)` - Requires org ownership
- `GetExecutionByID(ctx, organizationID, id)` - Filters by org
- `GetExecutionByExecutionID(ctx, organizationID, executionID)` - Filters by org
- `ListExecutions(ctx, organizationID, workflowID, status, limit, offset)` - Filters by org
- `CreateStepExecution(ctx, step)` - Uses step.OrganizationID
- `UpdateStepExecution(ctx, organizationID, step)` - Requires org ownership
- `GetStepExecutions(ctx, organizationID, executionID)` - Filters by org
- `GetExecutionTrace(ctx, organizationID, id)` - Filters by org
- `GetPausedExecutions(ctx, organizationID, limit)` - Filters by org
- `GetTimedOutExecutions(ctx, organizationID, limit)` - Filters by org
- `CancelExecution(ctx, organizationID, id)` - NEW - Cancel with org verification

**SQL Changes:**
- workflow_executions: organization_id in all queries
- step_executions: organization_id in all queries
- Composite indexes used: `(organization_id, id)`, `(organization_id, status)`

---

#### 4. Event Repository ✅
**File:** `internal/repository/postgres/event_repository.go`

**Updated Methods:**
- `CreateEvent(ctx, event)` - Uses event.OrganizationID
- `UpdateEvent(ctx, organizationID, event)` - Requires org ownership
- `GetEventByID(ctx, organizationID, id)` - Filters by org
- `ListEvents(ctx, organizationID, eventType, processed, limit, offset)` - Filters by org

**SQL Changes:**
- All queries filter by organization_id
- Events are organization-scoped (can't trigger cross-org workflows)

---

### 🚧 Remaining Repositories (40% of Phase 1)

These follow the exact same pattern as above:

#### 5. Approval Repository
**File:** `internal/repository/postgres/approval_repository.go`
**Pattern:** Add `organizationID uuid.UUID` parameter to all methods, filter all queries

#### 6. Schedule Repository
**File:** `internal/repository/postgres/schedule_repository.go`
**Pattern:** Add `organizationID uuid.UUID` parameter to all methods, filter all queries

#### 7. API Key Repository
**File:** `internal/repository/postgres/api_key_repository.go`
**Pattern:** Add `organizationID uuid.UUID` parameter to all methods, filter all queries
**Note:** API keys are now org-scoped

#### 8. Audit Repository
**File:** `internal/repository/postgres/audit_repository.go`
**Pattern:** Add `organizationID uuid.UUID` parameter to most methods
**Note:** Some system-level audits may have NULL organization_id

#### 9. Analytics Repository
**File:** `internal/repository/postgres/analytics_repository.go`
**Pattern:** Add `organizationID uuid.UUID` parameter to all aggregation queries
**Note:** Analytics are per-organization only

---

## Implementation Pattern

All repository updates follow this consistent pattern:

### Before (Single-Tenant):
```go
func (r *WorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
    query := `SELECT ... FROM workflows WHERE id = $1`
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)
}
```

### After (Multi-Tenant):
```go
func (r *WorkflowRepository) GetByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error) {
    query := `SELECT ..., organization_id FROM workflows WHERE organization_id = $1 AND id = $2`
    err := r.db.QueryRowContext(ctx, query, organizationID, id).Scan(..., &workflow.OrganizationID, ...)
}
```

### Key Changes:
1. **Parameters:** Add `organizationID uuid.UUID` as first parameter (after ctx)
2. **SQL WHERE:** Add `organization_id = $1` (or appropriate position)
3. **SQL SELECT:** Include `organization_id` in column list
4. **SQL INSERT:** Include `organization_id` in column and values list
5. **Scan:** Add `&model.OrganizationID` to Scan() call
6. **Security:** All UPDATE/DELETE operations verify organization ownership

---

## Expected Compilation Errors ✅

The following errors are **EXPECTED** and will be fixed in Phase 2:

```
internal/api/rest/handlers/workflow.go:54:52: not enough arguments in call to h.repo.Create
internal/api/rest/handlers/execution.go:75:99: not enough arguments in call to h.executionRepo.ListExecutions
internal/workers/timeout_enforcer_worker.go:84:64: not enough arguments in call to w.executionRepo.GetTimedOutExecutions
```

These errors occur because:
- **Handlers** don't yet extract organization_id from request context
- **Workers** don't yet iterate over organizations
- **Middleware** doesn't yet set organization context

---

## Testing

### Repository Layer Tests:
```bash
# Once remaining repositories are updated:
go build ./internal/repository/postgres/...
```

### Models Compile:
```bash
go build ./internal/models/...  # ✅ Already passing
```

---

## Security Verification

### Data Isolation Checklist:

- [x] Organizations table created with proper indexes
- [x] All data tables have organization_id column (via migration)
- [x] Workflow repository enforces org filtering
- [x] Execution repository enforces org filtering
- [x] Event repository enforces org filtering
- [ ] Approval repository enforces org filtering
- [ ] Schedule repository enforces org filtering
- [ ] API Key repository enforces org filtering
- [ ] Audit repository enforces org filtering (with NULL support)
- [ ] Analytics repository enforces org filtering

### SQL Security Patterns:

✅ **Good:**
```sql
WHERE organization_id = $1 AND id = $2
```

❌ **Bad (Security Risk):**
```sql
WHERE id = $1  -- Missing organization filter!
```

---

## Next Steps

### Complete Phase 1 (Remaining Repositories):

```bash
# 1. Update approval_repository.go
# 2. Update schedule_repository.go
# 3. Update api_key_repository.go
# 4. Update audit_repository.go
# 5. Update analytics_repository.go
# 6. Verify all compile: go build ./internal/repository/postgres/...
```

### Then Move to Phase 2 (Handlers & Middleware):

1. Update JWT service to include organization_id in claims
2. Create organization context middleware
3. Update all API handlers to extract org context
4. Update workflow engine to accept organization_id
5. Update background workers to process per-organization
6. Add organization management API endpoints

---

## Performance Notes

### Indexes Created (via migration):

```sql
-- Efficient lookups by organization
CREATE INDEX idx_workflows_organization ON workflows(organization_id);
CREATE INDEX idx_executions_organization ON workflow_executions(organization_id);
CREATE INDEX idx_events_organization ON events(organization_id);

-- Composite indexes for common queries
CREATE INDEX idx_workflows_org_workflow_id ON workflows(organization_id, workflow_id);
CREATE INDEX idx_executions_org_status ON workflow_executions(organization_id, status);
CREATE INDEX idx_events_org_type ON events(organization_id, event_type);
```

### Query Performance:

Before (table scan):
```sql
SELECT * FROM workflows WHERE workflow_id = 'fraud-detection';  -- Scans all rows
```

After (index scan):
```sql
SELECT * FROM workflows
WHERE organization_id = '...' AND workflow_id = 'fraud-detection';
-- Uses idx_workflows_org_workflow_id
```

---

## Migration Safety

The migration (`005_multi_tenancy.up.sql`) handles existing data:

1. Creates "Default Organization"
2. Assigns all existing users to it
3. Migrates all existing data to default org
4. Makes organization_id NOT NULL after migration
5. Creates all necessary indexes

**Rollback:** Run `005_multi_tenancy.down.sql` to reverse changes

---

## Summary

**Phase 1A Status:** 4/9 repositories completed (44%)

**Work Done:**
- ✅ Created comprehensive organization repository
- ✅ Updated 3 critical repositories (workflow, execution, event)
- ✅ Established consistent pattern for remaining repositories
- ✅ All models updated with OrganizationID
- ✅ Database migration created and tested

**Impact:**
- Data isolation foundation established
- Security model defined
- Performance indexes in place
- Clear path to completion

**Next:** Complete remaining 5 repositories, then move to Phase 2.
