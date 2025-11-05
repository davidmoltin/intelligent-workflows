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

func TestApproval_CreateAndApprove(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create approval request
	approvalReq := map[string]interface{}{
		"workflow_execution_id": "exec-123",
		"approver_role":         "manager",
		"context": map[string]interface{}{
			"reason":      "High value transaction",
			"amount":      5000.0,
			"customer_id": "cust-456",
		},
		"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	body, err := json.Marshal(approvalReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/approvals",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var approval map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&approval)
	require.NoError(t, err)
	assert.Contains(t, approval, "id")

	approvalID := approval["id"].(string)

	// Get approval details
	resp, err = http.Get(server.BaseURL + "/api/v1/approvals/" + approvalID)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Approve the request
	approveReq := map[string]interface{}{
		"decision": "approved",
		"comment":  "Approved by manager",
	}

	body, err = json.Marshal(approveReq)
	require.NoError(t, err)

	resp, err = http.Post(
		server.BaseURL+"/api/v1/approvals/"+approvalID+"/approve",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var approveResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&approveResp)
	require.NoError(t, err)
	assert.Equal(t, "approved", approveResp["status"])
}

func TestApproval_CreateAndReject(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create approval request
	approvalReq := map[string]interface{}{
		"workflow_execution_id": "exec-456",
		"approver_role":         "supervisor",
		"context": map[string]interface{}{
			"reason":      "Policy violation suspected",
			"customer_id": "cust-789",
		},
		"expires_at": time.Now().Add(12 * time.Hour).Format(time.RFC3339),
	}

	body, err := json.Marshal(approvalReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/approvals",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var approval map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&approval)
	require.NoError(t, err)

	approvalID := approval["id"].(string)

	// Reject the request
	rejectReq := map[string]interface{}{
		"decision": "rejected",
		"comment":  "Does not meet policy requirements",
	}

	body, err = json.Marshal(rejectReq)
	require.NoError(t, err)

	resp, err = http.Post(
		server.BaseURL+"/api/v1/approvals/"+approvalID+"/reject",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var rejectResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&rejectResp)
	require.NoError(t, err)
	assert.Equal(t, "rejected", rejectResp["status"])
}

func TestApproval_ListPendingApprovals(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create multiple approval requests
	for i := 1; i <= 5; i++ {
		approvalReq := map[string]interface{}{
			"workflow_execution_id": "exec-" + string(rune('0'+i)),
			"approver_role":         "manager",
			"context": map[string]interface{}{
				"index": i,
			},
			"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}

		body, _ := json.Marshal(approvalReq)
		resp, _ := http.Post(
			server.BaseURL+"/api/v1/approvals",
			"application/json",
			bytes.NewBuffer(body),
		)
		resp.Body.Close()
	}

	// List all approvals
	resp, err := http.Get(server.BaseURL + "/api/v1/approvals")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var approvals []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&approvals)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(approvals), 5)
}

func TestApproval_WorkflowWithWaitStep(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create workflow with wait step for approval
	workflow := models.CreateWorkflowRequest{
		WorkflowID: "approval-wait-test",
		Version:    "1.0.0",
		Name:       "Approval Wait Test Workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "transaction.submitted",
			},
			Steps: []models.Step{
				{
					ID:   "check_amount",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "transaction.amount",
						Operator: "gte",
						Value:    10000.0,
					},
					OnTrue:  "require_approval",
					OnFalse: "auto_approve",
				},
				{
					ID:   "require_approval",
					Type: "wait",
					Wait: &models.WaitConfig{
						ForEvent: "approval.granted",
						Timeout:  "24h",
						OnTimeout: &models.TimeoutAction{
							Action: "reject",
						},
					},
					Next: "approved",
				},
				{
					ID:   "approved",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
				{
					ID:   "auto_approve",
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

	// Trigger workflow with high value transaction
	event := map[string]interface{}{
		"event_type": "transaction.submitted",
		"source":     "payment-gateway",
		"payload": map[string]interface{}{
			"transaction_id": "txn-12345",
			"amount":         15000.0,
			"customer_id":    "cust-abc",
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

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// Wait for workflow to pause at wait step
	time.Sleep(1 * time.Second)

	// Verify execution is in waiting state
	resp, err = http.Get(server.BaseURL + "/api/v1/executions")
	require.NoError(t, err)
	defer resp.Body.Close()

	var executions []models.WorkflowExecution
	err = json.NewDecoder(resp.Body).Decode(&executions)
	require.NoError(t, err)

	if len(executions) > 0 {
		assert.Equal(t, "waiting", executions[0].Status)
	}
}

func TestApproval_ExpiredApproval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create approval request that expires immediately
	approvalReq := map[string]interface{}{
		"workflow_execution_id": "exec-expired",
		"approver_role":         "manager",
		"context": map[string]interface{}{
			"reason": "Test expired approval",
		},
		"expires_at": time.Now().Add(1 * time.Second).Format(time.RFC3339),
	}

	body, err := json.Marshal(approvalReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/approvals",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var approval map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&approval)
	require.NoError(t, err)

	approvalID := approval["id"].(string)

	// Wait for approval to expire
	time.Sleep(2 * time.Second)

	// Try to approve expired request
	approveReq := map[string]interface{}{
		"decision": "approved",
		"comment":  "Late approval",
	}

	body, err = json.Marshal(approveReq)
	require.NoError(t, err)

	resp, err = http.Post(
		server.BaseURL+"/api/v1/approvals/"+approvalID+"/approve",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should fail because approval has expired
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
