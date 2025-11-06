package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
)

// SimpleWorkflowRepository is a simple in-memory mock for testing
type SimpleWorkflowRepository struct {
	mu           sync.RWMutex
	workflows    map[uuid.UUID]*models.Workflow
	byWorkflowID map[string]map[string]*models.Workflow // workflowID -> version -> workflow
}

// NewSimpleWorkflowRepository creates a new simple workflow repository
func NewSimpleWorkflowRepository() *SimpleWorkflowRepository {
	return &SimpleWorkflowRepository{
		workflows:    make(map[uuid.UUID]*models.Workflow),
		byWorkflowID: make(map[string]map[string]*models.Workflow),
	}
}

// Create creates a new workflow
func (r *SimpleWorkflowRepository) Create(ctx context.Context, workflow *models.Workflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if workflow.ID == uuid.Nil {
		workflow.ID = uuid.New()
	}

	now := time.Now()
	workflow.CreatedAt = now
	workflow.UpdatedAt = now

	r.workflows[workflow.ID] = workflow

	// Index by workflow_id and version
	if _, exists := r.byWorkflowID[workflow.WorkflowID]; !exists {
		r.byWorkflowID[workflow.WorkflowID] = make(map[string]*models.Workflow)
	}
	r.byWorkflowID[workflow.WorkflowID][workflow.Version] = workflow

	return nil
}

// GetByID retrieves a workflow by ID
func (r *SimpleWorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflow, exists := r.workflows[id]
	if !exists {
		return nil, fmt.Errorf("workflow not found")
	}

	return workflow, nil
}

// GetByWorkflowID retrieves a workflow by workflow_id and version
func (r *SimpleWorkflowRepository) GetByWorkflowID(ctx context.Context, workflowID, version string) (*models.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions, exists := r.byWorkflowID[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow not found")
	}

	workflow, exists := versions[version]
	if !exists {
		return nil, fmt.Errorf("workflow version not found")
	}

	return workflow, nil
}

// List retrieves all workflows with pagination
func (r *SimpleWorkflowRepository) List(ctx context.Context, limit, offset int) ([]*models.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflows := make([]*models.Workflow, 0, len(r.workflows))
	for _, w := range r.workflows {
		workflows = append(workflows, w)
	}

	// Simple pagination
	start := offset
	if start > len(workflows) {
		return []*models.Workflow{}, nil
	}

	end := start + limit
	if end > len(workflows) {
		end = len(workflows)
	}

	return workflows[start:end], nil
}

// Update updates a workflow
func (r *SimpleWorkflowRepository) Update(ctx context.Context, workflow *models.Workflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.workflows[workflow.ID]; !exists {
		return fmt.Errorf("workflow not found")
	}

	workflow.UpdatedAt = time.Now()
	r.workflows[workflow.ID] = workflow

	// Update index
	if versions, exists := r.byWorkflowID[workflow.WorkflowID]; exists {
		versions[workflow.Version] = workflow
	}

	return nil
}

// Delete deletes a workflow
func (r *SimpleWorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	workflow, exists := r.workflows[id]
	if !exists {
		return fmt.Errorf("workflow not found")
	}

	delete(r.workflows, id)

	// Remove from index
	if versions, exists := r.byWorkflowID[workflow.WorkflowID]; exists {
		delete(versions, workflow.Version)
		if len(versions) == 0 {
			delete(r.byWorkflowID, workflow.WorkflowID)
		}
	}

	return nil
}

// FindByTriggerEvent finds workflows triggered by an event
func (r *SimpleWorkflowRepository) FindByTriggerEvent(ctx context.Context, eventType string) ([]*models.Workflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matching []*models.Workflow
	for _, w := range r.workflows {
		if w.Enabled && w.Definition.Trigger.Type == "event" && w.Definition.Trigger.Event == eventType {
			matching = append(matching, w)
		}
	}

	return matching, nil
}

// Count returns the total number of workflows
func (r *SimpleWorkflowRepository) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return int64(len(r.workflows)), nil
}

// Reset clears all workflows (useful for testing)
func (r *SimpleWorkflowRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.workflows = make(map[uuid.UUID]*models.Workflow)
	r.byWorkflowID = make(map[string]map[string]*models.Workflow)
}
