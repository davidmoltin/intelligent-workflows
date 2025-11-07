package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// AuditRepository handles audit log database operations
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// CreateAuditLog creates a new audit log entry
func (r *AuditRepository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO audit_log (
			id, organization_id, entity_type, entity_id, action, actor_id, actor_type, changes, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, timestamp`

	err := r.db.QueryRowContext(
		ctx, query,
		log.ID, log.OrganizationID, log.EntityType, log.EntityID, log.Action,
		log.ActorID, log.ActorType, log.Changes, log.Timestamp,
	).Scan(&log.ID, &log.Timestamp)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetAuditLogByID retrieves an audit log by ID within an organization
// Pass nil for organizationID to query system-level audit logs
func (r *AuditRepository) GetAuditLogByID(ctx context.Context, organizationID *uuid.UUID, id uuid.UUID) (*models.AuditLog, error) {
	log := &models.AuditLog{}
	var query string
	var err error

	if organizationID == nil {
		// Query system-level audit logs (organization_id IS NULL)
		query = `
			SELECT id, organization_id, entity_type, entity_id, action, actor_id, actor_type, changes, timestamp
			FROM audit_log
			WHERE id = $1 AND organization_id IS NULL`
		err = r.db.QueryRowContext(ctx, query, id).Scan(
			&log.ID, &log.OrganizationID, &log.EntityType, &log.EntityID, &log.Action,
			&log.ActorID, &log.ActorType, &log.Changes, &log.Timestamp,
		)
	} else {
		// Query organization-scoped audit logs
		query = `
			SELECT id, organization_id, entity_type, entity_id, action, actor_id, actor_type, changes, timestamp
			FROM audit_log
			WHERE organization_id = $1 AND id = $2`
		err = r.db.QueryRowContext(ctx, query, organizationID, id).Scan(
			&log.ID, &log.OrganizationID, &log.EntityType, &log.EntityID, &log.Action,
			&log.ActorID, &log.ActorType, &log.Changes, &log.Timestamp,
		)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("audit log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	return log, nil
}

// AuditLogFilters represents filters for audit log queries
type AuditLogFilters struct {
	EntityType *string
	EntityID   *uuid.UUID
	Action     *string
	ActorID    *uuid.UUID
	ActorType  *string
	StartTime  *time.Time
	EndTime    *time.Time
}

// ListAuditLogs retrieves audit logs with pagination and filters within an organization
// Pass nil for organizationID to query system-level audit logs
func (r *AuditRepository) ListAuditLogs(
	ctx context.Context,
	organizationID *uuid.UUID,
	filters *AuditLogFilters,
	limit, offset int,
) ([]models.AuditLog, int64, error) {
	// Build WHERE clause
	whereClauses := []string{}
	args := []interface{}{}
	argPos := 1

	// Add organization filter
	if organizationID == nil {
		whereClauses = append(whereClauses, "organization_id IS NULL")
	} else {
		whereClauses = append(whereClauses, fmt.Sprintf("organization_id = $%d", argPos))
		args = append(args, *organizationID)
		argPos++
	}

	if filters != nil {
		if filters.EntityType != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("entity_type = $%d", argPos))
			args = append(args, *filters.EntityType)
			argPos++
		}
		if filters.EntityID != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("entity_id = $%d", argPos))
			args = append(args, *filters.EntityID)
			argPos++
		}
		if filters.Action != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("action = $%d", argPos))
			args = append(args, *filters.Action)
			argPos++
		}
		if filters.ActorID != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("actor_id = $%d", argPos))
			args = append(args, *filters.ActorID)
			argPos++
		}
		if filters.ActorType != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("actor_type = $%d", argPos))
			args = append(args, *filters.ActorType)
			argPos++
		}
		if filters.StartTime != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= $%d", argPos))
			args = append(args, *filters.StartTime)
			argPos++
		}
		if filters.EndTime != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("timestamp <= $%d", argPos))
			args = append(args, *filters.EndTime)
			argPos++
		}
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", whereClause)
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get audit logs
	query := fmt.Sprintf(`
		SELECT id, organization_id, entity_type, entity_id, action, actor_id, actor_type, changes, timestamp
		FROM audit_log
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d`, whereClause, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		log := models.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.OrganizationID, &log.EntityType, &log.EntityID, &log.Action,
			&log.ActorID, &log.ActorType, &log.Changes, &log.Timestamp,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

// GetAuditLogsByEntity retrieves all audit logs for a specific entity within an organization
// Pass nil for organizationID to query system-level audit logs
func (r *AuditRepository) GetAuditLogsByEntity(
	ctx context.Context,
	organizationID *uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	limit, offset int,
) ([]models.AuditLog, error) {
	var query string
	var rows *sql.Rows
	var err error

	if organizationID == nil {
		query = `
			SELECT id, organization_id, entity_type, entity_id, action, actor_id, actor_type, changes, timestamp
			FROM audit_log
			WHERE organization_id IS NULL AND entity_type = $1 AND entity_id = $2
			ORDER BY timestamp DESC
			LIMIT $3 OFFSET $4`
		rows, err = r.db.QueryContext(ctx, query, entityType, entityID, limit, offset)
	} else {
		query = `
			SELECT id, organization_id, entity_type, entity_id, action, actor_id, actor_type, changes, timestamp
			FROM audit_log
			WHERE organization_id = $1 AND entity_type = $2 AND entity_id = $3
			ORDER BY timestamp DESC
			LIMIT $4 OFFSET $5`
		rows, err = r.db.QueryContext(ctx, query, organizationID, entityType, entityID, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by entity: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		log := models.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.OrganizationID, &log.EntityType, &log.EntityID, &log.Action,
			&log.ActorID, &log.ActorType, &log.Changes, &log.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetAuditLogsByActor retrieves all audit logs by a specific actor within an organization
// Pass nil for organizationID to query system-level audit logs
func (r *AuditRepository) GetAuditLogsByActor(
	ctx context.Context,
	organizationID *uuid.UUID,
	actorID uuid.UUID,
	limit, offset int,
) ([]models.AuditLog, error) {
	var query string
	var rows *sql.Rows
	var err error

	if organizationID == nil {
		query = `
			SELECT id, organization_id, entity_type, entity_id, action, actor_id, actor_type, changes, timestamp
			FROM audit_log
			WHERE organization_id IS NULL AND actor_id = $1
			ORDER BY timestamp DESC
			LIMIT $2 OFFSET $3`
		rows, err = r.db.QueryContext(ctx, query, actorID, limit, offset)
	} else {
		query = `
			SELECT id, organization_id, entity_type, entity_id, action, actor_id, actor_type, changes, timestamp
			FROM audit_log
			WHERE organization_id = $1 AND actor_id = $2
			ORDER BY timestamp DESC
			LIMIT $3 OFFSET $4`
		rows, err = r.db.QueryContext(ctx, query, organizationID, actorID, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by actor: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		log := models.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.OrganizationID, &log.EntityType, &log.EntityID, &log.Action,
			&log.ActorID, &log.ActorType, &log.Changes, &log.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}
