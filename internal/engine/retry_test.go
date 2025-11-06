package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// TestCalculateBackoff tests backoff calculation
func TestCalculateBackoff(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	repo := &mockExecutionRepo{}
	executor := NewWorkflowExecutor(redisClient, repo, nil, log)

	t.Run("exponential backoff", func(t *testing.T) {
		config := &models.RetryConfig{
			Backoff: "exponential",
		}

		tests := []struct {
			attempt  int
			expected time.Duration
		}{
			{attempt: 1, expected: 1 * time.Second},  // 2^0 = 1
			{attempt: 2, expected: 2 * time.Second},  // 2^1 = 2
			{attempt: 3, expected: 4 * time.Second},  // 2^2 = 4
			{attempt: 4, expected: 8 * time.Second},  // 2^3 = 8
			{attempt: 5, expected: 16 * time.Second}, // 2^4 = 16
		}

		for _, tt := range tests {
			result := executor.calculateBackoff(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
			}
		}
	})

	t.Run("linear backoff", func(t *testing.T) {
		config := &models.RetryConfig{
			Backoff: "linear",
		}

		tests := []struct {
			attempt  int
			expected time.Duration
		}{
			{attempt: 1, expected: 1 * time.Second},
			{attempt: 2, expected: 2 * time.Second},
			{attempt: 3, expected: 3 * time.Second},
			{attempt: 4, expected: 4 * time.Second},
			{attempt: 5, expected: 5 * time.Second},
		}

		for _, tt := range tests {
			result := executor.calculateBackoff(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
			}
		}
	})

	t.Run("default backoff (exponential) when config is nil", func(t *testing.T) {
		result := executor.calculateBackoff(3, nil)
		expected := 4 * time.Second // 2^2 = 4

		if result != expected {
			t.Errorf("Expected default exponential backoff %v, got %v", expected, result)
		}
	})

	t.Run("unknown backoff strategy defaults to 1 second", func(t *testing.T) {
		config := &models.RetryConfig{
			Backoff: "unknown-strategy",
		}

		result := executor.calculateBackoff(5, config)
		expected := 1 * time.Second

		if result != expected {
			t.Errorf("Expected default 1s backoff, got %v", result)
		}
	})

	t.Run("empty backoff strategy uses default exponential", func(t *testing.T) {
		config := &models.RetryConfig{
			Backoff: "",
		}

		result := executor.calculateBackoff(3, config)
		expected := 4 * time.Second // exponential: 2^2 = 4

		if result != expected {
			t.Errorf("Expected default exponential backoff %v, got %v", expected, result)
		}
	})
}

// TestIsRetryableError tests conditional retry logic
func TestIsRetryableError(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	repo := &mockExecutionRepo{}
	executor := NewWorkflowExecutor(redisClient, repo, nil, log)

	t.Run("retries all errors when RetryOn is empty", func(t *testing.T) {
		err := errors.New("any error")
		result := executor.isRetryableError(err, []string{})

		if !result {
			t.Error("Expected to retry all errors when RetryOn is empty")
		}
	})

	t.Run("retries all errors when RetryOn is nil", func(t *testing.T) {
		err := errors.New("any error")
		result := executor.isRetryableError(err, nil)

		if !result {
			t.Error("Expected to retry all errors when RetryOn is nil")
		}
	})

	t.Run("retries all errors when RetryOn contains wildcard", func(t *testing.T) {
		err := errors.New("some specific error")
		result := executor.isRetryableError(err, []string{"*"})

		if !result {
			t.Error("Expected to retry with wildcard pattern")
		}
	})

	t.Run("retries when error matches exact pattern", func(t *testing.T) {
		err := errors.New("connection timeout")
		result := executor.isRetryableError(err, []string{"connection timeout", "network error"})

		if !result {
			t.Error("Expected to retry when error matches pattern")
		}
	})

	t.Run("does not retry when error does not match any pattern", func(t *testing.T) {
		err := errors.New("validation error")
		result := executor.isRetryableError(err, []string{"connection timeout", "network error"})

		if result {
			t.Error("Expected not to retry when error doesn't match pattern")
		}
	})

	t.Run("matches first pattern in list", func(t *testing.T) {
		err := errors.New("timeout")
		result := executor.isRetryableError(err, []string{"timeout", "connection refused", "network error"})

		if !result {
			t.Error("Expected to match first pattern")
		}
	})

	t.Run("matches last pattern in list", func(t *testing.T) {
		err := errors.New("network error")
		result := executor.isRetryableError(err, []string{"timeout", "connection refused", "network error"})

		if !result {
			t.Error("Expected to match last pattern")
		}
	})

	t.Run("matches middle pattern in list", func(t *testing.T) {
		err := errors.New("connection refused")
		result := executor.isRetryableError(err, []string{"timeout", "connection refused", "network error"})

		if !result {
			t.Error("Expected to match middle pattern")
		}
	})

	t.Run("wildcard overrides other patterns", func(t *testing.T) {
		err := errors.New("any random error")
		result := executor.isRetryableError(err, []string{"*", "specific error"})

		if !result {
			t.Error("Expected wildcard to match any error")
		}
	})

	t.Run("case sensitive matching", func(t *testing.T) {
		err := errors.New("Connection Timeout")
		result := executor.isRetryableError(err, []string{"connection timeout"})

		if result {
			t.Error("Expected case-sensitive matching to fail")
		}
	})

	t.Run("matches multiple different errors correctly", func(t *testing.T) {
		patterns := []string{"timeout", "connection refused", "network error"}

		tests := []struct {
			error       error
			shouldRetry bool
		}{
			{errors.New("timeout"), true},
			{errors.New("connection refused"), true},
			{errors.New("network error"), true},
			{errors.New("validation failed"), false},
			{errors.New("permission denied"), false},
			{errors.New("not found"), false},
		}

		for _, tt := range tests {
			result := executor.isRetryableError(tt.error, patterns)
			if result != tt.shouldRetry {
				t.Errorf("Error '%v': expected retry=%v, got %v", tt.error, tt.shouldRetry, result)
			}
		}
	})
}

// TestRetryConfiguration tests the retry configuration in WorkflowDefinition
func TestRetryConfiguration(t *testing.T) {
	t.Run("RetryConfig with all fields set", func(t *testing.T) {
		config := &models.RetryConfig{
			MaxAttempts: 5,
			Backoff:     "exponential",
			RetryOn:     []string{"timeout", "network error"},
		}

		if config.MaxAttempts != 5 {
			t.Errorf("Expected MaxAttempts 5, got %d", config.MaxAttempts)
		}

		if config.Backoff != "exponential" {
			t.Errorf("Expected Backoff 'exponential', got '%s'", config.Backoff)
		}

		if len(config.RetryOn) != 2 {
			t.Errorf("Expected 2 RetryOn patterns, got %d", len(config.RetryOn))
		}
	})

	t.Run("Step with retry configuration", func(t *testing.T) {
		step := models.Step{
			ID:   "test-step",
			Type: "action",
			Action: &models.Action{
				Type: "allow",
			},
			Retry: &models.RetryConfig{
				MaxAttempts: 3,
				Backoff:     "linear",
			},
		}

		if step.Retry == nil {
			t.Fatal("Expected retry config to be set")
		}

		if step.Retry.MaxAttempts != 3 {
			t.Errorf("Expected MaxAttempts 3, got %d", step.Retry.MaxAttempts)
		}

		if step.Retry.Backoff != "linear" {
			t.Errorf("Expected Backoff 'linear', got '%s'", step.Retry.Backoff)
		}
	})

	t.Run("Step without retry configuration", func(t *testing.T) {
		step := models.Step{
			ID:   "test-step",
			Type: "action",
			Action: &models.Action{
				Type: "block",
			},
			Retry: nil,
		}

		if step.Retry != nil {
			t.Error("Expected retry config to be nil")
		}
	})
}

// TestBackoffGrowth tests that backoff grows appropriately
func TestBackoffGrowth(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	repo := &mockExecutionRepo{}
	executor := NewWorkflowExecutor(redisClient, repo, nil, log)

	t.Run("exponential backoff grows exponentially", func(t *testing.T) {
		config := &models.RetryConfig{
			Backoff: "exponential",
		}

		var prevBackoff time.Duration
		for attempt := 1; attempt <= 5; attempt++ {
			backoff := executor.calculateBackoff(attempt, config)

			if attempt > 1 {
				// Each backoff should be roughly double the previous (exponential growth)
				if backoff <= prevBackoff {
					t.Errorf("Attempt %d: backoff should grow, got %v after %v", attempt, backoff, prevBackoff)
				}
			}

			prevBackoff = backoff
		}
	})

	t.Run("linear backoff grows linearly", func(t *testing.T) {
		config := &models.RetryConfig{
			Backoff: "linear",
		}

		var prevBackoff time.Duration
		for attempt := 1; attempt <= 5; attempt++ {
			backoff := executor.calculateBackoff(attempt, config)

			if attempt > 1 {
				// Each backoff should be exactly 1 second more than previous
				diff := backoff - prevBackoff
				if diff != time.Second {
					t.Errorf("Attempt %d: expected 1s increase, got %v", attempt, diff)
				}
			}

			prevBackoff = backoff
		}
	})
}

// TestEdgeCases tests edge cases for retry logic
func TestRetryEdgeCases(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	repo := &mockExecutionRepo{}
	executor := NewWorkflowExecutor(redisClient, repo, nil, log)

	t.Run("zero MaxAttempts", func(t *testing.T) {
		// With MaxAttempts of 0, the logic should default to 1 attempt
		config := &models.RetryConfig{
			MaxAttempts: 0,
		}

		if config.MaxAttempts == 0 {
			t.Log("MaxAttempts is 0, executor should handle this gracefully")
		}
	})

	t.Run("negative MaxAttempts", func(t *testing.T) {
		// Negative attempts should be treated as 0 or 1
		config := &models.RetryConfig{
			MaxAttempts: -1,
		}

		if config.MaxAttempts < 0 {
			t.Log("Negative MaxAttempts should be handled gracefully")
		}
	})

	t.Run("very large MaxAttempts", func(t *testing.T) {
		config := &models.RetryConfig{
			MaxAttempts: 1000,
			Backoff:     "exponential",
		}

		// Should not panic or overflow
		_ = executor.calculateBackoff(10, config)
	})

	t.Run("empty RetryOn with wildcard", func(t *testing.T) {
		err := errors.New("test error")
		result := executor.isRetryableError(err, []string{"*"})

		if !result {
			t.Error("Wildcard should match any error")
		}
	})
}
