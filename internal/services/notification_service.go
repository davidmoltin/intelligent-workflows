package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// NotificationChannel represents different notification channels
type NotificationChannel string

const (
	ChannelEmail NotificationChannel = "email"
	ChannelSlack NotificationChannel = "slack"
)

// NotificationService handles sending notifications via various channels
type NotificationService struct {
	config      *config.NotificationConfig
	logger      *logger.Logger
	emailClient *EmailClient
	slackClient *SlackClient
	templates   *NotificationTemplates
}

// EmailClient handles email sending
type EmailClient struct {
	smtpHost string
	smtpPort int
	username string
	password string
	from     string
}

// SlackClient handles Slack notifications
type SlackClient struct {
	webhookURL string
	enabled    bool
}

// NotificationTemplates holds parsed email templates
type NotificationTemplates struct {
	ApprovalRequest  *template.Template
	ApprovalApproved *template.Template
	ApprovalRejected *template.Template
	ApprovalExpired  *template.Template
}

// ApprovalNotificationData holds data for approval notifications
type ApprovalNotificationData struct {
	ApprovalID   string
	RequestID    string
	EntityType   string
	EntityID     string
	RequesterID  string
	ApproverRole string
	Reason       string
	ExpiresAt    string
	ApprovalURL  string
	Timestamp    string
}

// NewNotificationService creates a new notification service
func NewNotificationService(cfg *config.NotificationConfig, log *logger.Logger) (*NotificationService, error) {
	// Initialize email client if enabled
	var emailClient *EmailClient
	if cfg.Email.Enabled {
		emailClient = &EmailClient{
			smtpHost: cfg.Email.SMTPHost,
			smtpPort: cfg.Email.SMTPPort,
			username: cfg.Email.SMTPUser,
			password: cfg.Email.SMTPPassword,
			from:     cfg.Email.FromAddress,
		}
	}

	// Initialize Slack client if enabled
	var slackClient *SlackClient
	if cfg.Slack.Enabled {
		slackClient = &SlackClient{
			webhookURL: cfg.Slack.WebhookURL,
			enabled:    true,
		}
	}

	// Load templates
	templates, err := loadNotificationTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load notification templates: %w", err)
	}

	return &NotificationService{
		config:      cfg,
		logger:      log,
		emailClient: emailClient,
		slackClient: slackClient,
		templates:   templates,
	}, nil
}

// SendApprovalRequestNotification sends notification for a new approval request
func (s *NotificationService) SendApprovalRequestNotification(
	ctx context.Context,
	approval *models.ApprovalRequest,
	approverEmail string,
) error {
	s.logger.Infof("Sending approval request notification for %s", approval.RequestID)

	// Prepare notification data
	data := s.prepareApprovalData(approval)

	// Send via enabled channels
	var errors []error

	if s.config.Email.Enabled && approverEmail != "" {
		if err := s.sendApprovalEmail(ctx, approverEmail, data, s.templates.ApprovalRequest); err != nil {
			s.logger.Errorf("Failed to send email notification: %v", err)
			errors = append(errors, err)
		}
	}

	if s.config.Slack.Enabled {
		if err := s.sendApprovalSlackMessage(ctx, approval, "new"); err != nil {
			s.logger.Errorf("Failed to send Slack notification: %v", err)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("notification errors: %v", errors)
	}

	s.logger.Infof("Approval notification sent successfully for %s", approval.RequestID)
	return nil
}

// SendApprovalDecisionNotification sends notification for approval decision
func (s *NotificationService) SendApprovalDecisionNotification(
	ctx context.Context,
	approval *models.ApprovalRequest,
	requesterEmail string,
) error {
	s.logger.Infof("Sending approval decision notification for %s", approval.RequestID)

	data := s.prepareApprovalData(approval)

	var tmpl *template.Template
	var messageType string

	switch approval.Status {
	case models.ApprovalStatusApproved:
		tmpl = s.templates.ApprovalApproved
		messageType = "approved"
	case models.ApprovalStatusRejected:
		tmpl = s.templates.ApprovalRejected
		messageType = "rejected"
	case models.ApprovalStatusExpired:
		tmpl = s.templates.ApprovalExpired
		messageType = "expired"
	default:
		return fmt.Errorf("invalid approval status for notification: %s", approval.Status)
	}

	var errors []error

	if s.config.Email.Enabled && requesterEmail != "" {
		if err := s.sendApprovalEmail(ctx, requesterEmail, data, tmpl); err != nil {
			s.logger.Errorf("Failed to send email notification: %v", err)
			errors = append(errors, err)
		}
	}

	if s.config.Slack.Enabled {
		if err := s.sendApprovalSlackMessage(ctx, approval, messageType); err != nil {
			s.logger.Errorf("Failed to send Slack notification: %v", err)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("notification errors: %v", errors)
	}

	return nil
}

// prepareApprovalData prepares approval data for templates
func (s *NotificationService) prepareApprovalData(approval *models.ApprovalRequest) ApprovalNotificationData {
	data := ApprovalNotificationData{
		ApprovalID:   approval.ID.String(),
		RequestID:    approval.RequestID,
		EntityType:   approval.EntityType,
		EntityID:     approval.EntityID,
		ApproverRole: approval.ApproverRole,
		Timestamp:    approval.RequestedAt.Format(time.RFC3339),
		ApprovalURL:  fmt.Sprintf("%s/approvals/%s", s.config.BaseURL, approval.ID.String()),
	}

	if approval.Reason != nil {
		data.Reason = *approval.Reason
	}

	if approval.RequesterID != nil {
		data.RequesterID = approval.RequesterID.String()
	}

	if approval.ExpiresAt != nil {
		data.ExpiresAt = approval.ExpiresAt.Format(time.RFC3339)
	}

	return data
}

// sendApprovalEmail sends an email notification
func (s *NotificationService) sendApprovalEmail(
	ctx context.Context,
	to string,
	data ApprovalNotificationData,
	tmpl *template.Template,
) error {
	if s.emailClient == nil {
		return fmt.Errorf("email client not configured")
	}

	// Render template
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	// Prepare email
	subject := fmt.Sprintf("Approval Request: %s", data.RequestID)
	if tmpl == s.templates.ApprovalApproved {
		subject = fmt.Sprintf("Approved: %s", data.RequestID)
	} else if tmpl == s.templates.ApprovalRejected {
		subject = fmt.Sprintf("Rejected: %s", data.RequestID)
	} else if tmpl == s.templates.ApprovalExpired {
		subject = fmt.Sprintf("Expired: %s", data.RequestID)
	}

	message := fmt.Sprintf("From: %s\r\n", s.emailClient.from)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n"
	message += body.String()

	// Send email
	auth := smtp.PlainAuth("", s.emailClient.username, s.emailClient.password, s.emailClient.smtpHost)
	addr := fmt.Sprintf("%s:%d", s.emailClient.smtpHost, s.emailClient.smtpPort)

	if err := smtp.SendMail(addr, auth, s.emailClient.from, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Infof("Email sent successfully to %s", to)
	return nil
}

// sendApprovalSlackMessage sends a Slack notification
func (s *NotificationService) sendApprovalSlackMessage(
	ctx context.Context,
	approval *models.ApprovalRequest,
	messageType string,
) error {
	if s.slackClient == nil || !s.slackClient.enabled {
		return fmt.Errorf("slack client not configured")
	}

	// Prepare Slack message
	var color, title, text string

	switch messageType {
	case "new":
		color = "#FFEB3B" // Yellow
		title = fmt.Sprintf("🔔 New Approval Request: %s", approval.RequestID)
		text = fmt.Sprintf("*Entity:* %s/%s\n*Approver Role:* %s", approval.EntityType, approval.EntityID, approval.ApproverRole)
	case "approved":
		color = "#4CAF50" // Green
		title = fmt.Sprintf("✅ Approval Approved: %s", approval.RequestID)
		text = fmt.Sprintf("*Entity:* %s/%s", approval.EntityType, approval.EntityID)
	case "rejected":
		color = "#F44336" // Red
		title = fmt.Sprintf("❌ Approval Rejected: %s", approval.RequestID)
		text = fmt.Sprintf("*Entity:* %s/%s", approval.EntityType, approval.EntityID)
	case "expired":
		color = "#9E9E9E" // Grey
		title = fmt.Sprintf("⏰ Approval Expired: %s", approval.RequestID)
		text = fmt.Sprintf("*Entity:* %s/%s", approval.EntityType, approval.EntityID)
	}

	if approval.Reason != nil && *approval.Reason != "" {
		text += fmt.Sprintf("\n*Reason:* %s", *approval.Reason)
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"title":  title,
				"text":   text,
				"footer": "Intelligent Workflows",
				"ts":     approval.RequestedAt.Unix(),
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	// Send HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.slackClient.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned non-200 status: %d", resp.StatusCode)
	}

	s.logger.Infof("Slack message sent successfully")
	return nil
}

// loadNotificationTemplates loads email templates
func loadNotificationTemplates() (*NotificationTemplates, error) {
	approvalRequestTmpl, err := template.New("approval_request").Parse(approvalRequestEmailTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse approval request template: %w", err)
	}

	approvalApprovedTmpl, err := template.New("approval_approved").Parse(approvalApprovedEmailTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse approval approved template: %w", err)
	}

	approvalRejectedTmpl, err := template.New("approval_rejected").Parse(approvalRejectedEmailTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse approval rejected template: %w", err)
	}

	approvalExpiredTmpl, err := template.New("approval_expired").Parse(approvalExpiredEmailTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse approval expired template: %w", err)
	}

	return &NotificationTemplates{
		ApprovalRequest:  approvalRequestTmpl,
		ApprovalApproved: approvalApprovedTmpl,
		ApprovalRejected: approvalRejectedTmpl,
		ApprovalExpired:  approvalExpiredTmpl,
	}, nil
}
