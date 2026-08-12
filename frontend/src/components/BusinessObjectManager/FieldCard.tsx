import React from 'react';

export type StorageType = 'GRAPH_TOPOLOGY' | 'OLTP_BRIDGE' | 'HYBRID' | 'CALCULATED';
export type RequirementLevel = 'REQUIRED' | 'OPTIONAL' | 'BACKEND_SPECIFIC' | 'CALCULATED' | 'INTERNAL';
export type BindingStatus = 'RESOLVED' | 'PARTIAL' | 'UNRESOLVED' | 'NOT_APPLICABLE';

export interface BackendMapping {
  backendId: string;
  backendName: string;
  sourceType: 'COLUMN' | 'EXPRESSION' | 'JSON_PATH' | 'UNBOUND';
  sourceExpression: string;
  dataType: string;
  status: BindingStatus;
}

export interface FieldCardProps {
  fieldId: string;
  semanticTermKey: string;
  displayName: string;
  fieldRole: 'DIMENSION' | 'MEASURE' | 'ATTRIBUTE' | 'KEY';
  storageType: StorageType;
  bridgeKey?: string; // e.g. "catalog_node.id -> orm.order.node_id"
  requirementLevel: RequirementLevel;
  mappings: BackendMapping[];
  starRocksExpression?: string;
  oltpExpression?: string;
  usageCount: { reports: number; apis: number; calcs: number };
  isCore: boolean;
  hasOverrides?: boolean;
  onMapColumn?: (fieldId: string) => void;
  onOverride?: (fieldId: string) => void;
  onViewLineage?: (fieldId: string) => void;
}

export const FieldCard: React.FC<FieldCardProps> = ({
  fieldId,
  displayName,
  semanticTermKey,
  fieldRole,
  storageType,
  bridgeKey,
  requirementLevel,
  mappings = [],
  usageCount = { reports: 0, apis: 0, calcs: 0 },
  isCore,
  onMapColumn,
  onOverride,
  onViewLineage,
}) => {
  const isUnresolved = mappings.some(m => m.status === 'UNRESOLVED' && requirementLevel === 'REQUIRED');

  return (
    <div className={`relative rounded-xl p-4 border transition-all duration-200 backdrop-blur-md ${
      isUnresolved 
        ? 'border-amber-500/50 bg-amber-950/10 shadow-[0_0_15px_rgba(245,158,11,0.15)]' 
        : isCore 
          ? 'border-amber-500/20 bg-slate-900/80 shadow-[0_0_10px_rgba(245,166,35,0.05)]' 
          : 'border-cyan-500/30 bg-slate-900/80'
    }`}>
      {/* Top Badge Rail */}
      <div className="flex items-center justify-between gap-2 mb-3">
        <div className="flex items-center gap-1.5">
          {isCore ? (
            <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-gradient-to-r from-amber-500 to-amber-600 text-slate-950 tracking-wider">
              ⬡ CORE
            </span>
          ) : (
            <span className="px-2 py-0.5 rounded text-[10px] font-bold border border-cyan-400/50 bg-cyan-950/30 text-cyan-400 tracking-wider">
              ✦ CUSTOM
            </span>
          )}

          {/* Requirement Badge */}
          <span className={`px-2 py-0.5 rounded text-[10px] font-semibold tracking-wide ${
            requirementLevel === 'REQUIRED' ? 'bg-red-500/20 text-red-400 border border-red-500/30' :
            requirementLevel === 'OPTIONAL' ? 'bg-slate-700/50 text-slate-400' :
            requirementLevel === 'CALCULATED' ? 'bg-purple-500/20 text-purple-300 border border-purple-500/30' :
            'bg-amber-500/20 text-amber-300'
          }`}>
            {requirementLevel}
          </span>
        </div>

        <span className="text-[11px] font-mono font-medium text-slate-400 tracking-wider">
          {fieldRole}
        </span>
      </div>

      {/* Field Identity */}
      <h4 className="text-base font-semibold text-slate-100 mb-0.5">{displayName}</h4>
      <p className="text-xs font-mono text-slate-400 mb-3">{semanticTermKey}</p>

      {/* Storage Type Layer (Rule 6 Enforcement) */}
      <div className={`rounded-lg p-2.5 mb-3 border text-xs ${
        storageType === 'OLTP_BRIDGE' 
          ? 'bg-amber-950/20 border-amber-500/30 text-amber-200' 
          : storageType === 'GRAPH_TOPOLOGY'
            ? 'bg-violet-950/20 border-violet-500/30 text-violet-200'
            : 'bg-purple-950/20 border-purple-500/30 text-purple-200'
      }`}>
        <div className="flex items-center gap-2 font-mono font-semibold">
          {storageType === 'OLTP_BRIDGE' && <span className="animate-pulse text-amber-400">⚡ OLTP FINANCIAL STATE</span>}
          {storageType === 'GRAPH_TOPOLOGY' && <span className="text-violet-400">◈ GRAPH TOPOLOGY</span>}
          {storageType === 'CALCULATED' && <span className="text-purple-400">◇ COMPUTED METRIC</span>}
          {storageType === 'HYBRID' && <span className="text-cyan-400">⚡◈ HYBRID STORAGE</span>}
        </div>
        {bridgeKey && (
          <p className="text-[11px] font-mono text-slate-400 mt-1">
            Bridge: <code className="text-slate-300">{bridgeKey}</code>
          </p>
        )}
      </div>

      {/* Backend Mappings Table */}
      <div className="space-y-1.5 mb-3">
        {mappings.map((m) => (
          <div key={m.backendId} className="flex items-center justify-between text-xs font-mono bg-slate-950/50 rounded px-2.5 py-1.5 border border-slate-800">
            <span className="text-slate-400">{m.backendName}</span>
            <span className="text-slate-200 truncate max-w-[180px]">{m.sourceExpression || 'Unbound'}</span>
            <span className={m.status === 'RESOLVED' ? 'text-emerald-400' : 'text-amber-400 font-bold'}>
              {m.status === 'RESOLVED' ? '✓' : '⚠'}
            </span>
          </div>
        ))}
      </div>

      {/* Usage Footnote & Actions */}
      <div className="pt-2 border-t border-slate-800 flex items-center justify-between text-xs text-slate-400">
        <span>Used in: {usageCount.reports} Rps • {usageCount.apis} APIs</span>
        <div className="flex items-center gap-2">
          {onMapColumn && <button onClick={() => onMapColumn(fieldId)} className="hover:text-cyan-400 transition-colors">Map</button>}
          {onOverride && <button onClick={() => onOverride(fieldId)} className="hover:text-amber-400 transition-colors">Override</button>}
          {onViewLineage && <button onClick={() => onViewLineage(fieldId)} className="hover:text-purple-400 transition-colors">Lineage</button>}
        </div>
      </div>
    </div>
  );
};
