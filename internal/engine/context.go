package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/config"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ContextBuilder handles building and enriching execution context
type ContextBuilder struct {
	redis      *redis.Client
	logger     *logger.Logger
	config     *config.ContextEnrichmentConfig
	httpClient *http.Client
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(redisClient *redis.Client, log *logger.Logger, cfg *config.ContextEnrichmentConfig) *ContextBuilder {
	return &ContextBuilder{
		redis:  redisClient,
		logger: log,
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
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

	cb.logger.Debugf("Context cache miss for resource: %s", resource)

	// Check if context enrichment is enabled
	if !cb.config.Enabled {
		cb.logger.Debugf("Context enrichment is disabled, returning empty data for resource: %s", resource)
		return map[string]interface{}{}, nil
	}

	// Load from microservice
	data, err := cb.fetchFromMicroservice(ctx, resource, currentContext)
	if err != nil {
		cb.logger.Errorf("Failed to fetch resource %s from microservice: %v", resource, err)
		// Return empty data on error to allow workflow to continue
		return map[string]interface{}{}, nil
	}

	// Cache the successful response
	if len(data) > 0 {
		if err := cb.setInCache(ctx, resource, currentContext, data, cb.config.CacheTTL); err != nil {
			cb.logger.Warnf("Failed to cache resource %s: %v", resource, err)
			// Continue even if caching fails
		}
	}

	return data, nil
}

// fetchFromMicroservice fetches resource data from external microservice
func (cb *ContextBuilder) fetchFromMicroservice(
	ctx context.Context,
	resource string,
	currentContext map[string]interface{},
) (map[string]interface{}, error) {
	// Get endpoint mapping for resource
	endpointTemplate, exists := cb.config.EndpointMapping[resource]
	if !exists {
		return nil, fmt.Errorf("no endpoint mapping found for resource: %s", resource)
	}

	// Extract identifier from context for the resource
	identifier, err := cb.extractIdentifier(resource, currentContext)
	if err != nil {
		return nil, fmt.Errorf("failed to extract identifier for resource %s: %w", resource, err)
	}

	// Build endpoint URL by replacing {id} with actual identifier
	endpoint := strings.ReplaceAll(endpointTemplate, "{id}", identifier)
	url := cb.config.BaseURL + endpoint

	// Attempt to fetch with retry logic
	var data map[string]interface{}
	var lastErr error

	for attempt := 0; attempt <= cb.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay with exponential backoff
			backoffDelay := cb.config.RetryDelay * time.Duration(1<<uint(attempt-1))
			cb.logger.Infof("Retrying fetch for resource %s (attempt %d/%d) after %v", resource, attempt, cb.config.MaxRetries, backoffDelay)

			select {
			case <-time.After(backoffDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		data, lastErr = cb.makeHTTPRequest(ctx, url, resource)
		if lastErr == nil {
			cb.logger.Infof("Successfully fetched resource %s from microservice: %s", resource, url)
			return data, nil
		}

		cb.logger.Warnf("Attempt %d/%d failed to fetch resource %s: %v", attempt+1, cb.config.MaxRetries+1, resource, lastErr)
	}

	return nil, fmt.Errorf("failed to fetch resource after %d attempts: %w", cb.config.MaxRetries+1, lastErr)
}

// makeHTTPRequest makes a single HTTP request to fetch resource data
func (cb *ContextBuilder) makeHTTPRequest(
	ctx context.Context,
	url string,
	resource string,
) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "IntelligentWorkflows/1.0")
	req.Header.Set("X-Request-ID", uuid.New().String())
	req.Header.Set("X-Resource-Type", resource)

	// Execute request
	resp, err := cb.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microservice returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read and parse response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return data, nil
}

// extractIdentifier extracts the resource identifier from context
func (cb *ContextBuilder) extractIdentifier(resource string, context map[string]interface{}) (string, error) {
	// Parse resource name to determine entity type (e.g., "order.details" -> "order")
	parts := strings.Split(resource, ".")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid resource format: %s", resource)
	}

	entityType := parts[0]

	// Try to extract ID from context based on entity type
	switch entityType {
	case "order":
		if order, ok := context["order"].(map[string]interface{}); ok {
			if id, ok := order["id"].(string); ok && id != "" {
				return id, nil
			}
		}
		// Try alternative field names
		if orderID, ok := context["order_id"].(string); ok && orderID != "" {
			return orderID, nil
		}

	case "customer":
		if customer, ok := context["customer"].(map[string]interface{}); ok {
			if id, ok := customer["id"].(string); ok && id != "" {
				return id, nil
			}
		}
		if customerID, ok := context["customer_id"].(string); ok && customerID != "" {
			return customerID, nil
		}

	case "product":
		if product, ok := context["product"].(map[string]interface{}); ok {
			if id, ok := product["id"].(string); ok && id != "" {
				return id, nil
			}
		}
		if productID, ok := context["product_id"].(string); ok && productID != "" {
			return productID, nil
		}

	case "payment":
		if payment, ok := context["payment"].(map[string]interface{}); ok {
			if id, ok := payment["id"].(string); ok && id != "" {
				return id, nil
			}
		}
		if paymentID, ok := context["payment_id"].(string); ok && paymentID != "" {
			return paymentID, nil
		}

	case "shipment":
		if shipment, ok := context["shipment"].(map[string]interface{}); ok {
			if id, ok := shipment["id"].(string); ok && id != "" {
				return id, nil
			}
		}
		if shipmentID, ok := context["shipment_id"].(string); ok && shipmentID != "" {
			return shipmentID, nil
		}

	case "user":
		if user, ok := context["user"].(map[string]interface{}); ok {
			if id, ok := user["id"].(string); ok && id != "" {
				return id, nil
			}
		}
		if userID, ok := context["user_id"].(string); ok && userID != "" {
			return userID, nil
		}

	case "subscription":
		if subscription, ok := context["subscription"].(map[string]interface{}); ok {
			if id, ok := subscription["id"].(string); ok && id != "" {
				return id, nil
			}
		}
		if subscriptionID, ok := context["subscription_id"].(string); ok && subscriptionID != "" {
			return subscriptionID, nil
		}

	case "invoice":
		if invoice, ok := context["invoice"].(map[string]interface{}); ok {
			if id, ok := invoice["id"].(string); ok && id != "" {
				return id, nil
			}
		}
		if invoiceID, ok := context["invoice_id"].(string); ok && invoiceID != "" {
			return invoiceID, nil
		}

	default:
		return "", fmt.Errorf("unknown entity type: %s", entityType)
	}

	return "", fmt.Errorf("identifier not found in context for entity type: %s", entityType)
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
