import { useEffect, useRef, useState, useCallback } from 'react';
import { devError } from '../../../api';

export interface WebSocketMessage {
  type: string;
  payload: unknown;
  timestamp: string;
}

export interface UpgradeStatusMessage {
  coreVersion: string;
  status: string;
  warnings?: string[];
  blockers?: string[];
}

export interface WebSocketError {
  type: 'connection' | 'message' | 'timeout' | 'unknown';
  message: string;
  timestamp: number;
  retryCount: number;
}

export interface ConnectionStats {
  connectedAt: number;
  messagesReceived: number;
  messagesSent: number;
  errors: number;
  reconnectAttempts: number;
  lastPing: number;
}

export const useWebSocket = (url: string) => {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);
  const [error, setError] = useState<WebSocketError | null>(null);
  const [connectionStats, setConnectionStats] = useState<ConnectionStats>({
    connectedAt: 0,
    messagesReceived: 0,
    messagesSent: 0,
    errors: 0,
    reconnectAttempts: 0,
    lastPing: 0,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const pingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttempts = useRef(0);
  const maxReconnectAttempts = 10;
  const baseReconnectDelay = 1000; // 1 second
  const maxReconnectDelay = 30000; // 30 seconds
  const pingInterval = 30000; // 30 seconds
  const connectionTimeout = 10000; // 10 seconds

  // Calculate exponential backoff delay
  const getReconnectDelay = useCallback((attempt: number): number => {
    const delay = Math.min(baseReconnectDelay * Math.pow(2, attempt), maxReconnectDelay);
    // Add jitter to prevent thundering herd
    return delay + Math.random() * 1000;
  }, []);

  // Send ping to keep connection alive
  const sendPing = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      const pingMessage: WebSocketMessage = {
        type: 'ping',
        payload: { timestamp: Date.now() },
        timestamp: Date.now().toString(),
      };

      wsRef.current.send(JSON.stringify(pingMessage));
      setConnectionStats(prev => ({ ...prev, messagesSent: prev.messagesSent + 1 }));
    }
  }, []);

  // Start ping interval
  const startPingInterval = useCallback(() => {
    if (pingIntervalRef.current) {
      clearInterval(pingIntervalRef.current);
    }
    pingIntervalRef.current = setInterval(sendPing, pingInterval);
  }, [sendPing]);

  // Stop ping interval
  const stopPingInterval = useCallback(() => {
    if (pingIntervalRef.current) {
      clearInterval(pingIntervalRef.current);
      pingIntervalRef.current = null;
    }
  }, []);

  const connect = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.CONNECTING) {
      return; // Already connecting
    }

    try {
      setError(null);
      const ws = new WebSocket(url);

      // Set connection timeout
      const connectionTimer = setTimeout(() => {
        if (ws.readyState === WebSocket.CONNECTING) {
          ws.close();
          setError({
            type: 'timeout',
            message: 'Connection timeout',
            timestamp: Date.now(),
            retryCount: reconnectAttempts.current,
          });
        }
      }, connectionTimeout);

      ws.onopen = () => {
        clearTimeout(connectionTimer);
        setIsConnected(true);
        setError(null);
        reconnectAttempts.current = 0;
        setConnectionStats(prev => ({
          ...prev,
          connectedAt: Date.now(),
          reconnectAttempts: reconnectAttempts.current,
        }));
        startPingInterval();
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          setLastMessage(message);
          setConnectionStats(prev => ({
            ...prev,
            messagesReceived: prev.messagesReceived + 1,
            lastPing: message.type === 'pong' ? Date.now() : prev.lastPing,
          }));

          // Handle pong responses
          if (message.type === 'pong') {
            setConnectionStats(prev => ({ ...prev, lastPing: Date.now() }));
          }
        } catch (err) {
          devError('Failed to parse WebSocket message:', err);
          setError({
            type: 'message',
            message: 'Failed to parse message',
            timestamp: Date.now(),
            retryCount: reconnectAttempts.current,
          });
          setConnectionStats(prev => ({ ...prev, errors: prev.errors + 1 }));
        }
      };

      ws.onclose = (event) => {
        clearTimeout(connectionTimer);
        setIsConnected(false);
        stopPingInterval();

        // Don't reconnect for normal closures
        if (event.code === 1000) {
          return;
        }

        // Attempt to reconnect with exponential backoff
        if (reconnectAttempts.current < maxReconnectAttempts) {
          reconnectAttempts.current++;
          const delay = getReconnectDelay(reconnectAttempts.current - 1);

          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, delay);
        } else {
          setError({
            type: 'connection',
            message: `Failed to reconnect after ${maxReconnectAttempts} attempts`,
            timestamp: Date.now(),
            retryCount: reconnectAttempts.current,
          });
        }
      };

      ws.onerror = (event) => {
        devError('WebSocket error:', event);
        setError({
          type: 'connection',
          message: 'WebSocket connection error',
          timestamp: Date.now(),
          retryCount: reconnectAttempts.current,
        });
        setConnectionStats(prev => ({ ...prev, errors: prev.errors + 1 }));
      };

      wsRef.current = ws;
    } catch (err) {
      devError('Failed to create WebSocket connection:', err);
      setError({
        type: 'connection',
        message: 'Failed to create WebSocket connection',
        timestamp: Date.now(),
        retryCount: reconnectAttempts.current,
      });
    }
  }, [url, getReconnectDelay, startPingInterval, stopPingInterval]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    stopPingInterval();

    if (wsRef.current) {
      wsRef.current.close(1000, 'Component unmounting');
      wsRef.current = null;
    }

    setIsConnected(false);
    setLastMessage(null);
    setError(null);
  }, [stopPingInterval]);

  const sendMessage = useCallback((message: WebSocketMessage) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      try {
        wsRef.current.send(JSON.stringify({
          ...message,
          timestamp: Date.now().toString(),
        }));
        setConnectionStats(prev => ({ ...prev, messagesSent: prev.messagesSent + 1 }));
      } catch (err) {
        devError('Failed to send WebSocket message:', err);
        setError({
          type: 'message',
          message: 'Failed to send message',
          timestamp: Date.now(),
          retryCount: reconnectAttempts.current,
        });
        setConnectionStats(prev => ({ ...prev, errors: prev.errors + 1 }));
      }
    } else {
      setError({
        type: 'connection',
        message: 'Cannot send message: WebSocket not connected',
        timestamp: Date.now(),
        retryCount: reconnectAttempts.current,
      });
    }
  }, []);

  const forceReconnect = useCallback(() => {
    disconnect();
    reconnectAttempts.current = 0;
    setTimeout(() => connect(), 100);
  }, [connect, disconnect]);

  useEffect(() => {
    connect();

    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return {
    isConnected,
    lastMessage,
    error,
    connectionStats,
    sendMessage,
    reconnect: forceReconnect,
    disconnect,
  };
};

// Hook specifically for upgrade status updates
export const useUpgradeWebSocket = () => {
  const { isConnected, lastMessage, error, connectionStats, sendMessage } = useWebSocket('/api/ws/upgrade-status');

  const [upgradeStatuses, setUpgradeStatuses] = useState<Map<string, UpgradeStatusMessage>>(new Map());

  useEffect(() => {
    if (lastMessage && lastMessage.type === 'upgrade_status') {
      const statusMessage = lastMessage.payload as UpgradeStatusMessage;
      setUpgradeStatuses(prev => {
        const newMap = new Map(prev);
        newMap.set(statusMessage.coreVersion, statusMessage);
        return newMap;
      });
    }
  }, [lastMessage]);

  const getStatusForVersion = useCallback((coreVersion: string): UpgradeStatusMessage | undefined => {
    return upgradeStatuses.get(coreVersion);
  }, [upgradeStatuses]);

  return {
    isConnected,
    error,
    connectionStats,
    upgradeStatuses: Array.from(upgradeStatuses.values()),
    getStatusForVersion,
    sendMessage,
  };
};
