package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/davidmoltin/intelligent-workflows/internal/api/rest/handlers"
	customMiddleware "github.com/davidmoltin/intelligent-workflows/internal/api/rest/middleware"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// Router holds the HTTP router and dependencies
type Router struct {
	router   *chi.Mux
	logger   *logger.Logger
	handlers *handlers.Handlers
}

// NewRouter creates a new HTTP router
func NewRouter(log *logger.Logger, h *handlers.Handlers) *Router {
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
		router:   r,
		logger:   log,
		handlers: h,
	}
}

// SetupRoutes configures all API routes
func (r *Router) SetupRoutes() {
	// Health endpoints
	r.router.Get("/health", r.handlers.Health.Health)
	r.router.Get("/ready", r.handlers.Health.Ready)

	// API v1
	r.router.Route("/api/v1", func(router chi.Router) {
		// Workflows
		router.Route("/workflows", func(router chi.Router) {
			router.Post("/", r.handlers.Workflow.Create)
			router.Get("/", r.handlers.Workflow.List)
			router.Get("/{id}", r.handlers.Workflow.Get)
			router.Put("/{id}", r.handlers.Workflow.Update)
			router.Delete("/{id}", r.handlers.Workflow.Delete)
			router.Post("/{id}/enable", r.handlers.Workflow.Enable)
			router.Post("/{id}/disable", r.handlers.Workflow.Disable)
		})

		// Events
		router.Route("/events", func(router chi.Router) {
			router.Post("/", r.handlers.Event.CreateEvent)
		})

		// Executions
		router.Route("/executions", func(router chi.Router) {
			router.Get("/", r.handlers.Execution.ListExecutions)
			router.Get("/{id}", r.handlers.Execution.GetExecution)
			router.Get("/{id}/trace", r.handlers.Execution.GetExecutionTrace)
		})

		// Approvals
		router.Route("/approvals", func(router chi.Router) {
			router.Get("/", r.handlers.Approval.ListApprovals)
			router.Get("/{id}", r.handlers.Approval.GetApproval)
			router.Post("/{id}/approve", r.handlers.Approval.ApproveRequest)
			router.Post("/{id}/reject", r.handlers.Approval.RejectRequest)
		})
	})
}

// Handler returns the http.Handler
func (r *Router) Handler() http.Handler {
	return r.router
}
