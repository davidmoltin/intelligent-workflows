package workers

import (
	"context"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
)

// ScheduleService defines the interface for schedule operations
type ScheduleService interface {
	GetDueSchedules(ctx context.Context) ([]*models.WorkflowSchedule, error)
	MarkTriggered(ctx context.Context, id uuid.UUID) error
}

// WorkflowTrigger defines the interface for triggering workflows
type WorkflowTrigger interface {
	TriggerWorkflowManually(ctx context.Context, workflowID uuid.UUID, payload map[string]interface{}) (*models.WorkflowExecution, error)
}

// SchedulerWorker handles periodic checking and triggering of scheduled workflows
type SchedulerWorker struct {
	scheduleService ScheduleService
	workflowTrigger WorkflowTrigger
	logger          *logger.Logger
	checkInterval   time.Duration
	stopCh          chan struct{}
	doneCh          chan struct{}
}

// NewSchedulerWorker creates a new scheduler worker
func NewSchedulerWorker(
	scheduleService ScheduleService,
	workflowTrigger WorkflowTrigger,
	logger *logger.Logger,
	checkInterval time.Duration,
) *SchedulerWorker {
	if checkInterval == 0 {
		checkInterval = 1 * time.Minute // Default to 1 minute
	}

	return &SchedulerWorker{
		scheduleService: scheduleService,
		workflowTrigger: workflowTrigger,
		logger:          logger,
		checkInterval:   checkInterval,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}
}

// Start starts the scheduler worker in the background
func (w *SchedulerWorker) Start(ctx context.Context) {
	w.logger.Info("Starting scheduler worker",
		logger.String("interval", w.checkInterval.String()),
	)

	go w.run(ctx)
}

// Stop stops the scheduler worker gracefully
func (w *SchedulerWorker) Stop() {
	w.logger.Info("Stopping scheduler worker")
	close(w.stopCh)
	<-w.doneCh
	w.logger.Info("Scheduler worker stopped")
}

// run is the main worker loop
func (w *SchedulerWorker) run(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processDueSchedules(ctx)

	for {
		select {
		case <-ticker.C:
			w.processDueSchedules(ctx)
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processDueSchedules checks for due schedules and triggers their workflows
func (w *SchedulerWorker) processDueSchedules(ctx context.Context) {
	w.logger.Debug("Checking for due schedules")

	// Get due schedules
	schedules, err := w.scheduleService.GetDueSchedules(ctx)
	if err != nil {
		w.logger.Errorf("Failed to get due schedules: %v", err)
		return
	}

	if len(schedules) == 0 {
		w.logger.Debug("No due schedules found")
		return
	}

	w.logger.Infof("Found %d due schedules to process", len(schedules))

	triggeredCount := 0
	errorCount := 0

	for _, schedule := range schedules {
		// Create payload with schedule context
		payload := map[string]interface{}{
			"schedule_id": schedule.ID.String(),
			"trigger_type": "schedule",
			"cron_expression": schedule.CronExpression,
			"timezone": schedule.Timezone,
		}

		w.logger.Infof(
			"Triggering scheduled workflow: workflow_id=%s, schedule_id=%s, cron=%s",
			schedule.WorkflowID,
			schedule.ID,
			schedule.CronExpression,
		)

		// Trigger the workflow
		execution, err := w.workflowTrigger.TriggerWorkflowManually(ctx, schedule.WorkflowID, payload)
		if err != nil {
			w.logger.Errorf(
				"Failed to trigger scheduled workflow %s (schedule %s): %v",
				schedule.WorkflowID,
				schedule.ID,
				err,
			)
			errorCount++
			continue
		}

		w.logger.Infof(
			"Successfully triggered scheduled workflow: workflow_id=%s, execution_id=%s",
			schedule.WorkflowID,
			execution.ID,
		)

		// Mark schedule as triggered (this also calculates next run time)
		if err := w.scheduleService.MarkTriggered(ctx, schedule.ID); err != nil {
			w.logger.Errorf(
				"Failed to mark schedule %s as triggered: %v",
				schedule.ID,
				err,
			)
			errorCount++
			continue
		}

		triggeredCount++
	}

	w.logger.Infof(
		"Scheduled workflows processed: triggered=%d, errors=%d",
		triggeredCount,
		errorCount,
	)
}
