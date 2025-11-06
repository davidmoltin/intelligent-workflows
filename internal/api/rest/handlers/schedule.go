package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/davidmoltin/intelligent-workflows/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ScheduleService defines the interface for schedule operations
type ScheduleService interface {
	CreateSchedule(ctx context.Context, workflowID uuid.UUID, req *models.CreateScheduleRequest) (*models.WorkflowSchedule, error)
	GetSchedule(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error)
	GetWorkflowSchedules(ctx context.Context, workflowID uuid.UUID) ([]*models.WorkflowSchedule, error)
	UpdateSchedule(ctx context.Context, id uuid.UUID, req *models.UpdateScheduleRequest) (*models.WorkflowSchedule, error)
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
	GetNextRuns(ctx context.Context, id uuid.UUID, count int) ([]time.Time, error)
	ListSchedules(ctx context.Context, limit, offset int) ([]*models.WorkflowSchedule, int64, error)
}

// ScheduleHandler handles schedule-related HTTP requests
type ScheduleHandler struct {
	logger          *logger.Logger
	scheduleService ScheduleService
}

// NewScheduleHandler creates a new schedule handler
func NewScheduleHandler(log *logger.Logger, scheduleService ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{
		logger:          log,
		scheduleService: scheduleService,
	}
}

// CreateSchedule handles POST /api/v1/workflows/:id/schedules
func (h *ScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	workflowIDStr := chi.URLParam(r, "id")
	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid workflow ID")
		return
	}

	var req models.CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	schedule, err := h.scheduleService.CreateSchedule(r.Context(), workflowID, &req)
	if err != nil {
		h.logger.Errorf("Failed to create schedule: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to create schedule: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(schedule)
}

// GetWorkflowSchedules handles GET /api/v1/workflows/:id/schedules
func (h *ScheduleHandler) GetWorkflowSchedules(w http.ResponseWriter, r *http.Request) {
	workflowIDStr := chi.URLParam(r, "id")
	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid workflow ID")
		return
	}

	schedules, err := h.scheduleService.GetWorkflowSchedules(r.Context(), workflowID)
	if err != nil {
		h.logger.Errorf("Failed to get workflow schedules: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to get schedules")
		return
	}

	response := map[string]interface{}{
		"schedules": schedules,
		"count":     len(schedules),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSchedule handles GET /api/v1/schedules/:id
func (h *ScheduleHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	schedule, err := h.scheduleService.GetSchedule(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get schedule: %v", err)
		RespondError(w, http.StatusNotFound, "Schedule not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

// UpdateSchedule handles PUT /api/v1/schedules/:id
func (h *ScheduleHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	var req models.UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	schedule, err := h.scheduleService.UpdateSchedule(r.Context(), id, &req)
	if err != nil {
		h.logger.Errorf("Failed to update schedule: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to update schedule: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

// DeleteSchedule handles DELETE /api/v1/schedules/:id
func (h *ScheduleHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	if err := h.scheduleService.DeleteSchedule(r.Context(), id); err != nil {
		h.logger.Errorf("Failed to delete schedule: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to delete schedule")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Schedule deleted successfully"})
}

// GetNextRuns handles GET /api/v1/schedules/:id/next-runs
func (h *ScheduleHandler) GetNextRuns(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	// Parse count parameter
	countStr := r.URL.Query().Get("count")
	count := 10 // default
	if countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 && c <= 100 {
			count = c
		}
	}

	nextRuns, err := h.scheduleService.GetNextRuns(r.Context(), id, count)
	if err != nil {
		h.logger.Errorf("Failed to get next runs: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to get next runs")
		return
	}

	response := models.NextRunsResponse{
		ScheduleID: id,
		NextRuns:   nextRuns,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListSchedules handles GET /api/v1/schedules
func (h *ScheduleHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	schedules, total, err := h.scheduleService.ListSchedules(r.Context(), limit, offset)
	if err != nil {
		h.logger.Errorf("Failed to list schedules: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to list schedules")
		return
	}

	// Calculate pagination
	page := offset / limit

	response := models.ScheduleListResponse{
		Schedules: schedules,
		Total:     total,
		Page:      page,
		PageSize:  limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
