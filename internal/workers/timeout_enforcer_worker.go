package workers

import (
	"context"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// TimeoutEnforcerWorker handles periodic timeout enforcement for running workflows
type TimeoutEnforcerWorker struct {
	executionRepo *postgres.ExecutionRepository
	logger        *logger.Logger
	checkInterval time.Duration
	stopCh        chan struct{}
	doneCh        chan struct{}
}

// NewTimeoutEnforcerWorker creates a new timeout enforcer worker
func NewTimeoutEnforcerWorker(
	executionRepo *postgres.ExecutionRepository,
	log *logger.Logger,
	checkInterval time.Duration,
) *TimeoutEnforcerWorker {
	if checkInterval == 0 {
		checkInterval = 1 * time.Minute // Default to 1 minute
	}

	return &TimeoutEnforcerWorker{
		executionRepo: executionRepo,
		logger:        log,
		checkInterval: checkInterval,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

// Start starts the worker in the background
func (w *TimeoutEnforcerWorker) Start(ctx context.Context) {
	w.logger.Info("Starting timeout enforcer worker",
		logger.String("interval", w.checkInterval.String()),
	)

	go w.run(ctx)
}

// Stop stops the worker gracefully
func (w *TimeoutEnforcerWorker) Stop() {
	w.logger.Info("Stopping timeout enforcer worker")
	close(w.stopCh)
	<-w.doneCh
	w.logger.Info("Timeout enforcer worker stopped")
}

// run is the main worker loop
func (w *TimeoutEnforcerWorker) run(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.enforceTimeouts(ctx)

	for {
		select {
		case <-ticker.C:
			w.enforceTimeouts(ctx)
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// enforceTimeouts checks for timed-out executions and fails them
func (w *TimeoutEnforcerWorker) enforceTimeouts(ctx context.Context) {
	w.logger.Debug("Checking for timed-out executions")

	// Find all running/waiting executions with timeout_at < NOW()
	executions, err := w.executionRepo.GetTimedOutExecutions(ctx, 100)
	if err != nil {
		w.logger.Errorf("Failed to fetch timed-out executions: %v", err)
		return
	}

	if len(executions) == 0 {
		w.logger.Debug("No timed-out executions found")
		return
	}

	w.logger.Infof("Found %d timed-out executions to process", len(executions))

	failedCount := 0
	for _, execution := range executions {
		if err := w.failTimedOutExecution(ctx, execution); err != nil {
			w.logger.Errorf("Failed to fail timed-out execution %s: %v", execution.ExecutionID, err)
			continue
		}
		failedCount++
	}

	w.logger.Infof("Timeout enforcement completed: failed=%d, errors=%d",
		failedCount, len(executions)-failedCount)
}

// failTimedOutExecution marks a timed-out execution as failed
func (w *TimeoutEnforcerWorker) failTimedOutExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	w.logger.Infof("Failing timed-out execution: %s (timeout: %v)",
		execution.ExecutionID, execution.TimeoutAt)

	// Update execution to failed status
	now := time.Now()
	execution.Status = models.ExecutionStatusFailed
	failedResult := models.ExecutionResultFailed
	execution.Result = &failedResult
	execution.CompletedAt = &now
	duration := int(now.Sub(execution.StartedAt).Milliseconds())
	execution.DurationMs = &duration

	// Set error message
	timeoutDuration := time.Duration(0)
	if execution.TimeoutDuration != nil {
		timeoutDuration = time.Duration(*execution.TimeoutDuration) * time.Second
	}
	errorMsg := "Workflow execution timed out"
	if timeoutDuration > 0 {
		errorMsg = "Workflow execution timed out after " + timeoutDuration.String()
	}
	execution.ErrorMessage = &errorMsg

	if err := w.executionRepo.UpdateExecution(ctx, execution); err != nil {
		return err
	}

	w.logger.Infof("Successfully failed timed-out execution: %s", execution.ExecutionID)
	return nil
}
