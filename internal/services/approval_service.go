package services

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
)

// ApprovalRepository defines the interface for approval persistence
type ApprovalRepository interface {
	CreateApproval(ctx context.Context, approval *models.ApprovalRequest) error
	UpdateApproval(ctx context.Context, approval *models.ApprovalRequest) error
	GetApprovalByID(ctx context.Context, id uuid.UUID) (*models.ApprovalRequest, error)
	GetApprovalByRequestID(ctx context.Context, requestID string) (*models.ApprovalRequest, error)
	ListApprovals(ctx context.Context, status *models.ApprovalStatus, approverID *uuid.UUID, limit, offset int) ([]models.ApprovalRequest, int64, error)
	GetApprovalsByExecution(ctx context.Context, executionID uuid.UUID) ([]models.ApprovalRequest, error)
}

// WorkflowResumer defines interface for resuming workflows
type WorkflowResumer interface {
	ResumeWorkflow(ctx context.Context, executionID uuid.UUID, approved bool) error
}

// ApprovalService handles approval workflow logic
type ApprovalService struct {
	approvalRepo         ApprovalRepository
	logger               *logger.Logger
	notificationSvc      *NotificationService
	workflowResumer      WorkflowResumer
	defaultApproverEmail string
}

// NewApprovalService creates a new approval service
func NewApprovalService(
	approvalRepo ApprovalRepository,
	log *logger.Logger,
	notificationSvc *NotificationService,
	workflowResumer WorkflowResumer,
) *ApprovalService {
	return &ApprovalService{
		approvalRepo:         approvalRepo,
		logger:               log,
		notificationSvc:      notificationSvc,
		workflowResumer:      workflowResumer,
		defaultApproverEmail: "approver@example.com", // TODO: Make this configurable
	}
}

// CreateApprovalRequest creates a new approval request
func (s *ApprovalService) CreateApprovalRequest(
	ctx context.Context,
	executionID uuid.UUID,
	entityType string,
	entityID string,
	requesterID *uuid.UUID,
	approverRole string,
	reason string,
	expiresIn *time.Duration,
) (*models.ApprovalRequest, error) {
	s.logger.Infof("Creating approval request for %s/%s", entityType, entityID)

	approval := &models.ApprovalRequest{
		ID:           uuid.New(),
		RequestID:    fmt.Sprintf("appr_%s", uuid.New().String()[:8]),
		ExecutionID:  executionID,
		EntityType:   entityType,
		EntityID:     entityID,
		RequesterID:  requesterID,
		ApproverRole: approverRole,
		Status:       models.ApprovalStatusPending,
		Reason:       &reason,
		RequestedAt:  time.Now(),
	}

	// Set expiration if provided
	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		approval.ExpiresAt = &expiresAt
	}

	if err := s.approvalRepo.CreateApproval(ctx, approval); err != nil {
		return nil, fmt.Errorf("failed to create approval: %w", err)
	}

	s.logger.Infof("Approval request created: %s", approval.RequestID)

	// Send notification to approver(s)
	if s.notificationSvc != nil {
		// In a real implementation, we would look up the approver email based on the role
		// For now, we'll use a default email or configured approver
		if err := s.notificationSvc.SendApprovalRequestNotification(ctx, approval, s.defaultApproverEmail); err != nil {
			// Log error but don't fail the approval creation
			s.logger.Errorf("Failed to send approval notification: %v", err)
		}
	}

	return approval, nil
}

// ApproveRequest approves an approval request
func (s *ApprovalService) ApproveRequest(
	ctx context.Context,
	approvalID uuid.UUID,
	approverID uuid.UUID,
	reason *string,
) (*models.ApprovalRequest, error) {
	s.logger.Infof("Approving request: %s by approver: %s", approvalID, approverID)

	// Get approval
	approval, err := s.approvalRepo.GetApprovalByID(ctx, approvalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval: %w", err)
	}

	// Validate status
	if approval.Status != models.ApprovalStatusPending {
		return nil, fmt.Errorf("approval is not pending (status: %s)", approval.Status)
	}

	// Check if expired
	if approval.ExpiresAt != nil && time.Now().After(*approval.ExpiresAt) {
		return nil, fmt.Errorf("approval has expired")
	}

	// Update approval
	approval.Status = models.ApprovalStatusApproved
	approval.ApproverID = &approverID
	approval.DecisionReason = reason
	now := time.Now()
	approval.DecidedAt = &now

	if err := s.approvalRepo.UpdateApproval(ctx, approval); err != nil {
		return nil, fmt.Errorf("failed to update approval: %w", err)
	}

	s.logger.Infof("Approval request approved: %s", approval.RequestID)

	// Resume workflow execution
	if s.workflowResumer != nil {
		if err := s.workflowResumer.ResumeWorkflow(ctx, approval.ExecutionID, true); err != nil {
			s.logger.Errorf("Failed to resume workflow after approval: %v", err)
			// Note: Approval is already saved, so we return the approval but log the error
		}
	}

	// Send notification about approval decision
	if s.notificationSvc != nil {
		requesterEmail := s.defaultApproverEmail // In real implementation, look up requester email
		if err := s.notificationSvc.SendApprovalDecisionNotification(ctx, approval, requesterEmail); err != nil {
			s.logger.Errorf("Failed to send approval decision notification: %v", err)
		}
	}

	return approval, nil
}

// RejectRequest rejects an approval request
func (s *ApprovalService) RejectRequest(
	ctx context.Context,
	approvalID uuid.UUID,
	approverID uuid.UUID,
	reason *string,
) (*models.ApprovalRequest, error) {
	s.logger.Infof("Rejecting request: %s by approver: %s", approvalID, approverID)

	// Get approval
	approval, err := s.approvalRepo.GetApprovalByID(ctx, approvalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval: %w", err)
	}

	// Validate status
	if approval.Status != models.ApprovalStatusPending {
		return nil, fmt.Errorf("approval is not pending (status: %s)", approval.Status)
	}

	// Check if expired
	if approval.ExpiresAt != nil && time.Now().After(*approval.ExpiresAt) {
		return nil, fmt.Errorf("approval has expired")
	}

	// Update approval
	approval.Status = models.ApprovalStatusRejected
	approval.ApproverID = &approverID
	approval.DecisionReason = reason
	now := time.Now()
	approval.DecidedAt = &now

	if err := s.approvalRepo.UpdateApproval(ctx, approval); err != nil {
		return nil, fmt.Errorf("failed to update approval: %w", err)
	}

	s.logger.Infof("Approval request rejected: %s", approval.RequestID)

	// Resume workflow execution with rejection status
	if s.workflowResumer != nil {
		if err := s.workflowResumer.ResumeWorkflow(ctx, approval.ExecutionID, false); err != nil {
			s.logger.Errorf("Failed to resume workflow after rejection: %v", err)
			// Note: Approval is already saved, so we return the approval but log the error
		}
	}

	// Send notification about rejection decision
	if s.notificationSvc != nil {
		requesterEmail := s.defaultApproverEmail // In real implementation, look up requester email
		if err := s.notificationSvc.SendApprovalDecisionNotification(ctx, approval, requesterEmail); err != nil {
			s.logger.Errorf("Failed to send rejection notification: %v", err)
		}
	}

	return approval, nil
}

// GetApproval retrieves an approval by ID
func (s *ApprovalService) GetApproval(ctx context.Context, approvalID uuid.UUID) (*models.ApprovalRequest, error) {
	return s.approvalRepo.GetApprovalByID(ctx, approvalID)
}

// ListPendingApprovals retrieves pending approvals for an approver
func (s *ApprovalService) ListPendingApprovals(
	ctx context.Context,
	approverID *uuid.UUID,
	limit, offset int,
) ([]models.ApprovalRequest, int64, error) {
	status := models.ApprovalStatusPending
	return s.approvalRepo.ListApprovals(ctx, &status, approverID, limit, offset)
}

// ListApprovals retrieves approvals with filters
func (s *ApprovalService) ListApprovals(
	ctx context.Context,
	status *models.ApprovalStatus,
	approverID *uuid.UUID,
	limit, offset int,
) ([]models.ApprovalRequest, int64, error) {
	return s.approvalRepo.ListApprovals(ctx, status, approverID, limit, offset)
}

// ExpireOldApprovals marks expired approvals as expired
func (s *ApprovalService) ExpireOldApprovals(ctx context.Context) error {
	s.logger.Infof("Checking for expired approvals")

	// This would be run periodically by a background worker
	// Get all pending approvals
	status := models.ApprovalStatusPending
	approvals, _, err := s.approvalRepo.ListApprovals(ctx, &status, nil, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to list approvals: %w", err)
	}

	now := time.Now()
	expiredCount := 0

	for _, approval := range approvals {
		if approval.ExpiresAt != nil && now.After(*approval.ExpiresAt) {
			// Mark as expired
			approval.Status = models.ApprovalStatusExpired
			approval.DecidedAt = &now

			if err := s.approvalRepo.UpdateApproval(ctx, &approval); err != nil {
				s.logger.Errorf("Failed to expire approval %s: %v", approval.RequestID, err)
				continue
			}

			expiredCount++
			s.logger.Infof("Approval expired: %s", approval.RequestID)

			// Send expiration notification
			if s.notificationSvc != nil {
				requesterEmail := s.defaultApproverEmail // In real implementation, look up requester email
				if err := s.notificationSvc.SendApprovalDecisionNotification(ctx, &approval, requesterEmail); err != nil {
					s.logger.Errorf("Failed to send expiration notification for %s: %v", approval.RequestID, err)
				}
			}
		}
	}

	if expiredCount > 0 {
		s.logger.Infof("Expired %d approvals", expiredCount)
	}

	return nil
}
