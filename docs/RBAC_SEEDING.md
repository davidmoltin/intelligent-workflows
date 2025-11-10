# RBAC Permission Seeding Guide

This document describes the RBAC (Role-Based Access Control) permission seeding system for the Intelligent Workflows application.

## Overview

The RBAC seeding system provides a robust, idempotent way to seed and manage default roles, permissions, and role-permission mappings. It can be run multiple times safely without creating duplicates.

## Architecture

### Components

1. **Seed Package** (`internal/seeds/rbac_seed.go`)
   - Core seeding logic
   - Idempotent operations using `ON CONFLICT`
   - Verification and statistics functions

2. **CLI Command** (`cmd/seed/main.go`)
   - Command-line interface for seeding operations
   - Multiple operation modes (seed, verify, stats)
   - Environment variable configuration

3. **Migration File** (`migrations/postgres/002_auth_system.up.sql`)
   - Updated with idempotent INSERT statements
   - Includes new permissions (execution:pause, execution:resume, granular role permissions)
   - Runs automatically on fresh database initialization

## Default RBAC Structure

### Roles (5 total)

| Role | Description | Use Case |
|------|-------------|----------|
| **admin** | Administrator with full system access | System administrators |
| **workflow_manager** | Can manage workflows and view executions | Power users who create and manage workflows |
| **workflow_viewer** | Can view workflows and executions | Read-only users monitoring workflows |
| **approver** | Can approve/reject approval requests | Users who handle approval steps |
| **agent** | AI agent with limited API access | Autonomous AI agents executing workflows |

### Permissions (23 total)

#### Workflow Permissions (5)
- `workflow:create` - Create new workflows
- `workflow:read` - View workflows
- `workflow:update` - Update workflows
- `workflow:delete` - Delete workflows
- `workflow:execute` - Execute workflows

#### Execution Permissions (4)
- `execution:read` - View workflow executions
- `execution:cancel` - Cancel running executions
- `execution:pause` - Pause running executions *(new)*
- `execution:resume` - Resume paused executions *(new)*

#### Approval Permissions (3)
- `approval:read` - View approval requests
- `approval:approve` - Approve requests
- `approval:reject` - Reject requests

#### Event Permissions (2)
- `event:create` - Create events
- `event:read` - View events

#### User Management Permissions (4)
- `user:create` - Create users
- `user:read` - View users
- `user:update` - Update users
- `user:delete` - Delete users

#### Role Management Permissions (5) *(new - granular)*
- `role:create` - Create roles
- `role:read` - View roles
- `role:update` - Update roles
- `role:delete` - Delete roles
- `role:assign` - Assign roles to users

### Role-Permission Mappings

| Role | Permission Count | Key Permissions |
|------|------------------|-----------------|
| **admin** | All (23) | Complete system access |
| **workflow_manager** | 11 | workflow:*, execution:*, event:read, approval:read |
| **workflow_viewer** | 4 | Read-only access to workflows, executions, events, approvals |
| **approver** | 4 | Approval workflow + execution viewing |
| **agent** | 4 | Workflow execution + event handling |

## Usage

### Prerequisites

- PostgreSQL database running (via Docker or local installation)
- Go 1.19+ installed
- Database environment variables configured (or defaults will be used)

### Environment Variables

The seed command uses the following environment variables:

```bash
DB_HOST=localhost      # Database host
DB_PORT=5432          # Database port
DB_USER=postgres      # Database user
DB_PASSWORD=postgres  # Database password
DB_NAME=workflows     # Database name
DB_SSL_MODE=disable   # SSL mode
```

### Command Options

#### 1. Seed RBAC Data (Default)

Seeds roles, permissions, and role-permission mappings:

```bash
# Using Go
go run ./cmd/seed

# Using Make
make seed-rbac

# Using compiled binary
./bin/seed
```

**Output:**
```
[RBAC Seed] Connecting to database...
[RBAC Seed] ✓ Database connection established

=== Seeding Mode ===
[RBAC Seed] Note: Existing data will be preserved (use --force to update)
[RBAC Seed] Starting RBAC seeding...
[RBAC Seed] Seeding roles...
[RBAC Seed]   ✓ Role: admin
[RBAC Seed]   ✓ Role: workflow_manager
[RBAC Seed]   ✓ Role: workflow_viewer
[RBAC Seed]   ✓ Role: approver
[RBAC Seed]   ✓ Role: agent
[RBAC Seed] Seeding permissions...
[RBAC Seed]   ✓ Permission: workflow:create
[RBAC Seed]   ✓ Permission: workflow:read
...
[RBAC Seed] Seeding role-permission mappings...
[RBAC Seed]   ✓ Admin: all permissions
[RBAC Seed]   ✓ workflow_manager: 11 permissions
...
[RBAC Seed] RBAC seeding completed successfully!
```

#### 2. Seed with Default Admin User

Seeds RBAC data AND creates a default admin user:

```bash
# Using Go
go run ./cmd/seed --users

# Using Make
make seed-rbac-users

# Using compiled binary
./bin/seed --users
```

**Default Admin Credentials:**
```
Username: admin
Email:    admin@example.com
Password: admin123
```

⚠️ **IMPORTANT:** Change the default password immediately after first login!

#### 3. Verify RBAC Data

Checks if all expected roles, permissions, and mappings exist:

```bash
# Using Go
go run ./cmd/seed --verify

# Using Make
make seed-verify

# Using compiled binary
./bin/seed --verify
```

**Output:**
```
=== Verification Mode ===
[RBAC Seed] Verifying RBAC data...
[RBAC Seed]   ✓ Role: admin
[RBAC Seed]   ✓ Role: workflow_manager
...
[RBAC Seed]   ✓ Permission: workflow:create
[RBAC Seed]   ✓ Permission: workflow:read
...
[RBAC Seed] Verifying role-permission mappings...
[RBAC Seed]   ✓ Admin: 23 permissions
[RBAC Seed]   ✓ workflow_manager: 11 permissions
...
[RBAC Seed] ✓ RBAC verification completed successfully!
```

#### 4. Show Statistics

Displays statistics about RBAC data:

```bash
# Using Go
go run ./cmd/seed --stats

# Using Make
make seed-stats

# Using compiled binary
./bin/seed --stats
```

**Output:**
```
=== RBAC Statistics ===
Roles: 5
Permissions: 23
Role-Permission Mappings: 50
Users: 1
User-Role Assignments: 1

=== Permissions by Role ===
admin: 23 permissions
agent: 4 permissions
approver: 4 permissions
workflow_manager: 11 permissions
workflow_viewer: 4 permissions
========================
```

### Advanced Usage

#### Force Update Existing Data

By default, the seed command preserves existing data. Use `--force` to update:

```bash
go run ./cmd/seed --force
```

This will:
- Update role descriptions if changed
- Update permission details if changed
- Add new permissions for existing roles
- Preserve user data and assignments

## Integration with Migrations

### Automatic Seeding on Fresh Install

When you initialize a fresh database using Docker:

```bash
make docker-up
```

The migration file `002_auth_system.up.sql` will automatically run and seed all RBAC data.

### Manual Migration Run

If you need to run migrations manually:

```bash
# Start PostgreSQL
make docker-up

# Run migrations
docker-compose exec -T postgres psql -U postgres -d workflows -f /docker-entrypoint-initdb.d/002_auth_system.up.sql
```

### Re-running Migrations

The migration file is now idempotent and can be re-run safely:

```bash
# Re-run the auth system migration
docker-compose exec -T postgres psql -U postgres -d workflows < migrations/postgres/002_auth_system.up.sql
```

This will:
- Skip existing roles (ON CONFLICT DO UPDATE)
- Skip existing permissions (ON CONFLICT DO UPDATE)
- Skip existing role-permission mappings (ON CONFLICT DO NOTHING)

## Workflow Examples

### Initial Setup

```bash
# 1. Start services
make docker-up

# 2. Verify RBAC data was seeded
make seed-verify

# 3. (Optional) Create default admin user
make seed-rbac-users
```

### Adding New Permissions

When you add new features that require new permissions:

1. **Update the seed package** (`internal/seeds/rbac_seed.go`):
```go
// Add to GetDefaultPermissions()
{"newfeature:create", "newfeature", "create", "Create new feature items"},
```

2. **Update role mappings** (if needed):
```go
// Add to GetDefaultRolePermissions()
{
    RoleName: "workflow_manager",
    PermissionNames: []string{
        // ... existing permissions
        "newfeature:create",
    },
}
```

3. **Run the seed command**:
```bash
make seed-rbac
```

4. **Verify**:
```bash
make seed-verify
make seed-stats
```

### Troubleshooting Missing Permissions

If you discover missing permissions in production:

```bash
# 1. Check current state
make seed-stats

# 2. Verify what's missing
make seed-verify

# 3. Re-seed (idempotent, safe to run)
make seed-rbac

# 4. Verify fix
make seed-verify
```

## CI/CD Integration

### Pre-deployment Verification

Add to your CI/CD pipeline:

```yaml
- name: Verify RBAC Seed Data
  run: make seed-verify
```

### Post-deployment Seeding

For new environments:

```yaml
- name: Seed RBAC Data
  run: make seed-rbac
  env:
    DB_HOST: ${{ secrets.DB_HOST }}
    DB_USER: ${{ secrets.DB_USER }}
    DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
    DB_NAME: ${{ secrets.DB_NAME }}
```

## Best Practices

### 1. Version Control
- Always update both the seed package AND migration file when adding permissions
- Keep the two in sync to ensure consistency between fresh installs and updates

### 2. Testing
- Run `make seed-verify` after any RBAC changes
- Test with a fresh database to ensure migrations work correctly
- Test with an existing database to ensure idempotency

### 3. Documentation
- Document new permissions in this file
- Update API documentation when adding new protected endpoints
- Communicate permission changes to your team

### 4. Security
- Never commit default user credentials to version control
- Change default passwords immediately in production
- Use strong, unique passwords for production admin accounts
- Audit permission changes in code review

### 5. Backwards Compatibility
- When adding new permissions, consider existing users and roles
- Use additive changes when possible (add permissions, don't remove)
- Communicate breaking changes clearly

## Database Schema

### Tables

```sql
-- Roles
CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Permissions
CREATE TABLE permissions (
    id UUID PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP
);

-- Role-Permission Mappings
CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id),
    permission_id UUID REFERENCES permissions(id),
    granted_at TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);

-- User-Role Assignments
CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id),
    role_id UUID REFERENCES roles(id),
    assigned_at TIMESTAMP,
    assigned_by UUID REFERENCES users(id),
    PRIMARY KEY (user_id, role_id)
);
```

## FAQ

### Q: Can I run the seed command multiple times?
**A:** Yes! The seed command is idempotent and safe to run multiple times. It uses `ON CONFLICT` clauses to avoid duplicates.

### Q: Will seeding overwrite my custom roles?
**A:** No. Seeding only affects the default roles and permissions. Custom roles you've created are not touched.

### Q: How do I add a new permission without a migration?
**A:** Use the seed command:
1. Update `internal/seeds/rbac_seed.go`
2. Run `make seed-rbac`
3. Run `make seed-verify`

### Q: What happens if I delete a default role?
**A:** Running the seed command will recreate it with default permissions.

### Q: Can I modify default role permissions?
**A:** Yes, but changes will be reverted if you re-run the seed command. For permanent changes, update both the seed package and migration file.

### Q: How do I reset RBAC data to defaults?
**A:**
```bash
# 1. Delete custom data (careful!)
# 2. Re-run migrations or seed command
make seed-rbac --force
```

## Support

For issues or questions:
- Check the verification output: `make seed-verify`
- Review the statistics: `make seed-stats`
- Check application logs for permission errors
- Verify JWT tokens contain expected roles and permissions

## Related Documentation

- [Authentication Guide](./api/AUTHENTICATION.md)
- [API Documentation](./api/README.md)
- [Database Migrations](../migrations/postgres/README.md)
- [Architecture Overview](../ARCHITECTURE.md)
