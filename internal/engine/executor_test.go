package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// getTestContextEnrichmentConfigForExecutor returns a test configuration with enrichment disabled
func getTestContextEnrichmentConfigForExecutor() *config.ContextEnrichmentConfig {
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

// Mock ExecutionRepository
type mockExecutionRepo struct {
	createExecutionFunc     func(ctx context.Context, organizationID uuid.UUID, execution *models.WorkflowExecution) error
	updateExecutionFunc     func(ctx context.Context, organizationID uuid.UUID, execution *models.WorkflowExecution) error
	getExecutionByIDFunc    func(ctx context.Context, organizationID, id uuid.UUID) (*models.WorkflowExecution, error)
	createStepExecutionFunc func(ctx context.Context, organizationID uuid.UUID, step *models.StepExecution) error
	updateStepExecutionFunc func(ctx context.Context, organizationID uuid.UUID, step *models.StepExecution) error
}

func (m *mockExecutionRepo) CreateExecution(ctx context.Context, organizationID uuid.UUID, execution *models.WorkflowExecution) error {
	if m.createExecutionFunc != nil {
		return m.createExecutionFunc(ctx, organizationID, execution)
	}
	return nil
}

func (m *mockExecutionRepo) UpdateExecution(ctx context.Context, organizationID uuid.UUID, execution *models.WorkflowExecution) error {
	if m.updateExecutionFunc != nil {
		return m.updateExecutionFunc(ctx, organizationID, execution)
	}
	return nil
}

func (m *mockExecutionRepo) GetExecutionByID(ctx context.Context, organizationID, id uuid.UUID) (*models.WorkflowExecution, error) {
	if m.getExecutionByIDFunc != nil {
		return m.getExecutionByIDFunc(ctx, organizationID, id)
	}
	return nil, errors.New("not found")
}

func (m *mockExecutionRepo) CreateStepExecution(ctx context.Context, organizationID uuid.UUID, step *models.StepExecution) error {
	if m.createStepExecutionFunc != nil {
		return m.createStepExecutionFunc(ctx, organizationID, step)
	}
	return nil
}

func (m *mockExecutionRepo) UpdateStepExecution(ctx context.Context, organizationID uuid.UUID, step *models.StepExecution) error {
	if m.updateStepExecutionFunc != nil {
		return m.updateStepExecutionFunc(ctx, organizationID, step)
	}
	return nil
}

func TestExecuteWaitStep(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	t.Run("pauses execution with timeout", func(t *testing.T) {
		var updatedExecution *models.WorkflowExecution

		repo := &mockExecutionRepo{
			updateExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				updatedExecution = execution
				return nil
			},
		}

		// Create a minimal Redis client (won't be used in this test)
		redisClient := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})

		executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

		execution := &models.WorkflowExecution{
			ID:          uuid.New(),
			ExecutionID: "test-exec",
			Status:      models.ExecutionStatusRunning,
		}

		step := &models.Step{
			ID:   "wait1",
			Type: "wait",
			Wait: &models.WaitConfig{
				Event:     "approval.granted",
				Timeout:   "24h",
				OnTimeout: "timeout_step",
			},
		}

		err := executor.executeWaitStep(ctx, execution, step, map[string]interface{}{})

		if err != ErrExecutionPaused {
			t.Errorf("Expected ErrExecutionPaused, got %v", err)
		}

		if updatedExecution == nil {
			t.Fatal("Execution should have been updated")
		}

		if updatedExecution.Status != models.ExecutionStatusWaiting {
			t.Errorf("Expected status waiting, got %s", updatedExecution.Status)
		}

		if updatedExecution.WaitState == nil {
			t.Fatal("WaitState should be set")
		}

		if updatedExecution.WaitState.Event != "approval.granted" {
			t.Errorf("Expected event approval.granted, got %s", updatedExecution.WaitState.Event)
		}

		if updatedExecution.WaitState.TimeoutAt == nil {
			t.Error("TimeoutAt should be set")
		}

		if updatedExecution.CurrentStepID == nil || *updatedExecution.CurrentStepID != "wait1" {
			t.Error("CurrentStepID should be set to wait1")
		}
	})

	t.Run("handles missing wait config", func(t *testing.T) {
		repo := &mockExecutionRepo{}
		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

		execution := &models.WorkflowExecution{
			ID:     uuid.New(),
			Status: models.ExecutionStatusRunning,
		}

		step := &models.Step{
			ID:   "wait1",
			Type: "wait",
			Wait: nil,
		}

		err := executor.executeWaitStep(ctx, execution, step, map[string]interface{}{})

		if err == nil || err == ErrExecutionPaused {
			t.Error("Expected error for missing wait config")
		}
	})

	t.Run("handles invalid timeout duration", func(t *testing.T) {
		repo := &mockExecutionRepo{}
		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

		execution := &models.WorkflowExecution{
			ID:     uuid.New(),
			Status: models.ExecutionStatusRunning,
		}

		step := &models.Step{
			ID:   "wait1",
			Type: "wait",
			Wait: &models.WaitConfig{
				Event:   "approval.granted",
				Timeout: "invalid",
			},
		}

		err := executor.executeWaitStep(ctx, execution, step, map[string]interface{}{})

		if err == nil || err == ErrExecutionPaused {
			t.Error("Expected error for invalid timeout duration")
		}
	})
}

func TestResumeExecution(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	t.Run("resumes waiting execution", func(t *testing.T) {
		workflowID := uuid.New()
		executionID := uuid.New()

		waitingSince := time.Now().Add(-1 * time.Hour)
		timeout := time.Now().Add(23 * time.Hour)

		savedExecution := &models.WorkflowExecution{
			ID:            executionID,
			WorkflowID:    workflowID,
			ExecutionID:   "exec-123",
			Status:        models.ExecutionStatusWaiting,
			CurrentStepID: stringPtr("wait1"),
			WaitState: &models.WaitState{
				Event:        "approval.granted",
				TimeoutAt:    &timeout,
				WaitingSince: waitingSince,
			},
			Context: models.JSONB{
				"order": map[string]interface{}{
					"id":    "ord-123",
					"total": 1500.0,
				},
			},
			StartedAt: time.Now().Add(-2 * time.Hour),
		}

		var updatedExecution *models.WorkflowExecution
		var createdSteps []*models.StepExecution

		repo := &mockExecutionRepo{
			getExecutionByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				if id == executionID {
					return savedExecution, nil
				}
				return nil, errors.New("not found")
			},
			updateExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				updatedExecution = execution
				return nil
			},
			createStepExecutionFunc: func(ctx context.Context, step *models.StepExecution) error {
				createdSteps = append(createdSteps, step)
				return nil
			},
			updateStepExecutionFunc: func(ctx context.Context, step *models.StepExecution) error {
				return nil
			},
		}

		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

		workflow := &models.Workflow{
			ID:         workflowID,
			WorkflowID: "test-wf",
			Name:       "Test Workflow",
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type:  "event",
					Event: "order.created",
				},
				Steps: []models.Step{
					{
						ID:   "wait1",
						Type: "wait",
						Wait: &models.WaitConfig{
							Event:   "approval.granted",
							Timeout: "24h",
						},
					},
					{
						ID:   "action1",
						Type: "action",
						Action: &models.Action{
							Type: "allow",
						},
					},
				},
			},
		}

		resumeData := map[string]interface{}{
			"approved": true,
		}

		result, err := executor.ResumeExecution(ctx, executionID, workflow, "approval.granted", resumeData)

		if err != nil {
			t.Fatalf("ResumeExecution failed: %v", err)
		}

		if result == nil {
			t.Fatal("Result should not be nil")
		}

		if updatedExecution.Status != models.ExecutionStatusCompleted {
			t.Errorf("Expected status completed, got %s", updatedExecution.Status)
		}

		if len(createdSteps) == 0 {
			t.Error("Expected at least one step to be executed")
		}
	})

	t.Run("rejects execution not in waiting state", func(t *testing.T) {
		executionID := uuid.New()

		repo := &mockExecutionRepo{
			getExecutionByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return &models.WorkflowExecution{
					ID:     executionID,
					Status: models.ExecutionStatusCompleted,
				}, nil
			},
		}

		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

		workflow := &models.Workflow{
			Definition: models.WorkflowDefinition{
				Steps: []models.Step{},
			},
		}

		_, err := executor.ResumeExecution(ctx, executionID, workflow, "approval.granted", nil)

		if err == nil {
			t.Error("Expected error for non-waiting execution")
		}
	})

	t.Run("rejects wrong event type", func(t *testing.T) {
		executionID := uuid.New()

		repo := &mockExecutionRepo{
			getExecutionByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return &models.WorkflowExecution{
					ID:     executionID,
					Status: models.ExecutionStatusWaiting,
					WaitState: &models.WaitState{
						Event: "approval.granted",
					},
				}, nil
			},
		}

		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

		workflow := &models.Workflow{
			Definition: models.WorkflowDefinition{
				Steps: []models.Step{},
			},
		}

		_, err := executor.ResumeExecution(ctx, executionID, workflow, "wrong.event", nil)

		if err == nil {
			t.Error("Expected error for wrong event type")
		}
	})
}

func TestExecuteConditionStep(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	repo := &mockExecutionRepo{}
	executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())
	ctx := context.Background()

	t.Run("evaluates true condition", func(t *testing.T) {
		step := &models.Step{
			ID:   "cond1",
			Type: "condition",
			Condition: &models.Condition{
				Field:    "order.total",
				Operator: "gt",
				Value:    1000.0,
			},
			OnTrue:  "step_true",
			OnFalse: "step_false",
		}

		execContext := map[string]interface{}{
			"order": map[string]interface{}{
				"total": 1500.0,
			},
		}

		nextStep, err := executor.executeConditionStep(ctx, step, execContext)

		if err != nil {
			t.Fatalf("executeConditionStep failed: %v", err)
		}

		if nextStep != "step_true" {
			t.Errorf("Expected step_true, got %s", nextStep)
		}
	})

	t.Run("evaluates false condition", func(t *testing.T) {
		step := &models.Step{
			ID:   "cond1",
			Type: "condition",
			Condition: &models.Condition{
				Field:    "order.total",
				Operator: "gt",
				Value:    2000.0,
			},
			OnTrue:  "step_true",
			OnFalse: "step_false",
		}

		execContext := map[string]interface{}{
			"order": map[string]interface{}{
				"total": 1500.0,
			},
		}

		nextStep, err := executor.executeConditionStep(ctx, step, execContext)

		if err != nil {
			t.Fatalf("executeConditionStep failed: %v", err)
		}

		if nextStep != "step_false" {
			t.Errorf("Expected step_false, got %s", nextStep)
		}
	})

	t.Run("handles missing condition", func(t *testing.T) {
		step := &models.Step{
			ID:      "cond1",
			Type:    "condition",
			OnTrue:  "step_true",
			OnFalse: "step_false",
		}

		execContext := map[string]interface{}{}

		_, err := executor.executeConditionStep(ctx, step, execContext)

		if err == nil {
			t.Error("Expected error for missing condition")
		}
	})
}

func stringPtr(s string) *string {
	return &s
}

// TestExecuteWorkflow_Timeout tests that workflows timeout correctly
func TestExecuteWorkflow_Timeout(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	t.Run("workflow times out after default timeout", func(t *testing.T) {
		executionCreated := false
		executionCompleted := false

		repo := &mockExecutionRepo{
			createExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				executionCreated = true
				// Verify timeout is stored in metadata
				if _, ok := execution.Metadata["timeout_seconds"]; !ok {
					t.Error("Expected timeout_seconds in metadata")
				}
				// Note: timeout will be 1ns = 0.000000001 seconds
				return nil
			},
			updateExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				executionCompleted = true
				// Verify it was marked as failed due to timeout
				if execution.Status != models.ExecutionStatusFailed {
					t.Errorf("Expected status failed, got %s", execution.Status)
				}
				if execution.ErrorMessage == nil || !contains(*execution.ErrorMessage, "timed out") {
					t.Error("Expected error message to contain 'timed out'")
				}
				return nil
			},
		}

		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

		// Create a workflow with a single step
		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "timeout-workflow",
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

		// Set an impossibly short timeout to guarantee timeout
		executor.defaultTimeout = 1 * time.Nanosecond

		execution, err := executor.Execute(ctx, workflow, "test.event", map[string]interface{}{})

		if !executionCreated {
			t.Error("Expected execution to be created")
		}

		// Should timeout
		if err == nil {
			t.Error("Expected timeout error")
		}

		if execution == nil {
			t.Fatal("Expected execution to be returned")
		}

		// Give a moment for update to complete
		time.Sleep(50 * time.Millisecond)

		if !executionCompleted {
			t.Error("Expected execution to be marked as completed")
		}
	})

	t.Run("workflow completes before timeout", func(t *testing.T) {
		executionCreated := false
		executionCompleted := false

		repo := &mockExecutionRepo{
			createExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				executionCreated = true
				return nil
			},
			updateExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				executionCompleted = true
				// Should complete successfully
				if execution.Status != models.ExecutionStatusCompleted {
					t.Errorf("Expected status completed, got %s", execution.Status)
				}
				return nil
			},
		}

		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "fast-workflow",
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

		// Generous timeout
		executor.defaultTimeout = 5 * time.Second

		execution, err := executor.Execute(ctx, workflow, "test.event", map[string]interface{}{})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !executionCreated {
			t.Error("Expected execution to be created")
		}

		if !executionCompleted {
			t.Error("Expected execution to be completed")
		}

		if execution == nil {
			t.Fatal("Expected execution to be returned")
		}

		if execution.Status != models.ExecutionStatusCompleted {
			t.Errorf("Expected status completed, got %s", execution.Status)
		}
	})

	t.Run("workflow with custom timeout", func(t *testing.T) {
		customTimeout := 60.0 // seconds

		repo := &mockExecutionRepo{
			createExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				// Verify custom timeout is used
				if timeoutVal, ok := execution.Metadata["timeout_seconds"]; !ok {
					t.Error("Expected timeout_seconds in metadata")
				} else {
					if timeout, ok := timeoutVal.(float64); ok {
						if timeout != customTimeout {
							t.Errorf("Expected timeout %v, got %v", customTimeout, timeout)
						}
					}
				}
				return nil
			},
			updateExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				return nil
			},
		}

		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "custom-timeout-workflow",
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type: "event",
					Data: map[string]interface{}{
						"timeout_seconds": customTimeout,
					},
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

		// Default timeout is shorter
		executor.defaultTimeout = 5 * time.Second

		execution, err := executor.Execute(ctx, workflow, "test.event", map[string]interface{}{})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if execution == nil {
			t.Fatal("Expected execution to be returned")
		}
	})
}

// TestGetWorkflowTimeout tests the timeout extraction logic
func TestGetWorkflowTimeout(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	executor := NewWorkflowExecutor(redisClient, &mockExecutionRepo{}, nil, log, nil, getTestContextEnrichmentConfigForExecutor())

	tests := []struct {
		name     string
		workflow *models.Workflow
		expected time.Duration
	}{
		{
			name: "uses default timeout when no custom timeout set",
			workflow: &models.Workflow{
				Definition: models.WorkflowDefinition{
					Trigger: models.TriggerDefinition{
						Type: "event",
					},
				},
			},
			expected: 30 * time.Second, // default
		},
		{
			name: "uses custom timeout from trigger data (float64)",
			workflow: &models.Workflow{
				Definition: models.WorkflowDefinition{
					Trigger: models.TriggerDefinition{
						Type: "event",
						Data: map[string]interface{}{
							"timeout_seconds": 120.0,
						},
					},
				},
			},
			expected: 120 * time.Second,
		},
		{
			name: "uses custom timeout from trigger data (int)",
			workflow: &models.Workflow{
				Definition: models.WorkflowDefinition{
					Trigger: models.TriggerDefinition{
						Type: "event",
						Data: map[string]interface{}{
							"timeout_seconds": 90,
						},
					},
				},
			},
			expected: 90 * time.Second,
		},
		{
			name: "ignores invalid timeout (zero)",
			workflow: &models.Workflow{
				Definition: models.WorkflowDefinition{
					Trigger: models.TriggerDefinition{
						Type: "event",
						Data: map[string]interface{}{
							"timeout_seconds": 0,
						},
					},
				},
			},
			expected: 30 * time.Second, // falls back to default
		},
		{
			name: "ignores invalid timeout (negative)",
			workflow: &models.Workflow{
				Definition: models.WorkflowDefinition{
					Trigger: models.TriggerDefinition{
						Type: "event",
						Data: map[string]interface{}{
							"timeout_seconds": -10.0,
						},
					},
				},
			},
			expected: 30 * time.Second, // falls back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := executor.getWorkflowTimeout(tt.workflow)
			if timeout != tt.expected {
				t.Errorf("Expected timeout %v, got %v", tt.expected, timeout)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
