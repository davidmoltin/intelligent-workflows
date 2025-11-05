package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurity_RBACWorkflowPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Create user with limited permissions (read-only)
	registerReq := map[string]interface{}{
		"username": "readonlyuser",
		"email":    "readonly@example.com",
		"password": "SecurePass123!",
	}

	body, err := json.Marshal(registerReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(t, err)

	accessToken := registerResp["access_token"].(string)

	// Try to create a workflow (should fail with read-only permissions)
	workflowReq := map[string]interface{}{
		"workflow_id": "rbac-test",
		"version":     "1.0.0",
		"name":        "RBAC Test Workflow",
		"definition": map[string]interface{}{
			"trigger": map[string]interface{}{
				"type":  "event",
				"event": "test.event",
			},
			"steps": []map[string]interface{}{
				{
					"id":   "step1",
					"type": "action",
					"action": map[string]interface{}{
						"action": "allow",
					},
				},
			},
		},
	}

	body, err = json.Marshal(workflowReq)
	require.NoError(t, err)

	req, err := http.NewRequest(
		"POST",
		server.BaseURL+"/api/v1/workflows",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should be forbidden due to insufficient permissions
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestSecurity_RBACApprovalPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Create regular user
	registerReq := map[string]interface{}{
		"username": "regularuser",
		"email":    "regular@example.com",
		"password": "SecurePass123!",
	}

	body, err := json.Marshal(registerReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(t, err)

	accessToken := registerResp["access_token"].(string)

	// Try to approve a request meant for managers
	approveReq := map[string]interface{}{
		"decision": "approved",
		"comment":  "Unauthorized approval attempt",
	}

	body, err = json.Marshal(approveReq)
	require.NoError(t, err)

	req, err := http.NewRequest(
		"POST",
		server.BaseURL+"/api/v1/approvals/test-approval-id/approve",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should be forbidden or not found
	assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound)
}

func TestSecurity_ResourceOwnership(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Create two users
	user1Token := createUserAndGetToken(t, server, "user1", "user1@example.com")
	user2Token := createUserAndGetToken(t, server, "user2", "user2@example.com")

	// User 1 creates a workflow
	workflowReq := map[string]interface{}{
		"workflow_id": "ownership-test",
		"version":     "1.0.0",
		"name":        "Ownership Test Workflow",
		"definition": map[string]interface{}{
			"trigger": map[string]interface{}{
				"type":  "event",
				"event": "test.event",
			},
			"steps": []map[string]interface{}{
				{
					"id":   "step1",
					"type": "action",
					"action": map[string]interface{}{
						"action": "allow",
					},
				},
			},
		},
	}

	body, err := json.Marshal(workflowReq)
	require.NoError(t, err)

	req, err := http.NewRequest(
		"POST",
		server.BaseURL+"/api/v1/workflows",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user1Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		var workflow map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&workflow)
		require.NoError(t, err)

		workflowID := workflow["id"].(string)

		// User 2 tries to delete User 1's workflow
		deleteReq, err := http.NewRequest(
			"DELETE",
			server.BaseURL+"/api/v1/workflows/"+workflowID,
			nil,
		)
		require.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+user2Token)

		resp, err = client.Do(deleteReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be forbidden
		assert.Equal(t, http.StatusForbidden, resp.StatusCode,
			"User should not be able to delete another user's workflow")
	}
}

func TestSecurity_AdminPrivilegeEscalation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Create regular user
	userToken := createUserAndGetToken(t, server, "reguser", "reg@example.com")

	// Try to assign admin role to self
	roleReq := map[string]interface{}{
		"role": "admin",
	}

	body, err := json.Marshal(roleReq)
	require.NoError(t, err)

	req, err := http.NewRequest(
		"POST",
		server.BaseURL+"/api/v1/users/me/roles",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should be forbidden
	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"User should not be able to escalate own privileges")
}

func TestSecurity_APIKeyScopeEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	server := NewSecurityTestServer(t)
	server.Start()
	defer server.Stop()

	// Create user and API key with limited scope
	userToken := createUserAndGetToken(t, server, "scopeuser", "scope@example.com")

	// Create API key with read-only scope
	apiKeyReq := map[string]interface{}{
		"name":   "Read-Only Key",
		"scopes": []string{"workflows:read"},
	}

	body, err := json.Marshal(apiKeyReq)
	require.NoError(t, err)

	req, err := http.NewRequest(
		"POST",
		server.BaseURL+"/api/v1/auth/api-keys",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		var apiKeyResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&apiKeyResp)
		require.NoError(t, err)

		apiKey := apiKeyResp["api_key"].(string)

		// Try to create workflow with read-only API key
		workflowReq := map[string]interface{}{
			"workflow_id": "scope-test",
			"version":     "1.0.0",
			"name":        "Scope Test",
			"definition": map[string]interface{}{
				"trigger": map[string]interface{}{
					"type":  "event",
					"event": "test.event",
				},
				"steps": []map[string]interface{}{},
			},
		}

		body, err = json.Marshal(workflowReq)
		require.NoError(t, err)

		req, err = http.NewRequest(
			"POST",
			server.BaseURL+"/api/v1/workflows",
			bytes.NewBuffer(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)

		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be forbidden due to insufficient scope
		assert.Equal(t, http.StatusForbidden, resp.StatusCode,
			"Read-only API key should not allow write operations")
	}
}

// Helper function
func createUserAndGetToken(t *testing.T, server *SecurityTestServer, username, email string) string {
	t.Helper()

	registerReq := map[string]interface{}{
		"username": username,
		"email":    email,
		"password": "SecurePass123!",
	}

	body, err := json.Marshal(registerReq)
	require.NoError(t, err)

	resp, err := http.Post(
		server.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(t, err)

	return registerResp["access_token"].(string)
}
