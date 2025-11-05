package workers

import (
	"context"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// ApprovalExpirationWorker handles periodic approval expiration checks
type ApprovalExpirationWorker struct {
	approvalService *services.ApprovalService
	logger          *logger.Logger
	checkInterval   time.Duration
	stopCh          chan struct{}
	doneCh          chan struct{}
}

// NewApprovalExpirationWorker creates a new approval expiration worker
func NewApprovalExpirationWorker(
	approvalService *services.ApprovalService,
	logger *logger.Logger,
	checkInterval time.Duration,
) *ApprovalExpirationWorker {
	if checkInterval == 0 {
		checkInterval = 5 * time.Minute // Default to 5 minutes
	}

	return &ApprovalExpirationWorker{
		approvalService: approvalService,
		logger:          logger,
		checkInterval:   checkInterval,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}
}

// Start starts the worker in the background
func (w *ApprovalExpirationWorker) Start(ctx context.Context) {
	w.logger.Info("Starting approval expiration worker",
		logger.String("interval", w.checkInterval.String()),
	)

	go w.run(ctx)
}

// Stop stops the worker gracefully
func (w *ApprovalExpirationWorker) Stop() {
	w.logger.Info("Stopping approval expiration worker")
	close(w.stopCh)
	<-w.doneCh
	w.logger.Info("Approval expiration worker stopped")
}

// run is the main worker loop
func (w *ApprovalExpirationWorker) run(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.checkExpiredApprovals(ctx)

	for {
		select {
		case <-ticker.C:
			w.checkExpiredApprovals(ctx)
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkExpiredApprovals checks and expires old approvals
func (w *ApprovalExpirationWorker) checkExpiredApprovals(ctx context.Context) {
	w.logger.Debug("Checking for expired approvals")

	if err := w.approvalService.ExpireOldApprovals(ctx); err != nil {
		w.logger.Errorf("Failed to expire approvals: %v", err)
		return
	}

	w.logger.Debug("Expired approvals check completed")
}
