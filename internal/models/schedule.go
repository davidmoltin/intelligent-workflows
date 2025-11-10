package models

import (
	"time"

	"github.com/google/uuid"
)

// WorkflowSchedule represents a cron-based schedule for a workflow
type WorkflowSchedule struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id" db:"organization_id"`
	WorkflowID      uuid.UUID  `json:"workflow_id" db:"workflow_id"`
	CronExpression  string     `json:"cron_expression" db:"cron_expression"`
	Timezone        string     `json:"timezone" db:"timezone"`
	Enabled         bool       `json:"enabled" db:"enabled"`
	LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty" db:"last_triggered_at"`
	NextTriggerAt   *time.Time `json:"next_trigger_at,omitempty" db:"next_trigger_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateScheduleRequest represents the request body for creating a schedule
type CreateScheduleRequest struct {
	CronExpression string `json:"cron_expression" validate:"required"`
	Timezone       string `json:"timezone"`
	Enabled        *bool  `json:"enabled"`
}

// UpdateScheduleRequest represents the request body for updating a schedule
type UpdateScheduleRequest struct {
	CronExpression *string `json:"cron_expression"`
	Timezone       *string `json:"timezone"`
	Enabled        *bool   `json:"enabled"`
}

// ScheduleListResponse represents the response for listing schedules
type ScheduleListResponse struct {
	Schedules []*WorkflowSchedule `json:"schedules"`
	Total     int64               `json:"total"`
	Page      int                 `json:"page"`
	PageSize  int                 `json:"page_size"`
}

// NextRunsResponse represents the response for previewing next runs
type NextRunsResponse struct {
	ScheduleID uuid.UUID `json:"schedule_id"`
	NextRuns   []time.Time `json:"next_runs"`
}
