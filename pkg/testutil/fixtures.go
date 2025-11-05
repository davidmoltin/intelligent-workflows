package testutil

import (
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// FixtureBuilder provides methods to create test fixtures
type FixtureBuilder struct{}

// NewFixtureBuilder creates a new fixture builder
func NewFixtureBuilder() *FixtureBuilder {
	return &FixtureBuilder{}
}

// Workflow creates a test workflow
func (fb *FixtureBuilder) Workflow(overrides ...func(*models.Workflow)) *models.Workflow {
	id := uuid.New()
	now := time.Now()

	workflow := &models.Workflow{
		ID:          id,
		WorkflowID:  "test-workflow-" + id.String()[:8],
		Version:     "1.0.0",
		Name:        "Test Workflow",
		Description: StringPtr("Test workflow description"),
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "order.created",
			},
			Steps: []models.Step{
				{
					ID:   "step1",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "order.total",
						Operator: "gt",
						Value:    1000.0,
					},
					OnTrue:  "step2",
					OnFalse: "step3",
				},
				{
					ID:   "step2",
					Type: "action",
					Action: &models.Action{
						Type:   "block",
						Reason: "High value order requires approval",
					},
					Execute: []models.ExecuteAction{
						{
							Type:       "notify",
							Recipients: []string{"admin@example.com"},
							Message:    "High value order blocked",
						},
					},
				},
				{
					ID:   "step3",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, override := range overrides {
		override(workflow)
	}

	return workflow
}

// Execution creates a test execution
func (fb *FixtureBuilder) Execution(workflowID uuid.UUID, overrides ...func(*models.WorkflowExecution)) *models.WorkflowExecution {
	id := uuid.New()
	now := time.Now()

	execution := &models.WorkflowExecution{
		ID:          id,
		ExecutionID: "exec-" + id.String()[:8],
		WorkflowID:  workflowID,
		TriggerEvent: "order.created",
		TriggerPayload: models.JSONB{
			"order_id": "order-123",
			"total":    1500.00,
		},
		Context: models.JSONB{
			"order": map[string]interface{}{
				"id":    "order-123",
				"total": 1500.00,
			},
		},
		Status:    models.ExecutionStatusRunning,
		StartedAt: now,
	}

	for _, override := range overrides {
		override(execution)
	}

	return execution
}

// Event creates a test event
func (fb *FixtureBuilder) Event(overrides ...func(*models.Event)) *models.Event {
	id := uuid.New()
	now := time.Now()

	event := &models.Event{
		ID:        id,
		EventID:   "evt-" + id.String()[:8],
		EventType: "order.created",
		Source:    "api",
		Payload: map[string]interface{}{
			"order_id": "order-123",
			"total":    1500.00,
			"customer_id": "cust-456",
		},
		ReceivedAt: now,
	}

	for _, override := range overrides {
		override(event)
	}

	return event
}

// Rule creates a test rule
func (fb *FixtureBuilder) Rule(overrides ...func(*models.Rule)) *models.Rule {
	id := uuid.New()
	now := time.Now()

	rule := &models.Rule{
		ID:          id,
		RuleID:      "rule-" + id.String()[:8],
		Name:        "Test Rule",
		Description: StringPtr("Test rule description"),
		RuleType:    models.RuleTypeCondition,
		Definition: models.RuleDefinition{
			Conditions: []models.Condition{
				{
					Field:    "order.total",
					Operator: "gt",
					Value:    1000.0,
				},
			},
		},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, override := range overrides {
		override(rule)
	}

	return rule
}

// StepExecution creates a test step execution
func (fb *FixtureBuilder) StepExecution(executionID uuid.UUID, overrides ...func(*models.StepExecution)) *models.StepExecution {
	id := uuid.New()
	now := time.Now()

	stepExec := &models.StepExecution{
		ID:          id,
		ExecutionID: executionID,
		StepID:      "step1",
		StepType:    "condition",
		Status:      models.StepStatusRunning,
		StartedAt:   now,
	}

	for _, override := range overrides {
		override(stepExec)
	}

	return stepExec
}

// Helper functions

// StringPtr returns a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// Int64Ptr returns a pointer to an int64
func Int64Ptr(i int64) *int64 {
	return &i
}

// IntPtr returns a pointer to an int
func IntPtr(i int) *int {
	return &i
}

// TimePtr returns a pointer to a time
func TimePtr(t time.Time) *time.Time {
	return &t
}

// UUIDPtr returns a pointer to a UUID
func UUIDPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
