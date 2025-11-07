# Multi-Tenancy Implementation Guide

## Overview

This document describes the multi-tenancy implementation for the Intelligent Workflows system, transforming it from a single-organization system to a fully multi-tenant architecture.

## Architecture Design

### Core Concepts

1. **Organization**: The primary tenant entity that owns all workflows, executions, and related data
2. **Organization Users**: Users can belong to multiple organizations with different roles per organization
3. **Data Isolation**: All queries are filtered by `organization_id` to ensure complete tenant isolation
4. **Org-Scoped RBAC**: Roles and permissions are scoped per organization

### Database Schema Changes

#### New Tables

```sql
-- Core tenant entity
organizations (
    id UUID PK,
    name VARCHAR(255),
    slug VARCHAR(255) UNIQUE,
    description TEXT,
    settings JSONB,
    is_active BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    created_by UUID FK -> users
)

-- User-Organization membership with per-org roles
organization_users (
    id UUID PK,
    organization_id UUID FK -> organizations,
    user_id UUID FK -> users,
    role_id UUID FK -> roles,
    is_active BOOLEAN,
    joined_at TIMESTAMP,
    invited_by UUID FK -> users,
    UNIQUE(organization_id, user_id)
)
```

#### Modified Tables (Added organization_id)

- workflows
- workflow_executions
- step_executions
- rules
- events
- approval_requests
- context_cache
- workflow_schedules
- api_keys
- audit_log (nullable for system-level audits)

#### Indexes

All tables with `organization_id` have composite indexes:
- `(organization_id, <primary_lookup_field>)`
- Individual index on `organization_id`

## Implementation Status

### ✅ Completed

1. **Database Migrations**
   - Created `005_multi_tenancy.up.sql` and `005_multi_tenancy.down.sql`
   - Added organization tables
   - Added organization_id to all data tables
   - Created appropriate indexes
   - Migration script to move existing data to default organization

2. **Models**
   - Created `Organization` and `OrganizationUser` models
   - Added `OrganizationID` field to all data models:
     - Workflow
     - WorkflowExecution
     - StepExecution
     - Event
     - ApprovalRequest
     - ContextCache
     - AuditLog (nullable)
     - Rule
     - WorkflowSchedule
     - APIKey
   - Updated `Claims` struct to include `OrganizationID`

3. **Repositories**
   - Created `OrganizationRepository` with full CRUD operations
   - Includes user management methods:
     - AddUser, RemoveUser
     - GetOrganizationUsers
     - UpdateUserRole
     - CheckUserAccess

### 🚧 Remaining Work

#### 1. Repository Layer Updates (HIGH PRIORITY)

All repositories need to be updated to accept `organizationID` parameter and filter queries:

**Pattern to apply:**

```go
// OLD
func (r *WorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
    query := `SELECT ... FROM workflows WHERE id = $1`
    // ...
}

// NEW
func (r *WorkflowRepository) GetByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error) {
    query := `SELECT ... FROM workflows WHERE organization_id = $1 AND id = $2`
    // ...
}
```

**Repositories to update:**
- [x] organization_repository.go (created)
- [ ] workflow_repository.go
- [ ] execution_repository.go
- [ ] event_repository.go
- [ ] approval_repository.go
- [ ] schedule_repository.go
- [ ] api_key_repository.go
- [ ] audit_repository.go
- [ ] analytics_repository.go

**Key methods per repository:**
- Create: Add `organizationID` parameter and column
- GetByID: Add `organization_id` filter
- List: Add `organization_id` filter
- Update: Add `organization_id` filter for security
- Delete: Add `organization_id` filter for security

#### 2. JWT and Authentication (HIGH PRIORITY)

**File:** `pkg/auth/jwt.go`

Update JWT token generation to include organization context:

```go
// Add to Claims
type Claims struct {
    UserID         uuid.UUID
    OrganizationID uuid.UUID  // NEW
    Username       string
    Email          string
    Roles          []string
    Permissions    []string
    jwt.StandardClaims
}
```

**File:** `internal/services/auth_service.go`

- Update Login to select user's default/first organization
- Update token generation to include organization_id
- Add endpoint to switch organization context (new token with different org_id)

#### 3. Middleware (HIGH PRIORITY)

**New File:** `internal/api/rest/middleware/organization.go`

Create middleware to extract and validate organization context:

```go
func OrganizationContext() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract claims from JWT (set by auth middleware)
            claims := r.Context().Value("claims").(*models.Claims)

            // Verify user has access to organization
            orgID := claims.OrganizationID
            userID := claims.UserID

            // Add to context
            ctx := context.WithValue(r.Context(), "organization_id", orgID)
            ctx = context.WithValue(ctx, "user_id", userID)

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

#### 4. API Handlers (HIGH PRIORITY)

**All handler files need updates:**

1. Extract `organizationID` from context
2. Pass `organizationID` to all repository calls
3. Validate ownership before operations

**Example pattern:**

```go
// OLD
func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    workflow, err := h.workflowRepo.GetByID(r.Context(), uuid.MustParse(id))
    // ...
}

// NEW
func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value("organization_id").(uuid.UUID)
    id := chi.URLParam(r, "id")
    workflow, err := h.workflowRepo.GetByID(r.Context(), orgID, uuid.MustParse(id))
    // ...
}
```

**New Handler:** `internal/api/rest/handlers/organization.go`

Create organization management endpoints:
- POST /api/v1/organizations - Create organization
- GET /api/v1/organizations - List user's organizations
- GET /api/v1/organizations/:id - Get organization details
- PUT /api/v1/organizations/:id - Update organization
- DELETE /api/v1/organizations/:id - Delete organization
- POST /api/v1/organizations/:id/users - Add user to organization
- GET /api/v1/organizations/:id/users - List organization users
- DELETE /api/v1/organizations/:id/users/:user_id - Remove user
- PUT /api/v1/organizations/:id/users/:user_id - Update user role

#### 5. Workflow Engine (CRITICAL)

**File:** `internal/engine/executor.go`

Update workflow execution to be org-aware:

```go
func (e *Executor) ExecuteWorkflow(
    ctx context.Context,
    organizationID uuid.UUID,  // ADD
    workflowID uuid.UUID,
    event *models.Event,
) (*models.WorkflowExecution, error) {
    // Add organization_id to execution record
    execution := &models.WorkflowExecution{
        ID:             uuid.New(),
        OrganizationID: organizationID,  // ADD
        WorkflowID:     workflowID,
        // ...
    }
}
```

**File:** `internal/engine/event_router.go`

Update event routing to filter by organization:

```go
func (r *EventRouter) RouteEvent(
    ctx context.Context,
    organizationID uuid.UUID,  // ADD
    event *models.Event,
) error {
    // Only match workflows within the same organization
    workflows, err := r.workflowRepo.ListByEventType(
        ctx,
        organizationID,  // ADD
        event.EventType,
    )
}
```

#### 6. Background Workers (CRITICAL)

All workers need to process per-organization:

**Files to update:**
- `internal/workers/approval_expiration.go`
- `internal/workers/workflow_resumer.go`
- `internal/workers/timeout_enforcer.go`
- `internal/workers/scheduler.go`

**Pattern:**

```go
// OLD
func (w *TimeoutEnforcer) enforceTimeouts(ctx context.Context) error {
    executions, err := w.executionRepo.GetTimedOut(ctx)
    // ...
}

// NEW - Process all organizations
func (w *TimeoutEnforcer) enforceTimeouts(ctx context.Context) error {
    orgs, err := w.orgRepo.List(ctx, 1, 1000)
    for _, org := range orgs {
        executions, err := w.executionRepo.GetTimedOut(ctx, org.ID)
        // ...
    }
}
```

#### 7. Seed Data (MEDIUM PRIORITY)

**File:** `internal/seeds/seed_rbac.go`

Update to create default organization and assign users:

```go
func SeedRBAC(db *sql.DB) error {
    // Existing role/permission seeding
    // ...

    // NEW: Create default organization
    defaultOrg := createDefaultOrganization(db)

    // NEW: Assign admin user to default org
    assignUserToOrganization(db, adminUserID, defaultOrg.ID, adminRoleID)
}
```

#### 8. Router Updates (MEDIUM PRIORITY)

**File:** `internal/api/rest/router.go`

Add organization middleware and routes:

```go
// Protected routes with organization context
r.Group(func(r chi.Router) {
    r.Use(middleware.JWTAuth(jwtSecret))
    r.Use(middleware.OrganizationContext())  // NEW

    // Existing routes...

    // NEW: Organization routes
    r.Route("/organizations", func(r chi.Router) {
        r.Get("/", handlers.ListOrganizations)
        r.Post("/", handlers.CreateOrganization)
        r.Get("/{id}", handlers.GetOrganization)
        r.Put("/{id}", handlers.UpdateOrganization)
        r.Delete("/{id}", handlers.DeleteOrganization)
        r.Post("/{id}/users", handlers.AddUserToOrganization)
        r.Get("/{id}/users", handlers.ListOrganizationUsers)
        // ...
    })
})
```

#### 9. Testing (HIGH PRIORITY)

Create integration tests to verify:
- Data isolation between organizations
- Users cannot access other org's data
- Workflows only trigger within same organization
- API endpoints enforce organization filtering

#### 10. Documentation (MEDIUM PRIORITY)

Update documentation:
- API documentation (OpenAPI spec)
- Authentication guide
- Migration guide for existing deployments
- Multi-tenancy architecture document

## Migration Strategy

### For Existing Deployments

1. **Backup Database**
   ```bash
   pg_dump -U postgres -d workflows > backup.sql
   ```

2. **Run Migration**
   ```bash
   migrate -path ./migrations/postgres -database "postgresql://..." up
   ```

   This will:
   - Create organizations and organization_users tables
   - Add organization_id columns to all tables
   - Create "Default Organization" with slug "default"
   - Assign all existing users to default organization
   - Migrate all existing data to default organization

3. **Deploy Updated Application**
   - All existing API calls continue to work
   - All users operate within "Default Organization"
   - System is now multi-tenant ready

4. **Create Additional Organizations**
   - Use organization API endpoints
   - Invite users to new organizations
   - Users can switch between organizations via JWT token refresh

## Security Considerations

### Data Isolation

1. **Repository Layer**: Every query MUST include `organization_id` filter
2. **Double Verification**:
   - Middleware validates user belongs to organization
   - Repository validates data belongs to organization
3. **No Cross-Org Access**: Workflows in Org A cannot trigger workflows in Org B

### RBAC Updates

Current: Global roles (admin applies to entire system)
New: Org-scoped roles (admin of Org A ≠ admin of Org B)

### API Keys

- API keys are now scoped to an organization
- Must include organization_id when creating API keys
- API key authentication sets organization context

## Performance Considerations

### Indexes

All queries by organization_id are indexed:
- `(organization_id, workflow_id)`
- `(organization_id, status)` for executions
- `(organization_id, event_type)` for events

### Query Patterns

```sql
-- Efficient: Uses composite index
SELECT * FROM workflows
WHERE organization_id = $1 AND workflow_id = $2;

-- Avoid: Missing organization filter
SELECT * FROM workflows WHERE workflow_id = $1;
```

### Caching

Consider caching organization data:
- Organization details
- User-organization memberships
- User roles per organization

## API Changes

### Breaking Changes

None. The migration adds organization_id to JWT claims, but existing endpoints continue to work with the default organization.

### New Endpoints

```
POST   /api/v1/organizations
GET    /api/v1/organizations
GET    /api/v1/organizations/:id
PUT    /api/v1/organizations/:id
DELETE /api/v1/organizations/:id
POST   /api/v1/organizations/:id/users
GET    /api/v1/organizations/:id/users
PUT    /api/v1/organizations/:id/users/:user_id
DELETE /api/v1/organizations/:id/users/:user_id
POST   /api/v1/auth/switch-organization (refresh token with different org)
```

### Request Headers

No changes. Organization context comes from JWT token.

### Response Bodies

All resources now include `organization_id`:

```json
{
  "id": "uuid",
  "organization_id": "uuid",
  "workflow_id": "fraud-detection",
  "name": "Fraud Detection",
  ...
}
```

## Rollback Plan

If issues arise:

1. **Stop Application**
2. **Restore Backup**
   ```bash
   psql -U postgres -d workflows < backup.sql
   ```
3. **Deploy Previous Version**

Or, run down migration:
```bash
migrate -path ./migrations/postgres -database "postgresql://..." down 1
```

This removes multi-tenancy schema changes.

## Next Steps

1. Complete repository layer updates (all queries must filter by org_id)
2. Update JWT service and middleware
3. Update all API handlers
4. Update workflow engine and workers
5. Add organization management endpoints
6. Write integration tests
7. Update documentation
8. Deploy to staging
9. Run full test suite
10. Deploy to production

## References

- Database Migration: `migrations/postgres/005_multi_tenancy.up.sql`
- Models: `internal/models/organization.go`
- Repository: `internal/repository/postgres/organization_repository.go`
