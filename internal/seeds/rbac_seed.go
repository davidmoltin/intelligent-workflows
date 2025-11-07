package seeds

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RBACSeeder handles seeding of roles, permissions, and default users
type RBACSeeder struct {
	db *sql.DB
}

// NewRBACSeeder creates a new RBAC seeder
func NewRBACSeeder(db *sql.DB) *RBACSeeder {
	return &RBACSeeder{db: db}
}

// Role represents a role definition
type Role struct {
	Name        string
	Description string
}

// Permission represents a permission definition
type Permission struct {
	Name        string
	Resource    string
	Action      string
	Description string
}

// RolePermission represents a role-permission mapping
type RolePermission struct {
	RoleName        string
	PermissionNames []string
}

// DefaultUser represents a default user to create
type DefaultUser struct {
	Username  string
	Email     string
	Password  string
	FirstName string
	LastName  string
	Roles     []string
}

// GetDefaultRoles returns the default roles to seed
func GetDefaultRoles() []Role {
	return []Role{
		{
			Name:        "admin",
			Description: "Administrator with full system access",
		},
		{
			Name:        "workflow_manager",
			Description: "Can manage workflows and view executions",
		},
		{
			Name:        "workflow_viewer",
			Description: "Can view workflows and executions",
		},
		{
			Name:        "approver",
			Description: "Can approve/reject approval requests",
		},
		{
			Name:        "agent",
			Description: "AI agent with limited API access",
		},
	}
}

// GetDefaultPermissions returns the default permissions to seed
func GetDefaultPermissions() []Permission {
	return []Permission{
		// Workflow permissions
		{"workflow:create", "workflow", "create", "Create new workflows"},
		{"workflow:read", "workflow", "read", "View workflows"},
		{"workflow:update", "workflow", "update", "Update workflows"},
		{"workflow:delete", "workflow", "delete", "Delete workflows"},
		{"workflow:execute", "workflow", "execute", "Execute workflows"},

		// Execution permissions
		{"execution:read", "execution", "read", "View workflow executions"},
		{"execution:cancel", "execution", "cancel", "Cancel running executions"},
		{"execution:pause", "execution", "pause", "Pause running executions"},
		{"execution:resume", "execution", "resume", "Resume paused executions"},

		// Approval permissions
		{"approval:read", "approval", "read", "View approval requests"},
		{"approval:approve", "approval", "approve", "Approve requests"},
		{"approval:reject", "approval", "reject", "Reject requests"},

		// Event permissions
		{"event:create", "event", "create", "Create events"},
		{"event:read", "event", "read", "View events"},

		// User management permissions
		{"user:create", "user", "create", "Create users"},
		{"user:read", "user", "read", "View users"},
		{"user:update", "user", "update", "Update users"},
		{"user:delete", "user", "delete", "Delete users"},

		// Role management permissions
		{"role:create", "role", "create", "Create roles"},
		{"role:read", "role", "read", "View roles"},
		{"role:update", "role", "update", "Update roles"},
		{"role:delete", "role", "delete", "Delete roles"},
		{"role:assign", "role", "assign", "Assign roles to users"},
	}
}

// GetDefaultRolePermissions returns the default role-permission mappings
func GetDefaultRolePermissions() []RolePermission {
	return []RolePermission{
		{
			RoleName: "admin",
			// Admin gets all permissions (will be dynamically assigned)
			PermissionNames: []string{},
		},
		{
			RoleName: "workflow_manager",
			PermissionNames: []string{
				"workflow:create", "workflow:read", "workflow:update", "workflow:delete", "workflow:execute",
				"execution:read", "execution:cancel", "execution:pause", "execution:resume",
				"event:read", "approval:read",
			},
		},
		{
			RoleName: "workflow_viewer",
			PermissionNames: []string{
				"workflow:read", "execution:read", "event:read", "approval:read",
			},
		},
		{
			RoleName: "approver",
			PermissionNames: []string{
				"approval:read", "approval:approve", "approval:reject", "execution:read",
			},
		},
		{
			RoleName: "agent",
			PermissionNames: []string{
				"workflow:execute", "event:create", "event:read", "execution:read",
			},
		},
	}
}

// GetDefaultUsers returns default users to create (optional, only admin)
func GetDefaultUsers() []DefaultUser {
	return []DefaultUser{
		{
			Username:  "admin",
			Email:     "admin@example.com",
			Password:  "admin123", // Should be changed on first login
			FirstName: "System",
			LastName:  "Administrator",
			Roles:     []string{"admin"},
		},
	}
}

// SeedAll seeds all RBAC data (roles, permissions, mappings, and optionally users)
func (s *RBACSeeder) SeedAll(ctx context.Context, seedUsers bool) error {
	log.Println("Starting RBAC seeding...")

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Seed roles
	if err := s.seedRoles(ctx, tx); err != nil {
		return fmt.Errorf("failed to seed roles: %w", err)
	}

	// Seed permissions
	if err := s.seedPermissions(ctx, tx); err != nil {
		return fmt.Errorf("failed to seed permissions: %w", err)
	}

	// Seed role-permission mappings
	if err := s.seedRolePermissions(ctx, tx); err != nil {
		return fmt.Errorf("failed to seed role-permissions: %w", err)
	}

	// Optionally seed default users
	if seedUsers {
		if err := s.seedDefaultUsers(ctx, tx); err != nil {
			return fmt.Errorf("failed to seed default users: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("RBAC seeding completed successfully!")
	return nil
}

// seedRoles seeds default roles
func (s *RBACSeeder) seedRoles(ctx context.Context, tx *sql.Tx) error {
	log.Println("Seeding roles...")

	roles := GetDefaultRoles()
	for _, role := range roles {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO roles (name, description, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
			ON CONFLICT (name) DO UPDATE
			SET description = EXCLUDED.description,
			    updated_at = NOW()
		`, role.Name, role.Description)

		if err != nil {
			return fmt.Errorf("failed to insert role %s: %w", role.Name, err)
		}
		log.Printf("  ✓ Role: %s", role.Name)
	}

	return nil
}

// seedPermissions seeds default permissions
func (s *RBACSeeder) seedPermissions(ctx context.Context, tx *sql.Tx) error {
	log.Println("Seeding permissions...")

	permissions := GetDefaultPermissions()
	for _, perm := range permissions {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO permissions (name, resource, action, description, created_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (name) DO UPDATE
			SET resource = EXCLUDED.resource,
			    action = EXCLUDED.action,
			    description = EXCLUDED.description
		`, perm.Name, perm.Resource, perm.Action, perm.Description)

		if err != nil {
			return fmt.Errorf("failed to insert permission %s: %w", perm.Name, err)
		}
		log.Printf("  ✓ Permission: %s", perm.Name)
	}

	return nil
}

// seedRolePermissions seeds role-permission mappings
func (s *RBACSeeder) seedRolePermissions(ctx context.Context, tx *sql.Tx) error {
	log.Println("Seeding role-permission mappings...")

	// First, handle admin role - give all permissions
	_, err := tx.ExecContext(ctx, `
		INSERT INTO role_permissions (role_id, permission_id, granted_at)
		SELECT r.id, p.id, NOW()
		FROM roles r
		CROSS JOIN permissions p
		WHERE r.name = 'admin'
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to assign permissions to admin role: %w", err)
	}
	log.Printf("  ✓ Admin: all permissions")

	// Handle other roles
	roleMappings := GetDefaultRolePermissions()
	for _, mapping := range roleMappings {
		if mapping.RoleName == "admin" || len(mapping.PermissionNames) == 0 {
			continue // Already handled admin
		}

		// Build the permissions list for SQL IN clause
		permCount := 0
		for _, permName := range mapping.PermissionNames {
			result, err := tx.ExecContext(ctx, `
				INSERT INTO role_permissions (role_id, permission_id, granted_at)
				SELECT r.id, p.id, NOW()
				FROM roles r
				CROSS JOIN permissions p
				WHERE r.name = $1 AND p.name = $2
				ON CONFLICT (role_id, permission_id) DO NOTHING
			`, mapping.RoleName, permName)

			if err != nil {
				return fmt.Errorf("failed to assign permission %s to role %s: %w", permName, mapping.RoleName, err)
			}

			rows, _ := result.RowsAffected()
			if rows > 0 {
				permCount++
			}
		}
		log.Printf("  ✓ %s: %d permissions", mapping.RoleName, len(mapping.PermissionNames))
	}

	return nil
}

// seedDefaultUsers seeds default users (optional)
func (s *RBACSeeder) seedDefaultUsers(ctx context.Context, tx *sql.Tx) error {
	log.Println("Seeding default users...")

	users := GetDefaultUsers()
	for _, user := range users {
		// Check if user already exists
		var exists bool
		err := tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 OR email = $2)
		`, user.Username, user.Email).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		if exists {
			log.Printf("  ⊘ User %s already exists, skipping", user.Username)
			continue
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		// Create user
		userID := uuid.New()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (id, username, email, password_hash, first_name, last_name, is_active, is_verified, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, true, true, NOW(), NOW())
		`, userID, user.Username, user.Email, string(hashedPassword), user.FirstName, user.LastName)

		if err != nil {
			return fmt.Errorf("failed to create user %s: %w", user.Username, err)
		}

		// Assign roles to user
		for _, roleName := range user.Roles {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO user_roles (user_id, role_id, assigned_at)
				SELECT $1, r.id, NOW()
				FROM roles r
				WHERE r.name = $2
				ON CONFLICT (user_id, role_id) DO NOTHING
			`, userID, roleName)

			if err != nil {
				return fmt.Errorf("failed to assign role %s to user %s: %w", roleName, user.Username, err)
			}
		}

		log.Printf("  ✓ User: %s (email: %s, password: %s)", user.Username, user.Email, user.Password)
	}

	return nil
}

// Verify checks if all expected RBAC data exists
func (s *RBACSeeder) Verify(ctx context.Context) error {
	log.Println("Verifying RBAC data...")

	// Verify roles
	expectedRoles := GetDefaultRoles()
	for _, role := range expectedRoles {
		var exists bool
		err := s.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM roles WHERE name = $1)
		`, role.Name).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check role %s: %w", role.Name, err)
		}

		if !exists {
			log.Printf("  ✗ Role missing: %s", role.Name)
			return fmt.Errorf("role %s does not exist", role.Name)
		}
		log.Printf("  ✓ Role: %s", role.Name)
	}

	// Verify permissions
	expectedPermissions := GetDefaultPermissions()
	for _, perm := range expectedPermissions {
		var exists bool
		err := s.db.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM permissions WHERE name = $1)
		`, perm.Name).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check permission %s: %w", perm.Name, err)
		}

		if !exists {
			log.Printf("  ✗ Permission missing: %s", perm.Name)
			return fmt.Errorf("permission %s does not exist", perm.Name)
		}
		log.Printf("  ✓ Permission: %s", perm.Name)
	}

	// Verify role-permission mappings
	log.Println("Verifying role-permission mappings...")
	roleMappings := GetDefaultRolePermissions()
	for _, mapping := range roleMappings {
		var count int

		if mapping.RoleName == "admin" {
			// Admin should have all permissions
			err := s.db.QueryRowContext(ctx, `
				SELECT COUNT(*)
				FROM role_permissions rp
				JOIN roles r ON r.id = rp.role_id
				WHERE r.name = 'admin'
			`).Scan(&count)

			if err != nil {
				return fmt.Errorf("failed to check admin permissions: %w", err)
			}

			expectedCount := len(expectedPermissions)
			if count < expectedCount {
				log.Printf("  ✗ Admin role has %d permissions, expected %d", count, expectedCount)
				return fmt.Errorf("admin role is missing permissions")
			}
			log.Printf("  ✓ Admin: %d permissions", count)
		} else if len(mapping.PermissionNames) > 0 {
			err := s.db.QueryRowContext(ctx, `
				SELECT COUNT(*)
				FROM role_permissions rp
				JOIN roles r ON r.id = rp.role_id
				WHERE r.name = $1
			`, mapping.RoleName).Scan(&count)

			if err != nil {
				return fmt.Errorf("failed to check permissions for role %s: %w", mapping.RoleName, err)
			}

			if count < len(mapping.PermissionNames) {
				log.Printf("  ✗ Role %s has %d permissions, expected %d", mapping.RoleName, count, len(mapping.PermissionNames))
			} else {
				log.Printf("  ✓ %s: %d permissions", mapping.RoleName, count)
			}
		}
	}

	log.Println("✓ RBAC verification completed successfully!")
	return nil
}

// Stats returns statistics about RBAC data
func (s *RBACSeeder) Stats(ctx context.Context) error {
	log.Println("\n=== RBAC Statistics ===")

	// Count roles
	var roleCount int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM roles").Scan(&roleCount)
	if err != nil {
		return fmt.Errorf("failed to count roles: %w", err)
	}
	log.Printf("Roles: %d", roleCount)

	// Count permissions
	var permCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM permissions").Scan(&permCount)
	if err != nil {
		return fmt.Errorf("failed to count permissions: %w", err)
	}
	log.Printf("Permissions: %d", permCount)

	// Count role-permission mappings
	var mappingCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM role_permissions").Scan(&mappingCount)
	if err != nil {
		return fmt.Errorf("failed to count role-permission mappings: %w", err)
	}
	log.Printf("Role-Permission Mappings: %d", mappingCount)

	// Count users
	var userCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}
	log.Printf("Users: %d", userCount)

	// Count user-role assignments
	var userRoleCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_roles").Scan(&userRoleCount)
	if err != nil {
		return fmt.Errorf("failed to count user-role assignments: %w", err)
	}
	log.Printf("User-Role Assignments: %d", userRoleCount)

	// Show breakdown by role
	log.Println("\n=== Permissions by Role ===")
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.name, COUNT(rp.permission_id) as perm_count
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		GROUP BY r.name
		ORDER BY r.name
	`)
	if err != nil {
		return fmt.Errorf("failed to query role permissions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var roleName string
		var permCount int
		if err := rows.Scan(&roleName, &permCount); err != nil {
			return fmt.Errorf("failed to scan role permission count: %w", err)
		}
		log.Printf("%s: %d permissions", roleName, permCount)
	}

	log.Println("========================\n")
	return nil
}
