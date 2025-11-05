package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
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
	approvals, total, err := h.approvalService.ListApprovals(r.Context(), status, approverID, limit, offset)
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
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid approval ID", http.StatusBadRequest)
		return
	}

	approval, err := h.approvalService.GetApproval(r.Context(), id)
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

	// TODO: Get approver ID from authentication context
	// For now, use a dummy approver ID
	approverID := uuid.New()

	approval, err := h.approvalService.ApproveRequest(r.Context(), id, approverID, req.Reason)
	if err != nil {
		h.logger.Errorf("Failed to approve request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approval)
}

// RejectRequest handles POST /api/v1/approvals/:id/reject
func (h *ApprovalHandler) RejectRequest(w http.ResponseWriter, r *http.Request) {
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

	// TODO: Get approver ID from authentication context
	// For now, use a dummy approver ID
	approverID := uuid.New()

	approval, err := h.approvalService.RejectRequest(r.Context(), id, approverID, req.Reason)
	if err != nil {
		h.logger.Errorf("Failed to reject request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approval)
}
