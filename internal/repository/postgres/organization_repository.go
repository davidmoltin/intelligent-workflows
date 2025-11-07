package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// OrganizationRepository handles organization database operations
type OrganizationRepository struct {
	db *sql.DB
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *sql.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create creates a new organization
func (r *OrganizationRepository) Create(ctx context.Context, req *models.CreateOrganizationRequest, createdBy uuid.UUID) (*models.Organization, error) {
	org := &models.Organization{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Settings:    req.Settings,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   createdBy,
	}

	query := `
		INSERT INTO organizations (
			id, name, slug, description, settings, is_active,
			created_at, updated_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(
		ctx, query,
		org.ID, org.Name, org.Slug, org.Description, org.Settings,
		org.IsActive, org.CreatedAt, org.UpdatedAt, org.CreatedBy,
	).Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return org, nil
}

// GetByID retrieves an organization by ID
func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	org := &models.Organization{}
	query := `
		SELECT id, name, slug, description, settings, is_active,
		       created_at, updated_at, created_by
		FROM organizations
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &org.Settings,
		&org.IsActive, &org.CreatedAt, &org.UpdatedAt, &org.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// GetBySlug retrieves an organization by slug
func (r *OrganizationRepository) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	org := &models.Organization{}
	query := `
		SELECT id, name, slug, description, settings, is_active,
		       created_at, updated_at, created_by
		FROM organizations
		WHERE slug = $1`

	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &org.Settings,
		&org.IsActive, &org.CreatedAt, &org.UpdatedAt, &org.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// List retrieves a paginated list of organizations
func (r *OrganizationRepository) List(ctx context.Context, page, pageSize int) ([]models.Organization, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM organizations WHERE is_active = true`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count organizations: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, name, slug, description, settings, is_active,
		       created_at, updated_at, created_by
		FROM organizations
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	orgs := []models.Organization{}
	for rows.Next() {
		var org models.Organization
		if err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Description, &org.Settings,
			&org.IsActive, &org.CreatedAt, &org.UpdatedAt, &org.CreatedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	return orgs, total, nil
}

// Update updates an organization
func (r *OrganizationRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateOrganizationRequest) (*models.Organization, error) {
	query := `
		UPDATE organizations
		SET name = COALESCE($1, name),
		    description = COALESCE($2, description),
		    settings = COALESCE($3, settings),
		    is_active = COALESCE($4, is_active),
		    updated_at = NOW()
		WHERE id = $5
		RETURNING id, name, slug, description, settings, is_active,
		          created_at, updated_at, created_by`

	org := &models.Organization{}
	err := r.db.QueryRowContext(
		ctx, query,
		req.Name, req.Description, req.Settings, req.IsActive, id,
	).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &org.Settings,
		&org.IsActive, &org.CreatedAt, &org.UpdatedAt, &org.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return org, nil
}

// Delete soft-deletes an organization (sets is_active to false)
func (r *OrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE organizations SET is_active = false, updated_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

// GetUserOrganizations retrieves all organizations for a user
func (r *OrganizationRepository) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]models.Organization, error) {
	query := `
		SELECT o.id, o.name, o.slug, o.description, o.settings, o.is_active,
		       o.created_at, o.updated_at, o.created_by
		FROM organizations o
		INNER JOIN organization_users ou ON o.id = ou.organization_id
		WHERE ou.user_id = $1 AND o.is_active = true AND ou.is_active = true
		ORDER BY o.name ASC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}
	defer rows.Close()

	orgs := []models.Organization{}
	for rows.Next() {
		var org models.Organization
		if err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Description, &org.Settings,
			&org.IsActive, &org.CreatedAt, &org.UpdatedAt, &org.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	return orgs, nil
}

// AddUser adds a user to an organization with a role
func (r *OrganizationRepository) AddUser(ctx context.Context, orgID, userID, roleID uuid.UUID, invitedBy *uuid.UUID) (*models.OrganizationUser, error) {
	orgUser := &models.OrganizationUser{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		RoleID:         roleID,
		IsActive:       true,
		JoinedAt:       time.Now(),
		InvitedBy:      invitedBy,
	}

	query := `
		INSERT INTO organization_users (
			id, organization_id, user_id, role_id, is_active, joined_at, invited_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, joined_at`

	err := r.db.QueryRowContext(
		ctx, query,
		orgUser.ID, orgUser.OrganizationID, orgUser.UserID, orgUser.RoleID,
		orgUser.IsActive, orgUser.JoinedAt, orgUser.InvitedBy,
	).Scan(&orgUser.ID, &orgUser.JoinedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to add user to organization: %w", err)
	}

	return orgUser, nil
}

// RemoveUser removes a user from an organization
func (r *OrganizationRepository) RemoveUser(ctx context.Context, orgID, userID uuid.UUID) error {
	query := `DELETE FROM organization_users WHERE organization_id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove user from organization: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found in organization")
	}

	return nil
}

// GetOrganizationUsers retrieves all users in an organization
func (r *OrganizationRepository) GetOrganizationUsers(ctx context.Context, orgID uuid.UUID) ([]models.OrganizationUser, error) {
	query := `
		SELECT ou.id, ou.organization_id, ou.user_id, ou.role_id, ou.is_active,
		       ou.joined_at, ou.invited_by,
		       u.username, u.email, u.first_name, u.last_name,
		       r.name as role_name
		FROM organization_users ou
		INNER JOIN users u ON ou.user_id = u.id
		INNER JOIN roles r ON ou.role_id = r.id
		WHERE ou.organization_id = $1 AND ou.is_active = true
		ORDER BY ou.joined_at DESC`

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization users: %w", err)
	}
	defer rows.Close()

	users := []models.OrganizationUser{}
	for rows.Next() {
		var ou models.OrganizationUser
		var user models.User
		var role models.Role

		if err := rows.Scan(
			&ou.ID, &ou.OrganizationID, &ou.UserID, &ou.RoleID, &ou.IsActive,
			&ou.JoinedAt, &ou.InvitedBy,
			&user.Username, &user.Email, &user.FirstName, &user.LastName,
			&role.Name,
		); err != nil {
			return nil, fmt.Errorf("failed to scan organization user: %w", err)
		}

		user.ID = ou.UserID
		role.ID = ou.RoleID
		ou.User = &user
		ou.Role = &role

		users = append(users, ou)
	}

	return users, nil
}

// UpdateUserRole updates a user's role in an organization
func (r *OrganizationRepository) UpdateUserRole(ctx context.Context, orgID, userID, roleID uuid.UUID) error {
	query := `
		UPDATE organization_users
		SET role_id = $1
		WHERE organization_id = $2 AND user_id = $3`

	result, err := r.db.ExecContext(ctx, query, roleID, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found in organization")
	}

	return nil
}

// GetUserRoleInOrganization retrieves a user's role in an organization
func (r *OrganizationRepository) GetUserRoleInOrganization(ctx context.Context, orgID, userID uuid.UUID) (*models.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN organization_users ou ON r.id = ou.role_id
		WHERE ou.organization_id = $1 AND ou.user_id = $2 AND ou.is_active = true`

	role := &models.Role{}
	err := r.db.QueryRowContext(ctx, query, orgID, userID).Scan(
		&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found in organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// CheckUserAccess checks if a user has access to an organization
func (r *OrganizationRepository) CheckUserAccess(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM organization_users
			WHERE organization_id = $1 AND user_id = $2 AND is_active = true
		)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, orgID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user access: %w", err)
	}

	return exists, nil
}
