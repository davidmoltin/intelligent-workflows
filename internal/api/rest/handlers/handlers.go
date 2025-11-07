package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/davidmoltin/intelligent-workflows/internal/engine"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/internal/websocket"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// Handlers aggregates all HTTP handlers
type Handlers struct {
	Health       *HealthHandler
	Workflow     *WorkflowHandler
	Event        *EventHandler
	Execution    *ExecutionHandler
	Approval     *ApprovalHandler
	Auth         *AuthHandler
	Organization *OrganizationHandler
	Docs         *DocsHandler
	AI           *AIHandler
	Analytics    *AnalyticsHandler
	Schedule     *ScheduleHandler
	Audit        *AuditHandler
	WebSocket    *websocket.Handler
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
	analyticsRepo *postgres.AnalyticsRepository,
	organizationRepo *postgres.OrganizationRepository,
	eventRouter *engine.EventRouter,
	approvalService *services.ApprovalService,
	authService *services.AuthService,
	scheduleService ScheduleService,
	workflowResumer *services.WorkflowResumerImpl,
	aiService *services.AIService,
	wsHub *websocket.Hub,
	auditService *services.AuditService,
	healthCheckers *HealthCheckers,
	version string,
) *Handlers {
	// Handle AI handler initialization
	var aiHandler *AIHandler
	if aiService != nil {
		aiHandler = NewAIHandler(aiService, log.Logger)
	}

	// Handle Audit handler initialization
	var auditHandler *AuditHandler
	if auditService != nil {
		auditHandler = NewAuditHandler(log, auditService)
	}

	return &Handlers{
		Health:       NewHealthHandler(log, healthCheckers.DB, healthCheckers.Redis, version),
		Workflow:     NewWorkflowHandler(log, workflowRepo, auditService),
		Event:        NewEventHandler(log, eventRouter),
		Execution:    NewExecutionHandler(log, executionRepo, workflowResumer),
		Approval:     NewApprovalHandler(log, approvalService),
		Auth:         NewAuthHandler(log, authService),
		Organization: NewOrganizationHandler(log, organizationRepo),
		Docs:         NewDocsHandler(),
		AI:           aiHandler,
		Analytics:    NewAnalyticsHandler(log, analyticsRepo),
		Schedule:     NewScheduleHandler(log, scheduleService),
		Audit:        auditHandler,
		WebSocket:    websocket.NewHandler(wsHub, log.Logger),
	}
}

// Common error types for safe error handling
var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("resource already exists")
	ErrInternalError      = errors.New("internal server error")
	ErrServiceUnavailable = errors.New("service unavailable")
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// RespondError sends a safe error response without leaking internal details
func RespondError(w http.ResponseWriter, statusCode int, publicMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: publicMsg,
	})
}

// RespondJSON sends a JSON response
func RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
