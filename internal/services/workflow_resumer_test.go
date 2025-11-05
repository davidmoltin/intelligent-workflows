package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
)

// Mock implementations
type mockWorkflowExecutor struct {
	resumeFunc func(
		ctx context.Context,
		executionID uuid.UUID,
		workflow *models.Workflow,
		resumeEvent string,
		resumeData map[string]interface{},
	) (*models.WorkflowExecution, error)
}

func (m *mockWorkflowExecutor) ResumeExecution(
	ctx context.Context,
	executionID uuid.UUID,
	workflow *models.Workflow,
	resumeEvent string,
	resumeData map[string]interface{},
) (*models.WorkflowExecution, error) {
	if m.resumeFunc != nil {
		return m.resumeFunc(ctx, executionID, workflow, resumeEvent, resumeData)
	}
	return nil, nil
}

type mockExecutionRepo struct {
	getByIDFunc    func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error)
	updateFunc     func(ctx context.Context, execution *models.WorkflowExecution) error
}

func (m *mockExecutionRepo) GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockExecutionRepo) UpdateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, execution)
	}
	return nil
}

type mockWorkflowRepo struct {
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*models.Workflow, error)
}

func (m *mockWorkflowRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func TestResumeWorkflow(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	t.Run("resumes approved workflow", func(t *testing.T) {
		executionID := uuid.New()
		workflowID := uuid.New()

		waitState := &models.WaitState{
			Event:        "approval.granted",
			WaitingSince: time.Now().Add(-1 * time.Hour),
		}

		execution := &models.WorkflowExecution{
			ID:         executionID,
			WorkflowID: workflowID,
			Status:     models.ExecutionStatusWaiting,
			WaitState:  waitState,
		}

		workflow := &models.Workflow{
			ID:         workflowID,
			WorkflowID: "test-wf",
			Definition: models.WorkflowDefinition{
				Steps: []models.Step{},
			},
		}

		var resumedExecution *models.WorkflowExecution
		var resumeEventCalled string

		mockExecutor := &mockWorkflowExecutor{
			resumeFunc: func(
				ctx context.Context,
				execID uuid.UUID,
				wf *models.Workflow,
				event string,
				data map[string]interface{},
			) (*models.WorkflowExecution, error) {
				resumedExecution = execution
				resumeEventCalled = event
				return execution, nil
			},
		}

		mockExecRepo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				if id == executionID {
					return execution, nil
				}
				return nil, errors.New("not found")
			},
		}

		mockWfRepo := &mockWorkflowRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
				if id == workflowID {
					return workflow, nil
				}
				return nil, errors.New("not found")
			},
		}

		resumer := NewWorkflowResumer(mockExecutor, mockExecRepo, mockWfRepo, log)

		err := resumer.ResumeWorkflow(ctx, executionID, true)

		if err != nil {
			t.Fatalf("ResumeWorkflow failed: %v", err)
		}

		if resumedExecution == nil {
			t.Error("Execution should have been resumed")
		}

		if resumeEventCalled != "approval.granted" {
			t.Errorf("Expected event approval.granted, got %s", resumeEventCalled)
		}
	})

	t.Run("cancels rejected workflow", func(t *testing.T) {
		executionID := uuid.New()
		workflowID := uuid.New()

		waitState := &models.WaitState{
			Event:        "approval.granted",
			WaitingSince: time.Now().Add(-1 * time.Hour),
		}

		execution := &models.WorkflowExecution{
			ID:         executionID,
			WorkflowID: workflowID,
			Status:     models.ExecutionStatusWaiting,
			WaitState:  waitState,
		}

		workflow := &models.Workflow{
			ID:         workflowID,
			WorkflowID: "test-wf",
		}

		var updatedExecution *models.WorkflowExecution

		mockExecutor := &mockWorkflowExecutor{}

		mockExecRepo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
			updateFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				updatedExecution = exec
				return nil
			},
		}

		mockWfRepo := &mockWorkflowRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
				return workflow, nil
			},
		}

		resumer := NewWorkflowResumer(mockExecutor, mockExecRepo, mockWfRepo, log)

		err := resumer.ResumeWorkflow(ctx, executionID, false)

		if err != nil {
			t.Fatalf("ResumeWorkflow failed: %v", err)
		}

		if updatedExecution == nil {
			t.Fatal("Execution should have been updated")
		}

		if updatedExecution.Status != models.ExecutionStatusCancelled {
			t.Errorf("Expected status cancelled, got %s", updatedExecution.Status)
		}

		if updatedExecution.Result == nil || *updatedExecution.Result != models.ExecutionResultBlocked {
			t.Error("Expected result blocked")
		}
	})

	t.Run("rejects non-waiting execution", func(t *testing.T) {
		executionID := uuid.New()

		execution := &models.WorkflowExecution{
			ID:     executionID,
			Status: models.ExecutionStatusCompleted,
		}

		mockExecutor := &mockWorkflowExecutor{}

		mockExecRepo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
		}

		mockWfRepo := &mockWorkflowRepo{}

		resumer := NewWorkflowResumer(mockExecutor, mockExecRepo, mockWfRepo, log)

		err := resumer.ResumeWorkflow(ctx, executionID, true)

		if err == nil {
			t.Error("Expected error for non-waiting execution")
		}
	})

	t.Run("handles missing execution", func(t *testing.T) {
		executionID := uuid.New()

		mockExecutor := &mockWorkflowExecutor{}

		mockExecRepo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return nil, errors.New("not found")
			},
		}

		mockWfRepo := &mockWorkflowRepo{}

		resumer := NewWorkflowResumer(mockExecutor, mockExecRepo, mockWfRepo, log)

		err := resumer.ResumeWorkflow(ctx, executionID, true)

		if err == nil {
			t.Error("Expected error for missing execution")
		}
	})

	t.Run("handles missing workflow", func(t *testing.T) {
		executionID := uuid.New()
		workflowID := uuid.New()

		execution := &models.WorkflowExecution{
			ID:         executionID,
			WorkflowID: workflowID,
			Status:     models.ExecutionStatusWaiting,
			WaitState:  &models.WaitState{},
		}

		mockExecutor := &mockWorkflowExecutor{}

		mockExecRepo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
		}

		mockWfRepo := &mockWorkflowRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
				return nil, errors.New("not found")
			},
		}

		resumer := NewWorkflowResumer(mockExecutor, mockExecRepo, mockWfRepo, log)

		err := resumer.ResumeWorkflow(ctx, executionID, true)

		if err == nil {
			t.Error("Expected error for missing workflow")
		}
	})
}
