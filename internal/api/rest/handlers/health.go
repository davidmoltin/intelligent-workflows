package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthHandler handles health check endpoints
type HealthHandler struct {
	logger  *logger.Logger
	db      HealthChecker
	redis   HealthChecker
	version string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(log *logger.Logger, db, redis HealthChecker, version string) *HealthHandler {
	return &HealthHandler{
		logger:  log,
		db:      db,
		redis:   redis,
		version: version,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string            `json:"status"`
	Version string            `json:"version"`
	Checks  map[string]string `json:"checks,omitempty"`
}

// Health is a simple health check endpoint
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:  "ok",
		Version: h.version,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Ready checks if the service is ready to accept traffic
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allHealthy := true

	// Check database
	if err := h.db.HealthCheck(ctx); err != nil {
		// Log detailed error internally
		h.logger.Errorf("Database health check failed: %v", err)
		checks["database"] = "unhealthy"
		allHealthy = false
	} else {
		checks["database"] = "healthy"
	}

	// Check Redis
	if err := h.redis.HealthCheck(ctx); err != nil {
		// Log detailed error internally
		h.logger.Errorf("Redis health check failed: %v", err)
		checks["redis"] = "unhealthy"
		allHealthy = false
	} else {
		checks["redis"] = "healthy"
	}

	status := "ready"
	statusCode := http.StatusOK

	if !allHealthy {
		status = "not ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:  status,
		Version: h.version,
		Checks:  checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
