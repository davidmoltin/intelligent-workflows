package services

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
)

// ExecutionRepository defines the interface for execution persistence
type ExecutionRepository interface {
	CreateExecution(ctx context.Context, execution *models.WorkflowExecution) error
	UpdateExecution(ctx context.Context, execution *models.WorkflowExecution) error
	GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error)
	GetPausedExecutions(ctx context.Context, limit int) ([]*models.WorkflowExecution, error)
}

// WorkflowEngine defines the interface for workflow execution
type WorkflowEngine interface {
	ResumePausedExecution(ctx context.Context, execution *models.WorkflowExecution) error
}

// WorkflowResumerImpl implements WorkflowResumer interface
type WorkflowResumerImpl struct {
	logger        *logger.Logger
	executionRepo ExecutionRepository
	engine        WorkflowEngine
}

// NewWorkflowResumer creates a new workflow resumer
func NewWorkflowResumer(log *logger.Logger, executionRepo ExecutionRepository, engine WorkflowEngine) *WorkflowResumerImpl {
	return &WorkflowResumerImpl{
		logger:        log,
		executionRepo: executionRepo,
		engine:        engine,
	}
}

// PauseExecution pauses a running workflow execution
func (w *WorkflowResumerImpl) PauseExecution(ctx context.Context, executionID uuid.UUID, reason string, stepID *uuid.UUID) error {
	w.logger.Infof("Pausing workflow execution %s: %s", executionID, reason)

	if w.executionRepo == nil {
		return fmt.Errorf("execution repository not configured")
	}

	// Get current execution
	execution, err := w.executionRepo.GetExecutionByID(ctx, executionID)
	if err != nil {
		w.logger.Errorf("Failed to get execution %s: %v", executionID, err)
		return fmt.Errorf("failed to get execution: %w", err)
	}

	// Validate execution can be paused
	if execution.Status != models.ExecutionStatusRunning {
		return fmt.Errorf("execution %s is not running (status: %s)", executionID, execution.Status)
	}

	// Update execution to paused state
	now := time.Now()
	execution.Status = models.ExecutionStatusPaused
	execution.PausedAt = &now
	execution.PausedReason = &reason
	if stepID != nil {
		execution.PausedStepID = stepID
	}

	// Save updated execution
	if err := w.executionRepo.UpdateExecution(ctx, execution); err != nil {
		w.logger.Errorf("Failed to pause execution %s: %v", executionID, err)
		return fmt.Errorf("failed to update execution: %w", err)
	}

	w.logger.Infof("Successfully paused execution %s", executionID)
	return nil
}

// ResumeWorkflow resumes a paused workflow execution
// This method maintains backward compatibility with the WorkflowResumer interface
func (w *WorkflowResumerImpl) ResumeWorkflow(ctx context.Context, executionID uuid.UUID, approved bool) error {
	w.logger.Infof("Resuming workflow execution %s with approval status: %v", executionID, approved)

	if w.executionRepo == nil {
		w.logger.Warn("Execution repository not configured, workflow resume is a no-op")
		return nil
	}

	// Get paused execution
	execution, err := w.executionRepo.GetExecutionByID(ctx, executionID)
	if err != nil {
		w.logger.Errorf("Failed to get execution %s: %v", executionID, err)
		return fmt.Errorf("failed to get execution: %w", err)
	}

	// Validate execution can be resumed
	if err := w.CanResume(execution); err != nil {
		return err
	}

	// Add approval decision to resume data
	if execution.ResumeData == nil {
		execution.ResumeData = make(models.JSONB)
	}
	execution.ResumeData["approved"] = approved
	execution.ResumeData["resumed_at"] = time.Now()

	// Resume the execution
	return w.resumeExecution(ctx, execution)
}

// ResumeExecution resumes a paused workflow execution with custom resume data
func (w *WorkflowResumerImpl) ResumeExecution(ctx context.Context, executionID uuid.UUID, resumeData models.JSONB) error {
	w.logger.Infof("Resuming workflow execution %s", executionID)

	if w.executionRepo == nil {
		return fmt.Errorf("execution repository not configured")
	}

	// Get paused execution
	execution, err := w.executionRepo.GetExecutionByID(ctx, executionID)
	if err != nil {
		w.logger.Errorf("Failed to get execution %s: %v", executionID, err)
		return fmt.Errorf("failed to get execution: %w", err)
	}

	// Validate execution can be resumed
	if err := w.CanResume(execution); err != nil {
		return err
	}

	// Merge resume data
	if execution.ResumeData == nil {
		execution.ResumeData = make(models.JSONB)
	}
	for key, value := range resumeData {
		execution.ResumeData[key] = value
	}

	// Resume the execution
	return w.resumeExecution(ctx, execution)
}

// resumeExecution is the internal method that performs the actual resumption
func (w *WorkflowResumerImpl) resumeExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	// Update execution state
	now := time.Now()
	execution.Status = models.ExecutionStatusRunning
	execution.LastResumedAt = &now
	execution.ResumeCount++

	// Clear pause information
	execution.PausedAt = nil
	execution.PausedReason = nil

	// Save updated execution
	if err := w.executionRepo.UpdateExecution(ctx, execution); err != nil {
		w.logger.Errorf("Failed to update execution %s: %v", execution.ID, err)
		return fmt.Errorf("failed to update execution: %w", err)
	}

	// Trigger workflow engine to continue execution
	if w.engine != nil {
		if err := w.engine.ResumePausedExecution(ctx, execution); err != nil {
			w.logger.Errorf("Failed to resume execution in engine %s: %v", execution.ID, err)
			return fmt.Errorf("failed to resume execution in engine: %w", err)
		}
	} else {
		w.logger.Warn("Workflow engine not configured, execution state updated but not resumed")
	}

	w.logger.Infof("Successfully resumed execution %s (resume count: %d)", execution.ID, execution.ResumeCount)
	return nil
}

// GetPausedExecutions retrieves paused executions pending resume
func (w *WorkflowResumerImpl) GetPausedExecutions(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
	if w.executionRepo == nil {
		return nil, fmt.Errorf("execution repository not configured")
	}

	executions, err := w.executionRepo.GetPausedExecutions(ctx, limit)
	if err != nil {
		w.logger.Errorf("Failed to get paused executions: %v", err)
		return nil, fmt.Errorf("failed to get paused executions: %w", err)
	}

	w.logger.Infof("Retrieved %d paused executions", len(executions))
	return executions, nil
}

// CanResume checks if an execution can be resumed
func (w *WorkflowResumerImpl) CanResume(execution *models.WorkflowExecution) error {
	if execution == nil {
		return fmt.Errorf("execution is nil")
	}

	if execution.Status != models.ExecutionStatusPaused {
		return fmt.Errorf("execution %s is not paused (status: %s)", execution.ID, execution.Status)
	}

	if execution.PausedAt == nil {
		return fmt.Errorf("execution %s has no pause timestamp", execution.ID)
	}

	// Check if execution has been paused for too long (e.g., 7 days)
	maxPauseDuration := 7 * 24 * time.Hour
	if time.Since(*execution.PausedAt) > maxPauseDuration {
		return fmt.Errorf("execution %s has been paused for too long (paused at: %v)", execution.ID, execution.PausedAt)
	}

	return nil
}
