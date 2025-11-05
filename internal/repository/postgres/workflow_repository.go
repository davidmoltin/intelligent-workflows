package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
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
func (r *WorkflowRepository) Create(ctx context.Context, req *models.CreateWorkflowRequest, createdBy *uuid.UUID) (*models.Workflow, error) {
	workflow := &models.Workflow{
		ID:          uuid.New(),
		WorkflowID:  req.WorkflowID,
		Version:     req.Version,
		Name:        req.Name,
		Description: req.Description,
		Definition:  req.Definition,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   createdBy,
		Tags:        req.Tags,
	}

	query := `
		INSERT INTO workflows (
			id, workflow_id, version, name, description, definition,
			enabled, created_at, updated_at, created_by, tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(
		ctx, query,
		workflow.ID, workflow.WorkflowID, workflow.Version, workflow.Name,
		workflow.Description, workflow.Definition, workflow.Enabled,
		workflow.CreatedAt, workflow.UpdatedAt, workflow.CreatedBy,
		pq.Array(workflow.Tags),
	).Scan(&workflow.ID, &workflow.CreatedAt, &workflow.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	return workflow, nil
}

// GetByID retrieves a workflow by ID
func (r *WorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
	workflow := &models.Workflow{}
	query := `
		SELECT id, workflow_id, version, name, description, definition,
		       enabled, created_at, updated_at, created_by, tags
		FROM workflows
		WHERE id = $1`

	var tags pq.StringArray
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&workflow.ID, &workflow.WorkflowID, &workflow.Version, &workflow.Name,
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

// GetByWorkflowID retrieves the latest version of a workflow by workflow_id
func (r *WorkflowRepository) GetByWorkflowID(ctx context.Context, workflowID string) (*models.Workflow, error) {
	workflow := &models.Workflow{}
	query := `
		SELECT id, workflow_id, version, name, description, definition,
		       enabled, created_at, updated_at, created_by, tags
		FROM workflows
		WHERE workflow_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	var tags pq.StringArray
	err := r.db.QueryRowContext(ctx, query, workflowID).Scan(
		&workflow.ID, &workflow.WorkflowID, &workflow.Version, &workflow.Name,
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

// List retrieves workflows with pagination
func (r *WorkflowRepository) List(ctx context.Context, enabled *bool, limit, offset int) ([]*models.Workflow, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM workflows WHERE ($1::boolean IS NULL OR enabled = $1)`
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, enabled).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count workflows: %w", err)
	}

	// Get workflows
	query := `
		SELECT id, workflow_id, version, name, description, definition,
		       enabled, created_at, updated_at, created_by, tags
		FROM workflows
		WHERE ($1::boolean IS NULL OR enabled = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, enabled, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []*models.Workflow
	for rows.Next() {
		workflow := &models.Workflow{}
		var tags pq.StringArray
		err := rows.Scan(
			&workflow.ID, &workflow.WorkflowID, &workflow.Version, &workflow.Name,
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

// Update updates a workflow
func (r *WorkflowRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateWorkflowRequest) (*models.Workflow, error) {
	query := `
		UPDATE workflows
		SET name = COALESCE($2, name),
		    description = COALESCE($3, description),
		    definition = COALESCE($4, definition),
		    tags = COALESCE($5, tags),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, workflow_id, version, name, description, definition,
		          enabled, created_at, updated_at, created_by, tags`

	workflow := &models.Workflow{}
	var tags pq.StringArray

	err := r.db.QueryRowContext(
		ctx, query,
		id, req.Name, req.Description, req.Definition, pq.Array(req.Tags),
	).Scan(
		&workflow.ID, &workflow.WorkflowID, &workflow.Version, &workflow.Name,
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

// Delete deletes a workflow
func (r *WorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workflows WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
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

// SetEnabled enables or disables a workflow
func (r *WorkflowRepository) SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	query := `UPDATE workflows SET enabled = $2, updated_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, enabled)
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
func (r *WorkflowRepository) GetWorkflowByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
	return r.GetByID(ctx, id)
}

// ListWorkflows is an adapter for List to match the engine interface
func (r *WorkflowRepository) ListWorkflows(ctx context.Context, enabled *bool, limit, offset int) ([]models.Workflow, int64, error) {
	workflows, total, err := r.List(ctx, enabled, limit, offset)
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
