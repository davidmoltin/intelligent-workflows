package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// WorkflowHandler handles workflow-related HTTP requests
type WorkflowHandler struct {
	logger *logger.Logger
	repo   *postgres.WorkflowRepository
}

// NewWorkflowHandler creates a new workflow handler
func NewWorkflowHandler(log *logger.Logger, repo *postgres.WorkflowRepository) *WorkflowHandler {
	return &WorkflowHandler{
		logger: log,
		repo:   repo,
	}
}

// Create creates a new workflow
func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	workflow, err := h.repo.Create(r.Context(), &req, nil)
	if err != nil {
		h.logger.Error("Failed to create workflow", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to create workflow")
		return
	}

	h.respondJSON(w, http.StatusCreated, workflow)
}

// Get retrieves a workflow by ID
func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid workflow ID")
		return
	}

	workflow, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get workflow", logger.Err(err))
		h.respondError(w, http.StatusNotFound, "Workflow not found")
		return
	}

	h.respondJSON(w, http.StatusOK, workflow)
}

// List retrieves a list of workflows
func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	enabledStr := r.URL.Query().Get("enabled")

	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var enabled *bool
	if enabledStr != "" {
		if e, err := strconv.ParseBool(enabledStr); err == nil {
			enabled = &e
		}
	}

	workflows, total, err := h.repo.List(r.Context(), enabled, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list workflows", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to list workflows")
		return
	}

	page := offset/limit + 1
	response := map[string]interface{}{
		"workflows": workflows,
		"total":     total,
		"page":      page,
		"page_size": limit,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// Update updates a workflow
func (h *WorkflowHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid workflow ID")
		return
	}

	var req models.UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	workflow, err := h.repo.Update(r.Context(), id, &req)
	if err != nil {
		h.logger.Error("Failed to update workflow", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to update workflow")
		return
	}

	h.respondJSON(w, http.StatusOK, workflow)
}

// Delete deletes a workflow
func (h *WorkflowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid workflow ID")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete workflow", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to delete workflow")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Enable enables a workflow
func (h *WorkflowHandler) Enable(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid workflow ID")
		return
	}

	if err := h.repo.SetEnabled(r.Context(), id, true); err != nil {
		h.logger.Error("Failed to enable workflow", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to enable workflow")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "Workflow enabled"})
}

// Disable disables a workflow
func (h *WorkflowHandler) Disable(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid workflow ID")
		return
	}

	if err := h.repo.SetEnabled(r.Context(), id, false); err != nil {
		h.logger.Error("Failed to disable workflow", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to disable workflow")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "Workflow disabled"})
}

// Helper methods

func (h *WorkflowHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *WorkflowHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
