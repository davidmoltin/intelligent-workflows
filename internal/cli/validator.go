package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
)

type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ValidateWorkflowFile validates a workflow definition from a file
func ValidateWorkflowFile(filename string) (*ValidationResult, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var workflow models.Workflow
	if err := json.Unmarshal(data, &workflow); err != nil {
		return &ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Invalid JSON: %v", err)},
		}, nil
	}

	return ValidateWorkflow(&workflow), nil
}

// ValidateWorkflow validates a workflow definition
func ValidateWorkflow(workflow *models.Workflow) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: []string{}}

	// Required fields
	if workflow.WorkflowID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "workflow_id is required")
	}

	if workflow.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "name is required")
	}

	if workflow.Version == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "version is required")
	}

	// Validate definition structure
	if err := validateDefinition(&workflow.Definition); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}

	return result
}

func validateDefinition(def *models.WorkflowDefinition) error {
	if def == nil {
		return fmt.Errorf("definition is required")
	}

	// Validate trigger
	if def.Trigger.Type == "" {
		return fmt.Errorf("trigger.type is required")
	}

	validTriggers := map[string]bool{
		"order.created":   true,
		"order.updated":   true,
		"order.cancelled": true,
		"payment.success": true,
		"payment.failed":  true,
		"inventory.low":   true,
		"customer.created": true,
		"custom":          true,
	}

	if !validTriggers[def.Trigger.Type] {
		return fmt.Errorf("invalid trigger type: %s", def.Trigger.Type)
	}

	// Validate steps
	if len(def.Steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}

	for i, step := range def.Steps {
		if step.ID == "" {
			return fmt.Errorf("step[%d].id is required", i)
		}
		if step.Type == "" {
			return fmt.Errorf("step[%d].type is required", i)
		}

		validStepTypes := map[string]bool{
			"condition": true,
			"action":    true,
			"approval":  true,
			"parallel":  true,
			"webhook":   true,
			"delay":     true,
		}

		if !validStepTypes[step.Type] {
			return fmt.Errorf("step[%d] has invalid type: %s", i, step.Type)
		}

		// Type-specific validation
		if step.Type == "action" && step.Action == nil {
			return fmt.Errorf("step[%d].action is required for action type", i)
		}

		if step.Type == "condition" && step.Condition == nil {
			return fmt.Errorf("step[%d].condition is required for condition type", i)
		}
	}

	return nil
}

// LoadWorkflowFromFile loads a workflow from a JSON file
func LoadWorkflowFromFile(filename string) (*models.Workflow, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var workflow models.Workflow
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	return &workflow, nil
}

// SaveWorkflowToFile saves a workflow to a JSON file
func SaveWorkflowToFile(workflow *models.Workflow, filename string) error {
	data, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
