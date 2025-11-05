package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RuleType represents the type of rule
type RuleType string

const (
	RuleTypeCondition  RuleType = "condition"
	RuleTypeValidation RuleType = "validation"
	RuleTypeEnrichment RuleType = "enrichment"
)

// Rule represents a reusable rule
type Rule struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	RuleID      string         `json:"rule_id" db:"rule_id"`
	Name        string         `json:"name" db:"name"`
	Description *string        `json:"description,omitempty" db:"description"`
	RuleType    RuleType       `json:"rule_type" db:"rule_type"`
	Definition  RuleDefinition `json:"definition" db:"definition"`
	Enabled     bool           `json:"enabled" db:"enabled"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// RuleDefinition represents the rule logic
type RuleDefinition struct {
	Conditions []Condition            `json:"conditions,omitempty"`
	Actions    []Action               `json:"actions,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// CreateRuleRequest represents the request to create a rule
type CreateRuleRequest struct {
	RuleID      string         `json:"rule_id" validate:"required"`
	Name        string         `json:"name" validate:"required"`
	Description *string        `json:"description,omitempty"`
	RuleType    RuleType       `json:"rule_type" validate:"required"`
	Definition  RuleDefinition `json:"definition" validate:"required"`
}

// UpdateRuleRequest represents the request to update a rule
type UpdateRuleRequest struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Definition  *RuleDefinition `json:"definition,omitempty"`
}

// TestRuleRequest represents the request to test a rule
type TestRuleRequest struct {
	Context map[string]interface{} `json:"context" validate:"required"`
}

// TestRuleResponse represents the response of testing a rule
type TestRuleResponse struct {
	Passed  bool                   `json:"passed"`
	Result  interface{}            `json:"result,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// JSONB scanning for RuleDefinition
func (r *RuleDefinition) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, r)
}

func (r RuleDefinition) Value() (driver.Value, error) {
	return json.Marshal(r)
}
