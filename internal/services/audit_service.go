package services

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
)

// AuditRepository defines the interface for audit log persistence
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
	GetAuditLogByID(ctx context.Context, organizationID *uuid.UUID, id uuid.UUID) (*models.AuditLog, error)
	ListAuditLogs(ctx context.Context, organizationID *uuid.UUID, filters *postgres.AuditLogFilters, limit, offset int) ([]models.AuditLog, int64, error)
	GetAuditLogsByEntity(ctx context.Context, organizationID *uuid.UUID, entityType string, entityID uuid.UUID, limit, offset int) ([]models.AuditLog, error)
	GetAuditLogsByActor(ctx context.Context, organizationID *uuid.UUID, actorID uuid.UUID, limit, offset int) ([]models.AuditLog, error)
}

// AuditService handles audit logging
type AuditService struct {
	auditRepo AuditRepository
	logger    *logger.Logger
}

// NewAuditService creates a new audit service
func NewAuditService(auditRepo AuditRepository, log *logger.Logger) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
		logger:    log,
	}
}

// LogAction logs an audit event
func (s *AuditService) LogAction(
	ctx context.Context,
	entityType string,
	entityID uuid.UUID,
	action string,
	actorID uuid.UUID,
	actorType string,
	changes map[string]interface{},
) error {
	auditLog := &models.AuditLog{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		ActorID:    actorID,
		ActorType:  actorType,
		Changes:    changes,
		Timestamp:  time.Now(),
	}

	if err := s.auditRepo.CreateAuditLog(ctx, auditLog); err != nil {
		s.logger.Errorf("Failed to create audit log: %v", err)
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	s.logger.Debugf("Audit log created: %s %s on %s/%s by %s", action, entityType, entityType, entityID, actorID)
	return nil
}

// LogWorkflowCreated logs workflow creation
func (s *AuditService) LogWorkflowCreated(
	ctx context.Context,
	workflowID uuid.UUID,
	actorID uuid.UUID,
	actorType string,
	workflowData map[string]interface{},
) error {
	return s.LogAction(ctx, "workflow", workflowID, "created", actorID, actorType, workflowData)
}

// LogWorkflowUpdated logs workflow updates
func (s *AuditService) LogWorkflowUpdated(
	ctx context.Context,
	workflowID uuid.UUID,
	actorID uuid.UUID,
	actorType string,
	changes map[string]interface{},
) error {
	return s.LogAction(ctx, "workflow", workflowID, "updated", actorID, actorType, changes)
}

// LogWorkflowDeleted logs workflow deletion
func (s *AuditService) LogWorkflowDeleted(
	ctx context.Context,
	workflowID uuid.UUID,
	actorID uuid.UUID,
	actorType string,
) error {
	return s.LogAction(ctx, "workflow", workflowID, "deleted", actorID, actorType, map[string]interface{}{})
}

// LogApprovalApproved logs approval decisions
func (s *AuditService) LogApprovalApproved(
	ctx context.Context,
	approvalID uuid.UUID,
	actorID uuid.UUID,
	reason *string,
) error {
	changes := map[string]interface{}{
		"decision": "approved",
	}
	if reason != nil {
		changes["reason"] = *reason
	}
	return s.LogAction(ctx, "approval", approvalID, "approved", actorID, "user", changes)
}

// LogApprovalRejected logs approval rejections
func (s *AuditService) LogApprovalRejected(
	ctx context.Context,
	approvalID uuid.UUID,
	actorID uuid.UUID,
	reason *string,
) error {
	changes := map[string]interface{}{
		"decision": "rejected",
	}
	if reason != nil {
		changes["reason"] = *reason
	}
	return s.LogAction(ctx, "approval", approvalID, "rejected", actorID, "user", changes)
}

// LogExecutionPaused logs execution pause
func (s *AuditService) LogExecutionPaused(
	ctx context.Context,
	executionID uuid.UUID,
	actorID uuid.UUID,
	actorType string,
	reason string,
) error {
	changes := map[string]interface{}{
		"reason": reason,
	}
	return s.LogAction(ctx, "execution", executionID, "paused", actorID, actorType, changes)
}

// LogExecutionResumed logs execution resume
func (s *AuditService) LogExecutionResumed(
	ctx context.Context,
	executionID uuid.UUID,
	actorID uuid.UUID,
	actorType string,
) error {
	return s.LogAction(ctx, "execution", executionID, "resumed", actorID, actorType, map[string]interface{}{})
}

// LogExecutionCancelled logs execution cancellation
func (s *AuditService) LogExecutionCancelled(
	ctx context.Context,
	executionID uuid.UUID,
	actorID uuid.UUID,
	actorType string,
	reason string,
) error {
	changes := map[string]interface{}{
		"reason": reason,
	}
	return s.LogAction(ctx, "execution", executionID, "cancelled", actorID, actorType, changes)
}

// LogAPIKeyCreated logs API key creation
func (s *AuditService) LogAPIKeyCreated(
	ctx context.Context,
	keyID uuid.UUID,
	actorID uuid.UUID,
	keyData map[string]interface{},
) error {
	return s.LogAction(ctx, "api_key", keyID, "created", actorID, "user", keyData)
}

// LogAPIKeyRevoked logs API key revocation
func (s *AuditService) LogAPIKeyRevoked(
	ctx context.Context,
	keyID uuid.UUID,
	actorID uuid.UUID,
) error {
	return s.LogAction(ctx, "api_key", keyID, "revoked", actorID, "user", map[string]interface{}{})
}

// LogScheduleCreated logs schedule creation
func (s *AuditService) LogScheduleCreated(
	ctx context.Context,
	scheduleID uuid.UUID,
	actorID uuid.UUID,
	scheduleData map[string]interface{},
) error {
	return s.LogAction(ctx, "schedule", scheduleID, "created", actorID, "user", scheduleData)
}

// LogScheduleUpdated logs schedule updates
func (s *AuditService) LogScheduleUpdated(
	ctx context.Context,
	scheduleID uuid.UUID,
	actorID uuid.UUID,
	changes map[string]interface{},
) error {
	return s.LogAction(ctx, "schedule", scheduleID, "updated", actorID, "user", changes)
}

// LogScheduleDeleted logs schedule deletion
func (s *AuditService) LogScheduleDeleted(
	ctx context.Context,
	scheduleID uuid.UUID,
	actorID uuid.UUID,
) error {
	return s.LogAction(ctx, "schedule", scheduleID, "deleted", actorID, "user", map[string]interface{}{})
}

// GetAuditLog retrieves an audit log by ID
// Pass organizationID for organization-scoped logs, or uuid.Nil for system-level logs
func (s *AuditService) GetAuditLog(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (*models.AuditLog, error) {
	var orgID *uuid.UUID
	if organizationID != uuid.Nil {
		orgID = &organizationID
	}
	return s.auditRepo.GetAuditLogByID(ctx, orgID, id)
}

// ListAuditLogs retrieves audit logs with filters
// Pass organizationID for organization-scoped logs, or uuid.Nil for system-level logs
func (s *AuditService) ListAuditLogs(
	ctx context.Context,
	organizationID uuid.UUID,
	filters *postgres.AuditLogFilters,
	limit, offset int,
) ([]models.AuditLog, int64, error) {
	var orgID *uuid.UUID
	if organizationID != uuid.Nil {
		orgID = &organizationID
	}
	return s.auditRepo.ListAuditLogs(ctx, orgID, filters, limit, offset)
}

// GetEntityAuditLogs retrieves audit logs for a specific entity
// Pass organizationID for organization-scoped logs, or uuid.Nil for system-level logs
func (s *AuditService) GetEntityAuditLogs(
	ctx context.Context,
	organizationID uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	limit, offset int,
) ([]models.AuditLog, error) {
	var orgID *uuid.UUID
	if organizationID != uuid.Nil {
		orgID = &organizationID
	}
	return s.auditRepo.GetAuditLogsByEntity(ctx, orgID, entityType, entityID, limit, offset)
}

// GetActorAuditLogs retrieves audit logs by actor
// Pass organizationID for organization-scoped logs, or uuid.Nil for system-level logs
func (s *AuditService) GetActorAuditLogs(
	ctx context.Context,
	organizationID uuid.UUID,
	actorID uuid.UUID,
	limit, offset int,
) ([]models.AuditLog, error) {
	var orgID *uuid.UUID
	if organizationID != uuid.Nil {
		orgID = &organizationID
	}
	return s.auditRepo.GetAuditLogsByActor(ctx, orgID, actorID, limit, offset)
}
