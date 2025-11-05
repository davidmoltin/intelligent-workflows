# Integration Tests

This directory contains integration tests for the Intelligent Workflows service.

## Overview

Integration tests verify that different components of the system work together correctly. They test interactions between:
- Database repositories and PostgreSQL
- API handlers and services
- Service layer and data layer
- External integrations (webhooks, notifications)

## Running Integration Tests

### Prerequisites

1. PostgreSQL database running (default: localhost:5432)
2. Redis instance running (default: localhost:6379)
3. Set environment variable to enable tests

### Run All Integration Tests

```bash
# Enable integration tests
export INTEGRATION_TESTS=1

# Run all integration tests
go test ./tests/integration/... -v

# Run with coverage
go test ./tests/integration/... -v -coverprofile=coverage.out

# Run specific test
go test ./tests/integration/... -v -run TestWorkflowRepository_Create
```

### Run Without Integration Tests

```bash
# Integration tests are skipped by default
go test ./tests/integration/...

# Or use short mode
go test ./tests/integration/... -short
```

## Test Structure

### Suite Pattern

Integration tests use a test suite pattern with shared setup and teardown:

```go
func TestMyIntegration(t *testing.T) {
    suite := SetupSuite(t)
    defer TeardownSuite(t)
    suite.ResetDatabase(t)

    // Test code here
}
```

### Test Database

Each test suite:
1. Creates a temporary test database
2. Runs migrations
3. Provides connection pool
4. Cleans up after tests

The database is automatically cleaned up after each test run.

## Writing Integration Tests

### Example: Testing a Repository

```go
func TestMyRepository_Method(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    suite := SetupSuite(t)
    defer TeardownSuite(t)
    suite.ResetDatabase(t)

    ctx := suite.GetContext(t)
    repo := postgres.NewMyRepository(suite.Pool)

    // Create test data using fixtures
    fixtures := testutil.NewFixtureBuilder()
    entity := fixtures.MyEntity()

    // Test the method
    err := repo.Create(ctx, entity)
    require.NoError(t, err)

    // Verify results
    retrieved, err := repo.GetByID(ctx, entity.ID)
    require.NoError(t, err)
    assert.Equal(t, entity.Name, retrieved.Name)
}
```

### Example: Testing API Handlers

```go
func TestWorkflowHandler_Create(t *testing.T) {
    suite := SetupSuite(t)
    defer TeardownSuite(t)
    suite.ResetDatabase(t)

    // Setup handler with dependencies
    repo := postgres.NewWorkflowRepository(suite.Pool)
    service := services.NewWorkflowService(repo)
    handler := handlers.NewWorkflowHandler(service)

    // Create test request
    req := httptest.NewRequest("POST", "/api/v1/workflows", body)
    w := httptest.NewRecorder()

    // Execute
    handler.Create(w, req)

    // Assert
    assert.Equal(t, http.StatusCreated, w.Code)
}
```

## Best Practices

1. **Isolation**: Each test should be independent and not rely on other tests
2. **Cleanup**: Always clean up test data using `suite.ResetDatabase(t)`
3. **Fixtures**: Use the fixture builder for creating test data
4. **Skip in Short Mode**: Allow tests to be skipped with `go test -short`
5. **Descriptive Names**: Use clear test names that describe what is being tested
6. **Assertions**: Use `require` for critical assertions, `assert` for non-critical ones

## CI/CD Integration

Integration tests are run in the CI pipeline:

```yaml
# .github/workflows/test.yml
- name: Run Integration Tests
  run: |
    docker-compose up -d postgres redis
    export INTEGRATION_TESTS=1
    go test ./tests/integration/... -v -coverprofile=coverage.out
  env:
    DB_HOST: localhost
    REDIS_HOST: localhost
```

## Troubleshooting

### Database Connection Issues

```bash
# Check PostgreSQL is running
docker-compose ps

# View logs
docker-compose logs postgres

# Reset database
docker-compose down -v
docker-compose up -d
```

### Test Cleanup Issues

If tests leave data behind:

```bash
# Manually clean up test databases
psql -h localhost -U postgres -c "DROP DATABASE workflows_test_*"
```

## Coverage Reports

Generate and view coverage:

```bash
# Generate coverage
go test ./tests/integration/... -coverprofile=coverage.out

# View in browser
go tool cover -html=coverage.out

# View summary
go tool cover -func=coverage.out
```
