package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus represents the status of a workflow execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusBlocked   ExecutionStatus = "blocked"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
	ExecutionStatusPaused    ExecutionStatus = "paused"
)

// ExecutionResult represents the result of a workflow execution
type ExecutionResult string

const (
	ExecutionResultAllowed  ExecutionResult = "allowed"
	ExecutionResultBlocked  ExecutionResult = "blocked"
	ExecutionResultExecuted ExecutionResult = "executed"
	ExecutionResultFailed   ExecutionResult = "failed"
)

// WorkflowExecution represents an execution instance of a workflow
type WorkflowExecution struct {
	ID             uuid.UUID        `json:"id" db:"id"`
	WorkflowID     uuid.UUID        `json:"workflow_id" db:"workflow_id"`
	ExecutionID    string           `json:"execution_id" db:"execution_id"`
	TriggerEvent   string           `json:"trigger_event" db:"trigger_event"`
	TriggerPayload JSONB            `json:"trigger_payload" db:"trigger_payload"`
	Context        JSONB            `json:"context" db:"context"`
	Status         ExecutionStatus  `json:"status" db:"status"`
	Result         *ExecutionResult `json:"result,omitempty" db:"result"`
	StartedAt      time.Time        `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs     *int             `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage   *string          `json:"error_message,omitempty" db:"error_message"`
	Metadata       JSONB            `json:"metadata,omitempty" db:"metadata"`

	// Workflow resumer fields
	PausedAt       *time.Time  `json:"paused_at,omitempty" db:"paused_at"`
	PausedReason   *string     `json:"paused_reason,omitempty" db:"paused_reason"`
	PausedStepID   *uuid.UUID  `json:"paused_step_id,omitempty" db:"paused_step_id"`
	NextStepID     *uuid.UUID  `json:"next_step_id,omitempty" db:"next_step_id"`
	ResumeData     JSONB       `json:"resume_data,omitempty" db:"resume_data"`
	ResumeCount    int         `json:"resume_count" db:"resume_count"`
	LastResumedAt  *time.Time  `json:"last_resumed_at,omitempty" db:"last_resumed_at"`
}

// StepExecutionStatus represents the status of a step execution
type StepExecutionStatus string

const (
	StepStatusPending   StepExecutionStatus = "pending"
	StepStatusRunning   StepExecutionStatus = "running"
	StepStatusCompleted StepExecutionStatus = "completed"
	StepStatusFailed    StepExecutionStatus = "failed"
	StepStatusSkipped   StepExecutionStatus = "skipped"
)

// StepExecution represents an execution of a workflow step
type StepExecution struct {
	ID           uuid.UUID           `json:"id" db:"id"`
	ExecutionID  uuid.UUID           `json:"execution_id" db:"execution_id"`
	StepID       string              `json:"step_id" db:"step_id"`
	StepType     string              `json:"step_type" db:"step_type"`
	Status       StepExecutionStatus `json:"status" db:"status"`
	Input        JSONB               `json:"input,omitempty" db:"input"`
	Output       JSONB               `json:"output,omitempty" db:"output"`
	StartedAt    time.Time           `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time          `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs   *int                `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage *string             `json:"error_message,omitempty" db:"error_message"`
}

// JSONB is a custom type for handling JSONB columns
type JSONB map[string]interface{}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		*j = make(map[string]interface{})
		return nil
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(j)
}

// ExecutionListResponse represents a paginated list of executions
type ExecutionListResponse struct {
	Executions []WorkflowExecution `json:"executions"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
}

// ExecutionTraceResponse represents the trace of a workflow execution
type ExecutionTraceResponse struct {
	Execution *WorkflowExecution `json:"execution"`
	Steps     []StepExecution    `json:"steps"`
	Workflow  *Workflow          `json:"workflow,omitempty"`
}
