package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// ScheduleRepository handles workflow schedule database operations
type ScheduleRepository struct {
	db *sql.DB
}

// NewScheduleRepository creates a new schedule repository
func NewScheduleRepository(db *sql.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// Create creates a new workflow schedule
func (r *ScheduleRepository) Create(ctx context.Context, schedule *models.WorkflowSchedule) error {
	query := `
		INSERT INTO workflow_schedules (
			id, workflow_id, cron_expression, timezone, enabled,
			last_triggered_at, next_trigger_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(
		ctx, query,
		schedule.ID, schedule.WorkflowID, schedule.CronExpression,
		schedule.Timezone, schedule.Enabled, schedule.LastTriggeredAt,
		schedule.NextTriggerAt, schedule.CreatedAt, schedule.UpdatedAt,
	).Scan(&schedule.ID, &schedule.CreatedAt, &schedule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	return nil
}

// GetByID retrieves a schedule by ID
func (r *ScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error) {
	schedule := &models.WorkflowSchedule{}
	query := `
		SELECT id, workflow_id, cron_expression, timezone, enabled,
		       last_triggered_at, next_trigger_at, created_at, updated_at
		FROM workflow_schedules
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&schedule.ID, &schedule.WorkflowID, &schedule.CronExpression,
		&schedule.Timezone, &schedule.Enabled, &schedule.LastTriggeredAt,
		&schedule.NextTriggerAt, &schedule.CreatedAt, &schedule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return schedule, nil
}

// GetByWorkflowID retrieves all schedules for a workflow
func (r *ScheduleRepository) GetByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.WorkflowSchedule, error) {
	query := `
		SELECT id, workflow_id, cron_expression, timezone, enabled,
		       last_triggered_at, next_trigger_at, created_at, updated_at
		FROM workflow_schedules
		WHERE workflow_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	schedules := []*models.WorkflowSchedule{}
	for rows.Next() {
		schedule := &models.WorkflowSchedule{}
		err := rows.Scan(
			&schedule.ID, &schedule.WorkflowID, &schedule.CronExpression,
			&schedule.Timezone, &schedule.Enabled, &schedule.LastTriggeredAt,
			&schedule.NextTriggerAt, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// GetDueSchedules retrieves all enabled schedules that are due to run
func (r *ScheduleRepository) GetDueSchedules(ctx context.Context) ([]*models.WorkflowSchedule, error) {
	query := `
		SELECT id, workflow_id, cron_expression, timezone, enabled,
		       last_triggered_at, next_trigger_at, created_at, updated_at
		FROM workflow_schedules
		WHERE enabled = true
		  AND next_trigger_at IS NOT NULL
		  AND next_trigger_at <= $1
		ORDER BY next_trigger_at ASC`

	rows, err := r.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query due schedules: %w", err)
	}
	defer rows.Close()

	schedules := []*models.WorkflowSchedule{}
	for rows.Next() {
		schedule := &models.WorkflowSchedule{}
		err := rows.Scan(
			&schedule.ID, &schedule.WorkflowID, &schedule.CronExpression,
			&schedule.Timezone, &schedule.Enabled, &schedule.LastTriggeredAt,
			&schedule.NextTriggerAt, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// Update updates a workflow schedule
func (r *ScheduleRepository) Update(ctx context.Context, schedule *models.WorkflowSchedule) error {
	query := `
		UPDATE workflow_schedules
		SET cron_expression = $2,
		    timezone = $3,
		    enabled = $4,
		    last_triggered_at = $5,
		    next_trigger_at = $6,
		    updated_at = $7
		WHERE id = $1`

	schedule.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx, query,
		schedule.ID, schedule.CronExpression, schedule.Timezone,
		schedule.Enabled, schedule.LastTriggeredAt, schedule.NextTriggerAt,
		schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}

// UpdateNextTrigger updates only the next_trigger_at and last_triggered_at fields
func (r *ScheduleRepository) UpdateNextTrigger(ctx context.Context, id uuid.UUID, lastTriggered, nextTrigger time.Time) error {
	query := `
		UPDATE workflow_schedules
		SET last_triggered_at = $2,
		    next_trigger_at = $3,
		    updated_at = $4
		WHERE id = $1`

	result, err := r.db.ExecContext(
		ctx, query,
		id, lastTriggered, nextTrigger, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update schedule trigger times: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}

// Delete deletes a workflow schedule
func (r *ScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workflow_schedules WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}

// List retrieves all schedules with pagination
func (r *ScheduleRepository) List(ctx context.Context, limit, offset int) ([]*models.WorkflowSchedule, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM workflow_schedules`
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count schedules: %w", err)
	}

	// Get schedules
	query := `
		SELECT id, workflow_id, cron_expression, timezone, enabled,
		       last_triggered_at, next_trigger_at, created_at, updated_at
		FROM workflow_schedules
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	schedules := []*models.WorkflowSchedule{}
	for rows.Next() {
		schedule := &models.WorkflowSchedule{}
		err := rows.Scan(
			&schedule.ID, &schedule.WorkflowID, &schedule.CronExpression,
			&schedule.Timezone, &schedule.Enabled, &schedule.LastTriggeredAt,
			&schedule.NextTriggerAt, &schedule.CreatedAt, &schedule.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, total, nil
}
