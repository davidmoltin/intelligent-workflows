package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/davidmoltin/intelligent-workflows/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RuleHandler handles rule-related HTTP requests
type RuleHandler struct {
	logger      *logger.Logger
	ruleService *services.RuleService
}

// NewRuleHandler creates a new rule handler
func NewRuleHandler(log *logger.Logger, ruleService *services.RuleService) *RuleHandler {
	return &RuleHandler{
		logger:      log,
		ruleService: ruleService,
	}
}

// Create creates a new rule
func (h *RuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	rule, err := h.ruleService.Create(r.Context(), &req)
	if err != nil {
		h.logger.Errorf("Failed to create rule", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to create rule")
		return
	}

	h.respondJSON(w, http.StatusCreated, rule)
}

// Get retrieves a rule by ID
func (h *RuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	rule, err := h.ruleService.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get rule", logger.Err(err))
		h.respondError(w, http.StatusNotFound, "Rule not found")
		return
	}

	h.respondJSON(w, http.StatusOK, rule)
}

// List retrieves a list of rules
func (h *RuleHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	enabledStr := r.URL.Query().Get("enabled")
	ruleTypeStr := r.URL.Query().Get("rule_type")

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

	var ruleType *models.RuleType
	if ruleTypeStr != "" {
		rt := models.RuleType(ruleTypeStr)
		// Validate rule type
		if rt == models.RuleTypeCondition || rt == models.RuleTypeValidation || rt == models.RuleTypeEnrichment {
			ruleType = &rt
		}
	}

	rules, total, err := h.ruleService.List(r.Context(), enabled, ruleType, limit, offset)
	if err != nil {
		h.logger.Errorf("Failed to list rules", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to list rules")
		return
	}

	page := offset/limit + 1
	response := map[string]interface{}{
		"rules":     rules,
		"total":     total,
		"page":      page,
		"page_size": limit,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// Update updates a rule
func (h *RuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	var req models.UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	rule, err := h.ruleService.Update(r.Context(), id, &req)
	if err != nil {
		h.logger.Errorf("Failed to update rule", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to update rule")
		return
	}

	h.respondJSON(w, http.StatusOK, rule)
}

// Delete deletes a rule
func (h *RuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	if err := h.ruleService.Delete(r.Context(), id); err != nil {
		h.logger.Errorf("Failed to delete rule", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to delete rule")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Enable enables a rule
func (h *RuleHandler) Enable(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	if err := h.ruleService.Enable(r.Context(), id); err != nil {
		h.logger.Errorf("Failed to enable rule", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to enable rule")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "Rule enabled"})
}

// Disable disables a rule
func (h *RuleHandler) Disable(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	if err := h.ruleService.Disable(r.Context(), id); err != nil {
		h.logger.Errorf("Failed to disable rule", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to disable rule")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "Rule disabled"})
}

// TestRule tests a rule with provided context
func (h *RuleHandler) TestRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	var req models.TestRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.ruleService.TestRule(r.Context(), id, &req)
	if err != nil {
		h.logger.Errorf("Failed to test rule", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to test rule")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *RuleHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *RuleHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
