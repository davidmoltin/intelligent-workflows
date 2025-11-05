package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/davidmoltin/intelligent-workflows/internal/engine"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/davidmoltin/intelligent-workflows/pkg/validator"
)

// EventHandler handles event-related HTTP requests
type EventHandler struct {
	logger      *logger.Logger
	eventRouter *engine.EventRouter
}

// NewEventHandler creates a new event handler
func NewEventHandler(log *logger.Logger, eventRouter *engine.EventRouter) *EventHandler {
	return &EventHandler{
		logger:      log,
		eventRouter: eventRouter,
	}
}

// CreateEvent handles POST /api/v1/events
func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEventRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorf("Failed to decode request: %v", err)
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Default source if not provided
	if req.Source == "" {
		req.Source = "api"
	}

	// Route event to workflows
	event, err := h.eventRouter.RouteEvent(r.Context(), req.EventType, req.Source, req.Payload)
	if err != nil {
		h.logger.Errorf("Failed to route event: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to process event")
		return
	}

	// Return event
	RespondJSON(w, http.StatusCreated, event)
}
