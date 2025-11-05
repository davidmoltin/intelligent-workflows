#!/bin/bash

# Simple load test using Apache Bench (ab)
# Install: sudo apt-get install apache2-utils

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
CONCURRENT="${CONCURRENT:-10}"
REQUESTS="${REQUESTS:-1000}"

echo "====================================="
echo "Simple Load Test"
echo "====================================="
echo "Base URL: $BASE_URL"
echo "Concurrent: $CONCURRENT"
echo "Requests: $REQUESTS"
echo "====================================="

# Test 1: Health endpoint
echo ""
echo "Test 1: Health Check"
ab -n $REQUESTS -c $CONCURRENT "$BASE_URL/health"

# Test 2: List workflows
echo ""
echo "Test 2: List Workflows"
ab -n $REQUESTS -c $CONCURRENT "$BASE_URL/api/v1/workflows"

# Test 3: Create workflow (using hey if available)
if command -v hey &> /dev/null; then
    echo ""
    echo "Test 3: Create Workflows (using hey)"

    # Create temp file with workflow JSON
    cat > /tmp/workflow.json <<EOF
{
  "workflow_id": "load-test-workflow",
  "version": "1.0.0",
  "name": "Load Test Workflow",
  "definition": {
    "trigger": {
      "type": "event",
      "event": "order.created"
    },
    "steps": [
      {
        "id": "step1",
        "type": "action",
        "action": {
          "action": "allow"
        }
      }
    ]
  }
}
EOF

    hey -n 100 -c 10 \
        -m POST \
        -H "Content-Type: application/json" \
        -D /tmp/workflow.json \
        "$BASE_URL/api/v1/workflows"

    rm /tmp/workflow.json
else
    echo ""
    echo "Test 3: Skipped (hey not installed)"
    echo "Install with: go install github.com/rakyll/hey@latest"
fi

echo ""
echo "====================================="
echo "Load test completed!"
echo "====================================="
