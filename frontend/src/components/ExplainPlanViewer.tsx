import React from 'react';
import { ExplainPlanResult } from '../types/explainPlan';

interface ExplainPlanViewerProps {
  plan: ExplainPlanResult | null;
  loading: boolean;
  onExecute: () => void;
}

export const ExplainPlanViewer: React.FC<ExplainPlanViewerProps> = ({
  plan,
  loading,
  onExecute,
}) => {
  if (loading) {
    return (
      <div className="p-4 rounded-lg bg-gray-900 text-gray-300 animate-pulse">
        Compiling AST Execution Plan...
      </div>
    );
  }

  if (!plan) return null;

  const getScoreBadgeColor = (score: number) => {
    if (score < 50) return 'bg-emerald-500/20 text-emerald-400 border-emerald-500/40';
    if (score < 150) return 'bg-amber-500/20 text-amber-400 border-amber-500/40';
    return 'bg-rose-500/20 text-rose-400 border-rose-500/40';
  };

  return (
    <div className="flex flex-col gap-4 p-5 rounded-xl bg-gray-950 border border-gray-800 text-gray-100 font-sans shadow-2xl">
      {/* Header Metric Cards */}
      <div className="flex items-center justify-between border-b border-gray-800 pb-4">
        <div>
          <h3 className="text-lg font-semibold text-white">Execution Plan & Complexity Analysis</h3>
          <p className="text-xs text-gray-400">BO: <span className="font-mono text-cyan-400">{plan.bo_key}</span> | Engine: <span className="font-mono text-cyan-400">{plan.dialect}</span></p>
        </div>

        <div className="flex items-center gap-3">
          <div className={`px-3 py-1.5 rounded-md text-xs font-mono font-bold border ${getScoreBadgeColor(plan.complexity_score)}`}>
            Cost Score: {plan.complexity_score}
          </div>
          {plan.has_cross_source_join && (
            <span className="px-2.5 py-1.5 rounded-md text-xs font-semibold bg-purple-500/20 text-purple-300 border border-purple-500/40">
              ⚡ Cross-Source Join (Federated)
            </span>
          )}
          <button
            onClick={onExecute}
            className="px-4 py-1.5 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white font-medium text-xs transition-all shadow-lg shadow-cyan-600/20"
          >
            Run Query
          </button>
        </div>
      </div>

      {/* Generated Dialect SQL Preview */}
      <div className="flex flex-col gap-1.5">
        <span className="text-xs font-medium text-gray-400 uppercase tracking-wider">Compiled AST Dialect SQL</span>
        <pre className="p-3.5 rounded-lg bg-gray-900 border border-gray-800 font-mono text-xs text-cyan-300 overflow-x-auto whitespace-pre-wrap">
          {plan.generated_sql}
        </pre>
      </div>

      {/* Execution Steps Breakdown */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-gray-400 uppercase tracking-wider">Execution Pipeline Steps</span>
        <div className="flex flex-col gap-2">
          {plan.execution_steps.map((step) => (
            <div
              key={step.step_id}
              className="flex items-center justify-between p-3 rounded-lg bg-gray-900/70 border border-gray-800/80 hover:border-gray-700 transition-colors"
            >
              <div className="flex items-center gap-3">
                <span className="w-6 h-6 rounded-full bg-gray-800 flex items-center justify-center text-xs font-mono text-gray-300 font-bold">
                  {step.step_id}
                </span>
                <div className="flex flex-col">
                  <span className="text-xs font-bold text-gray-200 font-mono">{step.operation}</span>
                  {step.condition && (
                    <span className="text-[11px] font-mono text-gray-400">ON: {step.condition}</span>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs font-mono text-gray-400">
                  Target: <span className="text-gray-200">{step.target_table}</span> ({step.alias})
                </span>
                <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-gray-800 text-gray-300">
                  Weight: {step.cost_weight}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
