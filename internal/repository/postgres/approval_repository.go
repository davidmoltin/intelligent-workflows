package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
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
			id, request_id, execution_id, entity_type, entity_id,
			requester_id, approver_role, approver_id, status, reason,
			decision_reason, requested_at, decided_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, requested_at`

	err := r.db.QueryRowContext(
		ctx, query,
		approval.ID, approval.RequestID, approval.ExecutionID,
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
func (r *ApprovalRepository) UpdateApproval(ctx context.Context, approval *models.ApprovalRequest) error {
	query := `
		UPDATE approval_requests
		SET status = $2,
		    approver_id = $3,
		    decision_reason = $4,
		    decided_at = $5
		WHERE id = $1`

	result, err := r.db.ExecContext(
		ctx, query,
		approval.ID, approval.Status, approval.ApproverID,
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

// GetApprovalByID retrieves an approval by ID
func (r *ApprovalRepository) GetApprovalByID(ctx context.Context, id uuid.UUID) (*models.ApprovalRequest, error) {
	approval := &models.ApprovalRequest{}
	query := `
		SELECT id, request_id, execution_id, entity_type, entity_id,
		       requester_id, approver_role, approver_id, status, reason,
		       decision_reason, requested_at, decided_at, expires_at
		FROM approval_requests
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&approval.ID, &approval.RequestID, &approval.ExecutionID,
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

// GetApprovalByRequestID retrieves an approval by request_id string
func (r *ApprovalRepository) GetApprovalByRequestID(ctx context.Context, requestID string) (*models.ApprovalRequest, error) {
	approval := &models.ApprovalRequest{}
	query := `
		SELECT id, request_id, execution_id, entity_type, entity_id,
		       requester_id, approver_role, approver_id, status, reason,
		       decision_reason, requested_at, decided_at, expires_at
		FROM approval_requests
		WHERE request_id = $1`

	err := r.db.QueryRowContext(ctx, query, requestID).Scan(
		&approval.ID, &approval.RequestID, &approval.ExecutionID,
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

// ListApprovals retrieves approvals with pagination and filters
func (r *ApprovalRepository) ListApprovals(
	ctx context.Context,
	status *models.ApprovalStatus,
	approverID *uuid.UUID,
	limit, offset int,
) ([]models.ApprovalRequest, int64, error) {
	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM approval_requests
		WHERE ($1::varchar IS NULL OR status = $1)
		  AND ($2::uuid IS NULL OR approver_id = $2)`

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, status, approverID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count approvals: %w", err)
	}

	// Get approvals
	query := `
		SELECT id, request_id, execution_id, entity_type, entity_id,
		       requester_id, approver_role, approver_id, status, reason,
		       decision_reason, requested_at, decided_at, expires_at
		FROM approval_requests
		WHERE ($1::varchar IS NULL OR status = $1)
		  AND ($2::uuid IS NULL OR approver_id = $2)
		ORDER BY requested_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, status, approverID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list approvals: %w", err)
	}
	defer rows.Close()

	var approvals []models.ApprovalRequest
	for rows.Next() {
		approval := models.ApprovalRequest{}
		err := rows.Scan(
			&approval.ID, &approval.RequestID, &approval.ExecutionID,
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

// GetApprovalsByExecution retrieves all approvals for an execution
func (r *ApprovalRepository) GetApprovalsByExecution(ctx context.Context, executionID uuid.UUID) ([]models.ApprovalRequest, error) {
	query := `
		SELECT id, request_id, execution_id, entity_type, entity_id,
		       requester_id, approver_role, approver_id, status, reason,
		       decision_reason, requested_at, decided_at, expires_at
		FROM approval_requests
		WHERE execution_id = $1
		ORDER BY requested_at DESC`

	rows, err := r.db.QueryContext(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approvals: %w", err)
	}
	defer rows.Close()

	var approvals []models.ApprovalRequest
	for rows.Next() {
		approval := models.ApprovalRequest{}
		err := rows.Scan(
			&approval.ID, &approval.RequestID, &approval.ExecutionID,
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
