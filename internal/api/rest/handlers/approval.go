package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/middleware"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ApprovalHandler handles approval-related HTTP requests
type ApprovalHandler struct {
	logger          *logger.Logger
	approvalService *services.ApprovalService
}

// NewApprovalHandler creates a new approval handler
func NewApprovalHandler(log *logger.Logger, approvalService *services.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{
		logger:          log,
		approvalService: approvalService,
	}
}

// ListApprovals handles GET /api/v1/approvals
func (h *ApprovalHandler) ListApprovals(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		http.Error(w, "Organization context required", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	statusStr := r.URL.Query().Get("status")
	approverIDStr := r.URL.Query().Get("approver_id")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Parse status
	var status *models.ApprovalStatus
	if statusStr != "" {
		s := models.ApprovalStatus(statusStr)
		status = &s
	}

	// Parse approver_id
	var approverID *uuid.UUID
	if approverIDStr != "" {
		id, err := uuid.Parse(approverIDStr)
		if err != nil {
			http.Error(w, "Invalid approver_id", http.StatusBadRequest)
			return
		}
		approverID = &id
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

	// Get approvals
	approvals, total, err := h.approvalService.ListApprovals(r.Context(), organizationID, status, approverID, limit, offset)
	if err != nil {
		h.logger.Errorf("Failed to list approvals: %v", err)
		http.Error(w, "Failed to retrieve approvals", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"approvals": approvals,
		"total":     total,
		"page":      offset / limit,
		"page_size": limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetApproval handles GET /api/v1/approvals/:id
func (h *ApprovalHandler) GetApproval(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		http.Error(w, "Organization context required", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid approval ID", http.StatusBadRequest)
		return
	}

	approval, err := h.approvalService.GetApproval(r.Context(), organizationID, id)
	if err != nil {
		h.logger.Errorf("Failed to get approval: %v", err)
		http.Error(w, "Approval not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approval)
}

// ApproveRequest handles POST /api/v1/approvals/:id/approve
func (h *ApprovalHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		RespondError(w, http.StatusUnauthorized, "Organization context required")
		return
	}

	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid approval ID", http.StatusBadRequest)
		return
	}

	var req models.ApprovalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorf("Failed to decode request: %v", err)
		// If body is empty, that's OK - reason is optional
		req.Reason = nil
	}

	// Get approver ID from authentication context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		h.logger.Error("User ID not found in context")
		RespondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	approval, err := h.approvalService.ApproveRequest(r.Context(), organizationID, id, userID, req.Reason)
	if err != nil {
		h.logger.Errorf("Failed to approve request: %v", err)
		// Don't leak internal error details
		RespondError(w, http.StatusBadRequest, "Failed to approve request")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approval)
}

// RejectRequest handles POST /api/v1/approvals/:id/reject
func (h *ApprovalHandler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	// Get organization ID from context
	organizationID := middleware.GetOrganizationID(r.Context())
	if organizationID == uuid.Nil {
		RespondError(w, http.StatusUnauthorized, "Organization context required")
		return
	}

	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid approval ID", http.StatusBadRequest)
		return
	}

	var req models.ApprovalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorf("Failed to decode request: %v", err)
		// If body is empty, that's OK - reason is optional
		req.Reason = nil
	}

	// Get approver ID from authentication context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		h.logger.Error("User ID not found in context")
		RespondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	approval, err := h.approvalService.RejectRequest(r.Context(), organizationID, id, userID, req.Reason)
	if err != nil {
		h.logger.Errorf("Failed to reject request: %v", err)
		// Don't leak internal error details
		RespondError(w, http.StatusBadRequest, "Failed to reject request")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approval)
}
