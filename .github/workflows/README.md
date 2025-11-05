# CI/CD Workflows Documentation

This directory contains all GitHub Actions workflows for the Intelligent Workflows project. Below is a comprehensive guide to each workflow and how to configure them.

## 📋 Table of Contents

- [Workflows Overview](#workflows-overview)
- [Workflow Details](#workflow-details)
  - [CI Pipeline (ci.yml)](#ci-pipeline-ciyml)
  - [Test Suite (test.yml)](#test-suite-testyml)
  - [Docker Build (docker.yml)](#docker-build-dockeryml)
  - [Frontend CI/CD (frontend.yml)](#frontend-cicd-frontendyml)
  - [Staging Deployment (staging.yml)](#staging-deployment-stagingyml)
  - [Release Process (release.yml)](#release-process-releaseyml)
- [Configuration Guide](#configuration-guide)
- [Deployment Setup](#deployment-setup)

---

## Workflows Overview

| Workflow | Purpose | Triggers | Status |
|----------|---------|----------|--------|
| **ci.yml** | Main CI pipeline with lint, test, build, security | Push to main/develop/claude/*, PRs | ✅ Active |
| **test.yml** | Comprehensive testing (unit, integration, e2e) | Push to main/develop, PRs | ✅ Active |
| **docker.yml** | Build and push Docker images | Push to main/develop, tags | ✅ Active |
| **frontend.yml** | Frontend build, lint, and tests | Push/PR with web/ changes | ✅ Active |
| **staging.yml** | Deploy to staging environment | Push to develop/claude/* | 🟡 Needs config |
| **release.yml** | Create releases and build artifacts | Git tags (v*) | ✅ Active |

---

## Workflow Details

### CI Pipeline (ci.yml)

**Purpose**: Main continuous integration pipeline for the backend

**Triggers**:
- Push to `main`, `develop`, `claude/**` branches
- Pull requests to `main`, `develop`

**Jobs**:
1. **Lint** - Go code linting with golangci-lint
2. **Test** - Unit and integration tests with coverage
3. **Build** - Compile binary for linux/amd64
4. **Security** - Security scanning with Gosec

**Artifacts**: Binary stored for 7 days

**Coverage**: Uploads to Codecov

---

### Test Suite (test.yml)

**Purpose**: Comprehensive testing with separate unit, integration, and e2e test jobs

**Triggers**:
- Push to `main`, `develop`
- Pull requests to `main`, `develop`

**Jobs**:
1. **Unit Tests** - Fast unit tests without external dependencies
2. **Integration Tests** - Tests with PostgreSQL and Redis
3. **E2E Tests** - Full end-to-end API tests
4. **Lint Check** - Code quality checks
5. **Coverage Report** - Aggregated coverage from all test types
6. **Test Summary** - Overall test status report

**Services Used**:
- PostgreSQL 15-alpine
- Redis 7-alpine

---

### Docker Build (docker.yml)

**Purpose**: Build and push Docker images to GitHub Container Registry

**Triggers**:
- Push to `main`, `develop`
- Git tags matching `v*`
- Pull requests to `main` (build only, no push)

**Jobs**:
1. **Build Backend Image** - API server only (Dockerfile)
2. **Build Fullstack Image** - API + Frontend (Dockerfile.fullstack)

**Images Built**:
- `ghcr.io/[owner]/[repo]:[tag]-backend` - Backend only
- `ghcr.io/[owner]/[repo]:[tag]-fullstack` - Complete application

**Tags Generated**:
- Branch name (e.g., `main-backend`, `develop-fullstack`)
- Git SHA (e.g., `main-abc1234-backend`)
- Semantic versions (e.g., `v1.2.3-backend`, `1.2-backend`)
- `latest-backend` / `latest-fullstack` for default branch
- PR number for pull requests

**Security**: Trivy vulnerability scanning with SARIF reports

**Platforms**: linux/amd64, linux/arm64

---

### Frontend CI/CD (frontend.yml)

**Purpose**: Build and test frontend application

**Triggers**:
- Push to `main`, `develop`, `claude/**` with changes in `web/`
- Pull requests to `main`, `develop` with changes in `web/`

**Jobs**:
1. **Lint** - ESLint and TypeScript type checking
2. **Test** - Frontend tests (when implemented)
3. **Build** - Production build with Vite
4. **Build Preview** - PR-specific builds with preview URLs

**Artifacts**: Frontend dist/ folder stored for 7 days

**Features**:
- Bundle size reporting
- PR comments with build status
- Environment-specific builds

**Node Version**: 20

---

### Staging Deployment (staging.yml)

**Purpose**: Deploy application to staging environment

**Triggers**:
- Push to `develop`, `claude/**` branches
- Manual workflow dispatch

**Jobs**:
1. **Build Backend** - Build and push staging Docker image
2. **Build Frontend** - Build frontend with staging config
3. **Deploy Staging** - Deploy to staging environment
4. **Smoke Tests** - Basic health checks
5. **Notify** - Deployment status notifications

**Environment**: staging

**Status**: 🟡 **Requires Configuration** - See [Deployment Setup](#deployment-setup)

**Deployment Options**:
- Kubernetes (kubectl)
- Docker Compose (SSH)
- AWS ECS
- Custom deployment method

---

### Release Process (release.yml)

**Purpose**: Create GitHub releases and build artifacts

**Triggers**:
- Git tags matching `v*` (e.g., `v1.0.0`)

**Jobs**:
1. **GoReleaser** - Build cross-platform binaries
2. **Docker Release** - Build and push release images

**Artifacts**:
- Linux, macOS, Windows binaries (amd64, arm64)
- Docker images with version tags
- GitHub release with changelog

---

## Configuration Guide

### Required Secrets

The workflows use the following secrets. Configure them in Settings → Secrets and variables → Actions:

#### Container Registry (Automatic)
- `GITHUB_TOKEN` - ✅ Automatically provided by GitHub Actions

#### Staging Deployment (Required for staging.yml)

**Option 1: Kubernetes**
- `KUBE_CONFIG` - Kubernetes config file content

**Option 2: Docker Compose**
- `SSH_PRIVATE_KEY` - SSH key for server access
- `STAGING_HOST` - Server hostname or IP
- `STAGING_USER` - SSH username

**Option 3: AWS ECS**
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret key

#### Optional Secrets
- `STAGING_API_URL` - Staging API endpoint (default: https://staging-api.example.com)
- `STAGING_WEB_URL` - Staging web endpoint (default: https://staging.example.com)
- `SLACK_WEBHOOK` - Slack webhook for notifications
- `CODECOV_TOKEN` - Codecov token for coverage reports

---

## Deployment Setup

### Prerequisites

The staging deployment workflow is currently in placeholder mode. To enable it:

### Option 1: Kubernetes Deployment

1. **Create Kubernetes Manifests**:
   ```bash
   # Create deployment directory
   mkdir -p deployments/k8s/staging
   ```

2. **Add your Kubernetes config as a secret**:
   - Go to Settings → Secrets → Actions
   - Add secret `KUBE_CONFIG` with your kubeconfig content

3. **Uncomment Kubernetes steps in staging.yml**:
   - Lines for "Set up kubectl"
   - Lines for "Configure kubectl"
   - Lines for "Deploy to Kubernetes"

4. **Update deployment command**:
   ```yaml
   kubectl set image deployment/intelligent-workflows-api \
     api=${{ needs.build-backend.outputs.image-tag }} \
     -n staging
   ```

### Option 2: Docker Compose Deployment

1. **Prepare staging server**:
   ```bash
   # SSH to your staging server
   ssh user@staging-server

   # Create project directory
   mkdir -p /opt/intelligent-workflows
   cd /opt/intelligent-workflows

   # Copy docker-compose.yml
   # Configure environment variables
   ```

2. **Add SSH secrets**:
   - `SSH_PRIVATE_KEY` - Your SSH private key
   - `STAGING_HOST` - Server IP or hostname
   - `STAGING_USER` - SSH username

3. **Uncomment Docker Compose steps in staging.yml**:
   - Lines for "Deploy via Docker Compose"

### Option 3: AWS ECS Deployment

1. **Set up ECS cluster**:
   - Create ECS cluster named "staging"
   - Create ECS service named "intelligent-workflows"
   - Configure task definition

2. **Add AWS secrets**:
   - `AWS_ACCESS_KEY_ID`
   - `AWS_SECRET_ACCESS_KEY`

3. **Uncomment AWS ECS steps in staging.yml**:
   - Lines for "Configure AWS credentials"
   - Lines for "Deploy to ECS"

### Smoke Tests Configuration

1. **Update health check endpoints** in staging.yml:
   ```yaml
   - name: Check API Health
     run: |
       response=$(curl -s -o /dev/null -w "%{http_code}" ${{ secrets.STAGING_API_URL }}/health)
       if [ $response -eq 200 ]; then
         echo "✅ API health check passed"
       else
         echo "❌ API health check failed"
         exit 1
       fi
   ```

2. **Uncomment smoke test steps** once endpoints are configured

---

## Docker Images

### Backend Image (Dockerfile)
- **Image**: `ghcr.io/[owner]/[repo]:[tag]-backend`
- **Contents**: Go API server only
- **Size**: ~20MB (alpine-based)
- **Port**: 8080
- **Use Case**: Microservices, backend-only deployments

### Fullstack Image (Dockerfile.fullstack)
- **Image**: `ghcr.io/[owner]/[repo]:[tag]-fullstack`
- **Contents**: Go API + React Frontend + Nginx
- **Size**: ~50MB (alpine-based)
- **Ports**: 80 (nginx), 8080 (api)
- **Use Case**: Single-container deployments, staging, demos

**Architecture**:
```
┌─────────────────────────────┐
│   Nginx (Port 80)          │
│   ├── /      → Frontend    │
│   └── /api/  → Backend     │
├─────────────────────────────┤
│   Go API (Port 8080)       │
├─────────────────────────────┤
│   Alpine Linux             │
└─────────────────────────────┘
```

---

## Environment Variables

### Backend (Go)
- `PORT` - API server port (default: 8080)
- `DB_HOST` - PostgreSQL host
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_NAME` - Database name
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `REDIS_HOST` - Redis host
- `REDIS_PORT` - Redis port (default: 6379)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

### Frontend (React)
- `VITE_API_BASE_URL` - Backend API base URL

---

## Monitoring and Alerts

### Build Status
- Check workflow runs in Actions tab
- Status badges available for README

### Coverage Reports
- Unit test coverage: Codecov
- Integration test coverage: Codecov
- E2E test coverage: Codecov

### Security Scanning
- Vulnerability reports: Security tab → Code scanning alerts
- Dependency scanning: Dependabot alerts

---

## Troubleshooting

### Common Issues

**Issue**: Frontend workflow doesn't trigger
- **Solution**: Ensure changes are in `web/` directory and path filters match

**Issue**: Docker build fails with "no space left on device"
- **Solution**: GitHub Actions has disk space limits. Clean up unused images or use build cache

**Issue**: Tests fail in CI but pass locally
- **Solution**: Check service versions (PostgreSQL, Redis) match local environment

**Issue**: Staging deployment shows "Deployment configuration needed"
- **Solution**: Follow the [Deployment Setup](#deployment-setup) guide to configure your deployment method

**Issue**: Container registry push fails with 403
- **Solution**: Ensure workflow has `packages: write` permission

---

## Development Workflow

### Feature Development
1. Create feature branch: `git checkout -b feature/my-feature`
2. Make changes and commit
3. Push branch - triggers CI pipeline
4. Create PR - triggers all relevant workflows
5. After PR approval, merge to `develop`
6. Develop branch auto-deploys to staging

### Release Process
1. Update version in code
2. Create tag: `git tag v1.0.0`
3. Push tag: `git push origin v1.0.0`
4. GitHub Actions creates release automatically
5. Review and publish release notes

---

## Metrics and Monitoring

### Build Times
- CI pipeline: ~5-8 minutes
- Full test suite: ~10-15 minutes
- Docker builds: ~8-12 minutes
- Frontend build: ~3-5 minutes

### Success Rates
Monitor workflow success rates in the Actions tab to identify flaky tests or infrastructure issues.

---

## Support and Contributions

For issues or questions:
1. Check workflow logs in Actions tab
2. Review this documentation
3. Open an issue with workflow logs attached
4. Tag with `ci/cd` label

When contributing workflow changes:
1. Test changes in a fork first
2. Document any new secrets or configuration
3. Update this README with changes
4. Test on a branch before merging

---

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Build Push Action](https://github.com/docker/build-push-action)
- [GoReleaser Documentation](https://goreleaser.com/)
- [Trivy Security Scanner](https://github.com/aquasecurity/trivy)

---

**Last Updated**: 2025-11-05
