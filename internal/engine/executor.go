package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	evaluator      *Evaluator
	contextBuilder *ContextBuilder
	actionExecutor *ActionExecutor
	executionRepo  ExecutionRepository
	workflowRepo   WorkflowRepository
	logger         *logger.Logger
	maxRetries     int
	defaultTimeout time.Duration
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor(
	redis *redis.Client,
	executionRepo ExecutionRepository,
	workflowRepo WorkflowRepository,
	log *logger.Logger,
) *WorkflowExecutor {
	return &WorkflowExecutor{
		evaluator:      NewEvaluator(),
		contextBuilder: NewContextBuilder(redis, log),
		actionExecutor: NewActionExecutor(log),
		executionRepo:  executionRepo,
		workflowRepo:   workflowRepo,
		logger:         log,
		maxRetries:     3,
		defaultTimeout: 30 * time.Second,
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

	// Apply workflow-level timeout
	var cancel context.CancelFunc
	timeout := we.parseTimeout(workflow.Definition.Timeout, we.defaultTimeout)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		we.logger.Infof("Workflow timeout set to: %v", timeout)
	}

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

	// Set timeout fields if timeout is configured
	if timeout > 0 {
		timeoutAt := execution.StartedAt.Add(timeout)
		execution.TimeoutAt = &timeoutAt
		timeoutSecs := int(timeout.Seconds())
		execution.TimeoutDuration = &timeoutSecs
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
		// Check if execution was paused (not a real error)
		if err == ErrExecutionPaused {
			we.logger.Infof("Workflow execution paused: %s", execution.ExecutionID)
			return execution, nil
		}

		// Check for timeout
		if ctx.Err() == context.DeadlineExceeded {
			we.logger.Errorf("Workflow execution timed out: %s", execution.ExecutionID)
			timeoutMsg := fmt.Sprintf("Workflow execution timed out after %v", timeout)
			we.completeExecution(context.Background(), execution, models.ExecutionResultFailed, timeoutMsg)
			return execution, fmt.Errorf("%s: %w", timeoutMsg, ctx.Err())
		}

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

	// Apply step-level timeout if configured
	stepCtx := ctx
	var cancel context.CancelFunc
	if step.Timeout != "" {
		timeout := we.parseTimeout(step.Timeout, 0)
		if timeout > 0 {
			stepCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
			we.logger.Infof("Step %s timeout set to: %v", step.ID, timeout)
		}
	}

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

		nextStepID, result, err := we.executeStep(stepCtx, execution, step, execContext)
		if err == nil {
			return nextStepID, result, nil
		}

		// Check for step timeout
		if stepCtx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("step %s timed out: %w", step.ID, stepCtx.Err())
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
		err = we.executeWaitStep(ctx, execution, step, execContext)
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

// ErrExecutionPaused is returned when a workflow execution is paused (waiting)
var ErrExecutionPaused = fmt.Errorf("execution paused for wait step")

// executeWaitStep executes a wait step by pausing the execution
func (we *WorkflowExecutor) executeWaitStep(
	ctx context.Context,
	execution *models.WorkflowExecution,
	step *models.Step,
	execContext map[string]interface{},
) error {
	if step.Wait == nil {
		return fmt.Errorf("wait step has no wait configuration")
	}

	we.logger.Infof("Wait step: pausing execution to wait for event %s", step.Wait.Event)

	// Calculate timeout if specified
	var timeoutAt *time.Time
	if step.Wait.Timeout != "" {
		duration, err := time.ParseDuration(step.Wait.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout duration: %w", err)
		}
		timeout := time.Now().Add(duration)
		timeoutAt = &timeout
	}

	// Create wait state
	waitState := &models.WaitState{
		Event:        step.Wait.Event,
		TimeoutAt:    timeoutAt,
		OnTimeout:    step.Wait.OnTimeout,
		WaitingSince: time.Now(),
	}

	// Update execution to waiting status
	execution.Status = models.ExecutionStatusWaiting
	execution.CurrentStepID = &step.ID
	execution.WaitState = waitState

	if err := we.executionRepo.UpdateExecution(ctx, execution); err != nil {
		return fmt.Errorf("failed to update execution to waiting state: %w", err)
	}

	we.logger.Infof("Execution %s paused, waiting for event: %s", execution.ExecutionID, step.Wait.Event)

	// Return special error to signal pause
	return ErrExecutionPaused
}

// ResumeExecution resumes a paused workflow execution
func (we *WorkflowExecutor) ResumeExecution(
	ctx context.Context,
	executionID uuid.UUID,
	workflow *models.Workflow,
	resumeEvent string,
	resumeData map[string]interface{},
) (*models.WorkflowExecution, error) {
	we.logger.Infof("Resuming workflow execution: %s with event: %s", executionID, resumeEvent)

	// Load execution from database
	execution, err := we.executionRepo.GetExecutionByID(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load execution: %w", err)
	}

	// Verify execution is in waiting state
	if execution.Status != models.ExecutionStatusWaiting {
		return nil, fmt.Errorf("execution is not in waiting state, current status: %s", execution.Status)
	}

	if execution.WaitState == nil {
		return nil, fmt.Errorf("execution has no wait state")
	}

	// Verify event matches what we're waiting for
	if execution.WaitState.Event != resumeEvent {
		return nil, fmt.Errorf("unexpected event: waiting for %s, got %s", execution.WaitState.Event, resumeEvent)
	}

	// Load and enrich context with resume data
	execContext := map[string]interface{}(execution.Context)
	if resumeData != nil {
		// Merge resume data into context
		execContext["resume_event"] = resumeData
	}

	// Reload context data from sources to ensure freshness
	if err := we.contextBuilder.BuildContextFromExisting(ctx, execContext, workflow.Definition.Context); err != nil {
		we.logger.Warnf("Failed to reload context: %v", err)
		// Continue with existing context
	}

	if err := we.contextBuilder.EnrichContext(ctx, execContext); err != nil {
		we.logger.Warnf("Failed to enrich context: %v", err)
	}

	execution.Context = execContext

	// Update execution status back to running
	execution.Status = models.ExecutionStatusRunning
	execution.WaitState = nil

	if err := we.executionRepo.UpdateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to update execution status: %w", err)
	}

	// Find the next step after the wait step
	nextStepID := ""
	if execution.CurrentStepID != nil {
		// Build step map
		stepMap := make(map[string]*models.Step)
		for i := range workflow.Definition.Steps {
			step := &workflow.Definition.Steps[i]
			stepMap[step.ID] = step
		}

		// Get the wait step to find what comes next
		waitStep, exists := stepMap[*execution.CurrentStepID]
		if !exists {
			return nil, fmt.Errorf("wait step not found: %s", *execution.CurrentStepID)
		}

		// Determine next step based on wait step metadata or sequential flow
		if waitStep.Metadata != nil {
			if next, ok := waitStep.Metadata["on_resume"].(string); ok {
				nextStepID = next
			}
		}

		// If no explicit next step, find the step after wait step in sequence
		if nextStepID == "" {
			for i, s := range workflow.Definition.Steps {
				if s.ID == *execution.CurrentStepID && i+1 < len(workflow.Definition.Steps) {
					nextStepID = workflow.Definition.Steps[i+1].ID
					break
				}
			}
		}
	}

	execution.CurrentStepID = nil

	// Continue execution from next step
	result, err := we.continueStepsFrom(ctx, execution, workflow, execContext, nextStepID)
	if err != nil {
		// Check if execution was paused again
		if err == ErrExecutionPaused {
			we.logger.Infof("Workflow execution paused again: %s", execution.ExecutionID)
			return execution, nil
		}

		we.logger.Errorf("Workflow execution failed after resume: %v", err)
		we.completeExecution(ctx, execution, models.ExecutionResultFailed, err.Error())
		return execution, err
	}

	// Complete execution successfully
	we.completeExecution(ctx, execution, result, "")

	we.logger.Infof("Resumed workflow execution completed: %s - Result: %s", execution.ExecutionID, result)

	return execution, nil
}

// continueStepsFrom continues execution from a specific step ID
func (we *WorkflowExecutor) continueStepsFrom(
	ctx context.Context,
	execution *models.WorkflowExecution,
	workflow *models.Workflow,
	execContext map[string]interface{},
	startStepID string,
) (models.ExecutionResult, error) {
	// Build step map for navigation
	stepMap := make(map[string]*models.Step)
	for i := range workflow.Definition.Steps {
		step := &workflow.Definition.Steps[i]
		stepMap[step.ID] = step
	}

	currentStepID := startStepID
	var finalResult models.ExecutionResult = models.ExecutionResultExecuted

	// Execute steps from the given starting point
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

// parseTimeout parses a timeout string (e.g., "5m", "30s", "1h") and returns a duration
// Returns defaultTimeout if timeoutStr is empty or invalid
func (we *WorkflowExecutor) parseTimeout(timeoutStr string, defaultTimeout time.Duration) time.Duration {
	if timeoutStr == "" {
		return defaultTimeout
	}

	duration, err := time.ParseDuration(timeoutStr)
	if err != nil {
		we.logger.Warnf("Invalid timeout format '%s', using default: %v", timeoutStr, defaultTimeout)
		return defaultTimeout
	}

	if duration <= 0 {
		we.logger.Warnf("Timeout must be positive, got %v, using default: %v", duration, defaultTimeout)
		return defaultTimeout
	}

	return duration
}

// ResumePausedExecution resumes a paused workflow execution
func (we *WorkflowExecutor) ResumePausedExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	we.logger.Infof("Resuming paused execution %s (resume count: %d)", execution.ID, execution.ResumeCount)

	// Validate execution state
	if execution.Status != models.ExecutionStatusRunning {
		return fmt.Errorf("execution must be in running state to resume (current: %s)", execution.Status)
	}

	// Load workflow definition
	workflow, err := we.workflowRepo.GetWorkflowByID(ctx, execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to load workflow: %w", err)
	}

	// Restore execution context from resume_data
	execContext := make(map[string]interface{})
	if execution.Context != nil {
		execContext = execution.Context
	}

	// Merge resume data into context (e.g., approval decision)
	if execution.ResumeData != nil {
		for key, value := range execution.ResumeData {
			execContext[key] = value
		}
	}

	// Determine where to resume from
	var startStepID string
	if execution.NextStepID != nil {
		// Resume from the next step
		startStepID = execution.NextStepID.String()
		we.logger.Infof("Resuming from next step: %s", startStepID)
	} else if execution.PausedStepID != nil {
		// Re-execute the paused step (e.g., after approval)
		startStepID = execution.PausedStepID.String()
		we.logger.Infof("Re-executing paused step: %s", startStepID)
	} else {
		// No specific step, start from the beginning
		if len(workflow.Definition.Steps) > 0 {
			startStepID = workflow.Definition.Steps[0].ID
		}
		we.logger.Warn("No paused/next step specified, starting from beginning")
	}

	// Continue execution from the specified step
	result, err := we.continueFromStep(ctx, execution, workflow, execContext, startStepID)
	if err != nil {
		we.logger.Errorf("Failed to resume workflow execution: %v", err)
		we.completeExecution(ctx, execution, models.ExecutionResultFailed, err.Error())
		return err
	}

	// Complete execution successfully
	we.completeExecution(ctx, execution, result, "")
	we.logger.Infof("Workflow execution resumed and completed: %s - Result: %s", execution.ExecutionID, result)

	return nil
}

// PauseCurrentExecution pauses the current workflow execution
func (we *WorkflowExecutor) PauseCurrentExecution(
	ctx context.Context,
	execution *models.WorkflowExecution,
	reason string,
	pausedStepID string,
	nextStepID string,
) error {
	we.logger.Infof("Pausing execution %s at step %s: %s", execution.ID, pausedStepID, reason)

	now := time.Now()
	execution.Status = models.ExecutionStatusPaused
	execution.PausedAt = &now
	execution.PausedReason = &reason

	if pausedStepID != "" {
		pausedID := uuid.MustParse(pausedStepID)
		execution.PausedStepID = &pausedID
	}

	if nextStepID != "" {
		nextID := uuid.MustParse(nextStepID)
		execution.NextStepID = &nextID
	}

	// Store current context in resume_data
	if execution.ResumeData == nil {
		execution.ResumeData = make(models.JSONB)
	}
	execution.ResumeData["paused_context"] = execution.Context

	// Update execution in database
	if err := we.executionRepo.UpdateExecution(ctx, execution); err != nil {
		return fmt.Errorf("failed to pause execution: %w", err)
	}

	we.logger.Infof("Successfully paused execution %s", execution.ID)
	return nil
}

// continueFromStep continues workflow execution from a specific step
func (we *WorkflowExecutor) continueFromStep(
	ctx context.Context,
	execution *models.WorkflowExecution,
	workflow *models.Workflow,
	execContext map[string]interface{},
	startStepID string,
) (models.ExecutionResult, error) {
	// Build step map for navigation
	stepMap := make(map[string]*models.Step)
	for i := range workflow.Definition.Steps {
		step := &workflow.Definition.Steps[i]
		stepMap[step.ID] = step
	}

	// Validate start step exists
	if startStepID != "" {
		if _, exists := stepMap[startStepID]; !exists {
			return models.ExecutionResultFailed, fmt.Errorf("start step not found: %s", startStepID)
		}
	}

	currentStepID := startStepID
	var finalResult models.ExecutionResult = models.ExecutionResultExecuted

	// Execute steps from the specified start point
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
