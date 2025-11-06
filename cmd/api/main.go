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
	"github.com/davidmoltin/intelligent-workflows/pkg/llm"
	"github.com/davidmoltin/intelligent-workflows/pkg/llm/providers/anthropic"
	"github.com/davidmoltin/intelligent-workflows/pkg/llm/providers/openai"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"go.uber.org/zap"
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
	executor := engine.NewWorkflowExecutor(redis.Client, executionRepo, workflowRepo, log)
	eventRouter := engine.NewEventRouter(workflowRepo, eventRepo, executor, log)

	// Initialize notification service
	notificationService, err := services.NewNotificationService(&cfg.Notification, log)
	if err != nil {
		return fmt.Errorf("failed to initialize notification service: %w", err)
	}

	// Initialize workflow resumer
	workflowResumer := services.NewWorkflowResumer(log, executionRepo, executor)

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

	// Initialize LLM client (if configured)
	var aiService *services.AIService
	if cfg.LLM.APIKey != "" {
		llmConfig := &llm.Config{
			Provider:     llm.Provider(cfg.LLM.Provider),
			APIKey:       cfg.LLM.APIKey,
			DefaultModel: cfg.LLM.DefaultModel,
			Timeout:      cfg.LLM.Timeout,
			MaxRetries:   cfg.LLM.MaxRetries,
			RetryDelay:   cfg.LLM.RetryDelay,
			BaseURL:      cfg.LLM.BaseURL,
		}

		var llmClient llm.Client
		switch llm.Provider(cfg.LLM.Provider) {
		case llm.ProviderAnthropic:
			llmClient, err = anthropic.NewClient(llmConfig)
		case llm.ProviderOpenAI:
			llmClient, err = openai.NewClient(llmConfig)
		default:
			log.Warn("Unknown LLM provider, AI features will be disabled",
				logger.String("provider", cfg.LLM.Provider))
		}

		if err != nil {
			log.Warn("Failed to initialize LLM client, AI features will be disabled",
				zap.Error(err),
				logger.String("provider", cfg.LLM.Provider))
		} else if llmClient != nil {
			aiService, err = services.NewAIService(llmClient, log.Logger)
			if err != nil {
				log.Warn("Failed to initialize AI service, AI features will be disabled",
					zap.Error(err))
			} else {
				log.Info("AI service initialized",
					logger.String("provider", string(llmClient.GetProvider())))
				defer aiService.Close()
			}
		}
	} else {
		log.Info("LLM API key not configured, AI features will be disabled")
	}

	// Initialize services
	approvalService := services.NewApprovalService(approvalRepo, log, notificationService, workflowResumer, cfg.App.DefaultApproverEmail)
	authService := services.NewAuthService(userRepo, apiKeyRepo, refreshTokenRepo, jwtManager, log)

	// Initialize and start approval expiration worker
	expirationWorker := workers.NewApprovalExpirationWorker(approvalService, log, 5*time.Minute)
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	expirationWorker.Start(workerCtx)

	// Initialize and start workflow resumer worker
	resumerWorker := workers.NewWorkflowResumerWorker(workflowResumer, log, 1*time.Minute)
	resumerWorker.Start(workerCtx)

	// Initialize handlers
	h := handlers.NewHandlers(
		log,
		workflowRepo,
		executionRepo,
		eventRouter,
		approvalService,
		authService,
		workflowResumer,
		aiService,
		&handlers.HealthCheckers{
			DB:    db,
			Redis: redis,
		},
		cfg.App.Version,
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
