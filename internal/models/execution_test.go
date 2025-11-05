package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONB_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
		check   func(t *testing.T, j JSONB)
	}{
		{
			name:  "nil value",
			value: nil,
			check: func(t *testing.T, j JSONB) {
				assert.NotNil(t, j)
				assert.Empty(t, j)
			},
		},
		{
			name:  "valid JSON bytes",
			value: []byte(`{"order_id": "123", "total": 1500.0}`),
			check: func(t *testing.T, j JSONB) {
				assert.Equal(t, "123", j["order_id"])
				assert.InDelta(t, 1500.0, j["total"], 0.001)
			},
		},
		{
			name:  "empty JSON object",
			value: []byte(`{}`),
			check: func(t *testing.T, j JSONB) {
				assert.NotNil(t, j)
				assert.Empty(t, j)
			},
		},
		{
			name:  "non-byte value",
			value: "string value",
			check: func(t *testing.T, j JSONB) {
				assert.NotNil(t, j)
				assert.Empty(t, j)
			},
		},
		{
			name:    "invalid JSON",
			value:   []byte(`{invalid json`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var j JSONB
			err := j.Scan(tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.check != nil {
				tt.check(t, j)
			}
		})
	}
}

func TestJSONB_Value(t *testing.T) {
	tests := []struct {
		name  string
		jsonb JSONB
		check func(t *testing.T, value interface{})
	}{
		{
			name: "non-empty JSONB",
			jsonb: JSONB{
				"order_id": "123",
				"total":    1500.0,
			},
			check: func(t *testing.T, value interface{}) {
				bytes, ok := value.([]byte)
				require.True(t, ok)

				var result map[string]interface{}
				err := json.Unmarshal(bytes, &result)
				require.NoError(t, err)

				assert.Equal(t, "123", result["order_id"])
				assert.InDelta(t, 1500.0, result["total"], 0.001)
			},
		},
		{
			name:  "nil JSONB",
			jsonb: nil,
			check: func(t *testing.T, value interface{}) {
				bytes, ok := value.([]byte)
				require.True(t, ok)

				var result map[string]interface{}
				err := json.Unmarshal(bytes, &result)
				require.NoError(t, err)

				assert.Empty(t, result)
			},
		},
		{
			name:  "empty JSONB",
			jsonb: JSONB{},
			check: func(t *testing.T, value interface{}) {
				bytes, ok := value.([]byte)
				require.True(t, ok)

				var result map[string]interface{}
				err := json.Unmarshal(bytes, &result)
				require.NoError(t, err)

				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.jsonb.Value()
			require.NoError(t, err)

			if tt.check != nil {
				tt.check(t, value)
			}
		})
	}
}

func TestJSONB_RoundTrip(t *testing.T) {
	original := JSONB{
		"order_id":    "order-123",
		"total":       1500.0,
		"customer_id": "cust-456",
		"items": []interface{}{
			map[string]interface{}{
				"sku":      "ITEM-1",
				"quantity": float64(2),
				"price":    50.0,
			},
			map[string]interface{}{
				"sku":      "ITEM-2",
				"quantity": float64(1),
				"price":    100.0,
			},
		},
	}

	// Convert to value
	value, err := original.Value()
	require.NoError(t, err)

	// Scan it back
	var scanned JSONB
	err = scanned.Scan(value)
	require.NoError(t, err)

	// Verify top-level fields
	assert.Equal(t, "order-123", scanned["order_id"])
	assert.InDelta(t, 1500.0, scanned["total"], 0.001)
	assert.Equal(t, "cust-456", scanned["customer_id"])

	// Verify nested array
	items, ok := scanned["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)

	item1, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "ITEM-1", item1["sku"])
}

func TestExecutionStatus_Constants(t *testing.T) {
	// Test that execution status constants are defined correctly
	statuses := []ExecutionStatus{
		ExecutionStatusPending,
		ExecutionStatusRunning,
		ExecutionStatusCompleted,
		ExecutionStatusFailed,
		ExecutionStatusBlocked,
		ExecutionStatusCancelled,
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}

	// Test that they're unique
	uniqueStatuses := make(map[ExecutionStatus]bool)
	for _, status := range statuses {
		assert.False(t, uniqueStatuses[status], "duplicate status: %s", status)
		uniqueStatuses[status] = true
	}
}

func TestExecutionResult_Constants(t *testing.T) {
	// Test that execution result constants are defined correctly
	results := []ExecutionResult{
		ExecutionResultAllowed,
		ExecutionResultBlocked,
		ExecutionResultExecuted,
		ExecutionResultFailed,
	}

	for _, result := range results {
		assert.NotEmpty(t, string(result))
	}

	// Test that they're unique
	uniqueResults := make(map[ExecutionResult]bool)
	for _, result := range results {
		assert.False(t, uniqueResults[result], "duplicate result: %s", result)
		uniqueResults[result] = true
	}
}

func TestStepExecutionStatus_Constants(t *testing.T) {
	// Test that step execution status constants are defined correctly
	statuses := []StepExecutionStatus{
		StepStatusPending,
		StepStatusRunning,
		StepStatusCompleted,
		StepStatusFailed,
		StepStatusSkipped,
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}

	// Test that they're unique
	uniqueStatuses := make(map[StepExecutionStatus]bool)
	for _, status := range statuses {
		assert.False(t, uniqueStatuses[status], "duplicate status: %s", status)
		uniqueStatuses[status] = true
	}
}
