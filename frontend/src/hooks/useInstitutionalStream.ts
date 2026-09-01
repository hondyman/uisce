import { useState, useEffect, useRef, useCallback } from 'react';

export interface StreamEventEnvelope {
  type: 'IBOR_POSITION_TICK' | 'ABOR_JOURNAL_POSTED' | 'TAX_LOSS_HARVESTED' | 'ZK_PROOF_VERIFIED' | 'RECON_BREAK_DETECTED';
  tenant_id: string;
  timestamp: string;
  payload: any;
}

export const useInstitutionalStream = (tenantId?: string) => {
  const [latestEvent, setLatestEvent] = useState<StreamEventEnvelope | null>(null);
  const [isConnected, setIsConnected] = useState<boolean>(false);
  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<NodeJS.Timeout | null>(null);

  const connect = useCallback(() => {
    if (!tenantId) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/api/v1/stream?tenant_id=${tenantId}`;

    const ws = new WebSocket(wsUrl);
    socketRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    };

    ws.onmessage = (event) => {
      try {
        const parsed: StreamEventEnvelope = JSON.parse(event.data);
        setLatestEvent(parsed);
      } catch (err) {
        console.error('[WebSocket] Malformed envelope received:', err);
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
      reconnectTimerRef.current = setTimeout(() => {
        connect();
      }, 3000);
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [tenantId]);

  useEffect(() => {
    connect();
    return () => {
      if (socketRef.current) {
        socketRef.current.close();
      }
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
    };
  }, [connect]);

  return { latestEvent, isConnected };
};
