import { useEffect, useRef, useCallback, useState } from 'react'
import {
  getWebSocketClient,
  type WebSocketMessage,
  type WebSocketMessageType,
  type WebSocketEventHandler,
  type SubscriptionData,
} from '@/lib/websocket'

export function useWebSocket() {
  const clientRef = useRef(getWebSocketClient({
    getToken: () => {
      // Get token from localStorage or your auth provider
      return localStorage.getItem('auth_token')
    },
  }))

  useEffect(() => {
    const client = clientRef.current
    client.connect()

    return () => {
      // Don't disconnect on component unmount, keep the connection alive
      // client.disconnect()
    }
  }, [])

  const subscribe = useCallback((channel: string, filters?: SubscriptionData['filters']) => {
    clientRef.current.subscribe(channel, filters)
  }, [])

  const unsubscribe = useCallback((channel: string) => {
    clientRef.current.unsubscribe(channel)
  }, [])

  const on = useCallback((eventType: WebSocketMessageType, handler: WebSocketEventHandler) => {
    return clientRef.current.on(eventType, handler)
  }, [])

  const off = useCallback((eventType: WebSocketMessageType, handler: WebSocketEventHandler) => {
    clientRef.current.off(eventType, handler)
  }, [])

  const isConnected = useCallback(() => {
    return clientRef.current.isConnected()
  }, [])

  return {
    subscribe,
    unsubscribe,
    on,
    off,
    isConnected,
  }
}

export function useWebSocketSubscription(
  channel: string,
  filters?: SubscriptionData['filters']
) {
  const { subscribe, unsubscribe } = useWebSocket()

  useEffect(() => {
    subscribe(channel, filters)

    return () => {
      unsubscribe(channel)
    }
  }, [channel, subscribe, unsubscribe, JSON.stringify(filters)])
}

export function useWebSocketEvent(
  eventType: WebSocketMessageType,
  handler: WebSocketEventHandler
) {
  const { on, off } = useWebSocket()

  useEffect(() => {
    const unsubscribe = on(eventType, handler)
    return unsubscribe
  }, [eventType, handler, on, off])
}

export function useExecutionUpdates(executionId?: string) {
  const [execution, setExecution] = useState<any>(null)
  const [isLive, setIsLive] = useState(false)
  const { subscribe, unsubscribe, on, off } = useWebSocket()

  useEffect(() => {
    if (!executionId) return

    setIsLive(true)
    const channel = `executions:${executionId}`
    subscribe(channel)

    const handleExecutionUpdate = (message: WebSocketMessage) => {
      setExecution(message.data)
    }

    const unsubscribeStarted = on('execution.started', handleExecutionUpdate)
    const unsubscribeCompleted = on('execution.completed', handleExecutionUpdate)
    const unsubscribeFailed = on('execution.failed', handleExecutionUpdate)
    const unsubscribePaused = on('execution.paused', handleExecutionUpdate)
    const unsubscribeResumed = on('execution.resumed', handleExecutionUpdate)
    const unsubscribeCancelled = on('execution.cancelled', handleExecutionUpdate)

    return () => {
      setIsLive(false)
      unsubscribe(channel)
      unsubscribeStarted()
      unsubscribeCompleted()
      unsubscribeFailed()
      unsubscribePaused()
      unsubscribeResumed()
      unsubscribeCancelled()
    }
  }, [executionId, subscribe, unsubscribe, on, off])

  return { execution, isLive }
}

export function useWorkflowExecutions(workflowId?: string) {
  const [executions, setExecutions] = useState<Map<string, any>>(new Map())
  const { subscribe, unsubscribe, on } = useWebSocket()

  useEffect(() => {
    if (!workflowId) return

    const channel = `workflows:${workflowId}`
    subscribe(channel)

    const handleExecutionEvent = (message: WebSocketMessage) => {
      if (message.data?.execution_id) {
        setExecutions(prev => {
          const next = new Map(prev)
          next.set(message.data.execution_id, message.data)
          return next
        })
      }
    }

    const unsubscribers = [
      on('execution.started', handleExecutionEvent),
      on('execution.completed', handleExecutionEvent),
      on('execution.failed', handleExecutionEvent),
      on('execution.paused', handleExecutionEvent),
      on('execution.resumed', handleExecutionEvent),
      on('execution.cancelled', handleExecutionEvent),
    ]

    return () => {
      unsubscribe(channel)
      unsubscribers.forEach(unsub => unsub())
    }
  }, [workflowId, subscribe, unsubscribe, on])

  return Array.from(executions.values())
}

export function useAllExecutions() {
  const [executions, setExecutions] = useState<Map<string, any>>(new Map())
  const { subscribe, unsubscribe, on } = useWebSocket()

  useEffect(() => {
    const channel = 'executions'
    subscribe(channel)

    const handleExecutionEvent = (message: WebSocketMessage) => {
      if (message.data?.execution_id) {
        setExecutions(prev => {
          const next = new Map(prev)
          next.set(message.data.execution_id, message.data)
          return next
        })
      }
    }

    const unsubscribers = [
      on('execution.started', handleExecutionEvent),
      on('execution.completed', handleExecutionEvent),
      on('execution.failed', handleExecutionEvent),
      on('execution.paused', handleExecutionEvent),
      on('execution.resumed', handleExecutionEvent),
      on('execution.cancelled', handleExecutionEvent),
    ]

    return () => {
      unsubscribe(channel)
      unsubscribers.forEach(unsub => unsub())
    }
  }, [subscribe, unsubscribe, on])

  return Array.from(executions.values())
}
