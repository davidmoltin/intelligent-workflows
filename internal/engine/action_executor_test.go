package engine

import (
	"context"
	"testing"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

func TestExecuteAction(t *testing.T) {
	log := logger.NewForTesting()
	executor := NewActionExecutor(log)
	ctx := context.Background()

	t.Run("executes allow action", func(t *testing.T) {
		step := &models.Step{
			ID:   "action1",
			Type: "action",
			Action: &models.Action{
				Type:   "allow",
				Reason: "Order looks safe",
			},
		}

		result, err := executor.ExecuteAction(ctx, step, map[string]interface{}{})

		if err != nil {
			t.Fatalf("ExecuteAction failed: %v", err)
		}

		if result.Action != "allow" {
			t.Errorf("Expected action allow, got %s", result.Action)
		}

		if !result.Success {
			t.Error("Expected success true")
		}
	})

	t.Run("executes block action", func(t *testing.T) {
		step := &models.Step{
			ID:   "action1",
			Type: "action",
			Action: &models.Action{
				Type:   "block",
				Reason: "High risk order",
			},
		}

		result, err := executor.ExecuteAction(ctx, step, map[string]interface{}{})

		if err != nil {
			t.Fatalf("ExecuteAction failed: %v", err)
		}

		if result.Action != "block" {
			t.Errorf("Expected action block, got %s", result.Action)
		}

		if !result.Success {
			t.Error("Expected success true")
		}
	})

	t.Run("executes execute actions", func(t *testing.T) {
		step := &models.Step{
			ID:   "action1",
			Type: "action",
			Action: &models.Action{
				Type: "execute",
			},
			Execute: []models.ExecuteAction{
				{
					Type:       "notify",
					Recipients: []string{"admin@example.com"},
					Message:    "Test notification",
				},
			},
		}

		result, err := executor.ExecuteAction(ctx, step, map[string]interface{}{})

		if err != nil {
			t.Fatalf("ExecuteAction failed: %v", err)
		}

		if result.Action != "execute" {
			t.Errorf("Expected action execute, got %s", result.Action)
		}

		if !result.Success {
			t.Error("Expected success true")
		}
	})

	t.Run("handles missing action", func(t *testing.T) {
		step := &models.Step{
			ID:     "action1",
			Type:   "action",
			Action: nil,
		}

		_, err := executor.ExecuteAction(ctx, step, map[string]interface{}{})

		if err == nil {
			t.Error("Expected error for missing action")
		}
	})
}
