package models

import (
	"encoding/json"
	"testing"
)

func TestWorkflowDefinitionValue(t *testing.T) {
	def := WorkflowDefinition{
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

	value, err := def.Value()
	if err != nil {
		t.Fatalf("Value() failed: %v", err)
	}

	if value == nil {
		t.Error("Value should not be nil")
	}

	// Should be valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(value.([]byte), &result)
	if err != nil {
		t.Errorf("Value should be valid JSON: %v", err)
	}
}

func TestWorkflowDefinitionScan(t *testing.T) {
	jsonData := `{"trigger":{"type":"event","event":"order.created"},"steps":[{"id":"step1","type":"condition"}]}`

	var def WorkflowDefinition
	err := def.Scan([]byte(jsonData))
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	if def.Trigger.Type != "event" {
		t.Errorf("Expected trigger type event, got %s", def.Trigger.Type)
	}

	if len(def.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(def.Steps))
	}
}

func TestWorkflowDefinitionScanNil(t *testing.T) {
	var def WorkflowDefinition
	err := def.Scan(nil)
	if err != nil {
		t.Errorf("Scan(nil) should not error: %v", err)
	}
}

func TestRuleDefinitionValue(t *testing.T) {
	def := RuleDefinition{
		Conditions: []Condition{
			{
				Field:    "order.total",
				Operator: "gt",
				Value:    1000.0,
			},
		},
	}

	value, err := def.Value()
	if err != nil {
		t.Fatalf("Value() failed: %v", err)
	}

	if value == nil {
		t.Error("Value should not be nil")
	}
}

func TestRuleDefinitionScan(t *testing.T) {
	jsonData := `{"conditions":[{"field":"order.total","operator":"gt","value":1000}]}`

	var def RuleDefinition
	err := def.Scan([]byte(jsonData))
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	if len(def.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(def.Conditions))
	}
}
