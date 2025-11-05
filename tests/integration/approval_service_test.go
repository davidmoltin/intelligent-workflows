package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

func TestApprovalService_CreateApprovalRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	approvalRepo := postgres.NewApprovalRepository(suite.DB.DB)

	// Create notification service with disabled channels for testing
	notificationCfg := &config.NotificationConfig{
		BaseURL: "http://localhost:8080",
		Email: config.EmailConfig{
			Enabled: false,
		},
		Slack: config.SlackConfig{
			Enabled: false,
		},
	}
	notificationSvc, err := services.NewNotificationService(notificationCfg, log)
	require.NoError(t, err)

	workflowResumer := services.NewWorkflowResumer(log, nil, nil)

	approvalService := services.NewApprovalService(approvalRepo, log, notificationSvc, workflowResumer)

	ctx := context.Background()
	executionID := uuid.New()
	requesterID := uuid.New()
	expiresIn := 24 * time.Hour

	approval, err := approvalService.CreateApprovalRequest(
		ctx,
		executionID,
		"deployment",
		"prod-123",
		&requesterID,
		"admin",
		"Production deployment requires approval",
		&expiresIn,
	)

	require.NoError(t, err)
	assert.NotNil(t, approval)
	assert.NotEqual(t, uuid.Nil, approval.ID)
	assert.NotEmpty(t, approval.RequestID)
	assert.Equal(t, executionID, approval.ExecutionID)
	assert.Equal(t, "deployment", approval.EntityType)
	assert.Equal(t, "prod-123", approval.EntityID)
	assert.Equal(t, models.ApprovalStatusPending, approval.Status)
	assert.NotNil(t, approval.ExpiresAt)
}

func TestApprovalService_ApproveRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	approvalRepo := postgres.NewApprovalRepository(suite.DB.DB)

	notificationCfg := &config.NotificationConfig{
		BaseURL: "http://localhost:8080",
		Email: config.EmailConfig{
			Enabled: false,
		},
		Slack: config.SlackConfig{
			Enabled: false,
		},
	}
	notificationSvc, err := services.NewNotificationService(notificationCfg, log)
	require.NoError(t, err)

	workflowResumer := services.NewWorkflowResumer(log, nil, nil)

	approvalService := services.NewApprovalService(approvalRepo, log, notificationSvc, workflowResumer)

	ctx := context.Background()

	// Create approval request
	executionID := uuid.New()
	requesterID := uuid.New()
	expiresIn := 24 * time.Hour

	approval, err := approvalService.CreateApprovalRequest(
		ctx,
		executionID,
		"deployment",
		"prod-123",
		&requesterID,
		"admin",
		"Production deployment requires approval",
		&expiresIn,
	)
	require.NoError(t, err)

	// Approve the request
	approverID := uuid.New()
	reason := "Looks good to me"

	approvedApproval, err := approvalService.ApproveRequest(ctx, approval.ID, approverID, &reason)

	require.NoError(t, err)
	assert.NotNil(t, approvedApproval)
	assert.Equal(t, models.ApprovalStatusApproved, approvedApproval.Status)
	assert.Equal(t, approverID, *approvedApproval.ApproverID)
	assert.Equal(t, reason, *approvedApproval.DecisionReason)
	assert.NotNil(t, approvedApproval.DecidedAt)
}

func TestApprovalService_RejectRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	approvalRepo := postgres.NewApprovalRepository(suite.DB.DB)

	notificationCfg := &config.NotificationConfig{
		BaseURL: "http://localhost:8080",
		Email: config.EmailConfig{
			Enabled: false,
		},
		Slack: config.SlackConfig{
			Enabled: false,
		},
	}
	notificationSvc, err := services.NewNotificationService(notificationCfg, log)
	require.NoError(t, err)

	workflowResumer := services.NewWorkflowResumer(log, nil, nil)

	approvalService := services.NewApprovalService(approvalRepo, log, notificationSvc, workflowResumer)

	ctx := context.Background()

	// Create approval request
	executionID := uuid.New()
	requesterID := uuid.New()
	expiresIn := 24 * time.Hour

	approval, err := approvalService.CreateApprovalRequest(
		ctx,
		executionID,
		"deployment",
		"prod-123",
		&requesterID,
		"admin",
		"Production deployment requires approval",
		&expiresIn,
	)
	require.NoError(t, err)

	// Reject the request
	approverID := uuid.New()
	reason := "Not ready for production"

	rejectedApproval, err := approvalService.RejectRequest(ctx, approval.ID, approverID, &reason)

	require.NoError(t, err)
	assert.NotNil(t, rejectedApproval)
	assert.Equal(t, models.ApprovalStatusRejected, rejectedApproval.Status)
	assert.Equal(t, approverID, *rejectedApproval.ApproverID)
	assert.Equal(t, reason, *rejectedApproval.DecisionReason)
	assert.NotNil(t, rejectedApproval.DecidedAt)
}

func TestApprovalService_ExpireOldApprovals(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	approvalRepo := postgres.NewApprovalRepository(suite.DB.DB)

	notificationCfg := &config.NotificationConfig{
		BaseURL: "http://localhost:8080",
		Email: config.EmailConfig{
			Enabled: false,
		},
		Slack: config.SlackConfig{
			Enabled: false,
		},
	}
	notificationSvc, err := services.NewNotificationService(notificationCfg, log)
	require.NoError(t, err)

	workflowResumer := services.NewWorkflowResumer(log, nil, nil)

	approvalService := services.NewApprovalService(approvalRepo, log, notificationSvc, workflowResumer)

	ctx := context.Background()

	// Create an approval that's already expired
	executionID := uuid.New()
	requesterID := uuid.New()
	expiresIn := -1 * time.Hour // Expired 1 hour ago

	approval, err := approvalService.CreateApprovalRequest(
		ctx,
		executionID,
		"deployment",
		"prod-123",
		&requesterID,
		"admin",
		"Production deployment requires approval",
		&expiresIn,
	)
	require.NoError(t, err)

	// Run expiration check
	err = approvalService.ExpireOldApprovals(ctx)
	require.NoError(t, err)

	// Verify the approval was marked as expired
	expiredApproval, err := approvalService.GetApproval(ctx, approval.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ApprovalStatusExpired, expiredApproval.Status)
	assert.NotNil(t, expiredApproval.DecidedAt)
}

func TestApprovalService_ListPendingApprovals(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	suite := SetupSuite(t)
	defer TeardownSuite(t)
	suite.ResetDatabase(t)

	log, err := logger.New("info", "json")
	require.NoError(t, err)

	approvalRepo := postgres.NewApprovalRepository(suite.DB.DB)

	notificationCfg := &config.NotificationConfig{
		BaseURL: "http://localhost:8080",
		Email: config.EmailConfig{
			Enabled: false,
		},
		Slack: config.SlackConfig{
			Enabled: false,
		},
	}
	notificationSvc, err := services.NewNotificationService(notificationCfg, log)
	require.NoError(t, err)

	workflowResumer := services.NewWorkflowResumer(log, nil, nil)

	approvalService := services.NewApprovalService(approvalRepo, log, notificationSvc, workflowResumer)

	ctx := context.Background()

	// Create multiple approval requests
	for i := 0; i < 3; i++ {
		executionID := uuid.New()
		requesterID := uuid.New()
		expiresIn := 24 * time.Hour

		_, err := approvalService.CreateApprovalRequest(
			ctx,
			executionID,
			"deployment",
			"prod-123",
			&requesterID,
			"admin",
			"Production deployment requires approval",
			&expiresIn,
		)
		require.NoError(t, err)
	}

	// List pending approvals
	approvals, total, err := approvalService.ListPendingApprovals(ctx, nil, 10, 0)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(approvals), 3)
	assert.GreaterOrEqual(t, int(total), 3)

	for _, approval := range approvals {
		assert.Equal(t, models.ApprovalStatusPending, approval.Status)
	}
}
