package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
)

// ContextBuilder handles building and enriching execution context
type ContextBuilder struct {
	redis  *redis.Client
	logger *logger.Logger
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(redisClient *redis.Client, log *logger.Logger) *ContextBuilder {
	return &ContextBuilder{
		redis:  redisClient,
		logger: log,
	}
}

// BuildContext builds the execution context from trigger payload and context definition
func (cb *ContextBuilder) BuildContext(
	ctx context.Context,
	triggerPayload map[string]interface{},
	contextDef models.ContextDefinition,
) (map[string]interface{}, error) {
	// Start with the trigger payload
	execContext := make(map[string]interface{})

	// Merge trigger payload into context
	for key, value := range triggerPayload {
		execContext[key] = value
	}

	// Load additional context data as specified
	if len(contextDef.Load) > 0 {
		for _, resource := range contextDef.Load {
			cb.logger.Infof("Loading context resource: %s", resource)

			data, err := cb.loadResource(ctx, resource, execContext)
			if err != nil {
				cb.logger.Errorf("Failed to load resource %s: %v", resource, err)
				// Continue loading other resources even if one fails
				continue
			}

			// Merge loaded data into context
			cb.mergeIntoContext(execContext, resource, data)
		}
	}

	return execContext, nil
}

// BuildContextFromExisting reloads context resources while preserving existing context
func (cb *ContextBuilder) BuildContextFromExisting(
	ctx context.Context,
	existingContext map[string]interface{},
	contextDef models.ContextDefinition,
) error {
	// Load additional context data as specified, refreshing stale data
	if len(contextDef.Load) > 0 {
		for _, resource := range contextDef.Load {
			cb.logger.Infof("Reloading context resource: %s", resource)

			data, err := cb.loadResource(ctx, resource, existingContext)
			if err != nil {
				cb.logger.Errorf("Failed to reload resource %s: %v", resource, err)
				// Continue loading other resources even if one fails
				continue
			}

			// Merge reloaded data into context
			cb.mergeIntoContext(existingContext, resource, data)
		}
	}

	return nil
}

// loadResource loads a specific resource (e.g., order.details, customer.history)
func (cb *ContextBuilder) loadResource(
	ctx context.Context,
	resource string,
	currentContext map[string]interface{},
) (map[string]interface{}, error) {
	// Try to load from cache first
	cached, err := cb.getFromCache(ctx, resource, currentContext)
	if err == nil && cached != nil {
		cb.logger.Debugf("Context cache hit for resource: %s", resource)
		return cached, nil
	}

	// If not in cache, this is a placeholder for external API calls
	// In a real implementation, this would call microservices to fetch data
	cb.logger.Debugf("Context cache miss for resource: %s", resource)

	// For now, return empty data
	// TODO: Implement actual resource loading from microservices
	return map[string]interface{}{}, nil
}

// getFromCache retrieves cached context data from Redis
func (cb *ContextBuilder) getFromCache(
	ctx context.Context,
	resource string,
	currentContext map[string]interface{},
) (map[string]interface{}, error) {
	// Build cache key from resource and context
	cacheKey := cb.buildCacheKey(resource, currentContext)

	data, err := cb.redis.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache miss")
	}
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	var cached map[string]interface{}
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	return cached, nil
}

// setInCache stores context data in Redis cache
func (cb *ContextBuilder) setInCache(
	ctx context.Context,
	resource string,
	currentContext map[string]interface{},
	data map[string]interface{},
	ttl time.Duration,
) error {
	cacheKey := cb.buildCacheKey(resource, currentContext)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return cb.redis.Set(ctx, cacheKey, jsonData, ttl).Err()
}

// buildCacheKey creates a cache key for a resource
func (cb *ContextBuilder) buildCacheKey(resource string, context map[string]interface{}) string {
	// Extract relevant identifiers from context to build a unique key
	// Example: "context:order.details:order_id:ord_123"

	var identifier string

	// Try to extract entity ID from context
	switch resource {
	case "order.details":
		if order, ok := context["order"].(map[string]interface{}); ok {
			if id, ok := order["id"].(string); ok {
				identifier = id
			}
		}
	case "customer.history":
		if customer, ok := context["customer"].(map[string]interface{}); ok {
			if id, ok := customer["id"].(string); ok {
				identifier = id
			}
		}
	case "product.inventory":
		if product, ok := context["product"].(map[string]interface{}); ok {
			if id, ok := product["id"].(string); ok {
				identifier = id
			}
		}
	}

	if identifier == "" {
		// Fallback to a generic identifier
		identifier = uuid.New().String()
	}

	return fmt.Sprintf("context:%s:%s", resource, identifier)
}

// mergeIntoContext merges loaded data into the context at the appropriate path
func (cb *ContextBuilder) mergeIntoContext(
	context map[string]interface{},
	resource string,
	data map[string]interface{},
) {
	// Parse resource path (e.g., "order.details" -> ["order", "details"])
	// and create nested structure

	if len(data) == 0 {
		return
	}

	// For now, just merge at the top level with the full resource name as key
	// In a more sophisticated implementation, this would create nested structures
	context[resource] = data
}

// EnrichContext adds computed fields and metadata to the context
func (cb *ContextBuilder) EnrichContext(
	ctx context.Context,
	execContext map[string]interface{},
) error {
	// Add execution metadata
	if execContext["_meta"] == nil {
		execContext["_meta"] = make(map[string]interface{})
	}

	meta := execContext["_meta"].(map[string]interface{})
	meta["enriched_at"] = time.Now().Unix()
	meta["version"] = "1.0"

	// Add computed fields (examples)
	cb.addComputedFields(execContext)

	return nil
}

// addComputedFields adds useful computed fields to the context
func (cb *ContextBuilder) addComputedFields(context map[string]interface{}) {
	// Example: Add day of week, hour, etc. for time-based rules
	now := time.Now()

	if context["_computed"] == nil {
		context["_computed"] = make(map[string]interface{})
	}

	computed := context["_computed"].(map[string]interface{})
	computed["current_time"] = now.Unix()
	computed["current_hour"] = now.Hour()
	computed["current_day_of_week"] = now.Weekday().String()
	computed["current_date"] = now.Format("2006-01-02")

	// Add order-specific computed fields if order exists
	if order, ok := context["order"].(map[string]interface{}); ok {
		if total, ok := order["total"].(float64); ok {
			computed["order_is_high_value"] = total >= 10000
			computed["order_is_medium_value"] = total >= 1000 && total < 10000
			computed["order_is_low_value"] = total < 1000
		}
	}

	// Add customer-specific computed fields if customer exists
	if customer, ok := context["customer"].(map[string]interface{}); ok {
		if createdAt, ok := customer["created_at"].(string); ok {
			// Parse created_at and compute account age
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				accountAgeDays := int(time.Since(t).Hours() / 24)
				computed["customer_account_age_days"] = accountAgeDays
				computed["customer_is_new"] = accountAgeDays < 30
			}
		}
	}
}

// ClearCache clears cache entries for specific resources
func (cb *ContextBuilder) ClearCache(ctx context.Context, pattern string) error {
	// Use SCAN to find and delete keys matching pattern
	iter := cb.redis.Scan(ctx, 0, fmt.Sprintf("context:%s:*", pattern), 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if len(keys) > 0 {
		if err := cb.redis.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("delete error: %w", err)
		}
		cb.logger.Infof("Cleared %d cache entries for pattern: %s", len(keys), pattern)
	}

	return nil
}
