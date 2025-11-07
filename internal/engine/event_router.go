package engine

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
)

// WorkflowRepository defines the interface for workflow data access
type WorkflowRepository interface {
	GetWorkflowByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error)
	ListWorkflows(ctx context.Context, organizationID uuid.UUID, enabled *bool, limit, offset int) ([]models.Workflow, int64, error)
}

// EventRepository defines the interface for event persistence
type EventRepository interface {
	CreateEvent(ctx context.Context, organizationID uuid.UUID, event *models.Event) error
	UpdateEvent(ctx context.Context, organizationID uuid.UUID, event *models.Event) error
	GetEventByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Event, error)
}

// EventRouter routes events to matching workflows
type EventRouter struct {
	workflowRepo WorkflowRepository
	eventRepo    EventRepository
	executor     *WorkflowExecutor
	logger       *logger.Logger
}

// NewEventRouter creates a new event router
func NewEventRouter(
	workflowRepo WorkflowRepository,
	eventRepo EventRepository,
	executor *WorkflowExecutor,
	log *logger.Logger,
) *EventRouter {
	return &EventRouter{
		workflowRepo: workflowRepo,
		eventRepo:    eventRepo,
		executor:     executor,
		logger:       log,
	}
}

// RouteEvent routes an event to matching workflows
func (er *EventRouter) RouteEvent(
	ctx context.Context,
	organizationID uuid.UUID,
	eventType string,
	source string,
	payload map[string]interface{},
) (*models.Event, error) {
	er.logger.Infof("Routing event: %s from %s for organization: %s", eventType, source, organizationID)

	// Create event record
	event := &models.Event{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		EventID:        fmt.Sprintf("evt_%s", uuid.New().String()[:8]),
		EventType:      eventType,
		Source:         source,
		Payload:        payload,
		ReceivedAt:     time.Now(),
	}

	if err := er.eventRepo.CreateEvent(ctx, organizationID, event); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Find matching workflows
	workflows, err := er.findMatchingWorkflows(ctx, organizationID, eventType)
	if err != nil {
		er.logger.Errorf("Failed to find matching workflows: %v", err)
		return event, err
	}

	if len(workflows) == 0 {
		er.logger.Infof("No workflows found for event type: %s", eventType)
		now := time.Now()
		event.ProcessedAt = &now
		er.eventRepo.UpdateEvent(ctx, organizationID, event)
		return event, nil
	}

	er.logger.Infof("Found %d matching workflows for event: %s", len(workflows), eventType)

	// Execute matching workflows
	triggeredWorkflows := make([]string, 0, len(workflows))

	for _, workflow := range workflows {
		er.logger.Infof("Triggering workflow: %s (ID: %s)", workflow.Name, workflow.ID)

		// Execute workflow asynchronously with panic recovery
		go func(wf models.Workflow) {
			execCtx := context.Background()
			er.safeExecuteWorkflow(execCtx, organizationID, &wf, eventType, payload)
		}(workflow)

		triggeredWorkflows = append(triggeredWorkflows, workflow.WorkflowID)
	}

	// Update event with triggered workflows
	event.TriggeredWorkflows = triggeredWorkflows
	now := time.Now()
	event.ProcessedAt = &now

	if err := er.eventRepo.UpdateEvent(ctx, organizationID, event); err != nil {
		er.logger.Errorf("Failed to update event: %v", err)
	}

	return event, nil
}

// safeExecuteWorkflow executes a workflow with panic recovery
func (er *EventRouter) safeExecuteWorkflow(
	ctx context.Context,
	organizationID uuid.UUID,
	workflow *models.Workflow,
	eventType string,
	payload map[string]interface{},
) {
	defer func() {
		if rec := recover(); rec != nil {
			stack := string(debug.Stack())
			er.logger.Errorf(
				"PANIC in workflow execution goroutine - workflow_id: %s, workflow_name: %s, event_type: %s, panic: %v, stack: %s",
				workflow.ID,
				workflow.Name,
				eventType,
				rec,
				stack,
			)

			// Create a failed execution record to track the panic
			now := time.Now()
			errorMsg := fmt.Sprintf("panic during execution: %v", rec)
			execution := &models.WorkflowExecution{
				ID:             uuid.New(),
				OrganizationID: organizationID,
				ExecutionID:    fmt.Sprintf("exec_panic_%s", uuid.New().String()[:8]),
				WorkflowID:     workflow.ID,
				TriggerEvent:   eventType,
				Status:         models.ExecutionStatusFailed,
				StartedAt:      now,
				CompletedAt:    &now,
				ErrorMessage:   &errorMsg,
				Metadata: models.JSONB{
					"panic_recovered": true,
					"stack_trace":     stack,
				},
			}

			// Attempt to record the panic execution (best effort)
			if er.executor.executionRepo != nil {
				if err := er.executor.executionRepo.CreateExecution(ctx, organizationID, execution); err != nil {
					er.logger.Errorf("Failed to record panic execution: %v", err)
				}
			}
		}
	}()

	// Execute the workflow
	_, err := er.executor.Execute(ctx, organizationID, workflow, eventType, payload)
	if err != nil {
		er.logger.Errorf("Workflow execution failed: %s - %v", workflow.Name, err)
	}
}

// findMatchingWorkflows finds workflows that match the event type
func (er *EventRouter) findMatchingWorkflows(
	ctx context.Context,
	organizationID uuid.UUID,
	eventType string,
) ([]models.Workflow, error) {
	// Get all enabled workflows for this organization
	enabled := true
	workflows, _, err := er.workflowRepo.ListWorkflows(ctx, organizationID, &enabled, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	// Filter workflows that match this event type
	matchingWorkflows := make([]models.Workflow, 0)

	for _, workflow := range workflows {
		if er.workflowMatchesEvent(workflow, eventType) {
			matchingWorkflows = append(matchingWorkflows, workflow)
		}
	}

	return matchingWorkflows, nil
}

// workflowMatchesEvent checks if a workflow should be triggered by an event
func (er *EventRouter) workflowMatchesEvent(workflow models.Workflow, eventType string) bool {
	trigger := workflow.Definition.Trigger

	// Check if trigger type is event
	if trigger.Type != "event" {
		return false
	}

	// Check if event type matches
	if trigger.Event == "" {
		return false
	}

	// Exact match
	if trigger.Event == eventType {
		return true
	}

	// Wildcard match (e.g., "order.*" matches "order.created", "order.updated")
	if len(trigger.Event) > 0 && trigger.Event[len(trigger.Event)-1] == '*' {
		prefix := trigger.Event[:len(trigger.Event)-1]
		if len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// ProcessScheduledWorkflows finds and executes workflows with schedule triggers
// NOTE: This method is deprecated. Schedule-based workflow execution is now handled
// by the SchedulerWorker in the workers package, which uses the WorkflowSchedule
// model and provides more robust cron-based scheduling with timezone support.
// This method is kept for backward compatibility but does not perform any actions.
func (er *EventRouter) ProcessScheduledWorkflows(ctx context.Context) error {
	er.logger.Debugf("ProcessScheduledWorkflows called - schedule triggers are now handled by SchedulerWorker")
	return nil
}

// TriggerWorkflowManually triggers a workflow manually
func (er *EventRouter) TriggerWorkflowManually(
	ctx context.Context,
	organizationID uuid.UUID,
	workflowID uuid.UUID,
	payload map[string]interface{},
) (*models.WorkflowExecution, error) {
	er.logger.Infof("Manually triggering workflow: %s for organization: %s", workflowID, organizationID)

	// Get workflow
	workflow, err := er.workflowRepo.GetWorkflowByID(ctx, organizationID, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if !workflow.Enabled {
		return nil, fmt.Errorf("workflow is disabled: %s", workflow.Name)
	}

	// Execute workflow
	execution, err := er.executor.Execute(ctx, organizationID, workflow, "manual", payload)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	return execution, nil
}
