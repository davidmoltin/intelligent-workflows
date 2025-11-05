package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// CreateWorkflow creates a new workflow
func (c *Client) CreateWorkflow(workflow *models.Workflow) (*models.Workflow, error) {
	resp, err := c.doRequest("POST", "/api/v1/workflows", workflow)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create workflow: %s (status: %d)", string(body), resp.StatusCode)
	}

	var result models.Workflow
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetWorkflows retrieves all workflows
func (c *Client) GetWorkflows() ([]models.Workflow, error) {
	resp, err := c.doRequest("GET", "/api/v1/workflows", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get workflows: %s (status: %d)", string(body), resp.StatusCode)
	}

	var workflows []models.Workflow
	if err := json.NewDecoder(resp.Body).Decode(&workflows); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return workflows, nil
}

// GetWorkflow retrieves a workflow by ID
func (c *Client) GetWorkflow(id string) (*models.Workflow, error) {
	resp, err := c.doRequest("GET", "/api/v1/workflows/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get workflow: %s (status: %d)", string(body), resp.StatusCode)
	}

	var workflow models.Workflow
	if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &workflow, nil
}

// UpdateWorkflow updates an existing workflow
func (c *Client) UpdateWorkflow(id string, workflow *models.Workflow) (*models.Workflow, error) {
	resp, err := c.doRequest("PUT", "/api/v1/workflows/"+id, workflow)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update workflow: %s (status: %d)", string(body), resp.StatusCode)
	}

	var result models.Workflow
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteWorkflow deletes a workflow
func (c *Client) DeleteWorkflow(id string) error {
	resp, err := c.doRequest("DELETE", "/api/v1/workflows/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete workflow: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// GetExecutions retrieves workflow executions
func (c *Client) GetExecutions(workflowID string) ([]models.WorkflowExecution, error) {
	path := "/api/v1/executions"
	if workflowID != "" {
		path += "?workflow_id=" + workflowID
	}

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get executions: %s (status: %d)", string(body), resp.StatusCode)
	}

	var executions []models.WorkflowExecution
	if err := json.NewDecoder(resp.Body).Decode(&executions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return executions, nil
}

// GetExecution retrieves a specific execution
func (c *Client) GetExecution(id string) (*models.WorkflowExecution, error) {
	resp, err := c.doRequest("GET", "/api/v1/executions/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get execution: %s (status: %d)", string(body), resp.StatusCode)
	}

	var execution models.WorkflowExecution
	if err := json.NewDecoder(resp.Body).Decode(&execution); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &execution, nil
}

// CreateEvent sends an event to the API
func (c *Client) CreateEvent(event *models.Event) error {
	resp, err := c.doRequest("POST", "/api/v1/events", event)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create event: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// HealthCheck checks if the API is healthy
func (c *Client) HealthCheck() error {
	resp, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API is not healthy (status: %d)", resp.StatusCode)
	}

	return nil
}
