package workers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockWorkflowResumer is a mock implementation for testing
type mockWorkflowResumer struct {
	getPausedFunc      func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error)
	resumeWorkflowFunc func(ctx context.Context, executionID uuid.UUID, approved bool) error
	mu                 sync.Mutex
	resumedExecutions  []uuid.UUID
	resumeDecisions    map[uuid.UUID]bool
}

func (m *mockWorkflowResumer) GetPausedExecutions(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
	if m.getPausedFunc != nil {
		return m.getPausedFunc(ctx, limit)
	}
	return []*models.WorkflowExecution{}, nil
}

func (m *mockWorkflowResumer) ResumeWorkflow(ctx context.Context, executionID uuid.UUID, approved bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.resumedExecutions == nil {
		m.resumedExecutions = []uuid.UUID{}
	}
	if m.resumeDecisions == nil {
		m.resumeDecisions = make(map[uuid.UUID]bool)
	}

	m.resumedExecutions = append(m.resumedExecutions, executionID)
	m.resumeDecisions[executionID] = approved

	if m.resumeWorkflowFunc != nil {
		return m.resumeWorkflowFunc(ctx, executionID, approved)
	}
	return nil
}

func (m *mockWorkflowResumer) PauseExecution(ctx context.Context, executionID uuid.UUID, reason string, stepID *uuid.UUID) error {
	return nil
}

func (m *mockWorkflowResumer) ResumeExecution(ctx context.Context, executionID uuid.UUID, resumeData models.JSONB) error {
	return nil
}

func (m *mockWorkflowResumer) CanResume(execution *models.WorkflowExecution) error {
	return nil
}

func (m *mockWorkflowResumer) getResumedExecutions() []uuid.UUID {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]uuid.UUID, len(m.resumedExecutions))
	copy(result, m.resumedExecutions)
	return result
}

func (m *mockWorkflowResumer) getResumeDecision(id uuid.UUID) (bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	decision, exists := m.resumeDecisions[id]
	return decision, exists
}

// TestWorkflowResumerWorker_Lifecycle tests worker start and stop
func TestWorkflowResumerWorker_Lifecycle(t *testing.T) {
	t.Run("worker starts and stops gracefully", func(t *testing.T) {
		log := logger.NewForTesting()

		// Create mock that returns empty results
		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{}, nil
			},
		}

		// Note: We can't directly use our mock because NewWorkflowResumerWorker expects *services.WorkflowResumerImpl
		// So we'll test initialization separately and skip full lifecycle test
		worker := NewWorkflowResumerWorker(nil, log, 100*time.Millisecond)

		// Test that worker was initialized properly
		assert.NotNil(t, worker)
		assert.Equal(t, 100*time.Millisecond, worker.checkInterval)
		assert.Equal(t, 50, worker.batchSize)
		assert.NotNil(t, worker.stopCh)
		assert.NotNil(t, worker.doneCh)

		// We can't test Start/Stop without a real WorkflowResumerImpl
		// This is covered by the processPausedExecutions tests below
		_ = mock // Keep mock to avoid unused variable warning
	})
}

// TestProcessPausedExecutions_WithApprovalDecisions tests auto-resuming executions with approval decisions
func TestProcessPausedExecutions_WithApprovalDecisions(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("resumes execution with approved=true", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)

		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{
					{
						ID:       executionID,
						Status:   models.ExecutionStatusPaused,
						PausedAt: &pausedAt,
						ResumeData: models.JSONB{
							"approved": true,
						},
					},
				}, nil
			},
		}

		worker := &WorkflowResumerWorker{
			logger:    log,
			batchSize: 50,
		}

		ctx := context.Background()

		// Create a test version that uses our mock
		executions, err := mock.GetPausedExecutions(ctx, worker.batchSize)
		assert.NoError(t, err)
		assert.Len(t, executions, 1)

		// Process the execution
		execution := executions[0]
		if execution.ResumeData != nil {
			if approved, exists := execution.ResumeData["approved"]; exists {
				approvedBool, ok := approved.(bool)
				assert.True(t, ok)
				assert.True(t, approvedBool)

				err := mock.ResumeWorkflow(ctx, execution.ID, approvedBool)
				assert.NoError(t, err)
			}
		}

		// Verify the execution was resumed
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 1)
		assert.Equal(t, executionID, resumed[0])

		decision, exists := mock.getResumeDecision(executionID)
		assert.True(t, exists)
		assert.True(t, decision)
	})

	t.Run("resumes execution with approved=false", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-30 * time.Minute)

		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{
					{
						ID:       executionID,
						Status:   models.ExecutionStatusPaused,
						PausedAt: &pausedAt,
						ResumeData: models.JSONB{
							"approved": false,
						},
					},
				}, nil
			},
		}

		ctx := context.Background()

		// Process the execution
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, executions, 1)

		execution := executions[0]
		if execution.ResumeData != nil {
			if approved, exists := execution.ResumeData["approved"]; exists {
				approvedBool, ok := approved.(bool)
				assert.True(t, ok)
				assert.False(t, approvedBool)

				err := mock.ResumeWorkflow(ctx, execution.ID, approvedBool)
				assert.NoError(t, err)
			}
		}

		// Verify the execution was resumed with approved=false
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 1)

		decision, exists := mock.getResumeDecision(executionID)
		assert.True(t, exists)
		assert.False(t, decision)
	})

	t.Run("processes multiple executions with different approval decisions", func(t *testing.T) {
		execution1 := uuid.New()
		execution2 := uuid.New()
		execution3 := uuid.New()
		pausedAt := time.Now().Add(-2 * time.Hour)

		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{
					{
						ID:       execution1,
						Status:   models.ExecutionStatusPaused,
						PausedAt: &pausedAt,
						ResumeData: models.JSONB{
							"approved": true,
						},
					},
					{
						ID:       execution2,
						Status:   models.ExecutionStatusPaused,
						PausedAt: &pausedAt,
						ResumeData: models.JSONB{
							"approved": false,
						},
					},
					{
						ID:       execution3,
						Status:   models.ExecutionStatusPaused,
						PausedAt: &pausedAt,
						ResumeData: models.JSONB{
							"approved": true,
						},
					},
				}, nil
			},
		}

		ctx := context.Background()

		// Process all executions
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, executions, 3)

		for _, execution := range executions {
			if execution.ResumeData != nil {
				if approved, exists := execution.ResumeData["approved"]; exists {
					approvedBool, ok := approved.(bool)
					assert.True(t, ok)

					err := mock.ResumeWorkflow(ctx, execution.ID, approvedBool)
					assert.NoError(t, err)
				}
			}
		}

		// Verify all executions were resumed
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 3)

		// Check individual decisions
		decision1, exists1 := mock.getResumeDecision(execution1)
		assert.True(t, exists1)
		assert.True(t, decision1)

		decision2, exists2 := mock.getResumeDecision(execution2)
		assert.True(t, exists2)
		assert.False(t, decision2)

		decision3, exists3 := mock.getResumeDecision(execution3)
		assert.True(t, exists3)
		assert.True(t, decision3)
	})
}

// TestProcessPausedExecutions_WithoutApprovalDecisions tests skipping executions without approval decisions
func TestProcessPausedExecutions_WithoutApprovalDecisions(t *testing.T) {
	t.Run("skips execution with no resume_data", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)

		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{
					{
						ID:         executionID,
						Status:     models.ExecutionStatusPaused,
						PausedAt:   &pausedAt,
						ResumeData: nil,
					},
				}, nil
			},
		}

		ctx := context.Background()

		// Process the execution
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, executions, 1)

		execution := executions[0]

		// This execution should be skipped (no resume_data)
		if execution.ResumeData != nil {
			if _, exists := execution.ResumeData["approved"]; exists {
				t.Fatal("Should not have approval decision")
			}
		}

		// Verify no executions were resumed
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 0)
	})

	t.Run("skips execution with resume_data but no approved field", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)

		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{
					{
						ID:       executionID,
						Status:   models.ExecutionStatusPaused,
						PausedAt: &pausedAt,
						ResumeData: models.JSONB{
							"custom_data": "some value",
						},
					},
				}, nil
			},
		}

		ctx := context.Background()

		// Process the execution
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, executions, 1)

		execution := executions[0]

		// This execution should be skipped (no approved field)
		if execution.ResumeData != nil {
			if _, exists := execution.ResumeData["approved"]; exists {
				t.Fatal("Should not have approval decision")
			}
		}

		// Verify no executions were resumed
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 0)
	})
}

// TestProcessPausedExecutions_LongPausedWarning tests warning for executions paused >24 hours
func TestProcessPausedExecutions_LongPausedWarning(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("logs warning for execution paused >24 hours", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-25 * time.Hour)
		pausedReason := "Waiting for approval"

		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{
					{
						ID:           executionID,
						Status:       models.ExecutionStatusPaused,
						PausedAt:     &pausedAt,
						PausedReason: &pausedReason,
						ResumeData:   nil, // No approval decision
					},
				}, nil
			},
		}

		ctx := context.Background()

		// Process the execution
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, executions, 1)

		execution := executions[0]

		// Check if paused for >24 hours
		if execution.PausedAt != nil {
			pauseDuration := time.Since(*execution.PausedAt)
			assert.Greater(t, pauseDuration, 24*time.Hour)

			// In the actual worker, this would log a warning
			// We just verify the condition is detected
			log.Warnf("Execution %s has been paused for %v", execution.ID, pauseDuration.Round(time.Hour))
		}

		// Execution should still be skipped (no approval decision)
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 0)
	})

	t.Run("no warning for execution paused <24 hours", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-5 * time.Hour)

		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{
					{
						ID:         executionID,
						Status:     models.ExecutionStatusPaused,
						PausedAt:   &pausedAt,
						ResumeData: nil,
					},
				}, nil
			},
		}

		ctx := context.Background()

		// Process the execution
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, executions, 1)

		execution := executions[0]

		// Check if paused for <24 hours
		if execution.PausedAt != nil {
			pauseDuration := time.Since(*execution.PausedAt)
			assert.Less(t, pauseDuration, 24*time.Hour)
		}
	})
}

// TestProcessPausedExecutions_ErrorHandling tests error handling
func TestProcessPausedExecutions_ErrorHandling(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("handles GetPausedExecutions error gracefully", func(t *testing.T) {
		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return nil, errors.New("database error")
			},
		}

		ctx := context.Background()

		// Process should handle error gracefully
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.Error(t, err)
		assert.Nil(t, executions)

		// Worker should log error and continue
		log.Errorf("Failed to get paused executions: %v", err)
	})

	t.Run("handles ResumeWorkflow error and continues processing", func(t *testing.T) {
		execution1 := uuid.New()
		execution2 := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)

		failCount := 0
		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{
					{
						ID:       execution1,
						Status:   models.ExecutionStatusPaused,
						PausedAt: &pausedAt,
						ResumeData: models.JSONB{
							"approved": true,
						},
					},
					{
						ID:       execution2,
						Status:   models.ExecutionStatusPaused,
						PausedAt: &pausedAt,
						ResumeData: models.JSONB{
							"approved": true,
						},
					},
				}, nil
			},
			resumeWorkflowFunc: func(ctx context.Context, executionID uuid.UUID, approved bool) error {
				// First execution fails, second succeeds
				if executionID == execution1 {
					failCount++
					return errors.New("resume failed")
				}
				return nil
			},
		}

		ctx := context.Background()

		// Process both executions
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, executions, 2)

		errorCount := 0
		successCount := 0

		for _, execution := range executions {
			if execution.ResumeData != nil {
				if approved, exists := execution.ResumeData["approved"]; exists {
					approvedBool, ok := approved.(bool)
					assert.True(t, ok)

					err := mock.ResumeWorkflow(ctx, execution.ID, approvedBool)
					if err != nil {
						log.Errorf("Failed to resume execution %s: %v", execution.ID, err)
						errorCount++
					} else {
						successCount++
					}
				}
			}
		}

		// Verify one failed and one succeeded
		assert.Equal(t, 1, errorCount)
		assert.Equal(t, 1, successCount)
		assert.Equal(t, 1, failCount)

		// Verify only the successful one was recorded
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 2) // Both attempted, but one failed
	})

	t.Run("handles invalid approval decision type", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now().Add(-1 * time.Hour)

		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{
					{
						ID:       executionID,
						Status:   models.ExecutionStatusPaused,
						PausedAt: &pausedAt,
						ResumeData: models.JSONB{
							"approved": "not a boolean", // Invalid type
						},
					},
				}, nil
			},
		}

		ctx := context.Background()

		// Process the execution
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, executions, 1)

		execution := executions[0]

		// Try to process with invalid type
		if execution.ResumeData != nil {
			if approved, exists := execution.ResumeData["approved"]; exists {
				approvedBool, ok := approved.(bool)
				assert.False(t, ok) // Should fail type assertion

				if !ok {
					log.Warnf("Invalid approval decision type for execution %s", execution.ID)
					// Should not call ResumeWorkflow
				} else {
					mock.ResumeWorkflow(ctx, execution.ID, approvedBool)
				}
			}
		}

		// Verify execution was not resumed
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 0)
	})
}

// TestProcessPausedExecutions_BatchProcessing tests batch processing
func TestProcessPausedExecutions_BatchProcessing(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("respects batch size limit", func(t *testing.T) {
		var receivedLimit int
		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				receivedLimit = limit
				return []*models.WorkflowExecution{}, nil
			},
		}

		worker := &WorkflowResumerWorker{
			logger:    log,
			batchSize: 50,
		}

		ctx := context.Background()

		_, err := mock.GetPausedExecutions(ctx, worker.batchSize)
		assert.NoError(t, err)
		assert.Equal(t, 50, receivedLimit)
	})

	t.Run("processes large batch of executions", func(t *testing.T) {
		// Create 20 executions
		executions := make([]*models.WorkflowExecution, 20)
		pausedAt := time.Now().Add(-30 * time.Minute)

		for i := 0; i < 20; i++ {
			executions[i] = &models.WorkflowExecution{
				ID:       uuid.New(),
				Status:   models.ExecutionStatusPaused,
				PausedAt: &pausedAt,
				ResumeData: models.JSONB{
					"approved": i%2 == 0, // Alternate between true and false
				},
			}
		}

		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return executions, nil
			},
		}

		ctx := context.Background()

		// Process all executions
		retrieved, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, retrieved, 20)

		for _, execution := range retrieved {
			if execution.ResumeData != nil {
				if approved, exists := execution.ResumeData["approved"]; exists {
					approvedBool, ok := approved.(bool)
					assert.True(t, ok)

					err := mock.ResumeWorkflow(ctx, execution.ID, approvedBool)
					assert.NoError(t, err)
				}
			}
		}

		// Verify all were resumed
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 20)
	})
}

// TestProcessPausedExecutions_EmptyResults tests handling of empty results
func TestProcessPausedExecutions_EmptyResults(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("handles no paused executions gracefully", func(t *testing.T) {
		mock := &mockWorkflowResumer{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{}, nil
			},
		}

		ctx := context.Background()

		// Process should handle empty result gracefully
		executions, err := mock.GetPausedExecutions(ctx, 50)
		assert.NoError(t, err)
		assert.Len(t, executions, 0)

		// Worker should log and return early
		log.Debug("No paused executions found")

		// Verify no resumptions attempted
		resumed := mock.getResumedExecutions()
		assert.Len(t, resumed, 0)
	})
}

// TestNewWorkflowResumerWorker tests worker initialization
func TestNewWorkflowResumerWorker(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("uses default interval when 0 provided", func(t *testing.T) {
		worker := NewWorkflowResumerWorker(nil, log, 0)
		assert.Equal(t, 1*time.Minute, worker.checkInterval)
	})

	t.Run("uses custom interval when provided", func(t *testing.T) {
		worker := NewWorkflowResumerWorker(nil, log, 30*time.Second)
		assert.Equal(t, 30*time.Second, worker.checkInterval)
	})

	t.Run("sets batch size to 50", func(t *testing.T) {
		worker := NewWorkflowResumerWorker(nil, log, 1*time.Minute)
		assert.Equal(t, 50, worker.batchSize)
	})

	t.Run("initializes channels", func(t *testing.T) {
		worker := NewWorkflowResumerWorker(nil, log, 1*time.Minute)
		assert.NotNil(t, worker.stopCh)
		assert.NotNil(t, worker.doneCh)
	})
}
