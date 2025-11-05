package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Check if E2E tests should run
	if os.Getenv("E2E_TESTS") == "" {
		os.Exit(0)
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestWorkflowAPI_CreateWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create workflow request
	workflow := models.CreateWorkflowRequest{
		WorkflowID: "test-workflow-1",
		Version:    "1.0.0",
		Name:       "Test Workflow",
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
						Reason: "High value order",
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

	// Make request
	resp, err := http.Post(
		server.BaseURL+"/api/v1/workflows",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var created models.Workflow
	err = json.NewDecoder(resp.Body).Decode(&created)
	require.NoError(t, err)

	assert.Equal(t, "test-workflow-1", created.WorkflowID)
	assert.Equal(t, "1.0.0", created.Version)
	assert.Equal(t, "Test Workflow", created.Name)
}

func TestWorkflowAPI_GetWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// First create a workflow
	workflow := models.CreateWorkflowRequest{
		WorkflowID: "test-workflow-2",
		Version:    "1.0.0",
		Name:       "Test Workflow 2",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "order.created",
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
	createResp, err := http.Post(
		server.BaseURL+"/api/v1/workflows",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer createResp.Body.Close()

	var created models.Workflow
	json.NewDecoder(createResp.Body).Decode(&created)

	// Now get the workflow
	resp, err := http.Get(server.BaseURL + "/api/v1/workflows/" + created.ID.String())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var retrieved models.Workflow
	err = json.NewDecoder(resp.Body).Decode(&retrieved)
	require.NoError(t, err)

	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, "test-workflow-2", retrieved.WorkflowID)
}

func TestWorkflowAPI_ListWorkflows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()
	server.ResetDatabase()

	// Create multiple workflows
	for i := 1; i <= 3; i++ {
		workflow := models.CreateWorkflowRequest{
			WorkflowID: "test-workflow-" + string(rune('0'+i)),
			Version:    "1.0.0",
			Name:       "Test Workflow " + string(rune('0'+i)),
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type:  "event",
					Event: "order.created",
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

	// List workflows
	resp, err := http.Get(server.BaseURL + "/api/v1/workflows")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var workflows []models.Workflow
	err = json.NewDecoder(resp.Body).Decode(&workflows)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(workflows), 3)
}

func TestHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := NewTestServer(t)
	server.Start()
	defer server.Stop()

	resp, err := http.Get(server.BaseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
