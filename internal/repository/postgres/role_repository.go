package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// RoleRepository handles role database operations
type RoleRepository struct {
	db *sql.DB
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *sql.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// GetByID retrieves a role by ID
func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	role := &models.Role{}
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&role.ID, &role.Name, &role.Description,
		&role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return role, nil
}

// GetByName retrieves a role by name
func (r *RoleRepository) GetByName(ctx context.Context, name string) (*models.Role, error) {
	role := &models.Role{}
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE name = $1`

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&role.ID, &role.Name, &role.Description,
		&role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return role, nil
}

// List retrieves all roles
func (r *RoleRepository) List(ctx context.Context) ([]*models.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []*models.Role
	for rows.Next() {
		role := &models.Role{}
		err := rows.Scan(
			&role.ID, &role.Name, &role.Description,
			&role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetRolePermissions retrieves all permissions for a role
func (r *RoleRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]*models.Permission, error) {
	query := `
		SELECT p.id, p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		INNER JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.resource, p.action`

	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*models.Permission
	for rows.Next() {
		permission := &models.Permission{}
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.Resource,
			&permission.Action, &permission.Description, &permission.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// GrantPermission grants a permission to a role
func (r *RoleRepository) GrantPermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id, granted_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (role_id, permission_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, roleID, permissionID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to grant permission: %w", err)
	}

	return nil
}

// RevokePermission revokes a permission from a role
func (r *RoleRepository) RevokePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`

	result, err := r.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to revoke permission: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("permission grant not found")
	}

	return nil
}

// PermissionRepository handles permission database operations
type PermissionRepository struct {
	db *sql.DB
}

// NewPermissionRepository creates a new permission repository
func NewPermissionRepository(db *sql.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// GetByID retrieves a permission by ID
func (r *PermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	permission := &models.Permission{}
	query := `
		SELECT id, name, resource, action, description, created_at
		FROM permissions
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&permission.ID, &permission.Name, &permission.Resource,
		&permission.Action, &permission.Description, &permission.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return permission, nil
}

// GetByName retrieves a permission by name
func (r *PermissionRepository) GetByName(ctx context.Context, name string) (*models.Permission, error) {
	permission := &models.Permission{}
	query := `
		SELECT id, name, resource, action, description, created_at
		FROM permissions
		WHERE name = $1`

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&permission.ID, &permission.Name, &permission.Resource,
		&permission.Action, &permission.Description, &permission.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return permission, nil
}

// List retrieves all permissions
func (r *PermissionRepository) List(ctx context.Context) ([]*models.Permission, error) {
	query := `
		SELECT id, name, resource, action, description, created_at
		FROM permissions
		ORDER BY resource, action`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*models.Permission
	for rows.Next() {
		permission := &models.Permission{}
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.Resource,
			&permission.Action, &permission.Description, &permission.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}
