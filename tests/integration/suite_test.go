package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/davidmoltin/intelligent-workflows/pkg/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IntegrationSuite holds the test suite configuration
type IntegrationSuite struct {
	DB      *testutil.TestDB
	Pool    *pgxpool.Pool
	BaseURL string
}

var suite *IntegrationSuite

// TestMain sets up and tears down the test suite
func TestMain(m *testing.M) {
	// Check if integration tests should run
	if os.Getenv("INTEGRATION_TESTS") == "" {
		fmt.Println("Skipping integration tests. Set INTEGRATION_TESTS=1 to run.")
		os.Exit(0)
	}

	// Setup would go here if we had a running test database
	// For now, we'll skip actual database setup in integration tests
	// and rely on E2E tests for full integration testing

	// Run tests
	code := m.Run()

	// Teardown would go here

	os.Exit(code)
}

// SetupSuite initializes the test suite
func SetupSuite(t *testing.T) *IntegrationSuite {
	t.Helper()

	if suite != nil {
		return suite
	}

	// Create test database
	db := testutil.SetupTestDB(t)

	// Run migrations
	migrationsPath := "../../migrations/postgres"
	if _, err := os.Stat(migrationsPath); err == nil {
		testutil.RunMigrations(t, db, migrationsPath)
	}

	suite = &IntegrationSuite{
		DB:      db,
		Pool:    db.Pool,
		BaseURL: "http://localhost:8080",
	}

	return suite
}

// TeardownSuite cleans up the test suite
func TeardownSuite(t *testing.T) {
	t.Helper()

	if suite != nil && suite.DB != nil {
		suite.DB.Teardown()
		suite = nil
	}
}

// ResetDatabase truncates all tables
func (s *IntegrationSuite) ResetDatabase(t *testing.T) {
	t.Helper()

	tables := []string{
		"step_executions",
		"workflow_executions",
		"events",
		"rules",
		"workflows",
	}

	s.DB.Truncate(tables...)
}

// GetContext returns a context for testing
func (s *IntegrationSuite) GetContext(t *testing.T) context.Context {
	t.Helper()
	return testutil.Context(t)
}
