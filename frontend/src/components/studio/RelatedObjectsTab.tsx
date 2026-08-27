import React from 'react';
import { Layers, GitCommit, ArrowUpRight, AlertCircle } from 'lucide-react';

export interface RelatedObjectAssociation {
  relId: string;
  relatedBoKey: string;
  relatedBoName: string;
  cardinality: '1:1' | '1:N' | 'M:1' | 'M:N';
  isIncluded: boolean;
  fieldsCount: number;
}

export const RelatedObjectsTab: React.FC<{
  relationships: RelatedObjectAssociation[];
  onToggleInclude: (relId: string) => void;
  onConfigureJoin: (relId: string) => void;
}> = ({ relationships, onToggleInclude, onConfigureJoin }) => {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-bold text-white uppercase tracking-wider">Related Business Objects</h3>
          <p className="text-xs text-slate-400">Select related entities to expand available fields while controlling join cardinality.</p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-3">
        {relationships.map((rel) => (
          <div
            key={rel.relId}
            className={`p-4 rounded-xl border transition-all flex items-center justify-between ${
              rel.isIncluded
                ? 'bg-teal-950/20 border-teal-500/50 shadow-[0_0_12px_rgba(20,184,166,0.1)]'
                : 'bg-[#070E1B] border-slate-800 hover:border-slate-700'
            }`}
          >
            <div className="flex items-center gap-3">
              <input
                type="checkbox"
                checked={rel.isIncluded}
                onChange={() => onToggleInclude(rel.relId)}
                className="rounded border-slate-700 text-teal-500 focus:ring-0 w-4 h-4"
              />
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-bold text-sm text-white">{rel.relatedBoName}</span>
                  <span className={`text-[10px] px-2 py-0.5 rounded font-mono font-bold ${
                    rel.cardinality === '1:N' || rel.cardinality === 'M:N'
                      ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30'
                      : 'bg-teal-500/20 text-teal-300 border border-teal-500/30'
                  }`}>
                    {rel.cardinality} {rel.cardinality === '1:N' && '⚠️ Fan-Out'}
                  </span>
                </div>
                <span className="text-xs text-slate-400 font-mono">Key: {rel.relatedBoKey} • {rel.fieldsCount} fields available</span>
              </div>
            </div>

            <button
              onClick={() => onConfigureJoin(rel.relId)}
              className="text-xs text-teal-400 hover:text-teal-300 font-semibold flex items-center gap-1"
            >
              Configure Join <ArrowUpRight className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};

export default RelatedObjectsTab;
