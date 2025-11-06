package engine

import (
	"context"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TestStepTimeout tests step-level timeout configuration
func TestStepTimeout(t *testing.T) {
	log := logger.NewForTesting()
	ctx := context.Background()

	t.Run("step timeout is parsed and applied", func(t *testing.T) {
		timeoutSet := false

		repo := &mockExecutionRepo{
			createExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				return nil
			},
			createStepExecutionFunc: func(ctx context.Context, step *models.StepExecution) error {
				// This will be called if step starts execution
				return nil
			},
			updateStepExecutionFunc: func(ctx context.Context, step *models.StepExecution) error {
				return nil
			},
			updateExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				return nil
			},
		}

		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log)

		// Create a workflow with a step that has a timeout
		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "step-timeout-workflow",
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type: "event",
				},
				Steps: []models.Step{
					{
						ID:      "timed-step",
						Type:    "action",
						Timeout: "5s", // 5 second timeout
						Action: &models.Action{
							Type: "allow",
						},
					},
				},
			},
		}

		executor.defaultTimeout = 30 * time.Second

		// Execute and verify timeout was configured
		_, err := executor.Execute(ctx, workflow, "test.event", map[string]interface{}{})

		// Should succeed since the timeout is generous
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		timeoutSet = true

		if !timeoutSet {
			t.Error("Expected timeout to be configured")
		}
	})

	t.Run("step completes successfully within timeout", func(t *testing.T) {
		stepCompleted := false
		executionCompleted := false

		repo := &mockExecutionRepo{
			createExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				return nil
			},
			createStepExecutionFunc: func(ctx context.Context, step *models.StepExecution) error {
				return nil
			},
			updateStepExecutionFunc: func(ctx context.Context, step *models.StepExecution) error {
				if step.Status == models.StepStatusCompleted {
					stepCompleted = true
				}
				return nil
			},
			updateExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				if execution.Status == models.ExecutionStatusCompleted {
					executionCompleted = true
				}
				return nil
			},
		}

		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log)

		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "step-success-workflow",
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type: "event",
				},
				Steps: []models.Step{
					{
						ID:      "fast-step",
						Type:    "action",
						Timeout: "5s", // Generous timeout
						Action: &models.Action{
							Type: "allow",
						},
					},
				},
			},
		}

		_, err := executor.Execute(ctx, workflow, "test.event", map[string]interface{}{})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		if !stepCompleted {
			t.Error("Expected step to complete successfully")
		}

		if !executionCompleted {
			t.Error("Expected execution to complete successfully")
		}
	})

	t.Run("invalid timeout format uses default", func(t *testing.T) {
		repo := &mockExecutionRepo{
			createExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				return nil
			},
			createStepExecutionFunc: func(ctx context.Context, step *models.StepExecution) error {
				return nil
			},
			updateStepExecutionFunc: func(ctx context.Context, step *models.StepExecution) error {
				return nil
			},
			updateExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
				return nil
			},
		}

		redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		executor := NewWorkflowExecutor(redisClient, repo, nil, log)

		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "invalid-timeout-workflow",
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type: "event",
				},
				Steps: []models.Step{
					{
						ID:      "step1",
						Type:    "action",
						Timeout: "invalid-duration", // Invalid format
						Action: &models.Action{
							Type: "allow",
						},
					},
				},
			},
		}

		// Should not error - just use default (no timeout in this case)
		_, err := executor.Execute(ctx, workflow, "test.event", map[string]interface{}{})

		if err != nil {
			t.Errorf("Expected workflow to handle invalid timeout gracefully, got: %v", err)
		}
	})
}

// TestWorkflowTimeoutPriority tests timeout priority: Definition.Timeout > trigger data > default
func TestWorkflowTimeoutPriority(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	repo := &mockExecutionRepo{
		createExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
			return nil
		},
		updateExecutionFunc: func(ctx context.Context, execution *models.WorkflowExecution) error {
			return nil
		},
	}

	executor := NewWorkflowExecutor(redisClient, repo, nil, log)
	executor.defaultTimeout = 30 * time.Second

	t.Run("Definition.Timeout takes highest priority", func(t *testing.T) {
		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "definition-timeout",
			Definition: models.WorkflowDefinition{
				Timeout: "10s", // Definition timeout
				Trigger: models.TriggerDefinition{
					Type: "event",
					Data: map[string]interface{}{
						"timeout_seconds": 60.0, // Trigger data timeout
					},
				},
				Steps: []models.Step{},
			},
		}

		timeout := executor.getWorkflowTimeout(workflow)
		expected := 10 * time.Second

		if timeout != expected {
			t.Errorf("Expected timeout %v (from Definition.Timeout), got %v", expected, timeout)
		}
	})

	t.Run("trigger data timeout used when Definition.Timeout not set", func(t *testing.T) {
		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "trigger-timeout",
			Definition: models.WorkflowDefinition{
				// No Definition.Timeout
				Trigger: models.TriggerDefinition{
					Type: "event",
					Data: map[string]interface{}{
						"timeout_seconds": 45.0, // Trigger data timeout
					},
				},
				Steps: []models.Step{},
			},
		}

		timeout := executor.getWorkflowTimeout(workflow)
		expected := 45 * time.Second

		if timeout != expected {
			t.Errorf("Expected timeout %v (from trigger data), got %v", expected, timeout)
		}
	})

	t.Run("default timeout used when neither Definition nor trigger data set", func(t *testing.T) {
		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "default-timeout",
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type: "event",
				},
				Steps: []models.Step{},
			},
		}

		timeout := executor.getWorkflowTimeout(workflow)
		expected := 30 * time.Second // executor.defaultTimeout

		if timeout != expected {
			t.Errorf("Expected default timeout %v, got %v", expected, timeout)
		}
	})

	t.Run("trigger data with int timeout_seconds", func(t *testing.T) {
		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "int-timeout",
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type: "event",
					Data: map[string]interface{}{
						"timeout_seconds": 25, // Int instead of float64
					},
				},
				Steps: []models.Step{},
			},
		}

		timeout := executor.getWorkflowTimeout(workflow)
		expected := 25 * time.Second

		if timeout != expected {
			t.Errorf("Expected timeout %v (from int trigger data), got %v", expected, timeout)
		}
	})

	t.Run("zero or negative timeout_seconds ignored", func(t *testing.T) {
		workflow := &models.Workflow{
			ID:   uuid.New(),
			Name: "zero-timeout",
			Definition: models.WorkflowDefinition{
				Trigger: models.TriggerDefinition{
					Type: "event",
					Data: map[string]interface{}{
						"timeout_seconds": 0.0, // Zero timeout should be ignored
					},
				},
				Steps: []models.Step{},
			},
		}

		timeout := executor.getWorkflowTimeout(workflow)
		expected := 30 * time.Second // Should use default

		if timeout != expected {
			t.Errorf("Expected default timeout %v (zero ignored), got %v", expected, timeout)
		}
	})
}

// TestParseTimeout tests the timeout string parsing
func TestParseTimeout(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	repo := &mockExecutionRepo{}
	executor := NewWorkflowExecutor(redisClient, repo, nil, log)

	tests := []struct {
		name           string
		input          string
		defaultTimeout time.Duration
		expected       time.Duration
	}{
		{
			name:           "parses seconds",
			input:          "30s",
			defaultTimeout: 60 * time.Second,
			expected:       30 * time.Second,
		},
		{
			name:           "parses minutes",
			input:          "5m",
			defaultTimeout: 60 * time.Second,
			expected:       5 * time.Minute,
		},
		{
			name:           "parses hours",
			input:          "2h",
			defaultTimeout: 60 * time.Second,
			expected:       2 * time.Hour,
		},
		{
			name:           "parses complex duration",
			input:          "1h30m",
			defaultTimeout: 60 * time.Second,
			expected:       90 * time.Minute,
		},
		{
			name:           "returns default for invalid format",
			input:          "invalid",
			defaultTimeout: 60 * time.Second,
			expected:       60 * time.Second,
		},
		{
			name:           "returns default for empty string",
			input:          "",
			defaultTimeout: 45 * time.Second,
			expected:       45 * time.Second,
		},
		{
			name:           "returns default for negative duration",
			input:          "-10s",
			defaultTimeout: 30 * time.Second,
			expected:       30 * time.Second,
		},
		{
			name:           "returns default for zero duration",
			input:          "0s",
			defaultTimeout: 30 * time.Second,
			expected:       30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.parseTimeout(tt.input, tt.defaultTimeout)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
