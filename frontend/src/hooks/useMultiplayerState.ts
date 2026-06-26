import { useCallback, useEffect, useRef, useState } from 'react';
import type {
  CollaborationState,
  User,
} from '../types/scenarios';

/**
 * WebSocket connection for real-time collaboration state
 */
class CollaborationStateManager {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 2000;
  private stateHandlers: Set<(state: CollaborationState) => void> = new Set();
  private errorHandlers: Set<(err: Error) => void> = new Set();

  constructor(private simulationId: string, private userId: string) {}

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        const wsUrl = `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/api/v1/simulations/${this.simulationId}/collaborate`;
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          console.log('Collaboration WebSocket connected');
          this.reconnectAttempts = 0;

          // Send user info on connection
          this.ws?.send(
            JSON.stringify({
              type: 'user_joined',
              userId: this.userId,
              timestamp: new Date(),
            })
          );

          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data);
            if (message.type === 'collaboration_state') {
              const state: CollaborationState = message.payload;
              this.stateHandlers.forEach(handler => handler(state));
            }
          } catch (err) {
            console.error('Failed to parse collaboration message:', err);
          }
        };

        this.ws.onerror = (event) => {
          const error = new Error('Collaboration WebSocket error');
          console.error(error);
          this.errorHandlers.forEach(handler => handler(error));
        };

        this.ws.onclose = () => {
          console.log('Collaboration WebSocket closed');
          if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            setTimeout(() => this.connect().catch(console.error), this.reconnectDelay);
          }
        };
      } catch (err) {
        reject(err instanceof Error ? err : new Error('Connection failed'));
      }
    });
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.send(
        JSON.stringify({
          type: 'user_left',
          userId: this.userId,
          timestamp: new Date(),
        })
      );
      this.ws.close();
      this.ws = null;
    }
  }

  onStateChange(handler: (state: CollaborationState) => void): void {
    this.stateHandlers.add(handler);
  }

  onError(handler: (err: Error) => void): void {
    this.errorHandlers.add(handler);
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  // Signal user is viewing/editing a cell
  setActiveCells(cells: string[]): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(
        JSON.stringify({
          type: 'active_cells_update',
          userId: this.userId,
          cells,
          timestamp: new Date(),
        })
      );
    }
  }

  // Signal user preference/filter
  setActiveMetric(metric: string): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(
        JSON.stringify({
          type: 'metric_update',
          userId: this.userId,
          metric,
          timestamp: new Date(),
        })
      );
    }
  }
}

/**
 * Hook for managing real-time multiplayer collaboration state
 * Tracks active users, their viewing focus, and synchronized state
 *
 * @param simulationId - ID of the simulation for collaboration
 * @param userId - Current user ID
 * @param enabled - Whether to enable collaboration
 *
 * @example
 * const {
 *   collaborators,
 *   isConnected,
 *   error,
 *   setActiveCells,
 *   setActiveMetric,
 * } = useMultiplayerState(simulationId, currentUserId, enabled);
 *
 * // Track which cells user is viewing
 * setActiveCells(['portfolio1-pnl', 'portfolio2-pnl']);
 *
 * // Notify others of metric focus
 * setActiveMetric('variance');
 */
export function useMultiplayerState(
  simulationId: string | null,
  userId: string | null,
  enabled: boolean = true
) {
  const [collaborators, setCollaborators] = useState<User[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [activeCellsByUser, setActiveCellsByUser] = useState<Map<string, string[]>>(new Map());
  const [activeMetricByUser, setActiveMetricByUser] = useState<Map<string, string>>(new Map());

  const managerRef = useRef<CollaborationStateManager | null>(null);

  useEffect(() => {
    if (!simulationId || !userId || !enabled) {
      return;
    }

    // Create and connect collaboration manager
    const manager = new CollaborationStateManager(simulationId, userId);
    managerRef.current = manager;

    const handleStateChange = (state: CollaborationState) => {
      setCollaborators(state.activeUsers);
      
      // Update tracked cells/metrics from state
      const cellsMap = new Map<string, string[]>();
      const metricsMap = new Map<string, string>();
      
      state.activeUsers.forEach(user => {
        if (user.activeCells) {
          cellsMap.set(user.id, user.activeCells);
        }
        if (user.activeMetric) {
          metricsMap.set(user.id, user.activeMetric);
        }
      });

      setActiveCellsByUser(cellsMap);
      setActiveMetricByUser(metricsMap);
    };

    const handleError = (err: Error) => {
      setError(err);
    };

    manager.onStateChange(handleStateChange);
    manager.onError(handleError);

    // Connect with small delay to ensure DOM is ready
    const connectTimer = setTimeout(() => {
      manager
        .connect()
        .then(() => setIsConnected(true))
        .catch(err => {
          setError(err);
          setIsConnected(false);
        });
    }, 100);

    // Cleanup on unmount
    return () => {
      clearTimeout(connectTimer);
      if (managerRef.current) {
        managerRef.current.disconnect();
        managerRef.current = null;
      }
    };
  }, [simulationId, userId, enabled]);

  // Set user's active cells
  const setActiveCells = useCallback((cells: string[]) => {
    managerRef.current?.setActiveCells(cells);
  }, []);

  // Set user's active metric
  const setActiveMetric = useCallback((metric: string) => {
    managerRef.current?.setActiveMetric(metric);
  }, []);

  // Manually disconnect
  const disconnect = useCallback(() => {
    if (managerRef.current) {
      managerRef.current.disconnect();
      setIsConnected(false);
    }
  }, []);

  // Get active users excluding current user
  const otherUsers = collaborators.filter(u => u.id !== userId);

  // Check if specific user is viewing cell
  const isUserViewingCell = useCallback(
    (userId: string, cellId: string): boolean => {
      const cells = activeCellsByUser.get(userId);
      return cells?.includes(cellId) ?? false;
    },
    [activeCellsByUser]
  );

  // Get all users viewing specific cell
  const getUsersViewingCell = useCallback(
    (cellId: string): User[] => {
      return collaborators.filter(
        u => activeCellsByUser.get(u.id)?.includes(cellId) ?? false
      );
    },
    [collaborators, activeCellsByUser]
  );

  // Get cursor positions per user
  const getCursorPosition = useCallback(
    (userId: string): string | null => {
      return activeMetricByUser.get(userId) ?? null;
    },
    [activeMetricByUser]
  );

  return {
    collaborators,
    otherUsers,
    isConnected,
    error,
    setActiveCells,
    setActiveMetric,
    disconnect,
    isUserViewingCell,
    getUsersViewingCell,
    getCursorPosition,
  };
}

export type UseMultiplayerStateReturn = ReturnType<typeof useMultiplayerState>;
