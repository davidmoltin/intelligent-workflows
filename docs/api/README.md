# Intelligent Workflows API Documentation

Welcome to the Intelligent Workflows API documentation! This directory contains comprehensive documentation for the API.

## Quick Links

- **[Interactive API Documentation (Swagger UI)](http://localhost:8080/api/v1/docs/ui)** - Try out the API directly in your browser
- **[OpenAPI Specification](http://localhost:8080/api/v1/docs/openapi.yaml)** - Complete API specification in OpenAPI 3.0 format
- **[Authentication Guide](./AUTHENTICATION.md)** - Detailed guide on JWT and API Key authentication
- **[API Examples](./EXAMPLES.md)** - Practical examples with curl commands

## Getting Started

### 1. Start the API Server

```bash
# From the project root
./bin/api

# Or using Docker
docker-compose up api
```

The API will be available at `http://localhost:8080`.

### 2. Access the Documentation

Once the server is running, open your browser and navigate to:

```
http://localhost:8080/api/v1/docs/ui
```

This will open the interactive Swagger UI where you can explore all endpoints and try them out directly.

### 3. Register and Authenticate

Before using most endpoints, you'll need to register and authenticate:

```bash
# Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "myuser",
    "email": "myuser@example.com",
    "password": "SecurePass123!",
    "first_name": "My",
    "last_name": "User"
  }'

# Login to get access token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "myuser",
    "password": "SecurePass123!"
  }'
```

Save the `access_token` from the login response and use it for authenticated requests:

```bash
curl -X GET http://localhost:8080/api/v1/workflows \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Documentation Structure

### OpenAPI Specification (`openapi.yaml`)

The OpenAPI 3.0 specification provides a complete, machine-readable description of the API:

- All endpoints with request/response schemas
- Authentication requirements
- Data models and validation rules
- Example requests and responses

You can use this specification to:
- Generate client libraries in various languages
- Import into API testing tools (Postman, Insomnia, etc.)
- Validate API contracts
- Auto-generate documentation

### Authentication Guide (`AUTHENTICATION.md`)

Comprehensive guide covering:
- JWT Bearer Token authentication
- API Key authentication
- Token refresh flow
- Security best practices
- Rate limiting
- Error handling
- Client library examples (JavaScript, Python)

### API Examples (`EXAMPLES.md`)

Practical, ready-to-use examples:
- Complete curl commands for all endpoints
- Sample request and response payloads
- Common workflows and use cases
- Error handling examples
- End-to-end workflow demo script

## API Overview

### Base URL

```
http://localhost:8080
```

For production, use your deployed API URL with HTTPS.

### Authentication

The API supports two authentication methods:

1. **JWT Bearer Token** (recommended for user authentication)
   ```
   Authorization: Bearer <access_token>
   ```

2. **API Key** (recommended for service-to-service)
   ```
   X-API-Key: <your_api_key>
   ```

### Rate Limits

- **100 requests per minute** per user/IP
- **Burst limit**: 200 requests
- Rate limit headers included in all responses

### API Versioning

All endpoints are versioned under `/api/v1/`. Future versions will be available at `/api/v2/`, etc.

## Available Endpoints

### Health & Readiness

- `GET /health` - Health check
- `GET /ready` - Readiness check (includes database and Redis status)

### Authentication

- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login and get tokens
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout and revoke refresh token
- `GET /api/v1/auth/me` - Get current user info
- `POST /api/v1/auth/change-password` - Change password
- `POST /api/v1/auth/api-keys` - Create API key
- `DELETE /api/v1/auth/api-keys/{id}` - Revoke API key

### Workflows

- `GET /api/v1/workflows` - List workflows
- `POST /api/v1/workflows` - Create workflow
- `GET /api/v1/workflows/{id}` - Get workflow details
- `PUT /api/v1/workflows/{id}` - Update workflow
- `DELETE /api/v1/workflows/{id}` - Delete workflow
- `POST /api/v1/workflows/{id}/enable` - Enable workflow
- `POST /api/v1/workflows/{id}/disable` - Disable workflow

### Events

- `POST /api/v1/events` - Emit event to trigger workflows

### Executions

- `GET /api/v1/executions` - List workflow executions
- `GET /api/v1/executions/{id}` - Get execution details
- `GET /api/v1/executions/{id}/trace` - Get execution trace with steps

### Approvals

- `GET /api/v1/approvals` - List approval requests
- `GET /api/v1/approvals/{id}` - Get approval details
- `POST /api/v1/approvals/{id}/approve` - Approve request
- `POST /api/v1/approvals/{id}/reject` - Reject request

## Tools and Integration

### Swagger UI

The built-in Swagger UI provides an interactive interface to:
- Browse all available endpoints
- View request/response schemas
- Try out API calls directly from your browser
- Authenticate and save credentials for the session

Access it at: `http://localhost:8080/api/v1/docs/ui`

### Postman

Import the OpenAPI specification into Postman:

1. Open Postman
2. Click "Import" → "Link"
3. Enter: `http://localhost:8080/api/v1/docs/openapi.yaml`
4. Click "Continue" and "Import"

### Client Generation

Generate client libraries using OpenAPI Generator:

```bash
# Install OpenAPI Generator
npm install -g @openapitools/openapi-generator-cli

# Generate TypeScript client
openapi-generator-cli generate \
  -i http://localhost:8080/api/v1/docs/openapi.yaml \
  -g typescript-axios \
  -o ./clients/typescript

# Generate Python client
openapi-generator-cli generate \
  -i http://localhost:8080/api/v1/docs/openapi.yaml \
  -g python \
  -o ./clients/python

# Generate Go client
openapi-generator-cli generate \
  -i http://localhost:8080/api/v1/docs/openapi.yaml \
  -g go \
  -o ./clients/go
```

## Response Formats

### Success Response

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "workflow_id": "order_approval",
  "name": "Order Approval Workflow",
  ...
}
```

### Error Response

```json
{
  "error": "Invalid credentials"
}
```

### Paginated Response

```json
{
  "workflows": [...],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

## Common Response Codes

- `200 OK` - Request succeeded
- `201 Created` - Resource created successfully
- `204 No Content` - Request succeeded with no response body
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Authentication required or invalid credentials
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

## Support

- **Documentation Issues**: [GitHub Issues](https://github.com/davidmoltin/intelligent-workflows/issues)
- **API Bugs**: [GitHub Issues](https://github.com/davidmoltin/intelligent-workflows/issues)
- **Feature Requests**: [GitHub Discussions](https://github.com/davidmoltin/intelligent-workflows/discussions)

## Contributing

Found an error in the documentation or want to improve it? Contributions are welcome!

1. Fork the repository
2. Make your changes to the documentation files
3. Submit a pull request

## Version History

### v1.0.0 (Current)

- Initial API release
- JWT and API Key authentication
- Workflow management
- Event emission
- Execution tracking
- Approval workflows
- Complete OpenAPI 3.0 specification
- Interactive Swagger UI
- Comprehensive documentation

## License

MIT License - See [LICENSE](../../LICENSE) for details.
