package handlers

import (
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// Handlers aggregates all HTTP handlers
type Handlers struct {
	Health   *HealthHandler
	Workflow *WorkflowHandler
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
	healthCheckers *HealthCheckers,
) *Handlers {
	return &Handlers{
		Health:   NewHealthHandler(log, healthCheckers.DB, healthCheckers.Redis),
		Workflow: NewWorkflowHandler(log, workflowRepo),
	}
}
