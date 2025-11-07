package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server            ServerConfig
	Database          DatabaseConfig
	Redis             RedisConfig
	Logger            LoggerConfig
	App               AppConfig
	Notification      NotificationConfig
	LLM               LLMConfig
	Workers           WorkersConfig
	ContextEnrichment ContextEnrichmentConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level  string
	Format string // json or text
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	Environment          string
	Version              string
	Name                 string
	DefaultApproverEmail string
}

// NotificationConfig holds notification service configuration
type NotificationConfig struct {
	BaseURL string
	Email   EmailConfig
	Slack   SlackConfig
}

// EmailConfig holds email notification configuration
type EmailConfig struct {
	Enabled      bool
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromAddress  string
}

// SlackConfig holds Slack notification configuration
type SlackConfig struct {
	Enabled    bool
	WebhookURL string
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider     string
	APIKey       string
	DefaultModel string
	Timeout      time.Duration
	MaxRetries   int
	RetryDelay   time.Duration
	BaseURL      string
}

// WorkersConfig holds background worker configuration
type WorkersConfig struct {
	ApprovalExpirationCheckInterval time.Duration
	WorkflowResumerCheckInterval    time.Duration
	TimeoutEnforcerCheckInterval    time.Duration
	SchedulerCheckInterval          time.Duration
}

// ContextEnrichmentConfig holds context enrichment service configuration
type ContextEnrichmentConfig struct {
	Enabled         bool
	BaseURL         string
	Timeout         time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	CacheTTL        time.Duration
	EndpointMapping map[string]string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Database:        getEnv("DB_NAME", "workflows"),
			SSLMode:         getEnv("DB_SSL_MODE", "require"), // Default to require for security
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		App: AppConfig{
			Environment:          getEnv("APP_ENV", "development"),
			Version:              getEnv("APP_VERSION", "0.1.0"),
			Name:                 getEnv("APP_NAME", "intelligent-workflows"),
			DefaultApproverEmail: getEnv("DEFAULT_APPROVER_EMAIL", "approver@example.com"),
		},
		Notification: NotificationConfig{
			BaseURL: getEnv("NOTIFICATION_BASE_URL", "http://localhost:8080"),
			Email: EmailConfig{
				Enabled:      getEnvAsBool("NOTIFICATION_EMAIL_ENABLED", false),
				SMTPHost:     getEnv("NOTIFICATION_SMTP_HOST", "smtp.gmail.com"),
				SMTPPort:     getEnvAsInt("NOTIFICATION_SMTP_PORT", 587),
				SMTPUser:     getEnv("NOTIFICATION_SMTP_USER", ""),
				SMTPPassword: getEnv("NOTIFICATION_SMTP_PASSWORD", ""),
				FromAddress:  getEnv("NOTIFICATION_FROM_ADDRESS", "noreply@example.com"),
			},
			Slack: SlackConfig{
				Enabled:    getEnvAsBool("NOTIFICATION_SLACK_ENABLED", false),
				WebhookURL: getEnv("NOTIFICATION_SLACK_WEBHOOK_URL", ""),
			},
		},
		LLM: LLMConfig{
			Provider:     getEnv("LLM_PROVIDER", "anthropic"),
			APIKey:       getEnv("LLM_API_KEY", ""),
			DefaultModel: getEnv("LLM_DEFAULT_MODEL", ""),
			Timeout:      getEnvAsDuration("LLM_TIMEOUT", 60*time.Second),
			MaxRetries:   getEnvAsInt("LLM_MAX_RETRIES", 3),
			RetryDelay:   getEnvAsDuration("LLM_RETRY_DELAY", 1*time.Second),
			BaseURL:      getEnv("LLM_BASE_URL", ""),
		},
		Workers: WorkersConfig{
			ApprovalExpirationCheckInterval: getEnvAsDuration("WORKER_APPROVAL_EXPIRATION_INTERVAL", 5*time.Minute),
			WorkflowResumerCheckInterval:    getEnvAsDuration("WORKER_WORKFLOW_RESUMER_INTERVAL", 1*time.Minute),
			TimeoutEnforcerCheckInterval:    getEnvAsDuration("WORKER_TIMEOUT_ENFORCER_INTERVAL", 1*time.Minute),
			SchedulerCheckInterval:          getEnvAsDuration("WORKER_SCHEDULER_INTERVAL", 1*time.Minute),
		},
		ContextEnrichment: ContextEnrichmentConfig{
			Enabled:    getEnvAsBool("CONTEXT_ENRICHMENT_ENABLED", true),
			BaseURL:    getEnv("CONTEXT_ENRICHMENT_BASE_URL", "http://localhost:8081"),
			Timeout:    getEnvAsDuration("CONTEXT_ENRICHMENT_TIMEOUT", 10*time.Second),
			MaxRetries: getEnvAsInt("CONTEXT_ENRICHMENT_MAX_RETRIES", 3),
			RetryDelay: getEnvAsDuration("CONTEXT_ENRICHMENT_RETRY_DELAY", 500*time.Millisecond),
			CacheTTL:   getEnvAsDuration("CONTEXT_ENRICHMENT_CACHE_TTL", 5*time.Minute),
			EndpointMapping: map[string]string{
				"order.details":      "/api/v1/orders/{id}/details",
				"customer.history":   "/api/v1/customers/{id}/history",
				"product.inventory":  "/api/v1/products/{id}/inventory",
				"payment.status":     "/api/v1/payments/{id}/status",
				"shipment.tracking":  "/api/v1/shipments/{id}/tracking",
				"user.preferences":   "/api/v1/users/{id}/preferences",
				"subscription.info":  "/api/v1/subscriptions/{id}/info",
				"invoice.details":    "/api/v1/invoices/{id}/details",
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}

	return nil
}

// DatabaseDSN returns the PostgreSQL connection string
func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// RedisAddr returns the Redis address
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
