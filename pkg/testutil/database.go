package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// TestDB represents a test database instance
type TestDB struct {
	Pool   *pgxpool.Pool
	DB     *sql.DB
	DBName string
	t      *testing.T
}

// SetupTestDB creates a test database for testing
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Connect to default postgres database to create test database
	ctx := context.Background()
	dbName := fmt.Sprintf("workflows_test_%d", randomID())

	// Connection string for postgres database
	adminConnStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	adminDB, err := sql.Open("pgx", adminConnStr)
	require.NoError(t, err, "Failed to connect to postgres database")
	defer adminDB.Close()

	// Create test database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	require.NoError(t, err, "Failed to create test database")

	// Connect to test database with pgxpool
	testConnStr := fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", dbName)
	pool, err := pgxpool.New(ctx, testConnStr)
	require.NoError(t, err, "Failed to connect to test database")

	// Also create a stdlib *sql.DB for repositories that need it
	config, err := pgxpool.ParseConfig(testConnStr)
	require.NoError(t, err, "Failed to parse connection string")

	stdDB := stdlib.OpenDB(*config.ConnConfig)

	return &TestDB{
		Pool:   pool,
		DB:     stdDB,
		DBName: dbName,
		t:      t,
	}
}

// Teardown drops the test database
func (db *TestDB) Teardown() {
	db.t.Helper()

	// Close connections
	if db.DB != nil {
		db.DB.Close()
	}
	if db.Pool != nil {
		db.Pool.Close()
	}

	// Connect to default postgres database to drop test database
	adminConnStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	adminDB, err := sql.Open("pgx", adminConnStr)
	if err != nil {
		db.t.Logf("Failed to connect to postgres database: %v", err)
		return
	}
	defer adminDB.Close()

	// Drop test database
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.DBName))
	if err != nil {
		db.t.Logf("Failed to drop test database: %v", err)
	}
}

// Truncate truncates all tables in the test database
func (db *TestDB) Truncate(tables ...string) {
	db.t.Helper()
	ctx := context.Background()

	for _, table := range tables {
		_, err := db.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(db.t, err, "Failed to truncate table %s", table)
	}
}

// RunMigrations runs database migrations on the test database
func (db *TestDB) RunMigrations(migrationsPath string) {
	db.t.Helper()
	// This would integrate with golang-migrate
	// For now, this is a placeholder
}

// randomID generates a random ID for test database names
func randomID() int64 {
	return int64(1000000 + (1000 * testing.AllocsPerRun(1, func() {})))
}
