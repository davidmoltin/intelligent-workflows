# Testing Guide

This document provides a comprehensive guide to testing in the Intelligent Workflows project.

## Table of Contents

- [Overview](#overview)
- [Test Types](#test-types)
- [Running Tests](#running-tests)
- [Writing Tests](#writing-tests)
- [Test Coverage](#test-coverage)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)

## Overview

The project uses a comprehensive testing strategy with multiple levels of testing:

1. **Unit Tests** - Test individual functions and methods in isolation
2. **Integration Tests** - Test component interactions with real dependencies
3. **E2E Tests** - Test complete workflows through HTTP API
4. **Load Tests** - Test system performance under load

## Test Types

### Unit Tests

Location: `*_test.go` files alongside source code

**Purpose:** Test individual units of code in isolation

**Characteristics:**
- Fast (< 10ms per test)
- No external dependencies (use mocks)
- High code coverage

**Example:**
```go
func TestConfig_Validate(t *testing.T) {
    cfg := &Config{
        Server: ServerConfig{Port: 0},
    }

    err := cfg.Validate()

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid server port")
}
```

**Run:**
```bash
# All unit tests
go test ./internal/... ./pkg/... -v

# Specific package
go test ./pkg/config -v

# With coverage
go test ./internal/... -coverprofile=coverage.out

# With race detection
go test ./internal/... -race
```

### Integration Tests

Location: `tests/integration/`

**Purpose:** Test component interactions with real dependencies (database, Redis)

**Characteristics:**
- Slower (10-100ms per test)
- Use real database/Redis
- Test repository layer, services

**Example:**
```go
func TestWorkflowRepository_Create(t *testing.T) {
    suite := SetupSuite(t)
    defer TeardownSuite(t)
    suite.ResetDatabase(t)

    ctx := suite.GetContext(t)
    repo := postgres.NewWorkflowRepository(suite.Pool)

    workflow := fixtures.Workflow()
    err := repo.Create(ctx, workflow)

    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, workflow.ID)
}
```

**Run:**
```bash
# Enable integration tests
export INTEGRATION_TESTS=1

# Run integration tests
go test ./tests/integration/... -v

# Prerequisites
docker-compose up -d postgres redis
```

### E2E Tests

Location: `tests/e2e/`

**Purpose:** Test complete system behavior through HTTP API

**Characteristics:**
- Slowest (100-1000ms per test)
- Real HTTP server
- Complete request/response cycles

**Example:**
```go
func TestWorkflowAPI_CreateWorkflow(t *testing.T) {
    server := NewTestServer(t)
    server.Start()
    defer server.Stop()

    workflow := models.CreateWorkflowRequest{ /* ... */ }
    body, _ := json.Marshal(workflow)

    resp, err := http.Post(
        server.BaseURL+"/api/v1/workflows",
        "application/json",
        bytes.NewBuffer(body),
    )

    assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

**Run:**
```bash
# Enable E2E tests
export E2E_TESTS=1

# Run E2E tests
go test ./tests/e2e/... -v

# Prerequisites
docker-compose up -d postgres redis
```

### Load Tests

Location: `tests/load/`

**Purpose:** Test system performance under load

**Tools:** K6, Vegeta, Hey, Apache Bench

**Run:**
```bash
# K6
k6 run tests/load/k6-workflow-load.js

# Simple load test
bash tests/load/simple-load.sh

# Vegeta
bash tests/load/vegeta-load.sh
```

See [tests/load/README.md](tests/load/README.md) for details.

## Running Tests

### Quick Start

```bash
# Run unit tests
go test ./internal/... ./pkg/... -v

# Run all tests
bash scripts/test-all.sh

# Run with coverage
bash scripts/test-coverage.sh
```

### Run Specific Tests

```bash
# Run tests in specific package
go test ./internal/models -v

# Run specific test
go test ./internal/models -run TestWorkflowDefinition_Scan -v

# Run tests matching pattern
go test ./... -run TestWorkflow -v
```

### Test Options

```bash
# Verbose output
go test ./... -v

# Race detection
go test ./... -race

# Coverage
go test ./... -coverprofile=coverage.out

# Short mode (skip slow tests)
go test ./... -short

# Parallel execution
go test ./... -parallel 4

# Timeout
go test ./... -timeout 30s

# Fail fast
go test ./... -failfast
```

### Docker-based Testing

```bash
# Start dependencies
docker-compose up -d postgres redis

# Run tests in container
docker-compose run --rm test

# Stop dependencies
docker-compose down
```

## Writing Tests

### Test Structure

```go
func TestFunctionName(t *testing.T) {
    // Arrange - Set up test data and dependencies
    input := "test"
    expected := "expected result"

    // Act - Execute the function being tested
    result := FunctionName(input)

    // Assert - Verify the results
    assert.Equal(t, expected, result)
}
```

### Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "valid", false},
        {"invalid input", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Using Test Fixtures

```go
func TestWithFixtures(t *testing.T) {
    fixtures := testutil.NewFixtureBuilder()

    // Create test workflow
    workflow := fixtures.Workflow(func(w *models.Workflow) {
        w.Name = "Custom Name"
        w.Enabled = false
    })

    // Use in test
    assert.Equal(t, "Custom Name", workflow.Name)
}
```

### Using Mocks

```go
func TestServiceWithMock(t *testing.T) {
    mockRepo := mocks.NewSimpleWorkflowRepository()
    service := NewWorkflowService(mockRepo)

    workflow := &models.Workflow{Name: "Test"}
    err := service.CreateWorkflow(context.Background(), workflow)

    assert.NoError(t, err)
}
```

### Testing HTTP Handlers

```go
func TestHandler(t *testing.T) {
    handler := NewWorkflowHandler(service)

    req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
    w := httptest.NewRecorder()

    handler.List(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

## Test Coverage

### Generate Coverage Report

```bash
# Run tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage summary
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Coverage Script

```bash
# Run coverage script
bash scripts/test-coverage.sh

# With integration tests
INTEGRATION_TESTS=1 bash scripts/test-coverage.sh

# With E2E tests
E2E_TESTS=1 bash scripts/test-coverage.sh

# Set coverage threshold
COVERAGE_THRESHOLD=70 bash scripts/test-coverage.sh

# Open report automatically
OPEN_REPORT=1 bash scripts/test-coverage.sh
```

### Coverage Targets

| Component | Target | Current |
|-----------|--------|---------|
| Overall | > 70% | TBD |
| Models | > 90% | TBD |
| Handlers | > 80% | TBD |
| Services | > 80% | TBD |
| Repositories | > 75% | TBD |

## CI/CD Integration

### GitHub Actions

Tests run automatically on:
- Push to `main` or `develop`
- Pull requests
- Manual workflow dispatch

**Workflow:** `.github/workflows/test.yml`

**Jobs:**
1. Unit Tests
2. Integration Tests
3. E2E Tests
4. Lint
5. Coverage Report

### Pre-commit Hooks

```bash
# Install pre-commit hooks
cp scripts/pre-commit.sh .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

# Runs on every commit:
# - go fmt
# - go vet
# - unit tests
```

### Code Coverage Badge

Add to README.md:

```markdown
[![codecov](https://codecov.io/gh/yourorg/intelligent-workflows/branch/main/graph/badge.svg)](https://codecov.io/gh/yourorg/intelligent-workflows)
```

## Best Practices

### General

1. **Test Behavior, Not Implementation** - Focus on what the code does, not how
2. **One Assertion Per Test** - Keep tests focused and simple
3. **Descriptive Names** - Test names should describe what is being tested
4. **AAA Pattern** - Arrange, Act, Assert
5. **Independent Tests** - Tests should not depend on each other
6. **Fast Tests** - Keep unit tests under 10ms

### Unit Tests

```go
// ✅ Good
func TestWorkflow_IsEnabled_ReturnsTrueWhenEnabled(t *testing.T) {
    workflow := &Workflow{Enabled: true}
    assert.True(t, workflow.IsEnabled())
}

// ❌ Bad
func TestWorkflow(t *testing.T) {
    w := &Workflow{Enabled: true}
    if !w.IsEnabled() {
        t.Fatal("should be enabled")
    }
}
```

### Integration Tests

```go
// ✅ Good - Clean database before test
func TestRepository(t *testing.T) {
    suite := SetupSuite(t)
    defer TeardownSuite(t)
    suite.ResetDatabase(t)

    // Test...
}

// ❌ Bad - No cleanup
func TestRepository(t *testing.T) {
    // Test...
}
```

### Mocking

```go
// ✅ Good - Mock external dependencies only
func TestService(t *testing.T) {
    mockRepo := mocks.NewSimpleWorkflowRepository()
    service := NewService(mockRepo)
    // ...
}

// ❌ Bad - Mock everything
func TestService(t *testing.T) {
    mockRepo := mocks.NewRepo()
    mockLogger := mocks.NewLogger()
    mockConfig := mocks.NewConfig()
    // Too many mocks!
}
```

### Test Data

```go
// ✅ Good - Use fixtures
workflow := fixtures.Workflow(func(w *Workflow) {
    w.Name = "Custom"
})

// ❌ Bad - Hardcoded test data everywhere
workflow := &Workflow{
    ID: uuid.New(),
    WorkflowID: "test",
    Version: "1.0.0",
    Name: "Test",
    // 50 more lines...
}
```

## Troubleshooting

### Tests Failing Locally But Passing in CI

```bash
# Run tests exactly as CI does
docker-compose up -d postgres redis
export INTEGRATION_TESTS=1
export E2E_TESTS=1
go test ./... -v
```

### Database Connection Errors

```bash
# Check if PostgreSQL is running
docker-compose ps postgres

# Check logs
docker-compose logs postgres

# Restart services
docker-compose down
docker-compose up -d
```

### Race Condition Errors

```bash
# Run with race detector
go test ./... -race

# Fix race conditions by:
# 1. Using proper locking (sync.Mutex)
# 2. Using channels for synchronization
# 3. Avoiding shared state
```

### Slow Tests

```bash
# Identify slow tests
go test ./... -v | grep -A 1 "PASS"

# Run only fast tests
go test ./... -short

# Parallelize tests
go test ./... -parallel 8
```

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testing Best Practices](https://golang.org/doc/effective_go#testing)
