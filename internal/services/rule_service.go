package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/engine"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/database"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	ruleCacheKeyPrefix = "rule:"
	ruleCacheTTL       = 5 * time.Minute
)

// RuleService handles rule business logic
type RuleService struct {
	ruleRepo  *postgres.RuleRepository
	evaluator *engine.Evaluator
	redis     *database.RedisClient
	logger    *logger.Logger
}

// NewRuleService creates a new rule service
func NewRuleService(
	ruleRepo *postgres.RuleRepository,
	evaluator *engine.Evaluator,
	redis *database.RedisClient,
	log *logger.Logger,
) *RuleService {
	return &RuleService{
		ruleRepo:  ruleRepo,
		evaluator: evaluator,
		redis:     redis,
		logger:    log,
	}
}

// Create creates a new rule
func (s *RuleService) Create(ctx context.Context, organizationID uuid.UUID, req *models.CreateRuleRequest) (*models.Rule, error) {
	// Validate rule definition
	if err := s.validateRuleDefinition(req.RuleType, &req.Definition); err != nil {
		return nil, fmt.Errorf("invalid rule definition: %w", err)
	}

	// Create rule
	rule, err := s.ruleRepo.Create(ctx, organizationID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	// Cache the rule
	if err := s.cacheRule(ctx, rule); err != nil {
		s.logger.Warn("Failed to cache rule", zap.Error(err), zap.String("rule_id", rule.RuleID))
	}

	s.logger.Info("Rule created", zap.String("rule_id", rule.RuleID), zap.String("type", string(rule.RuleType)))

	return rule, nil
}

// GetByID retrieves a rule by UUID
func (s *RuleService) GetByID(ctx context.Context, organizationID, id uuid.UUID) (*models.Rule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}

	return rule, nil
}

// GetByRuleID retrieves a rule by its rule_id string, with caching
func (s *RuleService) GetByRuleID(ctx context.Context, organizationID uuid.UUID, ruleID string) (*models.Rule, error) {
	// Try cache first
	rule, err := s.getCachedRule(ctx, ruleID)
	if err == nil && rule != nil {
		// Verify organization ownership
		if rule.OrganizationID != organizationID {
			return nil, fmt.Errorf("rule not found")
		}
		return rule, nil
	}

	// Cache miss - fetch from database
	rule, err = s.ruleRepo.GetByRuleID(ctx, organizationID, ruleID)
	if err != nil {
		return nil, err
	}

	// Cache the rule
	if err := s.cacheRule(ctx, rule); err != nil {
		s.logger.Warn("Failed to cache rule", zap.Error(err), zap.String("rule_id", ruleID))
	}

	return rule, nil
}

// List retrieves rules with optional filtering and pagination
func (s *RuleService) List(ctx context.Context, organizationID uuid.UUID, enabled *bool, ruleType *models.RuleType, limit, offset int) ([]*models.Rule, int64, error) {
	// Apply default limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rules, total, err := s.ruleRepo.List(ctx, organizationID, enabled, ruleType, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// Update updates a rule
func (s *RuleService) Update(ctx context.Context, organizationID, id uuid.UUID, req *models.UpdateRuleRequest) (*models.Rule, error) {
	// Get existing rule to validate type
	existingRule, err := s.ruleRepo.GetByID(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}

	// Validate rule definition if provided
	if req.Definition != nil {
		if err := s.validateRuleDefinition(existingRule.RuleType, req.Definition); err != nil {
			return nil, fmt.Errorf("invalid rule definition: %w", err)
		}
	}

	// Update rule
	rule, err := s.ruleRepo.Update(ctx, organizationID, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update rule: %w", err)
	}

	// Invalidate cache
	if err := s.invalidateRuleCache(ctx, rule.RuleID); err != nil {
		s.logger.Warn("Failed to invalidate rule cache", zap.Error(err), zap.String("rule_id", rule.RuleID))
	}

	s.logger.Info("Rule updated", zap.String("rule_id", rule.RuleID))

	return rule, nil
}

// Delete deletes a rule
func (s *RuleService) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	// Get rule to invalidate cache
	rule, err := s.ruleRepo.GetByID(ctx, organizationID, id)
	if err != nil {
		return err
	}

	// Delete rule
	if err := s.ruleRepo.Delete(ctx, organizationID, id); err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	// Invalidate cache
	if err := s.invalidateRuleCache(ctx, rule.RuleID); err != nil {
		s.logger.Warn("Failed to invalidate rule cache", zap.Error(err), zap.String("rule_id", rule.RuleID))
	}

	s.logger.Info("Rule deleted", zap.String("rule_id", rule.RuleID))

	return nil
}

// Enable enables a rule
func (s *RuleService) Enable(ctx context.Context, organizationID, id uuid.UUID) error {
	// Get rule to invalidate cache
	rule, err := s.ruleRepo.GetByID(ctx, organizationID, id)
	if err != nil {
		return err
	}

	// Enable rule
	if err := s.ruleRepo.Enable(ctx, organizationID, id); err != nil {
		return fmt.Errorf("failed to enable rule: %w", err)
	}

	// Invalidate cache
	if err := s.invalidateRuleCache(ctx, rule.RuleID); err != nil {
		s.logger.Warn("Failed to invalidate rule cache", zap.Error(err), zap.String("rule_id", rule.RuleID))
	}

	s.logger.Info("Rule enabled", zap.String("rule_id", rule.RuleID))

	return nil
}

// Disable disables a rule
func (s *RuleService) Disable(ctx context.Context, organizationID, id uuid.UUID) error {
	// Get rule to invalidate cache
	rule, err := s.ruleRepo.GetByID(ctx, organizationID, id)
	if err != nil {
		return err
	}

	// Disable rule
	if err := s.ruleRepo.Disable(ctx, organizationID, id); err != nil {
		return fmt.Errorf("failed to disable rule: %w", err)
	}

	// Invalidate cache
	if err := s.invalidateRuleCache(ctx, rule.RuleID); err != nil {
		s.logger.Warn("Failed to invalidate rule cache", zap.Error(err), zap.String("rule_id", rule.RuleID))
	}

	s.logger.Info("Rule disabled", zap.String("rule_id", rule.RuleID))

	return nil
}

// TestRule tests a rule against provided context
func (s *RuleService) TestRule(ctx context.Context, organizationID, id uuid.UUID, req *models.TestRuleRequest) (*models.TestRuleResponse, error) {
	// Get rule
	rule, err := s.ruleRepo.GetByID(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}

	// Test based on rule type
	switch rule.RuleType {
	case models.RuleTypeCondition:
		return s.testConditionRule(rule, req.Context)
	case models.RuleTypeValidation:
		return s.testValidationRule(rule, req.Context)
	case models.RuleTypeEnrichment:
		return s.testEnrichmentRule(rule, req.Context)
	default:
		return nil, fmt.Errorf("unsupported rule type: %s", rule.RuleType)
	}
}

// testConditionRule tests a condition rule
func (s *RuleService) testConditionRule(rule *models.Rule, context map[string]interface{}) (*models.TestRuleResponse, error) {
	// Evaluate all conditions
	results := make([]bool, 0, len(rule.Definition.Conditions))
	details := make(map[string]interface{})

	for i, condition := range rule.Definition.Conditions {
		result, err := s.evaluator.EvaluateCondition(&condition, context)
		if err != nil {
			return nil, fmt.Errorf("condition %d evaluation failed: %w", i, err)
		}
		results = append(results, result)
		details[fmt.Sprintf("condition_%d", i)] = result
	}

	// All conditions must pass
	passed := true
	for _, result := range results {
		if !result {
			passed = false
			break
		}
	}

	return &models.TestRuleResponse{
		Passed:  passed,
		Result:  results,
		Details: details,
	}, nil
}

// testValidationRule tests a validation rule
func (s *RuleService) testValidationRule(rule *models.Rule, context map[string]interface{}) (*models.TestRuleResponse, error) {
	// Validation rules check if all conditions pass
	errors := make([]string, 0)
	details := make(map[string]interface{})

	for i, condition := range rule.Definition.Conditions {
		result, err := s.evaluator.EvaluateCondition(&condition, context)
		if err != nil {
			return nil, fmt.Errorf("validation %d evaluation failed: %w", i, err)
		}

		details[fmt.Sprintf("validation_%d", i)] = result
		if !result {
			errors = append(errors, fmt.Sprintf("Validation %d failed", i))
		}
	}

	passed := len(errors) == 0

	return &models.TestRuleResponse{
		Passed:  passed,
		Result:  errors,
		Details: details,
	}, nil
}

// testEnrichmentRule tests an enrichment rule
func (s *RuleService) testEnrichmentRule(rule *models.Rule, context map[string]interface{}) (*models.TestRuleResponse, error) {
	// Enrichment rules add/modify context based on conditions and actions
	enrichedContext := make(map[string]interface{})
	for k, v := range context {
		enrichedContext[k] = v
	}

	details := make(map[string]interface{})
	applied := 0

	// Check conditions and apply actions
	for i, condition := range rule.Definition.Conditions {
		result, err := s.evaluator.EvaluateCondition(&condition, context)
		if err != nil {
			return nil, fmt.Errorf("condition %d evaluation failed: %w", i, err)
		}

		details[fmt.Sprintf("condition_%d", i)] = result

		// If condition passes and there's a corresponding action, apply it
		if result && i < len(rule.Definition.Actions) {
			action := rule.Definition.Actions[i]
			// Apply action metadata to enriched context
			if action.Metadata != nil {
				for k, v := range action.Metadata {
					enrichedContext[k] = v
				}
			}
			applied++
			details[fmt.Sprintf("action_%d", i)] = "applied"
		}
	}

	return &models.TestRuleResponse{
		Passed:  applied > 0,
		Result:  enrichedContext,
		Details: details,
	}, nil
}

// validateRuleDefinition validates a rule definition based on its type
func (s *RuleService) validateRuleDefinition(ruleType models.RuleType, definition *models.RuleDefinition) error {
	if definition == nil {
		return fmt.Errorf("rule definition is required")
	}

	switch ruleType {
	case models.RuleTypeCondition:
		// Condition rules must have at least one condition
		if len(definition.Conditions) == 0 {
			return fmt.Errorf("condition rules must have at least one condition")
		}

	case models.RuleTypeValidation:
		// Validation rules must have at least one condition
		if len(definition.Conditions) == 0 {
			return fmt.Errorf("validation rules must have at least one condition")
		}

	case models.RuleTypeEnrichment:
		// Enrichment rules must have both conditions and actions
		if len(definition.Conditions) == 0 {
			return fmt.Errorf("enrichment rules must have at least one condition")
		}
		if len(definition.Actions) == 0 {
			return fmt.Errorf("enrichment rules must have at least one action")
		}

	default:
		return fmt.Errorf("invalid rule type: %s", ruleType)
	}

	return nil
}

// cacheRule caches a rule in Redis
func (s *RuleService) cacheRule(ctx context.Context, rule *models.Rule) error {
	if s.redis == nil {
		return nil // Redis not configured
	}

	key := ruleCacheKeyPrefix + rule.RuleID
	data, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	return s.redis.Set(ctx, key, data, ruleCacheTTL)
}

// getCachedRule retrieves a cached rule from Redis
func (s *RuleService) getCachedRule(ctx context.Context, ruleID string) (*models.Rule, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("cache not configured")
	}

	key := ruleCacheKeyPrefix + ruleID
	data, err := s.redis.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var rule models.Rule
	if err := json.Unmarshal([]byte(data), &rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule: %w", err)
	}

	return &rule, nil
}

// invalidateRuleCache removes a rule from cache
func (s *RuleService) invalidateRuleCache(ctx context.Context, ruleID string) error {
	if s.redis == nil {
		return nil // Redis not configured
	}

	key := ruleCacheKeyPrefix + ruleID
	return s.redis.Delete(ctx, key)
}
