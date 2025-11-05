package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
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
			id, workflow_id, execution_id, trigger_event, trigger_payload,
			context, status, result, started_at, completed_at, duration_ms,
			error_message, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, started_at`

	err := r.db.QueryRowContext(
		ctx, query,
		execution.ID, execution.WorkflowID, execution.ExecutionID,
		execution.TriggerEvent, execution.TriggerPayload, execution.Context,
		execution.Status, execution.Result, execution.StartedAt,
		execution.CompletedAt, execution.DurationMs, execution.ErrorMessage,
		execution.Metadata,
	).Scan(&execution.ID, &execution.StartedAt)

	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// UpdateExecution updates an execution
func (r *ExecutionRepository) UpdateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	query := `
		UPDATE workflow_executions
		SET context = $2,
		    status = $3,
		    result = $4,
		    completed_at = $5,
		    duration_ms = $6,
		    error_message = $7,
		    metadata = $8,
		    paused_at = $9,
		    paused_reason = $10,
		    paused_step_id = $11,
		    next_step_id = $12,
		    resume_data = $13,
		    resume_count = $14,
		    last_resumed_at = $15
		WHERE id = $1`

	result, err := r.db.ExecContext(
		ctx, query,
		execution.ID, execution.Context, execution.Status,
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

// GetExecutionByID retrieves an execution by ID
func (r *ExecutionRepository) GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
	execution := &models.WorkflowExecution{}
	query := `
		SELECT id, workflow_id, execution_id, trigger_event, trigger_payload,
		       context, status, result, started_at, completed_at, duration_ms,
		       error_message, metadata, paused_at, paused_reason, paused_step_id,
		       next_step_id, resume_data, resume_count, last_resumed_at
		FROM workflow_executions
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&execution.ID, &execution.WorkflowID, &execution.ExecutionID,
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

// GetExecutionByExecutionID retrieves an execution by execution_id string
func (r *ExecutionRepository) GetExecutionByExecutionID(ctx context.Context, executionID string) (*models.WorkflowExecution, error) {
	execution := &models.WorkflowExecution{}
	query := `
		SELECT id, workflow_id, execution_id, trigger_event, trigger_payload,
		       context, status, result, started_at, completed_at, duration_ms,
		       error_message, metadata
		FROM workflow_executions
		WHERE execution_id = $1`

	err := r.db.QueryRowContext(ctx, query, executionID).Scan(
		&execution.ID, &execution.WorkflowID, &execution.ExecutionID,
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

// ListExecutions retrieves executions with pagination and filters
func (r *ExecutionRepository) ListExecutions(
	ctx context.Context,
	workflowID *uuid.UUID,
	status *models.ExecutionStatus,
	limit, offset int,
) ([]models.WorkflowExecution, int64, error) {
	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM workflow_executions
		WHERE ($1::uuid IS NULL OR workflow_id = $1)
		  AND ($2::varchar IS NULL OR status = $2)`

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, workflowID, status).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count executions: %w", err)
	}

	// Get executions
	query := `
		SELECT id, workflow_id, execution_id, trigger_event, trigger_payload,
		       context, status, result, started_at, completed_at, duration_ms,
		       error_message, metadata
		FROM workflow_executions
		WHERE ($1::uuid IS NULL OR workflow_id = $1)
		  AND ($2::varchar IS NULL OR status = $2)
		ORDER BY started_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, workflowID, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []models.WorkflowExecution
	for rows.Next() {
		execution := models.WorkflowExecution{}
		err := rows.Scan(
			&execution.ID, &execution.WorkflowID, &execution.ExecutionID,
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
			id, execution_id, step_id, step_type, status,
			input, output, started_at, completed_at, duration_ms, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, started_at`

	err := r.db.QueryRowContext(
		ctx, query,
		step.ID, step.ExecutionID, step.StepID, step.StepType,
		step.Status, step.Input, step.Output, step.StartedAt,
		step.CompletedAt, step.DurationMs, step.ErrorMessage,
	).Scan(&step.ID, &step.StartedAt)

	if err != nil {
		return fmt.Errorf("failed to create step execution: %w", err)
	}

	return nil
}

// UpdateStepExecution updates a step execution
func (r *ExecutionRepository) UpdateStepExecution(ctx context.Context, step *models.StepExecution) error {
	query := `
		UPDATE step_executions
		SET status = $2,
		    output = $3,
		    completed_at = $4,
		    duration_ms = $5,
		    error_message = $6
		WHERE id = $1`

	result, err := r.db.ExecContext(
		ctx, query,
		step.ID, step.Status, step.Output, step.CompletedAt,
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

// GetStepExecutions retrieves all step executions for an execution
func (r *ExecutionRepository) GetStepExecutions(ctx context.Context, executionID uuid.UUID) ([]models.StepExecution, error) {
	query := `
		SELECT id, execution_id, step_id, step_type, status,
		       input, output, started_at, completed_at, duration_ms, error_message
		FROM step_executions
		WHERE execution_id = $1
		ORDER BY started_at ASC`

	rows, err := r.db.QueryContext(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get step executions: %w", err)
	}
	defer rows.Close()

	var steps []models.StepExecution
	for rows.Next() {
		step := models.StepExecution{}
		err := rows.Scan(
			&step.ID, &step.ExecutionID, &step.StepID, &step.StepType,
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
func (r *ExecutionRepository) GetExecutionTrace(ctx context.Context, id uuid.UUID) (*models.ExecutionTraceResponse, error) {
	// Get execution
	execution, err := r.GetExecutionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get steps
	steps, err := r.GetStepExecutions(ctx, id)
	if err != nil {
		return nil, err
	}

	trace := &models.ExecutionTraceResponse{
		Execution: execution,
		Steps:     steps,
	}

	return trace, nil
}

// GetPausedExecutions retrieves paused executions ordered by paused_at
func (r *ExecutionRepository) GetPausedExecutions(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, execution_id, trigger_event, trigger_payload,
		       context, status, result, started_at, completed_at, duration_ms,
		       error_message, metadata, paused_at, paused_reason, paused_step_id,
		       next_step_id, resume_data, resume_count, last_resumed_at
		FROM workflow_executions
		WHERE status = $1
		ORDER BY paused_at ASC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, models.ExecutionStatusPaused, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get paused executions: %w", err)
	}
	defer rows.Close()

	var executions []*models.WorkflowExecution
	for rows.Next() {
		execution := &models.WorkflowExecution{}
		err := rows.Scan(
			&execution.ID, &execution.WorkflowID, &execution.ExecutionID,
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
