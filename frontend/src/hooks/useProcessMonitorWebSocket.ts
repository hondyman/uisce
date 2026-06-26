import { useState, useEffect, useRef, useCallback } from 'react';

export interface ProcessEvent {
  type: string; // step_started, step_completed, step_failed, workflow_started, workflow_completed
  workflow_id: string;
  workflow_type: string;
  step_name?: string;
  status: string; // running, completed, failed
  timestamp: string;
  tenant_id: string;
  tenant_instance_id: string;
  metadata?: Record<string, any>;
}

export interface UseProcessMonitorWSOptions {
  tenantId: string;
  datasourceId: string;
  filters?: {
    workflow_type?: string;
    status?: string;
  };
  onEvent?: (event: ProcessEvent) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  autoReconnect?: boolean;
  reconnectInterval?: number; // milliseconds
}

export interface UseProcessMonitorWSReturn {
  isConnected: boolean;
  events: ProcessEvent[];
  lastEvent: ProcessEvent | null;
  sendMessage: (message: any) => void;
  updateFilters: (filters: Record<string, string>) => void;
  clearEvents: () => void;
  disconnect: () => void;
  reconnect: () => void;
}

export function useProcessMonitorWebSocket(options: UseProcessMonitorWSOptions): UseProcessMonitorWSReturn {
  const {
    tenantId,
    datasourceId,
    filters = {},
    onEvent,
    onConnect,
    onDisconnect,
    onError,
    autoReconnect = true,
    reconnectInterval = 3000,
  } = options;

  const [isConnected, setIsConnected] = useState(false);
  const [events, setEvents] = useState<ProcessEvent[]>([]);
  const [lastEvent, setLastEvent] = useState<ProcessEvent | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const shouldReconnectRef = useRef(true);

  const connect = useCallback(() => {
    if (!tenantId || !datasourceId) {
      console.warn('Cannot connect: tenantId or datasourceId missing');
      return;
    }

    try {
      // Determine WebSocket URL based on current location
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.hostname;
      const port = window.location.port || (protocol === 'wss:' ? '443' : '80');
      
      // Use port 8080 for backend in development
      const wsPort = process.env.NODE_ENV === 'development' ? '8080' : port;
      const wsUrl = `${protocol}//${host}:${wsPort}/api/process-monitor/ws?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`;

      console.log('Connecting to Process Monitor WebSocket:', wsUrl);
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('Process Monitor WebSocket connected');
        setIsConnected(true);
        shouldReconnectRef.current = true;
        
        // Send initial filters if any
        if (Object.keys(filters).length > 0) {
          ws.send(JSON.stringify({
            type: 'update_filters',
            filters,
          }));
        }

        onConnect?.();
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          
          // Handle connection confirmation
          if (data.type === 'connected') {
            console.log('WebSocket connection confirmed:', data.message);
            return;
          }

          // Handle process events
          const processEvent: ProcessEvent = data;
          setLastEvent(processEvent);
          setEvents((prev) => [...prev.slice(-99), processEvent]); // Keep last 100 events
          
          onEvent?.(processEvent);
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      ws.onerror = (error) => {
        console.error('Process Monitor WebSocket error:', error);
        onError?.(error);
      };

      ws.onclose = () => {
        console.log('Process Monitor WebSocket disconnected');
        setIsConnected(false);
        wsRef.current = null;
        onDisconnect?.();

        // Auto-reconnect if enabled
        if (autoReconnect && shouldReconnectRef.current) {
          console.log(`Reconnecting in ${reconnectInterval}ms...`);
          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, reconnectInterval);
        }
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('Error creating WebSocket:', error);
    }
  }, [tenantId, datasourceId, filters, autoReconnect, reconnectInterval, onConnect, onDisconnect, onError, onEvent]);

  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false;
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  const reconnect = useCallback(() => {
    disconnect();
    shouldReconnectRef.current = true;
    setTimeout(connect, 100);
  }, [connect, disconnect]);

  const sendMessage = useCallback((message: any) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket not connected, cannot send message');
    }
  }, []);

  const updateFilters = useCallback((newFilters: Record<string, string>) => {
    sendMessage({
      type: 'update_filters',
      filters: newFilters,
    });
  }, [sendMessage]);

  const clearEvents = useCallback(() => {
    setEvents([]);
    setLastEvent(null);
  }, []);

  // Connect on mount
  useEffect(() => {
    connect();

    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return {
    isConnected,
    events,
    lastEvent,
    sendMessage,
    updateFilters,
    clearEvents,
    disconnect,
    reconnect,
  };
}
