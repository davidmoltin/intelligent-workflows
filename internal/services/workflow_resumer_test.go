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

// MockExecutionRepository for testing
type mockExecutionRepo struct {
	getByIDFunc          func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error)
	updateFunc           func(ctx context.Context, execution *models.WorkflowExecution) error
	getPausedFunc        func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error)
}

func (m *mockExecutionRepo) CreateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	return nil
}

func (m *mockExecutionRepo) UpdateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, execution)
	}
	return nil
}

func (m *mockExecutionRepo) GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockExecutionRepo) GetPausedExecutions(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
	if m.getPausedFunc != nil {
		return m.getPausedFunc(ctx, limit)
	}
	return nil, errors.New("not implemented")
}

// MockWorkflowEngine for testing
type mockWorkflowEngine struct {
	resumeFunc func(ctx context.Context, execution *models.WorkflowExecution) error
}

func (m *mockWorkflowEngine) ResumePausedExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	if m.resumeFunc != nil {
		return m.resumeFunc(ctx, execution)
	}
	return nil
}

// TestPauseExecution tests the PauseExecution method
func TestPauseExecution(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	t.Run("successfully pauses running execution", func(t *testing.T) {
		executionID := uuid.New()
		execution := &models.WorkflowExecution{
			ID:     executionID,
			Status: models.ExecutionStatusRunning,
		}

		var updatedExecution *models.WorkflowExecution
		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				if id == executionID {
					return execution, nil
				}
				return nil, errors.New("not found")
			},
			updateFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				updatedExecution = exec
				return nil
			},
		}

		resumer := NewWorkflowResumer(log, repo, nil)

		stepID := uuid.New()
		err := resumer.PauseExecution(ctx, executionID, "test pause", &stepID)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if updatedExecution == nil {
			t.Fatal("Expected execution to be updated")
		}

		if updatedExecution.Status != models.ExecutionStatusPaused {
			t.Errorf("Expected status paused, got %s", updatedExecution.Status)
		}

		if updatedExecution.PausedAt == nil {
			t.Error("Expected PausedAt to be set")
		}

		if updatedExecution.PausedReason == nil || *updatedExecution.PausedReason != "test pause" {
			t.Error("Expected PausedReason to be set")
		}

		if updatedExecution.PausedStepID == nil || *updatedExecution.PausedStepID != stepID {
			t.Error("Expected PausedStepID to be set")
		}
	})

	t.Run("fails to pause non-running execution", func(t *testing.T) {
		executionID := uuid.New()
		execution := &models.WorkflowExecution{
			ID:     executionID,
			Status: models.ExecutionStatusCompleted,
		}

		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
		}

		resumer := NewWorkflowResumer(log, repo, nil)

		err := resumer.PauseExecution(ctx, executionID, "test", nil)

		if err == nil {
			t.Fatal("Expected error for non-running execution")
		}

		if err.Error() != "execution "+executionID.String()+" is not running (status: completed)" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("fails when execution not found", func(t *testing.T) {
		executionID := uuid.New()

		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return nil, errors.New("not found")
			},
		}

		resumer := NewWorkflowResumer(log, repo, nil)

		err := resumer.PauseExecution(ctx, executionID, "test", nil)

		if err == nil {
			t.Fatal("Expected error when execution not found")
		}
	})

	t.Run("fails when update fails", func(t *testing.T) {
		executionID := uuid.New()
		execution := &models.WorkflowExecution{
			ID:     executionID,
			Status: models.ExecutionStatusRunning,
		}

		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
			updateFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				return errors.New("database error")
			},
		}

		resumer := NewWorkflowResumer(log, repo, nil)

		err := resumer.PauseExecution(ctx, executionID, "test", nil)

		if err == nil {
			t.Fatal("Expected error when update fails")
		}
	})

	t.Run("fails when repository is nil", func(t *testing.T) {
		resumer := NewWorkflowResumer(log, nil, nil)

		err := resumer.PauseExecution(ctx, uuid.New(), "test", nil)

		if err == nil {
			t.Fatal("Expected error when repository is nil")
		}

		if err.Error() != "execution repository not configured" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

// TestResumeWorkflow tests the ResumeWorkflow method
func TestResumeWorkflow(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	t.Run("successfully resumes paused execution with approval", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)
		execution := &models.WorkflowExecution{
			ID:          executionID,
			Status:      models.ExecutionStatusPaused,
			PausedAt:    &pausedAt,
			ResumeCount: 0,
		}

		var updatedExecution *models.WorkflowExecution
		var engineCalled bool

		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
			updateFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				updatedExecution = exec
				return nil
			},
		}

		engine := &mockWorkflowEngine{
			resumeFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				engineCalled = true
				return nil
			},
		}

		resumer := NewWorkflowResumer(log, repo, engine)

		err := resumer.ResumeWorkflow(ctx, executionID, true)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !engineCalled {
			t.Error("Expected engine to be called")
		}

		if updatedExecution == nil {
			t.Fatal("Expected execution to be updated")
		}

		if updatedExecution.Status != models.ExecutionStatusRunning {
			t.Errorf("Expected status running, got %s", updatedExecution.Status)
		}

		if updatedExecution.ResumeCount != 1 {
			t.Errorf("Expected resume count 1, got %d", updatedExecution.ResumeCount)
		}

		if updatedExecution.PausedAt != nil {
			t.Error("Expected PausedAt to be cleared")
		}

		if updatedExecution.PausedReason != nil {
			t.Error("Expected PausedReason to be cleared")
		}

		if updatedExecution.LastResumedAt == nil {
			t.Error("Expected LastResumedAt to be set")
		}

		if updatedExecution.ResumeData == nil {
			t.Fatal("Expected ResumeData to be set")
		}

		if approved, ok := updatedExecution.ResumeData["approved"].(bool); !ok || !approved {
			t.Error("Expected approved to be true in ResumeData")
		}
	})

	t.Run("successfully resumes with rejection", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)
		execution := &models.WorkflowExecution{
			ID:       executionID,
			Status:   models.ExecutionStatusPaused,
			PausedAt: &pausedAt,
		}

		var updatedExecution *models.WorkflowExecution

		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
			updateFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				updatedExecution = exec
				return nil
			},
		}

		engine := &mockWorkflowEngine{
			resumeFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				return nil
			},
		}

		resumer := NewWorkflowResumer(log, repo, engine)

		err := resumer.ResumeWorkflow(ctx, executionID, false)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if approved, ok := updatedExecution.ResumeData["approved"].(bool); !ok || approved {
			t.Error("Expected approved to be false in ResumeData")
		}
	})

	t.Run("fails to resume non-paused execution", func(t *testing.T) {
		executionID := uuid.New()
		execution := &models.WorkflowExecution{
			ID:     executionID,
			Status: models.ExecutionStatusRunning,
		}

		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
		}

		resumer := NewWorkflowResumer(log, repo, nil)

		err := resumer.ResumeWorkflow(ctx, executionID, true)

		if err == nil {
			t.Fatal("Expected error for non-paused execution")
		}
	})

	t.Run("handles nil repository gracefully", func(t *testing.T) {
		resumer := NewWorkflowResumer(log, nil, nil)

		err := resumer.ResumeWorkflow(ctx, uuid.New(), true)

		// Should return nil for backward compatibility
		if err != nil {
			t.Errorf("Expected nil error for backward compat, got %v", err)
		}
	})
}

// TestResumeExecution tests the ResumeExecution method
func TestResumeExecution(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	t.Run("successfully resumes with custom resume data", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)
		execution := &models.WorkflowExecution{
			ID:       executionID,
			Status:   models.ExecutionStatusPaused,
			PausedAt: &pausedAt,
			ResumeData: models.JSONB{
				"existing_key": "existing_value",
			},
		}

		var updatedExecution *models.WorkflowExecution

		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
			updateFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				updatedExecution = exec
				return nil
			},
		}

		engine := &mockWorkflowEngine{
			resumeFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				return nil
			},
		}

		resumer := NewWorkflowResumer(log, repo, engine)

		customData := models.JSONB{
			"custom_key": "custom_value",
			"user_input": 42,
		}

		err := resumer.ResumeExecution(ctx, executionID, customData)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if updatedExecution.ResumeData["existing_key"] != "existing_value" {
			t.Error("Expected existing data to be preserved")
		}

		if updatedExecution.ResumeData["custom_key"] != "custom_value" {
			t.Error("Expected custom data to be merged")
		}

		if updatedExecution.ResumeData["user_input"] != 42 {
			t.Error("Expected user input to be merged")
		}
	})

	t.Run("creates resume data if nil", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)
		execution := &models.WorkflowExecution{
			ID:         executionID,
			Status:     models.ExecutionStatusPaused,
			PausedAt:   &pausedAt,
			ResumeData: nil,
		}

		var updatedExecution *models.WorkflowExecution

		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
			updateFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				updatedExecution = exec
				return nil
			},
		}

		engine := &mockWorkflowEngine{}

		resumer := NewWorkflowResumer(log, repo, engine)

		customData := models.JSONB{
			"key": "value",
		}

		err := resumer.ResumeExecution(ctx, executionID, customData)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if updatedExecution.ResumeData == nil {
			t.Fatal("Expected ResumeData to be created")
		}

		if updatedExecution.ResumeData["key"] != "value" {
			t.Error("Expected custom data to be set")
		}
	})

	t.Run("fails when repository is nil", func(t *testing.T) {
		resumer := NewWorkflowResumer(log, nil, nil)

		err := resumer.ResumeExecution(ctx, uuid.New(), models.JSONB{})

		if err == nil {
			t.Fatal("Expected error when repository is nil")
		}
	})

	t.Run("works without engine (state update only)", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)
		execution := &models.WorkflowExecution{
			ID:       executionID,
			Status:   models.ExecutionStatusPaused,
			PausedAt: &pausedAt,
		}

		var updatedExecution *models.WorkflowExecution

		repo := &mockExecutionRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return execution, nil
			},
			updateFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
				updatedExecution = exec
				return nil
			},
		}

		resumer := NewWorkflowResumer(log, repo, nil)

		err := resumer.ResumeExecution(ctx, executionID, models.JSONB{})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if updatedExecution.Status != models.ExecutionStatusRunning {
			t.Error("Expected status to be updated even without engine")
		}
	})
}

// TestGetPausedExecutions tests the GetPausedExecutions method
func TestGetPausedExecutions(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	t.Run("successfully retrieves paused executions", func(t *testing.T) {
		pausedAt := time.Now().Add(-1 * time.Hour)
		expected := []*models.WorkflowExecution{
			{
				ID:       uuid.New(),
				Status:   models.ExecutionStatusPaused,
				PausedAt: &pausedAt,
			},
			{
				ID:       uuid.New(),
				Status:   models.ExecutionStatusPaused,
				PausedAt: &pausedAt,
			},
		}

		repo := &mockExecutionRepo{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				if limit != 50 {
					t.Errorf("Expected limit 50, got %d", limit)
				}
				return expected, nil
			},
		}

		resumer := NewWorkflowResumer(log, repo, nil)

		result, err := resumer.GetPausedExecutions(ctx, 50)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 executions, got %d", len(result))
		}
	})

	t.Run("returns empty list when no paused executions", func(t *testing.T) {
		repo := &mockExecutionRepo{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{}, nil
			},
		}

		resumer := NewWorkflowResumer(log, repo, nil)

		result, err := resumer.GetPausedExecutions(ctx, 50)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result) != 0 {
			t.Errorf("Expected 0 executions, got %d", len(result))
		}
	})

	t.Run("fails when repository is nil", func(t *testing.T) {
		resumer := NewWorkflowResumer(log, nil, nil)

		_, err := resumer.GetPausedExecutions(ctx, 50)

		if err == nil {
			t.Fatal("Expected error when repository is nil")
		}
	})

	t.Run("propagates repository errors", func(t *testing.T) {
		repo := &mockExecutionRepo{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return nil, errors.New("database error")
			},
		}

		resumer := NewWorkflowResumer(log, repo, nil)

		_, err := resumer.GetPausedExecutions(ctx, 50)

		if err == nil {
			t.Fatal("Expected error to be propagated")
		}
	})
}

// TestCanResume tests the CanResume validation method
func TestCanResume(t *testing.T) {
	log := logger.NewForTesting()
	resumer := NewWorkflowResumer(log, nil, nil)

	t.Run("allows resuming valid paused execution", func(t *testing.T) {
		pausedAt := time.Now().Add(-1 * time.Hour)
		execution := &models.WorkflowExecution{
			ID:       uuid.New(),
			Status:   models.ExecutionStatusPaused,
			PausedAt: &pausedAt,
		}

		err := resumer.CanResume(execution)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("rejects nil execution", func(t *testing.T) {
		err := resumer.CanResume(nil)

		if err == nil {
			t.Fatal("Expected error for nil execution")
		}

		if err.Error() != "execution is nil" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("rejects non-paused execution", func(t *testing.T) {
		execution := &models.WorkflowExecution{
			ID:     uuid.New(),
			Status: models.ExecutionStatusRunning,
		}

		err := resumer.CanResume(execution)

		if err == nil {
			t.Fatal("Expected error for non-paused execution")
		}
	})

	t.Run("rejects execution without pause timestamp", func(t *testing.T) {
		execution := &models.WorkflowExecution{
			ID:       uuid.New(),
			Status:   models.ExecutionStatusPaused,
			PausedAt: nil,
		}

		err := resumer.CanResume(execution)

		if err == nil {
			t.Fatal("Expected error for execution without pause timestamp")
		}
	})

	t.Run("rejects execution paused too long", func(t *testing.T) {
		// Paused 8 days ago (> 7 day limit)
		pausedAt := time.Now().Add(-8 * 24 * time.Hour)
		execution := &models.WorkflowExecution{
			ID:       uuid.New(),
			Status:   models.ExecutionStatusPaused,
			PausedAt: &pausedAt,
		}

		err := resumer.CanResume(execution)

		if err == nil {
			t.Fatal("Expected error for execution paused too long")
		}

		if err.Error() != "execution "+execution.ID.String()+" has been paused for too long (paused at: "+pausedAt.String()+")" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("allows execution paused just under limit", func(t *testing.T) {
		// Paused 6.9 days ago (< 7 day limit)
		pausedAt := time.Now().Add(-6*24*time.Hour - 23*time.Hour)
		execution := &models.WorkflowExecution{
			ID:       uuid.New(),
			Status:   models.ExecutionStatusPaused,
			PausedAt: &pausedAt,
		}

		err := resumer.CanResume(execution)

		if err != nil {
			t.Errorf("Expected no error for execution within time limit, got %v", err)
		}
	})
}

// TestResumeExecution_EngineFailure tests engine failure handling
func TestResumeExecution_EngineFailure(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	executionID := uuid.New()
	pausedAt := time.Now().Add(-1 * time.Hour)
	execution := &models.WorkflowExecution{
		ID:       executionID,
		Status:   models.ExecutionStatusPaused,
		PausedAt: &pausedAt,
	}

	var executionUpdated bool

	repo := &mockExecutionRepo{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
			return execution, nil
		},
		updateFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
			executionUpdated = true
			return nil
		},
	}

	engine := &mockWorkflowEngine{
		resumeFunc: func(ctx context.Context, exec *models.WorkflowExecution) error {
			return errors.New("engine failure")
		},
	}

	resumer := NewWorkflowResumer(log, repo, engine)

	err := resumer.ResumeExecution(ctx, executionID, models.JSONB{})

	if err == nil {
		t.Fatal("Expected error when engine fails")
	}

	if !executionUpdated {
		t.Error("Expected execution to be updated before engine call")
	}
}
