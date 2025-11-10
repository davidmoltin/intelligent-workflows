import type { Execution } from '@/types/workflow'

export type WebSocketMessageType =
  // Execution events
  | 'execution.created'
  | 'execution.started'
  | 'execution.completed'
  | 'execution.failed'
  | 'execution.paused'
  | 'execution.resumed'
  | 'execution.cancelled'
  | 'execution.blocked'
  // Step events
  | 'step.started'
  | 'step.completed'
  | 'step.failed'
  | 'step.skipped'
  // Approval events
  | 'approval.required'
  | 'approval.granted'
  | 'approval.denied'
  | 'approval.expired'
  // Connection management
  | 'ping'
  | 'pong'
  | 'error'
  | 'subscribe'
  | 'unsubscribe'
  | 'subscribed'
  | 'unsubscribed'

export interface WebSocketMessage {
  type: WebSocketMessageType
  timestamp: string
  data?: any
}

export interface ExecutionEventData {
  execution_id: string
  workflow_id: string
  status: string
  result?: string
  trigger_event?: string
  started_at?: string
  completed_at?: string
  duration_ms?: number
  error_message?: string
  context?: Record<string, any>
  paused_reason?: string
}

export interface StepEventData {
  execution_id: string
  step_id: string
  step_type: string
  status: string
  started_at?: string
  completed_at?: string
  duration_ms?: number
  error_message?: string
  output?: Record<string, any>
}

export interface SubscriptionData {
  channel: string
  filters?: {
    workflow_ids?: string[]
    execution_ids?: string[]
    statuses?: string[]
  }
}

export type WebSocketEventHandler = (message: WebSocketMessage) => void

export interface WebSocketClientOptions {
  url?: string
  autoReconnect?: boolean
  reconnectInterval?: number
  maxReconnectAttempts?: number
  getToken?: () => string | null
}

export class WebSocketClient {
  private ws: WebSocket | null = null
  private url: string
  private autoReconnect: boolean
  private reconnectInterval: number
  private maxReconnectAttempts: number
  private reconnectAttempts = 0
  private reconnectTimer: NodeJS.Timeout | null = null
  private isIntentionallyClosed = false
  private eventHandlers: Map<WebSocketMessageType, Set<WebSocketEventHandler>> = new Map()
  private getToken: () => string | null
  private subscriptions: Set<string> = new Set()

  constructor(options: WebSocketClientOptions = {}) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = import.meta.env.VITE_WS_URL || window.location.host
    this.url = options.url || `${protocol}//${host}/ws`
    this.autoReconnect = options.autoReconnect ?? true
    this.reconnectInterval = options.reconnectInterval ?? 3000
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 10
    this.getToken = options.getToken || (() => null)
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN || this.ws?.readyState === WebSocket.CONNECTING) {
      return
    }

    this.isIntentionallyClosed = false

    try {
      // Add token to URL as query parameter
      const token = this.getToken()
      const urlWithAuth = token ? `${this.url}?token=${encodeURIComponent(token)}` : this.url

      this.ws = new WebSocket(urlWithAuth)

      this.ws.onopen = this.handleOpen.bind(this)
      this.ws.onmessage = this.handleMessage.bind(this)
      this.ws.onerror = this.handleError.bind(this)
      this.ws.onclose = this.handleClose.bind(this)
    } catch (error) {
      console.error('WebSocket connection error:', error)
      this.scheduleReconnect()
    }
  }

  disconnect(): void {
    this.isIntentionallyClosed = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  subscribe(channel: string, filters?: SubscriptionData['filters']): void {
    this.subscriptions.add(channel)
    this.send({
      type: 'subscribe',
      timestamp: new Date().toISOString(),
      data: { channel, filters },
    })
  }

  unsubscribe(channel: string): void {
    this.subscriptions.delete(channel)
    this.send({
      type: 'unsubscribe',
      timestamp: new Date().toISOString(),
      data: { channel },
    })
  }

  on(eventType: WebSocketMessageType, handler: WebSocketEventHandler): () => void {
    if (!this.eventHandlers.has(eventType)) {
      this.eventHandlers.set(eventType, new Set())
    }
    this.eventHandlers.get(eventType)!.add(handler)

    // Return unsubscribe function
    return () => {
      this.off(eventType, handler)
    }
  }

  off(eventType: WebSocketMessageType, handler: WebSocketEventHandler): void {
    const handlers = this.eventHandlers.get(eventType)
    if (handlers) {
      handlers.delete(handler)
    }
  }

  send(message: WebSocketMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    } else {
      console.warn('WebSocket is not connected. Message not sent:', message)
    }
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  private handleOpen(): void {
    console.log('WebSocket connected')
    this.reconnectAttempts = 0

    // Resubscribe to all channels
    this.subscriptions.forEach(channel => {
      this.send({
        type: 'subscribe',
        timestamp: new Date().toISOString(),
        data: { channel },
      })
    })

    // Emit connection event
    this.emit({
      type: 'subscribed',
      timestamp: new Date().toISOString(),
    })
  }

  private handleMessage(event: MessageEvent): void {
    try {
      const message: WebSocketMessage = JSON.parse(event.data)
      this.emit(message)
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error)
    }
  }

  private handleError(event: Event): void {
    console.error('WebSocket error:', event)
  }

  private handleClose(event: CloseEvent): void {
    console.log('WebSocket closed:', event.code, event.reason)
    this.ws = null

    if (!this.isIntentionallyClosed && this.autoReconnect) {
      this.scheduleReconnect()
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached')
      return
    }

    this.reconnectAttempts++
    const delay = this.reconnectInterval * Math.min(this.reconnectAttempts, 5)

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})...`)

    this.reconnectTimer = setTimeout(() => {
      this.connect()
    }, delay)
  }

  private emit(message: WebSocketMessage): void {
    const handlers = this.eventHandlers.get(message.type)
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(message)
        } catch (error) {
          console.error('Error in WebSocket event handler:', error)
        }
      })
    }
  }
}

// Singleton instance
let wsClient: WebSocketClient | null = null

export function getWebSocketClient(options?: WebSocketClientOptions): WebSocketClient {
  if (!wsClient) {
    wsClient = new WebSocketClient(options)
  }
  return wsClient
}

export function disconnectWebSocket(): void {
  if (wsClient) {
    wsClient.disconnect()
    wsClient = null
  }
}
