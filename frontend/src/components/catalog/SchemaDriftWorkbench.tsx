import React, { useState, useEffect } from 'react';
import {
  Wrench,
  AlertOctagon,
  CheckCircle2,
  ArrowRight,
  Sparkles,
  RefreshCw,
  FileSpreadsheet,
} from 'lucide-react';

export interface DriftProposal {
  proposalId: string;
  boId: string;
  boName: string;
  fieldName: string;
  currentColumn: string;
  proposedColumn: string;
  confidenceScore: number;
  matchingStrategy: string;
  affectedReportsCount: number;
  remediationRationale: string;
  status: 'PENDING' | 'APPLIED' | 'REJECTED';
}

export const SchemaDriftWorkbench: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const [proposals, setProposals] = useState<DriftProposal[]>([]);
  const [isPatching, setIsPatching] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  const fetchProposals = async () => {
    try {
      const res = await fetch('/api/v1/catalog/drift/proposals', {
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (res.ok) {
        const data = await res.json();
        setProposals(data);
      }
    } catch (e) {
      console.error('Failed fetching drift proposals:', e);
    }
  };

  useEffect(() => {
    fetchProposals();
  }, [tenantId]);

  const handleApplyPatch = async (proposalId: string) => {
    setIsPatching(proposalId);
    try {
      const res = await fetch(`/api/v1/catalog/drift/proposals/${proposalId}/apply`, {
        method: 'POST',
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (res.ok) {
        setStatusMessage('Non-breaking binding hot-swap applied successfully.');
        setTimeout(() => setStatusMessage(null), 4000);
        await fetchProposals();
      }
    } finally {
      setIsPatching(null);
    }
  };

  return (
    <div className="flex flex-col h-full bg-[#030914] text-slate-100 border border-slate-800 rounded-xl overflow-hidden font-sans">
      <div className="p-6 bg-[#071526] border-b border-slate-800 flex items-center justify-between">
        <div>
          <h2 className="text-base font-bold text-slate-100 flex items-center gap-2">
            <Wrench className="w-5 h-5 text-cyan-400" />
            Self-Healing Semantic Mesh & Schema Drift Sentinel
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Detect upstream physical column renames, review confidence proposals, and execute 1-click binding hot-swaps.
          </p>
        </div>

        <button
          onClick={fetchProposals}
          className="p-2.5 bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded-lg text-slate-300 transition flex items-center gap-2 text-xs font-semibold"
        >
          <RefreshCw className="w-3.5 h-3.5" /> Scan Catalog
        </button>
      </div>

      {statusMessage && (
        <div className="bg-emerald-500/20 border-b border-emerald-500/40 px-6 py-2.5 text-xs text-emerald-300 flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 text-emerald-400" />
          {statusMessage}
        </div>
      )}

      <div className="p-6 space-y-4 flex-1 overflow-y-auto">
        {proposals.map((item) => (
          <div
            key={item.proposalId}
            className="p-5 bg-slate-900/60 border border-slate-800 rounded-xl space-y-4 hover:border-cyan-500/40 transition"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="p-2 bg-amber-500/10 border border-amber-500/30 text-amber-400 rounded-lg">
                  <AlertOctagon className="w-4 h-4" />
                </span>
                <div>
                  <h4 className="text-xs font-bold text-slate-100">
                    Business Object: <span className="text-cyan-400 font-mono">{item.boName}</span>
                  </h4>
                  <p className="text-[11px] text-slate-400">
                    Orphaned Field: <strong className="text-slate-200">{item.fieldName}</strong>
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-4">
                <div className="text-right">
                  <span className="text-[10px] text-slate-400 block uppercase">Match Confidence</span>
                  <span className="text-xs font-bold font-mono text-emerald-400">
                    {(item.confidenceScore * 100).toFixed(0)}% ({item.matchingStrategy})
                  </span>
                </div>
                <div className="text-right">
                  <span className="text-[10px] text-slate-400 block uppercase">Blast Radius</span>
                  <span className="text-xs font-bold font-mono text-purple-400 flex items-center gap-1">
                    <FileSpreadsheet className="w-3 h-3" /> {item.affectedReportsCount} Reports
                  </span>
                </div>
              </div>
            </div>

            <div className="p-3 bg-slate-950 rounded-lg border border-slate-800/80 flex items-center justify-between text-xs font-mono">
              <div className="flex items-center gap-2">
                <span className="text-red-400 line-through">{item.currentColumn}</span>
                <ArrowRight className="w-3.5 h-3.5 text-slate-500" />
                <span className="text-emerald-400 font-bold">{item.proposedColumn}</span>
              </div>
              <span className="text-[10px] text-slate-400 font-sans">{item.remediationRationale}</span>
            </div>

            <div className="flex items-center justify-end gap-3 pt-2 border-t border-slate-800/60">
              <button
                onClick={() => handleApplyPatch(item.proposalId)}
                disabled={isPatching === item.proposalId}
                className="px-4 py-2 bg-gradient-to-r from-cyan-500 to-emerald-400 text-slate-950 text-xs font-bold rounded-lg shadow hover:opacity-95 transition flex items-center gap-1.5 disabled:opacity-50"
              >
                <Sparkles className="w-3.5 h-3.5" />
                {isPatching === item.proposalId ? 'Applying Patch...' : '1-Click Hot-Swap Binding Patch'}
              </button>
            </div>
          </div>
        ))}

        {proposals.length === 0 && (
          <div className="p-12 text-center text-slate-400 text-xs bg-slate-900/20 rounded-xl border border-slate-800/40">
            All physical catalog columns are synchronized with Business Object bindings. No drift detected.
          </div>
        )}
      </div>
    </div>
  );
};

export default SchemaDriftWorkbench;
