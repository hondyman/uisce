import React, { useState, useEffect, useRef } from 'react';
import {
  Search,
  Bot,
  Sparkles,
  ShieldCheck,
  Code2,
  Database,
  X,
  CornerDownLeft,
  ArrowRight,
} from 'lucide-react';

interface OmniboxResult {
  drivingEntity: string;
  selectedFields: string[];
  hasDatePartition: boolean;
  hasEntityFilter: boolean;
  crossTierEngines: string[];
  rawQuery: string;
}

export const ConversationalOmniboxModal: React.FC<{
  isOpen: boolean;
  tenantId: string;
  onClose: () => void;
  onExecuteQuery?: (ast: OmniboxResult) => void;
}> = ({ isOpen, tenantId, onClose, onExecuteQuery }) => {
  const [prompt, setPrompt] = useState('');
  const [isCompiling, setIsCompiling] = useState(false);
  const [compiledAST, setCompiledAST] = useState<OmniboxResult | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 50);
    } else {
      setPrompt('');
      setCompiledAST(null);
    }
  }, [isOpen]);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!prompt.trim()) return;

    setIsCompiling(true);
    try {
      const res = await fetch('/api/v1/discovery/omnibox', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({ prompt }),
      });

      if (res.ok) {
        const data = await res.json();
        setCompiledAST(data.result);
      }
    } finally {
      setIsCompiling(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-24 bg-slate-950/80 backdrop-blur-sm p-4 font-sans">
      <div className="w-full max-w-2xl bg-[#071526] border border-slate-800 rounded-2xl shadow-2xl overflow-hidden text-slate-100 flex flex-col">
        {/* Input Bar */}
        <form onSubmit={handleSearch} className="flex items-center px-4 py-3.5 border-b border-slate-800 bg-[#030914]">
          <Search className="w-5 h-5 text-cyan-400 mr-3 flex-shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="Ask a question or search catalog (e.g. 'Show Apple price and tech sector allocation')..."
            className="w-full bg-transparent text-sm text-slate-100 placeholder-slate-500 outline-none"
          />
          {prompt && (
            <button
              type="submit"
              disabled={isCompiling}
              className="p-1.5 bg-cyan-500 hover:bg-cyan-400 text-slate-950 rounded-lg text-xs font-bold transition flex items-center gap-1"
            >
              <CornerDownLeft className="w-3.5 h-3.5" />
            </button>
          )}
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 text-slate-400 hover:text-slate-200 ml-2"
          >
            <X className="w-4 h-4" />
          </button>
        </form>

        {/* Dynamic Body */}
        <div className="p-6 space-y-4 max-h-[480px] overflow-y-auto">
          {isCompiling && (
            <div className="py-8 flex flex-col items-center justify-center text-slate-400 text-xs gap-2">
              <Bot className="w-6 h-6 text-cyan-400 animate-spin" />
              <span>Grounding prompt against catalog graph semantic terms...</span>
            </div>
          )}

          {compiledAST && !isCompiling && (
            <div className="space-y-4">
              {/* Intent Header */}
              <div className="p-4 bg-slate-900/80 border border-slate-800 rounded-xl flex items-center justify-between">
                <div>
                  <span className="text-[10px] text-slate-400 uppercase font-semibold block">
                    Resolved Business Object Target
                  </span>
                  <h4 className="text-sm font-bold text-cyan-400 font-mono flex items-center gap-1.5 mt-0.5">
                    <Database className="w-4 h-4" /> {compiledAST.drivingEntity}
                  </h4>
                </div>
                <span className="px-2.5 py-1 bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-[10px] rounded-full font-bold flex items-center gap-1">
                  <ShieldCheck className="w-3 h-3" /> Zero Hallucination Grounded
                </span>
              </div>

              {/* Semantic Terms Resolved */}
              <div>
                <span className="text-xs font-semibold text-slate-300 block mb-2">
                  Mapped Semantic Fields:
                </span>
                <div className="flex flex-wrap gap-2">
                  {compiledAST.selectedFields.map((field) => (
                    <span
                      key={field}
                      className="px-2.5 py-1 bg-slate-950 border border-slate-800 text-slate-200 text-xs font-mono rounded-lg"
                    >
                      {field}
                    </span>
                  ))}
                </div>
              </div>

              {/* Generated Query AST Preview */}
              <div className="p-3 bg-slate-950 rounded-lg border border-slate-800/80 font-mono text-xs text-slate-300 space-y-1">
                <div className="flex items-center justify-between text-slate-500 text-[10px] border-b border-slate-800/60 pb-1">
                  <span className="flex items-center gap-1">
                    <Code2 className="w-3 h-3 text-cyan-400" /> Compiled AST SQL Projection
                  </span>
                  <span>Engines: {compiledAST.crossTierEngines.join(', ')}</span>
                </div>
                <p className="text-cyan-300 pt-1">{compiledAST.rawQuery}</p>
              </div>

              {/* Action Button */}
              <div className="flex justify-end pt-2">
                <button
                  onClick={() => onExecuteQuery && onExecuteQuery(compiledAST)}
                  className="px-4 py-2 bg-gradient-to-r from-cyan-500 to-emerald-400 text-slate-950 text-xs font-bold rounded-lg shadow hover:opacity-95 transition flex items-center gap-1.5"
                >
                  <Sparkles className="w-3.5 h-3.5" /> Execute Grounded Query AST
                </button>
              </div>
            </div>
          )}

          {!compiledAST && !isCompiling && (
            <div className="space-y-3">
              <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider block">
                Suggested Financial Prompts
              </span>
              <div className="grid grid-cols-2 gap-2">
                {[
                  'What is the last close price for Apple?',
                  'Show all cash dividend corporate actions',
                  'Find active account master NAV balances',
                  'Triage open pricing exception breaks',
                ].map((sugg) => (
                  <button
                    key={sugg}
                    onClick={() => {
                      setPrompt(sugg);
                    }}
                    className="p-3 bg-slate-900/40 hover:bg-slate-900 border border-slate-800 rounded-xl text-left text-xs text-slate-300 hover:text-cyan-300 transition flex items-center justify-between group"
                  >
                    <span>{sugg}</span>
                    <ArrowRight className="w-3.5 h-3.5 text-slate-500 group-hover:text-cyan-400" />
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ConversationalOmniboxModal;
