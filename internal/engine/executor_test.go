package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Mock ExecutionRepository
type mockExecutionRepo struct {
	createExecutionFunc     func(ctx context.Context, execution *models.WorkflowExecution) error
	updateExecutionFunc     func(ctx context.Context, execution *models.WorkflowExecution) error
	getExecutionByIDFunc    func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error)
	createStepExecutionFunc func(ctx context.Context, step *models.StepExecution) error
	updateStepExecutionFunc func(ctx context.Context, step *models.StepExecution) error
}

func (m *mockExecutionRepo) CreateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	if m.createExecutionFunc != nil {
		return m.createExecutionFunc(ctx, execution)
	}
	return nil
}

func (m *mockExecutionRepo) UpdateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	if m.updateExecutionFunc != nil {
		return m.updateExecutionFunc(ctx, execution)
	}
	return nil
}

func (m *mockExecutionRepo) GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
	if m.getExecutionByIDFunc != nil {
		return m.getExecutionByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockExecutionRepo) CreateStepExecution(ctx context.Context, step *models.StepExecution) error {
	if m.createStepExecutionFunc != nil {
		return m.createStepExecutionFunc(ctx, step)
	}
	return nil
}

func (m *mockExecutionRepo) UpdateStepExecution(ctx context.Context, step *models.StepExecution) error {
	if m.updateStepExecutionFunc != nil {
		return m.updateStepExecutionFunc(ctx, step)
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

		executor := NewWorkflowExecutor(redisClient, repo, nil, log)

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
		executor := NewWorkflowExecutor(redisClient, repo, nil, log)

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
		executor := NewWorkflowExecutor(redisClient, repo, nil, log)

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
		executor := NewWorkflowExecutor(redisClient, repo, nil, log)

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
		executor := NewWorkflowExecutor(redisClient, repo, nil, log)

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
		executor := NewWorkflowExecutor(redisClient, repo, nil, log)

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
	executor := NewWorkflowExecutor(redisClient, repo, nil, log)
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
