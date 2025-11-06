package rest

import (
	"net/http"
	"os"
	"strings"

	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/handlers"
	customMiddleware "github.com/davidmoltin/intelligent-workflows/internal/api/rest/middleware"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/davidmoltin/intelligent-workflows/pkg/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Router holds the HTTP router and dependencies
type Router struct {
	router      *chi.Mux
	logger      *logger.Logger
	handlers    *handlers.Handlers
	authService *services.AuthService
	metrics     *metrics.Metrics
}

// NewRouter creates a new HTTP router
func NewRouter(log *logger.Logger, h *handlers.Handlers, authService *services.AuthService, m *metrics.Metrics) *Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.Logger(log))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Metrics middleware
	r.Use(customMiddleware.Metrics(m))

	// Security middleware
	r.Use(customMiddleware.SecurityHeaders())
	r.Use(customMiddleware.RequestSizeLimit(customMiddleware.GetMaxRequestSize()))

	// CORS - Configure allowed origins from environment
	allowedOrigins := []string{"http://localhost:3000"} // Default for development
	if originsEnv := os.Getenv("ALLOWED_ORIGINS"); originsEnv != "" {
		allowedOrigins = strings.Split(originsEnv, ",")
		// Trim whitespace from each origin
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
	}

	// Security: Never allow "*" with credentials enabled
	allowCredentials := true
	for _, origin := range allowedOrigins {
		if origin == "*" {
			log.Warn("CORS: Wildcard origin '*' detected with credentials enabled. Disabling credentials for security.")
			allowCredentials = false
			break
		}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: allowCredentials,
		MaxAge:           300,
	}))

	return &Router{
		router:      r,
		logger:      log,
		handlers:    h,
		authService: authService,
		metrics:     m,
	}
}

// SetupRoutes configures all API routes
func (r *Router) SetupRoutes() {
	// Prometheus metrics endpoint (no auth required)
	r.router.Handle("/metrics", promhttp.Handler())

	// Health endpoints (no auth required)
	r.router.Get("/health", r.handlers.Health.Health)
	r.router.Get("/ready", r.handlers.Health.Ready)

	// API v1
	r.router.Route("/api/v1", func(router chi.Router) {
		// API Documentation (public)
		router.Route("/docs", func(router chi.Router) {
			router.Get("/", r.handlers.Docs.RedirectToDocs)
			router.Get("/ui", r.handlers.Docs.ServeSwaggerUI)
			router.Get("/openapi.yaml", r.handlers.Docs.ServeOpenAPISpec)
		})

		// Auth endpoints (public)
		router.Route("/auth", func(router chi.Router) {
			router.Post("/register", r.handlers.Auth.Register)
			router.Post("/login", r.handlers.Auth.Login)
			router.Post("/refresh", r.handlers.Auth.RefreshToken)
			router.Post("/logout", r.handlers.Auth.Logout)

			// Protected auth endpoints (require authentication)
			router.Group(func(router chi.Router) {
				router.Use(customMiddleware.JWTAuth(r.authService, r.logger))
				router.Get("/me", r.handlers.Auth.Me)
				router.Post("/change-password", r.handlers.Auth.ChangePassword)
				router.Post("/api-keys", r.handlers.Auth.CreateAPIKey)
				router.Delete("/api-keys/{id}", r.handlers.Auth.RevokeAPIKey)
			})
		})

		// Protected routes (require authentication)
		router.Group(func(router chi.Router) {
			// Apply optional auth (JWT or API key)
			router.Use(customMiddleware.OptionalAuth(r.authService, r.logger))

			// Apply rate limiting (100 requests per minute per user)
			router.Use(customMiddleware.RateLimitWithConfig(100, 200, r.logger))

			// Workflows
			router.Route("/workflows", func(router chi.Router) {
				// Read operations
				router.With(customMiddleware.RequirePermission("workflow:read", r.logger)).Get("/", r.handlers.Workflow.List)
				router.With(customMiddleware.RequirePermission("workflow:read", r.logger)).Get("/{id}", r.handlers.Workflow.Get)

				// Write operations
				router.With(customMiddleware.RequirePermission("workflow:create", r.logger)).Post("/", r.handlers.Workflow.Create)
				router.With(customMiddleware.RequirePermission("workflow:update", r.logger)).Put("/{id}", r.handlers.Workflow.Update)
				router.With(customMiddleware.RequirePermission("workflow:delete", r.logger)).Delete("/{id}", r.handlers.Workflow.Delete)
				router.With(customMiddleware.RequirePermission("workflow:update", r.logger)).Post("/{id}/enable", r.handlers.Workflow.Enable)
				router.With(customMiddleware.RequirePermission("workflow:update", r.logger)).Post("/{id}/disable", r.handlers.Workflow.Disable)

				// Schedule operations
				router.With(customMiddleware.RequirePermission("workflow:read", r.logger)).Get("/{id}/schedules", r.handlers.Schedule.GetWorkflowSchedules)
				router.With(customMiddleware.RequirePermission("workflow:update", r.logger)).Post("/{id}/schedules", r.handlers.Schedule.CreateSchedule)
			})

			// Events
			router.Route("/events", func(router chi.Router) {
				router.With(customMiddleware.RequirePermission("event:create", r.logger)).Post("/", r.handlers.Event.CreateEvent)
			})

			// Executions
			router.Route("/executions", func(router chi.Router) {
				// List and read operations
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/", r.handlers.Execution.ListExecutions)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/paused", r.handlers.Execution.ListPausedExecutions)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/{id}", r.handlers.Execution.GetExecution)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/{id}/trace", r.handlers.Execution.GetExecutionTrace)

				// Control operations
				router.With(customMiddleware.RequirePermission("execution:cancel", r.logger)).Post("/{id}/pause", r.handlers.Execution.PauseExecution)
				router.With(customMiddleware.RequirePermission("execution:cancel", r.logger)).Post("/{id}/resume", r.handlers.Execution.ResumeExecution)
			})

			// Approvals
			router.Route("/approvals", func(router chi.Router) {
				router.With(customMiddleware.RequirePermission("approval:read", r.logger)).Get("/", r.handlers.Approval.ListApprovals)
				router.With(customMiddleware.RequirePermission("approval:read", r.logger)).Get("/{id}", r.handlers.Approval.GetApproval)
				router.With(customMiddleware.RequirePermission("approval:approve", r.logger)).Post("/{id}/approve", r.handlers.Approval.ApproveRequest)
				router.With(customMiddleware.RequirePermission("approval:reject", r.logger)).Post("/{id}/reject", r.handlers.Approval.RejectRequest)
			})

			// Analytics
			router.Route("/analytics", func(router chi.Router) {
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/dashboard", r.handlers.Analytics.GetDashboard)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/stats", r.handlers.Analytics.GetExecutionStats)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/trends", r.handlers.Analytics.GetExecutionTrends)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/workflows", r.handlers.Analytics.GetWorkflowStats)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/errors", r.handlers.Analytics.GetRecentErrors)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/steps", r.handlers.Analytics.GetStepStats)
			})

			// Schedules
			router.Route("/schedules", func(router chi.Router) {
				router.With(customMiddleware.RequirePermission("workflow:read", r.logger)).Get("/", r.handlers.Schedule.ListSchedules)
				router.With(customMiddleware.RequirePermission("workflow:read", r.logger)).Get("/{id}", r.handlers.Schedule.GetSchedule)
				router.With(customMiddleware.RequirePermission("workflow:update", r.logger)).Put("/{id}", r.handlers.Schedule.UpdateSchedule)
				router.With(customMiddleware.RequirePermission("workflow:delete", r.logger)).Delete("/{id}", r.handlers.Schedule.DeleteSchedule)
				router.With(customMiddleware.RequirePermission("workflow:read", r.logger)).Get("/{id}/next-runs", r.handlers.Schedule.GetNextRuns)
			})

			// Audit logs (only if audit handler is configured)
			if r.handlers.Audit != nil {
				router.Route("/audit-logs", func(router chi.Router) {
					router.With(customMiddleware.RequirePermission("audit:read", r.logger)).Get("/", r.handlers.Audit.ListAuditLogs)
					router.With(customMiddleware.RequirePermission("audit:read", r.logger)).Get("/{id}", r.handlers.Audit.GetAuditLog)
					router.With(customMiddleware.RequirePermission("audit:read", r.logger)).Get("/entity/{entity_type}/{entity_id}", r.handlers.Audit.GetEntityAuditLogs)
					router.With(customMiddleware.RequirePermission("audit:read", r.logger)).Get("/actor/{actor_id}", r.handlers.Audit.GetActorAuditLogs)
				})
			}

			// Organizations
			router.Route("/organizations", func(router chi.Router) {
				// List user's organizations and create
				router.With(customMiddleware.RequirePermission("organization:read", r.logger)).Get("/", r.handlers.Organization.List)
				router.With(customMiddleware.RequirePermission("organization:create", r.logger)).Post("/", r.handlers.Organization.Create)

				// Get by slug
				router.With(customMiddleware.RequirePermission("organization:read", r.logger)).Get("/slug/{slug}", r.handlers.Organization.GetBySlug)

				// Operations on specific organizations
				router.Route("/{id}", func(router chi.Router) {
					router.With(customMiddleware.RequirePermission("organization:read", r.logger)).Get("/", r.handlers.Organization.Get)
					router.With(customMiddleware.RequirePermission("organization:update", r.logger)).Put("/", r.handlers.Organization.Update)
					router.With(customMiddleware.RequirePermission("organization:delete", r.logger)).Delete("/", r.handlers.Organization.Delete)

					// User management within organization
					router.With(customMiddleware.RequirePermission("organization:read", r.logger)).Get("/users", r.handlers.Organization.ListUsers)
					router.With(customMiddleware.RequirePermission("organization:manage_users", r.logger)).Post("/users", r.handlers.Organization.AddUser)
					router.With(customMiddleware.RequirePermission("organization:manage_users", r.logger)).Delete("/users/{user_id}", r.handlers.Organization.RemoveUser)
					router.With(customMiddleware.RequirePermission("organization:manage_users", r.logger)).Put("/users/{user_id}/role", r.handlers.Organization.UpdateUserRole)

					// Access check
					router.With(customMiddleware.RequirePermission("organization:read", r.logger)).Get("/access", r.handlers.Organization.CheckUserAccess)
				})
			})
		})

		// AI endpoints (only if AI service is configured)
		if r.handlers.AI != nil {
			router.Route("/ai", func(router chi.Router) {
				router.Post("/chat", r.handlers.AI.Chat)
				router.Get("/capabilities", r.handlers.AI.GetCapabilities)
				router.Post("/interpret", r.handlers.AI.InterpretWorkflow)
			})
		}
	})
}

// Handler returns the http.Handler
func (r *Router) Handler() http.Handler {
	return r.router
}
