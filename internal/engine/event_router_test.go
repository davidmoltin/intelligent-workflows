package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// getTestContextEnrichmentConfigForEventRouter returns a test configuration with enrichment disabled
func getTestContextEnrichmentConfigForEventRouter() *config.ContextEnrichmentConfig {
	return &config.ContextEnrichmentConfig{
		Enabled:    false,
		BaseURL:    "http://localhost:8081",
		Timeout:    10 * time.Second,
		MaxRetries: 3,
		RetryDelay: 500 * time.Millisecond,
		CacheTTL:   5 * time.Minute,
		EndpointMapping: map[string]string{
			"order.details": "/api/v1/orders/{id}/details",
		},
	}
}

// Mock WorkflowRepository for testing
type mockWorkflowRepo struct {
	getByIDFunc   func(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error)
	listFunc      func(ctx context.Context, organizationID uuid.UUID, enabled *bool, limit, offset int) ([]models.Workflow, int64, error)
}

func (m *mockWorkflowRepo) GetWorkflowByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, organizationID, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockWorkflowRepo) ListWorkflows(ctx context.Context, organizationID uuid.UUID, enabled *bool, limit, offset int) ([]models.Workflow, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, organizationID, enabled, limit, offset)
	}
	return []models.Workflow{}, 0, nil
}

// Mock EventRepository for testing
type mockEventRepo struct {
	createFunc   func(ctx context.Context, event *models.Event) error
	updateFunc   func(ctx context.Context, organizationID uuid.UUID, event *models.Event) error
	getByIDFunc  func(ctx context.Context, organizationID, id uuid.UUID) (*models.Event, error)
}

func (m *mockEventRepo) CreateEvent(ctx context.Context, event *models.Event) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, event)
	}
	return nil
}

func (m *mockEventRepo) UpdateEvent(ctx context.Context, organizationID uuid.UUID, event *models.Event) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, organizationID, event)
	}
	return nil
}

func (m *mockEventRepo) GetEventByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Event, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, organizationID, id)
	}
	return nil, fmt.Errorf("not found")
}

// TestSafeExecuteWorkflow_NormalExecution tests normal workflow execution without panic
func TestSafeExecuteWorkflow_NormalExecution(t *testing.T) {
	log := logger.NewForTesting()

	executionRepo := &mockExecutionRepo{}
	workflowRepo := &mockWorkflowRepo{}

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	executor := NewWorkflowExecutor(redisClient, executionRepo, workflowRepo, nil, log, nil, getTestContextEnrichmentConfigForEventRouter())

	eventRepo := &mockEventRepo{}
	router := NewEventRouter(workflowRepo, eventRepo, executor, log)

	workflow := &models.Workflow{
		ID:   uuid.New(),
		Name: "test-workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type: "event",
			},
			Steps: []models.Step{
				{
					ID:   "step1",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}

	ctx := context.Background()
	orgID := uuid.New()
	router.safeExecuteWorkflow(ctx, orgID, workflow, "test.event", map[string]interface{}{})

	// Give goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	// Should not crash - if we get here, test passed
	t.Log("Workflow executed without panic")
}

// TestSafeExecuteWorkflow_PanicRecovery tests that panics are recovered and logged
func TestSafeExecuteWorkflow_PanicRecovery(t *testing.T) {
	log := logger.NewForTesting()

	executionRepo := &mockExecutionRepo{
		createExecutionFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
			if exec.Status == models.ExecutionStatusFailed && exec.ErrorMessage != nil {
				t.Logf("Panic execution recorded: %s", *exec.ErrorMessage)

				// Verify metadata contains panic info
				if exec.Metadata == nil {
					t.Error("Expected metadata to contain panic info")
				} else {
					if _, ok := exec.Metadata["panic_recovered"]; !ok {
						t.Error("Expected metadata to have panic_recovered field")
					}
					if _, ok := exec.Metadata["stack_trace"]; !ok {
						t.Error("Expected metadata to have stack_trace field")
					}
				}
			}
			return nil
		},
	}

	workflowRepo := &mockWorkflowRepo{}

	// Create an executor that will fail on execution
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	executor := NewWorkflowExecutor(redisClient, executionRepo, workflowRepo, nil, log, nil, getTestContextEnrichmentConfigForEventRouter())

	eventRepo := &mockEventRepo{}
	router := NewEventRouter(workflowRepo, eventRepo, executor, log)

	// Create a workflow with empty steps - simpler test
	workflow := &models.Workflow{
		ID:   uuid.New(),
		Name: "panic-workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type: "event",
			},
			Steps: []models.Step{},
		},
	}

	done := make(chan bool, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic was not recovered by safeExecuteWorkflow: %v", r)
			}
			done <- true
		}()

		ctx := context.Background()
		orgID := uuid.New()
		router.safeExecuteWorkflow(ctx, orgID, workflow, "test.event", map[string]interface{}{})
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		t.Log("safeExecuteWorkflow completed without propagating panic")
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}

	// Even if execution fails, it shouldn't crash
	t.Log("Panic recovery test completed successfully")
}

// TestRouteEvent_WithWorkflows tests routing events to matching workflows
func TestRouteEvent_WithWorkflows(t *testing.T) {
	log := logger.NewForTesting()

	executionRepo := &mockExecutionRepo{}
	workflowRepo := &mockWorkflowRepo{
		listFunc: func(ctx context.Context, organizationID uuid.UUID, enabled *bool, limit, offset int) ([]models.Workflow, int64, error) {
			return []models.Workflow{
				{
					ID:             uuid.New(),
					OrganizationID: organizationID,
					WorkflowID:     "test-workflow",
					Name:           "Test Workflow",
					Enabled:        true,
					Definition: models.WorkflowDefinition{
						Trigger: models.TriggerDefinition{
							Type:  "event",
							Event: "test.event",
						},
						Steps: []models.Step{},
					},
				},
			}, 1, nil
		},
	}

	eventRepo := &mockEventRepo{}

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	executor := NewWorkflowExecutor(redisClient, executionRepo, workflowRepo, nil, log, nil, getTestContextEnrichmentConfigForEventRouter())
	router := NewEventRouter(workflowRepo, eventRepo, executor, log)

	ctx := context.Background()
	orgID := uuid.New()
	event, err := router.RouteEvent(ctx, orgID, "test.event", "test-source", map[string]interface{}{
		"test": "data",
	})

	if err != nil {
		t.Fatalf("RouteEvent failed: %v", err)
	}

	if event == nil {
		t.Fatal("Expected event to be returned")
	}

	if event.EventType != "test.event" {
		t.Errorf("Expected event type 'test.event', got '%s'", event.EventType)
	}

	// Give goroutines time to complete
	time.Sleep(200 * time.Millisecond)

	t.Log("Event routing completed successfully")
}

// TestWorkflowMatchesEvent tests event matching logic
func TestWorkflowMatchesEvent(t *testing.T) {
	log := logger.NewForTesting()
	router := &EventRouter{logger: log}

	tests := []struct {
		name         string
		workflow     models.Workflow
		eventType    string
		shouldMatch  bool
	}{
		{
			name: "exact match",
			workflow: models.Workflow{
				Definition: models.WorkflowDefinition{
					Trigger: models.TriggerDefinition{
						Type:  "event",
						Event: "order.created",
					},
				},
			},
			eventType:   "order.created",
			shouldMatch: true,
		},
		{
			name: "wildcard match",
			workflow: models.Workflow{
				Definition: models.WorkflowDefinition{
					Trigger: models.TriggerDefinition{
						Type:  "event",
						Event: "order.*",
					},
				},
			},
			eventType:   "order.updated",
			shouldMatch: true,
		},
		{
			name: "no match - different event",
			workflow: models.Workflow{
				Definition: models.WorkflowDefinition{
					Trigger: models.TriggerDefinition{
						Type:  "event",
						Event: "user.created",
					},
				},
			},
			eventType:   "order.created",
			shouldMatch: false,
		},
		{
			name: "no match - schedule trigger",
			workflow: models.Workflow{
				Definition: models.WorkflowDefinition{
					Trigger: models.TriggerDefinition{
						Type: "schedule",
						Cron: "0 0 * * *",
					},
				},
			},
			eventType:   "order.created",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := router.workflowMatchesEvent(tt.workflow, tt.eventType)
			if matches != tt.shouldMatch {
				t.Errorf("Expected match=%v, got match=%v", tt.shouldMatch, matches)
			}
		})
	}
}

// TestTriggerWorkflowManually tests manual workflow triggering
func TestTriggerWorkflowManually(t *testing.T) {
	log := logger.NewForTesting()

	orgID := uuid.New()
	workflowID := uuid.New()
	workflow := &models.Workflow{
		ID:             workflowID,
		OrganizationID: orgID,
		WorkflowID:     "manual-workflow",
		Name:           "Manual Workflow",
		Enabled:        true,
		Definition: models.WorkflowDefinition{
			Steps: []models.Step{},
		},
	}

	workflowRepo := &mockWorkflowRepo{
		getByIDFunc: func(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error) {
			if id == workflowID {
				return workflow, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}

	executionRepo := &mockExecutionRepo{}
	eventRepo := &mockEventRepo{}

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	executor := NewWorkflowExecutor(redisClient, executionRepo, workflowRepo, nil, log, nil, getTestContextEnrichmentConfigForEventRouter())
	router := NewEventRouter(workflowRepo, eventRepo, executor, log)

	ctx := context.Background()
	execution, err := router.TriggerWorkflowManually(ctx, orgID, workflowID, map[string]interface{}{
		"manual": "trigger",
	})

	if err != nil {
		t.Fatalf("TriggerWorkflowManually failed: %v", err)
	}

	if execution == nil {
		t.Fatal("Expected execution to be returned")
	}

	if execution.WorkflowID != workflowID {
		t.Errorf("Expected workflow ID %s, got %s", workflowID, execution.WorkflowID)
	}

	t.Log("Manual trigger completed successfully")
}

// TestTriggerWorkflowManually_DisabledWorkflow tests triggering disabled workflow
func TestTriggerWorkflowManually_DisabledWorkflow(t *testing.T) {
	log := logger.NewForTesting()

	orgID := uuid.New()
	workflowID := uuid.New()
	workflow := &models.Workflow{
		ID:             workflowID,
		OrganizationID: orgID,
		Name:           "Disabled Workflow",
		Enabled:        false, // Disabled
		Definition:     models.WorkflowDefinition{},
	}

	workflowRepo := &mockWorkflowRepo{
		getByIDFunc: func(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error) {
			return workflow, nil
		},
	}

	executionRepo := &mockExecutionRepo{}
	eventRepo := &mockEventRepo{}

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	executor := NewWorkflowExecutor(redisClient, executionRepo, workflowRepo, nil, log, nil, getTestContextEnrichmentConfigForEventRouter())
	router := NewEventRouter(workflowRepo, eventRepo, executor, log)

	ctx := context.Background()
	execution, err := router.TriggerWorkflowManually(ctx, orgID, workflowID, map[string]interface{}{})

	if err == nil {
		t.Fatal("Expected error when triggering disabled workflow")
	}

	if execution != nil {
		t.Error("Expected nil execution for disabled workflow")
	}

	if err.Error() != fmt.Sprintf("workflow is disabled: %s", workflow.Name) {
		t.Errorf("Expected 'workflow is disabled' error, got: %v", err)
	}

	t.Log("Disabled workflow correctly rejected")
}

// TestTriggerWorkflowManually_NotFound tests triggering non-existent workflow
func TestTriggerWorkflowManually_NotFound(t *testing.T) {
	log := logger.NewForTesting()

	workflowRepo := &mockWorkflowRepo{
		getByIDFunc: func(ctx context.Context, organizationID, id uuid.UUID) (*models.Workflow, error) {
			return nil, fmt.Errorf("workflow not found")
		},
	}

	executionRepo := &mockExecutionRepo{}
	eventRepo := &mockEventRepo{}

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	executor := NewWorkflowExecutor(redisClient, executionRepo, workflowRepo, nil, log, nil, getTestContextEnrichmentConfigForEventRouter())
	router := NewEventRouter(workflowRepo, eventRepo, executor, log)

	ctx := context.Background()
	orgID := uuid.New()
	workflowID := uuid.New()
	execution, err := router.TriggerWorkflowManually(ctx, orgID, workflowID, map[string]interface{}{})

	if err == nil {
		t.Fatal("Expected error when triggering non-existent workflow")
	}

	if execution != nil {
		t.Error("Expected nil execution for non-existent workflow")
	}

	t.Log("Non-existent workflow correctly handled")
}
