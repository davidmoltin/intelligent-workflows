package services

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// ScheduleRepository defines the interface for schedule data access
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *models.WorkflowSchedule) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error)
	GetByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.WorkflowSchedule, error)
	GetDueSchedules(ctx context.Context) ([]*models.WorkflowSchedule, error)
	Update(ctx context.Context, schedule *models.WorkflowSchedule) error
	UpdateNextTrigger(ctx context.Context, id uuid.UUID, lastTriggered, nextTrigger time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*models.WorkflowSchedule, int64, error)
}

// ScheduleService handles workflow scheduling logic
type ScheduleService struct {
	scheduleRepo ScheduleRepository
	logger       *logger.Logger
	parser       cron.Parser
}

// NewScheduleService creates a new schedule service
func NewScheduleService(scheduleRepo ScheduleRepository, log *logger.Logger) *ScheduleService {
	// Create parser that supports standard cron format (5 fields) and optional seconds field (6 fields)
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		logger:       log,
		parser:       parser,
	}
}

// CreateSchedule creates a new workflow schedule
func (s *ScheduleService) CreateSchedule(ctx context.Context, workflowID uuid.UUID, req *models.CreateScheduleRequest) (*models.WorkflowSchedule, error) {
	// Validate cron expression
	schedule, err := s.parser.Parse(req.CronExpression)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Set default timezone if not provided
	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	// Calculate next trigger time
	nextTrigger := schedule.Next(time.Now().In(loc))

	// Set default enabled if not provided
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	// Create schedule
	newSchedule := &models.WorkflowSchedule{
		ID:             uuid.New(),
		WorkflowID:     workflowID,
		CronExpression: req.CronExpression,
		Timezone:       timezone,
		Enabled:        enabled,
		NextTriggerAt:  &nextTrigger,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.scheduleRepo.Create(ctx, newSchedule); err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	s.logger.Infof("Created schedule %s for workflow %s with cron %s", newSchedule.ID, workflowID, req.CronExpression)

	return newSchedule, nil
}

// GetSchedule retrieves a schedule by ID
func (s *ScheduleService) GetSchedule(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error) {
	return s.scheduleRepo.GetByID(ctx, id)
}

// GetWorkflowSchedules retrieves all schedules for a workflow
func (s *ScheduleService) GetWorkflowSchedules(ctx context.Context, workflowID uuid.UUID) ([]*models.WorkflowSchedule, error) {
	return s.scheduleRepo.GetByWorkflowID(ctx, workflowID)
}

// UpdateSchedule updates a workflow schedule
func (s *ScheduleService) UpdateSchedule(ctx context.Context, id uuid.UUID, req *models.UpdateScheduleRequest) (*models.WorkflowSchedule, error) {
	// Get existing schedule
	existing, err := s.scheduleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.CronExpression != nil {
		// Validate new cron expression
		_, err := s.parser.Parse(*req.CronExpression)
		if err != nil {
			return nil, fmt.Errorf("invalid cron expression: %w", err)
		}
		existing.CronExpression = *req.CronExpression
	}

	if req.Timezone != nil {
		// Validate new timezone
		_, err := time.LoadLocation(*req.Timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone: %w", err)
		}
		existing.Timezone = *req.Timezone
	}

	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	// Recalculate next trigger time if cron or timezone changed
	if req.CronExpression != nil || req.Timezone != nil {
		schedule, err := s.parser.Parse(existing.CronExpression)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cron expression: %w", err)
		}

		loc, err := time.LoadLocation(existing.Timezone)
		if err != nil {
			return nil, fmt.Errorf("failed to load timezone: %w", err)
		}

		nextTrigger := schedule.Next(time.Now().In(loc))
		existing.NextTriggerAt = &nextTrigger
	}

	// Save updates
	if err := s.scheduleRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	s.logger.Infof("Updated schedule %s", id)

	return existing, nil
}

// DeleteSchedule deletes a workflow schedule
func (s *ScheduleService) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	if err := s.scheduleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	s.logger.Infof("Deleted schedule %s", id)

	return nil
}

// GetDueSchedules retrieves all schedules that are due to run
func (s *ScheduleService) GetDueSchedules(ctx context.Context) ([]*models.WorkflowSchedule, error) {
	return s.scheduleRepo.GetDueSchedules(ctx)
}

// MarkTriggered marks a schedule as triggered and calculates the next run time
func (s *ScheduleService) MarkTriggered(ctx context.Context, id uuid.UUID) error {
	schedule, err := s.scheduleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Parse cron expression
	cronSchedule, err := s.parser.Parse(schedule.CronExpression)
	if err != nil {
		return fmt.Errorf("failed to parse cron expression: %w", err)
	}

	// Load timezone
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return fmt.Errorf("failed to load timezone: %w", err)
	}

	// Calculate next trigger time
	now := time.Now().In(loc)
	nextTrigger := cronSchedule.Next(now)

	// Update trigger times
	if err := s.scheduleRepo.UpdateNextTrigger(ctx, id, now, nextTrigger); err != nil {
		return fmt.Errorf("failed to update trigger times: %w", err)
	}

	s.logger.Debugf("Schedule %s marked as triggered, next run at %s", id, nextTrigger)

	return nil
}

// GetNextRuns calculates the next N run times for a schedule
func (s *ScheduleService) GetNextRuns(ctx context.Context, id uuid.UUID, count int) ([]time.Time, error) {
	if count <= 0 || count > 100 {
		count = 10 // Default to 10, max 100
	}

	schedule, err := s.scheduleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Parse cron expression
	cronSchedule, err := s.parser.Parse(schedule.CronExpression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cron expression: %w", err)
	}

	// Load timezone
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	// Calculate next N runs
	runs := make([]time.Time, count)
	current := time.Now().In(loc)
	for i := 0; i < count; i++ {
		current = cronSchedule.Next(current)
		runs[i] = current
	}

	return runs, nil
}

// ValidateCronExpression validates a cron expression
func (s *ScheduleService) ValidateCronExpression(expression string) error {
	_, err := s.parser.Parse(expression)
	return err
}

// ListSchedules retrieves all schedules with pagination
func (s *ScheduleService) ListSchedules(ctx context.Context, limit, offset int) ([]*models.WorkflowSchedule, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return s.scheduleRepo.List(ctx, limit, offset)
}
