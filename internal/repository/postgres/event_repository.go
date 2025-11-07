package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// EventRepository handles event database operations
type EventRepository struct {
	db *sql.DB
}

// NewEventRepository creates a new event repository
func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// CreateEvent creates a new event
func (r *EventRepository) CreateEvent(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO events (
			id, organization_id, event_id, event_type, source, payload,
			triggered_workflows, received_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, received_at`

	err := r.db.QueryRowContext(
		ctx, query,
		event.ID, event.OrganizationID, event.EventID, event.EventType, event.Source,
		event.Payload, pq.Array(event.TriggeredWorkflows),
		event.ReceivedAt, event.ProcessedAt,
	).Scan(&event.ID, &event.ReceivedAt)

	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

// UpdateEvent updates an event
func (r *EventRepository) UpdateEvent(ctx context.Context, organizationID uuid.UUID, event *models.Event) error {
	query := `
		UPDATE events
		SET triggered_workflows = $3,
		    processed_at = $4
		WHERE organization_id = $1 AND id = $2`

	result, err := r.db.ExecContext(
		ctx, query,
		organizationID, event.ID, pq.Array(event.TriggeredWorkflows), event.ProcessedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("event not found")
	}

	return nil
}

// GetEventByID retrieves an event by ID within an organization
func (r *EventRepository) GetEventByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Event, error) {
	event := &models.Event{}
	query := `
		SELECT id, organization_id, event_id, event_type, source, payload,
		       triggered_workflows, received_at, processed_at
		FROM events
		WHERE organization_id = $1 AND id = $2`

	var triggeredWorkflows pq.StringArray
	err := r.db.QueryRowContext(ctx, query, organizationID, id).Scan(
		&event.ID, &event.OrganizationID, &event.EventID, &event.EventType, &event.Source,
		&event.Payload, &triggeredWorkflows, &event.ReceivedAt,
		&event.ProcessedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	event.TriggeredWorkflows = triggeredWorkflows
	return event, nil
}

// ListEvents retrieves events with pagination and filters within an organization
func (r *EventRepository) ListEvents(
	ctx context.Context,
	organizationID uuid.UUID,
	eventType *string,
	processed *bool,
	limit, offset int,
) ([]models.Event, int64, error) {
	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM events
		WHERE organization_id = $1
		  AND ($2::varchar IS NULL OR event_type = $2)
		  AND ($3::boolean IS NULL OR (processed_at IS NOT NULL) = $3)`

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, organizationID, eventType, processed).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	// Get events
	query := `
		SELECT id, organization_id, event_id, event_type, source, payload,
		       triggered_workflows, received_at, processed_at
		FROM events
		WHERE organization_id = $1
		  AND ($2::varchar IS NULL OR event_type = $2)
		  AND ($3::boolean IS NULL OR (processed_at IS NOT NULL) = $3)
		ORDER BY received_at DESC
		LIMIT $4 OFFSET $5`

	rows, err := r.db.QueryContext(ctx, query, organizationID, eventType, processed, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		event := models.Event{}
		var triggeredWorkflows pq.StringArray
		err := rows.Scan(
			&event.ID, &event.OrganizationID, &event.EventID, &event.EventType, &event.Source,
			&event.Payload, &triggeredWorkflows, &event.ReceivedAt,
			&event.ProcessedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}
		event.TriggeredWorkflows = triggeredWorkflows
		events = append(events, event)
	}

	return events, total, nil
}
