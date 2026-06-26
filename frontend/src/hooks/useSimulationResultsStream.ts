import { useCallback, useEffect, useRef, useState } from 'react';
import type {
  SimulationResult,
  SimulationStreamMessage,
} from '../types/scenarios';

/**
 * WebSocket connection manager for streaming simulation results
 */
class SimulationResultsStream {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 2000;
  private messageHandlers: Set<(msg: SimulationStreamMessage) => void> = new Set();
  private errorHandlers: Set<(err: Error) => void> = new Set();

  constructor(private simulationId: string) {}

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        const wsUrl = `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/api/v1/simulations/${this.simulationId}/stream`;
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          console.log('WebSocket connected for simulation', this.simulationId);
          this.reconnectAttempts = 0;
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message: SimulationStreamMessage = JSON.parse(event.data);
            this.messageHandlers.forEach(handler => handler(message));
          } catch (err) {
            console.error('Failed to parse WebSocket message:', err);
          }
        };

        this.ws.onerror = (event) => {
          const error = new Error(`WebSocket error for simulation ${this.simulationId}`);
          console.error(error);
          this.errorHandlers.forEach(handler => handler(error));
        };

        this.ws.onclose = () => {
          console.log('WebSocket closed for simulation', this.simulationId);
          if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            setTimeout(() => this.connect().catch(console.error), this.reconnectDelay);
          }
        };
      } catch (err) {
        reject(err instanceof Error ? err : new Error('WebSocket connection failed'));
      }
    });
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  onMessage(handler: (msg: SimulationStreamMessage) => void): void {
    this.messageHandlers.add(handler);
  }

  onError(handler: (err: Error) => void): void {
    this.errorHandlers.add(handler);
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

/**
 * Hook for streaming simulation results via WebSocket
 * Automatically connects to simulation stream and handles reconnection
 *
 * @param simulationId - ID of the running simulation
 * @param enabled - Whether to enable WebSocket connection
 *
 * @example
 * const { results, isConnected, error } = useSimulationResultsStream(
 *   simulationId,
 *   isSimulating
 * );
 * // Results update in real-time as they arrive from backend
 */
export function useSimulationResultsStream(
  simulationId: string | null,
  enabled: boolean = true
) {
  const [results, setResults] = useState<SimulationResult[]>([]);
  const [progress, setProgress] = useState(0);
  const [totalPortfolios, setTotalPortfolios] = useState(0);
  const [processedPortfolios, setProcessedPortfolios] = useState(0);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const streamRef = useRef<SimulationResultsStream | null>(null);
  const connectionTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (!simulationId || !enabled) {
      return;
    }

    // Create and connect WebSocket stream
    const stream = new SimulationResultsStream(simulationId);
    streamRef.current = stream;

    const handleMessage = (message: SimulationStreamMessage) => {
      switch (message.type) {
        case 'progress':
          if (message.progress !== undefined) {
            setProgress(message.progress);
          }
          if (message.portfoliosProcessed !== undefined) {
            setProcessedPortfolios(message.portfoliosProcessed);
          }
          if (message.totalPortfolios !== undefined) {
            setTotalPortfolios(message.totalPortfolios);
          }
          break;

        case 'result':
          if (message.result) {
            setResults(prev => {
              const exists = prev.find(r => r.id === message.result!.id);
              if (exists) {
                return prev.map(r => (r.id === message.result!.id ? message.result! : r));
              }
              return [...prev, message.result];
            });
          }
          break;

        case 'complete':
          setProgress(100);
          break;

        case 'error':
          if (message.error) {
            setError(new Error(message.error));
          }
          break;

        default:
          console.warn('Unknown stream message type:', (message as any).type);
      }
    };

    const handleError = (err: Error) => {
      setError(err);
    };

    stream.onMessage(handleMessage);
    stream.onError(handleError);

    // Connect with timeout
    connectionTimeoutRef.current = setTimeout(() => {
      stream
        .connect()
        .then(() => setIsConnected(true))
        .catch(err => {
          setError(err);
          setIsConnected(false);
        });
    }, 100);

    // Cleanup on unmount
    return () => {
      if (connectionTimeoutRef.current) {
        clearTimeout(connectionTimeoutRef.current);
      }
      stream.disconnect();
      streamRef.current = null;
    };
  }, [simulationId, enabled]);

  // Manually disconnect
  const disconnect = useCallback(() => {
    if (streamRef.current) {
      streamRef.current.disconnect();
      setIsConnected(false);
    }
  }, []);

  return {
    results,
    progress,
    totalPortfolios,
    processedPortfolios,
    isConnected,
    error,
    disconnect,
  };
}

export type UseSimulationResultsStreamReturn = ReturnType<typeof useSimulationResultsStream>;
