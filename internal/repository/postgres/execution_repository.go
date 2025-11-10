package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// ExecutionRepository handles execution database operations
type ExecutionRepository struct {
	db *sql.DB
}

// NewExecutionRepository creates a new execution repository
func NewExecutionRepository(db *sql.DB) *ExecutionRepository {
	return &ExecutionRepository{db: db}
}

// CreateExecution creates a new workflow execution
func (r *ExecutionRepository) CreateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	query := `
		INSERT INTO workflow_executions (
			id, organization_id, workflow_id, execution_id, trigger_event, trigger_payload,
			context, status, result, started_at, completed_at, duration_ms,
			error_message, metadata, timeout_at, timeout_duration
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, started_at`

	err := r.db.QueryRowContext(
		ctx, query,
		execution.ID, execution.OrganizationID, execution.WorkflowID, execution.ExecutionID,
		execution.TriggerEvent, execution.TriggerPayload, execution.Context,
		execution.Status, execution.Result, execution.StartedAt,
		execution.CompletedAt, execution.DurationMs, execution.ErrorMessage,
		execution.Metadata, execution.TimeoutAt, execution.TimeoutDuration,
	).Scan(&execution.ID, &execution.StartedAt)

	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// UpdateExecution updates an execution
func (r *ExecutionRepository) UpdateExecution(ctx context.Context, organizationID uuid.UUID, execution *models.WorkflowExecution) error {
	query := `
		UPDATE workflow_executions
		SET context = $3,
		    status = $4,
		    result = $5,
		    completed_at = $6,
		    duration_ms = $7,
		    error_message = $8,
		    metadata = $9,
		    paused_at = $10,
		    paused_reason = $11,
		    paused_step_id = $12,
		    next_step_id = $13,
		    resume_data = $14,
		    resume_count = $15,
		    last_resumed_at = $16
		WHERE organization_id = $1 AND id = $2`

	result, err := r.db.ExecContext(
		ctx, query,
		organizationID, execution.ID, execution.Context, execution.Status,
		execution.Result, execution.CompletedAt, execution.DurationMs,
		execution.ErrorMessage, execution.Metadata,
		execution.PausedAt, execution.PausedReason, execution.PausedStepID,
		execution.NextStepID, execution.ResumeData, execution.ResumeCount,
		execution.LastResumedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("execution not found")
	}

	return nil
}

// GetExecutionByID retrieves an execution by ID within an organization
func (r *ExecutionRepository) GetExecutionByID(ctx context.Context, organizationID, id uuid.UUID) (*models.WorkflowExecution, error) {
	execution := &models.WorkflowExecution{}
	query := `
		SELECT id, organization_id, workflow_id, execution_id, trigger_event, trigger_payload,
		       context, status, result, started_at, completed_at, duration_ms,
		       error_message, metadata, paused_at, paused_reason, paused_step_id,
		       next_step_id, resume_data, resume_count, last_resumed_at
		FROM workflow_executions
		WHERE organization_id = $1 AND id = $2`

	err := r.db.QueryRowContext(ctx, query, organizationID, id).Scan(
		&execution.ID, &execution.OrganizationID, &execution.WorkflowID, &execution.ExecutionID,
		&execution.TriggerEvent, &execution.TriggerPayload, &execution.Context,
		&execution.Status, &execution.Result, &execution.StartedAt,
		&execution.CompletedAt, &execution.DurationMs, &execution.ErrorMessage,
		&execution.Metadata, &execution.PausedAt, &execution.PausedReason,
		&execution.PausedStepID, &execution.NextStepID, &execution.ResumeData,
		&execution.ResumeCount, &execution.LastResumedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return execution, nil
}

// GetExecutionByExecutionID retrieves an execution by execution_id string within an organization
func (r *ExecutionRepository) GetExecutionByExecutionID(ctx context.Context, organizationID uuid.UUID, executionID string) (*models.WorkflowExecution, error) {
	execution := &models.WorkflowExecution{}
	query := `
		SELECT id, organization_id, workflow_id, execution_id, trigger_event, trigger_payload,
		       context, status, result, started_at, completed_at, duration_ms,
		       error_message, metadata
		FROM workflow_executions
		WHERE organization_id = $1 AND execution_id = $2`

	err := r.db.QueryRowContext(ctx, query, organizationID, executionID).Scan(
		&execution.ID, &execution.OrganizationID, &execution.WorkflowID, &execution.ExecutionID,
		&execution.TriggerEvent, &execution.TriggerPayload, &execution.Context,
		&execution.Status, &execution.Result, &execution.StartedAt,
		&execution.CompletedAt, &execution.DurationMs, &execution.ErrorMessage,
		&execution.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return execution, nil
}

// ListExecutions retrieves executions with pagination and filters within an organization
func (r *ExecutionRepository) ListExecutions(
	ctx context.Context,
	organizationID uuid.UUID,
	workflowID *uuid.UUID,
	status *models.ExecutionStatus,
	limit, offset int,
) ([]models.WorkflowExecution, int64, error) {
	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM workflow_executions
		WHERE organization_id = $1
		  AND ($2::uuid IS NULL OR workflow_id = $2)
		  AND ($3::varchar IS NULL OR status = $3)`

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, organizationID, workflowID, status).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count executions: %w", err)
	}

	// Get executions
	query := `
		SELECT id, organization_id, workflow_id, execution_id, trigger_event, trigger_payload,
		       context, status, result, started_at, completed_at, duration_ms,
		       error_message, metadata
		FROM workflow_executions
		WHERE organization_id = $1
		  AND ($2::uuid IS NULL OR workflow_id = $2)
		  AND ($3::varchar IS NULL OR status = $3)
		ORDER BY started_at DESC
		LIMIT $4 OFFSET $5`

	rows, err := r.db.QueryContext(ctx, query, organizationID, workflowID, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []models.WorkflowExecution
	for rows.Next() {
		execution := models.WorkflowExecution{}
		err := rows.Scan(
			&execution.ID, &execution.OrganizationID, &execution.WorkflowID, &execution.ExecutionID,
			&execution.TriggerEvent, &execution.TriggerPayload, &execution.Context,
			&execution.Status, &execution.Result, &execution.StartedAt,
			&execution.CompletedAt, &execution.DurationMs, &execution.ErrorMessage,
			&execution.Metadata,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan execution: %w", err)
		}
		executions = append(executions, execution)
	}

	return executions, total, nil
}

// CreateStepExecution creates a new step execution
func (r *ExecutionRepository) CreateStepExecution(ctx context.Context, step *models.StepExecution) error {
	query := `
		INSERT INTO step_executions (
			id, organization_id, execution_id, step_id, step_type, status,
			input, output, started_at, completed_at, duration_ms, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, started_at`

	err := r.db.QueryRowContext(
		ctx, query,
		step.ID, step.OrganizationID, step.ExecutionID, step.StepID, step.StepType,
		step.Status, step.Input, step.Output, step.StartedAt,
		step.CompletedAt, step.DurationMs, step.ErrorMessage,
	).Scan(&step.ID, &step.StartedAt)

	if err != nil {
		return fmt.Errorf("failed to create step execution: %w", err)
	}

	return nil
}

// UpdateStepExecution updates a step execution
func (r *ExecutionRepository) UpdateStepExecution(ctx context.Context, organizationID uuid.UUID, step *models.StepExecution) error {
	query := `
		UPDATE step_executions
		SET status = $3,
		    output = $4,
		    completed_at = $5,
		    duration_ms = $6,
		    error_message = $7
		WHERE organization_id = $1 AND id = $2`

	result, err := r.db.ExecContext(
		ctx, query,
		organizationID, step.ID, step.Status, step.Output, step.CompletedAt,
		step.DurationMs, step.ErrorMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to update step execution: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("step execution not found")
	}

	return nil
}

// GetStepExecutions retrieves all step executions for an execution within an organization
func (r *ExecutionRepository) GetStepExecutions(ctx context.Context, organizationID, executionID uuid.UUID) ([]models.StepExecution, error) {
	query := `
		SELECT id, organization_id, execution_id, step_id, step_type, status,
		       input, output, started_at, completed_at, duration_ms, error_message
		FROM step_executions
		WHERE organization_id = $1 AND execution_id = $2
		ORDER BY started_at ASC`

	rows, err := r.db.QueryContext(ctx, query, organizationID, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get step executions: %w", err)
	}
	defer rows.Close()

	var steps []models.StepExecution
	for rows.Next() {
		step := models.StepExecution{}
		err := rows.Scan(
			&step.ID, &step.OrganizationID, &step.ExecutionID, &step.StepID, &step.StepType,
			&step.Status, &step.Input, &step.Output, &step.StartedAt,
			&step.CompletedAt, &step.DurationMs, &step.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step execution: %w", err)
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// GetExecutionTrace retrieves full execution trace with steps
func (r *ExecutionRepository) GetExecutionTrace(ctx context.Context, organizationID, id uuid.UUID) (*models.ExecutionTraceResponse, error) {
	// Get execution
	execution, err := r.GetExecutionByID(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}

	// Get steps
	steps, err := r.GetStepExecutions(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}

	trace := &models.ExecutionTraceResponse{
		Execution: execution,
		Steps:     steps,
	}

	return trace, nil
}

// GetPausedExecutions retrieves paused executions within an organization
func (r *ExecutionRepository) GetPausedExecutions(ctx context.Context, organizationID uuid.UUID, limit int) ([]*models.WorkflowExecution, error) {
	query := `
		SELECT id, organization_id, workflow_id, execution_id, trigger_event, trigger_payload,
		       context, status, result, started_at, completed_at, duration_ms,
		       error_message, metadata, paused_at, paused_reason, paused_step_id,
		       next_step_id, resume_data, resume_count, last_resumed_at
		FROM workflow_executions
		WHERE organization_id = $1 AND status = $2
		ORDER BY paused_at ASC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, organizationID, models.ExecutionStatusPaused, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get paused executions: %w", err)
	}
	defer rows.Close()

	var executions []*models.WorkflowExecution
	for rows.Next() {
		execution := &models.WorkflowExecution{}
		err := rows.Scan(
			&execution.ID, &execution.OrganizationID, &execution.WorkflowID, &execution.ExecutionID,
			&execution.TriggerEvent, &execution.TriggerPayload, &execution.Context,
			&execution.Status, &execution.Result, &execution.StartedAt,
			&execution.CompletedAt, &execution.DurationMs, &execution.ErrorMessage,
			&execution.Metadata, &execution.PausedAt, &execution.PausedReason,
			&execution.PausedStepID, &execution.NextStepID, &execution.ResumeData,
			&execution.ResumeCount, &execution.LastResumedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan paused execution: %w", err)
		}
		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating paused executions: %w", err)
	}

	return executions, nil
}

// GetTimedOutExecutions retrieves executions that have exceeded their timeout within an organization
func (r *ExecutionRepository) GetTimedOutExecutions(ctx context.Context, organizationID uuid.UUID, limit int) ([]*models.WorkflowExecution, error) {
	query := `
		SELECT id, organization_id, workflow_id, execution_id, trigger_event, trigger_payload,
		       context, status, result, started_at, completed_at, duration_ms,
		       error_message, metadata, paused_at, paused_reason, paused_step_id,
		       next_step_id, resume_data, resume_count, last_resumed_at,
		       timeout_at, timeout_duration
		FROM workflow_executions
		WHERE organization_id = $1
		  AND timeout_at IS NOT NULL
		  AND timeout_at < NOW()
		  AND status IN ($2, $3)
		ORDER BY timeout_at ASC
		LIMIT $4`

	rows, err := r.db.QueryContext(ctx, query,
		organizationID,
		models.ExecutionStatusRunning,
		models.ExecutionStatusWaiting,
		limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get timed-out executions: %w", err)
	}
	defer rows.Close()

	var executions []*models.WorkflowExecution
	for rows.Next() {
		execution := &models.WorkflowExecution{}
		err := rows.Scan(
			&execution.ID, &execution.OrganizationID, &execution.WorkflowID, &execution.ExecutionID,
			&execution.TriggerEvent, &execution.TriggerPayload, &execution.Context,
			&execution.Status, &execution.Result, &execution.StartedAt,
			&execution.CompletedAt, &execution.DurationMs, &execution.ErrorMessage,
			&execution.Metadata, &execution.PausedAt, &execution.PausedReason,
			&execution.PausedStepID, &execution.NextStepID, &execution.ResumeData,
			&execution.ResumeCount, &execution.LastResumedAt,
			&execution.TimeoutAt, &execution.TimeoutDuration,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timed-out execution: %w", err)
		}
		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timed-out executions: %w", err)
	}

	return executions, nil
}

// CancelExecution cancels a running execution
func (r *ExecutionRepository) CancelExecution(ctx context.Context, organizationID, id uuid.UUID) error {
	query := `
		UPDATE workflow_executions
		SET status = $3,
		    completed_at = NOW(),
		    duration_ms = EXTRACT(EPOCH FROM (NOW() - started_at))::INTEGER * 1000
		WHERE organization_id = $1 AND id = $2 AND status IN ($4, $5)`

	result, err := r.db.ExecContext(ctx, query,
		organizationID,
		id,
		models.ExecutionStatusCancelled,
		models.ExecutionStatusRunning,
		models.ExecutionStatusWaiting,
	)

	if err != nil {
		return fmt.Errorf("failed to cancel execution: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("execution not found or not in cancellable state")
	}

	return nil
}
