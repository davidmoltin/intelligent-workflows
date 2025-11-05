package handlers

import (
	"github.com/davidmoltin/intelligent-workflows/internal/engine"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// Handlers aggregates all HTTP handlers
type Handlers struct {
	Health    *HealthHandler
	Workflow  *WorkflowHandler
	Event     *EventHandler
	Execution *ExecutionHandler
	Approval  *ApprovalHandler
	Auth      *AuthHandler
	AI        *AIHandler
}

// HealthCheckers holds all health check dependencies
type HealthCheckers struct {
	DB    HealthChecker
	Redis HealthChecker
}

// NewHandlers creates a new handlers instance
func NewHandlers(
	log *logger.Logger,
	workflowRepo *postgres.WorkflowRepository,
	executionRepo *postgres.ExecutionRepository,
	eventRouter *engine.EventRouter,
	approvalService *services.ApprovalService,
	authService *services.AuthService,
	aiService *services.AIService,
	healthCheckers *HealthCheckers,
) *Handlers {
	// Handle AI handler initialization
	var aiHandler *AIHandler
	if aiService != nil {
		aiHandler = NewAIHandler(aiService, log.Logger)
	}

	return &Handlers{
		Health:    NewHealthHandler(log, healthCheckers.DB, healthCheckers.Redis),
		Workflow:  NewWorkflowHandler(log, workflowRepo),
		Event:     NewEventHandler(log, eventRouter),
		Execution: NewExecutionHandler(log, executionRepo),
		Approval:  NewApprovalHandler(log, approvalService),
		Auth:      NewAuthHandler(log, authService),
		AI:        aiHandler,
	}
}
