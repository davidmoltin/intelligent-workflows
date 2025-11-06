package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AuditHandler handles audit log-related HTTP requests
type AuditHandler struct {
	logger       *logger.Logger
	auditService *services.AuditService
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(log *logger.Logger, auditService *services.AuditService) *AuditHandler {
	return &AuditHandler{
		logger:       log,
		auditService: auditService,
	}
}

// ListAuditLogs handles GET /api/v1/audit-logs
func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	entityType := r.URL.Query().Get("entity_type")
	entityIDStr := r.URL.Query().Get("entity_id")
	action := r.URL.Query().Get("action")
	actorIDStr := r.URL.Query().Get("actor_id")
	actorType := r.URL.Query().Get("actor_type")
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Build filters
	filters := &postgres.AuditLogFilters{}

	if entityType != "" {
		filters.EntityType = &entityType
	}

	if entityIDStr != "" {
		entityID, err := uuid.Parse(entityIDStr)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "Invalid entity_id")
			return
		}
		filters.EntityID = &entityID
	}

	if action != "" {
		filters.Action = &action
	}

	if actorIDStr != "" {
		actorID, err := uuid.Parse(actorIDStr)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "Invalid actor_id")
			return
		}
		filters.ActorID = &actorID
	}

	if actorType != "" {
		filters.ActorType = &actorType
	}

	if startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "Invalid start_time format (use RFC3339)")
			return
		}
		filters.StartTime = &startTime
	}

	if endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "Invalid end_time format (use RFC3339)")
			return
		}
		filters.EndTime = &endTime
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

	// Get audit logs
	logs, total, err := h.auditService.ListAuditLogs(r.Context(), filters, limit, offset)
	if err != nil {
		h.logger.Errorf("Failed to list audit logs: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to retrieve audit logs")
		return
	}

	page := offset / limit
	response := map[string]interface{}{
		"audit_logs": logs,
		"total":      total,
		"page":       page,
		"page_size":  limit,
	}

	RespondJSON(w, http.StatusOK, response)
}

// GetAuditLog handles GET /api/v1/audit-logs/:id
func (h *AuditHandler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid audit log ID")
		return
	}

	log, err := h.auditService.GetAuditLog(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get audit log: %v", err)
		RespondError(w, http.StatusNotFound, "Audit log not found")
		return
	}

	RespondJSON(w, http.StatusOK, log)
}

// GetEntityAuditLogs handles GET /api/v1/audit-logs/entity/:entity_type/:entity_id
func (h *AuditHandler) GetEntityAuditLogs(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entity_type")
	entityIDStr := chi.URLParam(r, "entity_id")

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid entity ID")
		return
	}

	// Parse pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

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

	// Get audit logs for entity
	logs, err := h.auditService.GetEntityAuditLogs(r.Context(), entityType, entityID, limit, offset)
	if err != nil {
		h.logger.Errorf("Failed to get entity audit logs: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to retrieve entity audit logs")
		return
	}

	response := map[string]interface{}{
		"audit_logs":  logs,
		"entity_type": entityType,
		"entity_id":   entityID,
		"count":       len(logs),
	}

	RespondJSON(w, http.StatusOK, response)
}

// GetActorAuditLogs handles GET /api/v1/audit-logs/actor/:actor_id
func (h *AuditHandler) GetActorAuditLogs(w http.ResponseWriter, r *http.Request) {
	actorIDStr := chi.URLParam(r, "actor_id")

	actorID, err := uuid.Parse(actorIDStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid actor ID")
		return
	}

	// Parse pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

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

	// Get audit logs for actor
	logs, err := h.auditService.GetActorAuditLogs(r.Context(), actorID, limit, offset)
	if err != nil {
		h.logger.Errorf("Failed to get actor audit logs: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to retrieve actor audit logs")
		return
	}

	response := map[string]interface{}{
		"audit_logs": logs,
		"actor_id":   actorID,
		"count":      len(logs),
	}

	RespondJSON(w, http.StatusOK, response)
}
