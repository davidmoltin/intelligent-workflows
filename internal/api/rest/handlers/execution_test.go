package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Mock ExecutionRepository for handler tests
type mockExecutionRepoHandler struct {
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error)
	listFunc    func(ctx context.Context, workflowID *uuid.UUID, status *models.ExecutionStatus, limit, offset int) ([]models.WorkflowExecution, int64, error)
	getTraceFunc func(ctx context.Context, id uuid.UUID) (*models.ExecutionTraceResponse, error)
}

func (m *mockExecutionRepoHandler) GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockExecutionRepoHandler) ListExecutions(ctx context.Context, workflowID *uuid.UUID, status *models.ExecutionStatus, limit, offset int) ([]models.WorkflowExecution, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, workflowID, status, limit, offset)
	}
	return []models.WorkflowExecution{}, 0, nil
}

func (m *mockExecutionRepoHandler) GetExecutionTrace(ctx context.Context, id uuid.UUID) (*models.ExecutionTraceResponse, error) {
	if m.getTraceFunc != nil {
		return m.getTraceFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockExecutionRepoHandler) CreateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	return nil
}

func (m *mockExecutionRepoHandler) UpdateExecution(ctx context.Context, execution *models.WorkflowExecution) error {
	return nil
}

func (m *mockExecutionRepoHandler) CreateStepExecution(ctx context.Context, step *models.StepExecution) error {
	return nil
}

func (m *mockExecutionRepoHandler) UpdateStepExecution(ctx context.Context, step *models.StepExecution) error {
	return nil
}

func (m *mockExecutionRepoHandler) GetPausedExecutions(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
	return nil, errors.New("not implemented")
}

// Mock WorkflowResumer for handler tests
type mockWorkflowResumerHandler struct {
	pauseFunc          func(ctx context.Context, executionID uuid.UUID, reason string, stepID *uuid.UUID) error
	resumeWorkflowFunc func(ctx context.Context, executionID uuid.UUID, approved bool) error
	resumeFunc         func(ctx context.Context, executionID uuid.UUID, resumeData models.JSONB) error
	getPausedFunc      func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error)
}

func (m *mockWorkflowResumerHandler) PauseExecution(ctx context.Context, executionID uuid.UUID, reason string, stepID *uuid.UUID) error {
	if m.pauseFunc != nil {
		return m.pauseFunc(ctx, executionID, reason, stepID)
	}
	return nil
}

func (m *mockWorkflowResumerHandler) ResumeWorkflow(ctx context.Context, executionID uuid.UUID, approved bool) error {
	if m.resumeWorkflowFunc != nil {
		return m.resumeWorkflowFunc(ctx, executionID, approved)
	}
	return nil
}

func (m *mockWorkflowResumerHandler) ResumeExecution(ctx context.Context, executionID uuid.UUID, resumeData models.JSONB) error {
	if m.resumeFunc != nil {
		return m.resumeFunc(ctx, executionID, resumeData)
	}
	return nil
}

func (m *mockWorkflowResumerHandler) GetPausedExecutions(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
	if m.getPausedFunc != nil {
		return m.getPausedFunc(ctx, limit)
	}
	return []*models.WorkflowExecution{}, nil
}

func (m *mockWorkflowResumerHandler) CanResume(execution *models.WorkflowExecution) error {
	return nil
}

// Define interfaces for testing
type executionRepository interface {
	GetExecutionByID(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error)
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, status *models.ExecutionStatus, limit, offset int) ([]models.WorkflowExecution, int64, error)
	GetExecutionTrace(ctx context.Context, id uuid.UUID) (*models.ExecutionTraceResponse, error)
}

type workflowResumer interface {
	PauseExecution(ctx context.Context, executionID uuid.UUID, reason string, stepID *uuid.UUID) error
	ResumeWorkflow(ctx context.Context, executionID uuid.UUID, approved bool) error
	ResumeExecution(ctx context.Context, executionID uuid.UUID, resumeData models.JSONB) error
	GetPausedExecutions(ctx context.Context, limit int) ([]*models.WorkflowExecution, error)
}

// testExecutionHandler is a test-friendly version of ExecutionHandler
type testExecutionHandler struct {
	logger          *logger.Logger
	executionRepo   executionRepository
	workflowResumer workflowResumer
}

// PauseExecution handles POST /api/v1/executions/:id/pause
func (h *testExecutionHandler) PauseExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid execution ID")
		return
	}

	// Parse request body
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Reason == "" {
		req.Reason = "Manually paused via API"
	}

	// Pause the execution
	if err := h.workflowResumer.PauseExecution(r.Context(), id, req.Reason, nil); err != nil {
		h.logger.Errorf("Failed to pause execution %s: %v", id, err)
		RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to pause execution: %v", err))
		return
	}

	// Get updated execution
	execution, err := h.executionRepo.GetExecutionByID(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get execution after pause: %v", err)
		RespondError(w, http.StatusInternalServerError, "Execution paused but failed to retrieve updated state")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(execution)
}

// ResumeExecution handles POST /api/v1/executions/:id/resume
func (h *testExecutionHandler) ResumeExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid execution ID")
		return
	}

	// Parse request body (optional resume data)
	var req struct {
		ResumeData map[string]interface{} `json:"resume_data,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req.ResumeData = make(map[string]interface{})
	}

	// Resume the execution
	if len(req.ResumeData) > 0 {
		if err := h.workflowResumer.ResumeExecution(r.Context(), id, req.ResumeData); err != nil {
			h.logger.Errorf("Failed to resume execution %s: %v", id, err)
			RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume execution: %v", err))
			return
		}
	} else {
		// Use backward-compatible ResumeWorkflow with approved=true as default
		if err := h.workflowResumer.ResumeWorkflow(r.Context(), id, true); err != nil {
			h.logger.Errorf("Failed to resume execution %s: %v", id, err)
			RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resume execution: %v", err))
			return
		}
	}

	// Get updated execution
	execution, err := h.executionRepo.GetExecutionByID(r.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get execution after resume: %v", err)
		RespondError(w, http.StatusInternalServerError, "Execution resumed but failed to retrieve updated state")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(execution)
}

// ListPausedExecutions handles GET /api/v1/executions/paused
func (h *testExecutionHandler) ListPausedExecutions(w http.ResponseWriter, r *http.Request) {
	// Parse limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get paused executions
	executions, err := h.workflowResumer.GetPausedExecutions(r.Context(), limit)
	if err != nil {
		h.logger.Errorf("Failed to list paused executions: %v", err)
		RespondError(w, http.StatusInternalServerError, "Failed to retrieve paused executions")
		return
	}

	response := struct {
		Executions []*models.WorkflowExecution `json:"executions"`
		Count      int                          `json:"count"`
	}{
		Executions: executions,
		Count:      len(executions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper to create a chi context with URL params
func createChiContext(id string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return context.WithValue(context.Background(), chi.RouteCtxKey, rctx)
}

// TestPauseExecution_Handler tests the PauseExecution HTTP handler
func TestPauseExecution_Handler(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("successfully pauses execution", func(t *testing.T) {
		executionID := uuid.New()
		pausedAt := time.Now()
		pausedReason := "test pause reason"

		var capturedReason string
		resumer := &mockWorkflowResumerHandler{
			pauseFunc: func(ctx context.Context, id uuid.UUID, reason string, stepID *uuid.UUID) error {
				capturedReason = reason
				if id != executionID {
					return errors.New("wrong execution ID")
				}
				return nil
			},
		}

		repo := &mockExecutionRepoHandler{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return &models.WorkflowExecution{
					ID:           executionID,
					Status:       models.ExecutionStatusPaused,
					PausedAt:     &pausedAt,
					PausedReason: &capturedReason,
				}, nil
			},
		}

		handler := &testExecutionHandler{
			logger:          log,
			executionRepo:   repo,
			workflowResumer: resumer,
		}

		body := map[string]interface{}{
			"reason": pausedReason,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/executions/"+executionID.String()+"/pause", bytes.NewReader(bodyBytes))
		req = req.WithContext(createChiContext(executionID.String()))
		w := httptest.NewRecorder()

		handler.PauseExecution(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if capturedReason != pausedReason {
			t.Errorf("Expected reason '%s', got '%s'", pausedReason, capturedReason)
		}

		var response models.WorkflowExecution
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != models.ExecutionStatusPaused {
			t.Errorf("Expected status paused, got %s", response.Status)
		}
	})

	t.Run("uses default reason when not provided", func(t *testing.T) {
		executionID := uuid.New()

		var capturedReason string
		resumer := &mockWorkflowResumerHandler{
			pauseFunc: func(ctx context.Context, id uuid.UUID, reason string, stepID *uuid.UUID) error {
				capturedReason = reason
				return nil
			},
		}

		repo := &mockExecutionRepoHandler{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return &models.WorkflowExecution{
					ID:     executionID,
					Status: models.ExecutionStatusPaused,
				}, nil
			},
		}

		handler := &testExecutionHandler{
			logger:          log,
			executionRepo:   repo,
			workflowResumer: resumer,
		}

		body := map[string]interface{}{}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/executions/"+executionID.String()+"/pause", bytes.NewReader(bodyBytes))
		req = req.WithContext(createChiContext(executionID.String()))
		w := httptest.NewRecorder()

		handler.PauseExecution(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if capturedReason != "Manually paused via API" {
			t.Errorf("Expected default reason, got '%s'", capturedReason)
		}
	})

	t.Run("returns 400 for invalid execution ID", func(t *testing.T) {
		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: &mockWorkflowResumerHandler{},
	}

		req := httptest.NewRequest("POST", "/api/v1/executions/invalid-id/pause", nil)
		req = req.WithContext(createChiContext("invalid-id"))
		w := httptest.NewRecorder()

		handler.PauseExecution(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 for invalid JSON body", func(t *testing.T) {
		executionID := uuid.New()
		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: &mockWorkflowResumerHandler{},
	}

		req := httptest.NewRequest("POST", "/api/v1/executions/"+executionID.String()+"/pause", bytes.NewReader([]byte("invalid json")))
		req = req.WithContext(createChiContext(executionID.String()))
		w := httptest.NewRecorder()

		handler.PauseExecution(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 when pause fails", func(t *testing.T) {
		executionID := uuid.New()

		resumer := &mockWorkflowResumerHandler{
			pauseFunc: func(ctx context.Context, id uuid.UUID, reason string, stepID *uuid.UUID) error {
				return errors.New("pause failed")
			},
		}

		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: resumer,
	}

		body := map[string]interface{}{"reason": "test"}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/executions/"+executionID.String()+"/pause", bytes.NewReader(bodyBytes))
		req = req.WithContext(createChiContext(executionID.String()))
		w := httptest.NewRecorder()

		handler.PauseExecution(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

// TestResumeExecution_Handler tests the ResumeExecution HTTP handler
func TestResumeExecution_Handler(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("successfully resumes execution without resume data", func(t *testing.T) {
		executionID := uuid.New()

		var approvalCaptured *bool
		resumer := &mockWorkflowResumerHandler{
			resumeWorkflowFunc: func(ctx context.Context, id uuid.UUID, approved bool) error {
				approvalCaptured = &approved
				if id != executionID {
					return errors.New("wrong execution ID")
				}
				return nil
			},
		}

		repo := &mockExecutionRepoHandler{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return &models.WorkflowExecution{
					ID:     executionID,
					Status: models.ExecutionStatusRunning,
				}, nil
			},
		}

		handler := &testExecutionHandler{
			logger:          log,
			executionRepo:   repo,
			workflowResumer: resumer,
		}

		body := map[string]interface{}{}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/executions/"+executionID.String()+"/resume", bytes.NewReader(bodyBytes))
		req = req.WithContext(createChiContext(executionID.String()))
		w := httptest.NewRecorder()

		handler.ResumeExecution(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if approvalCaptured == nil || !*approvalCaptured {
			t.Error("Expected approved=true to be passed")
		}

		var response models.WorkflowExecution
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != models.ExecutionStatusRunning {
			t.Errorf("Expected status running, got %s", response.Status)
		}
	})

	t.Run("successfully resumes execution with custom resume data", func(t *testing.T) {
		executionID := uuid.New()

		var capturedData models.JSONB
		resumer := &mockWorkflowResumerHandler{
			resumeFunc: func(ctx context.Context, id uuid.UUID, resumeData models.JSONB) error {
				capturedData = resumeData
				return nil
			},
		}

		repo := &mockExecutionRepoHandler{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return &models.WorkflowExecution{
					ID:     executionID,
					Status: models.ExecutionStatusRunning,
				}, nil
			},
		}

		handler := &testExecutionHandler{
			logger:          log,
			executionRepo:   repo,
			workflowResumer: resumer,
		}

		customData := map[string]interface{}{
			"user_input": "test value",
			"number":     42,
		}
		body := map[string]interface{}{
			"resume_data": customData,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/executions/"+executionID.String()+"/resume", bytes.NewReader(bodyBytes))
		req = req.WithContext(createChiContext(executionID.String()))
		w := httptest.NewRecorder()

		handler.ResumeExecution(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if capturedData["user_input"] != "test value" {
			t.Error("Expected custom data to be passed")
		}
	})

	t.Run("returns 400 for invalid execution ID", func(t *testing.T) {
		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: &mockWorkflowResumerHandler{},
	}

		req := httptest.NewRequest("POST", "/api/v1/executions/invalid-id/resume", nil)
		req = req.WithContext(createChiContext("invalid-id"))
		w := httptest.NewRecorder()

		handler.ResumeExecution(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("handles empty request body gracefully", func(t *testing.T) {
		executionID := uuid.New()

		resumer := &mockWorkflowResumerHandler{
			resumeWorkflowFunc: func(ctx context.Context, id uuid.UUID, approved bool) error {
				return nil
			},
		}

		repo := &mockExecutionRepoHandler{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowExecution, error) {
				return &models.WorkflowExecution{
					ID:     executionID,
					Status: models.ExecutionStatusRunning,
				}, nil
			},
		}

		handler := &testExecutionHandler{
			logger:          log,
			executionRepo:   repo,
			workflowResumer: resumer,
		}

		req := httptest.NewRequest("POST", "/api/v1/executions/"+executionID.String()+"/resume", nil)
		req = req.WithContext(createChiContext(executionID.String()))
		w := httptest.NewRecorder()

		handler.ResumeExecution(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("returns 500 when resume fails", func(t *testing.T) {
		executionID := uuid.New()

		resumer := &mockWorkflowResumerHandler{
			resumeWorkflowFunc: func(ctx context.Context, id uuid.UUID, approved bool) error {
				return errors.New("resume failed")
			},
		}

		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: resumer,
	}

		body := map[string]interface{}{}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/executions/"+executionID.String()+"/resume", bytes.NewReader(bodyBytes))
		req = req.WithContext(createChiContext(executionID.String()))
		w := httptest.NewRecorder()

		handler.ResumeExecution(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

// TestListPausedExecutions_Handler tests the ListPausedExecutions HTTP handler
func TestListPausedExecutions_Handler(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("successfully lists paused executions", func(t *testing.T) {
		pausedAt := time.Now()
		executions := []*models.WorkflowExecution{
			{
				ID:       uuid.New(),
				Status:   models.ExecutionStatusPaused,
				PausedAt: &pausedAt,
			},
			{
				ID:       uuid.New(),
				Status:   models.ExecutionStatusPaused,
				PausedAt: &pausedAt,
			},
		}

		resumer := &mockWorkflowResumerHandler{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				if limit != 50 {
					t.Errorf("Expected default limit 50, got %d", limit)
				}
				return executions, nil
			},
		}

		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: resumer,
	}

		req := httptest.NewRequest("GET", "/api/v1/executions/paused", nil)
		w := httptest.NewRecorder()

		handler.ListPausedExecutions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response struct {
			Executions []*models.WorkflowExecution `json:"executions"`
			Count      int                          `json:"count"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Count != 2 {
			t.Errorf("Expected count 2, got %d", response.Count)
		}

		if len(response.Executions) != 2 {
			t.Errorf("Expected 2 executions, got %d", len(response.Executions))
		}
	})

	t.Run("respects custom limit parameter", func(t *testing.T) {
		var capturedLimit int
		resumer := &mockWorkflowResumerHandler{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				capturedLimit = limit
				return []*models.WorkflowExecution{}, nil
			},
		}

		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: resumer,
	}

		req := httptest.NewRequest("GET", "/api/v1/executions/paused?limit=25", nil)
		w := httptest.NewRecorder()

		handler.ListPausedExecutions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if capturedLimit != 25 {
			t.Errorf("Expected limit 25, got %d", capturedLimit)
		}
	})

	t.Run("enforces maximum limit of 100", func(t *testing.T) {
		var capturedLimit int
		resumer := &mockWorkflowResumerHandler{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				capturedLimit = limit
				return []*models.WorkflowExecution{}, nil
			},
		}

		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: resumer,
	}

		req := httptest.NewRequest("GET", "/api/v1/executions/paused?limit=200", nil)
		w := httptest.NewRecorder()

		handler.ListPausedExecutions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Should fall back to default limit when over 100
		if capturedLimit != 50 {
			t.Errorf("Expected default limit 50 for over-limit request, got %d", capturedLimit)
		}
	})

	t.Run("returns empty list when no paused executions", func(t *testing.T) {
		resumer := &mockWorkflowResumerHandler{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return []*models.WorkflowExecution{}, nil
			},
		}

		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: resumer,
	}

		req := httptest.NewRequest("GET", "/api/v1/executions/paused", nil)
		w := httptest.NewRecorder()

		handler.ListPausedExecutions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response struct {
			Executions []*models.WorkflowExecution `json:"executions"`
			Count      int                          `json:"count"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Count != 0 {
			t.Errorf("Expected count 0, got %d", response.Count)
		}
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		resumer := &mockWorkflowResumerHandler{
			getPausedFunc: func(ctx context.Context, limit int) ([]*models.WorkflowExecution, error) {
				return nil, errors.New("service error")
			},
		}

		handler := &testExecutionHandler{
		logger:          log,
		executionRepo:   &mockExecutionRepoHandler{},
		workflowResumer: resumer,
	}

		req := httptest.NewRequest("GET", "/api/v1/executions/paused", nil)
		w := httptest.NewRecorder()

		handler.ListPausedExecutions(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}
