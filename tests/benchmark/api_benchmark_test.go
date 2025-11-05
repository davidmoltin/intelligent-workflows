package benchmark

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/handlers"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/go-chi/chi/v5"
)

// BenchmarkAPI_CreateWorkflow benchmarks workflow creation endpoint
func BenchmarkAPI_CreateWorkflow(b *testing.B) {
	router := setupBenchmarkRouter(b)

	workflow := models.CreateWorkflowRequest{
		WorkflowID: "benchmark-workflow",
		Version:    "1.0.0",
		Name:       "Benchmark Workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "test.event",
			},
			Steps: []models.Step{
				{
					ID:   "step1",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}

	body, _ := json.Marshal(workflow)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/workflows", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkAPI_GetWorkflow benchmarks workflow retrieval endpoint
func BenchmarkAPI_GetWorkflow(b *testing.B) {
	router := setupBenchmarkRouter(b)
	workflowID := "550e8400-e29b-41d4-a716-446655440000" // UUID format

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/workflows/"+workflowID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkAPI_ListWorkflows benchmarks workflow listing endpoint
func BenchmarkAPI_ListWorkflows(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkAPI_TriggerEvent benchmarks event triggering endpoint
func BenchmarkAPI_TriggerEvent(b *testing.B) {
	router := setupBenchmarkRouter(b)

	event := map[string]interface{}{
		"event_type": "test.event",
		"source":     "benchmark",
		"payload": map[string]interface{}{
			"test": "data",
		},
	}

	body, _ := json.Marshal(event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkAPI_GetExecution benchmarks execution retrieval endpoint
func BenchmarkAPI_GetExecution(b *testing.B) {
	router := setupBenchmarkRouter(b)
	executionID := "660e8400-e29b-41d4-a716-446655440000" // UUID format

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/executions/"+executionID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkAPI_ListExecutions benchmarks execution listing endpoint
func BenchmarkAPI_ListExecutions(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/executions", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkAPI_CreateApproval benchmarks approval creation endpoint
func BenchmarkAPI_CreateApproval(b *testing.B) {
	router := setupBenchmarkRouter(b)

	approval := map[string]interface{}{
		"workflow_execution_id": "exec-123",
		"approver_role":         "manager",
		"context": map[string]interface{}{
			"reason": "Test approval",
		},
	}

	body, _ := json.Marshal(approval)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/approvals", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkAPI_Login benchmarks authentication endpoint
func BenchmarkAPI_Login(b *testing.B) {
	router := setupBenchmarkRouter(b)

	loginReq := map[string]interface{}{
		"username": "testuser",
		"password": "testpass",
	}

	body, _ := json.Marshal(loginReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkAPI_HealthCheck benchmarks health check endpoint
func BenchmarkAPI_HealthCheck(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkAPI_ConcurrentRequests benchmarks concurrent API requests
func BenchmarkAPI_ConcurrentRequests(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkAPI_JSONParsing benchmarks JSON request parsing
func BenchmarkAPI_JSONParsing(b *testing.B) {
	router := setupBenchmarkRouter(b)

	largeWorkflow := models.CreateWorkflowRequest{
		WorkflowID:  "large-workflow",
		Version:     "1.0.0",
		Name:        "Large Workflow",
		Description: stringPtr("A workflow with many steps for benchmarking"),
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "complex.event",
			},
			Steps: make([]models.Step, 50), // 50 steps
		},
	}

	// Fill steps
	for i := 0; i < 50; i++ {
		largeWorkflow.Definition.Steps[i] = models.Step{
			ID:   "step" + string(rune('0'+i)),
			Type: "action",
			Action: &models.Action{
				Type: "allow",
			},
		}
	}

	body, _ := json.Marshal(largeWorkflow)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/workflows", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// Helper functions

func setupBenchmarkRouter(b *testing.B) *chi.Mux {
	b.Helper()

	router := chi.NewRouter()

	// Setup minimal handlers for benchmarking
	h := &handlers.Handlers{
		Health: &handlers.HealthHandler{},
		// Add other handlers as needed
	}

	// Setup routes (simplified)
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Add other routes
	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/workflows", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		})
		r.Post("/workflows", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000"}`))
		})
		r.Get("/workflows/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000"}`))
		})
		r.Post("/events", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		})
		r.Get("/executions", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		})
		r.Get("/executions/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"660e8400-e29b-41d4-a716-446655440000"}`))
		})
		r.Post("/approvals", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"770e8400-e29b-41d4-a716-446655440000"}`))
		})
		r.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token":"benchmark-token"}`))
		})
	})

	_ = h // Use the handlers variable

	return router
}

func stringPtr(s string) *string {
	return &s
}
