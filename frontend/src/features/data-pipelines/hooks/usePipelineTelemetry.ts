import { useEffect, useRef, useState } from 'react';
import axios from '@/utils/axiosClient';
import type { PipelineExecutionRun } from '../types/pipeline';

export interface PipelineTelemetryEvent {
  type: 'snapshot' | 'step' | 'completion';
  run?: PipelineExecutionRun;
}

interface UsePipelineTelemetryOptions {
  runId: string;
  baseUrl?: string;
  onEvent?: (event: PipelineTelemetryEvent) => void;
  onError?: (error: Error) => void;
  maxReconnectAttempts?: number;
}

const DEFAULT_MAX_RECONNECT_ATTEMPTS = 5;

export function usePipelineTelemetry({
  runId,
  baseUrl = '',
  onEvent,
  onError,
  maxReconnectAttempts = DEFAULT_MAX_RECONNECT_ATTEMPTS,
}: UsePipelineTelemetryOptions) {
  const esRef = useRef<EventSource | null>(null);
  const reconnectAttempts = useRef(0);
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isRunTerminal = useRef(false);
  const [connected, setConnected] = useState(false);

  const clearReconnectTimeout = () => {
    if (reconnectTimeout.current !== null) {
      clearTimeout(reconnectTimeout.current);
      reconnectTimeout.current = null;
    }
  };

  const closeEventSource = () => {
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }
  };

  const connect = async () => {
    if (!runId) return;

    closeEventSource();
    clearReconnectTimeout();

    let token: string;
    try {
      const tokenRes = await axios.post<{ token: string }>(
        `/api/v1/data-pipelines/runs/${runId}/stream-token`,
        { expires_in: 60 }
      );
      token = tokenRes.data.token;
    } catch (err) {
      onError?.(new Error(`Failed to mint stream token: ${err}`));
      return;
    }

    const url = `${baseUrl}/api/v1/data-pipelines/runs/${runId}/telemetry?token=${encodeURIComponent(token)}`;
    const es = new EventSource(url);
    esRef.current = es;

    es.onopen = () => {
      setConnected(true);
      reconnectAttempts.current = 0;
    };

    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data) as PipelineExecutionRun;

        if (data.status === 'completed' || data.status === 'failed') {
          isRunTerminal.current = true;
          onEvent?.({ type: 'completion', run: data });
          closeEventSource();
          return;
        }

        const hasSteps =
          data.step_telemetry && Object.keys(data.step_telemetry).length > 0;
        onEvent?.({
          type: hasSteps ? 'step' : 'snapshot',
          run: data,
        });
      } catch {
        onError?.(new Error(`Failed to parse SSE message: ${e.data}`));
      }
    };

    es.onerror = () => {
      setConnected(false);
      closeEventSource();

      if (isRunTerminal.current) return;
      if (reconnectAttempts.current >= maxReconnectAttempts) {
        onError?.(new Error('Max reconnection attempts reached'));
        return;
      }

      const delay = Math.min(
        1000 * Math.pow(2, reconnectAttempts.current),
        30000
      );
      reconnectAttempts.current++;
      reconnectTimeout.current = setTimeout(connect, delay);
    };
  };

  useEffect(() => {
    if (!runId) return;
    isRunTerminal.current = false;
    reconnectAttempts.current = 0;
    connect();

    return () => {
      clearReconnectTimeout();
      closeEventSource();
    };
  }, [runId, baseUrl]);

  return { connected };
}

export async function createStreamToken(
  runId: string,
  expiresIn = 60
): Promise<{ token: string; expires_at: string }> {
  const res = await axios.post(`/api/v1/data-pipelines/runs/${runId}/stream-token`, {
    expires_in: expiresIn,
  });
  return res.data;
}
