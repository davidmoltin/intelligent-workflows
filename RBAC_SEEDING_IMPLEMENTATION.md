# RBAC Permission Seeding Implementation

## Summary

This implementation adds a comprehensive, idempotent RBAC (Role-Based Access Control) permission seeding system to the Intelligent Workflows application.

## Problem Statement

The RBAC tables existed but lacked:
- A programmatic way to seed permissions outside of migrations
- Ability to re-seed permissions in existing databases
- Verification mechanism for permission integrity
- Easy way to add new permissions without writing new migrations
- Default admin user creation for initial setup

## Solution

### 1. Seed Package (`internal/seeds/`)

Created a dedicated seed package with:
- **Idempotent seeding functions** - Can be run multiple times safely
- **Role definitions** - 5 default roles (admin, workflow_manager, workflow_viewer, approver, agent)
- **Permission definitions** - 23 permissions covering all resources
- **Role-permission mappings** - Pre-configured access levels for each role
- **Verification functions** - Check data integrity
- **Statistics functions** - View current RBAC state

**Key Features:**
- Uses `ON CONFLICT DO UPDATE/NOTHING` for idempotency
- Transaction-based for data consistency
- Optional default admin user creation
- Comprehensive logging and error handling

### 2. CLI Seed Command (`cmd/seed/`)

Created a command-line tool for seeding operations:

```bash
# Seed roles and permissions
./bin/seed

# Seed with default admin user
./bin/seed --users

# Verify RBAC data
./bin/seed --verify

# Show statistics
./bin/seed --stats
```

**Features:**
- Environment variable configuration
- Multiple operation modes
- Clear output with progress indicators
- Exit codes for CI/CD integration

### 3. Enhanced Migration (`migrations/postgres/002_auth_system.up.sql`)

Updated the migration file with:
- Idempotent INSERT statements using `ON CONFLICT`
- **7 new permissions:**
  - `execution:pause` - Pause running executions
  - `execution:resume` - Resume paused executions
  - `role:create` - Create roles (was `role:manage`)
  - `role:read` - View roles
  - `role:update` - Update roles
  - `role:delete` - Delete roles
  - `role:assign` - Assign roles to users
- Updated role mappings to include new permissions

**Benefits:**
- Can be re-run safely without creating duplicates
- Automatically seeds on fresh database initialization
- Keeps seed data in sync with migrations

### 4. Makefile Commands

Added convenient Make targets:

```bash
make seed-rbac          # Seed RBAC data
make seed-rbac-users    # Seed with admin user
make seed-verify        # Verify data integrity
make seed-stats         # Show statistics
```

### 5. Comprehensive Documentation

Created `docs/RBAC_SEEDING.md` with:
- Complete usage guide
- Permission reference
- Troubleshooting steps
- CI/CD integration examples
- Best practices
- FAQ section

### 6. Test Script

Created `scripts/test-rbac-seed.sh` for automated testing:
- Tests all seeding operations
- Verifies idempotency
- Checks database state directly
- Validates new permissions
- Color-coded output

## Permission Structure

### Roles Summary

| Role | Permissions | Purpose |
|------|-------------|---------|
| admin | 23 (all) | Full system access |
| workflow_manager | 11 | Create/manage workflows, control executions |
| workflow_viewer | 4 | Read-only access to workflows and executions |
| approver | 4 | Handle approval requests |
| agent | 4 | AI agent with limited API access |

### New Permissions Added

1. **execution:pause** - Allows pausing running workflow executions
2. **execution:resume** - Allows resuming paused executions
3. **role:create** - Create new custom roles
4. **role:read** - View roles and their permissions
5. **role:update** - Modify existing roles
6. **role:delete** - Delete custom roles
7. **role:assign** - Assign roles to users

### Permission Format

Permissions follow the format: `resource:action`

**Resources:** workflow, execution, approval, event, user, role
**Actions:** create, read, update, delete, execute, cancel, pause, resume, approve, reject, assign

## Files Changed/Created

### New Files
- `internal/seeds/rbac_seed.go` - Core seeding logic (580 lines)
- `cmd/seed/main.go` - CLI command (114 lines)
- `docs/RBAC_SEEDING.md` - Comprehensive documentation (550+ lines)
- `scripts/test-rbac-seed.sh` - Automated test script (180 lines)
- `RBAC_SEEDING_IMPLEMENTATION.md` - This implementation summary

### Modified Files
- `migrations/postgres/002_auth_system.up.sql` - Added idempotency and new permissions
- `Makefile` - Added seed commands to .PHONY and new targets

## Usage Examples

### Initial Setup

```bash
# Start services
make docker-up

# Seed RBAC data with admin user
make seed-rbac-users

# Verify
make seed-verify
```

**Output includes admin credentials:**
```
⚠️  DEFAULT ADMIN CREDENTIALS:
   Username: admin
   Email:    admin@example.com
   Password: admin123

⚠️  IMPORTANT: Change the default password after first login!
```

### Adding New Permissions

When adding a new feature:

1. Update `internal/seeds/rbac_seed.go`:
```go
{"newfeature:create", "newfeature", "create", "Create new feature"},
```

2. Add to appropriate roles:
```go
{
    RoleName: "workflow_manager",
    PermissionNames: []string{
        // ... existing
        "newfeature:create",
    },
}
```

3. Run seeding:
```bash
make seed-rbac
make seed-verify
```

### Verifying Production

```bash
# Check current state
DB_HOST=prod.example.com make seed-stats

# Verify integrity
DB_HOST=prod.example.com make seed-verify

# Fix if needed (safe, idempotent)
DB_HOST=prod.example.com make seed-rbac
```

## Technical Details

### Idempotency Strategy

**Roles & Permissions:**
```sql
INSERT INTO roles (name, description)
VALUES ('admin', 'Administrator')
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description;
```

**Role-Permission Mappings:**
```sql
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;
```

### Transaction Safety

All seeding operations run in a single transaction:
```go
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()

// Seed operations...

tx.Commit()
```

If any operation fails, all changes are rolled back.

### Verification Logic

The verify function checks:
1. All expected roles exist
2. All expected permissions exist
3. Each role has correct number of permissions
4. Admin role has all permissions

### Default User Creation

When `--users` flag is used:
- Checks if user already exists (skips if found)
- Hashes password using bcrypt
- Creates user with verified status
- Assigns specified roles
- Transaction ensures atomicity

## Testing

### Manual Testing

```bash
# Run the test script
./scripts/test-rbac-seed.sh
```

### Test Coverage

The test script validates:
- ✓ Build succeeds
- ✓ Initial state is correct
- ✓ Verification passes
- ✓ Re-seeding is idempotent
- ✓ Correct number of roles (5)
- ✓ Correct number of permissions (23)
- ✓ Admin has all permissions
- ✓ Default user creation works
- ✓ User role assignment works
- ✓ New permissions exist

### CI/CD Integration

Add to your pipeline:

```yaml
# .github/workflows/ci.yml
- name: Verify RBAC Data
  run: make seed-verify

- name: Test RBAC Seeding
  run: ./scripts/test-rbac-seed.sh
```

## Migration Path

### For Fresh Installations

No action needed. The migration runs automatically and seeds all data.

### For Existing Installations

```bash
# Option 1: Use seed command (recommended)
make seed-rbac

# Option 2: Re-run migration (if preferred)
docker-compose exec -T postgres psql -U postgres -d workflows \
  < migrations/postgres/002_auth_system.up.sql
```

Both are safe and idempotent.

## Benefits

### For Developers
- ✅ Easy to add new permissions
- ✅ No need to write migrations for permission changes
- ✅ Clear verification of RBAC state
- ✅ Type-safe permission definitions in Go

### For DevOps
- ✅ Idempotent operations safe for automation
- ✅ Environment variable configuration
- ✅ Exit codes for CI/CD
- ✅ Clear error messages

### For Testing
- ✅ Automated test script
- ✅ Quick database seeding for tests
- ✅ Verification functions
- ✅ Statistics for debugging

### For Production
- ✅ Safe to run on existing databases
- ✅ Transaction-based consistency
- ✅ Comprehensive logging
- ✅ Rollback on failure

## Security Considerations

### Default Admin User

- **Only created with `--users` flag** (not automatic)
- **Weak default password** (`admin123`) - Must be changed
- **Warning displayed** prominently after creation
- **Email domain** is `example.com` - Should be updated

**Production Recommendation:**
```bash
# DON'T use default user in production
# Instead, create admin via API after deployment
```

### Permission Audit

All seeding operations are logged:
```
[RBAC Seed] ✓ Role: admin
[RBAC Seed] ✓ Permission: workflow:create
[RBAC Seed] ✓ Admin: all permissions
```

### Database Credentials

Environment variables should use secrets in production:
```bash
# Use secrets manager, not environment variables
DB_PASSWORD=$(vault read -field=password secret/db)
```

## Future Enhancements

### Possible Improvements

1. **Permission Hierarchy**
   - Parent/child permission relationships
   - Inherited permissions

2. **Role Templates**
   - Pre-configured role bundles
   - Industry-specific templates

3. **Permission Groups**
   - Logical grouping of related permissions
   - Easier bulk assignment

4. **Audit Trail**
   - Track who granted/revoked permissions
   - Permission change history

5. **Dynamic Permissions**
   - Auto-discovery from API routes
   - Code annotation-based permissions

6. **Role Inheritance**
   - Base roles that other roles extend
   - Hierarchical role structure

## Conclusion

This implementation provides a robust, maintainable, and production-ready RBAC seeding system. It solves the original problem of missing default permissions while adding valuable features for development, testing, and operations.

The system is:
- ✅ **Idempotent** - Safe to run multiple times
- ✅ **Comprehensive** - Covers all resources and actions
- ✅ **Well-documented** - Clear guides and examples
- ✅ **Well-tested** - Automated test suite
- ✅ **Production-ready** - Transaction-safe and logged
- ✅ **Developer-friendly** - Easy to extend and maintain

## Quick Reference

```bash
# Development
make seed-rbac              # Seed RBAC data
make seed-rbac-users        # Seed with admin user
make seed-verify            # Verify data
make seed-stats             # Show statistics

# Go commands
go run ./cmd/seed           # Seed
go run ./cmd/seed --users   # Seed with users
go run ./cmd/seed --verify  # Verify
go run ./cmd/seed --stats   # Statistics

# Testing
./scripts/test-rbac-seed.sh # Run full test suite

# Build
go build -o bin/seed ./cmd/seed
```

## Support

For questions or issues:
- See documentation: `docs/RBAC_SEEDING.md`
- Run verification: `make seed-verify`
- Check statistics: `make seed-stats`
- Review logs for error details
