package workers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// MockExecutionRepository is a mock implementation of ExecutionRepository
type MockExecutionRepository struct {
	mock.Mock
}

func (m *MockExecutionRepository) CreateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *MockExecutionRepository) UpdateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *MockExecutionRepository) GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WorkflowExecution), args.Error(1)
}

func (m *MockExecutionRepository) GetPausedExecutions(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.WorkflowExecution), args.Error(1)
}

// MockWorkflowEngine is a mock implementation of WorkflowEngine
type MockWorkflowEngine struct {
	mock.Mock
}

func (m *MockWorkflowEngine) ResumePausedExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func TestNewWorkflowResumerWorker(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockResumer := &services.WorkflowResumerImpl{}
	checkInterval := 30 * time.Second

	worker := NewWorkflowResumerWorker(mockResumer, log, checkInterval)

	assert.NotNil(t, worker)
	assert.Equal(t, mockResumer, worker.workflowResumer)
	assert.Equal(t, log, worker.logger)
	assert.Equal(t, checkInterval, worker.checkInterval)
	assert.Equal(t, 50, worker.batchSize)
	assert.NotNil(t, worker.stopCh)
	assert.NotNil(t, worker.doneCh)
}

func TestNewWorkflowResumerWorker_DefaultInterval(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockResumer := &services.WorkflowResumerImpl{}

	// Pass 0 for check interval to test default
	worker := NewWorkflowResumerWorker(mockResumer, log, 0)

	assert.NotNil(t, worker)
	assert.Equal(t, 1*time.Minute, worker.checkInterval) // Should default to 1 minute
}

func TestWorkflowResumerWorker_StartStop(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	// Setup mock to return no paused executions
	mockRepo.On("GetPausedExecutions", mock.Anything, 50).Return([]*models.WorkflowExecution{}, nil)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	checkInterval := 100 * time.Millisecond // Short interval for testing

	worker := NewWorkflowResumerWorker(resumer, log, checkInterval)

	ctx := context.Background()

	// Start the worker
	worker.Start(ctx)

	// Let it run for a bit
	time.Sleep(150 * time.Millisecond)

	// Stop the worker
	worker.Stop()

	// Verify it stopped gracefully
	select {
	case <-worker.doneCh:
		// Worker stopped successfully
	case <-time.After(1 * time.Second):
		t.Fatal("Worker did not stop within timeout")
	}

	mockRepo.AssertExpectations(t)
}

func TestWorkflowResumerWorker_ProcessPausedExecutions_NoExecutions(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()

	// Setup mock to return no paused executions
	mockRepo.On("GetPausedExecutions", ctx, 50).Return([]*models.WorkflowExecution{}, nil)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	worker := NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	// Process paused executions
	worker.processPausedExecutions(ctx)

	mockRepo.AssertExpectations(t)
}

func TestWorkflowResumerWorker_ProcessPausedExecutions_WithApproval(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()

	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"

	// Create paused execution with approval decision
	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-2 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
		ResumeData: models.JSONB{
			"approved": true,
		},
		ResumeCount: 0,
	}

	// Setup mock expectations
	mockRepo.On("GetPausedExecutions", ctx, 50).Return([]*models.WorkflowExecution{execution}, nil)
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)
	mockRepo.On("UpdateExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)
	mockEngine.On("ResumePausedExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	worker := NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	// Process paused executions
	worker.processPausedExecutions(ctx)

	mockRepo.AssertExpectations(t)
	mockEngine.AssertExpectations(t)
}

func TestWorkflowResumerWorker_ProcessPausedExecutions_WithRejection(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()

	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"

	// Create paused execution with rejection decision
	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-2 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
		ResumeData: models.JSONB{
			"approved": false,
		},
		ResumeCount: 0,
	}

	// Setup mock expectations
	mockRepo.On("GetPausedExecutions", ctx, 50).Return([]*models.WorkflowExecution{execution}, nil)
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)
	mockRepo.On("UpdateExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)
	mockEngine.On("ResumePausedExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	worker := NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	// Process paused executions
	worker.processPausedExecutions(ctx)

	mockRepo.AssertExpectations(t)
	mockEngine.AssertExpectations(t)
}

func TestWorkflowResumerWorker_ProcessPausedExecutions_NoApprovalDecision(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()

	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for manual intervention"

	// Create paused execution WITHOUT approval decision
	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-2 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
		ResumeData:   nil, // No resume data
		ResumeCount:  0,
	}

	// Setup mock expectations - should NOT call ResumeWorkflow
	mockRepo.On("GetPausedExecutions", ctx, 50).Return([]*models.WorkflowExecution{execution}, nil)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	worker := NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	// Process paused executions - should skip this execution
	worker.processPausedExecutions(ctx)

	mockRepo.AssertExpectations(t)
	// Engine should NOT be called since execution has no approval decision
	mockEngine.AssertNotCalled(t, "ResumePausedExecution")
}

func TestWorkflowResumerWorker_ProcessPausedExecutions_LongPause(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()

	// Paused for 25 hours (should log warning)
	pausedAt := time.Now().Add(-25 * time.Hour)
	pausedReason := "waiting for approval"

	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-26 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
		ResumeData:   nil, // No resume data
		ResumeCount:  0,
	}

	// Setup mock expectations
	mockRepo.On("GetPausedExecutions", ctx, 50).Return([]*models.WorkflowExecution{execution}, nil)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	worker := NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	// Process paused executions - should warn about long pause
	worker.processPausedExecutions(ctx)

	mockRepo.AssertExpectations(t)
}

func TestWorkflowResumerWorker_ProcessPausedExecutions_MultipleExecutions(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()

	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"

	// Create multiple paused executions
	executions := []*models.WorkflowExecution{
		{
			ID:           uuid.New(),
			WorkflowID:   uuid.New(),
			Status:       models.ExecutionStatusPaused,
			StartedAt:    time.Now().Add(-2 * time.Hour),
			PausedAt:     &pausedAt,
			PausedReason: &pausedReason,
			ResumeData:   models.JSONB{"approved": true},
			ResumeCount:  0,
		},
		{
			ID:           uuid.New(),
			WorkflowID:   uuid.New(),
			Status:       models.ExecutionStatusPaused,
			StartedAt:    time.Now().Add(-2 * time.Hour),
			PausedAt:     &pausedAt,
			PausedReason: &pausedReason,
			ResumeData:   models.JSONB{"approved": false},
			ResumeCount:  0,
		},
		{
			ID:           uuid.New(),
			WorkflowID:   uuid.New(),
			Status:       models.ExecutionStatusPaused,
			StartedAt:    time.Now().Add(-2 * time.Hour),
			PausedAt:     &pausedAt,
			PausedReason: &pausedReason,
			ResumeData:   nil, // No approval decision - should be skipped
			ResumeCount:  0,
		},
	}

	// Setup mock expectations
	mockRepo.On("GetPausedExecutions", ctx, 50).Return(executions, nil)

	// First two should be resumed
	for i := 0; i < 2; i++ {
		mockRepo.On("GetExecutionByID", ctx, executions[i].ID).Return(executions[i], nil)
		mockRepo.On("UpdateExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)
		mockEngine.On("ResumePausedExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)
	}

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	worker := NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	// Process paused executions
	worker.processPausedExecutions(ctx)

	mockRepo.AssertExpectations(t)
	mockEngine.AssertExpectations(t)
}

func TestWorkflowResumerWorker_ProcessPausedExecutions_ResumeError(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()

	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"

	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-2 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
		ResumeData:   models.JSONB{"approved": true},
		ResumeCount:  0,
	}

	// Setup mock expectations - resume will fail
	mockRepo.On("GetPausedExecutions", ctx, 50).Return([]*models.WorkflowExecution{execution}, nil)
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)
	mockRepo.On("UpdateExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(assert.AnError)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	worker := NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	// Process paused executions - should handle error gracefully
	worker.processPausedExecutions(ctx)

	mockRepo.AssertExpectations(t)
}

func TestWorkflowResumerWorker_ProcessPausedExecutions_GetError(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()

	// Setup mock expectations - get will fail
	mockRepo.On("GetPausedExecutions", ctx, 50).Return(nil, assert.AnError)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	worker := NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	// Process paused executions - should handle error gracefully
	worker.processPausedExecutions(ctx)

	mockRepo.AssertExpectations(t)
}

func TestWorkflowResumerWorker_ContextCancellation(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	// Setup mock to return no paused executions
	mockRepo.On("GetPausedExecutions", mock.Anything, 50).Return([]*models.WorkflowExecution{}, nil)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	checkInterval := 100 * time.Millisecond

	worker := NewWorkflowResumerWorker(resumer, log, checkInterval)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start the worker
	worker.Start(ctx)

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Worker should stop
	select {
	case <-worker.doneCh:
		// Worker stopped successfully
	case <-time.After(1 * time.Second):
		t.Fatal("Worker did not stop after context cancellation")
	}
}

func TestWorkflowResumerWorker_InvalidApprovalType(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	// Create mock execution repository and engine
	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()

	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"

	// Create paused execution with invalid approval type (string instead of bool)
	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-2 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
		ResumeData: models.JSONB{
			"approved": "yes", // Invalid type
		},
		ResumeCount: 0,
	}

	// Setup mock expectations - should NOT call ResumeWorkflow
	mockRepo.On("GetPausedExecutions", ctx, 50).Return([]*models.WorkflowExecution{execution}, nil)

	resumer := services.NewWorkflowResumer(log, mockRepo, mockEngine)
	worker := NewWorkflowResumerWorker(resumer, log, 1*time.Minute)

	// Process paused executions - should skip this execution due to invalid type
	worker.processPausedExecutions(ctx)

	mockRepo.AssertExpectations(t)
	// Engine should NOT be called since approval type is invalid
	mockEngine.AssertNotCalled(t, "ResumePausedExecution")
}

func TestStringOrEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: "none",
		},
		{
			name: "non-nil pointer",
			input: func() *string {
				s := "test value"
				return &s
			}(),
			expected: "test value",
		},
		{
			name: "empty string",
			input: func() *string {
				s := ""
				return &s
			}(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringOrEmpty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
