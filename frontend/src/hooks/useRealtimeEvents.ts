import { useEffect, useRef, useCallback, useState } from 'react';
import { devDebug } from '../utils/devLogger';

/**
 * Phase 3.4: Frontend WebSocket Integration Hook
 * Real-time event streaming with auto-reconnect and backpressure handling
 */

export enum EventType {
  IncidentDetected = 'incident.detected',
  IncidentUpdated = 'incident.updated',
  IncidentResolved = 'incident.resolved',
  RCAStarted = 'rca.started',
  RCACompleted = 'rca.completed',
  RCAResultsAvailable = 'rca.results',
  ActionPlanned = 'action.planned',
  ActionStarted = 'action.started',
  ActionCompleted = 'action.completed',
  ActionFailed = 'action.failed',
  RegionFailover = 'region.failover',
  PropagationDetected = 'propagation.detected',
  PropagationBlocked = 'propagation.blocked',
}

export interface StreamedEvent {
  id: string;
  type: EventType;
  timestamp: string;
  tenant_id: string;
  incident_id?: string;
  region?: string;
  severity?: string;
  payload: Record<string, any>;
}

export interface ConnectionState {
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;
  reconnectAttempt: number;
  lastEventTime: number | null;
}

interface UseRealtimeEventsOptions {
  tenantId: string;
  regions?: string[];
  onError?: (error: string) => void;
  onEvent?: (event: StreamedEvent) => void;
  reconnectMaxAttempts?: number;
  reconnectDelayMs?: number;
}

/**
 * Hook for real-time event streaming via WebSocket
 * Handles connection management, auto-reconnect, and backpressure
 */
export function useRealtimeEvents(options: UseRealtimeEventsOptions) {
  const {
    tenantId,
    regions = [],
    onError,
    onEvent,
    reconnectMaxAttempts = 5,
    reconnectDelayMs = 1000,
  } = options;

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const heartbeatIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const eventQueueRef = useRef<StreamedEvent[]>([]);
  const isProcessingRef = useRef(false);

  const [state, setState] = useState<ConnectionState>({
    isConnected: false,
    isConnecting: false,
    error: null,
    reconnectAttempt: 0,
    lastEventTime: null,
  });

  // Build WebSocket URL
  const wsUrl = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const regionsParam = regions.length > 0 ? `&regions=${regions.join(',')}` : '';
    return `${protocol}//${window.location.host}/api/events?tenant_id=${tenantId}${regionsParam}`;
  }, [tenantId, regions]);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (state.isConnecting) {
      return;
    }

    setState(prev => ({ ...prev, isConnecting: true, error: null }));

    try {
      const ws = new WebSocket(wsUrl());

      ws.onopen = () => {
        devDebug('[WebSocket] Connected');
        setState(prev => ({
          ...prev,
          isConnected: true,
          isConnecting: false,
          reconnectAttempt: 0,
          error: null,
        }));

        // Start heartbeat
        heartbeatIntervalRef.current = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'ping' }));
          }
        }, 30000); // Send ping every 30 seconds
      };

      ws.onmessage = (event) => {
        try {
          const streamed: StreamedEvent = JSON.parse(event.data);

          // Queue event for processing
          eventQueueRef.current.push(streamed);
          setState(prev => ({ ...prev, lastEventTime: Date.now() }));

          // Process event queue
          processEventQueue();

          // Call user callback
          if (onEvent) {
            onEvent(streamed);
          }
        } catch (err) {
          console.error('[WebSocket] Failed to parse event:', err);
        }
      };

      ws.onerror = (err) => {
        console.error('[WebSocket] Error:', err);
        setState(prev => ({
          ...prev,
          error: 'WebSocket connection error',
          isConnected: false,
          isConnecting: false,
        }));

        if (onError) {
          onError('WebSocket connection error');
        }
      };

      ws.onclose = () => {
        devDebug('[WebSocket] Closed');
        clearInterval(heartbeatIntervalRef.current!);

        setState(prev => ({
          ...prev,
          isConnected: false,
          isConnecting: false,
        }));

        // Attempt reconnect
        if (state.reconnectAttempt < reconnectMaxAttempts) {
          const delay = reconnectDelayMs * Math.pow(2, state.reconnectAttempt);
          devDebug(`[WebSocket] Reconnecting in ${delay}ms (attempt ${state.reconnectAttempt + 1})`);

          reconnectTimeoutRef.current = setTimeout(() => {
            setState(prev => ({
              ...prev,
              reconnectAttempt: prev.reconnectAttempt + 1,
            }));
          }, delay);
        } else {
          const error = 'Max reconnect attempts reached';
          setState(prev => ({ ...prev, error }));
          if (onError) {
            onError(error);
          }
        }
      };

      wsRef.current = ws;
    } catch (err) {
      const error = err instanceof Error ? err.message : 'Unknown error';
      console.error('[WebSocket] Connection failed:', error);
      setState(prev => ({
        ...prev,
        error,
        isConnecting: false,
      }));

      if (onError) {
        onError(error);
      }
    }
  }, [wsUrl, state.isConnecting, state.reconnectAttempt, reconnectMaxAttempts, reconnectDelayMs, onError, onEvent]);

  // Process queued events
  const processEventQueue = useCallback(() => {
    if (isProcessingRef.current || eventQueueRef.current.length === 0) {
      return;
    }

    isProcessingRef.current = true;

    // Process events in microtask to avoid blocking
    queueMicrotask(() => {
      const batch = eventQueueRef.current.splice(0, 10); // Process 10 at a time

      for (const event of batch) {
        onEvent?.(event);
      }

      isProcessingRef.current = false;

      // Process remaining if available
      if (eventQueueRef.current.length > 0) {
        processEventQueue();
      }
    });
  }, [onEvent]);

  // Initialize connection
  useEffect(() => {
    connect();

    return () => {
      // Cleanup
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (heartbeatIntervalRef.current) {
        clearInterval(heartbeatIntervalRef.current);
      }
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.close();
      }
    };
  }, [connect]);

  // Trigger reconnect on state change
  useEffect(() => {
    if (!state.isConnected && !state.isConnecting && state.reconnectAttempt > 0) {
      connect();
    }
  }, [state.reconnectAttempt, connect, state.isConnected, state.isConnecting]);

  // Disconnect function
  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
    }
    setState({
      isConnected: false,
      isConnecting: false,
      error: null,
      reconnectAttempt: 0,
      lastEventTime: null,
    });
  }, []);

  return {
    state,
    connect,
    disconnect,
    eventQueue: eventQueueRef.current,
  };
}

/**
 * Hook for filtering and processing specific event types
 */
export function useEventListener(options: UseRealtimeEventsOptions & {
  eventTypes?: EventType[];
  batchSize?: number;
}) {
  const {
    eventTypes,
    batchSize = 20,
    ...realtimeOptions
  } = options;

  const [events, setEvents] = useState<StreamedEvent[]>([]);
  const [isProcessing, setIsProcessing] = useState(false);

  const handleEvent = useCallback((event: StreamedEvent) => {
    // Filter by event type if specified
    if (eventTypes && !eventTypes.includes(event.type)) {
      return;
    }

    setEvents(prev => {
      const updated = [event, ...prev];
      // Keep only recent events (last 100)
      return updated.slice(0, 100);
    });
  }, [eventTypes]);

  const { state, connect, disconnect } = useRealtimeEvents({
    ...realtimeOptions,
    onEvent: handleEvent,
  });

  // Clear old events
  const clearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  // Pause processing
  const pause = useCallback(() => {
    setIsProcessing(true);
    disconnect();
  }, [disconnect]);

  // Resume processing
  const resume = useCallback(() => {
    setIsProcessing(false);
    connect();
  }, [connect]);

  return {
    events,
    clearEvents,
    isProcessing,
    pause,
    resume,
    connectionState: state,
  };
}

/**
 * Hook for aggregated metrics from events
 */
export function useEventMetrics(options: UseRealtimeEventsOptions & {
  eventTypes?: EventType[];
}) {
  const [metrics, setMetrics] = useState({
    incidentsDetected: 0,
    rcasCompleted: 0,
    actionsExecuted: 0,
    propogationDetected: 0,
    failoversTriggered: 0,
    lastUpdate: Date.now(),
  });

  const handleEvent = useCallback((event: StreamedEvent) => {
    setMetrics(prev => {
      const updated = { ...prev, lastUpdate: Date.now() };

      switch (event.type) {
        case EventType.IncidentDetected:
          updated.incidentsDetected++;
          break;
        case EventType.RCACompleted:
        case EventType.RCAResultsAvailable:
          updated.rcasCompleted++;
          break;
        case EventType.ActionCompleted:
          updated.actionsExecuted++;
          break;
        case EventType.PropagationDetected:
          updated.propogationDetected++;
          break;
        case EventType.RegionFailover:
          updated.failoversTriggered++;
          break;
      }

      return updated;
    });
  }, []);

  const { state } = useRealtimeEvents({
    ...options,
    onEvent: handleEvent,
  });

  const resetMetrics = useCallback(() => {
    setMetrics({
      incidentsDetected: 0,
      rcasCompleted: 0,
      actionsExecuted: 0,
      propogationDetected: 0,
      failoversTriggered: 0,
      lastUpdate: Date.now(),
    });
  }, []);

  return {
    metrics,
    resetMetrics,
    connectionState: state,
  };
}
