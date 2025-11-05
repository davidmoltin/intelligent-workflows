#!/bin/bash

# Load test using Vegeta
# Install: go install github.com/tsenart/vegeta@latest

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
DURATION="${DURATION:-30s}"
RATE="${RATE:-50}"

echo "====================================="
echo "Vegeta Load Test"
echo "====================================="
echo "Base URL: $BASE_URL"
echo "Duration: $DURATION"
echo "Rate: $RATE req/s"
echo "====================================="

# Create targets file
cat > /tmp/vegeta-targets.txt <<EOF
GET ${BASE_URL}/health
GET ${BASE_URL}/api/v1/workflows
EOF

# Run attack
echo "Running load test..."
vegeta attack -duration=$DURATION -rate=$RATE -targets=/tmp/vegeta-targets.txt | \
    tee /tmp/vegeta-results.bin | \
    vegeta report

# Generate plots
echo ""
echo "Generating plots..."

# Latency plot
vegeta plot -title "Latency over time" /tmp/vegeta-results.bin > /tmp/vegeta-latency.html
echo "Latency plot: /tmp/vegeta-latency.html"

# Histogram
vegeta report -type='hist[0,10ms,20ms,30ms,40ms,50ms,100ms,200ms,500ms,1s,2s]' /tmp/vegeta-results.bin

# Cleanup
rm /tmp/vegeta-targets.txt /tmp/vegeta-results.bin

echo ""
echo "====================================="
echo "Load test completed!"
echo "Open /tmp/vegeta-latency.html to view results"
echo "====================================="
