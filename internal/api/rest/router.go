package rest

import (
	"net/http"

	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/handlers"
	customMiddleware "github.com/davidmoltin/intelligent-workflows/internal/api/rest/middleware"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Router holds the HTTP router and dependencies
type Router struct {
	router      *chi.Mux
	logger      *logger.Logger
	handlers    *handlers.Handlers
	authService *services.AuthService
}

// NewRouter creates a new HTTP router
func NewRouter(log *logger.Logger, h *handlers.Handlers, authService *services.AuthService) *Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.Logger(log))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // TODO: Configure from environment
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	return &Router{
		router:      r,
		logger:      log,
		handlers:    h,
		authService: authService,
	}
}

// SetupRoutes configures all API routes
func (r *Router) SetupRoutes() {
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
			})

			// Events
			router.Route("/events", func(router chi.Router) {
				router.With(customMiddleware.RequirePermission("event:create", r.logger)).Post("/", r.handlers.Event.CreateEvent)
			})

			// Executions
			router.Route("/executions", func(router chi.Router) {
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/", r.handlers.Execution.ListExecutions)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/{id}", r.handlers.Execution.GetExecution)
				router.With(customMiddleware.RequirePermission("execution:read", r.logger)).Get("/{id}/trace", r.handlers.Execution.GetExecutionTrace)
			})

			// Approvals
			router.Route("/approvals", func(router chi.Router) {
				router.With(customMiddleware.RequirePermission("approval:read", r.logger)).Get("/", r.handlers.Approval.ListApprovals)
				router.With(customMiddleware.RequirePermission("approval:read", r.logger)).Get("/{id}", r.handlers.Approval.GetApproval)
				router.With(customMiddleware.RequirePermission("approval:approve", r.logger)).Post("/{id}/approve", r.handlers.Approval.ApproveRequest)
				router.With(customMiddleware.RequirePermission("approval:reject", r.logger)).Post("/{id}/reject", r.handlers.Approval.RejectRequest)
			})
		})
	})
}

// Handler returns the http.Handler
func (r *Router) Handler() http.Handler {
	return r.router
}
