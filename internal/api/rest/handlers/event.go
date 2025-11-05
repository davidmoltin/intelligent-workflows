package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/davidmoltin/intelligent-workflows/internal/engine"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.EventType == "" {
		http.Error(w, "event_type is required", http.StatusBadRequest)
		return
	}

	if req.Payload == nil {
		http.Error(w, "payload is required", http.StatusBadRequest)
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
		http.Error(w, "Failed to process event", http.StatusInternalServerError)
		return
	}

	// Return event
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}
