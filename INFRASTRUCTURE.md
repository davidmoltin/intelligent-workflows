# Infrastructure & DevOps Guide

This document provides comprehensive information about the infrastructure setup, deployment procedures, and operational guidelines for the Intelligent Workflows system.

## Table of Contents

- [Overview](#overview)
- [Local Development](#local-development)
- [Docker Setup](#docker-setup)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Monitoring & Observability](#monitoring--observability)
- [CI/CD Pipeline](#cicd-pipeline)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

---

## Overview

The infrastructure stack consists of:

- **Application**: Go-based REST API
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **Monitoring**: Prometheus + Grafana
- **Container Orchestration**: Docker Compose (local) / Kubernetes (production)
- **CI/CD**: GitHub Actions

### Architecture Diagram

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌──────────────┐
│   Ingress   │────▶│  Workflows   │
│ (nginx/ALB) │     │     API      │
└─────────────┘     └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         ┌─────────┐  ┌──────┐   ┌──────────┐
         │PostgreSQL│  │Redis │   │Prometheus│
         └─────────┘  └──────┘   └──────────┘
                                       │
                                       ▼
                                  ┌─────────┐
                                  │ Grafana │
                                  └─────────┘
```

---

## Local Development

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Make
- kubectl (for Kubernetes operations)

### Quick Start

```bash
# Start the full stack with monitoring
make dev-full

# Or start just the API dependencies
make dev
```

### Available Make Commands

```bash
make help                  # Show all available commands

# Development
make dev                   # Start development environment
make dev-full              # Start full environment with monitoring

# Building
make build                 # Build API binary
make docker-build          # Build Docker image

# Testing
make test                  # Run tests
make test-coverage         # Run tests with coverage
make lint                  # Run linter

# Docker Operations
make docker-up             # Start all containers
make docker-down           # Stop all containers
make docker-restart        # Restart containers
make docker-rebuild        # Rebuild and restart
make docker-clean          # Clean all Docker resources

# Monitoring
make monitoring-up         # Start monitoring stack
make monitoring-down       # Stop monitoring stack
make metrics               # Open Prometheus
make dashboards            # Open Grafana

# Database
make db-shell              # Open PostgreSQL shell
make db-backup             # Backup database
make db-restore            # Restore database
make migrate-up            # Run migrations
make migrate-down          # Rollback migrations

# Health Checks
make health                # Check API health
make ready                 # Check API readiness

# CI/CD
make ci                    # Run CI pipeline locally
make security-scan         # Run security scan
```

---

## Docker Setup

### Docker Compose Services

The `docker-compose.yml` includes:

1. **postgres**: PostgreSQL database
2. **redis**: Redis cache
3. **api**: Workflows API server
4. **prometheus**: Metrics collection
5. **grafana**: Metrics visualization
6. **postgres-exporter**: PostgreSQL metrics exporter
7. **redis-exporter**: Redis metrics exporter

### Starting Services

```bash
# Start all services
docker-compose up -d

# Start specific services
docker-compose up -d postgres redis api

# View logs
docker-compose logs -f api

# Check status
docker-compose ps
```

### Service Endpoints

| Service           | Port | URL                        |
|-------------------|------|----------------------------|
| API               | 8080 | http://localhost:8080      |
| API Metrics       | 9090 | http://localhost:9090      |
| Prometheus        | 9091 | http://localhost:9091      |
| Grafana           | 3000 | http://localhost:3000      |
| PostgreSQL        | 5432 | localhost:5432             |
| Redis             | 6379 | localhost:6379             |

### Environment Variables

Configure via `.env` file or docker-compose.yml:

```bash
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=workflows
DB_SSL_MODE=disable

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# Server
SERVER_PORT=8080
METRICS_PORT=9090
LOG_LEVEL=info
LOG_FORMAT=json
APP_ENV=development
```

---

## Kubernetes Deployment

### Directory Structure

```
k8s/
├── base/                    # Base manifests
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   ├── hpa.yaml
│   ├── pdb.yaml
│   ├── servicemonitor.yaml
│   └── kustomization.yaml
└── overlays/               # Environment-specific configs
    ├── dev/
    ├── staging/
    └── production/
```

### Deployment Steps

#### 1. Validate Manifests

```bash
make k8s-validate
```

#### 2. Deploy to Environment

```bash
# Development
make k8s-deploy-dev

# Staging
make k8s-deploy-staging

# Production
make k8s-deploy-prod
```

#### 3. Verify Deployment

```bash
make k8s-status

# Or manually
kubectl get pods,svc,ingress -n intelligent-workflows
```

### Kustomize Overlays

Each environment has specific configurations:

**Development** (`k8s/overlays/dev/`):
- 1 replica
- Reduced resources (50m CPU, 64Mi RAM)
- Debug logging
- Development image tag

**Staging** (`k8s/overlays/staging/`):
- 2 replicas
- Standard resources (100m CPU, 128Mi RAM)
- Info logging
- Staging image tag

**Production** (`k8s/overlays/production/`):
- 3 replicas
- Enhanced resources (200m CPU, 256Mi RAM)
- Warning logging
- Latest/versioned image tag
- HPA enabled (3-10 replicas)
- PodDisruptionBudget configured

### Secrets Management

**IMPORTANT**: Never commit secrets to git!

#### Option 1: Manual Secrets

```bash
kubectl create secret generic workflows-api-secret \
  --from-literal=DB_PASSWORD=your-password \
  --from-literal=JWT_SECRET=your-jwt-secret \
  -n intelligent-workflows
```

#### Option 2: Sealed Secrets (Recommended)

```bash
# Install sealed-secrets controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.18.0/controller.yaml

# Create sealed secret
kubeseal --format yaml < secret.yaml > sealed-secret.yaml
kubectl apply -f sealed-secret.yaml
```

### Ingress Configuration

Update the domain in `k8s/base/ingress.yaml`:

```yaml
spec:
  tls:
    - hosts:
        - your-domain.com
        - api.your-domain.com
      secretName: workflows-api-tls
  rules:
    - host: your-domain.com
      # ...
```

### SSL/TLS Certificates

Using cert-manager:

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

---

## Monitoring & Observability

### Prometheus

**Local**: http://localhost:9091
**Queries**: Access via UI or API

Common queries:
```promql
# Request rate
sum(rate(http_requests_total[5m])) by (method, status)

# Error rate
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))

# Response time (P95)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Active connections
sum(http_active_connections)
```

### Grafana

**Local**: http://localhost:3000
**Default credentials**: admin/admin

Pre-configured dashboards:
1. **API Overview**: Request rates, latency, errors
2. **Workflow Metrics**: Execution stats, duration, success rate

### Alerts

Alert rules are defined in `monitoring/alerts/api_alerts.yml`:

- API Down
- High Error Rate (>5%)
- High Latency (P95 >1s)
- High Memory Usage (>90%)
- High CPU Usage (>80%)
- Database Connection Failures
- Redis Connection Failures
- Pod Restarting Too Often

### Metrics Endpoints

- **API Metrics**: http://localhost:9090/metrics
- **PostgreSQL Metrics**: http://localhost:9187/metrics
- **Redis Metrics**: http://localhost:9121/metrics

### ServiceMonitor (Kubernetes)

When using Prometheus Operator, the ServiceMonitor automatically discovers and scrapes metrics:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: workflows-api
spec:
  selector:
    matchLabels:
      app: workflows-api
  endpoints:
    - port: metrics
      interval: 30s
```

---

## CI/CD Pipeline

### GitHub Actions Workflows

#### 1. CI Pipeline (`.github/workflows/ci.yml`)

Triggered on: Push to main/develop/claude/**, Pull Requests

Steps:
- Lint code with golangci-lint
- Run tests with PostgreSQL & Redis services
- Build binary
- Security scan with gosec
- Upload coverage to Codecov

#### 2. Docker Build (`.github/workflows/docker.yml`)

Triggered on: Push to main/develop, Tags (v*)

Steps:
- Build multi-platform Docker images (amd64, arm64)
- Push to GitHub Container Registry
- Run Trivy security scan
- Tag with branch/version/SHA

#### 3. Release (`.github/workflows/release.yml`)

Triggered on: Tags (v*)

Steps:
- Create GitHub release with GoReleaser
- Build binaries for multiple platforms
- Generate changelog
- Build and push release Docker images

### Running CI Locally

```bash
# Run full CI pipeline
make ci

# Individual steps
make deps
make lint
make test
make build
```

### Container Registry

Images are published to GitHub Container Registry:

```
ghcr.io/davidmoltin/intelligent-workflows:latest
ghcr.io/davidmoltin/intelligent-workflows:develop
ghcr.io/davidmoltin/intelligent-workflows:v1.0.0
```

Pull image:
```bash
docker pull ghcr.io/davidmoltin/intelligent-workflows:latest
```

---

## Security

### Security Scanning

#### Gosec (Code Scanning)
```bash
make security-scan
```

#### Trivy (Container Scanning)
```bash
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image ghcr.io/davidmoltin/intelligent-workflows:latest
```

### Best Practices

1. **Secrets Management**
   - Never commit secrets to git
   - Use Kubernetes secrets or external secret managers (Vault, AWS Secrets Manager)
   - Rotate credentials regularly

2. **Container Security**
   - Run as non-root user
   - Use read-only root filesystem
   - Drop all capabilities
   - Scan images for vulnerabilities

3. **Network Security**
   - Enable TLS/SSL for all external communication
   - Use network policies in Kubernetes
   - Implement rate limiting

4. **Access Control**
   - Use RBAC in Kubernetes
   - Implement API authentication (JWT)
   - Audit access logs

---

## Troubleshooting

### Common Issues

#### 1. API won't start

```bash
# Check logs
docker-compose logs api

# Or in Kubernetes
kubectl logs -f -n intelligent-workflows -l app=workflows-api

# Check health
make health
make ready
```

#### 2. Database connection issues

```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Test connection
docker-compose exec api psql -h postgres -U postgres -d workflows

# Check migrations
make db-shell
# Then run: \dt
```

#### 3. High memory usage

```bash
# Check container stats
docker stats workflows-api

# Or in Kubernetes
kubectl top pods -n intelligent-workflows
```

#### 4. Monitoring not working

```bash
# Verify Prometheus targets
open http://localhost:9091/targets

# Check service discovery in Kubernetes
kubectl get servicemonitor -n intelligent-workflows
```

### Debug Mode

Enable debug logging:

```bash
# Docker Compose
docker-compose exec api sh -c 'export LOG_LEVEL=debug && ./api'

# Kubernetes
kubectl set env deployment/workflows-api LOG_LEVEL=debug -n intelligent-workflows
```

### Performance Profiling

```bash
# CPU profiling
curl http://localhost:9090/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Memory profiling
curl http://localhost:9090/debug/pprof/heap > mem.prof
go tool pprof mem.prof
```

---

## Maintenance

### Database Backups

```bash
# Create backup
make db-backup

# Restore backup
make db-restore BACKUP_FILE=backup_20241105_120000.sql
```

### Scaling

#### Horizontal Scaling (Kubernetes)

```bash
# Manual scaling
kubectl scale deployment workflows-api --replicas=5 -n intelligent-workflows

# Auto-scaling via HPA (already configured)
# Will automatically scale between 3-10 replicas based on CPU/Memory
```

#### Vertical Scaling

Update resources in `k8s/overlays/production/deployment-patch.yaml`:

```yaml
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi
```

### Rolling Updates

```bash
# Update image
kubectl set image deployment/workflows-api \
  api=ghcr.io/davidmoltin/intelligent-workflows:v1.1.0 \
  -n intelligent-workflows

# Watch rollout
kubectl rollout status deployment/workflows-api -n intelligent-workflows

# Rollback if needed
kubectl rollout undo deployment/workflows-api -n intelligent-workflows
```

---

## Support & Resources

- **Documentation**: See `/docs` directory
- **Architecture**: See `ARCHITECTURE.md`
- **Getting Started**: See `GETTING_STARTED.md`
- **Issues**: GitHub Issues

---

**Last Updated**: 2024-11-05
**Package**: F - Infrastructure & DevOps
