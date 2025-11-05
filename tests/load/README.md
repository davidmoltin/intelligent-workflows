## Load Testing

This directory contains load testing scripts for the Intelligent Workflows service.

## Overview

Load tests verify that the system can handle expected traffic volumes and identify performance bottlenecks. They measure:
- Throughput (requests per second)
- Latency (response times)
- Error rates
- Resource utilization

## Tools

We provide scripts for multiple load testing tools:

### 1. K6 (Recommended)
Modern load testing tool with JavaScript-based tests.

**Install:**
```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6
```

**Run:**
```bash
# Basic run
k6 run k6-workflow-load.js

# With custom URL
BASE_URL=http://production.example.com k6 run k6-workflow-load.js

# With custom duration
k6 run --duration 5m k6-workflow-load.js

# Output to InfluxDB for Grafana
k6 run --out influxdb=http://localhost:8086/k6 k6-workflow-load.js
```

### 2. Hey
Simple but effective load testing tool.

**Install:**
```bash
go install github.com/rakyll/hey@latest
```

**Run:**
```bash
# Basic GET request
hey -n 1000 -c 10 http://localhost:8080/api/v1/workflows

# POST request
hey -n 100 -c 10 \
    -m POST \
    -H "Content-Type: application/json" \
    -D workflow.json \
    http://localhost:8080/api/v1/workflows

# Run simple load test script
bash simple-load.sh
```

### 3. Vegeta
Powerful HTTP load testing tool.

**Install:**
```bash
go install github.com/tsenart/vegeta@latest
```

**Run:**
```bash
# Run vegeta load test
bash vegeta-load.sh

# Custom rate and duration
RATE=100 DURATION=1m bash vegeta-load.sh
```

### 4. Apache Bench (ab)
Classic tool included with Apache.

**Install:**
```bash
sudo apt-get install apache2-utils  # Linux
# macOS comes with ab pre-installed
```

**Run:**
```bash
# 1000 requests, 10 concurrent
ab -n 1000 -c 10 http://localhost:8080/health

# With keep-alive
ab -n 1000 -c 10 -k http://localhost:8080/health
```

## Load Test Scenarios

### Scenario 1: Baseline Performance
Verify basic performance under normal load.

```bash
# K6
k6 run --vus 10 --duration 1m k6-workflow-load.js

# Hey
hey -n 1000 -c 10 http://localhost:8080/api/v1/workflows
```

**Expected Results:**
- 95th percentile latency < 500ms
- Error rate < 1%
- Throughput > 100 req/s

### Scenario 2: Peak Load
Test system behavior under peak traffic.

```bash
# K6 with higher load
k6 run --vus 100 --duration 2m k6-workflow-load.js

# Vegeta with high rate
RATE=200 DURATION=2m bash vegeta-load.sh
```

**Expected Results:**
- 95th percentile latency < 1s
- Error rate < 5%
- System remains stable

### Scenario 3: Spike Test
Test response to sudden traffic spike.

```bash
# K6 spike test (built into k6-workflow-load.js)
k6 run k6-workflow-load.js
```

**Expected Results:**
- System handles spike without crashes
- Latency increases but stabilizes
- Error rate temporarily increases but recovers

### Scenario 4: Soak Test
Long-running test to identify memory leaks.

```bash
# K6 with long duration
k6 run --vus 20 --duration 1h k6-workflow-load.js
```

**Expected Results:**
- Memory usage remains stable
- No degradation over time
- No resource leaks

## Running Load Tests

### Prerequisites

1. **Start the service:**
   ```bash
   docker-compose up -d
   # Or
   go run cmd/api/main.go
   ```

2. **Verify health:**
   ```bash
   curl http://localhost:8080/health
   ```

### Run All Tests

```bash
# Run K6 tests
k6 run k6-workflow-load.js

# Run simple load tests
bash simple-load.sh

# Run vegeta tests
bash vegeta-load.sh
```

### Custom Configuration

```bash
# Custom base URL
export BASE_URL=http://staging.example.com

# Custom load parameters
export CONCURRENT=50
export REQUESTS=10000
export DURATION=5m
export RATE=100

# Run tests
bash simple-load.sh
```

## Analyzing Results

### K6 Results

K6 provides detailed metrics:

```
checks.........................: 100.00% ✓ 45000      ✗ 0
data_received..................: 6.7 MB  112 kB/s
data_sent......................: 4.1 MB  68 kB/s
http_req_blocked...............: avg=1.46ms   min=1µs     med=3µs      max=1.01s    p(90)=5µs      p(95)=6µs
http_req_connecting............: avg=528µs    min=0s      med=0s       max=345.95ms p(90)=0s       p(95)=0s
http_req_duration..............: avg=221.44ms min=5.69ms  med=213.04ms max=1.23s    p(90)=356.47ms p(95)=410.13ms
http_req_receiving.............: avg=112.87µs min=13µs    med=85µs     max=17.32ms  p(90)=181µs    p(95)=234µs
http_req_sending...............: avg=29.45µs  min=5µs     med=20µs     max=9.24ms   p(90)=48µs     p(95)=68µs
http_req_waiting...............: avg=221.3ms  min=5.64ms  med=212.93ms max=1.23s    p(90)=356.34ms p(95)=409.95ms
http_reqs......................: 15000   250/s
iteration_duration.............: avg=1.23s    min=1.02s   med=1.22s    max=3.08s    p(90)=1.39s    p(95)=1.47s
iterations.....................: 15000   250/s
vus............................: 100     min=0        max=100
vus_max........................: 100     min=100      max=100
```

Key metrics to monitor:
- `http_req_duration` - Response time (p95 < 500ms)
- `http_req_failed` - Error rate (< 1%)
- `http_reqs` - Throughput (requests/second)

### Hey Results

```
Summary:
  Total:        10.0123 secs
  Slowest:      0.5234 secs
  Fastest:      0.0123 secs
  Average:      0.0987 secs
  Requests/sec: 99.88

Response time histogram:
  0.012 [1]     |
  0.063 [234]   |■■■■■■■■■
  0.114 [456]   |■■■■■■■■■■■■■■■■■■
  0.165 [234]   |■■■■■■■■■
  0.216 [56]    |■■
```

### Performance Targets

| Metric | Target | Good | Needs Improvement |
|--------|--------|------|-------------------|
| Throughput | > 100 req/s | > 500 req/s | < 50 req/s |
| P95 Latency | < 500ms | < 200ms | > 1s |
| P99 Latency | < 1s | < 500ms | > 2s |
| Error Rate | < 1% | < 0.1% | > 5% |
| CPU Usage | < 70% | < 50% | > 90% |
| Memory | Stable | Stable | Growing |

## Troubleshooting

### High Error Rates

```bash
# Check server logs
docker-compose logs api

# Check database connections
docker-compose logs postgres

# Reduce load
export RATE=10
export CONCURRENT=5
```

### High Latency

Common causes:
1. Database slow queries
2. Network issues
3. Resource constraints
4. Lock contention

Solutions:
```bash
# Check database performance
docker-compose exec postgres psql -U postgres -d workflows -c "
    SELECT query, calls, mean_exec_time
    FROM pg_stat_statements
    ORDER BY mean_exec_time DESC
    LIMIT 10;
"

# Check resource usage
docker stats

# Scale horizontally
docker-compose up --scale api=3
```

### Memory Leaks

If memory grows over time:
```bash
# Run soak test
k6 run --vus 20 --duration 1h k6-workflow-load.js

# Monitor memory
while true; do
    docker stats --no-stream api | grep api
    sleep 60
done

# Generate heap profile
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

## CI/CD Integration

Add to `.github/workflows/load-test.yml`:

```yaml
name: Load Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Run daily at 2 AM
  workflow_dispatch:

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Start services
        run: docker-compose up -d

      - name: Wait for health
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:8080/health; do sleep 1; done'

      - name: Install k6
        run: |
          sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6

      - name: Run load test
        run: k6 run tests/load/k6-workflow-load.js

      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: /tmp/k6-*
```

## Best Practices

1. **Start Small**: Begin with low load and gradually increase
2. **Monitor**: Watch CPU, memory, database during tests
3. **Isolate**: Run load tests against isolated environments
4. **Warm Up**: Allow service to warm up before measuring
5. **Realistic Data**: Use realistic payloads and scenarios
6. **Consistent**: Run tests from same location/environment
7. **Trend**: Track performance over time
8. **Alert**: Set up alerts for performance regressions

## Resources

- [K6 Documentation](https://k6.io/docs/)
- [Vegeta Documentation](https://github.com/tsenart/vegeta)
- [Hey Documentation](https://github.com/rakyll/hey)
- [Load Testing Best Practices](https://k6.io/docs/testing-guides/test-types/)
