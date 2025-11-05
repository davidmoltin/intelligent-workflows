package e2e

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/api/rest"
	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/handlers"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestServer represents a test HTTP server
type TestServer struct {
	Server  *http.Server
	BaseURL string
	DB      *testutil.TestDB
	Pool    *pgxpool.Pool
	t       *testing.T
}

// NewTestServer creates a new test server
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Create test database
	db := testutil.SetupTestDB(t)

	// Run migrations
	migrationsPath := "../../migrations/postgres"
	testutil.RunMigrations(t, db, migrationsPath)

	// Create router
	router := chi.NewRouter()

	// Setup handlers (simplified - in real implementation would wire up all dependencies)
	h := &handlers.Handlers{
		// Health and readiness handlers
		Health: &handlers.HealthHandler{},
	}

	// Setup routes
	rest.SetupRoutes(router, h)

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

	ts := &TestServer{
		Server:  server,
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		DB:      db,
		Pool:    db.Pool,
		t:       t,
	}

	return ts
}

// Start starts the test server
func (ts *TestServer) Start() {
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

// Stop stops the test server
func (ts *TestServer) Stop() {
	ts.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ts.Server.Shutdown(ctx); err != nil {
		ts.t.Logf("Server shutdown error: %v", err)
	}

	ts.DB.Teardown()
}

// ResetDatabase truncates all tables
func (ts *TestServer) ResetDatabase() {
	ts.t.Helper()

	tables := []string{
		"step_executions",
		"workflow_executions",
		"events",
		"rules",
		"workflows",
	}

	ts.DB.Truncate(tables...)
}

// GetConfig returns test configuration
func GetConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:            "127.0.0.1",
			Port:            0, // Dynamic port
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "postgres",
			Password:        "postgres",
			Database:        "workflows_test",
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    2,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		Logger: config.LoggerConfig{
			Level:  "info",
			Format: "json",
		},
		App: config.AppConfig{
			Environment: "test",
			Version:     "test",
			Name:        "intelligent-workflows-test",
		},
	}
}
