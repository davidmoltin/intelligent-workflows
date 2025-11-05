package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowDefinition_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
		check   func(t *testing.T, wd *WorkflowDefinition)
	}{
		{
			name:  "nil value",
			value: nil,
			check: func(t *testing.T, wd *WorkflowDefinition) {
				assert.Empty(t, wd.Steps)
			},
		},
		{
			name: "valid JSON bytes",
			value: []byte(`{
				"trigger": {"type": "event", "event": "order.created"},
				"steps": [
					{"id": "step1", "type": "condition"}
				]
			}`),
			check: func(t *testing.T, wd *WorkflowDefinition) {
				assert.Equal(t, "event", wd.Trigger.Type)
				assert.Equal(t, "order.created", wd.Trigger.Event)
				assert.Len(t, wd.Steps, 1)
				assert.Equal(t, "step1", wd.Steps[0].ID)
			},
		},
		{
			name:  "non-byte value",
			value: "string value",
			check: func(t *testing.T, wd *WorkflowDefinition) {
				// Should not error, but should be empty
				assert.Empty(t, wd.Steps)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wd WorkflowDefinition
			err := wd.Scan(tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.check != nil {
				tt.check(t, &wd)
			}
		})
	}
}

func TestWorkflowDefinition_Value(t *testing.T) {
	wd := WorkflowDefinition{
		Trigger: TriggerDefinition{
			Type:  "event",
			Event: "order.created",
		},
		Steps: []Step{
			{
				ID:   "step1",
				Type: "condition",
			},
		},
	}

	value, err := wd.Value()
	require.NoError(t, err)

	bytes, ok := value.([]byte)
	require.True(t, ok, "value should be []byte")

	// Verify it's valid JSON
	var result WorkflowDefinition
	err = json.Unmarshal(bytes, &result)
	require.NoError(t, err)

	assert.Equal(t, "event", result.Trigger.Type)
	assert.Equal(t, "order.created", result.Trigger.Event)
	assert.Len(t, result.Steps, 1)
}

func TestWorkflowDefinition_RoundTrip(t *testing.T) {
	// Test scanning and then getting value back
	original := WorkflowDefinition{
		Trigger: TriggerDefinition{
			Type:  "event",
			Event: "order.created",
		},
		Context: ContextDefinition{
			Load: []string{"order.details", "customer.profile"},
		},
		Steps: []Step{
			{
				ID:   "step1",
				Type: "condition",
				Condition: &Condition{
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
				Action: &Action{
					Type:   "block",
					Reason: "High value order",
				},
			},
		},
	}

	// Convert to value
	value, err := original.Value()
	require.NoError(t, err)

	// Scan it back
	var scanned WorkflowDefinition
	err = scanned.Scan(value)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.Trigger.Type, scanned.Trigger.Type)
	assert.Equal(t, original.Trigger.Event, scanned.Trigger.Event)
	assert.Equal(t, original.Context.Load, scanned.Context.Load)
	assert.Len(t, scanned.Steps, 2)
	assert.Equal(t, "step1", scanned.Steps[0].ID)
	assert.Equal(t, "condition", scanned.Steps[0].Type)
	assert.NotNil(t, scanned.Steps[0].Condition)
	assert.Equal(t, "order.total", scanned.Steps[0].Condition.Field)
}

func TestCondition_NestedLogic(t *testing.T) {
	// Test that nested AND/OR conditions marshal/unmarshal correctly
	condition := Condition{
		And: []Condition{
			{
				Field:    "order.total",
				Operator: "gt",
				Value:    1000.0,
			},
			{
				Or: []Condition{
					{
						Field:    "customer.tier",
						Operator: "eq",
						Value:    "gold",
					},
					{
						Field:    "customer.tier",
						Operator: "eq",
						Value:    "platinum",
					},
				},
			},
		},
	}

	// Marshal
	bytes, err := json.Marshal(condition)
	require.NoError(t, err)

	// Unmarshal
	var result Condition
	err = json.Unmarshal(bytes, &result)
	require.NoError(t, err)

	// Verify
	assert.Len(t, result.And, 2)
	assert.Equal(t, "order.total", result.And[0].Field)
	assert.Len(t, result.And[1].Or, 2)
	assert.Equal(t, "customer.tier", result.And[1].Or[0].Field)
	assert.Equal(t, "gold", result.And[1].Or[0].Value)
}

func TestExecuteAction_Serialization(t *testing.T) {
	action := ExecuteAction{
		Type:       "webhook",
		URL:        "https://example.com/webhook",
		Method:     "POST",
		Headers:    map[string]string{"Authorization": "Bearer token"},
		Body:       map[string]interface{}{"order_id": "123", "amount": 1500.0},
		Recipients: []string{"admin@example.com"},
	}

	// Marshal
	bytes, err := json.Marshal(action)
	require.NoError(t, err)

	// Unmarshal
	var result ExecuteAction
	err = json.Unmarshal(bytes, &result)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "webhook", result.Type)
	assert.Equal(t, "https://example.com/webhook", result.URL)
	assert.Equal(t, "POST", result.Method)
	assert.Equal(t, "Bearer token", result.Headers["Authorization"])
	assert.Equal(t, "123", result.Body["order_id"])
	assert.InDelta(t, 1500.0, result.Body["amount"], 0.001)
}
