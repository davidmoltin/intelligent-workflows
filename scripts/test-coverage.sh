#!/bin/bash

# Test coverage script
# Generates test coverage reports for the project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}====================================="
echo "Test Coverage Report"
echo -e "=====================================${NC}"

# Create coverage directory
mkdir -p coverage

# Run unit tests with coverage
echo -e "\n${YELLOW}Running unit tests...${NC}"
go test ./internal/... ./pkg/... -coverprofile=coverage/unit.out -covermode=atomic

# Run integration tests with coverage (if enabled)
if [ "$INTEGRATION_TESTS" = "1" ]; then
    echo -e "\n${YELLOW}Running integration tests...${NC}"
    go test ./tests/integration/... -coverprofile=coverage/integration.out -covermode=atomic
else
    echo -e "\n${YELLOW}Skipping integration tests (set INTEGRATION_TESTS=1 to enable)${NC}"
fi

# Run E2E tests with coverage (if enabled)
if [ "$E2E_TESTS" = "1" ]; then
    echo -e "\n${YELLOW}Running E2E tests...${NC}"
    go test ./tests/e2e/... -coverprofile=coverage/e2e.out -covermode=atomic
else
    echo -e "\n${YELLOW}Skipping E2E tests (set E2E_TESTS=1 to enable)${NC}"
fi

# Merge coverage files
echo -e "\n${YELLOW}Merging coverage reports...${NC}"
echo "mode: atomic" > coverage/coverage.out

# Combine all coverage files
for f in coverage/*.out; do
    if [ "$f" != "coverage/coverage.out" ]; then
        tail -n +2 "$f" >> coverage/coverage.out
    fi
done

# Generate coverage report
echo -e "\n${YELLOW}Coverage Summary:${NC}"
go tool cover -func=coverage/coverage.out | tail -n 1

# Generate HTML coverage report
echo -e "\n${YELLOW}Generating HTML report...${NC}"
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Calculate coverage percentage
COVERAGE=$(go tool cover -func=coverage/coverage.out | tail -n 1 | awk '{print $3}' | sed 's/%//')

echo -e "\n${GREEN}====================================="
echo "Coverage: ${COVERAGE}%"
echo "HTML Report: coverage/coverage.html"
echo -e "=====================================${NC}"

# Check coverage threshold
THRESHOLD=${COVERAGE_THRESHOLD:-50}
if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo -e "${RED}Coverage ${COVERAGE}% is below threshold ${THRESHOLD}%${NC}"
    exit 1
else
    echo -e "${GREEN}Coverage ${COVERAGE}% meets threshold ${THRESHOLD}%${NC}"
fi

# Open HTML report in browser (optional)
if [ "$OPEN_REPORT" = "1" ]; then
    if command -v xdg-open &> /dev/null; then
        xdg-open coverage/coverage.html
    elif command -v open &> /dev/null; then
        open coverage/coverage.html
    fi
fi
