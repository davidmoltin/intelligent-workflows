package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// ExecutionRepository defines the interface for execution persistence
type ExecutionRepository interface {
	CreateExecution(ctx context.Context, execution *models.WorkflowExecution) error
	UpdateExecution(ctx context.Context, execution *models.WorkflowExecution) error
	GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error)
	CreateStepExecution(ctx context.Context, step *models.StepExecution) error
	UpdateStepExecution(ctx context.Context, step *models.StepExecution) error
}

// WorkflowExecutor executes workflows
type WorkflowExecutor struct {
	evaluator       *Evaluator
	contextBuilder  *ContextBuilder
	actionExecutor  *ActionExecutor
	executionRepo   ExecutionRepository
	logger          *logger.Logger
	maxRetries      int
	defaultTimeout  time.Duration
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor(
	redis *redis.Client,
	executionRepo ExecutionRepository,
	log *logger.Logger,
) *WorkflowExecutor {
	return &WorkflowExecutor{
		evaluator:       NewEvaluator(),
		contextBuilder:  NewContextBuilder(redis, log),
		actionExecutor:  NewActionExecutor(log),
		executionRepo:   executionRepo,
		logger:          log,
		maxRetries:      3,
		defaultTimeout:  30 * time.Second,
	}
}

// Execute executes a workflow
func (we *WorkflowExecutor) Execute(
	ctx context.Context,
	workflow *models.Workflow,
	triggerEvent string,
	triggerPayload map[string]interface{},
) (*models.WorkflowExecution, error) {
	we.logger.Infof("Starting workflow execution: %s (ID: %s)", workflow.Name, workflow.ID)

	// Create execution record
	execution := &models.WorkflowExecution{
		ID:             uuid.New(),
		WorkflowID:     workflow.ID,
		ExecutionID:    fmt.Sprintf("exec_%s", uuid.New().String()[:8]),
		TriggerEvent:   triggerEvent,
		TriggerPayload: triggerPayload,
		Status:         models.ExecutionStatusRunning,
		StartedAt:      time.Now(),
		Metadata:       make(models.JSONB),
	}

	if err := we.executionRepo.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Build execution context
	execContext, err := we.contextBuilder.BuildContext(ctx, triggerPayload, workflow.Definition.Context)
	if err != nil {
		we.logger.Errorf("Failed to build context: %v", err)
		we.completeExecution(ctx, execution, models.ExecutionResultFailed, fmt.Sprintf("Context build failed: %v", err))
		return execution, err
	}

	// Enrich context
	if err := we.contextBuilder.EnrichContext(ctx, execContext); err != nil {
		we.logger.Warnf("Failed to enrich context: %v", err)
		// Continue execution even if enrichment fails
	}

	execution.Context = execContext

	// Execute workflow steps
	result, err := we.executeSteps(ctx, execution, workflow, execContext)
	if err != nil {
		we.logger.Errorf("Workflow execution failed: %v", err)
		we.completeExecution(ctx, execution, models.ExecutionResultFailed, err.Error())
		return execution, err
	}

	// Complete execution successfully
	we.completeExecution(ctx, execution, result, "")

	we.logger.Infof("Workflow execution completed: %s - Result: %s", execution.ExecutionID, result)

	return execution, nil
}

// executeSteps executes workflow steps sequentially
func (we *WorkflowExecutor) executeSteps(
	ctx context.Context,
	execution *models.WorkflowExecution,
	workflow *models.Workflow,
	execContext map[string]interface{},
) (models.ExecutionResult, error) {
	// Build step map for navigation
	stepMap := make(map[string]*models.Step)
	for i := range workflow.Definition.Steps {
		step := &workflow.Definition.Steps[i]
		stepMap[step.ID] = step
	}

	// Start with first step
	if len(workflow.Definition.Steps) == 0 {
		return models.ExecutionResultExecuted, nil
	}

	currentStepID := workflow.Definition.Steps[0].ID
	var finalResult models.ExecutionResult = models.ExecutionResultExecuted

	// Execute steps
	for currentStepID != "" {
		step, exists := stepMap[currentStepID]
		if !exists {
			return models.ExecutionResultFailed, fmt.Errorf("step not found: %s", currentStepID)
		}

		we.logger.Infof("Executing step: %s (type: %s)", step.ID, step.Type)

		// Execute step with retry logic
		nextStepID, result, err := we.executeStepWithRetry(ctx, execution, step, execContext)
		if err != nil {
			return models.ExecutionResultFailed, fmt.Errorf("step %s failed: %w", step.ID, err)
		}

		// Update final result based on action result
		if result != nil && result.Action == "block" {
			finalResult = models.ExecutionResultBlocked
		} else if result != nil && result.Action == "allow" {
			finalResult = models.ExecutionResultAllowed
		}

		currentStepID = nextStepID
	}

	return finalResult, nil
}

// executeStepWithRetry executes a single step with retry logic
func (we *WorkflowExecutor) executeStepWithRetry(
	ctx context.Context,
	execution *models.WorkflowExecution,
	step *models.Step,
	execContext map[string]interface{},
) (string, *ActionResult, error) {
	var lastErr error
	maxAttempts := 1

	// Check if step has retry configuration
	if step.Retry != nil && step.Retry.MaxAttempts > 0 {
		maxAttempts = step.Retry.MaxAttempts
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			we.logger.Infof("Retrying step %s (attempt %d/%d)", step.ID, attempt, maxAttempts)

			// Apply backoff
			backoff := we.calculateBackoff(attempt, step.Retry)
			time.Sleep(backoff)
		}

		nextStepID, result, err := we.executeStep(ctx, execution, step, execContext)
		if err == nil {
			return nextStepID, result, nil
		}

		lastErr = err

		// Check if error is retryable
		if step.Retry != nil && !we.isRetryableError(err, step.Retry.RetryOn) {
			break
		}
	}

	return "", nil, fmt.Errorf("step failed after %d attempts: %w", maxAttempts, lastErr)
}

// executeStep executes a single workflow step
func (we *WorkflowExecutor) executeStep(
	ctx context.Context,
	execution *models.WorkflowExecution,
	step *models.Step,
	execContext map[string]interface{},
) (string, *ActionResult, error) {
	// Create step execution record
	stepExec := &models.StepExecution{
		ID:          uuid.New(),
		ExecutionID: execution.ID,
		StepID:      step.ID,
		StepType:    step.Type,
		Status:      models.StepStatusRunning,
		Input:       execContext,
		StartedAt:   time.Now(),
	}

	if err := we.executionRepo.CreateStepExecution(ctx, stepExec); err != nil {
		return "", nil, fmt.Errorf("failed to create step execution: %w", err)
	}

	var nextStepID string
	var actionResult *ActionResult
	var err error

	// Execute based on step type
	switch step.Type {
	case "condition":
		nextStepID, err = we.executeConditionStep(ctx, step, execContext)

	case "action":
		actionResult, err = we.executeActionStep(ctx, step, execContext)
		nextStepID = "" // Action steps end the flow

	case "parallel":
		err = we.executeParallelStep(ctx, execution, step, execContext)
		nextStepID = "" // Parallel steps end the flow for now

	case "wait":
		err = we.executeWaitStep(ctx, step, execContext)
		nextStepID = "" // Wait steps pause the flow

	default:
		err = fmt.Errorf("unsupported step type: %s", step.Type)
	}

	// Update step execution
	now := time.Now()
	stepExec.CompletedAt = &now
	duration := int(now.Sub(stepExec.StartedAt).Milliseconds())
	stepExec.DurationMs = &duration

	if err != nil {
		stepExec.Status = models.StepStatusFailed
		errMsg := err.Error()
		stepExec.ErrorMessage = &errMsg
	} else {
		stepExec.Status = models.StepStatusCompleted
		if actionResult != nil {
			output := make(models.JSONB)
			output["action"] = actionResult.Action
			output["success"] = actionResult.Success
			output["reason"] = actionResult.Reason
			output["data"] = actionResult.Data
			stepExec.Output = output
		}
	}

	if updateErr := we.executionRepo.UpdateStepExecution(ctx, stepExec); updateErr != nil {
		we.logger.Errorf("Failed to update step execution: %v", updateErr)
	}

	return nextStepID, actionResult, err
}

// executeConditionStep executes a condition step
func (we *WorkflowExecutor) executeConditionStep(
	ctx context.Context,
	step *models.Step,
	execContext map[string]interface{},
) (string, error) {
	if step.Condition == nil {
		return "", fmt.Errorf("condition step has no condition defined")
	}

	result, err := we.evaluator.EvaluateCondition(step.Condition, execContext)
	if err != nil {
		return "", fmt.Errorf("condition evaluation failed: %w", err)
	}

	we.logger.Infof("Condition evaluated to: %v", result)

	if result {
		return step.OnTrue, nil
	}
	return step.OnFalse, nil
}

// executeActionStep executes an action step
func (we *WorkflowExecutor) executeActionStep(
	ctx context.Context,
	step *models.Step,
	execContext map[string]interface{},
) (*ActionResult, error) {
	return we.actionExecutor.ExecuteAction(ctx, step, execContext)
}

// executeParallelStep executes steps in parallel
func (we *WorkflowExecutor) executeParallelStep(
	ctx context.Context,
	execution *models.WorkflowExecution,
	step *models.Step,
	execContext map[string]interface{},
) error {
	if step.Parallel == nil || len(step.Parallel.Steps) == 0 {
		return fmt.Errorf("parallel step has no steps defined")
	}

	we.logger.Infof("Executing %d steps in parallel", len(step.Parallel.Steps))

	var wg sync.WaitGroup
	results := make([]error, len(step.Parallel.Steps))

	for i, parallelStep := range step.Parallel.Steps {
		wg.Add(1)
		go func(index int, s models.Step) {
			defer wg.Done()
			_, _, err := we.executeStep(ctx, execution, &s, execContext)
			results[index] = err
		}(i, parallelStep)
	}

	wg.Wait()

	// Evaluate results based on strategy
	strategy := step.Parallel.Strategy
	if strategy == "" {
		strategy = "all_must_pass"
	}

	switch strategy {
	case "all_must_pass":
		for i, err := range results {
			if err != nil {
				return fmt.Errorf("parallel step %d failed: %w", i, err)
			}
		}
		return nil

	case "any_can_pass":
		for _, err := range results {
			if err == nil {
				return nil
			}
		}
		return fmt.Errorf("all parallel steps failed")

	case "best_effort":
		// Continue even if some steps fail
		failedCount := 0
		for _, err := range results {
			if err != nil {
				failedCount++
			}
		}
		we.logger.Infof("Parallel execution: %d/%d steps succeeded", len(results)-failedCount, len(results))
		return nil

	default:
		return fmt.Errorf("unknown parallel strategy: %s", strategy)
	}
}

// executeWaitStep executes a wait step (placeholder)
func (we *WorkflowExecutor) executeWaitStep(
	ctx context.Context,
	step *models.Step,
	execContext map[string]interface{},
) error {
	// Wait steps would pause execution and wait for an event or timeout
	// This is a simplified implementation
	we.logger.Infof("Wait step: waiting for event %s", step.Wait.Event)
	// TODO: Implement actual wait/pause mechanism
	return nil
}

// completeExecution marks an execution as complete
func (we *WorkflowExecutor) completeExecution(
	ctx context.Context,
	execution *models.WorkflowExecution,
	result models.ExecutionResult,
	errorMsg string,
) {
	now := time.Now()
	execution.CompletedAt = &now
	duration := int(now.Sub(execution.StartedAt).Milliseconds())
	execution.DurationMs = &duration
	execution.Result = &result

	if errorMsg != "" {
		execution.Status = models.ExecutionStatusFailed
		execution.ErrorMessage = &errorMsg
	} else {
		execution.Status = models.ExecutionStatusCompleted
	}

	if err := we.executionRepo.UpdateExecution(ctx, execution); err != nil {
		we.logger.Errorf("Failed to update execution: %v", err)
	}
}

// calculateBackoff calculates backoff duration for retries
func (we *WorkflowExecutor) calculateBackoff(attempt int, retryConfig *models.RetryConfig) time.Duration {
	if retryConfig == nil || retryConfig.Backoff == "" {
		// Default: exponential backoff
		return time.Duration(1<<uint(attempt-1)) * time.Second
	}

	switch retryConfig.Backoff {
	case "linear":
		return time.Duration(attempt) * time.Second
	case "exponential":
		return time.Duration(1<<uint(attempt-1)) * time.Second
	default:
		return time.Second
	}
}

// isRetryableError checks if an error should be retried
func (we *WorkflowExecutor) isRetryableError(err error, retryOn []string) bool {
	if len(retryOn) == 0 {
		// Retry all errors by default
		return true
	}

	errMsg := err.Error()
	for _, pattern := range retryOn {
		if pattern == "*" {
			return true
		}
		// Simple string matching (could be enhanced with regex)
		if fmt.Sprintf("%v", err) == pattern || errMsg == pattern {
			return true
		}
	}

	return false
}
