package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Mock ScheduleRepository for testing
type mockScheduleRepo struct {
	createFunc        func(ctx context.Context, schedule *models.WorkflowSchedule) error
	getByIDFunc       func(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error)
	getByWorkflowFunc func(ctx context.Context, workflowID uuid.UUID) ([]*models.WorkflowSchedule, error)
	getDueFunc        func(ctx context.Context) ([]*models.WorkflowSchedule, error)
	updateFunc        func(ctx context.Context, schedule *models.WorkflowSchedule) error
	updateNextFunc    func(ctx context.Context, id uuid.UUID, lastTriggered, nextTrigger time.Time) error
	deleteFunc        func(ctx context.Context, id uuid.UUID) error
	listFunc          func(ctx context.Context, limit, offset int) ([]*models.WorkflowSchedule, int64, error)
}

func (m *mockScheduleRepo) Create(ctx context.Context, schedule *models.WorkflowSchedule) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, schedule)
	}
	return nil
}

func (m *mockScheduleRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockScheduleRepo) GetByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.WorkflowSchedule, error) {
	if m.getByWorkflowFunc != nil {
		return m.getByWorkflowFunc(ctx, workflowID)
	}
	return []*models.WorkflowSchedule{}, nil
}

func (m *mockScheduleRepo) GetDueSchedules(ctx context.Context) ([]*models.WorkflowSchedule, error) {
	if m.getDueFunc != nil {
		return m.getDueFunc(ctx)
	}
	return []*models.WorkflowSchedule{}, nil
}

func (m *mockScheduleRepo) Update(ctx context.Context, schedule *models.WorkflowSchedule) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, schedule)
	}
	return nil
}

func (m *mockScheduleRepo) UpdateNextTrigger(ctx context.Context, id uuid.UUID, lastTriggered, nextTrigger time.Time) error {
	if m.updateNextFunc != nil {
		return m.updateNextFunc(ctx, id, lastTriggered, nextTrigger)
	}
	return nil
}

func (m *mockScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockScheduleRepo) List(ctx context.Context, limit, offset int) ([]*models.WorkflowSchedule, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, limit, offset)
	}
	return []*models.WorkflowSchedule{}, 0, nil
}

// TestCreateSchedule tests schedule creation
func TestCreateSchedule(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("creates schedule with valid cron expression", func(t *testing.T) {
		workflowID := uuid.New()
		var capturedSchedule *models.WorkflowSchedule

		repo := &mockScheduleRepo{
			createFunc: func(ctx context.Context, schedule *models.WorkflowSchedule) error {
				capturedSchedule = schedule
				return nil
			},
		}

		service := NewScheduleService(repo, log)

		req := &models.CreateScheduleRequest{
			CronExpression: "0 0 9 * * *", // Daily at 9 AM (with seconds field)
			Timezone:       "UTC",
		}

		schedule, err := service.CreateSchedule(context.Background(), workflowID, req)

		assert.NoError(t, err)
		assert.NotNil(t, schedule)
		assert.Equal(t, workflowID, schedule.WorkflowID)
		assert.Equal(t, "0 0 9 * * *", schedule.CronExpression)
		assert.Equal(t, "UTC", schedule.Timezone)
		assert.True(t, schedule.Enabled)
		assert.NotNil(t, schedule.NextTriggerAt)
		assert.NotNil(t, capturedSchedule)
	})

	t.Run("rejects invalid cron expression", func(t *testing.T) {
		workflowID := uuid.New()

		repo := &mockScheduleRepo{}
		service := NewScheduleService(repo, log)

		req := &models.CreateScheduleRequest{
			CronExpression: "invalid cron",
			Timezone:       "UTC",
		}

		schedule, err := service.CreateSchedule(context.Background(), workflowID, req)

		assert.Error(t, err)
		assert.Nil(t, schedule)
		assert.Contains(t, err.Error(), "invalid cron expression")
	})

	t.Run("rejects invalid timezone", func(t *testing.T) {
		workflowID := uuid.New()

		repo := &mockScheduleRepo{}
		service := NewScheduleService(repo, log)

		req := &models.CreateScheduleRequest{
			CronExpression: "0 0 9 * * *",
			Timezone:       "InvalidTimezone",
		}

		schedule, err := service.CreateSchedule(context.Background(), workflowID, req)

		assert.Error(t, err)
		assert.Nil(t, schedule)
		assert.Contains(t, err.Error(), "invalid timezone")
	})

	t.Run("uses default timezone when not provided", func(t *testing.T) {
		workflowID := uuid.New()

		repo := &mockScheduleRepo{
			createFunc: func(ctx context.Context, schedule *models.WorkflowSchedule) error {
				return nil
			},
		}

		service := NewScheduleService(repo, log)

		req := &models.CreateScheduleRequest{
			CronExpression: "0 0 9 * * *",
		}

		schedule, err := service.CreateSchedule(context.Background(), workflowID, req)

		assert.NoError(t, err)
		assert.NotNil(t, schedule)
		assert.Equal(t, "UTC", schedule.Timezone)
	})

	t.Run("respects enabled flag", func(t *testing.T) {
		workflowID := uuid.New()
		enabled := false

		repo := &mockScheduleRepo{
			createFunc: func(ctx context.Context, schedule *models.WorkflowSchedule) error {
				return nil
			},
		}

		service := NewScheduleService(repo, log)

		req := &models.CreateScheduleRequest{
			CronExpression: "0 0 9 * * *",
			Timezone:       "UTC",
			Enabled:        &enabled,
		}

		schedule, err := service.CreateSchedule(context.Background(), workflowID, req)

		assert.NoError(t, err)
		assert.NotNil(t, schedule)
		assert.False(t, schedule.Enabled)
	})
}

// TestUpdateSchedule tests schedule updates
func TestUpdateSchedule(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("updates cron expression", func(t *testing.T) {
		scheduleID := uuid.New()
		existingSchedule := &models.WorkflowSchedule{
			ID:             scheduleID,
			WorkflowID:     uuid.New(),
			CronExpression: "0 0 9 * * *",
			Timezone:       "UTC",
			Enabled:        true,
		}

		repo := &mockScheduleRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error) {
				return existingSchedule, nil
			},
			updateFunc: func(ctx context.Context, schedule *models.WorkflowSchedule) error {
				return nil
			},
		}

		service := NewScheduleService(repo, log)

		newCron := "0 0 12 * * *" // Change to noon (with seconds)
		req := &models.UpdateScheduleRequest{
			CronExpression: &newCron,
		}

		schedule, err := service.UpdateSchedule(context.Background(), scheduleID, req)

		assert.NoError(t, err)
		assert.NotNil(t, schedule)
		assert.Equal(t, newCron, schedule.CronExpression)
		assert.NotNil(t, schedule.NextTriggerAt) // Should recalculate
	})

	t.Run("rejects invalid cron expression in update", func(t *testing.T) {
		scheduleID := uuid.New()
		existingSchedule := &models.WorkflowSchedule{
			ID:             scheduleID,
			WorkflowID:     uuid.New(),
			CronExpression: "0 0 9 * * *",
			Timezone:       "UTC",
			Enabled:        true,
		}

		repo := &mockScheduleRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error) {
				return existingSchedule, nil
			},
		}

		service := NewScheduleService(repo, log)

		invalidCron := "invalid"
		req := &models.UpdateScheduleRequest{
			CronExpression: &invalidCron,
		}

		schedule, err := service.UpdateSchedule(context.Background(), scheduleID, req)

		assert.Error(t, err)
		assert.Nil(t, schedule)
		assert.Contains(t, err.Error(), "invalid cron expression")
	})

	t.Run("updates enabled flag", func(t *testing.T) {
		scheduleID := uuid.New()
		existingSchedule := &models.WorkflowSchedule{
			ID:             scheduleID,
			WorkflowID:     uuid.New(),
			CronExpression: "0 0 9 * * *",
			Timezone:       "UTC",
			Enabled:        true,
		}

		repo := &mockScheduleRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error) {
				return existingSchedule, nil
			},
			updateFunc: func(ctx context.Context, schedule *models.WorkflowSchedule) error {
				return nil
			},
		}

		service := NewScheduleService(repo, log)

		enabled := false
		req := &models.UpdateScheduleRequest{
			Enabled: &enabled,
		}

		schedule, err := service.UpdateSchedule(context.Background(), scheduleID, req)

		assert.NoError(t, err)
		assert.NotNil(t, schedule)
		assert.False(t, schedule.Enabled)
	})
}

// TestMarkTriggered tests marking a schedule as triggered
func TestMarkTriggered(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("calculates next trigger time", func(t *testing.T) {
		scheduleID := uuid.New()
		schedule := &models.WorkflowSchedule{
			ID:             scheduleID,
			WorkflowID:     uuid.New(),
			CronExpression: "0 0 * * * *", // Every hour
			Timezone:       "UTC",
			Enabled:        true,
		}

		var capturedLastTriggered, capturedNextTrigger time.Time

		repo := &mockScheduleRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error) {
				return schedule, nil
			},
			updateNextFunc: func(ctx context.Context, id uuid.UUID, lastTriggered, nextTrigger time.Time) error {
				capturedLastTriggered = lastTriggered
				capturedNextTrigger = nextTrigger
				return nil
			},
		}

		service := NewScheduleService(repo, log)

		err := service.MarkTriggered(context.Background(), scheduleID)

		assert.NoError(t, err)
		assert.NotZero(t, capturedLastTriggered)
		assert.NotZero(t, capturedNextTrigger)
		assert.True(t, capturedNextTrigger.After(capturedLastTriggered))
	})
}

// TestGetNextRuns tests calculating next run times
func TestGetNextRuns(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("calculates next N runs", func(t *testing.T) {
		scheduleID := uuid.New()
		schedule := &models.WorkflowSchedule{
			ID:             scheduleID,
			WorkflowID:     uuid.New(),
			CronExpression: "0 0 * * * *", // Every hour
			Timezone:       "UTC",
			Enabled:        true,
		}

		repo := &mockScheduleRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error) {
				return schedule, nil
			},
		}

		service := NewScheduleService(repo, log)

		runs, err := service.GetNextRuns(context.Background(), scheduleID, 5)

		assert.NoError(t, err)
		assert.Len(t, runs, 5)

		// Verify each run is after the previous one
		for i := 1; i < len(runs); i++ {
			assert.True(t, runs[i].After(runs[i-1]))
		}
	})

	t.Run("limits count to maximum", func(t *testing.T) {
		scheduleID := uuid.New()
		schedule := &models.WorkflowSchedule{
			ID:             scheduleID,
			WorkflowID:     uuid.New(),
			CronExpression: "0 0 * * * *",
			Timezone:       "UTC",
			Enabled:        true,
		}

		repo := &mockScheduleRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.WorkflowSchedule, error) {
				return schedule, nil
			},
		}

		service := NewScheduleService(repo, log)

		runs, err := service.GetNextRuns(context.Background(), scheduleID, 150) // Over max

		assert.NoError(t, err)
		assert.Len(t, runs, 10) // Should default to 10
	})
}

// TestValidateCronExpression tests cron expression validation
func TestValidateCronExpression(t *testing.T) {
	log := logger.NewForTesting()
	service := NewScheduleService(&mockScheduleRepo{}, log)

	t.Run("validates correct cron expressions", func(t *testing.T) {
		validExpressions := []string{
			"0 0 9 * * *",         // Daily at 9 AM (with seconds)
			"0 */15 * * * *",      // Every 15 minutes (with seconds)
			"0 0 0 * * MON",       // Every Monday at midnight
			"0 0 0,12 * * *",      // Twice daily
			"@hourly",             // Descriptor
			"0 0 0 * * *",         // With seconds
		}

		for _, expr := range validExpressions {
			err := service.ValidateCronExpression(expr)
			assert.NoError(t, err, "Expression should be valid: %s", expr)
		}
	})

	t.Run("rejects invalid cron expressions", func(t *testing.T) {
		invalidExpressions := []string{
			"invalid",
			"0 60 * * * *",       // Invalid minute
			"* * * * * * *",      // Too many fields
			"",                   // Empty
		}

		for _, expr := range invalidExpressions {
			err := service.ValidateCronExpression(expr)
			assert.Error(t, err, "Expression should be invalid: %s", expr)
		}
	})
}

// TestDeleteSchedule tests schedule deletion
func TestDeleteSchedule(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("deletes existing schedule", func(t *testing.T) {
		scheduleID := uuid.New()
		deleted := false

		repo := &mockScheduleRepo{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				if id == scheduleID {
					deleted = true
					return nil
				}
				return errors.New("not found")
			},
		}

		service := NewScheduleService(repo, log)

		err := service.DeleteSchedule(context.Background(), scheduleID)

		assert.NoError(t, err)
		assert.True(t, deleted)
	})

	t.Run("returns error for non-existent schedule", func(t *testing.T) {
		scheduleID := uuid.New()

		repo := &mockScheduleRepo{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return errors.New("not found")
			},
		}

		service := NewScheduleService(repo, log)

		err := service.DeleteSchedule(context.Background(), scheduleID)

		assert.Error(t, err)
	})
}

// TestListSchedules tests schedule listing
func TestListSchedules(t *testing.T) {
	log := logger.NewForTesting()

	t.Run("lists schedules with pagination", func(t *testing.T) {
		repo := &mockScheduleRepo{
			listFunc: func(ctx context.Context, limit, offset int) ([]*models.WorkflowSchedule, int64, error) {
				return []*models.WorkflowSchedule{
					{ID: uuid.New()},
					{ID: uuid.New()},
				}, 2, nil
			},
		}

		service := NewScheduleService(repo, log)

		schedules, total, err := service.ListSchedules(context.Background(), 10, 0)

		assert.NoError(t, err)
		assert.Len(t, schedules, 2)
		assert.Equal(t, int64(2), total)
	})

	t.Run("enforces max limit", func(t *testing.T) {
		var capturedLimit int

		repo := &mockScheduleRepo{
			listFunc: func(ctx context.Context, limit, offset int) ([]*models.WorkflowSchedule, int64, error) {
				capturedLimit = limit
				return []*models.WorkflowSchedule{}, 0, nil
			},
		}

		service := NewScheduleService(repo, log)

		_, _, err := service.ListSchedules(context.Background(), 150, 0) // Over max

		assert.NoError(t, err)
		assert.Equal(t, 50, capturedLimit) // Should default to 50
	})
}
