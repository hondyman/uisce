import React, { useState, useEffect } from 'react';
import { Calculator, AlertTriangle, Sparkles, X, Bot } from 'lucide-react';

interface Props {
  tenantId: string;
}

export const SemanticDiscoveryOmnibox: React.FC<Props> = ({ tenantId }) => {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [query, setQuery] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [result, setResult] = useState<any | null>(null);

  // Keyboard shortcut listener (Cmd + K or Ctrl + K)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setIsOpen((prev) => !prev);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;

    setLoading(true);
    try {
      const res = await fetch('/api/v1/discovery/search', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({ query }),
      });
      const data = await res.json();
      setResult(data);
    } catch (err) {
      console.error('Discovery search failed:', err);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-md flex items-start justify-center pt-20 p-4 font-sans">
      <div className="bg-slate-900 border border-slate-700/80 rounded-2xl shadow-2xl w-full max-w-3xl overflow-hidden flex flex-col max-h-[80vh]">
        {/* Search Input Bar */}
        <form onSubmit={handleSearch} className="flex items-center px-4 py-3 border-b border-slate-800 bg-slate-950/50">
          <Sparkles className="text-sky-400 mr-3 shrink-0 animate-pulse" size={20} />
          <input
            type="text"
            autoFocus
            placeholder="Ask Uisce AI... (e.g., 'Show open pricing breaks for Fund Alpha and run Monte Carlo VaR')"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="bg-transparent text-white placeholder-slate-500 text-sm focus:outline-none w-full font-medium"
          />
          {loading && <div className="text-xs text-sky-400 font-mono animate-pulse mr-2">Evaluating...</div>}
          <button type="button" onClick={() => setIsOpen(false)} className="text-slate-500 hover:text-slate-300">
            <X size={18} />
          </button>
        </form>

        {/* Results Body */}
        <div className="p-6 overflow-y-auto space-y-6 flex-1 text-slate-200">
          {!result && !loading && (
            <div className="text-center py-12 text-slate-500 text-xs font-mono">
              Press <kbd className="bg-slate-800 px-1.5 py-0.5 rounded text-slate-300">Enter</kbd> to execute conversational discovery.
            </div>
          )}

          {result && (
            <>
              {/* WASM Analytics Cards */}
              {result.wasm_metrics && result.wasm_metrics.length > 0 && (
                <div>
                  <h4 className="text-xs font-bold uppercase tracking-wider text-sky-400 flex items-center gap-1.5 mb-3">
                    <Calculator size={14} /> WASM Calculated Metrics
                  </h4>
                  <div className="grid grid-cols-2 gap-4">
                    {result.wasm_metrics.map((m: any) => (
                      <div key={m.metric_key} className="bg-slate-800/90 border border-sky-500/30 p-4 rounded-xl">
                        <span className="text-[11px] text-slate-400 font-medium">{m.metric_label}</span>
                        <div className="text-2xl font-bold font-mono text-emerald-400 mt-1">{m.formatted}</div>
                        <span className="text-[10px] text-slate-500 font-mono mt-2 block">Execution Latency: {m.duration_us}µs</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* MDM Exception Breaks */}
              {result.mdm_breaks && result.mdm_breaks.length > 0 && (
                <div>
                  <h4 className="text-xs font-bold uppercase tracking-wider text-amber-400 flex items-center gap-1.5 mb-3">
                    <AlertTriangle size={14} /> Open Master Data Breaks
                  </h4>
                  <div className="space-y-3">
                    {result.mdm_breaks.map((b: any) => (
                      <div key={b.exception_id} className="bg-slate-800/60 border border-amber-500/30 p-4 rounded-xl">
                        <div className="flex justify-between items-center mb-1">
                          <span className="text-xs font-bold text-white font-mono">{b.entity_key}</span>
                          <span className="text-[10px] bg-amber-500/20 text-amber-300 px-2 py-0.5 rounded uppercase font-semibold">
                            {b.field_name}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 text-xs text-emerald-300 bg-emerald-950/40 p-2 rounded-lg border border-emerald-500/20 mt-2">
                          <Bot size={14} className="shrink-0" />
                          <span>{b.ai_recommendation}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};
