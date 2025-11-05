# End-to-End (E2E) Tests

This directory contains end-to-end tests for the Intelligent Workflows service.

## Overview

E2E tests verify the complete system behavior from the perspective of an external client. They:
- Start a real HTTP server
- Use real database connections
- Make actual HTTP requests
- Verify full request/response cycles
- Test complete user workflows

## Running E2E Tests

### Prerequisites

1. PostgreSQL database running (default: localhost:5432)
2. Redis instance running (default: localhost:6379)
3. Set environment variable to enable tests

### Run All E2E Tests

```bash
# Enable E2E tests
export E2E_TESTS=1

# Run all E2E tests
go test ./tests/e2e/... -v

# Run with coverage
go test ./tests/e2e/... -v -coverprofile=coverage.out

# Run specific test
go test ./tests/e2e/... -v -run TestWorkflowAPI_CreateWorkflow
```

### Run Without E2E Tests

```bash
# E2E tests are skipped by default
go test ./tests/e2e/...

# Or use short mode
go test ./tests/e2e/... -short
```

## Test Structure

### Test Server Pattern

E2E tests use a test server that:
1. Creates a temporary test database
2. Runs migrations
3. Starts an HTTP server on a random port
4. Provides a base URL for making requests
5. Cleans up after tests

Example:

```go
func TestMyAPI(t *testing.T) {
    server := NewTestServer(t)
    server.Start()
    defer server.Stop()
    server.ResetDatabase()

    // Make HTTP requests to server.BaseURL
}
```

### Making Requests

```go
// POST request
body, _ := json.Marshal(requestData)
resp, err := http.Post(
    server.BaseURL+"/api/v1/workflows",
    "application/json",
    bytes.NewBuffer(body),
)
defer resp.Body.Close()

// GET request
resp, err := http.Get(server.BaseURL + "/api/v1/workflows")
defer resp.Body.Close()

// Decode response
var result MyResponse
json.NewDecoder(resp.Body).Decode(&result)
```

## Writing E2E Tests

### Example: Testing Workflow Creation

```go
func TestWorkflowAPI_CreateWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    server := NewTestServer(t)
    server.Start()
    defer server.Stop()
    server.ResetDatabase()

    // Create request
    workflow := models.CreateWorkflowRequest{
        WorkflowID: "test-workflow",
        Version:    "1.0.0",
        Name:       "Test Workflow",
        Definition: models.WorkflowDefinition{
            // ... workflow definition
        },
    }

    body, _ := json.Marshal(workflow)

    // Make request
    resp, err := http.Post(
        server.BaseURL+"/api/v1/workflows",
        "application/json",
        bytes.NewBuffer(body),
    )
    require.NoError(t, err)
    defer resp.Body.Close()

    // Verify response
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    var created models.Workflow
    json.NewDecoder(resp.Body).Decode(&created)
    assert.Equal(t, "test-workflow", created.WorkflowID)
}
```

### Example: Testing Complete User Flow

```go
func TestCompleteWorkflowFlow(t *testing.T) {
    server := NewTestServer(t)
    server.Start()
    defer server.Stop()
    server.ResetDatabase()

    // 1. Create workflow
    workflow := createWorkflow(t, server)

    // 2. Trigger workflow with event
    event := triggerEvent(t, server, workflow)

    // 3. Verify execution was created
    execution := getExecution(t, server, event)

    // 4. Verify execution completed successfully
    assert.Equal(t, models.ExecutionStatusCompleted, execution.Status)
}
```

## Best Practices

1. **Isolation**: Each test starts fresh with `server.ResetDatabase()`
2. **Cleanup**: Always defer `server.Stop()`
3. **Resource Management**: Close response bodies with `defer resp.Body.Close()`
4. **Skip in Short Mode**: Allow tests to be skipped with `go test -short`
5. **Real Scenarios**: Test complete user workflows, not just single endpoints
6. **Error Cases**: Test error scenarios and edge cases

## Testing Scenarios

### API Endpoints
- ✅ Create workflow
- ✅ Get workflow by ID
- ✅ List workflows
- ✅ Update workflow
- ✅ Delete workflow
- ✅ Health check

### Workflow Execution (TODO)
- Trigger workflow with event
- Verify execution status
- Check step execution trace
- Test conditional logic
- Test parallel execution

### Error Handling (TODO)
- Invalid requests
- Not found errors
- Validation errors
- Database errors

## CI/CD Integration

E2E tests run in the CI pipeline after integration tests:

```yaml
# .github/workflows/test.yml
- name: Run E2E Tests
  run: |
    docker-compose up -d postgres redis
    export E2E_TESTS=1
    go test ./tests/e2e/... -v
  env:
    DB_HOST: localhost
    REDIS_HOST: localhost
```

## Troubleshooting

### Server Start Issues

```bash
# Check if port is already in use
lsof -i :8080

# View server logs
# Tests output server errors to test logs
```

### Database Issues

```bash
# Verify PostgreSQL is running
docker-compose ps postgres

# Check logs
docker-compose logs postgres
```

### Timing Issues

Some tests may need to wait for async operations:

```go
// Wait for execution to complete
time.Sleep(100 * time.Millisecond)

// Or use polling
for i := 0; i < 10; i++ {
    execution := getExecution(t, server, id)
    if execution.Status == models.ExecutionStatusCompleted {
        break
    }
    time.Sleep(100 * time.Millisecond)
}
```

## Performance

E2E tests are slower than unit/integration tests because they:
- Start a real HTTP server
- Make network requests
- Use real database connections

Typical test times:
- Unit tests: <10ms
- Integration tests: 10-100ms
- E2E tests: 100-1000ms

Keep E2E tests focused on critical user paths. Use integration/unit tests for detailed testing.
