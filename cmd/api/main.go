package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/api/rest"
	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/handlers"
	"github.com/davidmoltin/intelligent-workflows/internal/engine"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/internal/workers"
	"github.com/davidmoltin/intelligent-workflows/pkg/auth"
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
	executionRepo := postgres.NewExecutionRepository(db.DB)
	eventRepo := postgres.NewEventRepository(db.DB)
	approvalRepo := postgres.NewApprovalRepository(db.DB)
	userRepo := postgres.NewUserRepository(db.DB)
	apiKeyRepo := postgres.NewAPIKeyRepository(db.DB)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(db.DB)

	// Initialize workflow engine components
	executor := engine.NewWorkflowExecutor(redis.Client, executionRepo, log)
	eventRouter := engine.NewEventRouter(workflowRepo, eventRepo, executor, log)

	// Initialize notification service
	notificationService, err := services.NewNotificationService(&cfg.Notification, log)
	if err != nil {
		return fmt.Errorf("failed to initialize notification service: %w", err)
	}

	// Initialize workflow resumer
	workflowResumer := services.NewWorkflowResumer(log)

	// Initialize JWT manager
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Fail in production if JWT_SECRET is not set
		if cfg.App.Environment == "production" {
			return fmt.Errorf("JWT_SECRET environment variable must be set in production")
		}
		// Allow default in development, but warn
		jwtSecret = "default-secret-change-this-in-production"
		log.Warn("JWT_SECRET not set, using default (INSECURE - only for development)")
	}
	jwtManager := auth.NewJWTManager(jwtSecret)

	// Initialize services
	approvalService := services.NewApprovalService(approvalRepo, log, notificationService, workflowResumer)
	authService := services.NewAuthService(userRepo, apiKeyRepo, refreshTokenRepo, jwtManager, log)

	// Initialize and start approval expiration worker
	expirationWorker := workers.NewApprovalExpirationWorker(approvalService, log, 5*time.Minute)
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	expirationWorker.Start(workerCtx)

	// Initialize handlers
	h := handlers.NewHandlers(
		log,
		workflowRepo,
		executionRepo,
		eventRouter,
		approvalService,
		authService,
		&handlers.HealthCheckers{
			DB:    db,
			Redis: redis,
		},
	)

	// Initialize router
	router := rest.NewRouter(log, h, authService)
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

		// Stop background workers first
		expirationWorker.Stop()

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
