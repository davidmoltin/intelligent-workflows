package testutil

import (
	"database/sql"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
)

// RunMigrations runs database migrations for testing
func RunMigrations(t *testing.T, db *TestDB, migrationsPath string) {
	t.Helper()

	// Get underlying *sql.DB from pgxpool
	connStr := db.Pool.Config().ConnString()
	sqlDB, err := sql.Open("pgx", connStr)
	require.NoError(t, err, "Failed to open sql.DB connection")
	defer sqlDB.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	require.NoError(t, err, "Failed to create postgres driver")

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	require.NoError(t, err, "Failed to create migrate instance")

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, "Failed to run migrations")
	}

	t.Cleanup(func() {
		m.Close()
		sqlDB.Close()
	})
}

// MigrateDown rolls back all migrations for testing
func MigrateDown(t *testing.T, db *TestDB, migrationsPath string) {
	t.Helper()

	connStr := db.Pool.Config().ConnString()
	sqlDB, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	defer sqlDB.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	require.NoError(t, err)

	err = m.Down()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err, "Failed to rollback migrations")
	}
}
