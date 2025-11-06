package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// HTTP Metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPDuration        *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// Workflow Metrics
	WorkflowExecutionsTotal *prometheus.CounterVec
	WorkflowDuration        *prometheus.HistogramVec
	WorkflowStepDuration    *prometheus.HistogramVec
	WorkflowErrors          *prometheus.CounterVec
	ActiveWorkflows         *prometheus.GaugeVec

	// Database Metrics
	DBConnectionsActive      prometheus.Gauge
	DBConnectionsFailed      *prometheus.CounterVec
	DBQueryDuration          *prometheus.HistogramVec
	DBQueryErrors            *prometheus.CounterVec

	// Redis Metrics
	RedisConnectionsActive  prometheus.Gauge
	RedisConnectionsFailed  *prometheus.CounterVec
	RedisOperationDuration  *prometheus.HistogramVec
	RedisOperationErrors    *prometheus.CounterVec

	// Business Logic Metrics
	ApprovalsTotal          *prometheus.CounterVec
	ApprovalDuration        *prometheus.HistogramVec
	NotificationsSent       *prometheus.CounterVec
	AIRequestsTotal         *prometheus.CounterVec
	AIRequestDuration       *prometheus.HistogramVec

	// Worker Metrics
	WorkerJobsProcessed     *prometheus.CounterVec
	WorkerJobDuration       *prometheus.HistogramVec
	WorkerErrors            *prometheus.CounterVec

	// Authentication Metrics
	AuthRequestsTotal       *prometheus.CounterVec
	AuthFailuresTotal       *prometheus.CounterVec
	AuthTokenValidations    *prometheus.CounterVec
}

// New creates and registers all Prometheus metrics
func New() *Metrics {
	m := &Metrics{
		// HTTP Metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path", "status"},
		),

		// Workflow Metrics
		WorkflowExecutionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_executions_total",
				Help: "Total number of workflow executions",
			},
			[]string{"workflow_id", "status"},
		),
		WorkflowDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_execution_duration_seconds",
				Help:    "Workflow execution duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
			},
			[]string{"workflow_id"},
		),
		WorkflowStepDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "workflow_step_duration_seconds",
				Help:    "Workflow step execution duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 0.01s to ~10s
			},
			[]string{"workflow_id", "step_name", "status"},
		),
		WorkflowErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "workflow_execution_errors_total",
				Help: "Total number of workflow execution errors",
			},
			[]string{"workflow_id", "error_type"},
		),
		ActiveWorkflows: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "active_workflow_executions",
				Help: "Number of currently running workflow executions",
			},
			[]string{"workflow_id"},
		),

		// Database Metrics
		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
			},
		),
		DBConnectionsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_connections_failed_total",
				Help: "Total number of failed database connection attempts",
			},
			[]string{"database"},
		),
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query execution time in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
			},
			[]string{"query_type", "table"},
		),
		DBQueryErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_query_errors_total",
				Help: "Total number of database query errors",
			},
			[]string{"query_type", "error_type"},
		),

		// Redis Metrics
		RedisConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "redis_connections_active",
				Help: "Number of active Redis connections",
			},
		),
		RedisConnectionsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_connections_failed_total",
				Help: "Total number of failed Redis connection attempts",
			},
			[]string{"operation"},
		),
		RedisOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "redis_operations_duration_seconds",
				Help:    "Redis operation duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
			},
			[]string{"operation"},
		),
		RedisOperationErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_operations_errors_total",
				Help: "Total number of Redis operation errors",
			},
			[]string{"operation"},
		),

		// Business Logic Metrics
		ApprovalsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "approvals_total",
				Help: "Total number of approvals",
			},
			[]string{"status", "workflow_id"},
		),
		ApprovalDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "approvals_duration_seconds",
				Help:    "Time to approval in seconds",
				Buckets: prometheus.ExponentialBuckets(60, 2, 10), // 1min to ~17hrs
			},
			[]string{"workflow_id"},
		),
		NotificationsSent: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_sent_total",
				Help: "Total number of notifications sent",
			},
			[]string{"type", "status"},
		),
		AIRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_requests_total",
				Help: "Total number of AI API requests",
			},
			[]string{"model", "status"},
		),
		AIRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ai_request_duration_seconds",
				Help:    "AI API request duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
			},
			[]string{"model"},
		),

		// Worker Metrics
		WorkerJobsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "worker_jobs_processed_total",
				Help: "Total number of jobs processed by workers",
			},
			[]string{"worker_type", "status"},
		),
		WorkerJobDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "worker_job_duration_seconds",
				Help:    "Worker job processing duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
			},
			[]string{"worker_type"},
		),
		WorkerErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "worker_errors_total",
				Help: "Total number of worker errors",
			},
			[]string{"worker_type"},
		),

		// Authentication Metrics
		AuthRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_requests_total",
				Help: "Total number of authentication requests",
			},
			[]string{"method", "status"},
		),
		AuthFailuresTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_failures_total",
				Help: "Total number of authentication failures",
			},
			[]string{"reason"},
		),
		AuthTokenValidations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_token_validations_total",
				Help: "Total number of token validation attempts",
			},
			[]string{"valid"},
		),
	}

	return m
}
