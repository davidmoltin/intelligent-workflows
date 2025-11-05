# Mocks

This directory contains mock implementations for testing.

## Overview

Mocks are generated using [mockery](https://github.com/vektra/mockery) to create test doubles for interfaces. This allows for isolated unit testing without needing actual implementations.

## Generating Mocks

### Install Mockery

```bash
go install github.com/vektra/mockery/v2@latest
```

### Generate All Mocks

```bash
# Generate mocks using configuration file
mockery

# Or generate for specific interface
mockery --name=WorkflowRepository --dir=./internal/repository --output=./internal/mocks
```

### Configuration

Mockery is configured via `.mockery.yaml` in the project root. The configuration specifies:
- Which interfaces to mock
- Output directory structure
- Package naming
- Whether to use expecter pattern

## Using Mocks in Tests

### Example: Mocking Repository

```go
import (
    "testing"
    "github.com/davidmoltin/intelligent-workflows/internal/mocks"
    "github.com/davidmoltin/intelligent-workflows/internal/models"
    "github.com/stretchr/testify/mock"
)

func TestServiceWithMock(t *testing.T) {
    // Create mock
    mockRepo := mocks.NewWorkflowRepository(t)

    // Setup expectations
    mockRepo.EXPECT().
        GetByID(mock.Anything, testID).
        Return(&models.Workflow{
            ID: testID,
            Name: "Test Workflow",
        }, nil).
        Once()

    // Use mock in service
    service := NewMyService(mockRepo)

    // Call service method
    result, err := service.DoSomething(testID)

    // Verify
    assert.NoError(t, err)
    assert.Equal(t, "Test Workflow", result.Name)

    // Mock automatically verifies expectations
}
```

### Example: Mock with Multiple Calls

```go
func TestMultipleCalls(t *testing.T) {
    mockRepo := mocks.NewWorkflowRepository(t)

    // Expect method to be called twice
    mockRepo.EXPECT().
        Create(mock.Anything, mock.AnythingOfType("*models.Workflow")).
        Return(nil).
        Times(2)

    // Use mock
    service := NewMyService(mockRepo)
    service.CreateWorkflow(workflow1)
    service.CreateWorkflow(workflow2)

    // Expectations are verified automatically
}
```

### Example: Mock with Custom Logic

```go
func TestCustomBehavior(t *testing.T) {
    mockRepo := mocks.NewWorkflowRepository(t)

    // Use Run to execute custom logic
    mockRepo.EXPECT().
        Create(mock.Anything, mock.Anything).
        Run(func(ctx context.Context, w *models.Workflow) {
            // Custom logic
            w.ID = uuid.New()
            w.CreatedAt = time.Now()
        }).
        Return(nil)

    service := NewMyService(mockRepo)
    workflow := &models.Workflow{Name: "Test"}

    err := service.CreateWorkflow(workflow)

    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, workflow.ID)
}
```

## Available Mocks

Once generated, mocks will be available for:

### Repository Interfaces
- `WorkflowRepository` - CRUD operations for workflows
- `ExecutionRepository` - Execution tracking
- `EventRepository` - Event storage
- `RuleRepository` - Rule management

### Service Interfaces
- `WorkflowService` - Workflow business logic
- `ExecutionService` - Execution orchestration
- `NotificationService` - Notifications

## Mock Patterns

### Setup and Verify Pattern

```go
// Setup
mock.EXPECT().Method(args).Return(value)

// Execute
result := service.CallMethod()

// Verify happens automatically
```

### Return Different Values on Sequential Calls

```go
mockRepo.EXPECT().
    GetByID(mock.Anything, id).
    Return(&workflow1, nil).
    Once()

mockRepo.EXPECT().
    GetByID(mock.Anything, id).
    Return(&workflow2, nil).
    Once()
```

### Match Any Argument

```go
// Match any context
mock.EXPECT().Method(mock.Anything, specificArg)

// Match any argument of specific type
mock.EXPECT().Method(mock.AnythingOfType("string"))

// Custom matcher
mock.EXPECT().Method(mock.MatchedBy(func(w *Workflow) bool {
    return w.Name == "expected"
}))
```

## Best Practices

1. **Use Mocks for Unit Tests**: Mocks are perfect for testing business logic in isolation
2. **Use Real Dependencies for Integration Tests**: Don't mock in integration tests
3. **Don't Over-Mock**: Only mock external dependencies, not internal logic
4. **Verify Behavior**: Use expectations to verify method calls
5. **Keep Mocks Simple**: Complex mock setups might indicate design issues

## Regenerating Mocks

When interfaces change:

```bash
# Regenerate all mocks
mockery

# Or regenerate specific mock
mockery --name=WorkflowRepository
```

## Troubleshooting

### Mock Not Found

```bash
# Ensure mockery is installed
which mockery

# Regenerate mocks
mockery
```

### Import Errors

Make sure mock package imports are correct:

```go
import "github.com/davidmoltin/intelligent-workflows/internal/mocks"
```

### Expectation Not Met

If a test fails with "expectation not met":
- Check that the method is actually called
- Verify arguments match expectations
- Ensure call count matches (Once(), Times(n))

## Manual Mocks

For simple cases, you can create manual mocks:

```go
// internal/mocks/simple_repo.go
type SimpleWorkflowRepo struct {
    workflows map[uuid.UUID]*models.Workflow
}

func NewSimpleWorkflowRepo() *SimpleWorkflowRepo {
    return &SimpleWorkflowRepo{
        workflows: make(map[uuid.UUID]*models.Workflow),
    }
}

func (r *SimpleWorkflowRepo) Create(ctx context.Context, w *models.Workflow) error {
    w.ID = uuid.New()
    r.workflows[w.ID] = w
    return nil
}

func (r *SimpleWorkflowRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
    w, ok := r.workflows[id]
    if !ok {
        return nil, ErrNotFound
    }
    return w, nil
}
```

Use manual mocks when:
- Interface is simple
- You need to maintain state across calls
- You want to simulate specific behavior
