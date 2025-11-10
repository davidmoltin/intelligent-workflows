package websocket

import (
	"encoding/json"
	"time"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Execution event types
	MessageTypeExecutionCreated   MessageType = "execution.created"
	MessageTypeExecutionStarted   MessageType = "execution.started"
	MessageTypeExecutionCompleted MessageType = "execution.completed"
	MessageTypeExecutionFailed    MessageType = "execution.failed"
	MessageTypeExecutionPaused    MessageType = "execution.paused"
	MessageTypeExecutionResumed   MessageType = "execution.resumed"
	MessageTypeExecutionCancelled MessageType = "execution.cancelled"
	MessageTypeExecutionBlocked   MessageType = "execution.blocked"

	// Step event types
	MessageTypeStepStarted   MessageType = "step.started"
	MessageTypeStepCompleted MessageType = "step.completed"
	MessageTypeStepFailed    MessageType = "step.failed"
	MessageTypeStepSkipped   MessageType = "step.skipped"

	// Approval event types
	MessageTypeApprovalRequired MessageType = "approval.required"
	MessageTypeApprovalGranted  MessageType = "approval.granted"
	MessageTypeApprovalDenied   MessageType = "approval.denied"
	MessageTypeApprovalExpired  MessageType = "approval.expired"

	// Connection management
	MessageTypePing         MessageType = "ping"
	MessageTypePong         MessageType = "pong"
	MessageTypeError        MessageType = "error"
	MessageTypeSubscribe    MessageType = "subscribe"
	MessageTypeUnsubscribe  MessageType = "unsubscribe"
	MessageTypeSubscribed   MessageType = "subscribed"
	MessageTypeUnsubscribed MessageType = "unsubscribed"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// ExecutionEventData contains execution event details
type ExecutionEventData struct {
	ExecutionID   string                 `json:"execution_id"`
	WorkflowID    string                 `json:"workflow_id"`
	Status        string                 `json:"status"`
	Result        string                 `json:"result,omitempty"`
	TriggerEvent  string                 `json:"trigger_event,omitempty"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	DurationMs    *int                   `json:"duration_ms,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
	PausedReason  string                 `json:"paused_reason,omitempty"`
}

// StepEventData contains step execution event details
type StepEventData struct {
	ExecutionID  string                 `json:"execution_id"`
	StepID       string                 `json:"step_id"`
	StepType     string                 `json:"step_type"`
	Status       string                 `json:"status"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	DurationMs   *int                   `json:"duration_ms,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Output       map[string]interface{} `json:"output,omitempty"`
}

// ApprovalEventData contains approval event details
type ApprovalEventData struct {
	ApprovalID   string     `json:"approval_id"`
	ExecutionID  string     `json:"execution_id"`
	WorkflowID   string     `json:"workflow_id"`
	Status       string     `json:"status"`
	Reason       string     `json:"reason,omitempty"`
	ApprovedBy   string     `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// ErrorData contains error details
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SubscriptionData contains subscription request details
type SubscriptionData struct {
	Channel    string   `json:"channel"`     // e.g., "executions", "executions:{id}", "workflows:{id}"
	Filters    Filters  `json:"filters,omitempty"`
}

// Filters for subscription
type Filters struct {
	WorkflowIDs   []string `json:"workflow_ids,omitempty"`
	ExecutionIDs  []string `json:"execution_ids,omitempty"`
	Statuses      []string `json:"statuses,omitempty"`
}

// NewMessage creates a new message with the current timestamp
func NewMessage(msgType MessageType, data interface{}) (*Message, error) {
	var rawData json.RawMessage
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		rawData = jsonData
	}

	return &Message{
		Type:      msgType,
		Timestamp: time.Now().UTC(),
		Data:      rawData,
	}, nil
}

// ToJSON converts a message to JSON bytes
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseMessage parses a JSON message
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
