package validators

import (
	"fmt"
	"strings"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
)

// WorkflowValidator validates workflow definitions
type WorkflowValidator struct{}

// NewWorkflowValidator creates a new workflow validator
func NewWorkflowValidator() *WorkflowValidator {
	return &WorkflowValidator{}
}

// Validate validates a complete workflow definition
func (v *WorkflowValidator) Validate(workflow *models.Workflow) error {
	var errors []string

	// Validate basic fields
	if workflow.Name == "" {
		errors = append(errors, "workflow name is required")
	}

	if workflow.WorkflowID == "" {
		errors = append(errors, "workflow_id is required")
	}

	if workflow.Version == "" {
		errors = append(errors, "version is required")
	}

	// Validate trigger
	if err := v.validateTrigger(&workflow.Definition.Trigger); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate steps
	if len(workflow.Definition.Steps) == 0 {
		errors = append(errors, "workflow must have at least one step")
	} else {
		if err := v.validateSteps(workflow.Definition.Steps); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("workflow validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateTrigger validates the trigger definition
func (v *WorkflowValidator) validateTrigger(trigger *models.TriggerDefinition) error {
	if trigger.Type == "" {
		return fmt.Errorf("trigger type is required")
	}

	validTypes := map[string]bool{
		"event":    true,
		"schedule": true,
		"manual":   true,
	}

	if !validTypes[trigger.Type] {
		return fmt.Errorf("invalid trigger type '%s', must be one of: event, schedule, manual", trigger.Type)
	}

	// Event triggers require an event name
	if trigger.Type == "event" && trigger.Event == "" {
		return fmt.Errorf("event trigger requires event name")
	}

	// Schedule triggers require a cron expression
	if trigger.Type == "schedule" && trigger.Cron == "" {
		return fmt.Errorf("schedule trigger requires cron expression")
	}

	return nil
}

// validateSteps validates all steps in the workflow
func (v *WorkflowValidator) validateSteps(steps []models.Step) error {
	var errors []string

	// Build step ID map for reference validation
	stepIDs := make(map[string]bool)
	for _, step := range steps {
		if step.ID == "" {
			errors = append(errors, "all steps must have an ID")
			continue
		}

		if stepIDs[step.ID] {
			errors = append(errors, fmt.Sprintf("duplicate step ID: %s", step.ID))
		}
		stepIDs[step.ID] = true
	}

	// Validate each step
	for i, step := range steps {
		if err := v.validateStep(&step, stepIDs, i); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Check for circular dependencies
	if err := v.validateNoCycles(steps); err != nil {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("step validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateStep validates a single step
func (v *WorkflowValidator) validateStep(step *models.Step, stepIDs map[string]bool, index int) error {
	var errors []string

	// Validate step type
	validTypes := map[string]bool{
		"condition": true,
		"action":    true,
		"parallel":  true,
		"execute":   true,
		"wait":      true,
	}

	if !validTypes[step.Type] {
		errors = append(errors, fmt.Sprintf("step %s has invalid type '%s'", step.ID, step.Type))
	}

	// Type-specific validation
	switch step.Type {
	case "condition":
		if step.Condition == nil {
			errors = append(errors, fmt.Sprintf("step %s (condition) must have a condition", step.ID))
		} else {
			if err := v.validateCondition(step.Condition); err != nil {
				errors = append(errors, fmt.Sprintf("step %s: %v", step.ID, err))
			}
		}

		// Validate on_true and on_false references
		if step.OnTrue != "" && !stepIDs[step.OnTrue] {
			errors = append(errors, fmt.Sprintf("step %s references non-existent on_true step: %s", step.ID, step.OnTrue))
		}
		if step.OnFalse != "" && !stepIDs[step.OnFalse] {
			errors = append(errors, fmt.Sprintf("step %s references non-existent on_false step: %s", step.ID, step.OnFalse))
		}

	case "action":
		if step.Action == nil {
			errors = append(errors, fmt.Sprintf("step %s (action) must have an action", step.ID))
		}

	case "parallel":
		if step.Parallel == nil {
			errors = append(errors, fmt.Sprintf("step %s (parallel) must have parallel configuration", step.ID))
		} else {
			if len(step.Parallel.Steps) == 0 {
				errors = append(errors, fmt.Sprintf("step %s (parallel) must have at least one sub-step", step.ID))
			}

			validStrategies := map[string]bool{
				"all_must_pass": true,
				"any_can_pass":  true,
				"best_effort":   true,
			}
			if step.Parallel.Strategy != "" && !validStrategies[step.Parallel.Strategy] {
				errors = append(errors, fmt.Sprintf("step %s has invalid parallel strategy: %s", step.ID, step.Parallel.Strategy))
			}
		}

	case "execute":
		if len(step.Execute) == 0 {
			errors = append(errors, fmt.Sprintf("step %s (execute) must have at least one execute action", step.ID))
		} else {
			for j, action := range step.Execute {
				if err := v.validateExecuteAction(&action); err != nil {
					errors = append(errors, fmt.Sprintf("step %s action %d: %v", step.ID, j, err))
				}
			}
		}

	case "wait":
		if step.Wait == nil {
			errors = append(errors, fmt.Sprintf("step %s (wait) must have wait configuration", step.ID))
		} else {
			if step.Wait.Event == "" {
				errors = append(errors, fmt.Sprintf("step %s (wait) must have an event to wait for", step.ID))
			}
			if step.Wait.OnTimeout != "" && !stepIDs[step.Wait.OnTimeout] {
				errors = append(errors, fmt.Sprintf("step %s references non-existent on_timeout step: %s", step.ID, step.Wait.OnTimeout))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// validateCondition validates a condition
func (v *WorkflowValidator) validateCondition(cond *models.Condition) error {
	if cond.Field == "" && len(cond.And) == 0 && len(cond.Or) == 0 {
		return fmt.Errorf("condition must have field or and/or clauses")
	}

	if cond.Field != "" {
		// Validate operator
		validOperators := map[string]bool{
			"eq":       true,
			"neq":      true,
			"gt":       true,
			"gte":      true,
			"lt":       true,
			"lte":      true,
			"in":       true,
			"contains": true,
			"regex":    true,
		}

		if !validOperators[cond.Operator] {
			return fmt.Errorf("invalid operator '%s'", cond.Operator)
		}

		if cond.Value == nil {
			return fmt.Errorf("condition value is required")
		}
	}

	// Recursively validate nested conditions
	for _, andCond := range cond.And {
		if err := v.validateCondition(&andCond); err != nil {
			return fmt.Errorf("and condition: %v", err)
		}
	}

	for _, orCond := range cond.Or {
		if err := v.validateCondition(&orCond); err != nil {
			return fmt.Errorf("or condition: %v", err)
		}
	}

	return nil
}

// validateExecuteAction validates an execute action
func (v *WorkflowValidator) validateExecuteAction(action *models.ExecuteAction) error {
	if action.Type == "" {
		return fmt.Errorf("execute action type is required")
	}

	// Type-specific validation
	switch action.Type {
	case "notify":
		if len(action.Recipients) == 0 {
			return fmt.Errorf("notify action requires recipients")
		}
		if action.Message == "" {
			return fmt.Errorf("notify action requires message")
		}

	case "webhook", "http_request":
		if action.URL == "" {
			return fmt.Errorf("%s action requires URL", action.Type)
		}
		if action.Method != "" {
			validMethods := map[string]bool{
				"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true,
			}
			if !validMethods[action.Method] {
				return fmt.Errorf("invalid HTTP method: %s", action.Method)
			}
		}

	case "create_record", "update_record", "delete_record":
		if action.Entity == "" {
			return fmt.Errorf("%s action requires entity", action.Type)
		}
		if action.Type != "create_record" && action.EntityID == "" {
			return fmt.Errorf("%s action requires entity_id", action.Type)
		}
	}

	return nil
}

// validateNoCycles checks for circular dependencies in the workflow
func (v *WorkflowValidator) validateNoCycles(steps []models.Step) error {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, step := range steps {
		graph[step.ID] = []string{}

		if step.OnTrue != "" {
			graph[step.ID] = append(graph[step.ID], step.OnTrue)
		}
		if step.OnFalse != "" {
			graph[step.ID] = append(graph[step.ID], step.OnFalse)
		}
		if step.Wait != nil && step.Wait.OnTimeout != "" {
			graph[step.ID] = append(graph[step.ID], step.Wait.OnTimeout)
		}
	}

	// DFS to detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for _, step := range steps {
		if !visited[step.ID] {
			if hasCycle(step.ID) {
				return fmt.Errorf("circular dependency detected in workflow steps")
			}
		}
	}

	return nil
}
