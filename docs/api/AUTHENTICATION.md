# API Authentication Guide

This guide covers authentication methods for the Intelligent Workflows API.

## Table of Contents

- [Overview](#overview)
- [Authentication Methods](#authentication-methods)
  - [JWT Bearer Token](#jwt-bearer-token)
  - [API Keys](#api-keys)
- [Getting Started](#getting-started)
- [Authentication Flow](#authentication-flow)
- [Security Best Practices](#security-best-practices)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)

## Overview

The Intelligent Workflows API supports two authentication methods:

1. **JWT Bearer Tokens** - For user authentication (15-minute expiration)
2. **API Keys** - For service-to-service authentication (configurable expiration)

Both methods provide secure access to protected endpoints with role-based access control (RBAC) and permission-based authorization.

## Authentication Methods

### JWT Bearer Token

JWT (JSON Web Token) authentication is recommended for user-facing applications and provides short-lived access tokens with automatic expiration.

#### Token Structure

```
Authorization: Bearer <access_token>
```

#### Token Properties

- **Access Token Lifetime**: 15 minutes
- **Refresh Token Lifetime**: 7 days
- **Algorithm**: HS256 (HMAC SHA-256)
- **Token Type**: Bearer

#### JWT Claims

Each access token contains the following claims:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "johndoe",
  "email": "john.doe@example.com",
  "roles": ["user", "admin"],
  "permissions": ["workflow:read", "workflow:create", "workflow:update"],
  "iat": 1704067200,
  "exp": 1704068100,
  "nbf": 1704067200,
  "iss": "intelligent-workflows"
}
```

#### Obtaining Access Tokens

**1. Register a New User**

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john.doe@example.com",
    "password": "SecurePass123!",
    "first_name": "John",
    "last_name": "Doe"
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "johndoe",
  "email": "john.doe@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "is_active": true,
  "is_verified": false,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**2. Login**

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "password": "SecurePass123!"
  }'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "dGhpc19pc19hX3JlZnJlc2hfdG9rZW5fZXhhbXBsZQ==",
  "expires_in": 900,
  "token_type": "Bearer",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "is_active": true,
    "is_verified": false,
    "last_login_at": "2024-01-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**3. Using the Access Token**

```bash
curl -X GET http://localhost:8080/api/v1/workflows \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### Refreshing Tokens

When your access token expires (after 15 minutes), use the refresh token to obtain a new access token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "dGhpc19pc19hX3JlZnJlc2hfdG9rZW5fZXhhbXBsZQ=="
  }'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "bmV3X3JlZnJlc2hfdG9rZW5fZXhhbXBsZQ==",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

**Note**: Each refresh operation issues a new refresh token and invalidates the old one (token rotation).

#### Logout

To revoke your refresh token:

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "dGhpc19pc19hX3JlZnJlc2hfdG9rZW5fZXhhbXBsZQ=="
  }'
```

### API Keys

API Keys are recommended for service-to-service authentication, automated scripts, and long-running processes.

#### API Key Structure

```
X-API-Key: <your_api_key>
```

#### API Key Properties

- **Format**: Base64-encoded 32-byte random value
- **Prefix**: First 8 characters stored for identification (e.g., `sk_live_`)
- **Hash Algorithm**: SHA-256 (stored securely)
- **Configurable Expiration**: Optional expiration date
- **Scopes**: Optional permission scopes

#### Creating API Keys

API keys can only be created by authenticated users with valid JWT tokens:

```bash
curl -X POST http://localhost:8080/api/v1/auth/api-keys \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production API Key",
    "scopes": ["workflow:read", "workflow:create", "event:create"],
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

Response:
```json
{
  "api_key": "iwf_prod_1234567890abcdefghijklmnopqrstuvwxyz",
  "key_prefix": "iwf_prod",
  "name": "Production API Key",
  "expires_at": "2025-12-31T23:59:59Z"
}
```

**IMPORTANT**: The full API key is only shown once during creation. Store it securely.

#### Using API Keys

```bash
curl -X GET http://localhost:8080/api/v1/workflows \
  -H "X-API-Key: iwf_prod_1234567890abcdefghijklmnopqrstuvwxyz"
```

#### Revoking API Keys

To revoke an API key:

```bash
curl -X DELETE http://localhost:8080/api/v1/auth/api-keys/{key_id} \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Getting Started

### Quick Start Example

Here's a complete example of authenticating and making an API request:

```bash
#!/bin/bash

# 1. Login and get access token
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "password": "SecurePass123!"
  }')

ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')
REFRESH_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.refresh_token')

# 2. Make authenticated request
curl -X GET http://localhost:8080/api/v1/workflows \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# 3. When token expires, refresh it
REFRESH_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")

NEW_ACCESS_TOKEN=$(echo $REFRESH_RESPONSE | jq -r '.access_token')

# 4. Use new access token
curl -X GET http://localhost:8080/api/v1/executions \
  -H "Authorization: Bearer $NEW_ACCESS_TOKEN"
```

### Client Library Examples

#### JavaScript/Node.js

```javascript
const axios = require('axios');

class IntelligentWorkflowsClient {
  constructor(baseURL = 'http://localhost:8080') {
    this.baseURL = baseURL;
    this.accessToken = null;
    this.refreshToken = null;
  }

  async login(username, password) {
    const response = await axios.post(`${this.baseURL}/api/v1/auth/login`, {
      username,
      password
    });

    this.accessToken = response.data.access_token;
    this.refreshToken = response.data.refresh_token;

    return response.data;
  }

  async refreshAccessToken() {
    const response = await axios.post(`${this.baseURL}/api/v1/auth/refresh`, {
      refresh_token: this.refreshToken
    });

    this.accessToken = response.data.access_token;
    this.refreshToken = response.data.refresh_token;

    return response.data;
  }

  async request(method, endpoint, data = null) {
    try {
      const config = {
        method,
        url: `${this.baseURL}${endpoint}`,
        headers: {
          'Authorization': `Bearer ${this.accessToken}`,
          'Content-Type': 'application/json'
        }
      };

      if (data) {
        config.data = data;
      }

      return await axios(config);
    } catch (error) {
      if (error.response?.status === 401) {
        // Token expired, refresh and retry
        await this.refreshAccessToken();
        return this.request(method, endpoint, data);
      }
      throw error;
    }
  }

  async getWorkflows() {
    const response = await this.request('GET', '/api/v1/workflows');
    return response.data;
  }

  async createWorkflow(workflow) {
    const response = await this.request('POST', '/api/v1/workflows', workflow);
    return response.data;
  }
}

// Usage
(async () => {
  const client = new IntelligentWorkflowsClient();

  await client.login('johndoe', 'SecurePass123!');

  const workflows = await client.getWorkflows();
  console.log('Workflows:', workflows);
})();
```

#### Python

```python
import requests
from typing import Optional, Dict, Any

class IntelligentWorkflowsClient:
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.access_token: Optional[str] = None
        self.refresh_token: Optional[str] = None

    def login(self, username: str, password: str) -> Dict[str, Any]:
        response = requests.post(
            f"{self.base_url}/api/v1/auth/login",
            json={"username": username, "password": password}
        )
        response.raise_for_status()

        data = response.json()
        self.access_token = data["access_token"]
        self.refresh_token = data["refresh_token"]

        return data

    def refresh_access_token(self) -> Dict[str, Any]:
        response = requests.post(
            f"{self.base_url}/api/v1/auth/refresh",
            json={"refresh_token": self.refresh_token}
        )
        response.raise_for_status()

        data = response.json()
        self.access_token = data["access_token"]
        self.refresh_token = data["refresh_token"]

        return data

    def request(self, method: str, endpoint: str, data: Optional[Dict] = None) -> requests.Response:
        headers = {
            "Authorization": f"Bearer {self.access_token}",
            "Content-Type": "application/json"
        }

        url = f"{self.base_url}{endpoint}"

        try:
            response = requests.request(method, url, json=data, headers=headers)
            response.raise_for_status()
            return response
        except requests.HTTPError as e:
            if e.response.status_code == 401:
                # Token expired, refresh and retry
                self.refresh_access_token()
                response = requests.request(method, url, json=data, headers=headers)
                response.raise_for_status()
                return response
            raise

    def get_workflows(self) -> Dict[str, Any]:
        response = self.request("GET", "/api/v1/workflows")
        return response.json()

    def create_workflow(self, workflow: Dict[str, Any]) -> Dict[str, Any]:
        response = self.request("POST", "/api/v1/workflows", workflow)
        return response.json()

# Usage
if __name__ == "__main__":
    client = IntelligentWorkflowsClient()

    # Login
    client.login("johndoe", "SecurePass123!")

    # Get workflows
    workflows = client.get_workflows()
    print("Workflows:", workflows)
```

## Authentication Flow

### JWT Flow Diagram

```
┌─────────┐                                  ┌─────────────┐
│ Client  │                                  │   API       │
└────┬────┘                                  └──────┬──────┘
     │                                              │
     │  POST /auth/login                            │
     │  { username, password }                      │
     ├─────────────────────────────────────────────>│
     │                                              │
     │              200 OK                          │
     │  { access_token, refresh_token, user }       │
     │<─────────────────────────────────────────────┤
     │                                              │
     │  GET /workflows                              │
     │  Authorization: Bearer <access_token>        │
     ├─────────────────────────────────────────────>│
     │                                              │
     │              200 OK                          │
     │              { workflows[] }                 │
     │<─────────────────────────────────────────────┤
     │                                              │
     │  ... 15 minutes later ...                    │
     │                                              │
     │  GET /executions                             │
     │  Authorization: Bearer <expired_token>       │
     ├─────────────────────────────────────────────>│
     │                                              │
     │              401 Unauthorized                │
     │<─────────────────────────────────────────────┤
     │                                              │
     │  POST /auth/refresh                          │
     │  { refresh_token }                           │
     ├─────────────────────────────────────────────>│
     │                                              │
     │              200 OK                          │
     │  { access_token, refresh_token }             │
     │<─────────────────────────────────────────────┤
     │                                              │
     │  GET /executions                             │
     │  Authorization: Bearer <new_access_token>    │
     ├─────────────────────────────────────────────>│
     │                                              │
     │              200 OK                          │
     │              { executions[] }                │
     │<─────────────────────────────────────────────┤
     │                                              │
```

## Security Best Practices

### General

1. **Always use HTTPS in production** - Never send tokens over unencrypted connections
2. **Store tokens securely** - Use secure storage mechanisms (e.g., HTTP-only cookies, secure key stores)
3. **Never expose tokens in URLs** - Always use headers for authentication
4. **Implement token rotation** - Use the refresh token flow to obtain new access tokens
5. **Logout properly** - Always revoke refresh tokens when users log out

### JWT Tokens

1. **Short expiration times** - Access tokens expire after 15 minutes to minimize security risks
2. **Implement auto-refresh** - Automatically refresh tokens before they expire
3. **Validate token expiration** - Check token expiration on the client side
4. **Handle 401 errors** - Implement automatic token refresh on authentication failures
5. **Secure token storage** - Never store tokens in localStorage for production applications

### API Keys

1. **Treat API keys like passwords** - Never commit them to source control
2. **Use environment variables** - Store API keys in environment variables or secure vaults
3. **Rotate keys regularly** - Create new keys and revoke old ones periodically
4. **Limit key scopes** - Grant only necessary permissions to each API key
5. **Set expiration dates** - Use expiring keys for temporary access
6. **Monitor key usage** - Track last_used_at timestamps to detect unauthorized use
7. **Revoke compromised keys** - Immediately revoke any key that may be compromised

### Password Security

1. **Use strong passwords** - Minimum 8 characters with complexity requirements
2. **Never log passwords** - Ensure passwords are never logged or exposed
3. **Implement rate limiting** - Prevent brute-force attacks on login endpoints
4. **Use bcrypt hashing** - Passwords are hashed with bcrypt (cost factor 12)
5. **Force re-authentication** - Require password re-entry for sensitive operations

## Error Handling

### Common Authentication Errors

#### 401 Unauthorized

**Causes:**
- Invalid credentials
- Expired access token
- Invalid or revoked API key
- Missing authentication header

**Response:**
```json
{
  "error": "Invalid credentials"
}
```

**Solution:** Re-authenticate or refresh your access token.

#### 403 Forbidden

**Causes:**
- Insufficient permissions for the requested operation
- Missing required role

**Response:**
```json
{
  "error": "Insufficient permissions"
}
```

**Solution:** Ensure your account has the required permissions/roles.

#### 429 Too Many Requests

**Causes:**
- Exceeded rate limit (100 requests per minute)

**Response:**
```json
{
  "error": "Rate limit exceeded"
}
```

**Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1704067260
```

**Solution:** Wait until the rate limit resets (check `X-RateLimit-Reset` header).

## Rate Limiting

The API implements per-user rate limiting to prevent abuse:

- **Rate Limit**: 100 requests per minute per user/IP
- **Burst Limit**: 200 requests
- **Identifier**: User ID (if authenticated) or IP address
- **Response Code**: 429 Too Many Requests

### Rate Limit Headers

All responses include rate limit information:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704067260
```

- `X-RateLimit-Limit`: Maximum requests per minute
- `X-RateLimit-Remaining`: Requests remaining in current window
- `X-RateLimit-Reset`: Unix timestamp when rate limit resets

### Handling Rate Limits

```javascript
async function makeRequestWithRateLimit(url) {
  try {
    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });

    if (response.status === 429) {
      const resetTime = parseInt(response.headers.get('X-RateLimit-Reset'));
      const waitTime = (resetTime * 1000) - Date.now();

      console.log(`Rate limited. Waiting ${waitTime}ms...`);
      await new Promise(resolve => setTimeout(resolve, waitTime));

      // Retry request
      return makeRequestWithRateLimit(url);
    }

    return response.json();
  } catch (error) {
    console.error('Request failed:', error);
    throw error;
  }
}
```

## Permissions Reference

### Workflow Permissions

- `workflow:read` - View workflows
- `workflow:create` - Create new workflows
- `workflow:update` - Update existing workflows (including enable/disable)
- `workflow:delete` - Delete workflows

### Event Permissions

- `event:create` - Emit events to trigger workflows

### Execution Permissions

- `execution:read` - View workflow executions and traces

### Approval Permissions

- `approval:read` - View approval requests
- `approval:approve` - Approve approval requests
- `approval:reject` - Reject approval requests

## Support

For additional help with authentication:

1. Check the [API Reference](http://localhost:8080/api/v1/docs/ui)
2. Review the [OpenAPI Specification](http://localhost:8080/api/v1/docs/openapi.yaml)
3. Open an issue on [GitHub](https://github.com/davidmoltin/intelligent-workflows/issues)
