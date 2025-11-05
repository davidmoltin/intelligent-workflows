package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

func TestNewNotificationService(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	cfg := &config.NotificationConfig{
		BaseURL: "http://localhost:8080",
		Email: config.EmailConfig{
			Enabled:      false,
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     587,
			SMTPUser:     "test@example.com",
			SMTPPassword: "password",
			FromAddress:  "noreply@example.com",
		},
		Slack: config.SlackConfig{
			Enabled:    false,
			WebhookURL: "https://hooks.slack.com/test",
		},
	}

	svc, err := NewNotificationService(cfg, log)

	require.NoError(t, err)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.templates)
	assert.NotNil(t, svc.templates.ApprovalRequest)
	assert.NotNil(t, svc.templates.ApprovalApproved)
	assert.NotNil(t, svc.templates.ApprovalRejected)
	assert.NotNil(t, svc.templates.ApprovalExpired)
}

func TestLoadNotificationTemplates(t *testing.T) {
	templates, err := loadNotificationTemplates()

	require.NoError(t, err)
	assert.NotNil(t, templates)
	assert.NotNil(t, templates.ApprovalRequest)
	assert.NotNil(t, templates.ApprovalApproved)
	assert.NotNil(t, templates.ApprovalRejected)
	assert.NotNil(t, templates.ApprovalExpired)
}

func TestPrepareApprovalData(t *testing.T) {
	log, err := logger.New("info", "json")
	require.NoError(t, err)

	cfg := &config.NotificationConfig{
		BaseURL: "http://localhost:8080",
		Email: config.EmailConfig{
			Enabled: false,
		},
		Slack: config.SlackConfig{
			Enabled: false,
		},
	}

	svc, err := NewNotificationService(cfg, log)
	require.NoError(t, err)

	requesterID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)
	reason := "Test reason"

	approval := &models.ApprovalRequest{
		ID:           uuid.New(),
		RequestID:    "appr_12345678",
		ExecutionID:  uuid.New(),
		EntityType:   "deployment",
		EntityID:     "prod-123",
		RequesterID:  &requesterID,
		ApproverRole: "admin",
		Status:       models.ApprovalStatusPending,
		Reason:       &reason,
		RequestedAt:  time.Now(),
		ExpiresAt:    &expiresAt,
	}

	data := svc.prepareApprovalData(approval)

	assert.Equal(t, approval.ID.String(), data.ApprovalID)
	assert.Equal(t, approval.RequestID, data.RequestID)
	assert.Equal(t, approval.EntityType, data.EntityType)
	assert.Equal(t, approval.EntityID, data.EntityID)
	assert.Equal(t, approval.ApproverRole, data.ApproverRole)
	assert.Equal(t, *approval.Reason, data.Reason)
	assert.Equal(t, requesterID.String(), data.RequesterID)
	assert.NotEmpty(t, data.ExpiresAt)
	assert.Contains(t, data.ApprovalURL, approval.ID.String())
}

func TestApprovalRequestEmailTemplate(t *testing.T) {
	templates, err := loadNotificationTemplates()
	require.NoError(t, err)

	data := ApprovalNotificationData{
		ApprovalID:   uuid.New().String(),
		RequestID:    "appr_12345678",
		EntityType:   "deployment",
		EntityID:     "prod-123",
		ApproverRole: "admin",
		Reason:       "Production deployment requires approval",
		ExpiresAt:    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		ApprovalURL:  "http://localhost:8080/approvals/123",
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	var result string
	err = templates.ApprovalRequest.Execute(&testWriter{output: &result}, data)

	require.NoError(t, err)
	assert.Contains(t, result, data.RequestID)
	assert.Contains(t, result, data.EntityType)
	assert.Contains(t, result, data.EntityID)
	assert.Contains(t, result, data.ApproverRole)
	assert.Contains(t, result, data.Reason)
	assert.Contains(t, result, "Approval Request")
}

func TestApprovalApprovedEmailTemplate(t *testing.T) {
	templates, err := loadNotificationTemplates()
	require.NoError(t, err)

	data := ApprovalNotificationData{
		ApprovalID:   uuid.New().String(),
		RequestID:    "appr_12345678",
		EntityType:   "deployment",
		EntityID:     "prod-123",
		ApproverRole: "admin",
		Reason:       "Production deployment requires approval",
		ApprovalURL:  "http://localhost:8080/approvals/123",
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	var result string
	err = templates.ApprovalApproved.Execute(&testWriter{output: &result}, data)

	require.NoError(t, err)
	assert.Contains(t, result, data.RequestID)
	assert.Contains(t, result, data.EntityType)
	assert.Contains(t, result, "Approval Approved")
}

func TestApprovalRejectedEmailTemplate(t *testing.T) {
	templates, err := loadNotificationTemplates()
	require.NoError(t, err)

	data := ApprovalNotificationData{
		ApprovalID:   uuid.New().String(),
		RequestID:    "appr_12345678",
		EntityType:   "deployment",
		EntityID:     "prod-123",
		ApproverRole: "admin",
		Reason:       "Production deployment requires approval",
		ApprovalURL:  "http://localhost:8080/approvals/123",
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	var result string
	err = templates.ApprovalRejected.Execute(&testWriter{output: &result}, data)

	require.NoError(t, err)
	assert.Contains(t, result, data.RequestID)
	assert.Contains(t, result, data.EntityType)
	assert.Contains(t, result, "Approval Rejected")
}

func TestApprovalExpiredEmailTemplate(t *testing.T) {
	templates, err := loadNotificationTemplates()
	require.NoError(t, err)

	data := ApprovalNotificationData{
		ApprovalID:   uuid.New().String(),
		RequestID:    "appr_12345678",
		EntityType:   "deployment",
		EntityID:     "prod-123",
		ApproverRole: "admin",
		Reason:       "Production deployment requires approval",
		ExpiresAt:    time.Now().Format(time.RFC3339),
		ApprovalURL:  "http://localhost:8080/approvals/123",
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	var result string
	err = templates.ApprovalExpired.Execute(&testWriter{output: &result}, data)

	require.NoError(t, err)
	assert.Contains(t, result, data.RequestID)
	assert.Contains(t, result, data.EntityType)
	assert.Contains(t, result, "Approval Expired")
}

// testWriter is a helper to capture template execution output
type testWriter struct {
	output *string
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	*w.output += string(p)
	return len(p), nil
}
