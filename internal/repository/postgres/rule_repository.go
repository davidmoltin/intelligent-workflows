package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// RuleRepository handles rule database operations
type RuleRepository struct {
	db *sql.DB
}

// NewRuleRepository creates a new rule repository
func NewRuleRepository(db *sql.DB) *RuleRepository {
	return &RuleRepository{db: db}
}

// Create creates a new rule
func (r *RuleRepository) Create(ctx context.Context, req *models.CreateRuleRequest) (*models.Rule, error) {
	query := `
		INSERT INTO rules (
			rule_id, name, description, rule_type, definition, enabled
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, rule_id, name, description, rule_type, definition, enabled, created_at, updated_at`

	rule := &models.Rule{}
	enabled := true

	err := r.db.QueryRowContext(
		ctx, query,
		req.RuleID, req.Name, req.Description, req.RuleType, req.Definition, enabled,
	).Scan(
		&rule.ID, &rule.RuleID, &rule.Name, &rule.Description,
		&rule.RuleType, &rule.Definition, &rule.Enabled,
		&rule.CreatedAt, &rule.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	return rule, nil
}

// GetByID retrieves a rule by UUID
func (r *RuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Rule, error) {
	query := `
		SELECT id, rule_id, name, description, rule_type, definition, enabled, created_at, updated_at
		FROM rules
		WHERE id = $1`

	rule := &models.Rule{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rule.ID, &rule.RuleID, &rule.Name, &rule.Description,
		&rule.RuleType, &rule.Definition, &rule.Enabled,
		&rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	return rule, nil
}

// GetByRuleID retrieves a rule by its rule_id string
func (r *RuleRepository) GetByRuleID(ctx context.Context, ruleID string) (*models.Rule, error) {
	query := `
		SELECT id, rule_id, name, description, rule_type, definition, enabled, created_at, updated_at
		FROM rules
		WHERE rule_id = $1`

	rule := &models.Rule{}
	err := r.db.QueryRowContext(ctx, query, ruleID).Scan(
		&rule.ID, &rule.RuleID, &rule.Name, &rule.Description,
		&rule.RuleType, &rule.Definition, &rule.Enabled,
		&rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	return rule, nil
}

// List retrieves rules with optional filtering and pagination
func (r *RuleRepository) List(ctx context.Context, enabled *bool, ruleType *models.RuleType, limit, offset int) ([]*models.Rule, int64, error) {
	// Count total
	countQuery := `
		SELECT COUNT(*) FROM rules
		WHERE ($1::boolean IS NULL OR enabled = $1)
		AND ($2::text IS NULL OR rule_type = $2)`

	var total int64
	var ruleTypeStr *string
	if ruleType != nil {
		str := string(*ruleType)
		ruleTypeStr = &str
	}

	err := r.db.QueryRowContext(ctx, countQuery, enabled, ruleTypeStr).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count rules: %w", err)
	}

	// Get rules
	query := `
		SELECT id, rule_id, name, description, rule_type, definition, enabled, created_at, updated_at
		FROM rules
		WHERE ($1::boolean IS NULL OR enabled = $1)
		AND ($2::text IS NULL OR rule_type = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, enabled, ruleTypeStr, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list rules: %w", err)
	}
	defer rows.Close()

	var rules []*models.Rule
	for rows.Next() {
		rule := &models.Rule{}
		err := rows.Scan(
			&rule.ID, &rule.RuleID, &rule.Name, &rule.Description,
			&rule.RuleType, &rule.Definition, &rule.Enabled,
			&rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, total, nil
}

// Update updates a rule
func (r *RuleRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateRuleRequest) (*models.Rule, error) {
	query := `
		UPDATE rules
		SET name = COALESCE($2, name),
		    description = COALESCE($3, description),
		    definition = COALESCE($4, definition),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, rule_id, name, description, rule_type, definition, enabled, created_at, updated_at`

	rule := &models.Rule{}
	err := r.db.QueryRowContext(
		ctx, query,
		id, req.Name, req.Description, req.Definition,
	).Scan(
		&rule.ID, &rule.RuleID, &rule.Name, &rule.Description,
		&rule.RuleType, &rule.Definition, &rule.Enabled,
		&rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update rule: %w", err)
	}

	return rule, nil
}

// Delete deletes a rule
func (r *RuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM rules WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}

// Enable enables a rule
func (r *RuleRepository) Enable(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE rules SET enabled = true, updated_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to enable rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}

// Disable disables a rule
func (r *RuleRepository) Disable(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE rules SET enabled = false, updated_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to disable rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}
