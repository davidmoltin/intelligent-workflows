package workers

import (
	"context"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// WorkflowResumerWorker handles periodic resumption of paused workflows
type WorkflowResumerWorker struct {
	workflowResumer *services.WorkflowResumerImpl
	logger          *logger.Logger
	checkInterval   time.Duration
	batchSize       int
	stopCh          chan struct{}
	doneCh          chan struct{}
}

// NewWorkflowResumerWorker creates a new workflow resumer worker
func NewWorkflowResumerWorker(
	workflowResumer *services.WorkflowResumerImpl,
	logger *logger.Logger,
	checkInterval time.Duration,
) *WorkflowResumerWorker {
	if checkInterval == 0 {
		checkInterval = 1 * time.Minute // Default to 1 minute
	}

	return &WorkflowResumerWorker{
		workflowResumer: workflowResumer,
		logger:          logger,
		checkInterval:   checkInterval,
		batchSize:       50, // Process up to 50 paused executions per check
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}
}

// Start starts the worker in the background
func (w *WorkflowResumerWorker) Start(ctx context.Context) {
	w.logger.Info("Starting workflow resumer worker",
		logger.String("interval", w.checkInterval.String()),
		logger.Int("batch_size", w.batchSize),
	)

	go w.run(ctx)
}

// Stop stops the worker gracefully
func (w *WorkflowResumerWorker) Stop() {
	w.logger.Info("Stopping workflow resumer worker")
	close(w.stopCh)
	<-w.doneCh
	w.logger.Info("Workflow resumer worker stopped")
}

// run is the main worker loop
func (w *WorkflowResumerWorker) run(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processPausedExecutions(ctx)

	for {
		select {
		case <-ticker.C:
			w.processPausedExecutions(ctx)
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processPausedExecutions checks for paused executions and resumes those ready
func (w *WorkflowResumerWorker) processPausedExecutions(ctx context.Context) {
	w.logger.Debug("Checking for paused executions ready to resume")

	// Get paused executions
	executions, err := w.workflowResumer.GetPausedExecutions(ctx, w.batchSize)
	if err != nil {
		w.logger.Errorf("Failed to get paused executions: %v", err)
		return
	}

	if len(executions) == 0 {
		w.logger.Debug("No paused executions found")
		return
	}

	w.logger.Infof("Found %d paused executions to process", len(executions))

	resumedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, execution := range executions {
		// Check if execution has approval decision in resume_data
		if execution.ResumeData != nil {
			if approved, exists := execution.ResumeData["approved"]; exists {
				// This execution has an approval decision, try to resume it
				approvedBool, ok := approved.(bool)
				if !ok {
					w.logger.Warnf("Invalid approval decision type for execution %s", execution.ID)
					errorCount++
					continue
				}

				w.logger.Infof("Auto-resuming execution %s (approved: %v)", execution.ID, approvedBool)

				if err := w.workflowResumer.ResumeWorkflow(ctx, execution.ID, approvedBool); err != nil {
					w.logger.Errorf("Failed to resume execution %s: %v", execution.ID, err)
					errorCount++
					continue
				}

				resumedCount++
				continue
			}
		}

		// Check if execution has been paused for too long (warn but don't auto-resume)
		if execution.PausedAt != nil {
			pauseDuration := time.Since(*execution.PausedAt)
			if pauseDuration > 24*time.Hour {
				w.logger.Warnf(
					"Execution %s has been paused for %v (reason: %s) - may need manual intervention",
					execution.ID,
					pauseDuration.Round(time.Hour),
					stringOrEmpty(execution.PausedReason),
				)
			}
		}

		skippedCount++
	}

	w.logger.Infof(
		"Paused executions processed: resumed=%d, skipped=%d, errors=%d",
		resumedCount,
		skippedCount,
		errorCount,
	)
}

// stringOrEmpty safely extracts string from pointer
func stringOrEmpty(s *string) string {
	if s == nil {
		return "none"
	}
	return *s
}
