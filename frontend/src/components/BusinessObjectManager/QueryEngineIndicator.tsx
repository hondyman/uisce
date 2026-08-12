import React from 'react';

export type EngineRouteState = 'OLAP' | 'OLTP' | 'HYBRID_SEAM' | 'FEDERATED';

export interface QueryEngineIndicatorProps {
  state: EngineRouteState;
  estimatedLatencyMs: number;
  estimatedRows: number;
  hasWatermarkCrossed?: boolean;
  asOfDate?: string;
}

export const QueryEngineIndicator: React.FC<QueryEngineIndicatorProps> = ({
  state,
  estimatedLatencyMs,
  estimatedRows,
  asOfDate,
}) => {
  return (
    <div className={`rounded-xl p-3 border text-xs backdrop-blur-md transition-all ${
      state === 'HYBRID_SEAM'
        ? 'bg-amber-950/20 border-amber-500/40 text-amber-200'
        : state === 'OLAP'
          ? 'bg-cyan-950/20 border-cyan-500/40 text-cyan-200'
          : state === 'OLTP'
            ? 'bg-emerald-950/20 border-emerald-500/40 text-emerald-200'
            : 'bg-purple-950/20 border-purple-500/40 text-purple-200'
    }`}>
      <div className="flex items-center justify-between font-mono font-bold mb-1">
        <div className="flex items-center gap-2">
          {state === 'OLAP' && <span>⚡ StarRocks (OLAP)</span>}
          {state === 'OLTP' && <span>🐘 PostgreSQL (OLTP)</span>}
          {state === 'HYBRID_SEAM' && <span className="animate-pulse">⚡🐘 Hybrid OLTP + Iceberg Seam</span>}
          {state === 'FEDERATED' && <span>🔀 Federated (Trino)</span>}
        </div>
        <span>~{estimatedLatencyMs}ms • {estimatedRows.toLocaleString()} rows</span>
      </div>

      {state === 'HYBRID_SEAM' && (
        <div className="mt-2 text-[11px] text-amber-300/90 space-y-1 font-sans">
          <p className="font-mono">Route: QueryBuilder → CBO → buildUnionSafeQuery / GLBalanceResolver</p>
          <p>
            ⚠️ Query crosses the 90-day hot/cold watermark. Hot side: OLTP • Cold side: Iceberg.
            {asOfDate && <span> Point-in-Time pinned: <strong>{asOfDate}</strong></span>}
          </p>
        </div>
      )}
    </div>
  );
};
