/**
 * NBA (Next Best Action) React Hooks
 * 
 * Provides data fetching, state management, and WebSocket subscriptions
 * for the AI-Driven Next Best Action Engine.
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { fetchAPI } from '../api';

// ============================================================================
// Types
// ============================================================================

export interface ActionTemplate {
  email_subject?: string;
  email_body?: string;
  call_script?: string;
  meeting_agenda?: string;
  [key: string]: unknown;
}

export interface NextBestAction {
  action_id: string;
  client_id: string;
  client_name: string;
  action_type: string;
  action_name: string;
  confidence: number;
  urgency_score: number;
  expected_value: number;
  success_probability: number;
  trigger_signal: string;
  reasoning: string;
  recommended_channel: string;
  estimated_duration_minutes: number;
  template_content: ActionTemplate;
  recommended_at?: string;
}

export interface ClientSignal {
  signal_id: string;
  client_id: string;
  signal_type: string;
  signal_value: number;
  metadata: Record<string, unknown>;
  detected_at: string;
  expires_at?: string;
  processed: boolean;
}

export interface ActionCatalogItem {
  action_id: string;
  action_code: string;
  action_name: string;
  category: string;
  description: string;
  default_channel: string;
  estimated_duration_minutes: number;
  estimated_revenue_impact: number;
  required_signals: string[];
  template_content: ActionTemplate;
  active: boolean;
}

export interface OutcomeStats {
  total_executed: number;
  total_completed: number;
  total_dismissed: number;
  avg_completion_rate: number;
  avg_revenue_generated: number;
  by_action_type: Record<string, {
    executed: number;
    completed: number;
    dismissed: number;
    avg_revenue: number;
  }>;
}

export interface ExecuteActionRequest {
  action_id: string;
  advisor_id?: string;
  notes?: string;
}

export interface CompleteActionRequest {
  action_id: string;
  outcome: 'SUCCESS' | 'PARTIAL' | 'FAILED';
  revenue_generated?: number;
  notes?: string;
}

export interface DismissActionRequest {
  action_id: string;
  reason?: string;
}

export interface NBAWebSocketMessage {
  type: 'new_recommendation' | 'signal_detected' | 'action_updated' | 'pong';
  payload: NextBestAction | ClientSignal | { action_id: string; status: string };
  timestamp: string;
}

// ============================================================================
// useNBARecommendations - Main recommendations hook
// ============================================================================

export interface UseNBARecommendationsOptions {
  advisorId?: string;
  clientId?: string;
  autoRefresh?: boolean;
  refreshInterval?: number; // ms
}

export interface UseNBARecommendationsResult {
  recommendations: NextBestAction[];
  loading: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
  executeAction: (actionId: string, notes?: string) => Promise<void>;
  completeAction: (request: CompleteActionRequest) => Promise<void>;
  dismissAction: (actionId: string, reason?: string) => Promise<void>;
}

export function useNBARecommendations(
  options: UseNBARecommendationsOptions = {}
): UseNBARecommendationsResult {
  const { advisorId, clientId, autoRefresh = false, refreshInterval = 30000 } = options;
  
  const [recommendations, setRecommendations] = useState<NextBestAction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const refreshTimerRef = useRef<NodeJS.Timeout | null>(null);

  const fetchRecommendations = useCallback(async () => {
    try {
      setError(null);
      const params = new URLSearchParams();
      if (advisorId) params.set('advisor_id', advisorId);
      if (clientId) params.set('client_id', clientId);
      
      const queryString = params.toString();
      const url = `/nba/recommendations${queryString ? `?${queryString}` : ''}`;
      
      const data = await fetchAPI<NextBestAction[]>(url);
      setRecommendations(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch recommendations'));
    } finally {
      setLoading(false);
    }
  }, [advisorId, clientId]);

  const refresh = useCallback(async () => {
    setLoading(true);
    await fetchRecommendations();
  }, [fetchRecommendations]);

  const executeAction = useCallback(async (actionId: string, notes?: string) => {
    try {
      await fetchAPI('/nba/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action_id: actionId, notes } as ExecuteActionRequest),
      });
      // Update local state
      setRecommendations(prev =>
        prev.map(r =>
          r.action_id === actionId
            ? { ...r, status: 'EXECUTING' as never }
            : r
        )
      );
    } catch (err) {
      throw err instanceof Error ? err : new Error('Failed to execute action');
    }
  }, []);

  const completeAction = useCallback(async (request: CompleteActionRequest) => {
    try {
      await fetchAPI('/nba/complete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });
      // Remove from local state (completed actions are no longer recommendations)
      setRecommendations(prev => prev.filter(r => r.action_id !== request.action_id));
    } catch (err) {
      throw err instanceof Error ? err : new Error('Failed to complete action');
    }
  }, []);

  const dismissAction = useCallback(async (actionId: string, reason?: string) => {
    try {
      await fetchAPI('/nba/dismiss', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action_id: actionId, reason } as DismissActionRequest),
      });
      // Remove from local state
      setRecommendations(prev => prev.filter(r => r.action_id !== actionId));
    } catch (err) {
      throw err instanceof Error ? err : new Error('Failed to dismiss action');
    }
  }, []);

  // Initial load
  useEffect(() => {
    fetchRecommendations();
  }, [fetchRecommendations]);

  // Auto-refresh
  useEffect(() => {
    if (autoRefresh && refreshInterval > 0) {
      refreshTimerRef.current = setInterval(fetchRecommendations, refreshInterval);
      return () => {
        if (refreshTimerRef.current) {
          clearInterval(refreshTimerRef.current);
        }
      };
    }
  }, [autoRefresh, refreshInterval, fetchRecommendations]);

  return {
    recommendations,
    loading,
    error,
    refresh,
    executeAction,
    completeAction,
    dismissAction,
  };
}

// ============================================================================
// useNBASignals - Client signals hook
// ============================================================================

export interface UseNBASignalsOptions {
  clientId?: string;
  signalTypes?: string[];
  limit?: number;
}

export interface UseNBASignalsResult {
  signals: ClientSignal[];
  loading: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
}

export function useNBASignals(options: UseNBASignalsOptions = {}): UseNBASignalsResult {
  const { clientId, signalTypes, limit = 100 } = options;
  
  const [signals, setSignals] = useState<ClientSignal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchSignals = useCallback(async () => {
    try {
      setError(null);
      const params = new URLSearchParams();
      if (clientId) params.set('client_id', clientId);
      if (signalTypes?.length) params.set('signal_types', signalTypes.join(','));
      params.set('limit', limit.toString());
      
      const data = await fetchAPI<ClientSignal[]>(`/nba/signals?${params.toString()}`);
      setSignals(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch signals'));
    } finally {
      setLoading(false);
    }
  }, [clientId, signalTypes, limit]);

  const refresh = useCallback(async () => {
    setLoading(true);
    await fetchSignals();
  }, [fetchSignals]);

  useEffect(() => {
    fetchSignals();
  }, [fetchSignals]);

  return { signals, loading, error, refresh };
}

// ============================================================================
// useNBACatalog - Action catalog hook
// ============================================================================

export interface UseNBACatalogResult {
  catalog: ActionCatalogItem[];
  loading: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
}

export function useNBACatalog(): UseNBACatalogResult {
  const [catalog, setCatalog] = useState<ActionCatalogItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchCatalog = useCallback(async () => {
    try {
      setError(null);
      const data = await fetchAPI<ActionCatalogItem[]>('/nba/catalog');
      setCatalog(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch catalog'));
    } finally {
      setLoading(false);
    }
  }, []);

  const refresh = useCallback(async () => {
    setLoading(true);
    await fetchCatalog();
  }, [fetchCatalog]);

  useEffect(() => {
    fetchCatalog();
  }, [fetchCatalog]);

  return { catalog, loading, error, refresh };
}

// ============================================================================
// useNBAStats - Outcome statistics hook
// ============================================================================

export interface UseNBAStatsOptions {
  advisorId?: string;
  startDate?: string;
  endDate?: string;
}

export interface UseNBAStatsResult {
  stats: OutcomeStats | null;
  loading: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
}

export function useNBAStats(options: UseNBAStatsOptions = {}): UseNBAStatsResult {
  const { advisorId, startDate, endDate } = options;
  
  const [stats, setStats] = useState<OutcomeStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchStats = useCallback(async () => {
    try {
      setError(null);
      const params = new URLSearchParams();
      if (advisorId) params.set('advisor_id', advisorId);
      if (startDate) params.set('start_date', startDate);
      if (endDate) params.set('end_date', endDate);
      
      const queryString = params.toString();
      const url = `/nba/stats${queryString ? `?${queryString}` : ''}`;
      
      const data = await fetchAPI<OutcomeStats>(url);
      setStats(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to fetch stats'));
    } finally {
      setLoading(false);
    }
  }, [advisorId, startDate, endDate]);

  const refresh = useCallback(async () => {
    setLoading(true);
    await fetchStats();
  }, [fetchStats]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  return { stats, loading, error, refresh };
}

// ============================================================================
// useNBAWebSocket - Real-time WebSocket subscription
// ============================================================================

export interface UseNBAWebSocketOptions {
  advisorId?: string;
  onNewRecommendation?: (action: NextBestAction) => void;
  onSignalDetected?: (signal: ClientSignal) => void;
  onActionUpdated?: (update: { action_id: string; status: string }) => void;
  autoReconnect?: boolean;
  maxReconnectAttempts?: number;
}

export interface UseNBAWebSocketResult {
  isConnected: boolean;
  error: Error | null;
  lastMessage: NBAWebSocketMessage | null;
  connectionStats: {
    connectedAt: number;
    messagesReceived: number;
    reconnectAttempts: number;
  };
  connect: () => void;
  disconnect: () => void;
}

export function useNBAWebSocket(
  options: UseNBAWebSocketOptions = {}
): UseNBAWebSocketResult {
  const {
    advisorId,
    onNewRecommendation,
    onSignalDetected,
    onActionUpdated,
    autoReconnect = true,
    maxReconnectAttempts = 10,
  } = options;

  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [lastMessage, setLastMessage] = useState<NBAWebSocketMessage | null>(null);
  const [connectionStats, setConnectionStats] = useState({
    connectedAt: 0,
    messagesReceived: 0,
    reconnectAttempts: 0,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const pingIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const getWebSocketUrl = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const params = advisorId ? `?advisor_id=${advisorId}` : '';
    return `${protocol}//${host}/api/nba/stream${params}`;
  }, [advisorId]);

  const cleanup = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (pingIntervalRef.current) {
      clearInterval(pingIntervalRef.current);
      pingIntervalRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  const connect = useCallback(() => {
    cleanup();
    
    try {
      const url = getWebSocketUrl();
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        setIsConnected(true);
        setError(null);
        reconnectAttemptsRef.current = 0;
        setConnectionStats(prev => ({
          ...prev,
          connectedAt: Date.now(),
          reconnectAttempts: 0,
        }));

        // Start ping interval to keep connection alive
        pingIntervalRef.current = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'ping', timestamp: new Date().toISOString() }));
          }
        }, 30000);
      };

      ws.onmessage = (event) => {
        try {
          const message: NBAWebSocketMessage = JSON.parse(event.data);
          setLastMessage(message);
          setConnectionStats(prev => ({
            ...prev,
            messagesReceived: prev.messagesReceived + 1,
          }));

          // Route message to appropriate callback
          switch (message.type) {
            case 'new_recommendation':
              onNewRecommendation?.(message.payload as NextBestAction);
              break;
            case 'signal_detected':
              onSignalDetected?.(message.payload as ClientSignal);
              break;
            case 'action_updated':
              onActionUpdated?.(message.payload as { action_id: string; status: string });
              break;
            case 'pong':
              // Heartbeat response, nothing to do
              break;
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      ws.onerror = (event) => {
        setError(new Error('WebSocket connection error'));
        console.error('NBA WebSocket error:', event);
      };

      ws.onclose = (_event) => {
        setIsConnected(false);
        if (pingIntervalRef.current) {
          clearInterval(pingIntervalRef.current);
          pingIntervalRef.current = null;
        }

        // Attempt reconnection if enabled
        if (autoReconnect && reconnectAttemptsRef.current < maxReconnectAttempts) {
          reconnectAttemptsRef.current++;
          const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
          
          setConnectionStats(prev => ({
            ...prev,
            reconnectAttempts: reconnectAttemptsRef.current,
          }));

          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, delay);
        }
      };
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to create WebSocket'));
    }
  }, [cleanup, getWebSocketUrl, autoReconnect, maxReconnectAttempts, onNewRecommendation, onSignalDetected, onActionUpdated]);

  const disconnect = useCallback(() => {
    autoReconnect && (reconnectAttemptsRef.current = maxReconnectAttempts); // Prevent auto-reconnect
    cleanup();
    setIsConnected(false);
  }, [cleanup, autoReconnect, maxReconnectAttempts]);

  // Auto-connect on mount
  useEffect(() => {
    connect();
    return () => {
      cleanup();
    };
  }, [connect, cleanup]);

  return {
    isConnected,
    error,
    lastMessage,
    connectionStats,
    connect,
    disconnect,
  };
}

// ============================================================================
// useNBAIntegrated - Combined hook with WebSocket updates
// ============================================================================

export interface UseNBAIntegratedOptions extends UseNBARecommendationsOptions {
  enableWebSocket?: boolean;
}

export interface UseNBAIntegratedResult extends UseNBARecommendationsResult {
  wsConnected: boolean;
  wsError: Error | null;
}

export function useNBAIntegrated(
  options: UseNBAIntegratedOptions = {}
): UseNBAIntegratedResult {
  const { enableWebSocket = true, ...recommendationsOptions } = options;
  
  const {
    recommendations,
    loading,
    error,
    refresh,
    executeAction,
    completeAction,
    dismissAction,
  } = useNBARecommendations(recommendationsOptions);

  const [localRecommendations, setLocalRecommendations] = useState<NextBestAction[]>([]);

  // Sync recommendations from API
  useEffect(() => {
    setLocalRecommendations(recommendations);
  }, [recommendations]);

  // Handle WebSocket updates
  const handleNewRecommendation = useCallback((action: NextBestAction) => {
    setLocalRecommendations(prev => {
      // Avoid duplicates
      if (prev.some(r => r.action_id === action.action_id)) {
        return prev;
      }
      // Add new recommendation at the top
      return [action, ...prev];
    });
  }, []);

  const handleActionUpdated = useCallback((update: { action_id: string; status: string }) => {
    if (update.status === 'COMPLETED' || update.status === 'DISMISSED') {
      setLocalRecommendations(prev => prev.filter(r => r.action_id !== update.action_id));
    }
  }, []);

  const {
    isConnected: wsConnected,
    error: wsError,
  } = useNBAWebSocket(
    enableWebSocket
      ? {
          advisorId: recommendationsOptions.advisorId,
          onNewRecommendation: handleNewRecommendation,
          onActionUpdated: handleActionUpdated,
        }
      : { autoReconnect: false }
  );

  return {
    recommendations: localRecommendations,
    loading,
    error,
    refresh,
    executeAction,
    completeAction,
    dismissAction,
    wsConnected,
    wsError,
  };
}

export default {
  useNBARecommendations,
  useNBASignals,
  useNBACatalog,
  useNBAStats,
  useNBAWebSocket,
  useNBAIntegrated,
};
