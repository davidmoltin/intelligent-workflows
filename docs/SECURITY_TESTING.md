# Security Testing Guide

## Overview

This document outlines the security testing framework and best practices for the Intelligent Workflows platform. Security testing is a critical part of our development process to ensure the platform is protected against common vulnerabilities and threats.

## Table of Contents

1. [Security Test Suite](#security-test-suite)
2. [Running Security Tests](#running-security-tests)
3. [Security Test Categories](#security-test-categories)
4. [Security Best Practices](#security-best-practices)
5. [Vulnerability Assessment](#vulnerability-assessment)
6. [Penetration Testing](#penetration-testing)
7. [Security Monitoring](#security-monitoring)

## Security Test Suite

The security test suite is located in `tests/security/` and includes:

- **Authentication Security Tests** (`auth_security_test.go`)
  - Brute force protection
  - Weak password rejection
  - JWT token validation
  - Password hashing security
  - API key secrecy
  - SQL injection protection
  - XSS protection
  - Rate limiting
  - CORS configuration
  - Secure headers

- **RBAC Security Tests** (`rbac_security_test.go`)
  - Workflow permissions
  - Approval permissions
  - Resource ownership
  - Privilege escalation prevention
  - API key scope enforcement

## Running Security Tests

### Run All Security Tests

```bash
# Run all security tests
SECURITY_TESTS=1 go test ./tests/security/... -v

# Run with coverage
SECURITY_TESTS=1 go test ./tests/security/... -v -coverprofile=security_coverage.out
```

### Run Specific Test Categories

```bash
# Authentication security tests only
go test ./tests/security/... -run TestSecurity_Auth -v

# RBAC security tests only
go test ./tests/security/... -run TestSecurity_RBAC -v
```

### Integration with CI/CD

Security tests are automatically run in the CI/CD pipeline on:
- Pull requests to `main` and `develop`
- Nightly security scans
- Before production deployments

## Security Test Categories

### 1. Authentication & Authorization

**Tests:**
- Password strength validation
- JWT token generation and validation
- Token expiration handling
- API key management
- Multi-factor authentication (if implemented)

**Example:**
```go
func TestSecurity_WeakPasswordRejection(t *testing.T) {
    // Test that weak passwords are rejected
    weakPasswords := []string{"123456", "password", "qwerty"}
    for _, pwd := range weakPasswords {
        // Attempt registration with weak password
        // Assert: Should return 400 Bad Request
    }
}
```

### 2. Input Validation

**Tests:**
- SQL injection prevention
- XSS (Cross-Site Scripting) protection
- Command injection prevention
- Path traversal prevention
- JSON/XML bomb attacks

**Key Areas:**
- User registration/login
- Workflow creation
- Event triggering
- API endpoints

### 3. Access Control

**Tests:**
- Role-Based Access Control (RBAC)
- Resource ownership verification
- Privilege escalation prevention
- API scope enforcement
- Cross-tenant data access prevention

**Example:**
```go
func TestSecurity_ResourceOwnership(t *testing.T) {
    // User A creates a workflow
    // User B attempts to delete User A's workflow
    // Assert: Should return 403 Forbidden
}
```

### 4. Rate Limiting

**Tests:**
- Per-user rate limiting
- Per-IP rate limiting
- Burst protection
- Brute force protection

**Default Limits:**
- 100 requests per minute per user
- 200 burst limit
- Login attempts: 5 per 15 minutes

### 5. Data Protection

**Tests:**
- Password hashing (bcrypt)
- Sensitive data in responses
- API key storage and transmission
- Encryption at rest
- Encryption in transit (TLS)

### 6. Session Management

**Tests:**
- Token expiration
- Token refresh mechanism
- Session invalidation on logout
- Concurrent session handling

**Token Lifetimes:**
- Access token: 15 minutes
- Refresh token: 7 days

## Security Best Practices

### 1. Authentication

✅ **DO:**
- Use bcrypt for password hashing (cost factor 12)
- Implement strong password policies
- Use JWT with proper signing and validation
- Rotate API keys regularly
- Implement rate limiting on authentication endpoints

❌ **DON'T:**
- Store passwords in plaintext
- Use weak hashing algorithms (MD5, SHA1)
- Expose password hashes in API responses
- Allow unlimited login attempts

### 2. Authorization

✅ **DO:**
- Implement role-based access control (RBAC)
- Verify resource ownership before operations
- Use principle of least privilege
- Validate API key scopes

❌ **DON'T:**
- Trust client-side authorization
- Skip permission checks
- Allow privilege escalation
- Use overly broad permissions

### 3. Input Validation

✅ **DO:**
- Validate all user inputs
- Use parameterized queries
- Sanitize HTML/JavaScript
- Implement request size limits
- Validate content types

❌ **DON'T:**
- Trust user input
- Concatenate SQL queries
- Execute user-provided code
- Allow unbounded input

### 4. API Security

✅ **DO:**
- Use HTTPS only
- Implement CORS properly
- Set security headers
- Version your APIs
- Rate limit all endpoints

❌ **DON'T:**
- Expose sensitive information in errors
- Allow unlimited requests
- Use predictable IDs
- Disable security features in production

### 5. Error Handling

✅ **DO:**
- Log security events
- Return generic error messages
- Monitor failed authentication attempts
- Alert on suspicious activity

❌ **DON'T:**
- Expose stack traces
- Reveal system information
- Log sensitive data
- Ignore security errors

## Vulnerability Assessment

### Automated Scanning

We use the following tools for automated vulnerability scanning:

1. **gosec** - Go security checker
   ```bash
   gosec ./...
   ```

2. **Trivy** - Container vulnerability scanner
   ```bash
   trivy image intelligent-workflows:latest
   ```

3. **OWASP Dependency-Check**
   ```bash
   dependency-check --project "Intelligent Workflows" --scan .
   ```

### Manual Testing

Perform manual security testing for:
- Business logic flaws
- Race conditions
- Authentication bypass
- Authorization bypass
- Data leakage

## Penetration Testing

### Scope

**In Scope:**
- All API endpoints
- Authentication mechanisms
- Authorization controls
- Input validation
- Session management

**Out of Scope:**
- Denial of Service (DoS) attacks
- Physical security
- Social engineering
- Third-party services

### Testing Methodology

1. **Reconnaissance**
   - Map all endpoints
   - Identify authentication mechanisms
   - Discover input validation points

2. **Vulnerability Scanning**
   - Run automated scanners
   - Identify potential vulnerabilities
   - Prioritize findings

3. **Exploitation**
   - Attempt to exploit vulnerabilities
   - Test authentication bypass
   - Test authorization bypass
   - Test injection attacks

4. **Reporting**
   - Document all findings
   - Provide remediation steps
   - Assign severity levels
   - Track resolution

### Severity Levels

- **Critical**: Immediate risk, requires immediate action
- **High**: Significant risk, requires prompt action
- **Medium**: Moderate risk, should be addressed soon
- **Low**: Minor risk, address when convenient
- **Info**: No immediate risk, informational

## Security Monitoring

### Logging

Log the following security events:
- Failed login attempts
- Successful logins
- Password changes
- API key creation/revocation
- Permission changes
- Unauthorized access attempts
- Rate limit violations

### Metrics

Monitor:
- Failed authentication rate
- Rate limit hits
- Suspicious patterns
- Error rates
- Response times

### Alerting

Alert on:
- Multiple failed login attempts
- Privilege escalation attempts
- Unusual API usage patterns
- Security scan detection
- Data exfiltration attempts

## Security Checklist

Before production deployment:

- [ ] All security tests passing
- [ ] gosec scan clean
- [ ] Trivy scan clean
- [ ] OWASP top 10 vulnerabilities addressed
- [ ] Rate limiting configured
- [ ] CORS properly configured
- [ ] Security headers set
- [ ] TLS/HTTPS enforced
- [ ] Secrets not in code
- [ ] Logging configured
- [ ] Monitoring configured
- [ ] Incident response plan in place

## Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [CWE Top 25](https://cwe.mitre.org/top25/)

## Contact

For security issues or concerns:
- **Security Team**: security@example.com
- **Bug Bounty**: bugbounty@example.com

**Note**: Do not report security vulnerabilities through public GitHub issues.
