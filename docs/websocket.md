# WebSocket Real-Time Updates

This document describes the WebSocket implementation for real-time execution updates in the Intelligent Workflows platform.

## Overview

The WebSocket implementation provides real-time updates for workflow executions, steps, and approvals. It uses a pub/sub architecture with Redis for distributed support, allowing multiple server instances to broadcast events to connected clients.

## Architecture

### Backend Components

1. **WebSocket Hub** (`internal/websocket/hub.go`)
   - Manages all active WebSocket connections
   - Handles client registration/unregistration
   - Broadcasts messages to subscribed clients
   - Integrates with Redis pub/sub for distributed broadcasting

2. **WebSocket Client** (`internal/websocket/client.go`)
   - Represents an individual WebSocket connection
   - Manages subscriptions and filters
   - Handles message routing and ping/pong

3. **Message Types** (`internal/websocket/message.go`)
   - Defines all WebSocket message types and data structures
   - Execution events: created, started, completed, failed, paused, resumed, cancelled, blocked
   - Step events: started, completed, failed, skipped
   - Approval events: required, granted, denied, expired

4. **HTTP Handler** (`internal/websocket/handler.go`)
   - Handles WebSocket upgrade requests
   - Validates authentication (JWT or API key)
   - Creates and registers new clients

### Frontend Components

1. **WebSocket Client** (`web/src/lib/websocket.ts`)
   - Manages WebSocket connection lifecycle
   - Automatic reconnection with exponential backoff
   - Event-based message handling
   - Subscription management

2. **React Hooks** (`web/src/hooks/useWebSocket.ts`)
   - `useWebSocket`: Core hook for WebSocket operations
   - `useWebSocketSubscription`: Subscribe to channels
   - `useWebSocketEvent`: Listen to specific event types
   - `useExecutionUpdates`: Real-time updates for a specific execution
   - `useWorkflowExecutions`: Real-time updates for all executions of a workflow
   - `useAllExecutions`: Real-time updates for all executions

## API Reference

### WebSocket Endpoint

```
ws://localhost:8080/ws
wss://production-host/ws
```

Authentication is required via:
- JWT token in Authorization header (Bearer token)
- API key in X-API-Key header
- Token as query parameter: `?token=<jwt_token>`

### Message Format

All WebSocket messages follow this structure:

```json
{
  "type": "execution.started",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    // Event-specific data
  }
}
```

### Message Types

#### Execution Events

- `execution.created`: New execution created
- `execution.started`: Execution started
- `execution.completed`: Execution completed successfully
- `execution.failed`: Execution failed
- `execution.paused`: Execution paused (waiting for event/approval)
- `execution.resumed`: Execution resumed
- `execution.cancelled`: Execution cancelled
- `execution.blocked`: Execution blocked by action

#### Step Events

- `step.started`: Step execution started
- `step.completed`: Step execution completed
- `step.failed`: Step execution failed
- `step.skipped`: Step execution skipped

#### Approval Events

- `approval.required`: Approval required
- `approval.granted`: Approval granted
- `approval.denied`: Approval denied
- `approval.expired`: Approval expired

#### Connection Management

- `ping`: Ping message (client to server)
- `pong`: Pong response (server to client)
- `subscribe`: Subscribe to channel
- `unsubscribe`: Unsubscribe from channel
- `subscribed`: Subscription confirmed
- `unsubscribed`: Unsubscription confirmed
- `error`: Error message

### Subscription Channels

Clients can subscribe to different channels to filter events:

1. **All Executions**: `executions`
   - Receives all execution events

2. **Specific Execution**: `executions:{execution_id}`
   - Receives events for a specific execution only

3. **Workflow Executions**: `workflows:{workflow_id}`
   - Receives all execution events for a specific workflow

4. **All Approvals**: `approvals`
   - Receives all approval events

### Subscription Filters

When subscribing to a channel, you can specify filters:

```json
{
  "type": "subscribe",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "channel": "executions",
    "filters": {
      "workflow_ids": ["uuid1", "uuid2"],
      "execution_ids": ["exec_abc123"],
      "statuses": ["running", "failed"]
    }
  }
}
```

## Usage Examples

### Backend Integration

The executor automatically broadcasts events when executions change state:

```go
// Execution started
execution.Status = models.ExecutionStatusRunning
we.broadcastExecutionEvent(execution)

// Execution completed
execution.Status = models.ExecutionStatusCompleted
we.broadcastExecutionEvent(execution)
```

### Frontend Usage

#### Basic WebSocket Connection

```typescript
import { getWebSocketClient } from '@/lib/websocket'

const ws = getWebSocketClient({
  getToken: () => localStorage.getItem('auth_token')
})

ws.connect()
```

#### Subscribe to Execution Updates

```typescript
import { useExecutionUpdates } from '@/hooks/useWebSocket'

function ExecutionDetailPage() {
  const { execution, isLive } = useExecutionUpdates(executionId)

  return (
    <div>
      {isLive && <Badge>Live</Badge>}
      <div>Status: {execution?.status}</div>
    </div>
  )
}
```

#### Subscribe to All Executions

```typescript
import { useAllExecutions } from '@/hooks/useWebSocket'

function ExecutionsListPage() {
  const liveExecutions = useAllExecutions()

  return (
    <div>
      {liveExecutions.map(exec => (
        <ExecutionCard key={exec.execution_id} execution={exec} />
      ))}
    </div>
  )
}
```

#### Custom Event Handler

```typescript
import { useWebSocketEvent } from '@/hooks/useWebSocket'

function MyComponent() {
  useWebSocketEvent('execution.failed', (message) => {
    const data = message.data
    console.log('Execution failed:', data.execution_id)
    // Show notification, update UI, etc.
  })

  return <div>...</div>
}
```

## Configuration

### Backend Configuration

WebSocket settings can be configured via environment variables:

```bash
# Server settings (affects WebSocket)
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s

# Redis settings (required for distributed WebSocket)
REDIS_HOST=localhost
REDIS_PORT=6379

# CORS settings (must include WebSocket origin)
ALLOWED_ORIGINS=http://localhost:3000,https://app.example.com
```

### Frontend Configuration

WebSocket URL can be configured via environment variable:

```bash
# .env
VITE_WS_URL=ws://localhost:8080
```

If not set, it defaults to the same host as the web application.

## Connection Management

### Automatic Reconnection

The WebSocket client automatically reconnects on disconnect with exponential backoff:

- Initial interval: 3 seconds
- Max attempts: 10
- Backoff multiplier: increases with each attempt (capped at 5x)

### Heartbeat

The server sends ping messages every 54 seconds (configurable). The client must respond with pong to keep the connection alive.

If no pong is received within 60 seconds, the connection is closed.

### Subscription Recovery

When reconnecting, the client automatically resubscribes to all previously subscribed channels.

## Distributed Architecture

### Redis Pub/Sub

The WebSocket hub uses Redis pub/sub to broadcast events across multiple server instances:

1. When an execution event occurs, the server publishes it to Redis
2. All server instances receive the event via Redis subscription
3. Each server broadcasts the event to its connected WebSocket clients

This ensures that clients receive events regardless of which server instance they're connected to.

### Channel Structure

Redis pub/sub channel: `ws:broadcast:all`

All WebSocket broadcasts are published to this channel, with the message containing:
- Channel (execution, workflow, etc.)
- Message type
- Event data
- Filters (workflow_id, execution_id, status)

## Performance Considerations

### Connection Limits

- Default: Unlimited connections per server
- Recommended: Configure load balancer for connection limits
- Memory usage: ~1KB per connection

### Message Buffer

- Client send buffer: 256 messages
- If buffer is full, connection is closed
- Recommendation: Implement flow control for high-volume scenarios

### Broadcast Performance

- Hub broadcast channel: 256 messages
- If channel is full, messages are dropped (logged as warning)
- Recommendation: Monitor hub metrics and scale horizontally if needed

## Monitoring

### Metrics

The WebSocket hub exposes statistics via `/ws/stats` endpoint (requires authentication):

```json
{
  "total_clients": 42,
  "total_users": 15,
  "channels": {
    "register": 0,
    "unregister": 0,
    "broadcast": 5
  }
}
```

### Logging

WebSocket events are logged with:
- Connection established/closed
- Client registration/unregistration
- Subscription changes
- Broadcast statistics

## Security

### Authentication

All WebSocket connections require authentication:
- JWT tokens (from `/api/v1/auth/login`)
- API keys (from `/api/v1/auth/api-keys`)

### Authorization

Clients can only subscribe to:
- Public channels (executions, workflows, approvals)
- Their own user-specific channels

### Message Validation

All incoming messages are validated:
- JSON structure
- Message type
- Data format

Invalid messages receive an error response and are not processed.

## Troubleshooting

### Connection Issues

1. **Unable to connect**
   - Check authentication token is valid
   - Verify CORS settings allow WebSocket origin
   - Check firewall/proxy allows WebSocket upgrades

2. **Frequent disconnections**
   - Check network stability
   - Verify ping/pong is working
   - Check server logs for errors

3. **No events received**
   - Verify subscription is active (`subscribed` event received)
   - Check filters are not too restrictive
   - Verify events are being triggered on backend

### Debug Mode

Enable WebSocket debug logging in browser console:

```typescript
localStorage.setItem('debug', 'websocket:*')
```

Check server logs for WebSocket events:

```bash
# Set log level to debug
LOG_LEVEL=debug
```

## Future Enhancements

Potential improvements for the WebSocket implementation:

1. **Rate Limiting**: Prevent abuse by limiting messages per client
2. **Compression**: Add per-message deflate compression for large payloads
3. **Binary Protocol**: Use binary format for improved performance
4. **Message History**: Store recent messages for late-joining clients
5. **Priority Channels**: Separate channels for high-priority events
6. **Client Capabilities**: Negotiate features during handshake
7. **User Presence**: Track online/offline status of users

## References

- [WebSocket RFC 6455](https://tools.ietf.org/html/rfc6455)
- [gorilla/websocket](https://github.com/gorilla/websocket)
- [Redis Pub/Sub](https://redis.io/topics/pubsub)
