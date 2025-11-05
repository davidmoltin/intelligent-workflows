package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// WorkflowExecutor interface for resuming executions
type WorkflowExecutor interface {
	ResumeExecution(
		ctx context.Context,
		executionID uuid.UUID,
		workflow *models.Workflow,
		resumeEvent string,
		resumeData map[string]interface{},
	) (*models.WorkflowExecution, error)
}

// ExecutionRepository interface for loading executions
type ExecutionRepository interface {
	GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error)
	UpdateExecution(ctx context.Context, execution *models.WorkflowExecution) error
}

// WorkflowRepository interface for loading workflows
type WorkflowRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error)
}

// WorkflowResumerImpl implements WorkflowResumer interface
type WorkflowResumerImpl struct {
	executor       WorkflowExecutor
	executionRepo  ExecutionRepository
	workflowRepo   WorkflowRepository
	logger         *logger.Logger
}

// NewWorkflowResumer creates a new workflow resumer
func NewWorkflowResumer(
	executor WorkflowExecutor,
	executionRepo ExecutionRepository,
	workflowRepo WorkflowRepository,
	log *logger.Logger,
) *WorkflowResumerImpl {
	return &WorkflowResumerImpl{
		executor:      executor,
		executionRepo: executionRepo,
		workflowRepo:  workflowRepo,
		logger:        log,
	}
}

// ResumeWorkflow resumes a paused workflow execution
func (w *WorkflowResumerImpl) ResumeWorkflow(ctx context.Context, executionID uuid.UUID, approved bool) error {
	w.logger.Infof("Resuming workflow execution %s with approval status: %v", executionID, approved)

	// Load the execution
	execution, err := w.executionRepo.GetExecutionByID(ctx, executionID)
	if err != nil {
		return fmt.Errorf("failed to load execution: %w", err)
	}

	// Verify execution is in waiting state
	if execution.Status != models.ExecutionStatusWaiting {
		return fmt.Errorf("execution is not in waiting state, current status: %s", execution.Status)
	}

	// Load the workflow
	workflow, err := w.workflowRepo.GetByID(ctx, execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to load workflow: %w", err)
	}

	// Handle rejection
	if !approved {
		w.logger.Infof("Workflow %s rejected - cancelling execution", executionID)
		execution.Status = models.ExecutionStatusCancelled
		result := models.ExecutionResultBlocked
		execution.Result = &result

		if err := w.executionRepo.UpdateExecution(ctx, execution); err != nil {
			return fmt.Errorf("failed to update execution: %w", err)
		}

		return nil
	}

	// Resume execution with approval event
	resumeData := map[string]interface{}{
		"approved":    true,
		"approved_at": fmt.Sprintf("%d", execution.WaitState.WaitingSince.Unix()),
	}

	_, err = w.executor.ResumeExecution(
		ctx,
		executionID,
		workflow,
		"approval.granted",
		resumeData,
	)
	if err != nil {
		return fmt.Errorf("failed to resume execution: %w", err)
	}

	w.logger.Infof("Workflow %s approved and resumed successfully", executionID)
	return nil
}
