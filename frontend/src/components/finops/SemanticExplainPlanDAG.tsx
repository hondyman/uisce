import React, { useMemo } from 'react';
import ReactFlow, {
  Background,
  Controls,
  Handle,
  Position,
  NodeProps,
  Edge,
  Node,
} from 'reactflow';
import 'reactflow/dist/style.css';
import {
  ShieldAlert,
  Zap,
  HardDrive,
  DollarSign,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  TrendingDown,
} from 'lucide-react';

export interface ExplainPlanData {
  planId: string;
  complexityScore: number;
  costBand: 'LOW' | 'MODERATE' | 'EXPENSIVE' | 'FORBIDDEN';
  canExecute: boolean;
  estimatedBytes: number;
  attributedCostUSD: number;
  nodes: Array<{
    id: string;
    type: string;
    label: string;
    engine: string;
    scannedRows: number;
    durationMs: number;
    costUSD: number;
    status: 'COMPLETED' | 'SKIPPED' | 'BLOCKED';
    details: string;
  }>;
  edges: Array<{ id: string; source: string; target: string }>;
  recommendations: string[];
}

const CustomPlanNode: React.FC<NodeProps> = ({ data }) => {
  const isBlocked = data.status === 'BLOCKED';
  const isSkipped = data.status === 'SKIPPED';

  return (
    <div
      className={`p-4 rounded-xl border shadow-xl w-64 bg-[#071526] transition-all ${
        isBlocked
          ? 'border-red-500/80 shadow-red-950/40'
          : isSkipped
          ? 'border-slate-800 opacity-50'
          : 'border-cyan-500/50 shadow-cyan-950/20'
      }`}
    >
      <Handle type="target" position={Position.Top} className="!bg-cyan-400 !w-2 !h-2" />
      
      <div className="flex items-center justify-between pb-2 border-b border-slate-800">
        <span className="text-[10px] font-mono uppercase tracking-wider text-slate-400">
          {data.engine}
        </span>
        <span
          className={`flex items-center gap-1 text-[10px] font-bold px-1.5 py-0.5 rounded ${
            data.status === 'COMPLETED'
              ? 'bg-emerald-500/20 text-emerald-400'
              : data.status === 'BLOCKED'
              ? 'bg-red-500/20 text-red-400'
              : 'bg-slate-800 text-slate-400'
          }`}
        >
          {data.status === 'COMPLETED' ? (
            <CheckCircle2 className="w-3 h-3" />
          ) : (
            <XCircle className="w-3 h-3" />
          )}
          {data.status}
        </span>
      </div>

      <div className="my-2">
        <h4 className="text-xs font-bold text-slate-100">{data.label}</h4>
        <p className="text-[10px] text-slate-400 mt-0.5 line-clamp-2">{data.details}</p>
      </div>

      <div className="grid grid-cols-2 gap-2 pt-2 border-t border-slate-800/80 text-[10px] font-mono">
        <div>
          <span className="text-slate-500 block">Latency</span>
          <span className="text-slate-200 font-semibold">{data.durationMs}ms</span>
        </div>
        <div className="text-right">
          <span className="text-slate-500 block">Scanned</span>
          <span className="text-cyan-400 font-semibold">
            {data.scannedRows.toLocaleString()} rows
          </span>
        </div>
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-cyan-400 !w-2 !h-2" />
    </div>
  );
};

export const SemanticExplainPlanDAG: React.FC<{
  plan: ExplainPlanData;
  onClose?: () => void;
}> = ({ plan, onClose }) => {
  const nodeTypes = useMemo(() => ({ customPlan: CustomPlanNode }), []);

  const flowNodes: Node[] = useMemo(() => {
    const layoutCoords: Record<string, { x: number; y: number }> = {
      'node-1': { x: 250, y: 20 },
      'node-2': { x: 80, y: 180 },
      'node-3': { x: 420, y: 180 },
      'node-4': { x: 250, y: 360 },
    };

    return plan.nodes.map((n) => ({
      id: n.id,
      type: 'customPlan',
      position: layoutCoords[n.id] || { x: 200, y: 100 },
      data: n,
    }));
  }, [plan.nodes]);

  const flowEdges: Edge[] = useMemo(() => {
    return plan.edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      animated: plan.canExecute,
      style: { stroke: plan.canExecute ? '#06b6d4' : '#ef4444', strokeWidth: 2 },
    }));
  }, [plan.edges, plan.canExecute]);

  return (
    <div className="flex flex-col h-full bg-[#030914] text-slate-100 border border-slate-800 rounded-xl overflow-hidden font-sans">
      {/* Top Banner: Complexity Scoring & FinOps Metering */}
      <div className="p-6 bg-[#071526] border-b border-slate-800 flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-base font-bold text-slate-100 flex items-center gap-2">
              <Zap className="w-5 h-5 text-cyan-400" />
              Visual Explain-Plan DAG & Cost Governor
            </h2>
            <span
              className={`px-3 py-1 rounded-full text-xs font-bold font-mono uppercase tracking-wider ${
                plan.costBand === 'LOW'
                  ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                  : plan.costBand === 'MODERATE'
                  ? 'bg-cyan-500/20 text-cyan-400 border border-cyan-500/30'
                  : plan.costBand === 'EXPENSIVE'
                  ? 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
                  : 'bg-red-500/20 text-red-400 border border-red-500/30 animate-pulse'
              }`}
            >
              Score: {plan.complexityScore} / 100 ({plan.costBand})
            </span>
          </div>
          <p className="text-xs text-slate-400 mt-1">
            Plan ID: <span className="font-mono text-slate-300">{plan.planId}</span>
          </p>
        </div>

        {/* Financial Chargeback Attribution Chips */}
        <div className="flex items-center gap-4">
          <div className="p-2.5 bg-slate-900 border border-slate-800 rounded-lg text-right">
            <span className="text-[10px] text-slate-400 uppercase font-semibold flex items-center gap-1 justify-end">
              <HardDrive className="w-3 h-3 text-cyan-400" /> Est. Scanned Volume
            </span>
            <span className="text-xs font-bold font-mono text-slate-100">
              {(plan.estimatedBytes / (1024 * 1024)).toFixed(1)} MB
            </span>
          </div>

          <div className="p-2.5 bg-slate-900 border border-slate-800 rounded-lg text-right">
            <span className="text-[10px] text-slate-400 uppercase font-semibold flex items-center gap-1 justify-end">
              <DollarSign className="w-3 h-3 text-emerald-400" /> Attributed FinOps Cost
            </span>
            <span className="text-xs font-bold font-mono text-emerald-400">
              ${plan.attributedCostUSD.toFixed(6)}
            </span>
          </div>
        </div>
      </div>

      {/* Circuit Breaker Alert Banner */}
      {!plan.canExecute && (
        <div className="p-4 bg-red-950/40 border-b border-red-500/40 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <ShieldAlert className="w-5 h-5 text-red-400 flex-shrink-0 animate-bounce" />
            <div>
              <h4 className="text-xs font-bold text-red-300 uppercase tracking-wider">
                Cardinal Rule 8 Cost Circuit Breaker Tripped
              </h4>
              <p className="text-xs text-red-200 mt-0.5">
                Query AST complexity score ({plan.complexityScore}) exceeds tenant threshold (80). Execution blocked to prevent runaway cloud compute costs.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Main Execution Canvas */}
      <div className="flex-1 relative min-h-[420px] bg-[#02060d]">
        <ReactFlow
          nodes={flowNodes}
          edges={flowEdges}
          nodeTypes={nodeTypes}
          fitView
          attributionPosition="bottom-left"
        >
          <Background color="#1e293b" gap={16} />
          <Controls className="!bg-slate-900 !border-slate-800 !fill-slate-300" />
        </ReactFlow>
      </div>

      {/* Optimization Recommendations Drawer */}
      {plan.recommendations.length > 0 && (
        <div className="p-4 bg-[#071526] border-t border-slate-800 space-y-2">
          <h4 className="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center gap-1.5">
            <TrendingDown className="w-4 h-4 text-amber-400" />
            Query Optimization & Remediation Proposals
          </h4>
          <div className="grid grid-cols-2 gap-2">
            {plan.recommendations.map((rec, i) => (
              <div
                key={i}
                className="p-2.5 bg-slate-900/80 border border-slate-800 rounded-lg text-xs text-slate-300 flex items-start gap-2"
              >
                <AlertTriangle className="w-3.5 h-3.5 text-amber-400 flex-shrink-0 mt-0.5" />
                <span>{rec}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default SemanticExplainPlanDAG;
