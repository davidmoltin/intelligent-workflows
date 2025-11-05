package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// ExecutionHandler handles execution-related HTTP requests
type ExecutionHandler struct {
	logger        *logger.Logger
	executionRepo *postgres.ExecutionRepository
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(log *logger.Logger, executionRepo *postgres.ExecutionRepository) *ExecutionHandler {
	return &ExecutionHandler{
		logger:        log,
		executionRepo: executionRepo,
	}
}

// ListExecutions handles GET /api/v1/executions
func (h *ExecutionHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	workflowIDStr := r.URL.Query().Get("workflow_id")
	statusStr := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Parse workflow_id
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		id, err := uuid.Parse(workflowIDStr)
		if err != nil {
			http.Error(w, "Invalid workflow_id", http.StatusBadRequest)
			return
		}
		workflowID = &id
	}

	// Parse status
	var status *models.ExecutionStatus
	if statusStr != "" {
		s := models.ExecutionStatus(statusStr)
		status = &s
	}

	// Parse pagination
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get executions
	executions, total, err := h.executionRepo.ListExecutions(r.Context(), workflowID, status, limit, offset)
	if err != nil {
		h.logger.Errorf("Failed to list executions: %v", err)
		http.Error(w, "Failed to retrieve executions", http.StatusInternalServerError)
		return
	}

	// Calculate pagination
	page := offset / limit

	response := models.ExecutionListResponse{
		Executions: executions,
		Total:      total,
		Page:       page,
		PageSize:   limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetExecution handles GET /api/v1/executions/:id
func (h *ExecutionHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	execution, err := h.executionRepo.GetExecutionByID(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get execution: %v", err)
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// GetExecutionTrace handles GET /api/v1/executions/:id/trace
func (h *ExecutionHandler) GetExecutionTrace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	trace, err := h.executionRepo.GetExecutionTrace(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get execution trace: %v", err)
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trace)
}
