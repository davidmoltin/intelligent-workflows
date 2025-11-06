package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/internal/workers"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// MockWorkflowEngine for testing
type MockWorkflowEngine struct {
	mock.Mock
}

func (m *MockWorkflowEngine) ResumePausedExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func TestWorkflowResumer_PauseAndResume_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create repositories
	executionRepo := postgres.NewExecutionRepository(suite.DB.DB)

	// Create mock engine
	mockEngine := new(MockWorkflowEngine)

	// Create workflow resumer
	resumer := services.NewWorkflowResumer(log, executionRepo, mockEngine)

	ctx := context.Background()

	// Create a running execution
	workflowID := uuid.New()
	execution := &models.WorkflowExecution{
		ID:             uuid.New(),
		WorkflowID:     workflowID,
		ExecutionID:    "test-exec-001",
		TriggerEvent:   "test.event",
		TriggerPayload: models.JSONB{"test": "data"},
		Context:        models.JSONB{},
		Status:         models.ExecutionStatusRunning,
		StartedAt:      time.Now(),
		Metadata:       models.JSONB{},
	}
	err = executionRepo.CreateExecution(ctx, execution)
	require.NoError(t, err)

	// Pause the execution
	stepID := uuid.New()
	pauseReason := "waiting for approval"
	err = resumer.PauseExecution(ctx, execution.ID, pauseReason, &stepID)
	require.NoError(t, err)

	// Verify execution is paused
	pausedExec, err := executionRepo.GetExecutionByID(ctx, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusPaused, pausedExec.Status)
	assert.NotNil(t, pausedExec.PausedAt)
	assert.NotNil(t, pausedExec.PausedReason)
	assert.Equal(t, pauseReason, *pausedExec.PausedReason)
	assert.NotNil(t, pausedExec.PausedStepID)
	assert.Equal(t, stepID, *pausedExec.PausedStepID)

	// Setup mock engine expectation for resume
	mockEngine.On("ResumePausedExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	// Resume the execution
	err = resumer.ResumeWorkflow(ctx, execution.ID, true)
	require.NoError(t, err)

	// Verify execution is running again
	resumedExec, err := executionRepo.GetExecutionByID(ctx, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusRunning, resumedExec.Status)
	assert.Nil(t, resumedExec.PausedAt)
	assert.Nil(t, resumedExec.PausedReason)
	assert.NotNil(t, resumedExec.LastResumedAt)
	assert.Equal(t, 1, resumedExec.ResumeCount)
	assert.NotNil(t, resumedExec.ResumeData)
	assert.Equal(t, true, resumedExec.ResumeData["approved"])

	mockEngine.AssertExpectations(t)
}

func TestWorkflowResumer_GetPausedExecutions_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create repositories
	executionRepo := postgres.NewExecutionRepository(suite.DB.DB)

	// Create workflow resumer
	resumer := services.NewWorkflowResumer(log, executionRepo, nil)

	ctx := context.Background()
	workflowID := uuid.New()

	// Create multiple paused executions
	pausedAt1 := time.Now().Add(-2 * time.Hour)
	pausedAt2 := time.Now().Add(-1 * time.Hour)
	pausedAt3 := time.Now().Add(-30 * time.Minute)
	pausedReason := "waiting for approval"

	executions := []*models.WorkflowExecution{
		{
			ID:             uuid.New(),
			WorkflowID:     workflowID,
			ExecutionID:    "test-exec-001",
			TriggerEvent:   "test.event",
			TriggerPayload: models.JSONB{"test": "data"},
			Context:        models.JSONB{},
			Status:         models.ExecutionStatusPaused,
			StartedAt:      time.Now().Add(-3 * time.Hour),
			PausedAt:       &pausedAt1,
			PausedReason:   &pausedReason,
			Metadata:       models.JSONB{},
		},
		{
			ID:             uuid.New(),
			WorkflowID:     workflowID,
			ExecutionID:    "test-exec-002",
			TriggerEvent:   "test.event",
			TriggerPayload: models.JSONB{"test": "data"},
			Context:        models.JSONB{},
			Status:         models.ExecutionStatusPaused,
			StartedAt:      time.Now().Add(-2 * time.Hour),
			PausedAt:       &pausedAt2,
			PausedReason:   &pausedReason,
			Metadata:       models.JSONB{},
		},
		{
			ID:             uuid.New(),
			WorkflowID:     workflowID,
			ExecutionID:    "test-exec-003",
			TriggerEvent:   "test.event",
			TriggerPayload: models.JSONB{"test": "data"},
			Context:        models.JSONB{},
			Status:         models.ExecutionStatusPaused,
			StartedAt:      time.Now().Add(-1 * time.Hour),
			PausedAt:       &pausedAt3,
			PausedReason:   &pausedReason,
			Metadata:       models.JSONB{},
		},
		// Create a running execution (should not be returned)
		{
			ID:             uuid.New(),
			WorkflowID:     workflowID,
			ExecutionID:    "test-exec-004",
			TriggerEvent:   "test.event",
			TriggerPayload: models.JSONB{"test": "data"},
			Context:        models.JSONB{},
			Status:         models.ExecutionStatusRunning,
			StartedAt:      time.Now(),
			Metadata:       models.JSONB{},
		},
	}

	for _, exec := range executions {
		err = executionRepo.CreateExecution(ctx, exec)
		require.NoError(t, err)
	}

	// Get paused executions
	pausedExecs, err := resumer.GetPausedExecutions(ctx, 10)
	require.NoError(t, err)

	// Should return only the 3 paused executions, ordered by paused_at (oldest first)
	assert.Len(t, pausedExecs, 3)
	assert.Equal(t, "test-exec-001", pausedExecs[0].ExecutionID) // Oldest pause
	assert.Equal(t, "test-exec-002", pausedExecs[1].ExecutionID)
	assert.Equal(t, "test-exec-003", pausedExecs[2].ExecutionID) // Newest pause
}

func TestWorkflowResumer_MultipleResumesCycle_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create repositories
	executionRepo := postgres.NewExecutionRepository(suite.DB.DB)

	// Create mock engine
	mockEngine := new(MockWorkflowEngine)
	mockEngine.On("ResumePausedExecution", mock.Anything, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	// Create workflow resumer
	resumer := services.NewWorkflowResumer(log, executionRepo, mockEngine)

	ctx := context.Background()
	workflowID := uuid.New()

	// Create a running execution
	execution := &models.WorkflowExecution{
		ID:             uuid.New(),
		WorkflowID:     workflowID,
		ExecutionID:    "test-exec-multi-resume",
		TriggerEvent:   "test.event",
		TriggerPayload: models.JSONB{"test": "data"},
		Context:        models.JSONB{},
		Status:         models.ExecutionStatusRunning,
		StartedAt:      time.Now(),
		Metadata:       models.JSONB{},
	}
	err = executionRepo.CreateExecution(ctx, execution)
	require.NoError(t, err)

	// Test multiple pause/resume cycles
	for i := 1; i <= 3; i++ {
		// Pause
		pauseReason := "pause iteration"
		err = resumer.PauseExecution(ctx, execution.ID, pauseReason, nil)
		require.NoError(t, err, "pause iteration %d failed", i)

		// Verify paused
		pausedExec, err := executionRepo.GetExecutionByID(ctx, execution.ID)
		require.NoError(t, err, "get execution iteration %d failed", i)
		assert.Equal(t, models.ExecutionStatusPaused, pausedExec.Status)

		// Resume
		resumeData := models.JSONB{
			"iteration": i,
			"approved":  true,
		}
		err = resumer.ResumeExecution(ctx, execution.ID, resumeData)
		require.NoError(t, err, "resume iteration %d failed", i)

		// Verify resumed
		resumedExec, err := executionRepo.GetExecutionByID(ctx, execution.ID)
		require.NoError(t, err, "get resumed execution iteration %d failed", i)
		assert.Equal(t, models.ExecutionStatusRunning, resumedExec.Status)
		assert.Equal(t, i, resumedExec.ResumeCount, "resume count should be %d", i)

		// Convert to float64 for comparison since JSON numbers are float64
		iterValue, ok := resumedExec.ResumeData["iteration"].(float64)
		assert.True(t, ok)
		assert.Equal(t, float64(i), iterValue)
	}

	// Final verification
	finalExec, err := executionRepo.GetExecutionByID(ctx, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, finalExec.ResumeCount)
	assert.NotNil(t, finalExec.LastResumedAt)

	mockEngine.AssertExpectations(t)
}

func TestWorkflowResumer_CanResume_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create repositories
	executionRepo := postgres.NewExecutionRepository(suite.DB.DB)

	// Create workflow resumer
	resumer := services.NewWorkflowResumer(log, executionRepo, nil)

	ctx := context.Background()
	workflowID := uuid.New()

	tests := []struct {
		name          string
		execution     *models.WorkflowExecution
		shouldSucceed bool
		errorContains string
	}{
		{
			name: "valid paused execution",
			execution: &models.WorkflowExecution{
				ID:             uuid.New(),
				WorkflowID:     workflowID,
				ExecutionID:    "test-can-resume-valid",
				TriggerEvent:   "test.event",
				TriggerPayload: models.JSONB{"test": "data"},
				Context:        models.JSONB{},
				Status:         models.ExecutionStatusPaused,
				StartedAt:      time.Now().Add(-2 * time.Hour),
				PausedAt:       func() *time.Time { t := time.Now().Add(-1 * time.Hour); return &t }(),
				PausedReason:   func() *string { s := "test"; return &s }(),
				Metadata:       models.JSONB{},
			},
			shouldSucceed: true,
		},
		{
			name: "not paused execution",
			execution: &models.WorkflowExecution{
				ID:             uuid.New(),
				WorkflowID:     workflowID,
				ExecutionID:    "test-can-resume-running",
				TriggerEvent:   "test.event",
				TriggerPayload: models.JSONB{"test": "data"},
				Context:        models.JSONB{},
				Status:         models.ExecutionStatusRunning,
				StartedAt:      time.Now(),
				Metadata:       models.JSONB{},
			},
			shouldSucceed: false,
			errorContains: "is not paused",
		},
		{
			name: "paused too long",
			execution: &models.WorkflowExecution{
				ID:             uuid.New(),
				WorkflowID:     workflowID,
				ExecutionID:    "test-can-resume-expired",
				TriggerEvent:   "test.event",
				TriggerPayload: models.JSONB{"test": "data"},
				Context:        models.JSONB{},
				Status:         models.ExecutionStatusPaused,
				StartedAt:      time.Now().Add(-10 * 24 * time.Hour),
				PausedAt:       func() *time.Time { t := time.Now().Add(-8 * 24 * time.Hour); return &t }(),
				PausedReason:   func() *string { s := "test"; return &s }(),
				Metadata:       models.JSONB{},
			},
			shouldSucceed: false,
			errorContains: "has been paused for too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create execution
			err = executionRepo.CreateExecution(ctx, tt.execution)
			require.NoError(t, err)

			// Test can resume
			err = resumer.CanResume(tt.execution)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			}
		})
	}
}

func TestWorkflowResumerWorker_ProcessPausedExecutions_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create repositories
	executionRepo := postgres.NewExecutionRepository(suite.DB.DB)

	// Create mock engine
	mockEngine := new(MockWorkflowEngine)
	mockEngine.On("ResumePausedExecution", mock.Anything, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	// Create workflow resumer and worker
	resumer := services.NewWorkflowResumer(log, executionRepo, mockEngine)
	worker := workers.NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	ctx := context.Background()
	workflowID := uuid.New()

	// Create paused executions with approval decisions
	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"

	executions := []struct {
		executionID string
		approved    bool
	}{
		{"test-worker-001", true},
		{"test-worker-002", false},
	}

	for _, exec := range executions {
		execution := &models.WorkflowExecution{
			ID:             uuid.New(),
			WorkflowID:     workflowID,
			ExecutionID:    exec.executionID,
			TriggerEvent:   "test.event",
			TriggerPayload: models.JSONB{"test": "data"},
			Context:        models.JSONB{},
			Status:         models.ExecutionStatusPaused,
			StartedAt:      time.Now().Add(-2 * time.Hour),
			PausedAt:       &pausedAt,
			PausedReason:   &pausedReason,
			ResumeData:     models.JSONB{"approved": exec.approved},
			Metadata:       models.JSONB{},
		}
		err = executionRepo.CreateExecution(ctx, execution)
		require.NoError(t, err)
	}

	// Start worker
	worker.Start(ctx)

	// Let worker process executions
	time.Sleep(200 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// Verify executions were resumed
	for _, exec := range executions {
		execution, err := executionRepo.GetExecutionByExecutionID(ctx, exec.executionID)
		require.NoError(t, err)
		assert.Equal(t, models.ExecutionStatusRunning, execution.Status, "execution %s should be running", exec.executionID)
		assert.Equal(t, 1, execution.ResumeCount, "execution %s should have resume count 1", exec.executionID)
		assert.Equal(t, exec.approved, execution.ResumeData["approved"], "execution %s should have correct approval", exec.executionID)
	}

	mockEngine.AssertExpectations(t)
}

func TestWorkflowResumer_ResumeData_MergeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create repositories
	executionRepo := postgres.NewExecutionRepository(suite.DB.DB)

	// Create mock engine
	mockEngine := new(MockWorkflowEngine)
	mockEngine.On("ResumePausedExecution", mock.Anything, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	// Create workflow resumer
	resumer := services.NewWorkflowResumer(log, executionRepo, mockEngine)

	ctx := context.Background()
	workflowID := uuid.New()

	// Create a paused execution with existing resume data
	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"
	execution := &models.WorkflowExecution{
		ID:             uuid.New(),
		WorkflowID:     workflowID,
		ExecutionID:    "test-merge-resume-data",
		TriggerEvent:   "test.event",
		TriggerPayload: models.JSONB{"test": "data"},
		Context:        models.JSONB{},
		Status:         models.ExecutionStatusPaused,
		StartedAt:      time.Now().Add(-2 * time.Hour),
		PausedAt:       &pausedAt,
		PausedReason:   &pausedReason,
		ResumeData: models.JSONB{
			"initial_key": "initial_value",
			"count":       1,
		},
		Metadata: models.JSONB{},
	}
	err = executionRepo.CreateExecution(ctx, execution)
	require.NoError(t, err)

	// Resume with additional data
	additionalData := models.JSONB{
		"approved": true,
		"approver": "test@example.com",
		"count":    2, // Should override existing value
		"new_key":  "new_value",
	}
	err = resumer.ResumeExecution(ctx, execution.ID, additionalData)
	require.NoError(t, err)

	// Verify resume data was merged correctly
	resumedExec, err := executionRepo.GetExecutionByID(ctx, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusRunning, resumedExec.Status)
	assert.Equal(t, "initial_value", resumedExec.ResumeData["initial_key"])
	assert.Equal(t, true, resumedExec.ResumeData["approved"])
	assert.Equal(t, "test@example.com", resumedExec.ResumeData["approver"])
	assert.Equal(t, "new_value", resumedExec.ResumeData["new_key"])
	// Count should be overridden to 2
	countValue, ok := resumedExec.ResumeData["count"].(float64) // JSON numbers are float64
	assert.True(t, ok)
	assert.Equal(t, float64(2), countValue)

	mockEngine.AssertExpectations(t)
}
