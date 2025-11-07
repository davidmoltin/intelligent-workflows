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

// OrganizationHandler handles organization-related HTTP requests
type OrganizationHandler struct {
	logger *logger.Logger
	repo   *postgres.OrganizationRepository
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(log *logger.Logger, repo *postgres.OrganizationRepository) *OrganizationHandler {
	return &OrganizationHandler{
		logger: log,
		repo:   repo,
	}
}

// Create creates a new organization
func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	organization, err := h.repo.Create(r.Context(), &req, userID)
	if err != nil {
		h.logger.Errorf("Failed to create organization", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to create organization")
		return
	}

	h.respondJSON(w, http.StatusCreated, organization)
}

// Get retrieves an organization by ID
func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	organization, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get organization", logger.Err(err))
		h.respondError(w, http.StatusNotFound, "Organization not found")
		return
	}

	h.respondJSON(w, http.StatusOK, organization)
}

// GetBySlug retrieves an organization by slug
func (h *OrganizationHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		h.respondError(w, http.StatusBadRequest, "Organization slug is required")
		return
	}

	organization, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		h.logger.Errorf("Failed to get organization by slug", logger.Err(err))
		h.respondError(w, http.StatusNotFound, "Organization not found")
		return
	}

	h.respondJSON(w, http.StatusOK, organization)
}

// List retrieves organizations for the current user
func (h *OrganizationHandler) List(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	organizations, err := h.repo.GetUserOrganizations(r.Context(), userID)
	if err != nil {
		h.logger.Errorf("Failed to list user organizations", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to list organizations")
		return
	}

	h.respondJSON(w, http.StatusOK, organizations)
}

// Update updates an organization
func (h *OrganizationHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	var req models.UpdateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	organization, err := h.repo.Update(r.Context(), id, &req)
	if err != nil {
		h.logger.Errorf("Failed to update organization", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to update organization")
		return
	}

	h.respondJSON(w, http.StatusOK, organization)
}

// Delete deletes an organization
func (h *OrganizationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.logger.Errorf("Failed to delete organization", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to delete organization")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddUser adds a user to an organization
func (h *OrganizationHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	var req struct {
		UserID uuid.UUID `json:"user_id" validate:"required"`
		RoleID uuid.UUID `json:"role_id" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get inviting user from context
	invitedBy := middleware.GetUserID(r.Context())

	orgUser, err := h.repo.AddUser(r.Context(), orgID, req.UserID, req.RoleID, &invitedBy)
	if err != nil {
		h.logger.Errorf("Failed to add user to organization", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to add user to organization")
		return
	}

	h.respondJSON(w, http.StatusCreated, orgUser)
}

// RemoveUser removes a user from an organization
func (h *OrganizationHandler) RemoveUser(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := h.repo.RemoveUser(r.Context(), orgID, userID); err != nil {
		h.logger.Errorf("Failed to remove user from organization", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to remove user from organization")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListUsers lists all users in an organization
func (h *OrganizationHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	users, total, err := h.repo.ListUsers(r.Context(), orgID, limit, offset)
	if err != nil {
		h.logger.Errorf("Failed to list organization users", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to list organization users")
		return
	}

	response := map[string]interface{}{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// UpdateUserRole updates a user's role in an organization
func (h *OrganizationHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		RoleID uuid.UUID `json:"role_id" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.repo.UpdateUserRole(r.Context(), orgID, userID, req.RoleID); err != nil {
		h.logger.Errorf("Failed to update user role", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to update user role")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CheckUserAccess checks if a user has access to an organization
func (h *OrganizationHandler) CheckUserAccess(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid organization ID")
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	hasAccess, err := h.repo.CheckUserAccess(r.Context(), orgID, userID)
	if err != nil {
		h.logger.Errorf("Failed to check user access", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to check user access")
		return
	}

	response := map[string]bool{"has_access": hasAccess}
	h.respondJSON(w, http.StatusOK, response)
}

// respondJSON sends a JSON response
func (h *OrganizationHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (h *OrganizationHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// getActorFromContext extracts actor information from request context
func (h *OrganizationHandler) getActorFromContext(r *http.Request) (uuid.UUID, string) {
	userID := middleware.GetUserID(r.Context())
	authType := "user"
	if at, ok := r.Context().Value("auth_type").(string); ok {
		authType = at
	}
	return userID, authType
}
