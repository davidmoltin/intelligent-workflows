package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/sony/gobreaker"
	_ "github.com/lib/pq"
)

// PostgresDB wraps the database connection
type PostgresDB struct {
	DB             *sql.DB
	circuitBreaker *gobreaker.CircuitBreaker
	logger         *logger.Logger
}

// NewPostgresDB creates a new PostgreSQL database connection with retry logic
func NewPostgresDB(cfg *config.Config, log *logger.Logger) (*PostgresDB, error) {
	dsn := cfg.DatabaseDSN()

	// Retry backoff schedule: 1s, 2s, 5s, 10s
	backoff := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
	}

	var db *sql.DB
	var err error

	// Attempt connection with exponential backoff
	for attempt := 0; attempt < len(backoff); attempt++ {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Warnf("Database connection attempt %d/%d failed: %v", attempt+1, len(backoff), err)
			if attempt < len(backoff)-1 {
				log.Infof("Retrying in %v...", backoff[attempt])
				time.Sleep(backoff[attempt])
			}
			continue
		}

		// Configure connection pool
		db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
		db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
		db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

		// Test connection with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()

		if err == nil {
			// Success!
			log.Info("PostgreSQL connection established",
				logger.String("host", cfg.Database.Host),
				logger.Int("port", cfg.Database.Port),
				logger.String("database", cfg.Database.Database),
				logger.Int("attempt", attempt+1),
			)

			// Initialize circuit breaker
			cb := initCircuitBreaker(log)

			return &PostgresDB{
				DB:             db,
				circuitBreaker: cb,
				logger:         log,
			}, nil
		}

		// Ping failed, close and retry
		db.Close()
		log.Warnf("Database ping attempt %d/%d failed: %v", attempt+1, len(backoff), err)

		if attempt < len(backoff)-1 {
			log.Infof("Retrying in %v...", backoff[attempt])
			time.Sleep(backoff[attempt])
		}
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", len(backoff), err)
}

// initCircuitBreaker creates and configures a circuit breaker for database operations
func initCircuitBreaker(log *logger.Logger) *gobreaker.CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        "database",
		MaxRequests: 3, // Max concurrent requests in half-open state
		Interval:    10 * time.Second,  // Window for counting failures
		Timeout:     60 * time.Second,  // Time to wait before half-open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit if we have at least 3 requests and 60% failure rate
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			shouldTrip := counts.Requests >= 3 && failureRatio >= 0.6

			if shouldTrip {
				log.Errorf(
					"Circuit breaker tripping: requests=%d, failures=%d, ratio=%.2f",
					counts.Requests,
					counts.TotalFailures,
					failureRatio,
				)
			}

			return shouldTrip
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Warnf("Circuit breaker state changed: %s -> %s", from.String(), to.String())
		},
	}

	return gobreaker.NewCircuitBreaker(settings)
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.DB.Close()
}

// HealthCheck performs a health check on the database
func (p *PostgresDB) HealthCheck(ctx context.Context) error {
	// Don't use circuit breaker for health checks as they're used to determine health
	return p.DB.PingContext(ctx)
}

// Stats returns database statistics
func (p *PostgresDB) Stats() sql.DBStats {
	return p.DB.Stats()
}

// ExecContext executes a query with circuit breaker protection
func (p *PostgresDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	result, err := p.circuitBreaker.Execute(func() (interface{}, error) {
		return p.DB.ExecContext(ctx, query, args...)
	})

	if err != nil {
		return nil, err
	}

	return result.(sql.Result), nil
}

// QueryContext executes a query with circuit breaker protection
func (p *PostgresDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	result, err := p.circuitBreaker.Execute(func() (interface{}, error) {
		return p.DB.QueryContext(ctx, query, args...)
	})

	if err != nil {
		return nil, err
	}

	return result.(*sql.Rows), nil
}

// QueryRowContext executes a query that returns a single row with circuit breaker protection
func (p *PostgresDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// QueryRow doesn't return an error, so we execute directly
	// The error is deferred until Scan() is called
	return p.DB.QueryRowContext(ctx, query, args...)
}

// CircuitBreakerState returns the current state of the circuit breaker
func (p *PostgresDB) CircuitBreakerState() gobreaker.State {
	return p.circuitBreaker.State()
}

// IsCircuitBreakerOpen returns true if the circuit breaker is open
func (p *PostgresDB) IsCircuitBreakerOpen() bool {
	return p.circuitBreaker.State() == gobreaker.StateOpen
}
