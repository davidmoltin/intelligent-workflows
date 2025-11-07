package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// ApprovalRepository handles approval database operations
type ApprovalRepository struct {
	db *sql.DB
}

// NewApprovalRepository creates a new approval repository
func NewApprovalRepository(db *sql.DB) *ApprovalRepository {
	return &ApprovalRepository{db: db}
}

// CreateApproval creates a new approval request
func (r *ApprovalRepository) CreateApproval(ctx context.Context, approval *models.ApprovalRequest) error {
	query := `
		INSERT INTO approval_requests (
			id, organization_id, request_id, execution_id, entity_type, entity_id,
			requester_id, approver_role, approver_id, status, reason,
			decision_reason, requested_at, decided_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, requested_at`

	err := r.db.QueryRowContext(
		ctx, query,
		approval.ID, approval.OrganizationID, approval.RequestID, approval.ExecutionID,
		approval.EntityType, approval.EntityID, approval.RequesterID,
		approval.ApproverRole, approval.ApproverID, approval.Status,
		approval.Reason, approval.DecisionReason, approval.RequestedAt,
		approval.DecidedAt, approval.ExpiresAt,
	).Scan(&approval.ID, &approval.RequestedAt)

	if err != nil {
		return fmt.Errorf("failed to create approval: %w", err)
	}

	return nil
}

// UpdateApproval updates an approval request
func (r *ApprovalRepository) UpdateApproval(ctx context.Context, organizationID uuid.UUID, approval *models.ApprovalRequest) error {
	query := `
		UPDATE approval_requests
		SET status = $3,
		    approver_id = $4,
		    decision_reason = $5,
		    decided_at = $6
		WHERE organization_id = $1 AND id = $2`

	result, err := r.db.ExecContext(
		ctx, query,
		organizationID, approval.ID, approval.Status, approval.ApproverID,
		approval.DecisionReason, approval.DecidedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("approval not found")
	}

	return nil
}

// GetApprovalByID retrieves an approval by ID within an organization
func (r *ApprovalRepository) GetApprovalByID(ctx context.Context, organizationID, id uuid.UUID) (*models.ApprovalRequest, error) {
	approval := &models.ApprovalRequest{}
	query := `
		SELECT id, organization_id, request_id, execution_id, entity_type, entity_id,
		       requester_id, approver_role, approver_id, status, reason,
		       decision_reason, requested_at, decided_at, expires_at
		FROM approval_requests
		WHERE organization_id = $1 AND id = $2`

	err := r.db.QueryRowContext(ctx, query, organizationID, id).Scan(
		&approval.ID, &approval.OrganizationID, &approval.RequestID, &approval.ExecutionID,
		&approval.EntityType, &approval.EntityID, &approval.RequesterID,
		&approval.ApproverRole, &approval.ApproverID, &approval.Status,
		&approval.Reason, &approval.DecisionReason, &approval.RequestedAt,
		&approval.DecidedAt, &approval.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("approval not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get approval: %w", err)
	}

	return approval, nil
}

// GetApprovalByRequestID retrieves an approval by request_id string within an organization
func (r *ApprovalRepository) GetApprovalByRequestID(ctx context.Context, organizationID uuid.UUID, requestID string) (*models.ApprovalRequest, error) {
	approval := &models.ApprovalRequest{}
	query := `
		SELECT id, organization_id, request_id, execution_id, entity_type, entity_id,
		       requester_id, approver_role, approver_id, status, reason,
		       decision_reason, requested_at, decided_at, expires_at
		FROM approval_requests
		WHERE organization_id = $1 AND request_id = $2`

	err := r.db.QueryRowContext(ctx, query, organizationID, requestID).Scan(
		&approval.ID, &approval.OrganizationID, &approval.RequestID, &approval.ExecutionID,
		&approval.EntityType, &approval.EntityID, &approval.RequesterID,
		&approval.ApproverRole, &approval.ApproverID, &approval.Status,
		&approval.Reason, &approval.DecisionReason, &approval.RequestedAt,
		&approval.DecidedAt, &approval.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("approval not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get approval: %w", err)
	}

	return approval, nil
}

// ListApprovals retrieves approvals with pagination and filters within an organization
func (r *ApprovalRepository) ListApprovals(
	ctx context.Context,
	organizationID uuid.UUID,
	status *models.ApprovalStatus,
	approverID *uuid.UUID,
	limit, offset int,
) ([]models.ApprovalRequest, int64, error) {
	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM approval_requests
		WHERE organization_id = $1
		  AND ($2::varchar IS NULL OR status = $2)
		  AND ($3::uuid IS NULL OR approver_id = $3)`

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, organizationID, status, approverID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count approvals: %w", err)
	}

	// Get approvals
	query := `
		SELECT id, organization_id, request_id, execution_id, entity_type, entity_id,
		       requester_id, approver_role, approver_id, status, reason,
		       decision_reason, requested_at, decided_at, expires_at
		FROM approval_requests
		WHERE organization_id = $1
		  AND ($2::varchar IS NULL OR status = $2)
		  AND ($3::uuid IS NULL OR approver_id = $3)
		ORDER BY requested_at DESC
		LIMIT $4 OFFSET $5`

	rows, err := r.db.QueryContext(ctx, query, organizationID, status, approverID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list approvals: %w", err)
	}
	defer rows.Close()

	var approvals []models.ApprovalRequest
	for rows.Next() {
		approval := models.ApprovalRequest{}
		err := rows.Scan(
			&approval.ID, &approval.OrganizationID, &approval.RequestID, &approval.ExecutionID,
			&approval.EntityType, &approval.EntityID, &approval.RequesterID,
			&approval.ApproverRole, &approval.ApproverID, &approval.Status,
			&approval.Reason, &approval.DecisionReason, &approval.RequestedAt,
			&approval.DecidedAt, &approval.ExpiresAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan approval: %w", err)
		}
		approvals = append(approvals, approval)
	}

	return approvals, total, nil
}

// GetApprovalsByExecution retrieves all approvals for an execution within an organization
func (r *ApprovalRepository) GetApprovalsByExecution(ctx context.Context, organizationID, executionID uuid.UUID) ([]models.ApprovalRequest, error) {
	query := `
		SELECT id, organization_id, request_id, execution_id, entity_type, entity_id,
		       requester_id, approver_role, approver_id, status, reason,
		       decision_reason, requested_at, decided_at, expires_at
		FROM approval_requests
		WHERE organization_id = $1 AND execution_id = $2
		ORDER BY requested_at DESC`

	rows, err := r.db.QueryContext(ctx, query, organizationID, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approvals: %w", err)
	}
	defer rows.Close()

	var approvals []models.ApprovalRequest
	for rows.Next() {
		approval := models.ApprovalRequest{}
		err := rows.Scan(
			&approval.ID, &approval.OrganizationID, &approval.RequestID, &approval.ExecutionID,
			&approval.EntityType, &approval.EntityID, &approval.RequesterID,
			&approval.ApproverRole, &approval.ApproverID, &approval.Status,
			&approval.Reason, &approval.DecisionReason, &approval.RequestedAt,
			&approval.DecidedAt, &approval.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval: %w", err)
		}
		approvals = append(approvals, approval)
	}

	return approvals, nil
}

// GetExpiredApprovals retrieves approval requests that have expired within an organization
func (r *ApprovalRepository) GetExpiredApprovals(ctx context.Context, organizationID uuid.UUID, limit int) ([]models.ApprovalRequest, error) {
	query := `
		SELECT id, organization_id, request_id, execution_id, entity_type, entity_id,
		       requester_id, approver_role, approver_id, status, reason,
		       decision_reason, requested_at, decided_at, expires_at
		FROM approval_requests
		WHERE organization_id = $1
		  AND status = $2
		  AND expires_at IS NOT NULL
		  AND expires_at < NOW()
		ORDER BY expires_at ASC
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, organizationID, models.ApprovalStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired approvals: %w", err)
	}
	defer rows.Close()

	var approvals []models.ApprovalRequest
	for rows.Next() {
		approval := models.ApprovalRequest{}
		err := rows.Scan(
			&approval.ID, &approval.OrganizationID, &approval.RequestID, &approval.ExecutionID,
			&approval.EntityType, &approval.EntityID, &approval.RequesterID,
			&approval.ApproverRole, &approval.ApproverID, &approval.Status,
			&approval.Reason, &approval.DecisionReason, &approval.RequestedAt,
			&approval.DecidedAt, &approval.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval: %w", err)
		}
		approvals = append(approvals, approval)
	}

	return approvals, nil
}
