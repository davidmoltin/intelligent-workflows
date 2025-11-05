package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// ActionResult represents the result of an action execution
type ActionResult struct {
	Action  string                 `json:"action"` // allow, block, execute
	Success bool                   `json:"success"`
	Reason  string                 `json:"reason,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// ActionExecutor handles executing workflow actions
type ActionExecutor struct {
	logger     *logger.Logger
	httpClient *http.Client
}

// NewActionExecutor creates a new action executor
func NewActionExecutor(log *logger.Logger) *ActionExecutor {
	return &ActionExecutor{
		logger: log,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteAction executes a workflow action
func (ae *ActionExecutor) ExecuteAction(
	ctx context.Context,
	step *models.Step,
	execContext map[string]interface{},
) (*ActionResult, error) {
	if step.Action == nil {
		return nil, fmt.Errorf("step has no action defined")
	}

	result := &ActionResult{
		Action:  step.Action.Type,
		Success: true,
		Data:    make(map[string]interface{}),
	}

	switch step.Action.Type {
	case "allow":
		result.Reason = "Action allowed by workflow"
		ae.logger.Infof("Action allowed: %s", step.ID)

	case "block":
		result.Reason = step.Action.Reason
		if result.Reason == "" {
			result.Reason = "Action blocked by workflow"
		}
		ae.logger.Infof("Action blocked: %s - %s", step.ID, result.Reason)

	case "execute":
		// Execute additional actions defined in the Execute field
		if len(step.Execute) > 0 {
			executeResults, err := ae.executeActions(ctx, step.Execute, execContext)
			if err != nil {
				result.Success = false
				result.Error = err.Error()
				return result, err
			}
			result.Data["execute_results"] = executeResults
		}

	default:
		return nil, fmt.Errorf("unsupported action type: %s", step.Action.Type)
	}

	return result, nil
}

// executeActions executes a list of execute actions
func (ae *ActionExecutor) executeActions(
	ctx context.Context,
	actions []models.ExecuteAction,
	execContext map[string]interface{},
) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0, len(actions))

	for _, action := range actions {
		result, err := ae.executeSingleAction(ctx, action, execContext)
		if err != nil {
			ae.logger.Errorf("Failed to execute action %s: %v", action.Type, err)
			result = map[string]interface{}{
				"type":    action.Type,
				"success": false,
				"error":   err.Error(),
			}
		}
		results = append(results, result)
	}

	return results, nil
}

// executeSingleAction executes a single execute action
func (ae *ActionExecutor) executeSingleAction(
	ctx context.Context,
	action models.ExecuteAction,
	execContext map[string]interface{},
) (map[string]interface{}, error) {
	switch action.Type {
	case "notify":
		return ae.executeNotify(ctx, action, execContext)

	case "webhook", "http_request":
		return ae.executeWebhook(ctx, action, execContext)

	case "create_record":
		return ae.executeCreateRecord(ctx, action, execContext)

	case "update_record":
		return ae.executeUpdateRecord(ctx, action, execContext)

	case "log":
		return ae.executeLog(ctx, action, execContext)

	default:
		return nil, fmt.Errorf("unsupported execute action type: %s", action.Type)
	}
}

// executeNotify sends a notification
func (ae *ActionExecutor) executeNotify(
	ctx context.Context,
	action models.ExecuteAction,
	execContext map[string]interface{},
) (map[string]interface{}, error) {
	ae.logger.Infof("Executing notify action to: %v", action.Recipients)

	// In a real implementation, this would integrate with:
	// - Email service (SendGrid, AWS SES, etc.)
	// - SMS service (Twilio, etc.)
	// - Push notification service
	// - Slack/Teams webhooks

	result := map[string]interface{}{
		"type":       "notify",
		"success":    true,
		"recipients": action.Recipients,
		"message":    action.Message,
		"sent_at":    time.Now().Unix(),
	}

	// Placeholder: Log the notification
	ae.logger.Infof("Notification sent to %v: %s", action.Recipients, action.Message)

	return result, nil
}

// executeWebhook calls an external webhook
func (ae *ActionExecutor) executeWebhook(
	ctx context.Context,
	action models.ExecuteAction,
	execContext map[string]interface{},
) (map[string]interface{}, error) {
	if action.URL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	method := action.Method
	if method == "" {
		method = "POST"
	}

	// Prepare request body
	var bodyBytes []byte
	var err error

	if action.Body != nil {
		// Interpolate context variables into body
		interpolatedBody := ae.interpolateVariables(action.Body, execContext)
		bodyBytes, err = json.Marshal(interpolatedBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, action.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "IntelligentWorkflows/1.0")
	req.Header.Set("X-Request-ID", uuid.New().String())

	for key, value := range action.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	ae.logger.Infof("Calling webhook: %s %s", method, action.URL)
	resp, err := ae.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := map[string]interface{}{
		"type":        "webhook",
		"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
		"url":         action.URL,
		"method":      method,
		"status_code": resp.StatusCode,
		"response":    string(respBody),
		"called_at":   time.Now().Unix(),
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	ae.logger.Infof("Webhook call successful: %s - Status %d", action.URL, resp.StatusCode)

	return result, nil
}

// executeCreateRecord creates a record (placeholder for microservice integration)
func (ae *ActionExecutor) executeCreateRecord(
	ctx context.Context,
	action models.ExecuteAction,
	execContext map[string]interface{},
) (map[string]interface{}, error) {
	ae.logger.Infof("Executing create_record: entity=%s", action.Entity)

	// In a real implementation, this would call the appropriate microservice
	// to create a record (e.g., create order, product, customer, etc.)

	recordID := uuid.New().String()

	result := map[string]interface{}{
		"type":       "create_record",
		"success":    true,
		"entity":     action.Entity,
		"record_id":  recordID,
		"data":       action.Data,
		"created_at": time.Now().Unix(),
	}

	ae.logger.Infof("Record created: %s/%s", action.Entity, recordID)

	return result, nil
}

// executeUpdateRecord updates a record (placeholder for microservice integration)
func (ae *ActionExecutor) executeUpdateRecord(
	ctx context.Context,
	action models.ExecuteAction,
	execContext map[string]interface{},
) (map[string]interface{}, error) {
	ae.logger.Infof("Executing update_record: entity=%s, id=%s", action.Entity, action.EntityID)

	// In a real implementation, this would call the appropriate microservice
	// to update a record

	result := map[string]interface{}{
		"type":       "update_record",
		"success":    true,
		"entity":     action.Entity,
		"entity_id":  action.EntityID,
		"data":       action.Data,
		"updated_at": time.Now().Unix(),
	}

	ae.logger.Infof("Record updated: %s/%s", action.Entity, action.EntityID)

	return result, nil
}

// executeLog logs a message
func (ae *ActionExecutor) executeLog(
	ctx context.Context,
	action models.ExecuteAction,
	execContext map[string]interface{},
) (map[string]interface{}, error) {
	message := action.Message
	if message == "" {
		message = fmt.Sprintf("Log action executed: %v", action.Data)
	}

	ae.logger.Infof("Workflow log: %s", message)

	result := map[string]interface{}{
		"type":      "log",
		"success":   true,
		"message":   message,
		"logged_at": time.Now().Unix(),
	}

	return result, nil
}

// interpolateVariables replaces variables in data with values from context
// Example: "${order.id}" becomes the actual order ID from context
func (ae *ActionExecutor) interpolateVariables(
	data map[string]interface{},
	context map[string]interface{},
) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		switch v := value.(type) {
		case string:
			// Check if it's a variable reference like "${order.id}"
			if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
				varName := strings.TrimSuffix(strings.TrimPrefix(v, "${"), "}")
				result[key] = ae.getContextValue(varName, context)
			} else {
				result[key] = v
			}
		case map[string]interface{}:
			// Recursively interpolate nested maps
			result[key] = ae.interpolateVariables(v, context)
		default:
			result[key] = value
		}
	}

	return result
}

// getContextValue retrieves a value from context using dot notation
func (ae *ActionExecutor) getContextValue(path string, context map[string]interface{}) interface{} {
	parts := strings.Split(path, ".")
	current := context

	for i, part := range parts {
		val, exists := current[part]
		if !exists {
			return nil
		}

		if i == len(parts)-1 {
			return val
		}

		nextMap, ok := val.(map[string]interface{})
		if !ok {
			return nil
		}
		current = nextMap
	}

	return nil
}
