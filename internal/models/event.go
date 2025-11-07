package models

import (
	"time"

	"github.com/google/uuid"
)

// Event represents an event that triggers workflows
type Event struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	OrganizationID     uuid.UUID  `json:"organization_id" db:"organization_id"`
	EventID            string     `json:"event_id" db:"event_id"`
	EventType          string     `json:"event_type" db:"event_type"`
	Source             string     `json:"source" db:"source"`
	Payload            JSONB      `json:"payload" db:"payload"`
	TriggeredWorkflows []string   `json:"triggered_workflows,omitempty" db:"triggered_workflows"`
	ReceivedAt         time.Time  `json:"received_at" db:"received_at"`
	ProcessedAt        *time.Time `json:"processed_at,omitempty" db:"processed_at"`
}

// CreateEventRequest represents the request to emit an event
type CreateEventRequest struct {
	EventType string                 `json:"event_type" validate:"required"`
	Source    string                 `json:"source"`
	Payload   map[string]interface{} `json:"payload" validate:"required"`
}

// ApprovalStatus represents the status of an approval request
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

// ApprovalRequest represents an approval request
type ApprovalRequest struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	OrganizationID uuid.UUID      `json:"organization_id" db:"organization_id"`
	RequestID      string         `json:"request_id" db:"request_id"`
	ExecutionID    uuid.UUID      `json:"execution_id" db:"execution_id"`
	EntityType     string         `json:"entity_type" db:"entity_type"`
	EntityID       string         `json:"entity_id" db:"entity_id"`
	RequesterID    *uuid.UUID     `json:"requester_id,omitempty" db:"requester_id"`
	ApproverRole   string         `json:"approver_role" db:"approver_role"`
	ApproverID     *uuid.UUID     `json:"approver_id,omitempty" db:"approver_id"`
	Status         ApprovalStatus `json:"status" db:"status"`
	Reason         *string        `json:"reason,omitempty" db:"reason"`
	DecisionReason *string        `json:"decision_reason,omitempty" db:"decision_reason"`
	RequestedAt    time.Time      `json:"requested_at" db:"requested_at"`
	DecidedAt      *time.Time     `json:"decided_at,omitempty" db:"decided_at"`
	ExpiresAt      *time.Time     `json:"expires_at,omitempty" db:"expires_at"`
}

// ApprovalDecisionRequest represents the request to approve/reject
type ApprovalDecisionRequest struct {
	Decision string  `json:"decision" validate:"required,oneof=approve reject"`
	Reason   *string `json:"reason,omitempty"`
}

// ContextCache represents cached context data
type ContextCache struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID uuid.UUID  `json:"organization_id" db:"organization_id"`
	CacheKey       string     `json:"cache_key" db:"cache_key"`
	EntityType     string     `json:"entity_type" db:"entity_type"`
	EntityID       string     `json:"entity_id" db:"entity_id"`
	Data           JSONB      `json:"data" db:"data"`
	CachedAt       time.Time  `json:"cached_at" db:"cached_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID *uuid.UUID `json:"organization_id,omitempty" db:"organization_id"` // Nullable for system-level audits
	EntityType     string     `json:"entity_type" db:"entity_type"`
	EntityID       uuid.UUID  `json:"entity_id" db:"entity_id"`
	Action         string     `json:"action" db:"action"`
	ActorID        uuid.UUID  `json:"actor_id" db:"actor_id"`
	ActorType      string     `json:"actor_type" db:"actor_type"` // user, ai_agent, system
	Changes        JSONB      `json:"changes" db:"changes"`
	Timestamp      time.Time  `json:"timestamp" db:"timestamp"`
}
