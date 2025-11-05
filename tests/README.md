# Intelligent Workflows - Test Suite

## Overview

This directory contains the comprehensive test suite for the Intelligent Workflows platform, including E2E tests, integration tests, performance benchmarks, load tests, and security tests.

## Quick Start

```bash
# Run all unit tests
go test ./... -short

# Run E2E tests
E2E_TESTS=1 go test ./tests/e2e/... -v

# Run integration tests
INTEGRATION_TESTS=1 go test ./tests/integration/... -v

# Run security tests
SECURITY_TESTS=1 go test ./tests/security/... -v

# Run benchmarks
go test ./tests/benchmark/... -bench=. -benchmem

# Run load tests
k6 run tests/load/k6-workflow-load.js
```

## Test Structure

### `/e2e` - End-to-End Tests

Full API testing with real HTTP requests.

**Test Files:**
- `workflow_api_test.go` - Basic workflow CRUD operations
- `auth_test.go` - Authentication and authorization flows
- `execution_test.go` - Workflow execution and event handling
- `approval_test.go` - Approval workflow testing
- `server.go` - Test server setup

**Run:**
```bash
E2E_TESTS=1 go test ./tests/e2e/... -v
```

### `/integration` - Integration Tests

Tests with real PostgreSQL and Redis instances.

**Test Files:**
- `suite_test.go` - Test suite setup
- `workflow_repository_test.go` - Database operations
- `approval_service_test.go` - Service layer integration

**Requirements:**
- PostgreSQL database
- Redis instance

**Run:**
```bash
INTEGRATION_TESTS=1 go test ./tests/integration/... -v
```

### `/benchmark` - Performance Benchmarks

Go benchmark tests for critical paths.

**Test Files:**
- `workflow_benchmark_test.go` - Workflow execution benchmarks
- `api_benchmark_test.go` - API endpoint benchmarks

**Run:**
```bash
# All benchmarks
go test ./tests/benchmark/... -bench=.

# With memory stats
go test ./tests/benchmark/... -bench=. -benchmem

# Specific benchmark
go test ./tests/benchmark/... -bench=BenchmarkWorkflowExecutor_SimpleWorkflow
```

### `/load` - Load Tests (K6)

Load and stress testing with K6.

**Test Files:**
- `k6-workflow-load.js` - Workflow API load testing
- `k6-auth-load.js` - Authentication load testing
- `k6-approval-load.js` - Approval workflow load testing

**Run:**
```bash
# Install K6 first
brew install k6  # macOS
# or
sudo apt-get install k6  # Linux

# Run load tests
k6 run tests/load/k6-workflow-load.js
k6 run tests/load/k6-auth-load.js
k6 run tests/load/k6-approval-load.js

# Against specific environment
BASE_URL=https://staging.example.com k6 run tests/load/k6-workflow-load.js
```

### `/security` - Security Tests

Security and penetration testing.

**Test Files:**
- `auth_security_test.go` - Authentication security
- `rbac_security_test.go` - Authorization and RBAC
- `server.go` - Security test server setup

**Run:**
```bash
SECURITY_TESTS=1 go test ./tests/security/... -v
```

## Test Coverage

### Current Status
- **Target**: 60%+ overall coverage
- **Critical Paths**: 80%+ coverage

### Generate Coverage Report
```bash
# Generate coverage
go test ./... -coverprofile=coverage.out

# View HTML report
go tool cover -html=coverage.out

# View by package
go tool cover -func=coverage.out
```

## Test Scenarios

### E2E Test Scenarios

#### Authentication
- ✅ User registration with validation
- ✅ User login with credentials
- ✅ Token refresh mechanism
- ✅ API key creation and usage
- ✅ Protected endpoint access
- ✅ Invalid credentials handling

#### Workflow Operations
- ✅ Create workflow
- ✅ Get workflow by ID
- ✅ List workflows
- ✅ Update workflow
- ✅ Delete workflow

#### Workflow Execution
- ✅ Event-triggered execution
- ✅ Conditional branching
- ✅ Multiple step execution
- ✅ Execution history tracking
- ✅ Execution filtering and listing

#### Approval Workflows
- ✅ Create approval request
- ✅ Approve request
- ✅ Reject request
- ✅ List pending approvals
- ✅ Workflow with wait steps
- ✅ Expired approval handling

### Performance Benchmarks

#### Workflow Engine
- Simple workflow execution
- Conditional workflow execution
- Complex multi-step workflows
- Concurrent execution

#### Evaluator
- Simple condition evaluation
- Complex condition evaluation
- Nested field extraction
- Regex matching

#### Context Builder
- Simple context building
- External data enrichment

#### API Endpoints
- Workflow CRUD operations
- Event triggering
- Execution queries
- Authentication

### Load Test Scenarios

#### Workflow Load Test
- Ramp: 10 → 50 → 100 users
- Operations: Create, Read, List, Trigger
- Target: p95 < 500ms

#### Auth Load Test
- Login/Registration load
- Token refresh load
- API key usage
- Target: p95 < 300ms

#### Approval Load Test
- Approval creation
- Approval processing
- Approval queries
- Target: p95 < 400ms

### Security Test Scenarios

#### Authentication Security
- ✅ Brute force protection
- ✅ Weak password rejection
- ✅ JWT token validation
- ✅ Expired token rejection
- ✅ Password hashing security
- ✅ API key secrecy
- ✅ SQL injection protection
- ✅ XSS protection
- ✅ Rate limiting
- ✅ CORS configuration
- ✅ Secure headers

#### Authorization Security
- ✅ RBAC enforcement
- ✅ Resource ownership
- ✅ Privilege escalation prevention
- ✅ API scope enforcement
- ✅ Cross-user access prevention

## CI/CD Integration

Tests are automatically run in GitHub Actions:

```yaml
# Unit Tests
- Always run on PR
- Fast feedback (<2 min)

# Integration Tests
- Run on PR to main/develop
- Requires database setup

# E2E Tests
- Run on PR to main/develop
- Full API testing

# Security Tests
- Run on all PRs
- Nightly security scans

# Load Tests
- Run before deployments
- Performance regression detection
```

## Performance Targets

| Operation | Target (p95) | Target (p99) |
|-----------|--------------|--------------|
| Workflow Creation | 500ms | 800ms |
| Event Trigger | 200ms | 400ms |
| Login | 200ms | 300ms |
| Token Refresh | 100ms | 200ms |
| Approval Creation | 300ms | 500ms |

## Troubleshooting

### E2E Tests Not Running
```bash
# Ensure E2E_TESTS env variable is set
E2E_TESTS=1 go test ./tests/e2e/... -v
```

### Integration Tests Failing
```bash
# Start dependencies
docker-compose -f docker-compose.test.yml up -d

# Check database connection
psql -h localhost -U postgres -d workflows_test

# Run tests
INTEGRATION_TESTS=1 go test ./tests/integration/... -v
```

### K6 Not Found
```bash
# Install K6
brew install k6  # macOS
sudo apt-get install k6  # Linux
```

### Security Tests Failing
```bash
# Run security tests with verbose output
SECURITY_TESTS=1 go test ./tests/security/... -v

# Run gosec separately
gosec ./...
```

## Best Practices

1. **Run tests locally** before pushing
2. **Keep tests fast** - use `-short` for quick checks
3. **Clean up test data** - reset state between tests
4. **Use descriptive names** - tests should document behavior
5. **Mock external services** - in unit tests
6. **Test error cases** - not just happy paths
7. **Parallelize when safe** - use `t.Parallel()`
8. **Monitor performance** - track benchmark trends

## Resources

- [Testing Guide](../docs/TESTING_GUIDE.md)
- [Security Testing](../docs/SECURITY_TESTING.md)
- [Go Testing Docs](https://golang.org/pkg/testing/)
- [K6 Documentation](https://k6.io/docs/)

## Contributing

When adding new features:

1. Add unit tests (minimum 60% coverage)
2. Add integration tests for database operations
3. Add E2E tests for API endpoints
4. Add security tests for new auth/authz logic
5. Update benchmarks if performance-critical
6. Update this README with new test scenarios

## Support

For questions or issues with tests:
- Check the [Testing Guide](../docs/TESTING_GUIDE.md)
- Open an issue on GitHub
- Contact the team
