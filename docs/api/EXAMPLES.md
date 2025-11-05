# API Examples

This guide provides practical examples for common API operations with complete request and response examples.

## Table of Contents

- [Setup](#setup)
- [Authentication](#authentication)
- [Workflows](#workflows)
- [Events](#events)
- [Executions](#executions)
- [Approvals](#approvals)
- [Error Handling](#error-handling)
- [Complete Workflow Example](#complete-workflow-example)

## Setup

All examples use `curl` and assume the API is running at `http://localhost:8080`.

For production, replace with your actual API base URL and always use HTTPS.

```bash
export API_BASE_URL="http://localhost:8080"
export ACCESS_TOKEN=""  # Will be set after login
```

## Authentication

### Register a New User

**Request:**
```bash
curl -X POST $API_BASE_URL/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "SecurePassword123!",
    "first_name": "Alice",
    "last_name": "Smith"
  }'
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "alice",
  "email": "alice@example.com",
  "first_name": "Alice",
  "last_name": "Smith",
  "is_active": true,
  "is_verified": false,
  "last_login_at": null,
  "created_at": "2024-01-01T10:00:00Z",
  "updated_at": "2024-01-01T10:00:00Z"
}
```

### Login

**Request:**
```bash
curl -X POST $API_BASE_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "password": "SecurePassword123!"
  }'
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAiLCJ1c2VyX2lkIjoiNTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAwIiwidXNlcm5hbWUiOiJhbGljZSIsImVtYWlsIjoiYWxpY2VAZXhhbXBsZS5jb20iLCJyb2xlcyI6WyJ1c2VyIl0sInBlcm1pc3Npb25zIjpbIndvcmtmbG93OnJlYWQiLCJ3b3JrZmxvdzpjcmVhdGUiXSwiaWF0IjoxNzA0MTA4ODAwLCJleHAiOjE3MDQxMDk3MDAsImlzcyI6ImludGVsbGlnZW50LXdvcmtmbG93cyJ9.abcd1234",
  "refresh_token": "ZGVmZzU2Nzg5MGFiY2QxMjM0NTY3ODkw",
  "expires_in": 900,
  "token_type": "Bearer",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "alice",
    "email": "alice@example.com",
    "first_name": "Alice",
    "last_name": "Smith",
    "is_active": true,
    "is_verified": false,
    "last_login_at": "2024-01-01T10:00:00Z",
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T10:00:00Z"
  }
}
```

**Save the access token:**
```bash
export ACCESS_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Get Current User Info

**Request:**
```bash
curl -X GET $API_BASE_URL/api/v1/auth/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "alice",
  "email": "alice@example.com",
  "roles": ["user"],
  "permissions": ["workflow:read", "workflow:create", "workflow:update", "event:create", "execution:read"]
}
```

### Create API Key

**Request:**
```bash
curl -X POST $API_BASE_URL/api/v1/auth/api-keys \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI/CD Pipeline Key",
    "scopes": ["workflow:read", "event:create", "execution:read"],
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

**Response:**
```json
{
  "api_key": "iwf_cicd_abcdef1234567890xyz9876543210fedcba",
  "key_prefix": "iwf_cicd",
  "name": "CI/CD Pipeline Key",
  "expires_at": "2025-12-31T23:59:59Z"
}
```

**Save the API key:**
```bash
export API_KEY="iwf_cicd_abcdef1234567890xyz9876543210fedcba"
```

## Workflows

### Create a Workflow

This example creates an order approval workflow that automatically approves small orders and requires manual approval for large ones.

**Request:**
```bash
curl -X POST $API_BASE_URL/api/v1/workflows \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "order_approval",
    "version": "1.0.0",
    "name": "Order Approval Workflow",
    "description": "Automatically approve orders under $1000, require approval for larger orders",
    "definition": {
      "trigger": {
        "type": "event",
        "event": "order.created"
      },
      "steps": [
        {
          "id": "check_amount",
          "type": "condition",
          "condition": {
            "field": "payload.amount",
            "operator": "gt",
            "value": 1000
          },
          "on_true": {
            "action": "request_approval",
            "approver_role": "manager",
            "reason": "Order amount exceeds $1000 threshold"
          },
          "on_false": {
            "action": "auto_approve"
          }
        }
      ]
    },
    "tags": ["orders", "approval", "automation"]
  }'
```

**Response:**
```json
{
  "id": "660f9511-f39c-52e5-b827-557766551111",
  "workflow_id": "order_approval",
  "version": "1.0.0",
  "name": "Order Approval Workflow",
  "description": "Automatically approve orders under $1000, require approval for larger orders",
  "definition": {
    "trigger": {
      "type": "event",
      "event": "order.created"
    },
    "steps": [
      {
        "id": "check_amount",
        "type": "condition",
        "condition": {
          "field": "payload.amount",
          "operator": "gt",
          "value": 1000
        },
        "on_true": {
          "action": "request_approval",
          "approver_role": "manager",
          "reason": "Order amount exceeds $1000 threshold"
        },
        "on_false": {
          "action": "auto_approve"
        }
      }
    ]
  },
  "enabled": false,
  "tags": ["orders", "approval", "automation"],
  "created_at": "2024-01-01T10:05:00Z",
  "updated_at": "2024-01-01T10:05:00Z"
}
```

### Enable a Workflow

**Request:**
```bash
curl -X POST $API_BASE_URL/api/v1/workflows/660f9511-f39c-52e5-b827-557766551111/enable \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "message": "Workflow enabled successfully"
}
```

### List Workflows

**Request:**
```bash
curl -X GET "$API_BASE_URL/api/v1/workflows?enabled=true&limit=10&offset=0" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "workflows": [
    {
      "id": "660f9511-f39c-52e5-b827-557766551111",
      "workflow_id": "order_approval",
      "version": "1.0.0",
      "name": "Order Approval Workflow",
      "description": "Automatically approve orders under $1000, require approval for larger orders",
      "definition": {
        "trigger": {
          "type": "event",
          "event": "order.created"
        },
        "steps": [...]
      },
      "enabled": true,
      "tags": ["orders", "approval", "automation"],
      "created_at": "2024-01-01T10:05:00Z",
      "updated_at": "2024-01-01T10:05:30Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10
}
```

### Get a Specific Workflow

**Request:**
```bash
curl -X GET $API_BASE_URL/api/v1/workflows/660f9511-f39c-52e5-b827-557766551111 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "id": "660f9511-f39c-52e5-b827-557766551111",
  "workflow_id": "order_approval",
  "version": "1.0.0",
  "name": "Order Approval Workflow",
  "description": "Automatically approve orders under $1000, require approval for larger orders",
  "definition": {
    "trigger": {
      "type": "event",
      "event": "order.created"
    },
    "steps": [
      {
        "id": "check_amount",
        "type": "condition",
        "condition": {
          "field": "payload.amount",
          "operator": "gt",
          "value": 1000
        },
        "on_true": {
          "action": "request_approval",
          "approver_role": "manager",
          "reason": "Order amount exceeds $1000 threshold"
        },
        "on_false": {
          "action": "auto_approve"
        }
      }
    ]
  },
  "enabled": true,
  "tags": ["orders", "approval", "automation"],
  "created_at": "2024-01-01T10:05:00Z",
  "updated_at": "2024-01-01T10:05:30Z"
}
```

### Update a Workflow

**Request:**
```bash
curl -X PUT $API_BASE_URL/api/v1/workflows/660f9511-f39c-52e5-b827-557766551111 \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Automatically approve orders under $1000, require manager approval for orders $1000-$5000, and director approval for orders over $5000",
    "definition": {
      "trigger": {
        "type": "event",
        "event": "order.created"
      },
      "steps": [
        {
          "id": "check_small_amount",
          "type": "condition",
          "condition": {
            "field": "payload.amount",
            "operator": "lte",
            "value": 1000
          },
          "on_true": {
            "action": "auto_approve"
          },
          "on_false": {
            "next_step": "check_medium_amount"
          }
        },
        {
          "id": "check_medium_amount",
          "type": "condition",
          "condition": {
            "field": "payload.amount",
            "operator": "lte",
            "value": 5000
          },
          "on_true": {
            "action": "request_approval",
            "approver_role": "manager"
          },
          "on_false": {
            "action": "request_approval",
            "approver_role": "director"
          }
        }
      ]
    }
  }'
```

**Response:**
```json
{
  "id": "660f9511-f39c-52e5-b827-557766551111",
  "workflow_id": "order_approval",
  "version": "1.0.0",
  "name": "Order Approval Workflow",
  "description": "Automatically approve orders under $1000, require manager approval for orders $1000-$5000, and director approval for orders over $5000",
  "definition": {
    "trigger": {
      "type": "event",
      "event": "order.created"
    },
    "steps": [...]
  },
  "enabled": true,
  "tags": ["orders", "approval", "automation"],
  "created_at": "2024-01-01T10:05:00Z",
  "updated_at": "2024-01-01T10:15:00Z"
}
```

### Delete a Workflow

**Request:**
```bash
curl -X DELETE $API_BASE_URL/api/v1/workflows/660f9511-f39c-52e5-b827-557766551111 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```
HTTP/1.1 204 No Content
```

## Events

### Emit an Event (Small Order - Auto Approved)

**Request:**
```bash
curl -X POST $API_BASE_URL/api/v1/events \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "order.created",
    "source": "order-service",
    "payload": {
      "order_id": "ORD-12345",
      "customer_id": "CUST-67890",
      "customer_name": "John Doe",
      "amount": 750.00,
      "currency": "USD",
      "items": [
        {
          "product_id": "PROD-111",
          "name": "Widget",
          "quantity": 5,
          "price": 150.00
        }
      ]
    }
  }'
```

**Response:**
```json
{
  "id": "770fa622-g40d-63f6-c938-668877662222",
  "event_type": "order.created",
  "source": "order-service",
  "payload": {
    "order_id": "ORD-12345",
    "customer_id": "CUST-67890",
    "customer_name": "John Doe",
    "amount": 750.00,
    "currency": "USD",
    "items": [
      {
        "product_id": "PROD-111",
        "name": "Widget",
        "quantity": 5,
        "price": 150.00
      }
    ]
  },
  "created_at": "2024-01-01T11:00:00Z"
}
```

### Emit an Event (Large Order - Requires Approval)

**Request:**
```bash
curl -X POST $API_BASE_URL/api/v1/events \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "order.created",
    "source": "order-service",
    "payload": {
      "order_id": "ORD-12346",
      "customer_id": "CUST-67891",
      "customer_name": "Jane Smith",
      "amount": 3500.00,
      "currency": "USD",
      "items": [
        {
          "product_id": "PROD-222",
          "name": "Premium Widget",
          "quantity": 10,
          "price": 350.00
        }
      ]
    }
  }'
```

**Response:**
```json
{
  "id": "880fb733-h51e-74g7-d049-779988773333",
  "event_type": "order.created",
  "source": "order-service",
  "payload": {
    "order_id": "ORD-12346",
    "customer_id": "CUST-67891",
    "customer_name": "Jane Smith",
    "amount": 3500.00,
    "currency": "USD",
    "items": [
      {
        "product_id": "PROD-222",
        "name": "Premium Widget",
        "quantity": 10,
        "price": 350.00
      }
    ]
  },
  "created_at": "2024-01-01T11:05:00Z"
}
```

## Executions

### List Executions

**Request:**
```bash
curl -X GET "$API_BASE_URL/api/v1/executions?workflow_id=660f9511-f39c-52e5-b827-557766551111&limit=10" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "executions": [
    {
      "id": "990fc844-i62f-85h8-e150-880099884444",
      "execution_id": "exec_20240101_110500",
      "workflow_id": "660f9511-f39c-52e5-b827-557766551111",
      "trigger_event": {
        "event_type": "order.created",
        "source": "order-service"
      },
      "trigger_payload": {
        "order_id": "ORD-12346",
        "amount": 3500.00
      },
      "context": {
        "order_id": "ORD-12346",
        "customer_id": "CUST-67891"
      },
      "status": "blocked",
      "result": "blocked",
      "started_at": "2024-01-01T11:05:00Z",
      "completed_at": null,
      "duration_ms": null,
      "error_message": null,
      "metadata": {
        "approval_required": true
      }
    },
    {
      "id": "aa0fd955-j73g-96i9-f261-991100995555",
      "execution_id": "exec_20240101_110000",
      "workflow_id": "660f9511-f39c-52e5-b827-557766551111",
      "trigger_event": {
        "event_type": "order.created",
        "source": "order-service"
      },
      "trigger_payload": {
        "order_id": "ORD-12345",
        "amount": 750.00
      },
      "context": {
        "order_id": "ORD-12345",
        "customer_id": "CUST-67890"
      },
      "status": "completed",
      "result": "allowed",
      "started_at": "2024-01-01T11:00:00Z",
      "completed_at": "2024-01-01T11:00:01.250Z",
      "duration_ms": 1250,
      "error_message": null,
      "metadata": {
        "auto_approved": true
      }
    }
  ],
  "total": 2,
  "page": 1,
  "page_size": 10
}
```

### Get Execution Details

**Request:**
```bash
curl -X GET $API_BASE_URL/api/v1/executions/990fc844-i62f-85h8-e150-880099884444 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "id": "990fc844-i62f-85h8-e150-880099884444",
  "execution_id": "exec_20240101_110500",
  "workflow_id": "660f9511-f39c-52e5-b827-557766551111",
  "trigger_event": {
    "event_type": "order.created",
    "source": "order-service"
  },
  "trigger_payload": {
    "order_id": "ORD-12346",
    "customer_id": "CUST-67891",
    "customer_name": "Jane Smith",
    "amount": 3500.00,
    "currency": "USD"
  },
  "context": {
    "order_id": "ORD-12346",
    "customer_id": "CUST-67891",
    "approval_request_id": "req_20240101_110500"
  },
  "status": "blocked",
  "result": "blocked",
  "started_at": "2024-01-01T11:05:00Z",
  "completed_at": null,
  "duration_ms": null,
  "error_message": null,
  "metadata": {
    "approval_required": true,
    "approver_role": "manager"
  }
}
```

### Get Execution Trace

**Request:**
```bash
curl -X GET $API_BASE_URL/api/v1/executions/990fc844-i62f-85h8-e150-880099884444/trace \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "execution": {
    "id": "990fc844-i62f-85h8-e150-880099884444",
    "execution_id": "exec_20240101_110500",
    "workflow_id": "660f9511-f39c-52e5-b827-557766551111",
    "status": "blocked",
    "started_at": "2024-01-01T11:05:00Z"
  },
  "steps": [
    {
      "id": "bb0fe066-k84h-07j0-g372-002211006666",
      "execution_id": "990fc844-i62f-85h8-e150-880099884444",
      "step_id": "check_amount",
      "step_type": "condition",
      "status": "completed",
      "input": {
        "field": "payload.amount",
        "operator": "gt",
        "value": 1000,
        "actual_value": 3500
      },
      "output": {
        "result": true,
        "action": "request_approval"
      },
      "error": null,
      "started_at": "2024-01-01T11:05:00.100Z",
      "completed_at": "2024-01-01T11:05:00.250Z"
    }
  ],
  "workflow": {
    "id": "660f9511-f39c-52e5-b827-557766551111",
    "workflow_id": "order_approval",
    "name": "Order Approval Workflow"
  }
}
```

## Approvals

### List Pending Approvals

**Request:**
```bash
curl -X GET "$API_BASE_URL/api/v1/approvals?status=pending" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "approvals": [
    {
      "id": "cc0ff177-l95i-18k1-h483-113322117777",
      "request_id": "req_20240101_110500",
      "execution_id": "990fc844-i62f-85h8-e150-880099884444",
      "entity_type": "order",
      "entity_id": "ORD-12346",
      "requester_id": "550e8400-e29b-41d4-a716-446655440000",
      "approver_id": null,
      "approver_role": "manager",
      "status": "pending",
      "reason": "Order amount exceeds $1000 threshold",
      "decision_reason": null,
      "requested_at": "2024-01-01T11:05:00Z",
      "decided_at": null,
      "expires_at": "2024-01-02T11:05:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

### Get Approval Details

**Request:**
```bash
curl -X GET $API_BASE_URL/api/v1/approvals/cc0ff177-l95i-18k1-h483-113322117777 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "id": "cc0ff177-l95i-18k1-h483-113322117777",
  "request_id": "req_20240101_110500",
  "execution_id": "990fc844-i62f-85h8-e150-880099884444",
  "entity_type": "order",
  "entity_id": "ORD-12346",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000",
  "approver_id": null,
  "approver_role": "manager",
  "status": "pending",
  "reason": "Order amount exceeds $1000 threshold",
  "decision_reason": null,
  "requested_at": "2024-01-01T11:05:00Z",
  "decided_at": null,
  "expires_at": "2024-01-02T11:05:00Z"
}
```

### Approve a Request

**Request:**
```bash
curl -X POST $API_BASE_URL/api/v1/approvals/cc0ff177-l95i-18k1-h483-113322117777/approve \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "decision": "approved",
    "reason": "Customer has good payment history and order details verified"
  }'
```

**Response:**
```json
{
  "id": "cc0ff177-l95i-18k1-h483-113322117777",
  "request_id": "req_20240101_110500",
  "execution_id": "990fc844-i62f-85h8-e150-880099884444",
  "entity_type": "order",
  "entity_id": "ORD-12346",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000",
  "approver_id": "660f9511-f39c-52e5-b827-557766551122",
  "approver_role": "manager",
  "status": "approved",
  "reason": "Order amount exceeds $1000 threshold",
  "decision_reason": "Customer has good payment history and order details verified",
  "requested_at": "2024-01-01T11:05:00Z",
  "decided_at": "2024-01-01T11:10:00Z",
  "expires_at": "2024-01-02T11:05:00Z"
}
```

### Reject a Request

**Request:**
```bash
curl -X POST $API_BASE_URL/api/v1/approvals/cc0ff177-l95i-18k1-h483-113322117777/reject \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "decision": "rejected",
    "reason": "Insufficient documentation provided for large order"
  }'
```

**Response:**
```json
{
  "id": "cc0ff177-l95i-18k1-h483-113322117777",
  "request_id": "req_20240101_110500",
  "execution_id": "990fc844-i62f-85h8-e150-880099884444",
  "entity_type": "order",
  "entity_id": "ORD-12346",
  "requester_id": "550e8400-e29b-41d4-a716-446655440000",
  "approver_id": "660f9511-f39c-52e5-b827-557766551122",
  "approver_role": "manager",
  "status": "rejected",
  "reason": "Order amount exceeds $1000 threshold",
  "decision_reason": "Insufficient documentation provided for large order",
  "requested_at": "2024-01-01T11:05:00Z",
  "decided_at": "2024-01-01T11:10:00Z",
  "expires_at": "2024-01-02T11:05:00Z"
}
```

## Error Handling

### 401 Unauthorized - Invalid Token

**Request:**
```bash
curl -X GET $API_BASE_URL/api/v1/workflows \
  -H "Authorization: Bearer invalid_token"
```

**Response:**
```json
{
  "error": "Invalid token"
}
```

### 403 Forbidden - Insufficient Permissions

**Request:**
```bash
curl -X DELETE $API_BASE_URL/api/v1/workflows/660f9511-f39c-52e5-b827-557766551111 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "error": "Insufficient permissions: workflow:delete required"
}
```

### 404 Not Found

**Request:**
```bash
curl -X GET $API_BASE_URL/api/v1/workflows/00000000-0000-0000-0000-000000000000 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "error": "Workflow not found"
}
```

### 429 Rate Limit Exceeded

**Request:**
```bash
# After making 100 requests in a minute
curl -X GET $API_BASE_URL/api/v1/workflows \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Response:**
```json
{
  "error": "Rate limit exceeded"
}
```

**Response Headers:**
```
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1704067260
```

## Complete Workflow Example

Here's a complete end-to-end example that demonstrates the entire workflow lifecycle:

```bash
#!/bin/bash

# Configuration
API_BASE_URL="http://localhost:8080"

echo "=== 1. Register User ==="
curl -s -X POST $API_BASE_URL/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "workflow_demo",
    "email": "demo@example.com",
    "password": "DemoPass123!",
    "first_name": "Demo",
    "last_name": "User"
  }' | jq '.'

echo -e "\n=== 2. Login ==="
LOGIN_RESPONSE=$(curl -s -X POST $API_BASE_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "workflow_demo",
    "password": "DemoPass123!"
  }')

ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')
echo "Access Token: ${ACCESS_TOKEN:0:50}..."

echo -e "\n=== 3. Create Workflow ==="
WORKFLOW_RESPONSE=$(curl -s -X POST $API_BASE_URL/api/v1/workflows \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "order_approval",
    "version": "1.0.0",
    "name": "Order Approval Workflow",
    "description": "Demo workflow for order approval",
    "definition": {
      "trigger": {
        "type": "event",
        "event": "order.created"
      },
      "steps": [
        {
          "id": "check_amount",
          "type": "condition",
          "condition": {
            "field": "payload.amount",
            "operator": "gt",
            "value": 1000
          },
          "on_true": {
            "action": "request_approval"
          },
          "on_false": {
            "action": "auto_approve"
          }
        }
      ]
    },
    "tags": ["demo"]
  }')

WORKFLOW_ID=$(echo $WORKFLOW_RESPONSE | jq -r '.id')
echo $WORKFLOW_RESPONSE | jq '.'

echo -e "\n=== 4. Enable Workflow ==="
curl -s -X POST $API_BASE_URL/api/v1/workflows/$WORKFLOW_ID/enable \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

echo -e "\n=== 5. Emit Small Order Event (Auto-Approved) ==="
curl -s -X POST $API_BASE_URL/api/v1/events \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "order.created",
    "source": "demo",
    "payload": {
      "order_id": "DEMO-001",
      "amount": 500
    }
  }' | jq '.'

sleep 2

echo -e "\n=== 6. Emit Large Order Event (Requires Approval) ==="
curl -s -X POST $API_BASE_URL/api/v1/events \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "order.created",
    "source": "demo",
    "payload": {
      "order_id": "DEMO-002",
      "amount": 2500
    }
  }' | jq '.'

sleep 2

echo -e "\n=== 7. List Executions ==="
curl -s -X GET "$API_BASE_URL/api/v1/executions?workflow_id=$WORKFLOW_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

echo -e "\n=== 8. List Pending Approvals ==="
APPROVALS_RESPONSE=$(curl -s -X GET "$API_BASE_URL/api/v1/approvals?status=pending" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

APPROVAL_ID=$(echo $APPROVALS_RESPONSE | jq -r '.approvals[0].id')
echo $APPROVALS_RESPONSE | jq '.'

echo -e "\n=== 9. Approve Request ==="
curl -s -X POST $API_BASE_URL/api/v1/approvals/$APPROVAL_ID/approve \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Approved for demo purposes"
  }' | jq '.'

echo -e "\n=== 10. Cleanup: Disable and Delete Workflow ==="
curl -s -X POST $API_BASE_URL/api/v1/workflows/$WORKFLOW_ID/disable \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq '.'

curl -s -X DELETE $API_BASE_URL/api/v1/workflows/$WORKFLOW_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN"

echo -e "\nDemo completed!"
```

Save this script as `demo.sh`, make it executable (`chmod +x demo.sh`), and run it to see the complete workflow in action.

## Additional Resources

- [OpenAPI Specification](http://localhost:8080/api/v1/docs/openapi.yaml)
- [Interactive API Documentation](http://localhost:8080/api/v1/docs/ui)
- [Authentication Guide](./AUTHENTICATION.md)
- [GitHub Repository](https://github.com/davidmoltin/intelligent-workflows)
