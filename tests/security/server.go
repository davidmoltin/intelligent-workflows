package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/pkg/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SecurityTestServer represents a test HTTP server for security testing
type SecurityTestServer struct {
	Server  *http.Server
	BaseURL string
	DB      *testutil.TestDB
	Pool    *pgxpool.Pool
	t       *testing.T
}

// NewSecurityTestServer creates a new security test server
func NewSecurityTestServer(t *testing.T) *SecurityTestServer {
	t.Helper()

	// Check if security tests should run
	if os.Getenv("SECURITY_TESTS") == "" && !testing.Short() {
		// Run security tests by default in non-short mode
	}

	// Create test database
	db := testutil.SetupTestDB(t)

	// Run migrations
	migrationsPath := "../../migrations/postgres"
	testutil.RunMigrations(t, db, migrationsPath)

	// Create router with minimal setup for security testing
	router := chi.NewRouter()

	// Add basic health endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Add basic API routes for testing
	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"access_token":"test","refresh_token":"test"}`))
		})
		r.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"test","refresh_token":"test"}`))
		})
		r.Post("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"test","refresh_token":"test"}`))
		})
		r.Post("/auth/api-keys", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"test","api_key":"test-key"}`))
		})
		r.Get("/auth/me", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"username":"test","email":"test@example.com"}`))
		})
		r.Get("/workflows", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		})
		r.Post("/workflows", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"test-id"}`))
		})
		r.Delete("/workflows/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		r.Post("/users/me/roles", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		r.Post("/approvals/{id}/approve", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
	})

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	ts := &SecurityTestServer{
		Server:  server,
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		DB:      db,
		Pool:    db.Pool,
		t:       t,
	}

	return ts
}

// Start starts the security test server
func (ts *SecurityTestServer) Start() {
	ts.t.Helper()

	go func() {
		if err := ts.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ts.t.Errorf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ts.t.Fatal("Server failed to start within timeout")
		case <-ticker.C:
			resp, err := http.Get(ts.BaseURL + "/health")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// Stop stops the security test server
func (ts *SecurityTestServer) Stop() {
	ts.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ts.Server.Shutdown(ctx); err != nil {
		ts.t.Logf("Server shutdown error: %v", err)
	}

	ts.DB.Teardown()
}

// ResetDatabase truncates all tables
func (ts *SecurityTestServer) ResetDatabase() {
	ts.t.Helper()

	tables := []string{
		"api_keys",
		"user_permissions",
		"user_roles",
		"permissions",
		"roles",
		"users",
		"step_executions",
		"workflow_executions",
		"approvals",
		"events",
		"workflows",
	}

	ts.DB.Truncate(tables...)
}
