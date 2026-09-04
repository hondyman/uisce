import { useEffect, useRef, useCallback, useState } from 'react';
import axios from '@/utils/axiosClient';
import type { PipelineExecutionRun } from '../types/pipeline';

export interface PipelineTelemetryEvent {
  type: 'snapshot' | 'step' | 'completion' | 'error';
  run?: PipelineExecutionRun;
  stepKey?: string;
  step?: Record<string, unknown>;
}

interface UsePipelineTelemetryOptions {
  runId: string;
  token: string;
  baseUrl?: string;
  onEvent?: (event: PipelineTelemetryEvent) => void;
  onError?: (error: Error) => void;
  maxReconnectAttempts?: number;
}

const DEFAULT_MAX_RECONNECT_ATTEMPTS = 5;

export function usePipelineTelemetry({
  runId,
  token,
  baseUrl = '',
  onEvent,
  onError,
  maxReconnectAttempts = DEFAULT_MAX_RECONNECT_ATTEMPTS,
}: UsePipelineTelemetryOptions) {
  const esRef = useRef<EventSource | null>(null);
  const reconnectAttempts = useRef(0);
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [connected, setConnected] = useState(false);

  const connect = useCallback(() => {
    if (esRef.current) {
      esRef.current.close();
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
        const hasSteps = data.step_telemetry && Object.keys(data.step_telemetry).length > 0;
        const event: PipelineTelemetryEvent = {
          type: data.status === 'completed' || data.status === 'failed' ? 'completion' : hasSteps ? 'step' : 'snapshot',
          run: data,
        };
        onEvent?.(event);
      } catch {
        onError?.(new Error(`Failed to parse SSE message: ${e.data}`));
      }
    };

    es.onerror = () => {
      setConnected(false);
      es.close();

      if (reconnectAttempts.current < maxReconnectAttempts) {
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 30000);
        reconnectAttempts.current++;
        reconnectTimeout.current = setTimeout(() => {
          connect();
        }, delay);
      } else {
        onError?.(new Error('Max reconnection attempts reached'));
      }
    };
  }, [runId, token, baseUrl, maxReconnectAttempts, onEvent, onError]);

  useEffect(() => {
    if (!runId || !token) return;
    connect();
    return () => {
      if (reconnectTimeout.current) clearTimeout(reconnectTimeout.current);
      if (esRef.current) esRef.current.close();
    };
  }, [connect, runId, token]);

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
