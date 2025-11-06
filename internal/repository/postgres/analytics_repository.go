package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// AnalyticsRepository handles analytics database operations
type AnalyticsRepository struct {
	db *sql.DB
}

// NewAnalyticsRepository creates a new analytics repository
func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// GetExecutionStats returns overall execution statistics
func (r *AnalyticsRepository) GetExecutionStats(ctx context.Context, workflowID *uuid.UUID, timeRange string) (*models.ExecutionStats, error) {
	var startTime time.Time
	switch timeRange {
	case "1h":
		startTime = time.Now().Add(-1 * time.Hour)
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	default:
		startTime = time.Now().Add(-24 * time.Hour)
	}

	query := `
		SELECT
			COUNT(*) as total_executions,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(CASE WHEN status = 'running' THEN 1 END) as running,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'blocked' THEN 1 END) as blocked,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled,
			COUNT(CASE WHEN status = 'paused' THEN 1 END) as paused,
			AVG(CASE WHEN duration_ms IS NOT NULL THEN duration_ms END) as avg_duration_ms,
			MIN(CASE WHEN duration_ms IS NOT NULL THEN duration_ms END) as min_duration_ms,
			MAX(CASE WHEN duration_ms IS NOT NULL THEN duration_ms END) as max_duration_ms
		FROM workflow_executions
		WHERE started_at >= $1
		  AND ($2::uuid IS NULL OR workflow_id = $2)`

	stats := &models.ExecutionStats{}
	var avgDuration, minDuration, maxDuration sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, startTime, workflowID).Scan(
		&stats.TotalExecutions,
		&stats.Completed,
		&stats.Failed,
		&stats.Running,
		&stats.Pending,
		&stats.Blocked,
		&stats.Cancelled,
		&stats.Paused,
		&avgDuration,
		&minDuration,
		&maxDuration,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get execution stats: %w", err)
	}

	if avgDuration.Valid {
		stats.AvgDurationMs = int(avgDuration.Float64)
	}
	if minDuration.Valid {
		stats.MinDurationMs = int(minDuration.Float64)
	}
	if maxDuration.Valid {
		stats.MaxDurationMs = int(maxDuration.Float64)
	}

	// Calculate success rate
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.Completed) / float64(stats.TotalExecutions) * 100
		stats.FailureRate = float64(stats.Failed) / float64(stats.TotalExecutions) * 100
	}

	return stats, nil
}

// GetExecutionTrends returns execution trends over time
func (r *AnalyticsRepository) GetExecutionTrends(ctx context.Context, workflowID *uuid.UUID, timeRange string, interval string) ([]models.ExecutionTrend, error) {
	var startTime time.Time
	var truncInterval string

	switch timeRange {
	case "1h":
		startTime = time.Now().Add(-1 * time.Hour)
		truncInterval = "minute"
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
		truncInterval = "hour"
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
		truncInterval = "day"
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
		truncInterval = "day"
	default:
		startTime = time.Now().Add(-24 * time.Hour)
		truncInterval = "hour"
	}

	// Override if custom interval provided
	if interval != "" {
		truncInterval = interval
	}

	query := fmt.Sprintf(`
		SELECT
			date_trunc('%s', started_at) as time_bucket,
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(CASE WHEN status = 'running' THEN 1 END) as running
		FROM workflow_executions
		WHERE started_at >= $1
		  AND ($2::uuid IS NULL OR workflow_id = $2)
		GROUP BY time_bucket
		ORDER BY time_bucket ASC`, truncInterval)

	rows, err := r.db.QueryContext(ctx, query, startTime, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution trends: %w", err)
	}
	defer rows.Close()

	var trends []models.ExecutionTrend
	for rows.Next() {
		var trend models.ExecutionTrend
		err := rows.Scan(
			&trend.Timestamp,
			&trend.Total,
			&trend.Completed,
			&trend.Failed,
			&trend.Running,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trend: %w", err)
		}
		trends = append(trends, trend)
	}

	return trends, nil
}

// GetWorkflowStats returns statistics per workflow
func (r *AnalyticsRepository) GetWorkflowStats(ctx context.Context, timeRange string, limit int) ([]models.WorkflowStats, error) {
	var startTime time.Time
	switch timeRange {
	case "1h":
		startTime = time.Now().Add(-1 * time.Hour)
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	default:
		startTime = time.Now().Add(-24 * time.Hour)
	}

	query := `
		SELECT
			we.workflow_id,
			w.name as workflow_name,
			COUNT(*) as total_executions,
			COUNT(CASE WHEN we.status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN we.status = 'failed' THEN 1 END) as failed,
			AVG(CASE WHEN we.duration_ms IS NOT NULL THEN we.duration_ms END) as avg_duration_ms
		FROM workflow_executions we
		LEFT JOIN workflows w ON we.workflow_id = w.id
		WHERE we.started_at >= $1
		GROUP BY we.workflow_id, w.name
		ORDER BY total_executions DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, startTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow stats: %w", err)
	}
	defer rows.Close()

	var stats []models.WorkflowStats
	for rows.Next() {
		var stat models.WorkflowStats
		var avgDuration sql.NullFloat64
		var workflowName sql.NullString

		err := rows.Scan(
			&stat.WorkflowID,
			&workflowName,
			&stat.TotalExecutions,
			&stat.Completed,
			&stat.Failed,
			&avgDuration,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow stat: %w", err)
		}

		if workflowName.Valid {
			stat.WorkflowName = workflowName.String
		}
		if avgDuration.Valid {
			stat.AvgDurationMs = int(avgDuration.Float64)
		}

		// Calculate success rate
		if stat.TotalExecutions > 0 {
			stat.SuccessRate = float64(stat.Completed) / float64(stat.TotalExecutions) * 100
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

// GetRecentErrors returns recent execution errors
func (r *AnalyticsRepository) GetRecentErrors(ctx context.Context, workflowID *uuid.UUID, limit int) ([]models.ExecutionError, error) {
	query := `
		SELECT
			we.id,
			we.workflow_id,
			w.name as workflow_name,
			we.execution_id,
			we.error_message,
			we.started_at,
			we.completed_at
		FROM workflow_executions we
		LEFT JOIN workflows w ON we.workflow_id = w.id
		WHERE we.status = 'failed'
		  AND we.error_message IS NOT NULL
		  AND ($1::uuid IS NULL OR we.workflow_id = $1)
		ORDER BY we.started_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workflowID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent errors: %w", err)
	}
	defer rows.Close()

	var errors []models.ExecutionError
	for rows.Next() {
		var execError models.ExecutionError
		var workflowName sql.NullString

		err := rows.Scan(
			&execError.ExecutionID,
			&execError.WorkflowID,
			&workflowName,
			&execError.ExecutionIDStr,
			&execError.ErrorMessage,
			&execError.StartedAt,
			&execError.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan error: %w", err)
		}

		if workflowName.Valid {
			execError.WorkflowName = workflowName.String
		}

		errors = append(errors, execError)
	}

	return errors, nil
}

// GetStepStats returns statistics for individual steps
func (r *AnalyticsRepository) GetStepStats(ctx context.Context, workflowID *uuid.UUID, timeRange string) ([]models.StepStats, error) {
	var startTime time.Time
	switch timeRange {
	case "1h":
		startTime = time.Now().Add(-1 * time.Hour)
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	default:
		startTime = time.Now().Add(-24 * time.Hour)
	}

	query := `
		SELECT
			se.step_id,
			se.step_type,
			COUNT(*) as total_executions,
			COUNT(CASE WHEN se.status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN se.status = 'failed' THEN 1 END) as failed,
			AVG(CASE WHEN se.duration_ms IS NOT NULL THEN se.duration_ms END) as avg_duration_ms
		FROM step_executions se
		JOIN workflow_executions we ON se.execution_id = we.id
		WHERE se.started_at >= $1
		  AND ($2::uuid IS NULL OR we.workflow_id = $2)
		GROUP BY se.step_id, se.step_type
		ORDER BY total_executions DESC`

	rows, err := r.db.QueryContext(ctx, query, startTime, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get step stats: %w", err)
	}
	defer rows.Close()

	var stats []models.StepStats
	for rows.Next() {
		var stat models.StepStats
		var avgDuration sql.NullFloat64

		err := rows.Scan(
			&stat.StepID,
			&stat.StepType,
			&stat.TotalExecutions,
			&stat.Completed,
			&stat.Failed,
			&avgDuration,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step stat: %w", err)
		}

		if avgDuration.Valid {
			stat.AvgDurationMs = int(avgDuration.Float64)
		}

		// Calculate success rate
		if stat.TotalExecutions > 0 {
			stat.SuccessRate = float64(stat.Completed) / float64(stat.TotalExecutions) * 100
		}

		stats = append(stats, stat)
	}

	return stats, nil
}
