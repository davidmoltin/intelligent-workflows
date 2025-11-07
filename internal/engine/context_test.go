package engine

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// getTestContextEnrichmentConfig returns a test configuration with enrichment disabled
func getTestContextEnrichmentConfig() *config.ContextEnrichmentConfig {
	return &config.ContextEnrichmentConfig{
		Enabled:    false, // Disabled for tests by default
		BaseURL:    "http://localhost:8081",
		Timeout:    10 * time.Second,
		MaxRetries: 3,
		RetryDelay: 500 * time.Millisecond,
		CacheTTL:   5 * time.Minute,
		EndpointMapping: map[string]string{
			"order.details":     "/api/v1/orders/{id}/details",
			"customer.history":  "/api/v1/customers/{id}/history",
			"product.inventory": "/api/v1/products/{id}/inventory",
		},
	}
}

func TestBuildContext(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	cfg := getTestContextEnrichmentConfig()
	builder := NewContextBuilder(redisClient, log, cfg)
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("builds context from trigger payload", func(t *testing.T) {
		triggerPayload := map[string]interface{}{
			"order_id":    "ord-123",
			"total":       1500.0,
			"customer_id": "cust-456",
		}

		contextDef := models.ContextDefinition{
			Load: []string{},
		}

		execContext, err := builder.BuildContext(ctx, orgID, triggerPayload, contextDef)

		if err != nil {
			t.Fatalf("BuildContext failed: %v", err)
		}

		if execContext["order_id"] != "ord-123" {
			t.Errorf("Expected order_id ord-123, got %v", execContext["order_id"])
		}

		if execContext["total"] != 1500.0 {
			t.Errorf("Expected total 1500.0, got %v", execContext["total"])
		}
	})

	t.Run("builds context with load resources", func(t *testing.T) {
		triggerPayload := map[string]interface{}{
			"order_id": "ord-123",
		}

		contextDef := models.ContextDefinition{
			Load: []string{"order.details", "customer.history"},
		}

		execContext, err := builder.BuildContext(ctx, orgID, triggerPayload, contextDef)

		if err != nil {
			t.Fatalf("BuildContext failed: %v", err)
		}

		if execContext["order_id"] != "ord-123" {
			t.Error("Trigger payload should be in context")
		}

		// Resources would be loaded but in our mock they return empty data
		// Just verify no error occurred
	})
}

func TestBuildContextFromExisting(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	cfg := getTestContextEnrichmentConfig()
	builder := NewContextBuilder(redisClient, log, cfg)
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("reloads context resources", func(t *testing.T) {
		existingContext := map[string]interface{}{
			"order_id": "ord-123",
			"total":    1500.0,
		}

		contextDef := models.ContextDefinition{
			Load: []string{"order.details"},
		}

		err := builder.BuildContextFromExisting(ctx, orgID, existingContext, contextDef)

		if err != nil {
			t.Fatalf("BuildContextFromExisting failed: %v", err)
		}

		// Verify existing context is preserved
		if existingContext["order_id"] != "ord-123" {
			t.Error("Existing context should be preserved")
		}
	})

	t.Run("handles empty load list", func(t *testing.T) {
		existingContext := map[string]interface{}{
			"order_id": "ord-123",
		}

		contextDef := models.ContextDefinition{
			Load: []string{},
		}

		err := builder.BuildContextFromExisting(ctx, orgID, existingContext, contextDef)

		if err != nil {
			t.Fatalf("BuildContextFromExisting failed: %v", err)
		}
	})
}

func TestEnrichContext(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	cfg := getTestContextEnrichmentConfig()
	builder := NewContextBuilder(redisClient, log, cfg)
	ctx := context.Background()

	t.Run("adds computed fields", func(t *testing.T) {
		execContext := map[string]interface{}{
			"order": map[string]interface{}{
				"id":    "ord-123",
				"total": 1500.0,
			},
		}

		err := builder.EnrichContext(ctx, execContext)

		if err != nil {
			t.Fatalf("EnrichContext failed: %v", err)
		}

		// Check metadata
		meta, ok := execContext["_meta"].(map[string]interface{})
		if !ok {
			t.Fatal("_meta should be a map")
		}

		if meta["version"] != "1.0" {
			t.Error("Expected version 1.0 in metadata")
		}

		// Check computed fields
		computed, ok := execContext["_computed"].(map[string]interface{})
		if !ok {
			t.Fatal("_computed should be a map")
		}

		if _, ok := computed["current_time"]; !ok {
			t.Error("current_time should be computed")
		}

		if _, ok := computed["current_hour"]; !ok {
			t.Error("current_hour should be computed")
		}

		if _, ok := computed["order_is_medium_value"]; !ok {
			t.Error("order_is_medium_value should be computed")
		}
	})

	t.Run("handles empty context", func(t *testing.T) {
		execContext := map[string]interface{}{}

		err := builder.EnrichContext(ctx, execContext)

		if err != nil {
			t.Fatalf("EnrichContext failed: %v", err)
		}

		if _, ok := execContext["_meta"]; !ok {
			t.Error("_meta should be added to empty context")
		}
	})
}

func TestBuildCacheKey(t *testing.T) {
	log := logger.NewForTesting()
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	cfg := getTestContextEnrichmentConfig()
	builder := NewContextBuilder(redisClient, log, cfg)
	orgID := uuid.MustParse("12345678-1234-1234-1234-123456789012")

	t.Run("builds key for order.details", func(t *testing.T) {
		context := map[string]interface{}{
			"order": map[string]interface{}{
				"id": "ord-123",
			},
		}

		key := builder.buildCacheKey(orgID, "order.details", context)

		expectedKey := fmt.Sprintf("context:%s:order.details:ord-123", orgID.String())
		if key != expectedKey {
			t.Errorf("Expected %s, got %s", expectedKey, key)
		}
	})

	t.Run("builds key for customer.history", func(t *testing.T) {
		context := map[string]interface{}{
			"customer": map[string]interface{}{
				"id": "cust-456",
			},
		}

		key := builder.buildCacheKey(orgID, "customer.history", context)

		expectedKey := fmt.Sprintf("context:%s:customer.history:cust-456", orgID.String())
		if key != expectedKey {
			t.Errorf("Expected %s, got %s", expectedKey, key)
		}
	})

	t.Run("builds fallback key for unknown resource", func(t *testing.T) {
		context := map[string]interface{}{}

		key := builder.buildCacheKey(orgID, "unknown.resource", context)

		if len(key) == 0 {
			t.Error("Key should not be empty")
		}

		// Should contain the organization ID and resource name
		expectedPrefix := fmt.Sprintf("context:%s:unknown.resource:", orgID.String())
		if !strings.HasPrefix(key, expectedPrefix) {
			t.Errorf("Key should start with %s, got %s", expectedPrefix, key)
		}
	})
}
