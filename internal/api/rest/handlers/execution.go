package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ExecutionHandler handles execution-related HTTP requests
type ExecutionHandler struct {
	logger          *logger.Logger
	executionRepo   *postgres.ExecutionRepository
	workflowResumer *services.WorkflowResumerImpl
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(log *logger.Logger, executionRepo *postgres.ExecutionRepository, workflowResumer *services.WorkflowResumerImpl) *ExecutionHandler {
	return &ExecutionHandler{
		logger:          log,
		executionRepo:   executionRepo,
		workflowResumer: workflowResumer,
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

// PauseExecution handles POST /api/v1/executions/:id/pause
func (h *ExecutionHandler) PauseExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid execution ID")
		return
	}

	// Parse request body
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Reason == "" {
		req.Reason = "Manually paused via API"
	}

	// Pause the execution
	if err := h.workflowResumer.PauseExecution(r.Context(), id, req.Reason, nil); err != nil {
		h.logger.Errorf("Failed to pause execution %s: %v", id, err)
		RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pause execution: %v", err))
		return
	}

	// Get updated execution
	execution, err := h.executionRepo.GetExecutionByID(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get execution after pause: %v", err)
		RespondError(w, http.StatusInternalServerError, "Execution paused but failed to retrieve updated state")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(execution)
}

// ResumeExecution handles POST /api/v1/executions/:id/resume
func (h *ExecutionHandler) ResumeExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid execution ID")
		return
	}

	// Parse request body (optional resume data)
	var req struct {
		ResumeData map[string]interface{} `json:"resume_data,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req.ResumeData = make(map[string]interface{})
	}

	// Resume the execution
	if len(req.ResumeData) > 0 {
		if err := h.workflowResumer.ResumeExecution(r.Context(), id, req.ResumeData); err != nil {
			h.logger.Errorf("Failed to resume execution %s: %v", id, err)
			RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume execution: %v", err))
			return
		}
	} else {
		// Use backward-compatible ResumeWorkflow with approved=true as default
		if err := h.workflowResumer.ResumeWorkflow(r.Context(), id, true); err != nil {
			h.logger.Errorf("Failed to resume execution %s: %v", id, err)
			RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume execution: %v", err))
			return
		}
	}

	// Get updated execution
	execution, err := h.executionRepo.GetExecutionByID(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get execution after resume: %v", err)
		RespondError(w, http.StatusInternalServerError, "Execution resumed but failed to retrieve updated state")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(execution)
}

// ListPausedExecutions handles GET /api/v1/executions/paused
func (h *ExecutionHandler) ListPausedExecutions(w http.ResponseWriter, r *http.Request) {
	// Parse limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get paused executions
	executions, err := h.workflowResumer.GetPausedExecutions(r.Context(), limit)
	if err != nil {
		h.logger.Errorf("Failed to list paused executions: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to retrieve paused executions")
		return
	}

	response := struct {
		Executions []*models.WorkflowExecution `json:"executions"`
		Count      int                         `json:"count"`
	}{
		Executions: executions,
		Count:      len(executions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
