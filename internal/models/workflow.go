package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Workflow represents a workflow definition
type Workflow struct {
	ID             uuid.UUID          `json:"id" db:"id"`
	OrganizationID uuid.UUID          `json:"organization_id" db:"organization_id"`
	WorkflowID     string             `json:"workflow_id" db:"workflow_id"`
	Version        string             `json:"version" db:"version"`
	Name           string             `json:"name" db:"name"`
	Description    *string            `json:"description,omitempty" db:"description"`
	Definition     WorkflowDefinition `json:"definition" db:"definition"`
	Enabled        bool               `json:"enabled" db:"enabled"`
	CreatedAt      time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" db:"updated_at"`
	CreatedBy      *uuid.UUID         `json:"created_by,omitempty" db:"created_by"`
	Tags           []string           `json:"tags,omitempty" db:"tags"`
}

// WorkflowDefinition represents the complete workflow definition
type WorkflowDefinition struct {
	Trigger TriggerDefinition `json:"trigger"`
	Context ContextDefinition `json:"context,omitempty"`
	Steps   []Step            `json:"steps"`
	Timeout string            `json:"timeout,omitempty"` // Global timeout duration, e.g., "5m", "1h", "30s"
}

// TriggerDefinition defines what starts the workflow
type TriggerDefinition struct {
	Type  string                 `json:"type"` // event, schedule, manual
	Event string                 `json:"event,omitempty"`
	Cron  string                 `json:"cron,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// ContextDefinition defines what data to load
type ContextDefinition struct {
	Load []string `json:"load,omitempty"` // e.g., ["order.details", "customer.history"]
}

// Step represents a workflow step
type Step struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // condition, action, parallel, execute, wait
	Condition *Condition             `json:"condition,omitempty"`
	Action    *Action                `json:"action,omitempty"`
	OnTrue    string                 `json:"on_true,omitempty"`
	OnFalse   string                 `json:"on_false,omitempty"`
	Parallel  *ParallelStep          `json:"parallel,omitempty"`
	Execute   []ExecuteAction        `json:"execute,omitempty"`
	Wait      *WaitConfig            `json:"wait,omitempty"`
	Retry     *RetryConfig           `json:"retry,omitempty"`
	Timeout   string                 `json:"timeout,omitempty"` // Step-level timeout, e.g., "30s", "2m"
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Condition represents a conditional expression
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, neq, gt, gte, lt, lte, in, contains, regex
	Value    interface{} `json:"value"`
	And      []Condition `json:"and,omitempty"`
	Or       []Condition `json:"or,omitempty"`
}

// Action represents an action to take
type Action struct {
	Type     string                 `json:"action"` // allow, block, execute
	Reason   string                 `json:"reason,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ParallelStep represents parallel execution of steps
type ParallelStep struct {
	Steps    []Step `json:"steps"`
	Strategy string `json:"strategy"` // all_must_pass, any_can_pass, best_effort
}

// ExecuteAction represents an action to execute
type ExecuteAction struct {
	Type       string                 `json:"type"` // notify, webhook, create_record, http_request, etc.
	Recipients []string               `json:"recipients,omitempty"`
	Message    string                 `json:"message,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Method     string                 `json:"method,omitempty"`
	Headers    map[string]string      `json:"headers,omitempty"`
	Body       map[string]interface{} `json:"body,omitempty"`
	Entity     string                 `json:"entity,omitempty"`
	EntityID   string                 `json:"entity_id,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// WaitConfig represents wait/timeout configuration
type WaitConfig struct {
	Event     string `json:"event"`
	Timeout   string `json:"timeout"` // duration string, e.g., "24h"
	OnTimeout string `json:"on_timeout"`
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxAttempts int      `json:"max_attempts"`
	Backoff     string   `json:"backoff"` // linear, exponential
	RetryOn     []string `json:"retry_on,omitempty"`
}

// CreateWorkflowRequest represents the request to create a workflow
type CreateWorkflowRequest struct {
	WorkflowID  string             `json:"workflow_id" validate:"required"`
	Version     string             `json:"version" validate:"required"`
	Name        string             `json:"name" validate:"required"`
	Description *string            `json:"description,omitempty"`
	Definition  WorkflowDefinition `json:"definition" validate:"required"`
	Tags        []string           `json:"tags,omitempty"`
}

// UpdateWorkflowRequest represents the request to update a workflow
type UpdateWorkflowRequest struct {
	Name        *string             `json:"name,omitempty"`
	Description *string             `json:"description,omitempty"`
	Definition  *WorkflowDefinition `json:"definition,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
}

// JSONB scanning for WorkflowDefinition
func (w *WorkflowDefinition) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, w)
}

func (w WorkflowDefinition) Value() (driver.Value, error) {
	return json.Marshal(w)
}
