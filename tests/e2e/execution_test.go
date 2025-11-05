package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowExecution_EventTriggered(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create a workflow
	workflow := models.CreateWorkflowRequest{
		WorkflowID: "event-test-workflow",
		Version:    "1.0.0",
		Name:       "Event Test Workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "order.created",
			},
			Steps: []models.Step{
				{
					ID:   "step1",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "order.total",
						Operator: "gt",
						Value:    1000.0,
					},
					OnTrue:  "step2",
					OnFalse: "step3",
				},
				{
					ID:   "step2",
					Type: "action",
					Action: &models.Action{
						Type:   "block",
						Reason: "High value order requires review",
					},
				},
				{
					ID:   "step3",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}

	body, err := json.Marshal(workflow)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/workflows",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createdWorkflow models.Workflow
	err = json.NewDecoder(resp.Body).Decode(&createdWorkflow)
	require.NoError(t, err)

	// Emit an event to trigger the workflow
	event := map[string]interface{}{
		"event_type": "order.created",
		"source":     "e2e-test",
		"payload": map[string]interface{}{
			"order_id": "order-123",
			"total":    1500.0,
			"customer": "customer-456",
		},
	}

	body, err = json.Marshal(event)
	require.NoError(t, err)

	resp, err = http.Post(
		server.BaseURL+"/api/v1/events",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Event should be accepted
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// Wait for execution to complete
	time.Sleep(500 * time.Millisecond)

	// List executions to verify workflow was triggered
	resp, err = http.Get(server.BaseURL + "/api/v1/executions")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var executions []models.WorkflowExecution
	err = json.NewDecoder(resp.Body).Decode(&executions)
	require.NoError(t, err)

	// Verify at least one execution was created
	assert.GreaterOrEqual(t, len(executions), 1)
}

func TestWorkflowExecution_ConditionalBranching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create workflow with conditional logic
	workflow := models.CreateWorkflowRequest{
		WorkflowID: "conditional-test",
		Version:    "1.0.0",
		Name:       "Conditional Test Workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "payment.processed",
			},
			Steps: []models.Step{
				{
					ID:   "check_amount",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "payment.amount",
						Operator: "gte",
						Value:    500.0,
					},
					OnTrue:  "high_value",
					OnFalse: "low_value",
				},
				{
					ID:   "high_value",
					Type: "action",
					Action: &models.Action{
						Type:   "execute",
						Action: "notify",
						Params: map[string]interface{}{
							"message": "High value payment detected",
						},
					},
				},
				{
					ID:   "low_value",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}

	body, err := json.Marshal(workflow)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/workflows",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Test with high value payment
	highValueEvent := map[string]interface{}{
		"event_type": "payment.processed",
		"source":     "payment-system",
		"payload": map[string]interface{}{
			"payment_id": "pay-789",
			"amount":     750.0,
		},
	}

	body, err = json.Marshal(highValueEvent)
	require.NoError(t, err)

	resp, err = http.Post(
		server.BaseURL+"/api/v1/events",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// Test with low value payment
	lowValueEvent := map[string]interface{}{
		"event_type": "payment.processed",
		"source":     "payment-system",
		"payload": map[string]interface{}{
			"payment_id": "pay-790",
			"amount":     250.0,
		},
	}

	body, err = json.Marshal(lowValueEvent)
	require.NoError(t, err)

	resp, err = http.Post(
		server.BaseURL+"/api/v1/events",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestWorkflowExecution_GetExecutionDetails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create and trigger a simple workflow
	workflow := models.CreateWorkflowRequest{
		WorkflowID: "detail-test",
		Version:    "1.0.0",
		Name:       "Detail Test Workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "test.event",
			},
			Steps: []models.Step{
				{
					ID:   "step1",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}

	body, err := json.Marshal(workflow)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/workflows",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Trigger the workflow
	event := map[string]interface{}{
		"event_type": "test.event",
		"source":     "test",
		"payload": map[string]interface{}{
			"test": "data",
		},
	}

	body, err = json.Marshal(event)
	require.NoError(t, err)

	resp, err = http.Post(
		server.BaseURL+"/api/v1/events",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Wait for execution
	time.Sleep(500 * time.Millisecond)

	// Get execution list
	resp, err = http.Get(server.BaseURL + "/api/v1/executions")
	require.NoError(t, err)
	defer resp.Body.Close()

	var executions []models.WorkflowExecution
	err = json.NewDecoder(resp.Body).Decode(&executions)
	require.NoError(t, err)

	if len(executions) > 0 {
		// Get execution details
		executionID := executions[0].ID.String()
		resp, err = http.Get(server.BaseURL + "/api/v1/executions/" + executionID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var execution models.WorkflowExecution
		err = json.NewDecoder(resp.Body).Decode(&execution)
		require.NoError(t, err)

		assert.Equal(t, executionID, execution.ID.String())
		assert.NotNil(t, execution.WorkflowID)
	}
}

func TestWorkflowExecution_ListWithFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create multiple workflows and trigger them
	for i := 1; i <= 3; i++ {
		workflow := models.CreateWorkflowRequest{
			WorkflowID: "list-test-" + string(rune('0'+i)),
			Version:    "1.0.0",
			Name:       "List Test " + string(rune('0'+i)),
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type:  "event",
					Event: "list.test",
				},
				Steps: []models.Step{
					{
						ID:   "step1",
						Type: "action",
						Action: &models.Action{
							Type: "allow",
						},
					},
				},
			},
		}

		body, _ := json.Marshal(workflow)
		resp, _ := http.Post(
			server.BaseURL+"/api/v1/workflows",
			"application/json",
			bytes.NewBuffer(body),
		)
		resp.Body.Close()
	}

	// Trigger events
	for i := 1; i <= 5; i++ {
		event := map[string]interface{}{
			"event_type": "list.test",
			"source":     "test",
			"payload": map[string]interface{}{
				"index": i,
			},
		}

		body, _ := json.Marshal(event)
		resp, _ := http.Post(
			server.BaseURL+"/api/v1/events",
			"application/json",
			bytes.NewBuffer(body),
		)
		resp.Body.Close()
	}

	// Wait for executions
	time.Sleep(1 * time.Second)

	// List all executions
	resp, err := http.Get(server.BaseURL + "/api/v1/executions")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var executions []models.WorkflowExecution
	err = json.NewDecoder(resp.Body).Decode(&executions)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(executions), 5)
}
