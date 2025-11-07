package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// WorkflowRepository handles workflow database operations
type WorkflowRepository struct {
	db *sql.DB
}

// NewWorkflowRepository creates a new workflow repository
func NewWorkflowRepository(db *sql.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// Create creates a new workflow
func (r *WorkflowRepository) Create(ctx context.Context, organizationID uuid.UUID, req *models.CreateWorkflowRequest, createdBy *uuid.UUID) (*models.Workflow, error) {
	workflow := &models.Workflow{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		WorkflowID:     req.WorkflowID,
		Version:        req.Version,
		Name:           req.Name,
		Description:    req.Description,
		Definition:     req.Definition,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      createdBy,
		Tags:           req.Tags,
	}

	query := `
		INSERT INTO workflows (
			id, organization_id, workflow_id, version, name, description, definition,
			enabled, created_at, updated_at, created_by, tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(
		ctx, query,
		workflow.ID, workflow.OrganizationID, workflow.WorkflowID, workflow.Version, workflow.Name,
		workflow.Description, workflow.Definition, workflow.Enabled,
		workflow.CreatedAt, workflow.UpdatedAt, workflow.CreatedBy,
		pq.Array(workflow.Tags),
	).Scan(&workflow.ID, &workflow.CreatedAt, &workflow.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	return workflow, nil
}

// GetByID retrieves a workflow by ID within an organization
func (r *WorkflowRepository) GetByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error) {
	workflow := &models.Workflow{}
	query := `
		SELECT id, organization_id, workflow_id, version, name, description, definition,
		       enabled, created_at, updated_at, created_by, tags
		FROM workflows
		WHERE organization_id = $1 AND id = $2`

	var tags pq.StringArray
	err := r.db.QueryRowContext(ctx, query, organizationID, id).Scan(
		&workflow.ID, &workflow.OrganizationID, &workflow.WorkflowID, &workflow.Version, &workflow.Name,
		&workflow.Description, &workflow.Definition, &workflow.Enabled,
		&workflow.CreatedAt, &workflow.UpdatedAt, &workflow.CreatedBy, &tags,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	workflow.Tags = tags
	return workflow, nil
}

// GetByWorkflowID retrieves the latest version of a workflow by workflow_id within an organization
func (r *WorkflowRepository) GetByWorkflowID(ctx context.Context, organizationID uuid.UUID, workflowID string) (*models.Workflow, error) {
	workflow := &models.Workflow{}
	query := `
		SELECT id, organization_id, workflow_id, version, name, description, definition,
		       enabled, created_at, updated_at, created_by, tags
		FROM workflows
		WHERE organization_id = $1 AND workflow_id = $2
		ORDER BY created_at DESC
		LIMIT 1`

	var tags pq.StringArray
	err := r.db.QueryRowContext(ctx, query, organizationID, workflowID).Scan(
		&workflow.ID, &workflow.OrganizationID, &workflow.WorkflowID, &workflow.Version, &workflow.Name,
		&workflow.Description, &workflow.Definition, &workflow.Enabled,
		&workflow.CreatedAt, &workflow.UpdatedAt, &workflow.CreatedBy, &tags,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	workflow.Tags = tags
	return workflow, nil
}

// List retrieves workflows with pagination within an organization
func (r *WorkflowRepository) List(ctx context.Context, organizationID uuid.UUID, enabled *bool, limit, offset int) ([]*models.Workflow, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM workflows WHERE organization_id = $1 AND ($2::boolean IS NULL OR enabled = $2)`
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, organizationID, enabled).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count workflows: %w", err)
	}

	// Get workflows
	query := `
		SELECT id, organization_id, workflow_id, version, name, description, definition,
		       enabled, created_at, updated_at, created_by, tags
		FROM workflows
		WHERE organization_id = $1 AND ($2::boolean IS NULL OR enabled = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, organizationID, enabled, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []*models.Workflow
	for rows.Next() {
		workflow := &models.Workflow{}
		var tags pq.StringArray
		err := rows.Scan(
			&workflow.ID, &workflow.OrganizationID, &workflow.WorkflowID, &workflow.Version, &workflow.Name,
			&workflow.Description, &workflow.Definition, &workflow.Enabled,
			&workflow.CreatedAt, &workflow.UpdatedAt, &workflow.CreatedBy, &tags,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan workflow: %w", err)
		}
		workflow.Tags = tags
		workflows = append(workflows, workflow)
	}

	return workflows, total, nil
}

// Update updates a workflow within an organization
func (r *WorkflowRepository) Update(ctx context.Context, organizationID, id uuid.UUID, req *models.UpdateWorkflowRequest) (*models.Workflow, error) {
	query := `
		UPDATE workflows
		SET name = COALESCE($3, name),
		    description = COALESCE($4, description),
		    definition = COALESCE($5, definition),
		    tags = COALESCE($6, tags),
		    updated_at = NOW()
		WHERE organization_id = $1 AND id = $2
		RETURNING id, organization_id, workflow_id, version, name, description, definition,
		          enabled, created_at, updated_at, created_by, tags`

	workflow := &models.Workflow{}
	var tags pq.StringArray

	err := r.db.QueryRowContext(
		ctx, query,
		organizationID, id, req.Name, req.Description, req.Definition, pq.Array(req.Tags),
	).Scan(
		&workflow.ID, &workflow.OrganizationID, &workflow.WorkflowID, &workflow.Version, &workflow.Name,
		&workflow.Description, &workflow.Definition, &workflow.Enabled,
		&workflow.CreatedAt, &workflow.UpdatedAt, &workflow.CreatedBy, &tags,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	workflow.Tags = tags
	return workflow, nil
}

// Delete deletes a workflow within an organization
func (r *WorkflowRepository) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	query := `DELETE FROM workflows WHERE organization_id = $1 AND id = $2`
	result, err := r.db.ExecContext(ctx, query, organizationID, id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("workflow not found")
	}

	return nil
}

// SetEnabled enables or disables a workflow within an organization
func (r *WorkflowRepository) SetEnabled(ctx context.Context, organizationID, id uuid.UUID, enabled bool) error {
	query := `UPDATE workflows SET enabled = $3, updated_at = NOW() WHERE organization_id = $1 AND id = $2`
	result, err := r.db.ExecContext(ctx, query, organizationID, id, enabled)
	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("workflow not found")
	}

	return nil
}

// GetWorkflowByID is an alias for GetByID to match the engine interface
func (r *WorkflowRepository) GetWorkflowByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error) {
	return r.GetByID(ctx, organizationID, id)
}

// ListWorkflows is an adapter for List to match the engine interface
func (r *WorkflowRepository) ListWorkflows(ctx context.Context, organizationID uuid.UUID, enabled *bool, limit, offset int) ([]models.Workflow, int64, error) {
	workflows, total, err := r.List(ctx, organizationID, enabled, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Convert []*models.Workflow to []models.Workflow
	result := make([]models.Workflow, len(workflows))
	for i, w := range workflows {
		result[i] = *w
	}

	return result, total, nil
}

// ListByEventType retrieves workflows that match an event type within an organization
func (r *WorkflowRepository) ListByEventType(ctx context.Context, organizationID uuid.UUID, eventType string) ([]models.Workflow, error) {
	query := `
		SELECT id, organization_id, workflow_id, version, name, description, definition,
		       enabled, created_at, updated_at, created_by, tags
		FROM workflows
		WHERE organization_id = $1 AND enabled = true
		  AND definition->'trigger'->>'type' = 'event'
		  AND definition->'trigger'->>'event' = $2
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, organizationID, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows by event type: %w", err)
	}
	defer rows.Close()

	var workflows []models.Workflow
	for rows.Next() {
		var workflow models.Workflow
		var tags pq.StringArray
		err := rows.Scan(
			&workflow.ID, &workflow.OrganizationID, &workflow.WorkflowID, &workflow.Version, &workflow.Name,
			&workflow.Description, &workflow.Definition, &workflow.Enabled,
			&workflow.CreatedAt, &workflow.UpdatedAt, &workflow.CreatedBy, &tags,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}
		workflow.Tags = tags
		workflows = append(workflows, workflow)
	}

	return workflows, nil
}
