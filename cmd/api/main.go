package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/davidmoltin/intelligent-workflows/internal/api/rest"
	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/handlers"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/database"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	log, err := logger.New(cfg.Logger.Level, cfg.Logger.Format)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer log.Sync()

	logger.SetDefault(log)
	log.Info("Starting Intelligent Workflows API",
		logger.String("version", cfg.App.Version),
		logger.String("environment", cfg.App.Environment),
	)

	// Initialize PostgreSQL
	db, err := database.NewPostgresDB(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Initialize Redis
	redis, err := database.NewRedisClient(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to initialize redis: %w", err)
	}
	defer redis.Close()

	// Initialize repositories
	workflowRepo := postgres.NewWorkflowRepository(db.DB)

	// Initialize handlers
	h := handlers.NewHandlers(
		log,
		workflowRepo,
		&handlers.HealthCheckers{
			DB:    db,
			Redis: redis,
		},
	)

	// Initialize router
	router := rest.NewRouter(log, h)
	router.SetupRoutes()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router.Handler(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("API server listening", logger.String("address", addr))
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for interrupt signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or an error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Info("Shutdown signal received", logger.String("signal", sig.String()))

		// Give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		// Gracefully shutdown the server
		if err := server.Shutdown(ctx); err != nil {
			server.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		log.Info("Server stopped gracefully")
	}

	return nil
}
