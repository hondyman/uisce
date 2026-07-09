import React, { useMemo } from 'react';
import type {
  ResolvedCapabilities,
  ResolvedField,
} from '@/types/mutability';
import { getFieldTierTag } from '@/types/mutability';

export interface AdaptiveSemanticCanvasProps {
  fields: ResolvedField[];
  dataset: Record<string, unknown>[];
  capabilities: ResolvedCapabilities;
  className?: string;
  emptyMessage?: string;
}

/**
 * AdaptiveSemanticCanvas renders a read-only table that respects the
 * per-field hydration state. Fields whose hydrationState is
 * UNBOUND_FALLBACK_NULL are rendered as `-- field unavailable --` (italic,
 * muted) so the user sees the gap without losing context. Fields sourced
 * from the hot tier get a small "Hot Tier Only" badge.
 *
 * Cardinal Rule 1: every rendering decision reads from the resolved
 * capability payload — never from engine_type by name in this component.
 */
export const AdaptiveSemanticCanvas: React.FC<AdaptiveSemanticCanvasProps> = ({
  fields,
  dataset,
  capabilities,
  className,
  emptyMessage = '-- field unavailable --',
}) => {
  const isCqrsRoute =
    capabilities.mutabilityMode === 'ASYNCHRONOUS_CQRS_QUEUE';

  const tierTags = useMemo(() => {
    const out: Record<string, ReturnType<typeof getFieldTierTag>> = {};
    for (const f of fields) {
      out[f.semanticTermKey] = getFieldTierTag(f, capabilities);
    }
    return out;
  }, [fields, capabilities]);

  return (
    <div
      className={
        className ||
        'p-4 rounded-xl border border-slate-800 bg-slate-950 font-mono text-xs text-slate-300'
      }
      data-testid="adaptive-canvas"
    >
      <div className="flex justify-between items-center mb-4 pb-2 border-b border-slate-900">
        <span className="text-slate-400 font-bold">Semantic Engine Viewport</span>
        <span
          className={`text-[10px] px-2 py-0.5 rounded border ${
            isCqrsRoute
              ? 'bg-blue-500/10 text-blue-400 border-blue-500/20'
              : 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
          }`}
        >
          {isCqrsRoute ? 'CQRS Event Stream Node' : 'Direct Storage Mode'}
        </span>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-slate-900 text-slate-400 uppercase tracking-tight border-b border-slate-800">
              {fields.map((f) => {
                const tag = tierTags[f.semanticTermKey];
                return (
                  <th key={f.semanticTermKey} className="p-3">
                    <div className="flex items-center gap-1.5">
                      <span>{f.displayLabel}</span>
                      {tag === 'hot' && (
                        <span
                          className="text-[8px] text-amber-400 bg-amber-500/10 px-1 rounded animate-pulse"
                          title="Source field lives in the hot compute tier"
                        >
                          Hot Tier Only
                        </span>
                      )}
                      {tag === 'cold' && (
                        <span
                          className="text-[8px] text-indigo-400 bg-indigo-500/10 px-1 rounded"
                          title="Source field lives in the cold store"
                        >
                          Cold Tier
                        </span>
                      )}
                      {f.hydrationState === 'UNBOUND_FALLBACK_NULL' && tag !== 'hot' && (
                        <span
                          className="text-[8px] text-amber-400 bg-amber-500/10 px-1 rounded animate-pulse"
                          title="Field missing from source backend"
                        >
                          Unbound
                        </span>
                      )}
                    </div>
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {dataset.length === 0 ? (
              <tr>
                <td
                  className="p-6 text-center text-slate-600 italic"
                  colSpan={Math.max(1, fields.length)}
                >
                  No rows to display
                </td>
              </tr>
            ) : (
              dataset.map((row, idx) => (
                <tr
                  key={idx}
                  className="hover:bg-slate-900/50 border-b border-slate-900"
                >
                  {fields.map((f) => {
                    if (f.hydrationState === 'UNBOUND_FALLBACK_NULL') {
                      return (
                        <td
                          key={f.semanticTermKey}
                          className="p-3 text-slate-600 bg-slate-900/10 italic selection:bg-transparent"
                        >
                          {emptyMessage}
                        </td>
                      );
                    }
                    const v = row[f.semanticTermKey];
                    return (
                      <td key={f.semanticTermKey} className="p-3 text-slate-100">
                        {v === undefined || v === null || v === ''
                          ? '--'
                          : String(v)}
                      </td>
                    );
                  })}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default AdaptiveSemanticCanvas;