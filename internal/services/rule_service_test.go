package services

import (
	"context"
	"testing"

	"github.com/davidmoltin/intelligent-workflows/internal/engine"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

func TestRuleService_ValidateRuleDefinition(t *testing.T) {
	log, _ := logger.New("info", "json")
	evaluator := engine.NewEvaluator()
	service := NewRuleService(nil, evaluator, nil, log)

	tests := []struct {
		name        string
		ruleType    models.RuleType
		definition  *models.RuleDefinition
		expectError bool
	}{
		{
			name:     "valid condition rule",
			ruleType: models.RuleTypeCondition,
			definition: &models.RuleDefinition{
				Conditions: []models.Condition{
					{Field: "order.total", Operator: "gt", Value: 1000.0},
				},
			},
			expectError: false,
		},
		{
			name:     "condition rule without conditions",
			ruleType: models.RuleTypeCondition,
			definition: &models.RuleDefinition{
				Conditions: []models.Condition{},
			},
			expectError: true,
		},
		{
			name:     "validation rule without conditions",
			ruleType: models.RuleTypeValidation,
			definition: &models.RuleDefinition{
				Conditions: []models.Condition{},
			},
			expectError: true,
		},
		{
			name:     "enrichment rule without actions",
			ruleType: models.RuleTypeEnrichment,
			definition: &models.RuleDefinition{
				Conditions: []models.Condition{
					{Field: "order.total", Operator: "gt", Value: 1000.0},
				},
				Actions: []models.Action{},
			},
			expectError: true,
		},
		{
			name:     "valid enrichment rule",
			ruleType: models.RuleTypeEnrichment,
			definition: &models.RuleDefinition{
				Conditions: []models.Condition{
					{Field: "order.total", Operator: "gt", Value: 1000.0},
				},
				Actions: []models.Action{
					{Type: "execute", Metadata: map[string]interface{}{"priority": "high"}},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateRuleDefinition(tt.ruleType, tt.definition)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestRuleService_TestConditionRule(t *testing.T) {
	log, _ := logger.New("info", "json")
	evaluator := engine.NewEvaluator()
	service := NewRuleService(nil, evaluator, nil, log)

	rule := &models.Rule{
		RuleType: models.RuleTypeCondition,
		Definition: models.RuleDefinition{
			Conditions: []models.Condition{
				{Field: "order.total", Operator: "gt", Value: 1000.0},
			},
		},
	}

	tests := []struct {
		name           string
		context        map[string]interface{}
		expectedPassed bool
	}{
		{
			name: "condition passes",
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total": 1500.0,
				},
			},
			expectedPassed: true,
		},
		{
			name: "condition fails",
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total": 500.0,
				},
			},
			expectedPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.testConditionRule(rule, tt.context)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if response.Passed != tt.expectedPassed {
				t.Errorf("expected passed=%v, got=%v", tt.expectedPassed, response.Passed)
			}
		})
	}
}

func TestRuleService_TestValidationRule(t *testing.T) {
	log, _ := logger.New("info", "json")
	evaluator := engine.NewEvaluator()
	service := NewRuleService(nil, evaluator, nil, log)

	rule := &models.Rule{
		RuleType: models.RuleTypeValidation,
		Definition: models.RuleDefinition{
			Conditions: []models.Condition{
				{Field: "email", Operator: "regex", Value: `^[^\s@]+@[^\s@]+\.[^\s@]+$`},
				{Field: "age", Operator: "gte", Value: 18.0},
			},
		},
	}

	tests := []struct {
		name           string
		context        map[string]interface{}
		expectedPassed bool
	}{
		{
			name: "all validations pass",
			context: map[string]interface{}{
				"email": "test@example.com",
				"age":   25.0,
			},
			expectedPassed: true,
		},
		{
			name: "email validation fails",
			context: map[string]interface{}{
				"email": "invalid-email",
				"age":   25.0,
			},
			expectedPassed: false,
		},
		{
			name: "age validation fails",
			context: map[string]interface{}{
				"email": "test@example.com",
				"age":   15.0,
			},
			expectedPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.testValidationRule(rule, tt.context)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if response.Passed != tt.expectedPassed {
				t.Errorf("expected passed=%v, got=%v", tt.expectedPassed, response.Passed)
			}
		})
	}
}

func TestRuleService_TestEnrichmentRule(t *testing.T) {
	log, _ := logger.New("info", "json")
	evaluator := engine.NewEvaluator()
	service := NewRuleService(nil, evaluator, nil, log)

	rule := &models.Rule{
		RuleType: models.RuleTypeEnrichment,
		Definition: models.RuleDefinition{
			Conditions: []models.Condition{
				{Field: "order.total", Operator: "gt", Value: 1000.0},
			},
			Actions: []models.Action{
				{
					Type: "execute",
					Metadata: map[string]interface{}{
						"priority": "high",
						"tier":     "premium",
					},
				},
			},
		},
	}

	ctx := context.Background()
	testContext := map[string]interface{}{
		"order": map[string]interface{}{
			"total": 1500.0,
		},
	}

	response, err := service.testEnrichmentRule(rule, testContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !response.Passed {
		t.Errorf("expected enrichment to pass")
	}

	// Check if enriched context has the new fields
	enrichedCtx, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be a map")
	}

	if enrichedCtx["priority"] != "high" {
		t.Errorf("expected priority to be 'high', got %v", enrichedCtx["priority"])
	}

	if enrichedCtx["tier"] != "premium" {
		t.Errorf("expected tier to be 'premium', got %v", enrichedCtx["tier"])
	}

	// Original context should be preserved
	order, ok := enrichedCtx["order"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected order to be preserved in enriched context")
	}

	if order["total"] != 1500.0 {
		t.Errorf("expected order.total to be 1500.0, got %v", order["total"])
	}

	_ = ctx // unused but keeping context for consistency
}
