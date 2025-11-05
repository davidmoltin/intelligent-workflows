## Testing Guide

## Overview

This document provides comprehensive guidance on testing the Intelligent Workflows platform, including unit tests, integration tests, E2E tests, performance tests, and security tests.

## Table of Contents

1. [Test Structure](#test-structure)
2. [Running Tests](#running-tests)
3. [Unit Tests](#unit-tests)
4. [Integration Tests](#integration-tests)
5. [End-to-End Tests](#end-to-end-tests)
6. [Performance Tests](#performance-tests)
7. [Security Tests](#security-tests)
8. [Test Coverage](#test-coverage)
9. [Best Practices](#best-practices)

## Test Structure

```
intelligent-workflows/
├── internal/
│   ├── engine/
│   │   ├── executor_test.go       # Unit tests
│   │   ├── evaluator_test.go
│   │   └── context_test.go
│   ├── services/
│   │   └── *_test.go
│   └── models/
│       └── *_test.go
├── tests/
│   ├── integration/               # Integration tests
│   │   ├── suite_test.go
│   │   └── *_test.go
│   ├── e2e/                       # End-to-end tests
│   │   ├── workflow_api_test.go
│   │   ├── auth_test.go
│   │   ├── execution_test.go
│   │   ├── approval_test.go
│   │   └── server.go
│   ├── benchmark/                 # Performance benchmarks
│   │   ├── workflow_benchmark_test.go
│   │   └── api_benchmark_test.go
│   ├── load/                      # Load tests (K6)
│   │   ├── k6-workflow-load.js
│   │   ├── k6-auth-load.js
│   │   └── k6-approval-load.js
│   └── security/                  # Security tests
│       ├── auth_security_test.go
│       ├── rbac_security_test.go
│       └── server.go
└── pkg/
    └── testutil/                  # Test utilities
        ├── assertions.go
        ├── database.go
        ├── fixtures.go
        └── context.go
```

## Running Tests

### All Tests

```bash
# Run all tests
make test

# Run all tests with coverage
make test-coverage

# Run all tests with race detection
go test -race ./...
```

### Unit Tests

```bash
# Run unit tests only
go test ./internal/... ./pkg/... -short

# Run specific package
go test ./internal/engine -v

# Run specific test
go test ./internal/engine -run TestExecutor_Execute -v
```

### Integration Tests

```bash
# Run integration tests (requires PostgreSQL and Redis)
INTEGRATION_TESTS=1 go test ./tests/integration/... -v

# Run with Docker Compose
docker-compose -f docker-compose.test.yml up -d
INTEGRATION_TESTS=1 go test ./tests/integration/... -v
docker-compose -f docker-compose.test.yml down
```

### End-to-End Tests

```bash
# Run E2E tests
E2E_TESTS=1 go test ./tests/e2e/... -v

# Run specific E2E test suite
E2E_TESTS=1 go test ./tests/e2e/... -run TestAuth -v
E2E_TESTS=1 go test ./tests/e2e/... -run TestWorkflowExecution -v
E2E_TESTS=1 go test ./tests/e2e/... -run TestApproval -v
```

### Performance Tests

```bash
# Run Go benchmarks
go test ./tests/benchmark/... -bench=. -benchmem

# Run specific benchmark
go test ./tests/benchmark/... -bench=BenchmarkWorkflowExecutor_SimpleWorkflow -benchtime=10s

# Generate CPU profile
go test ./tests/benchmark/... -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

### Load Tests (K6)

```bash
# Install K6
brew install k6  # macOS
# or
sudo apt-get install k6  # Linux

# Run workflow load test
k6 run tests/load/k6-workflow-load.js

# Run auth load test
k6 run tests/load/k6-auth-load.js

# Run approval load test
k6 run tests/load/k6-approval-load.js

# Run against specific URL
BASE_URL=http://localhost:8080 k6 run tests/load/k6-workflow-load.js

# Run with output to InfluxDB (for Grafana visualization)
k6 run --out influxdb=http://localhost:8086/k6 tests/load/k6-workflow-load.js
```

### Security Tests

```bash
# Run security tests
SECURITY_TESTS=1 go test ./tests/security/... -v

# Run gosec security scanner
gosec ./...

# Run Trivy container scanner
trivy image intelligent-workflows:latest
```

## Unit Tests

### Purpose
- Test individual functions and methods in isolation
- Fast execution
- No external dependencies

### Example

```go
func TestEvaluator_Evaluate_SimpleCondition(t *testing.T) {
    evaluator := engine.NewEvaluator()

    condition := &models.Condition{
        Field:    "amount",
        Operator: "gt",
        Value:    100.0,
    }

    data := map[string]interface{}{
        "amount": 150.0,
    }

    result, err := evaluator.Evaluate(condition, data)
    require.NoError(t, err)
    assert.True(t, result)
}
```

### Best Practices

1. **Use table-driven tests** for multiple scenarios
2. **Mock external dependencies** using interfaces
3. **Test error cases** as well as success cases
4. **Keep tests focused** on a single behavior
5. **Use descriptive test names**

## Integration Tests

### Purpose
- Test component interactions
- Use real databases and services
- Verify data persistence and retrieval

### Setup

Integration tests require:
- PostgreSQL database
- Redis instance

```bash
# Start dependencies
docker-compose -f docker-compose.test.yml up -d

# Run tests
INTEGRATION_TESTS=1 go test ./tests/integration/... -v
```

### Example

```go
func TestWorkflowRepository_Create(t *testing.T) {
    suite := setupIntegrationTestSuite(t)
    defer suite.Teardown()

    workflow := &models.Workflow{
        WorkflowID: "test-workflow",
        Version:    "1.0.0",
        Name:       "Test Workflow",
        // ...
    }

    err := suite.WorkflowRepo.Create(context.Background(), workflow)
    require.NoError(t, err)

    // Verify workflow was created
    retrieved, err := suite.WorkflowRepo.GetByID(context.Background(), workflow.ID)
    require.NoError(t, err)
    assert.Equal(t, workflow.WorkflowID, retrieved.WorkflowID)
}
```

## End-to-End Tests

### Purpose
- Test complete user workflows
- Verify API endpoints work correctly
- Test real-world scenarios

### Test Scenarios

1. **Authentication Flow** (`auth_test.go`)
   - User registration
   - Login
   - Token refresh
   - API key creation
   - Protected endpoint access

2. **Workflow Execution** (`execution_test.go`)
   - Event-triggered workflows
   - Conditional branching
   - Execution tracking
   - Execution filtering

3. **Approval Workflow** (`approval_test.go`)
   - Approval creation
   - Approval decision (approve/reject)
   - Pending approvals listing
   - Wait steps
   - Expired approvals

### Example

```go
func TestWorkflowExecution_EventTriggered(t *testing.T) {
    server := NewTestServer(t)
    server.Start()
    defer server.Stop()

    // Create workflow
    workflow := createTestWorkflow()
    resp := createWorkflow(server, workflow)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    // Trigger event
    event := createTestEvent()
    resp = triggerEvent(server, event)
    assert.Equal(t, http.StatusAccepted, resp.StatusCode)

    // Verify execution
    executions := listExecutions(server)
    assert.GreaterOrEqual(t, len(executions), 1)
}
```

## Performance Tests

### Go Benchmarks

Run benchmarks to measure performance of critical code paths:

```bash
# Run all benchmarks
go test ./tests/benchmark/... -bench=.

# Run with memory allocation stats
go test ./tests/benchmark/... -bench=. -benchmem

# Compare benchmarks
go test ./tests/benchmark/... -bench=. -count=5 > old.txt
# Make changes
go test ./tests/benchmark/... -bench=. -count=5 > new.txt
benchcmp old.txt new.txt
```

### Load Tests (K6)

K6 load tests simulate real-world traffic patterns:

#### Workflow Load Test
- Tests workflow CRUD operations
- Tests event triggering
- Simulates 10-100 concurrent users
- Target: 95th percentile < 500ms

#### Auth Load Test
- Tests login/registration
- Tests token refresh
- Tests API key usage
- Target: 95th percentile < 300ms

#### Approval Load Test
- Tests approval creation
- Tests approval processing
- Tests approval queries
- Target: 95th percentile < 400ms

### Performance Targets

| Operation | Target (p95) | Target (p99) |
|-----------|--------------|--------------|
| Workflow Creation | 500ms | 800ms |
| Event Trigger | 200ms | 400ms |
| Login | 200ms | 300ms |
| Token Refresh | 100ms | 200ms |
| Approval Creation | 300ms | 500ms |

## Security Tests

See [SECURITY_TESTING.md](./SECURITY_TESTING.md) for detailed security testing documentation.

### Quick Start

```bash
# Run all security tests
SECURITY_TESTS=1 go test ./tests/security/... -v

# Run security scanners
gosec ./...
trivy image intelligent-workflows:latest
```

## Test Coverage

### Current Coverage

- **Target**: 60%+ overall coverage
- **Critical Paths**: 80%+ coverage required

### Generate Coverage Report

```bash
# Generate coverage
make test-coverage

# View HTML report
go tool cover -html=coverage.out

# View coverage by package
go tool cover -func=coverage.out
```

### Coverage Requirements

- **Engine Package**: 80%+ (critical business logic)
- **API Handlers**: 70%+
- **Services**: 70%+
- **Repositories**: 60%+
- **Models**: 50%+

## Best Practices

### General

1. **Write tests first** (TDD approach when possible)
2. **Keep tests independent** - no test should depend on another
3. **Use descriptive names** - test names should describe what is being tested
4. **Test one thing** - each test should verify one behavior
5. **Use assertions wisely** - prefer require for critical checks, assert for others

### Test Data

1. **Use fixtures** for complex test data
2. **Generate random data** for uniqueness
3. **Clean up after tests** - reset database state
4. **Use meaningful test data** - avoid "test1", "test2"

### Mocking

1. **Mock external dependencies** (APIs, databases in unit tests)
2. **Use interfaces** for easier mocking
3. **Verify mock calls** when behavior is important
4. **Don't over-mock** - use real implementations when simple

### Performance

1. **Use -short flag** for quick tests during development
2. **Run full suite** in CI/CD
3. **Profile slow tests** to identify bottlenecks
4. **Parallelize tests** when safe

### CI/CD Integration

```yaml
# .github/workflows/ci.yml
jobs:
  test:
    steps:
      - name: Unit Tests
        run: go test ./... -short -race -coverprofile=coverage.out

      - name: Integration Tests
        env:
          INTEGRATION_TESTS: 1
        run: go test ./tests/integration/... -v

      - name: E2E Tests
        env:
          E2E_TESTS: 1
        run: go test ./tests/e2e/... -v

      - name: Security Tests
        env:
          SECURITY_TESTS: 1
        run: go test ./tests/security/... -v

      - name: Security Scan
        run: gosec ./...
```

## Troubleshooting

### Tests Hanging

```bash
# Run with timeout
go test ./... -timeout 30s
```

### Database Connection Issues

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check connection
psql -h localhost -U postgres -d workflows_test
```

### Redis Connection Issues

```bash
# Check Redis is running
docker ps | grep redis

# Test connection
redis-cli ping
```

### Race Conditions

```bash
# Run with race detector
go test -race ./...

# Fix data races before merging
```

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [K6 Documentation](https://k6.io/docs/)
- [Testing Best Practices](https://github.com/golang/go/wiki/TestComments)
