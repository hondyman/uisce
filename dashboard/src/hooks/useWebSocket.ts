import { useEffect, useState, useCallback, useRef } from 'react'
import { RealTimeEvent } from '../types'

const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8081'

interface UseWebSocketOptions {
  tenantId: string;
  regions?: string[];
  onEvent?: (event: RealTimeEvent) => void;
  autoConnect?: boolean;
}

export function useWebSocket({
  tenantId,
  regions = [],
  onEvent,
  autoConnect = true
}: UseWebSocketOptions) {
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    try {
      const url = new URL('/ws', WS_URL)
      url.searchParams.set('tenant_id', tenantId)
      
      const ws = new WebSocket(url.toString())

      ws.onopen = () => {
        console.log('WebSocket connected')
        setIsConnected(true)
        setError(null)

        // Subscribe to regions
        regions.forEach((region) => {
          ws.send(JSON.stringify({
            action: 'subscribe',
            region
          }))
        })
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          
          // Skip ping messages
          if (data.type === 'ping') return

          const rtEvent: RealTimeEvent = {
            id: data.id || Math.random().toString(),
            type: data.type,
            tenant_id: data.tenant_id,
            region: data.region,
            timestamp: data.timestamp || new Date().toISOString(),
            data: data.data || {}
          }

          onEvent?.(rtEvent)
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      ws.onerror = (event) => {
        console.error('WebSocket error:', event)
        setError('Connection error')
      }

      ws.onclose = () => {
        console.log('WebSocket disconnected')
        setIsConnected(false)

        // Attempt to reconnect after 3 seconds
        reconnectTimeoutRef.current = setTimeout(() => {
          console.log('Attempting to reconnect...')
          connect()
        }, 3000)
      }

      wsRef.current = ws
    } catch (err) {
      console.error('WebSocket connection failed:', err)
      setError(err instanceof Error ? err.message : 'Failed to connect')
    }
  }, [tenantId, regions, onEvent])

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setIsConnected(false)
  }, [])

  const subscribe = useCallback((region: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        action: 'subscribe',
        region
      }))
    }
  }, [])

  const unsubscribe = useCallback((region: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        action: 'unsubscribe',
        region
      }))
    }
  }, [])

  useEffect(() => {
    if (autoConnect) {
      connect()
      return () => {
        disconnect()
      }
    }
  }, [connect, disconnect, autoConnect])

  return {
    isConnected,
    error,
    connect,
    disconnect,
    subscribe,
    unsubscribe
  }
}
