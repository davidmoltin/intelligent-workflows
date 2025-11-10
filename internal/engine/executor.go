package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/websocket"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/davidmoltin/intelligent-workflows/pkg/metrics"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ExecutionRepository defines the interface for execution persistence
type ExecutionRepository interface {
	CreateExecution(ctx context.Context, execution *models.WorkflowExecution) error
	UpdateExecution(ctx context.Context, organizationID uuid.UUID, execution *models.WorkflowExecution) error
	GetExecutionByID(ctx context.Context, organizationID, id uuid.UUID) (*models.WorkflowExecution, error)
	CreateStepExecution(ctx context.Context, step *models.StepExecution) error
	UpdateStepExecution(ctx context.Context, organizationID uuid.UUID, step *models.StepExecution) error
	GetTimedOutExecutions(ctx context.Context, organizationID uuid.UUID, limit int) ([]*models.WorkflowExecution, error)
}

// RuleService interface for loading rules
type RuleService interface {
	GetByRuleID(ctx context.Context, organizationID uuid.UUID, ruleID string) (*models.Rule, error)
}

// WorkflowExecutor executes workflows
type WorkflowExecutor struct {
	evaluator      *Evaluator
	contextBuilder *ContextBuilder
	actionExecutor *ActionExecutor
	executionRepo  ExecutionRepository
	workflowRepo   WorkflowRepository
	ruleService    RuleService
	wsHub          *websocket.Hub
	logger         *logger.Logger
	metrics        *metrics.Metrics
	maxRetries     int
	defaultTimeout time.Duration
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor(
	redis *redis.Client,
	executionRepo ExecutionRepository,
	workflowRepo WorkflowRepository,
	wsHub *websocket.Hub,
	log *logger.Logger,
	m *metrics.Metrics,
	contextEnrichmentCfg *config.ContextEnrichmentConfig,
) *WorkflowExecutor {
	return &WorkflowExecutor{
		evaluator:      NewEvaluator(),
		contextBuilder: NewContextBuilder(redis, log, contextEnrichmentCfg),
		actionExecutor: NewActionExecutor(log),
		executionRepo:  executionRepo,
		workflowRepo:   workflowRepo,
		wsHub:          wsHub,
		logger:         log,
		metrics:        m,
		maxRetries:     3,
		defaultTimeout: 30 * time.Second,
	}
}

// SetRuleService sets the rule service for the executor (optional dependency)
func (we *WorkflowExecutor) SetRuleService(ruleService RuleService) {
	we.ruleService = ruleService
}

// SetApprovalService sets the approval service for the action executor (optional dependency)
func (we *WorkflowExecutor) SetApprovalService(approvalService ApprovalService) {
	we.actionExecutor.SetApprovalService(approvalService)
}

// Execute executes a workflow
func (we *WorkflowExecutor) Execute(
	ctx context.Context,
	organizationID uuid.UUID,
	workflow *models.Workflow,
	triggerEvent string,
	triggerPayload map[string]interface{},
) (*models.WorkflowExecution, error) {
	// Track execution start time for metrics
	startTime := time.Now()
	workflowIDStr := workflow.ID.String()

	// Increment active workflows gauge
	if we.metrics != nil {
		we.metrics.ActiveWorkflows.WithLabelValues(workflowIDStr).Inc()
		defer we.metrics.ActiveWorkflows.WithLabelValues(workflowIDStr).Dec()
	}

	we.logger.Infof("Starting workflow execution: %s (ID: %s) for organization: %s", workflow.Name, workflow.ID, organizationID)

	// Get timeout for this workflow (check Definition.Timeout first, then trigger data, then default)
	timeout := we.getWorkflowTimeout(workflow)

	// Apply workflow-level timeout
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		we.logger.Infof("Workflow timeout set to: %v", timeout)
	}

	// Create execution record
	execution := &models.WorkflowExecution{
		ID:             uuid.New(),
		OrganizationID: organizationID,
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
		// Also store in metadata for observability
		execution.Metadata["timeout_seconds"] = timeout.Seconds()
	}

	if err := we.executionRepo.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Set execution context for action executor (needed for approval requests)
	we.actionExecutor.SetExecutionContext(execution.ID)

	// Broadcast execution started event
	we.broadcastExecutionEvent(execution)

	// Build execution context
	execContext, err := we.contextBuilder.BuildContext(ctx, workflow.OrganizationID, triggerPayload, workflow.Definition.Context)
	if err != nil {
		we.logger.Errorf("Failed to build context: %v", err)
		we.completeExecution(ctx, execution, models.ExecutionResultFailed, fmt.Sprintf("Context build failed: %v", err))
		// Record metrics for failed execution
		if we.metrics != nil {
			we.metrics.WorkflowExecutionsTotal.WithLabelValues(workflowIDStr, "failed").Inc()
			we.metrics.WorkflowDuration.WithLabelValues(workflowIDStr).Observe(time.Since(startTime).Seconds())
			we.metrics.WorkflowErrors.WithLabelValues(workflowIDStr, "context_build_error").Inc()
		}
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
			// Record metrics for paused execution
			if we.metrics != nil {
				we.metrics.WorkflowExecutionsTotal.WithLabelValues(workflowIDStr, "paused").Inc()
				we.metrics.WorkflowDuration.WithLabelValues(workflowIDStr).Observe(time.Since(startTime).Seconds())
			}
			return execution, nil
		}

		// Check for timeout
		if ctx.Err() == context.DeadlineExceeded {
			we.logger.Errorf("Workflow execution timed out: %s", execution.ExecutionID)
			timeoutMsg := fmt.Sprintf("Workflow execution timed out after %v", timeout)
			we.completeExecution(context.Background(), execution, models.ExecutionResultFailed, timeoutMsg)
			// Record metrics for timeout
			if we.metrics != nil {
				we.metrics.WorkflowExecutionsTotal.WithLabelValues(workflowIDStr, "timeout").Inc()
				we.metrics.WorkflowDuration.WithLabelValues(workflowIDStr).Observe(time.Since(startTime).Seconds())
				we.metrics.WorkflowErrors.WithLabelValues(workflowIDStr, "timeout").Inc()
			}
			return execution, fmt.Errorf("%s: %w", timeoutMsg, ctx.Err())
		}

		we.logger.Errorf("Workflow execution failed: %v", err)
		we.completeExecution(ctx, execution, models.ExecutionResultFailed, err.Error())
		// Record metrics for failed execution
		if we.metrics != nil {
			we.metrics.WorkflowExecutionsTotal.WithLabelValues(workflowIDStr, "failed").Inc()
			we.metrics.WorkflowDuration.WithLabelValues(workflowIDStr).Observe(time.Since(startTime).Seconds())
			we.metrics.WorkflowErrors.WithLabelValues(workflowIDStr, "execution_error").Inc()
		}
		return execution, err
	}

	// Complete execution successfully
	we.completeExecution(ctx, execution, result, "")

	we.logger.Infof("Workflow execution completed: %s - Result: %s", execution.ExecutionID, result)

	// Record metrics for successful execution
	if we.metrics != nil {
		we.metrics.WorkflowExecutionsTotal.WithLabelValues(workflowIDStr, string(result)).Inc()
		we.metrics.WorkflowDuration.WithLabelValues(workflowIDStr).Observe(time.Since(startTime).Seconds())
	}

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
		// Check for context cancellation (timeout or manual cancellation)
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return models.ExecutionResultFailed, fmt.Errorf("workflow execution timed out")
			}
			return models.ExecutionResultFailed, ctx.Err()
		default:
			// Continue execution
		}

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
		ID:             uuid.New(),
		OrganizationID: execution.OrganizationID,
		ExecutionID:    execution.ID,
		StepID:         step.ID,
		StepType:       step.Type,
		Status:         models.StepStatusRunning,
		Input:          execContext,
		StartedAt:      time.Now(),
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
		nextStepID, err = we.executeConditionStep(ctx, execution, step, execContext)

	case "action":
		actionResult, err = we.executeActionStep(ctx, step, execContext)
		nextStepID = "" // Action steps end the flow

	case "execute":
		actionResult, err = we.executeExecuteStep(ctx, step, execContext)
		nextStepID = "" // Execute steps end the flow

	case "parallel":
		err = we.executeParallelStep(ctx, execution, step, execContext)
		nextStepID = step.Next // Support next step after parallel

	case "foreach":
		err = we.executeForEachStep(ctx, execution, step, execContext)
		nextStepID = step.Next // Support next step after foreach

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

	if updateErr := we.executionRepo.UpdateStepExecution(ctx, stepExec.OrganizationID, stepExec); updateErr != nil {
		we.logger.Errorf("Failed to update step execution: %v", updateErr)
	}

	return nextStepID, actionResult, err
}

// executeConditionStep executes a condition step
func (we *WorkflowExecutor) executeConditionStep(
	ctx context.Context,
	execution *models.WorkflowExecution,
	step *models.Step,
	execContext map[string]interface{},
) (string, error) {
	var condition *models.Condition

	// Check if step references a rule
	if step.RuleID != "" {
		if we.ruleService == nil {
			return "", fmt.Errorf("rule_id specified but rule service not configured")
		}

		// Load the rule with organization_id for proper multi-tenancy isolation
		rule, err := we.ruleService.GetByRuleID(ctx, execution.OrganizationID, step.RuleID)
		if err != nil {
			return "", fmt.Errorf("failed to load rule %s: %w", step.RuleID, err)
		}

		// Check if rule is enabled
		if !rule.Enabled {
			return "", fmt.Errorf("rule %s is disabled", step.RuleID)
		}

		// Check if rule is a condition type
		if rule.RuleType != models.RuleTypeCondition {
			return "", fmt.Errorf("rule %s is not a condition rule (type: %s)", step.RuleID, rule.RuleType)
		}

		// Use the first condition from the rule definition
		if len(rule.Definition.Conditions) == 0 {
			return "", fmt.Errorf("rule %s has no conditions defined", step.RuleID)
		}

		condition = &rule.Definition.Conditions[0]
		we.logger.Infof("Using rule %s for condition evaluation", step.RuleID)
	} else if step.Condition != nil {
		// Use inline condition
		condition = step.Condition
	} else {
		return "", fmt.Errorf("condition step has no condition or rule_id defined")
	}

	// Evaluate the condition
	result, err := we.evaluator.EvaluateCondition(condition, execContext)
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

// executeExecuteStep executes an execute step (webhooks, notifications, etc.)
func (we *WorkflowExecutor) executeExecuteStep(
	ctx context.Context,
	step *models.Step,
	execContext map[string]interface{},
) (*ActionResult, error) {
	if len(step.Execute) == 0 {
		return nil, fmt.Errorf("execute step has no execute actions defined")
	}

	// Create a synthetic Action with type "execute" to reuse existing logic
	syntheticStep := &models.Step{
		ID:   step.ID,
		Type: "action",
		Action: &models.Action{
			Type: "execute",
		},
		Execute: step.Execute,
	}

	return we.actionExecutor.ExecuteAction(ctx, syntheticStep, execContext)
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

// executeForEachStep executes steps for each item in a collection
func (we *WorkflowExecutor) executeForEachStep(
	ctx context.Context,
	execution *models.WorkflowExecution,
	step *models.Step,
	execContext map[string]interface{},
) error {
	if step.ForEach == nil || len(step.ForEach.Steps) == 0 {
		return fmt.Errorf("foreach step has no steps defined")
	}

	if step.ForEach.Items == "" {
		return fmt.Errorf("foreach step has no items specified")
	}

	if step.ForEach.ItemVar == "" {
		return fmt.Errorf("foreach step has no item_var specified")
	}

	// Resolve the items collection from context
	items, err := we.resolveItemsCollection(step.ForEach.Items, execContext)
	if err != nil {
		return fmt.Errorf("failed to resolve items collection: %w", err)
	}

	we.logger.Infof("Executing foreach loop over %d items", len(items))

	// Execute steps for each item
	for i, item := range items {
		// Create a new context with the item variable
		itemContext := make(map[string]interface{})
		for k, v := range execContext {
			itemContext[k] = v
		}
		itemContext[step.ForEach.ItemVar] = item
		itemContext["_index"] = i

		we.logger.Infof("Executing foreach iteration %d/%d", i+1, len(items))

		// Execute all steps for this item
		for _, foreachStep := range step.ForEach.Steps {
			_, _, err := we.executeStep(ctx, execution, &foreachStep, itemContext)
			if err != nil {
				return fmt.Errorf("foreach iteration %d, step %s failed: %w", i, foreachStep.ID, err)
			}
		}
	}

	we.logger.Infof("Foreach loop completed: processed %d items", len(items))
	return nil
}

// resolveItemsCollection resolves a collection from a variable reference or JSONPath
func (we *WorkflowExecutor) resolveItemsCollection(items string, execContext map[string]interface{}) ([]interface{}, error) {
	// Handle variable references like {{items}} or {{order.line_items}}
	if len(items) > 4 && items[:2] == "{{" && items[len(items)-2:] == "}}" {
		varPath := items[2 : len(items)-2]
		value := we.evaluator.ResolveVariable(varPath, execContext)

		if value == nil {
			return nil, fmt.Errorf("variable %s not found in context", varPath)
		}

		// Convert value to array
		switch v := value.(type) {
		case []interface{}:
			return v, nil
		case []map[string]interface{}:
			result := make([]interface{}, len(v))
			for i, item := range v {
				result[i] = item
			}
			return result, nil
		default:
			return nil, fmt.Errorf("variable %s is not a collection (type: %T)", varPath, value)
		}
	}

	// If not a variable reference, try to interpret as literal JSON array
	return nil, fmt.Errorf("items must be a variable reference like {{variable}}")
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

	if err := we.executionRepo.UpdateExecution(ctx, execution.OrganizationID, execution); err != nil {
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
	// Use workflow's organization ID if available, otherwise uuid.Nil
	orgID := uuid.Nil
	if workflow != nil {
		orgID = workflow.OrganizationID
	}
	execution, err := we.executionRepo.GetExecutionByID(ctx, orgID, executionID)
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
	if err := we.contextBuilder.BuildContextFromExisting(ctx, workflow.OrganizationID, execContext, workflow.Definition.Context); err != nil {
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

	if err := we.executionRepo.UpdateExecution(ctx, execution.OrganizationID, execution); err != nil {
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

	if err := we.executionRepo.UpdateExecution(ctx, execution.OrganizationID, execution); err != nil {
		we.logger.Errorf("Failed to update execution: %v", err)
	}

	// Broadcast execution completion event
	we.broadcastExecutionEvent(execution)
}

// broadcastExecutionEvent broadcasts an execution state change via WebSocket
func (we *WorkflowExecutor) broadcastExecutionEvent(execution *models.WorkflowExecution) {
	if we.wsHub == nil {
		return
	}

	var msgType websocket.MessageType
	switch execution.Status {
	case models.ExecutionStatusRunning:
		msgType = websocket.MessageTypeExecutionStarted
	case models.ExecutionStatusCompleted:
		msgType = websocket.MessageTypeExecutionCompleted
	case models.ExecutionStatusFailed:
		msgType = websocket.MessageTypeExecutionFailed
	case models.ExecutionStatusPaused, models.ExecutionStatusWaiting:
		msgType = websocket.MessageTypeExecutionPaused
	case models.ExecutionStatusCancelled:
		msgType = websocket.MessageTypeExecutionCancelled
	default:
		return
	}

	eventData := &websocket.ExecutionEventData{
		ExecutionID:  execution.ExecutionID,
		WorkflowID:   execution.WorkflowID.String(),
		Status:       string(execution.Status),
		TriggerEvent: execution.TriggerEvent,
		StartedAt:    &execution.StartedAt,
		CompletedAt:  execution.CompletedAt,
		DurationMs:   execution.DurationMs,
		Context:      execution.Context,
	}

	if execution.Result != nil {
		eventData.Result = string(*execution.Result)
	}

	if execution.ErrorMessage != nil {
		eventData.ErrorMessage = *execution.ErrorMessage
	}

	if execution.PausedReason != nil {
		eventData.PausedReason = *execution.PausedReason
	}

	we.wsHub.BroadcastExecutionEvent(msgType, eventData)
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
	workflow, err := we.workflowRepo.GetWorkflowByID(ctx, execution.OrganizationID, execution.WorkflowID)
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
	if err := we.executionRepo.UpdateExecution(ctx, execution.OrganizationID, execution); err != nil {
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

// getWorkflowTimeout returns the timeout duration for a workflow
// It checks (1) workflow Definition.Timeout, (2) trigger data timeout_seconds, or (3) default
func (we *WorkflowExecutor) getWorkflowTimeout(workflow *models.Workflow) time.Duration {
	// First, check if workflow has timeout defined in Definition.Timeout
	if workflow.Definition.Timeout != "" {
		timeout := we.parseTimeout(workflow.Definition.Timeout, 0)
		if timeout > 0 {
			return timeout
		}
	}

	// Second, check if workflow has custom timeout in trigger metadata
	if workflow.Definition.Trigger.Data != nil {
		if timeoutVal, ok := workflow.Definition.Trigger.Data["timeout_seconds"]; ok {
			// Try to convert to float64 (JSON numbers are float64)
			if timeoutSeconds, ok := timeoutVal.(float64); ok && timeoutSeconds > 0 {
				return time.Duration(timeoutSeconds) * time.Second
			}
			// Try to convert to int
			if timeoutSeconds, ok := timeoutVal.(int); ok && timeoutSeconds > 0 {
				return time.Duration(timeoutSeconds) * time.Second
			}
		}
	}

	// Return default timeout
	return we.defaultTimeout
}
