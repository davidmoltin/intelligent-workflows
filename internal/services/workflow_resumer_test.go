package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
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

func TestNewWorkflowResumer(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	assert.NotNil(t, resumer)
	assert.Equal(t, log, resumer.logger)
	assert.Equal(t, mockRepo, resumer.executionRepo)
	assert.Equal(t, mockEngine, resumer.engine)
}

func TestPauseExecution_Success(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()
	stepID := uuid.New()
	reason := "waiting for approval"

	// Create a running execution
	execution := &models.WorkflowExecution{
		ID:         executionID,
		WorkflowID: uuid.New(),
		Status:     models.ExecutionStatusRunning,
		StartedAt:  time.Now(),
	}

	// Setup expectations
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)
	mockRepo.On("UpdateExecution", ctx, mock.MatchedBy(func(exec *models.WorkflowExecution) bool {
		return exec.ID == executionID &&
			exec.Status == models.ExecutionStatusPaused &&
			exec.PausedAt != nil &&
			exec.PausedReason != nil &&
			*exec.PausedReason == reason &&
			exec.PausedStepID != nil &&
			*exec.PausedStepID == stepID
	})).Return(nil)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test pause execution
	err = resumer.PauseExecution(ctx, executionID, reason, &stepID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestPauseExecution_ExecutionNotFound(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()
	reason := "waiting for approval"

	// Setup expectations
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(nil, fmt.Errorf("execution not found"))

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test pause execution
	err = resumer.PauseExecution(ctx, executionID, reason, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get execution")
	mockRepo.AssertExpectations(t)
}

func TestPauseExecution_NotRunning(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()
	reason := "waiting for approval"

	// Create a completed execution
	execution := &models.WorkflowExecution{
		ID:         executionID,
		WorkflowID: uuid.New(),
		Status:     models.ExecutionStatusCompleted,
		StartedAt:  time.Now(),
	}

	// Setup expectations
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test pause execution
	err = resumer.PauseExecution(ctx, executionID, reason, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not running")
	mockRepo.AssertExpectations(t)
}

func TestPauseExecution_NilRepository(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	ctx := context.Background()
	executionID := uuid.New()
	reason := "waiting for approval"

	resumer := NewWorkflowResumer(log, nil, nil)

	// Test pause execution with nil repository
	err = resumer.PauseExecution(ctx, executionID, reason, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution repository not configured")
}

func TestResumeWorkflow_Success(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()
	approved := true

	// Create a paused execution
	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"
	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-2 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
		ResumeData:   make(models.JSONB),
		ResumeCount:  0,
	}

	// Setup expectations
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)
	mockRepo.On("UpdateExecution", ctx, mock.MatchedBy(func(exec *models.WorkflowExecution) bool {
		return exec.ID == executionID &&
			exec.Status == models.ExecutionStatusRunning &&
			exec.PausedAt == nil &&
			exec.PausedReason == nil &&
			exec.LastResumedAt != nil &&
			exec.ResumeCount == 1 &&
			exec.ResumeData["approved"] == approved
	})).Return(nil)
	mockEngine.On("ResumePausedExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test resume workflow
	err = resumer.ResumeWorkflow(ctx, executionID, approved)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockEngine.AssertExpectations(t)
}

func TestResumeWorkflow_NotPaused(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()
	approved := true

	// Create a running execution (not paused)
	execution := &models.WorkflowExecution{
		ID:         executionID,
		WorkflowID: uuid.New(),
		Status:     models.ExecutionStatusRunning,
		StartedAt:  time.Now(),
	}

	// Setup expectations
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test resume workflow
	err = resumer.ResumeWorkflow(ctx, executionID, approved)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not paused")
	mockRepo.AssertExpectations(t)
}

func TestResumeWorkflow_PausedTooLong(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()
	approved := true

	// Create a paused execution that's been paused for 8 days
	pausedAt := time.Now().Add(-8 * 24 * time.Hour)
	pausedReason := "waiting for approval"
	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-9 * 24 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
	}

	// Setup expectations
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test resume workflow
	err = resumer.ResumeWorkflow(ctx, executionID, approved)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has been paused for too long")
	mockRepo.AssertExpectations(t)
}

func TestResumeExecution_WithResumeData(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()

	// Create custom resume data
	resumeData := models.JSONB{
		"approved":   true,
		"approver":   "john.doe@example.com",
		"notes":      "looks good",
		"extra_data": map[string]interface{}{"key": "value"},
	}

	// Create a paused execution
	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"
	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-2 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
		ResumeData:   make(models.JSONB),
		ResumeCount:  0,
	}

	// Setup expectations
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)
	mockRepo.On("UpdateExecution", ctx, mock.MatchedBy(func(exec *models.WorkflowExecution) bool {
		// Verify all resume data is merged
		return exec.ID == executionID &&
			exec.Status == models.ExecutionStatusRunning &&
			exec.ResumeData["approved"] == true &&
			exec.ResumeData["approver"] == "john.doe@example.com" &&
			exec.ResumeData["notes"] == "looks good"
	})).Return(nil)
	mockEngine.On("ResumePausedExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test resume execution with custom data
	err = resumer.ResumeExecution(ctx, executionID, resumeData)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockEngine.AssertExpectations(t)
}

func TestResumeExecution_WithoutEngine(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)

	ctx := context.Background()
	executionID := uuid.New()
	resumeData := models.JSONB{"approved": true}

	// Create a paused execution
	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"
	execution := &models.WorkflowExecution{
		ID:           executionID,
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-2 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
		ResumeData:   make(models.JSONB),
		ResumeCount:  0,
	}

	// Setup expectations
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)
	mockRepo.On("UpdateExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	// Create resumer without engine
	resumer := NewWorkflowResumer(log, mockRepo, nil)

	// Test resume execution without engine
	err = resumer.ResumeExecution(ctx, executionID, resumeData)

	// Should update status but log warning about no engine
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetPausedExecutions_Success(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	limit := 50

	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"

	// Create some paused executions
	executions := []*models.WorkflowExecution{
		{
			ID:           uuid.New(),
			WorkflowID:   uuid.New(),
			Status:       models.ExecutionStatusPaused,
			StartedAt:    time.Now().Add(-2 * time.Hour),
			PausedAt:     &pausedAt,
			PausedReason: &pausedReason,
		},
		{
			ID:           uuid.New(),
			WorkflowID:   uuid.New(),
			Status:       models.ExecutionStatusPaused,
			StartedAt:    time.Now().Add(-3 * time.Hour),
			PausedAt:     &pausedAt,
			PausedReason: &pausedReason,
		},
	}

	// Setup expectations
	mockRepo.On("GetPausedExecutions", ctx, limit).Return(executions, nil)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test get paused executions
	result, err := resumer.GetPausedExecutions(ctx, limit)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, executions[0].ID, result[0].ID)
	assert.Equal(t, executions[1].ID, result[1].ID)
	mockRepo.AssertExpectations(t)
}

func TestGetPausedExecutions_NilRepository(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	ctx := context.Background()
	limit := 50

	resumer := NewWorkflowResumer(log, nil, nil)

	// Test get paused executions with nil repository
	result, err := resumer.GetPausedExecutions(ctx, limit)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "execution repository not configured")
}

func TestCanResume_Success(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	pausedAt := time.Now().Add(-1 * time.Hour)
	pausedReason := "waiting for approval"

	execution := &models.WorkflowExecution{
		ID:           uuid.New(),
		WorkflowID:   uuid.New(),
		Status:       models.ExecutionStatusPaused,
		StartedAt:    time.Now().Add(-2 * time.Hour),
		PausedAt:     &pausedAt,
		PausedReason: &pausedReason,
	}

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test can resume
	err = resumer.CanResume(execution)

	assert.NoError(t, err)
}

func TestCanResume_NilExecution(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test can resume with nil execution
	err = resumer.CanResume(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution is nil")
}

func TestCanResume_NotPaused(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	execution := &models.WorkflowExecution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Status:     models.ExecutionStatusRunning,
		StartedAt:  time.Now(),
	}

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test can resume
	err = resumer.CanResume(execution)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not paused")
}

func TestCanResume_NoPauseTimestamp(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	execution := &models.WorkflowExecution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Status:     models.ExecutionStatusPaused,
		StartedAt:  time.Now(),
		PausedAt:   nil, // Missing pause timestamp
	}

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test can resume
	err = resumer.CanResume(execution)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no pause timestamp")
}

func TestCanResume_PausedTooLong(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	// Paused for 8 days (exceeds 7 day limit)
	pausedAt := time.Now().Add(-8 * 24 * time.Hour)
	execution := &models.WorkflowExecution{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Status:     models.ExecutionStatusPaused,
		StartedAt:  time.Now().Add(-9 * 24 * time.Hour),
		PausedAt:   &pausedAt,
	}

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test can resume
	err = resumer.CanResume(execution)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has been paused for too long")
}

func TestResumeWorkflow_NilRepository(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	ctx := context.Background()
	executionID := uuid.New()

	resumer := NewWorkflowResumer(log, nil, nil)

	// Test resume workflow with nil repository - should be a no-op
	err = resumer.ResumeWorkflow(ctx, executionID, true)

	assert.NoError(t, err) // Should succeed as a no-op
}

func TestResumeExecution_MultipleResumes(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	mockRepo := new(MockExecutionRepository)
	mockEngine := new(MockWorkflowEngine)

	ctx := context.Background()
	executionID := uuid.New()

	// Create a paused execution that has been resumed before
	pausedAt := time.Now().Add(-1 * time.Hour)
	lastResumedAt := time.Now().Add(-2 * time.Hour)
	pausedReason := "waiting for second approval"
	execution := &models.WorkflowExecution{
		ID:            executionID,
		WorkflowID:    uuid.New(),
		Status:        models.ExecutionStatusPaused,
		StartedAt:     time.Now().Add(-3 * time.Hour),
		PausedAt:      &pausedAt,
		PausedReason:  &pausedReason,
		ResumeData:    make(models.JSONB),
		ResumeCount:   2, // Already resumed twice
		LastResumedAt: &lastResumedAt,
	}

	// Setup expectations
	mockRepo.On("GetExecutionByID", ctx, executionID).Return(execution, nil)
	mockRepo.On("UpdateExecution", ctx, mock.MatchedBy(func(exec *models.WorkflowExecution) bool {
		return exec.ID == executionID &&
			exec.Status == models.ExecutionStatusRunning &&
			exec.ResumeCount == 3 // Should increment to 3
	})).Return(nil)
	mockEngine.On("ResumePausedExecution", ctx, mock.AnythingOfType("*models.WorkflowExecution")).Return(nil)

	resumer := NewWorkflowResumer(log, mockRepo, mockEngine)

	// Test resume execution
	resumeData := models.JSONB{"approved": true}
	err = resumer.ResumeExecution(ctx, executionID, resumeData)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockEngine.AssertExpectations(t)
}
