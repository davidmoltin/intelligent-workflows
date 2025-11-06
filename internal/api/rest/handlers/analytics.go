package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/middleware"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
)

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	logger        *logger.Logger
	analyticsRepo *postgres.AnalyticsRepository
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(log *logger.Logger, analyticsRepo *postgres.AnalyticsRepository) *AnalyticsHandler {
	return &AnalyticsHandler{
		logger:        log,
		analyticsRepo: analyticsRepo,
	}
}

// GetDashboard handles GET /api/v1/analytics/dashboard
func (h *AnalyticsHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		http.Error(w, "Organization context required", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	workflowIDStr := r.URL.Query().Get("workflow_id")
	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "24h"
	}

	// Parse workflow_id if provided
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		id, err := uuid.Parse(workflowIDStr)
		if err != nil {
			http.Error(w, "Invalid workflow_id", http.StatusBadRequest)
			return
		}
		workflowID = &id
	}

	// Get execution stats
	stats, err := h.analyticsRepo.GetExecutionStats(r.Context(), organizationID, workflowID, timeRange)
	if err != nil {
		h.logger.Errorf("Failed to get execution stats: %v", err)
		http.Error(w, "Failed to retrieve execution stats", http.StatusInternalServerError)
		return
	}

	// Get execution trends
	trends, err := h.analyticsRepo.GetExecutionTrends(r.Context(), organizationID, workflowID, timeRange, "")
	if err != nil {
		h.logger.Errorf("Failed to get execution trends: %v", err)
		http.Error(w, "Failed to retrieve execution trends", http.StatusInternalServerError)
		return
	}

	// Get workflow stats (only if not filtering by specific workflow)
	var workflowStats []models.WorkflowStats
	if workflowID == nil {
		workflowStats, err = h.analyticsRepo.GetWorkflowStats(r.Context(), organizationID, timeRange, 10)
		if err != nil {
			h.logger.Errorf("Failed to get workflow stats: %v", err)
			// Don't fail the entire request for this
			workflowStats = []models.WorkflowStats{}
		}
	}

	// Get recent errors
	recentErrors, err := h.analyticsRepo.GetRecentErrors(r.Context(), organizationID, workflowID, 10)
	if err != nil {
		h.logger.Errorf("Failed to get recent errors: %v", err)
		// Don't fail the entire request for this
		recentErrors = []models.ExecutionError{}
	}

	// Get step stats if filtering by workflow
	var stepStats []models.StepStats
	if workflowID != nil {
		stepStats, err = h.analyticsRepo.GetStepStats(r.Context(), organizationID, workflowID, timeRange)
		if err != nil {
			h.logger.Errorf("Failed to get step stats: %v", err)
			// Don't fail the entire request for this
			stepStats = []models.StepStats{}
		}
	}

	dashboard := &models.AnalyticsDashboard{
		Stats:         stats,
		Trends:        trends,
		WorkflowStats: workflowStats,
		RecentErrors:  recentErrors,
		StepStats:     stepStats,
		TimeRange:     timeRange,
		GeneratedAt:   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// GetExecutionStats handles GET /api/v1/analytics/stats
func (h *AnalyticsHandler) GetExecutionStats(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		http.Error(w, "Organization context required", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	workflowIDStr := r.URL.Query().Get("workflow_id")
	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "24h"
	}

	// Parse workflow_id if provided
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		id, err := uuid.Parse(workflowIDStr)
		if err != nil {
			http.Error(w, "Invalid workflow_id", http.StatusBadRequest)
			return
		}
		workflowID = &id
	}

	stats, err := h.analyticsRepo.GetExecutionStats(r.Context(), organizationID, workflowID, timeRange)
	if err != nil {
		h.logger.Errorf("Failed to get execution stats: %v", err)
		http.Error(w, "Failed to retrieve execution stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetExecutionTrends handles GET /api/v1/analytics/trends
func (h *AnalyticsHandler) GetExecutionTrends(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		http.Error(w, "Organization context required", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	workflowIDStr := r.URL.Query().Get("workflow_id")
	timeRange := r.URL.Query().Get("time_range")
	interval := r.URL.Query().Get("interval")
	if timeRange == "" {
		timeRange = "24h"
	}

	// Parse workflow_id if provided
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		id, err := uuid.Parse(workflowIDStr)
		if err != nil {
			http.Error(w, "Invalid workflow_id", http.StatusBadRequest)
			return
		}
		workflowID = &id
	}

	trends, err := h.analyticsRepo.GetExecutionTrends(r.Context(), organizationID, workflowID, timeRange, interval)
	if err != nil {
		h.logger.Errorf("Failed to get execution trends: %v", err)
		http.Error(w, "Failed to retrieve execution trends", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trends)
}

// GetWorkflowStats handles GET /api/v1/analytics/workflows
func (h *AnalyticsHandler) GetWorkflowStats(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		http.Error(w, "Organization context required", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "24h"
	}

	stats, err := h.analyticsRepo.GetWorkflowStats(r.Context(), organizationID, timeRange, 20)
	if err != nil {
		h.logger.Errorf("Failed to get workflow stats: %v", err)
		http.Error(w, "Failed to retrieve workflow stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetRecentErrors handles GET /api/v1/analytics/errors
func (h *AnalyticsHandler) GetRecentErrors(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		http.Error(w, "Organization context required", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	workflowIDStr := r.URL.Query().Get("workflow_id")

	// Parse workflow_id if provided
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		id, err := uuid.Parse(workflowIDStr)
		if err != nil {
			http.Error(w, "Invalid workflow_id", http.StatusBadRequest)
			return
		}
		workflowID = &id
	}

	errors, err := h.analyticsRepo.GetRecentErrors(r.Context(), organizationID, workflowID, 20)
	if err != nil {
		h.logger.Errorf("Failed to get recent errors: %v", err)
		http.Error(w, "Failed to retrieve recent errors", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(errors)
}

// GetStepStats handles GET /api/v1/analytics/steps
func (h *AnalyticsHandler) GetStepStats(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		http.Error(w, "Organization context required", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	workflowIDStr := r.URL.Query().Get("workflow_id")
	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "24h"
	}

	// Parse workflow_id if provided
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		id, err := uuid.Parse(workflowIDStr)
		if err != nil {
			http.Error(w, "Invalid workflow_id", http.StatusBadRequest)
			return
		}
		workflowID = &id
	}

	stats, err := h.analyticsRepo.GetStepStats(r.Context(), organizationID, workflowID, timeRange)
	if err != nil {
		h.logger.Errorf("Failed to get step stats: %v", err)
		http.Error(w, "Failed to retrieve step stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
