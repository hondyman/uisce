import { useState, useEffect, useRef, useCallback } from 'react';
import { devLog, devWarn, devError } from '../utils/devLogger';

export interface WebSocketMessage {
  type: string;
  data: any;
  timestamp: string;
  userId?: string;
}

export interface RealTimeData {
  [fundId: string]: {
    metrics: any;
    lastUpdate: string;
  };
}

export const useWebSocket = (audience: string, userId?: string) => {
  const [isConnected, setIsConnected] = useState(false);
  const [realTimeData, setRealTimeData] = useState<RealTimeData>({});
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected');
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>();
  const reconnectAttempts = useRef(0);
  const maxReconnectAttempts = 5;

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    setConnectionStatus('connecting');

  const _wsBackend = (import.meta.env.VITE_API_BASE_URL as string) || 'http://localhost:29080';
  const wsUrl = `ws://localhost:29080/api/ws?audience=${audience}${userId ? `&userId=${userId}` : ''}`;
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      devLog('WebSocket connected');
      setIsConnected(true);
      setConnectionStatus('connected');
      reconnectAttempts.current = 0;
    };

    ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);
  devLog('WebSocket message received:', message);

        switch (message.type) {
          case 'fund_update':
            setRealTimeData(prev => ({
              ...prev,
              [message.data.fundId]: {
                metrics: message.data.metrics,
                lastUpdate: message.timestamp
              }
            }));
            break;
      case 'connection_ack':
      devLog('Connection acknowledged by server');
            break;
          case 'error':
      devError('WebSocket error:', message.data);
            break;
          default:
      devWarn('Unknown message type:', message.type);
        }
      } catch (error) {
    devError('Failed to parse WebSocket message:', error);
      }
    };

    ws.onclose = (event) => {
      devLog('WebSocket disconnected:', event.code, event.reason);
      setIsConnected(false);
      setConnectionStatus('disconnected');
      wsRef.current = null;

      // Attempt to reconnect if not a normal closure
      if (event.code !== 1000 && reconnectAttempts.current < maxReconnectAttempts) {
        reconnectAttempts.current++;
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 30000);
    devLog(`Attempting to reconnect in ${delay}ms (attempt ${reconnectAttempts.current}/${maxReconnectAttempts})`);

        reconnectTimeoutRef.current = setTimeout(() => {
          connect();
        }, delay);
      }
    };

    ws.onerror = (error) => {
      devError('WebSocket error:', error);
      setConnectionStatus('error');
    };

    wsRef.current = ws;
  }, [audience, userId]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    if (wsRef.current) {
      wsRef.current.close(1000, 'Client disconnecting');
      wsRef.current = null;
    }
    setIsConnected(false);
    setConnectionStatus('disconnected');
  }, []);

  const sendMessage = useCallback((message: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    } else {
      devWarn('WebSocket is not connected. Message not sent:', message);
    }
  }, []);

  useEffect(() => {
    connect();

    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return {
    isConnected,
    connectionStatus,
    realTimeData,
    sendMessage,
    reconnect: connect,
    disconnect
  };
};
