import React, { useState, useEffect } from 'react';
import { AlertTriangle, CheckCircle, Bot, RefreshCw, ArrowRight } from 'lucide-react';

interface VendorFeedPayload {
  vendor_name: string;
  value: number;
  trust_score: number;
}

interface MDMExceptionItem {
  exception_id: string;
  workflow_id: string;
  entity_key: string;
  bo_type: string;
  field_name: string;
  competing_values: VendorFeedPayload[];
  ai_recommendation: string;
  created_at: string;
}

export const MDMBreakManagementStudio: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const [exceptions, setExceptions] = useState<MDMExceptionItem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [overrideValue, setOverrideValue] = useState<{ [key: string]: number }>({});

  useEffect(() => {
    fetchOpenExceptions();
  }, [tenantId]);

  const fetchOpenExceptions = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/mdm/exceptions?tenantId=${tenantId}`);
      const data = await res.json();
      setExceptions(data.exceptions || []);
    } catch (err) {
      console.error('Failed fetching MDM breaks:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleResolveBreak = async (workflowId: string, value: number) => {
    try {
      // Dispatches Temporal Signal channel: MDM_STEWARD_OVERRIDE_SIGNAL
      await fetch('/api/v1/mdm/exceptions/signal', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
          workflow_id: workflowId,
          signal_name: 'MDM_STEWARD_OVERRIDE_SIGNAL',
          override_value: value,
        }),
      });

      // Refresh break queue
      fetchOpenExceptions();
    } catch (err) {
      alert(`Failed to resolve break: ${err}`);
    }
  };

  if (loading) return <div className="p-6 text-slate-400">Loading MDM Exception Queue...</div>;

  return (
    <div className="p-8 bg-slate-900 min-h-screen text-slate-100 font-sans">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-wide">Uisce MDM Exception Management Studio</h1>
          <p className="text-sm text-slate-400">Human-in-the-Loop Temporal Workflow Orchestration</p>
        </div>
        <button 
          onClick={fetchOpenExceptions}
          className="flex items-center gap-2 bg-slate-800 hover:bg-slate-700 px-4 py-2 rounded-lg text-xs font-semibold text-slate-300 transition"
        >
          <RefreshCw size={14} /> Refresh Queue
        </button>
      </div>

      {exceptions.length === 0 ? (
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-xl p-12 text-center">
          <CheckCircle className="mx-auto text-emerald-400 mb-3" size={48} />
          <h3 className="text-lg font-semibold text-white">All Clear! No Open Data Breaks</h3>
          <p className="text-slate-400 text-sm mt-1">Vendor feeds are operating within line-rate tolerance thresholds.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6">
          {exceptions.map((ex) => (
            <div key={ex.exception_id} className="bg-slate-800/80 border border-amber-500/30 rounded-xl p-6 shadow-xl">
              <div className="flex justify-between items-start mb-4">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-amber-500/20 text-amber-400 rounded-lg">
                    <AlertTriangle size={20} />
                  </div>
                  <div>
                    <span className="text-xs font-mono bg-slate-700 px-2 py-0.5 rounded text-slate-300">{ex.bo_type}</span>
                    <h3 className="text-lg font-bold text-white mt-1">
                      Entity: <span className="text-sky-400 font-mono">{ex.entity_key}</span> — Field: <span className="text-amber-300 font-mono">{ex.field_name}</span>
                    </h3>
                  </div>
                </div>
                <span className="text-xs text-slate-400 font-mono">{new Date(ex.created_at).toLocaleTimeString()}</span>
              </div>

              {/* Competing Vendor Feeds */}
              <div className="grid grid-cols-3 gap-4 mb-6">
                {ex.competing_values.map((v) => (
                  <div key={v.vendor_name} className="bg-slate-900/90 p-4 rounded-lg border border-slate-700/60">
                    <div className="flex justify-between items-center mb-2">
                      <span className="text-xs font-bold text-slate-300 tracking-wider">{v.vendor_name}</span>
                      <span className="text-[10px] bg-slate-800 text-slate-400 px-1.5 py-0.5 rounded font-mono">Trust: {v.trust_score}</span>
                    </div>
                    <div className="text-xl font-bold font-mono text-white">${v.value.toFixed(2)}</div>
                    <button
                      onClick={() => handleResolveBreak(ex.workflow_id, v.value)}
                      className="w-full mt-3 bg-sky-600/20 hover:bg-sky-600/40 border border-sky-500/40 text-sky-300 py-1.5 rounded text-xs font-semibold flex items-center justify-center gap-1 transition"
                    >
                      Accept Value <ArrowRight size={12} />
                    </button>
                  </div>
                ))}
              </div>

              {/* AI Steward Copilot Recommendation */}
              <div className="bg-emerald-950/30 border border-emerald-500/30 rounded-lg p-4 mb-4 flex items-start gap-3">
                <Bot className="text-emerald-400 mt-0.5 shrink-0" size={18} />
                <div>
                  <h4 className="text-xs font-bold text-emerald-400 uppercase tracking-wider mb-0.5">AI Data Steward Copilot (MCP) Recommendation</h4>
                  <p className="text-xs text-emerald-200/90 leading-relaxed">{ex.ai_recommendation}</p>
                </div>
              </div>

              {/* Manual Override Input */}
              <div className="flex gap-3 pt-2 border-t border-slate-700/50">
                <input
                  type="number"
                  step="0.01"
                  placeholder="Enter manual override value..."
                  value={overrideValue[ex.exception_id] || ''}
                  onChange={(e) => setOverrideValue({ ...overrideValue, [ex.exception_id]: parseFloat(e.target.value) })}
                  className="bg-slate-900 border border-slate-700 rounded-lg px-3 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-sky-500 flex-1"
                />
                <button
                  onClick={() => handleResolveBreak(ex.workflow_id, overrideValue[ex.exception_id])}
                  disabled={!overrideValue[ex.exception_id]}
                  className="bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white px-4 py-1.5 rounded-lg text-xs font-semibold transition"
                >
                  Apply Steward Override Signal
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
