package engine

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
)

// Evaluator handles condition evaluation
type Evaluator struct{}

// NewEvaluator creates a new condition evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// EvaluateCondition evaluates a condition against context data
func (e *Evaluator) EvaluateCondition(condition *models.Condition, context map[string]interface{}) (bool, error) {
	if condition == nil {
		return true, nil
	}

	// Handle AND conditions
	if len(condition.And) > 0 {
		for _, subCondition := range condition.And {
			result, err := e.EvaluateCondition(&subCondition, context)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// Handle OR conditions
	if len(condition.Or) > 0 {
		for _, subCondition := range condition.Or {
			result, err := e.EvaluateCondition(&subCondition, context)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// Evaluate single condition
	fieldValue, err := e.getFieldValue(condition.Field, context)
	if err != nil {
		return false, fmt.Errorf("failed to get field value: %w", err)
	}

	return e.compareValues(fieldValue, condition.Operator, condition.Value)
}

// getFieldValue extracts a field value from context using dot notation
// Example: "order.total" retrieves context["order"]["total"]
func (e *Evaluator) getFieldValue(field string, context map[string]interface{}) (interface{}, error) {
	parts := strings.Split(field, ".")
	current := context

	for i, part := range parts {
		val, exists := current[part]
		if !exists {
			return nil, fmt.Errorf("field not found: %s", field)
		}

		// Last part - return the value
		if i == len(parts)-1 {
			return val, nil
		}

		// Navigate deeper
		nextMap, ok := val.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid path: %s is not an object", part)
		}
		current = nextMap
	}

	return nil, fmt.Errorf("field not found: %s", field)
}

// compareValues compares two values using the specified operator
func (e *Evaluator) compareValues(fieldValue interface{}, operator string, conditionValue interface{}) (bool, error) {
	switch operator {
	case "eq", "==":
		return e.equals(fieldValue, conditionValue), nil

	case "neq", "!=":
		return !e.equals(fieldValue, conditionValue), nil

	case "gt", ">":
		return e.greaterThan(fieldValue, conditionValue)

	case "gte", ">=":
		result, err := e.greaterThan(fieldValue, conditionValue)
		if err != nil {
			return false, err
		}
		return result || e.equals(fieldValue, conditionValue), nil

	case "lt", "<":
		return e.lessThan(fieldValue, conditionValue)

	case "lte", "<=":
		result, err := e.lessThan(fieldValue, conditionValue)
		if err != nil {
			return false, err
		}
		return result || e.equals(fieldValue, conditionValue), nil

	case "in":
		return e.in(fieldValue, conditionValue)

	case "contains":
		return e.contains(fieldValue, conditionValue)

	case "regex":
		return e.matchesRegex(fieldValue, conditionValue)

	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// equals checks if two values are equal
func (e *Evaluator) equals(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// greaterThan checks if a > b (for numbers)
func (e *Evaluator) greaterThan(a, b interface{}) (bool, error) {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)

	if !aOk || !bOk {
		return false, fmt.Errorf("cannot compare non-numeric values: %v > %v", a, b)
	}

	return aFloat > bFloat, nil
}

// lessThan checks if a < b (for numbers)
func (e *Evaluator) lessThan(a, b interface{}) (bool, error) {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)

	if !aOk || !bOk {
		return false, fmt.Errorf("cannot compare non-numeric values: %v < %v", a, b)
	}

	return aFloat < bFloat, nil
}

// in checks if a value is in a list
func (e *Evaluator) in(value interface{}, list interface{}) (bool, error) {
	switch listVal := list.(type) {
	case []interface{}:
		for _, item := range listVal {
			if e.equals(value, item) {
				return true, nil
			}
		}
		return false, nil
	case []string:
		valueStr := fmt.Sprintf("%v", value)
		for _, item := range listVal {
			if valueStr == item {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("'in' operator requires a list, got %T", list)
	}
}

// contains checks if a string contains a substring or if a list contains an item
func (e *Evaluator) contains(haystack interface{}, needle interface{}) (bool, error) {
	switch h := haystack.(type) {
	case string:
		needleStr := fmt.Sprintf("%v", needle)
		return strings.Contains(h, needleStr), nil
	case []interface{}:
		for _, item := range h {
			if e.equals(item, needle) {
				return true, nil
			}
		}
		return false, nil
	case []string:
		needleStr := fmt.Sprintf("%v", needle)
		for _, item := range h {
			if item == needleStr {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("'contains' operator requires a string or list, got %T", haystack)
	}
}

// matchesRegex checks if a string matches a regex pattern
func (e *Evaluator) matchesRegex(value interface{}, pattern interface{}) (bool, error) {
	valueStr := fmt.Sprintf("%v", value)
	patternStr := fmt.Sprintf("%v", pattern)

	matched, err := regexp.MatchString(patternStr, valueStr)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return matched, nil
}

// toFloat64 converts various numeric types to float64
func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}
