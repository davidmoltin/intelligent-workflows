#!/bin/bash

# Run all tests (unit, integration, E2E)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}====================================="
echo "Running All Tests"
echo -e "=====================================${NC}"

# Run unit tests
echo -e "\n${YELLOW}1. Unit Tests${NC}"
go test ./internal/... ./pkg/... -v -race -timeout 30s

# Run integration tests (if enabled)
if [ "$SKIP_INTEGRATION" != "1" ]; then
    echo -e "\n${YELLOW}2. Integration Tests${NC}"
    export INTEGRATION_TESTS=1
    go test ./tests/integration/... -v -timeout 2m
else
    echo -e "\n${YELLOW}2. Integration Tests (SKIPPED)${NC}"
fi

# Run E2E tests (if enabled)
if [ "$SKIP_E2E" != "1" ]; then
    echo -e "\n${YELLOW}3. E2E Tests${NC}"
    export E2E_TESTS=1
    go test ./tests/e2e/... -v -timeout 5m
else
    echo -e "\n${YELLOW}3. E2E Tests (SKIPPED)${NC}"
fi

echo -e "\n${GREEN}====================================="
echo "All Tests Passed! ✓"
echo -e "=====================================${NC}"
