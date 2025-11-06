package models

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionStats represents overall execution statistics
type ExecutionStats struct {
	TotalExecutions int     `json:"total_executions"`
	Completed       int     `json:"completed"`
	Failed          int     `json:"failed"`
	Running         int     `json:"running"`
	Pending         int     `json:"pending"`
	Blocked         int     `json:"blocked"`
	Cancelled       int     `json:"cancelled"`
	Paused          int     `json:"paused"`
	SuccessRate     float64 `json:"success_rate"`
	FailureRate     float64 `json:"failure_rate"`
	AvgDurationMs   int     `json:"avg_duration_ms"`
	MinDurationMs   int     `json:"min_duration_ms"`
	MaxDurationMs   int     `json:"max_duration_ms"`
}

// ExecutionTrend represents execution trends over time
type ExecutionTrend struct {
	Timestamp time.Time `json:"timestamp"`
	Total     int       `json:"total"`
	Completed int       `json:"completed"`
	Failed    int       `json:"failed"`
	Running   int       `json:"running"`
}

// WorkflowStats represents statistics per workflow
type WorkflowStats struct {
	WorkflowID      uuid.UUID `json:"workflow_id"`
	WorkflowName    string    `json:"workflow_name"`
	TotalExecutions int       `json:"total_executions"`
	Completed       int       `json:"completed"`
	Failed          int       `json:"failed"`
	SuccessRate     float64   `json:"success_rate"`
	AvgDurationMs   int       `json:"avg_duration_ms"`
}

// ExecutionError represents a recent execution error
type ExecutionError struct {
	ExecutionID    uuid.UUID  `json:"execution_id"`
	WorkflowID     uuid.UUID  `json:"workflow_id"`
	WorkflowName   string     `json:"workflow_name"`
	ExecutionIDStr string     `json:"execution_id_str"`
	ErrorMessage   string     `json:"error_message"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

// StepStats represents statistics for individual steps
type StepStats struct {
	StepID          string  `json:"step_id"`
	StepType        string  `json:"step_type"`
	TotalExecutions int     `json:"total_executions"`
	Completed       int     `json:"completed"`
	Failed          int     `json:"failed"`
	SuccessRate     float64 `json:"success_rate"`
	AvgDurationMs   int     `json:"avg_duration_ms"`
}

// AnalyticsDashboard represents the full analytics dashboard response
type AnalyticsDashboard struct {
	Stats          *ExecutionStats    `json:"stats"`
	Trends         []ExecutionTrend   `json:"trends"`
	WorkflowStats  []WorkflowStats    `json:"workflow_stats"`
	RecentErrors   []ExecutionError   `json:"recent_errors"`
	StepStats      []StepStats        `json:"step_stats,omitempty"`
	TimeRange      string             `json:"time_range"`
	GeneratedAt    time.Time          `json:"generated_at"`
}
