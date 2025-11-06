package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/middleware"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/davidmoltin/intelligent-workflows/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// WorkflowHandler handles workflow-related HTTP requests
type WorkflowHandler struct {
	logger       *logger.Logger
	repo         *postgres.WorkflowRepository
	auditService AuditService
}

// AuditService defines interface for audit logging
type AuditService interface {
	LogWorkflowCreated(ctx context.Context, workflowID uuid.UUID, actorID uuid.UUID, actorType string, workflowData map[string]interface{}) error
	LogWorkflowUpdated(ctx context.Context, workflowID uuid.UUID, actorID uuid.UUID, actorType string, changes map[string]interface{}) error
	LogWorkflowDeleted(ctx context.Context, workflowID uuid.UUID, actorID uuid.UUID, actorType string) error
}

// NewWorkflowHandler creates a new workflow handler
func NewWorkflowHandler(log *logger.Logger, repo *postgres.WorkflowRepository, auditService AuditService) *WorkflowHandler {
	return &WorkflowHandler{
		logger:       log,
		repo:         repo,
		auditService: auditService,
	}
}

// Create creates a new workflow
func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "Organization context required")
		return
	}

	workflow, err := h.repo.Create(r.Context(), organizationID, &req, nil)
	if err != nil {
		h.logger.Errorf("Failed to create workflow", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to create workflow")
		return
	}

	// Log audit event
	if h.auditService != nil {
		actorID, actorType := h.getActorFromContext(r)
		workflowData := map[string]interface{}{
			"name":        workflow.Name,
			"description": workflow.Description,
			"enabled":     workflow.Enabled,
		}
		if err := h.auditService.LogWorkflowCreated(r.Context(), workflow.ID, actorID, actorType, workflowData); err != nil {
			h.logger.Errorf("Failed to log audit event: %v", err)
		}
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

	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "Organization context required")
		return
	}

	workflow, err := h.repo.GetByID(r.Context(), organizationID, id)
	if err != nil {
		h.logger.Errorf("Failed to get workflow", logger.Err(err))
		h.respondError(w, http.StatusNotFound, "Workflow not found")
		return
	}

	h.respondJSON(w, http.StatusOK, workflow)
}

// List retrieves a list of workflows
func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "Organization context required")
		return
	}

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

	workflows, total, err := h.repo.List(r.Context(), organizationID, enabled, limit, offset)
	if err != nil {
		h.logger.Errorf("Failed to list workflows", logger.Err(err))
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

	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "Organization context required")
		return
	}

	var req models.UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	workflow, err := h.repo.Update(r.Context(), organizationID, id, &req)
	if err != nil {
		h.logger.Errorf("Failed to update workflow", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to update workflow")
		return
	}

	// Log audit event
	if h.auditService != nil {
		actorID, actorType := h.getActorFromContext(r)
		changes := make(map[string]interface{})
		if req.Name != nil {
			changes["name"] = *req.Name
		}
		if req.Description != nil {
			changes["description"] = *req.Description
		}
		if req.Definition != nil {
			changes["definition"] = "updated"
		}
		if err := h.auditService.LogWorkflowUpdated(r.Context(), id, actorID, actorType, changes); err != nil {
			h.logger.Errorf("Failed to log audit event: %v", err)
		}
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

	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "Organization context required")
		return
	}

	if err := h.repo.Delete(r.Context(), organizationID, id); err != nil {
		h.logger.Errorf("Failed to delete workflow", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to delete workflow")
		return
	}

	// Log audit event
	if h.auditService != nil {
		actorID, actorType := h.getActorFromContext(r)
		if err := h.auditService.LogWorkflowDeleted(r.Context(), id, actorID, actorType); err != nil {
			h.logger.Errorf("Failed to log audit event: %v", err)
		}
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

	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "Organization context required")
		return
	}

	if err := h.repo.SetEnabled(r.Context(), organizationID, id, true); err != nil {
		h.logger.Errorf("Failed to enable workflow", logger.Err(err))
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

	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "Organization context required")
		return
	}

	if err := h.repo.SetEnabled(r.Context(), organizationID, id, false); err != nil {
		h.logger.Errorf("Failed to disable workflow", logger.Err(err))
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

// getActorFromContext extracts actor ID and type from request context
func (h *WorkflowHandler) getActorFromContext(r *http.Request) (uuid.UUID, string) {
	// Try to get user ID from context
	if userID, ok := r.Context().Value("user_id").(uuid.UUID); ok {
		return userID, "user"
	}

	// Try to get API key from context (for system/service accounts)
	if apiKey, ok := r.Context().Value("api_key").(string); ok && apiKey != "" {
		// Use a system UUID for API key based requests
		return uuid.MustParse("00000000-0000-0000-0000-000000000001"), "api_key"
	}

	// Default to system actor
	return uuid.MustParse("00000000-0000-0000-0000-000000000000"), "system"
}
