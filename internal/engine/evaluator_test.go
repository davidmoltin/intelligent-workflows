package engine

import (
	"testing"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
)

func TestEvaluateCondition(t *testing.T) {
	evaluator := NewEvaluator()

	tests := []struct {
		name      string
		condition *models.Condition
		context   map[string]interface{}
		expected  bool
		shouldErr bool
	}{
		{
			name: "simple eq string",
			condition: &models.Condition{
				Field:    "status",
				Operator: "eq",
				Value:    "active",
			},
			context: map[string]interface{}{
				"status": "active",
			},
			expected: true,
		},
		{
			name: "simple neq string",
			condition: &models.Condition{
				Field:    "status",
				Operator: "neq",
				Value:    "inactive",
			},
			context: map[string]interface{}{
				"status": "active",
			},
			expected: true,
		},
		{
			name: "gt number",
			condition: &models.Condition{
				Field:    "order.total",
				Operator: "gt",
				Value:    1000.0,
			},
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total": 1500.0,
				},
			},
			expected: true,
		},
		{
			name: "lt number",
			condition: &models.Condition{
				Field:    "order.total",
				Operator: "lt",
				Value:    2000.0,
			},
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total": 1500.0,
				},
			},
			expected: true,
		},
		{
			name: "gte number",
			condition: &models.Condition{
				Field:    "order.total",
				Operator: "gte",
				Value:    1500.0,
			},
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total": 1500.0,
				},
			},
			expected: true,
		},
		{
			name: "lte number",
			condition: &models.Condition{
				Field:    "order.total",
				Operator: "lte",
				Value:    1500.0,
			},
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total": 1500.0,
				},
			},
			expected: true,
		},
		{
			name: "in array",
			condition: &models.Condition{
				Field:    "status",
				Operator: "in",
				Value:    []interface{}{"active", "pending", "processing"},
			},
			context: map[string]interface{}{
				"status": "active",
			},
			expected: true,
		},
		{
			name: "not in array",
			condition: &models.Condition{
				Field:    "status",
				Operator: "in",
				Value:    []interface{}{"cancelled", "failed"},
			},
			context: map[string]interface{}{
				"status": "active",
			},
			expected: false,
		},
		{
			name: "contains substring",
			condition: &models.Condition{
				Field:    "email",
				Operator: "contains",
				Value:    "@example.com",
			},
			context: map[string]interface{}{
				"email": "user@example.com",
			},
			expected: true,
		},
		{
			name: "AND condition - both true",
			condition: &models.Condition{
				Field:    "order.total",
				Operator: "gt",
				Value:    1000.0,
				And: []models.Condition{
					{
						Field:    "order.status",
						Operator: "eq",
						Value:    "pending",
					},
				},
			},
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total":  1500.0,
					"status": "pending",
				},
			},
			expected: true,
		},
		{
			name: "AND condition - one false",
			condition: &models.Condition{
				Field:    "order.total",
				Operator: "gt",
				Value:    1000.0,
				And: []models.Condition{
					{
						Field:    "order.status",
						Operator: "eq",
						Value:    "cancelled",
					},
				},
			},
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total":  1500.0,
					"status": "pending",
				},
			},
			expected: false,
		},
		{
			name: "OR condition - one true",
			condition: &models.Condition{
				Field:    "order.total",
				Operator: "lt",
				Value:    100.0,
				Or: []models.Condition{
					{
						Field:    "order.status",
						Operator: "eq",
						Value:    "pending",
					},
				},
			},
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total":  1500.0,
					"status": "pending",
				},
			},
			expected: true,
		},
		{
			name: "OR condition - both false",
			condition: &models.Condition{
				Field:    "order.total",
				Operator: "lt",
				Value:    100.0,
				Or: []models.Condition{
					{
						Field:    "order.status",
						Operator: "eq",
						Value:    "cancelled",
					},
				},
			},
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total":  1500.0,
					"status": "pending",
				},
			},
			expected: false,
		},
		{
			name: "nested field access",
			condition: &models.Condition{
				Field:    "customer.address.country",
				Operator: "eq",
				Value:    "US",
			},
			context: map[string]interface{}{
				"customer": map[string]interface{}{
					"address": map[string]interface{}{
						"country": "US",
					},
				},
			},
			expected: true,
		},
		{
			name: "missing field",
			condition: &models.Condition{
				Field:    "nonexistent.field",
				Operator: "eq",
				Value:    "value",
			},
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total": 1500.0,
				},
			},
			expected:  false,
			shouldErr: true,
		},
		{
			name: "regex match",
			condition: &models.Condition{
				Field:    "email",
				Operator: "regex",
				Value:    "^[a-z]+@example\\.com$",
			},
			context: map[string]interface{}{
				"email": "user@example.com",
			},
			expected: true,
		},
		{
			name: "regex no match",
			condition: &models.Condition{
				Field:    "email",
				Operator: "regex",
				Value:    "^[0-9]+@example\\.com$",
			},
			context: map[string]interface{}{
				"email": "user@example.com",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateCondition(tt.condition, tt.context)

			if tt.shouldErr {
				if err == nil {
					t.Error("Expected an error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetFieldValue(t *testing.T) {
	evaluator := NewEvaluator()

	tests := []struct {
		name      string
		field     string
		context   map[string]interface{}
		expected  interface{}
		shouldErr bool
	}{
		{
			name:  "simple field",
			field: "status",
			context: map[string]interface{}{
				"status": "active",
			},
			expected:  "active",
			shouldErr: false,
		},
		{
			name:  "nested field",
			field: "order.total",
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total": 1500.0,
				},
			},
			expected:  1500.0,
			shouldErr: false,
		},
		{
			name:  "deeply nested field",
			field: "customer.address.city",
			context: map[string]interface{}{
				"customer": map[string]interface{}{
					"address": map[string]interface{}{
						"city": "New York",
					},
				},
			},
			expected:  "New York",
			shouldErr: false,
		},
		{
			name:  "missing field",
			field: "nonexistent",
			context: map[string]interface{}{
				"status": "active",
			},
			expected:  nil,
			shouldErr: true,
		},
		{
			name:  "missing nested field",
			field: "order.missing.field",
			context: map[string]interface{}{
				"order": map[string]interface{}{
					"total": 1500.0,
				},
			},
			expected:  nil,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := evaluator.getFieldValue(tt.field, tt.context)

			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if value != tt.expected {
				t.Errorf("Expected value=%v, got %v", tt.expected, value)
			}
		})
	}
}
