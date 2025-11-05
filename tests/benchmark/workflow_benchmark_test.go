package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/engine"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// BenchmarkWorkflowExecutor_SimpleWorkflow benchmarks simple workflow execution
func BenchmarkWorkflowExecutor_SimpleWorkflow(b *testing.B) {
	executor := setupBenchmarkExecutor(b)

	workflow := &models.Workflow{
		ID:         uuid.New(),
		WorkflowID: "benchmark-simple",
		Version:    "1.0.0",
		Name:       "Simple Benchmark Workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "test.event",
			},
			Steps: []models.Step{
				{
					ID:   "step1",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}

	ctx := context.Background()
	event := map[string]interface{}{
		"test": "data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(ctx, workflow, event)
	}
}

// BenchmarkWorkflowExecutor_ConditionalWorkflow benchmarks conditional workflow execution
func BenchmarkWorkflowExecutor_ConditionalWorkflow(b *testing.B) {
	executor := setupBenchmarkExecutor(b)

	workflow := &models.Workflow{
		ID:         uuid.New(),
		WorkflowID: "benchmark-conditional",
		Version:    "1.0.0",
		Name:       "Conditional Benchmark Workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "order.created",
			},
			Steps: []models.Step{
				{
					ID:   "check_total",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "order.total",
						Operator: "gt",
						Value:    1000.0,
					},
					OnTrue:  "high_value",
					OnFalse: "low_value",
				},
				{
					ID:   "high_value",
					Type: "action",
					Action: &models.Action{
						Type: "block",
					},
				},
				{
					ID:   "low_value",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}

	ctx := context.Background()
	event := map[string]interface{}{
		"order": map[string]interface{}{
			"total": 1500.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(ctx, workflow, event)
	}
}

// BenchmarkWorkflowExecutor_ComplexWorkflow benchmarks complex workflow with multiple conditions
func BenchmarkWorkflowExecutor_ComplexWorkflow(b *testing.B) {
	executor := setupBenchmarkExecutor(b)

	workflow := &models.Workflow{
		ID:         uuid.New(),
		WorkflowID: "benchmark-complex",
		Version:    "1.0.0",
		Name:       "Complex Benchmark Workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "transaction.created",
			},
			Steps: []models.Step{
				{
					ID:   "check_amount",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "transaction.amount",
						Operator: "gte",
						Value:    5000.0,
					},
					OnTrue:  "check_country",
					OnFalse: "auto_approve",
				},
				{
					ID:   "check_country",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "transaction.country",
						Operator: "in",
						Value:    []interface{}{"US", "CA", "GB"},
					},
					OnTrue:  "check_customer",
					OnFalse: "block_country",
				},
				{
					ID:   "check_customer",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "customer.verified",
						Operator: "eq",
						Value:    true,
					},
					OnTrue:  "auto_approve",
					OnFalse: "manual_review",
				},
				{
					ID:   "auto_approve",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
				{
					ID:   "block_country",
					Type: "action",
					Action: &models.Action{
						Type: "block",
					},
				},
				{
					ID:   "manual_review",
					Type: "action",
					Action: &models.Action{
						Type: "block",
					},
				},
			},
		},
	}

	ctx := context.Background()
	event := map[string]interface{}{
		"transaction": map[string]interface{}{
			"amount":  7500.0,
			"country": "US",
		},
		"customer": map[string]interface{}{
			"verified": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(ctx, workflow, event)
	}
}

// BenchmarkEvaluator_SimpleCondition benchmarks simple condition evaluation
func BenchmarkEvaluator_SimpleCondition(b *testing.B) {
	evaluator := engine.NewEvaluator()

	condition := &models.Condition{
		Field:    "amount",
		Operator: "gt",
		Value:    100.0,
	}

	data := map[string]interface{}{
		"amount": 150.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, data)
	}
}

// BenchmarkEvaluator_ComplexCondition benchmarks complex condition evaluation
func BenchmarkEvaluator_ComplexCondition(b *testing.B) {
	evaluator := engine.NewEvaluator()

	condition := &models.Condition{
		LogicalOp: "AND",
		Conditions: []models.Condition{
			{
				Field:    "amount",
				Operator: "gte",
				Value:    1000.0,
			},
			{
				Field:    "country",
				Operator: "in",
				Value:    []interface{}{"US", "CA", "GB"},
			},
			{
				Field:    "verified",
				Operator: "eq",
				Value:    true,
			},
		},
	}

	data := map[string]interface{}{
		"amount":   1500.0,
		"country":  "US",
		"verified": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, data)
	}
}

// BenchmarkEvaluator_NestedFieldExtraction benchmarks nested field extraction
func BenchmarkEvaluator_NestedFieldExtraction(b *testing.B) {
	evaluator := engine.NewEvaluator()

	condition := &models.Condition{
		Field:    "order.customer.address.country",
		Operator: "eq",
		Value:    "US",
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"customer": map[string]interface{}{
				"address": map[string]interface{}{
					"country": "US",
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, data)
	}
}

// BenchmarkEvaluator_RegexMatch benchmarks regex condition evaluation
func BenchmarkEvaluator_RegexMatch(b *testing.B) {
	evaluator := engine.NewEvaluator()

	condition := &models.Condition{
		Field:    "email",
		Operator: "regex",
		Value:    "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
	}

	data := map[string]interface{}{
		"email": "user@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.Evaluate(condition, data)
	}
}

// BenchmarkContextBuilder_SimpleContext benchmarks simple context building
func BenchmarkContextBuilder_SimpleContext(b *testing.B) {
	builder := setupBenchmarkContextBuilder(b)

	contextDef := models.ContextDefinition{
		Fields: []models.ContextField{
			{
				Name:   "order_id",
				Source: "event",
				Path:   "order.id",
			},
			{
				Name:   "customer_id",
				Source: "event",
				Path:   "customer.id",
			},
		},
	}

	event := map[string]interface{}{
		"order": map[string]interface{}{
			"id": "order-123",
		},
		"customer": map[string]interface{}{
			"id": "customer-456",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = builder.BuildContext(ctx, &contextDef, event)
	}
}

// BenchmarkConcurrentWorkflowExecution benchmarks concurrent workflow execution
func BenchmarkConcurrentWorkflowExecution(b *testing.B) {
	executor := setupBenchmarkExecutor(b)

	workflow := &models.Workflow{
		ID:         uuid.New(),
		WorkflowID: "benchmark-concurrent",
		Version:    "1.0.0",
		Name:       "Concurrent Benchmark Workflow",
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type:  "event",
				Event: "test.event",
			},
			Steps: []models.Step{
				{
					ID:   "step1",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}

	event := map[string]interface{}{
		"test": "data",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			_, _ = executor.Execute(ctx, workflow, event)
		}
	})
}

// Helper functions

func setupBenchmarkExecutor(b *testing.B) *engine.Executor {
	b.Helper()

	// Create a minimal executor for benchmarking
	// In real implementation, this would set up all necessary dependencies
	return &engine.Executor{
		Evaluator: engine.NewEvaluator(),
		Timeout:   30 * time.Second,
	}
}

func setupBenchmarkContextBuilder(b *testing.B) *engine.ContextBuilder {
	b.Helper()

	// Create a minimal context builder for benchmarking
	return &engine.ContextBuilder{
		Timeout: 5 * time.Second,
	}
}
