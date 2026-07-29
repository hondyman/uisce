import React, { useEffect, useState, useRef } from 'react';

interface DriftCount {
  tenant_id: string;
  count: number;
  last_seen: string;
  job_id?: string;
  severity?: string;
  description?: string;
}

interface DriftBadgeProps {
  tenantId: string;
  className?: string;
}

const DriftBadge: React.FC<DriftBadgeProps> = ({ tenantId, className = '' }) => {
  const [driftCount, setDriftCount] = useState<number>(0);
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    let mounted = true;

    const connect = () => {
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${proto}//${window.location.host}/api/v1/ws/drift?tenant_id=${encodeURIComponent(tenantId)}`;

      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        if (mounted) setIsConnected(true);
      };

      ws.onmessage = (event) => {
        try {
          const data: DriftCount = JSON.parse(event.data);
          if (data.tenant_id === tenantId) {
            setDriftCount(data.count);
          }
        } catch {
          // ignore parse errors
        }
      };

      ws.onclose = () => {
        if (mounted) {
          setIsConnected(false);
          reconnectTimer.current = setTimeout(connect, 5000);
        }
      };

      ws.onerror = () => {
        ws.close();
      };
    };

    connect();

    return () => {
      mounted = false;
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      if (wsRef.current) wsRef.current.close();
    };
  }, [tenantId]);

  if (driftCount === 0) return null;

  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800 ${className}`}>
      <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
      {driftCount} drift{driftCount !== 1 ? 's' : ''}
      {isConnected && <span className="w-1.5 h-1.5 rounded-full bg-green-500" title="Live" />}
    </span>
  );
};

export default DriftBadge;
