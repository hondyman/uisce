import React, { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';

const NodeWrapper = ({ children, selected, className = '' }: { children: React.ReactNode; selected?: boolean; className?: string }) => (
  <div className={`relative flex flex-col items-center justify-center rounded-xl border bg-white dark:bg-[#18232f] p-4 shadow-lg min-w-[180px] transition-all duration-200 ${selected ? 'border-primary ring-2 ring-primary/20' : 'border-gray-200 dark:border-gray-700'} ${className}`}>
    {children}
  </div>
);

export const StartNode = memo(({ selected }: NodeProps) => {
  return (
    <div className={`flex items-center gap-2 rounded-full border-2 border-green-500 bg-white dark:bg-[#18232f] px-6 py-3 shadow-lg ${selected ? 'ring-2 ring-green-500/20' : ''}`}>
      <span className="material-symbols-outlined text-green-500">play_circle</span>
      <span className="font-semibold text-gray-900 dark:text-white">Start</span>
      <Handle type="source" position={Position.Right} className="!bg-green-500 !w-3 !h-3" />
    </div>
  );
});

export const EndNode = memo(({ selected }: NodeProps) => {
  return (
    <div className={`flex items-center gap-2 rounded-full border-2 border-red-500 bg-white dark:bg-[#18232f] px-6 py-3 shadow-lg ${selected ? 'ring-2 ring-red-500/20' : ''}`}>
      <Handle type="target" position={Position.Left} className="!bg-red-500 !w-3 !h-3" />
      <span className="material-symbols-outlined text-red-500">stop_circle</span>
      <span className="font-semibold text-gray-900 dark:text-white">End</span>
    </div>
  );
});

export const ActivityNode = memo(({ data, selected }: NodeProps) => {
  return (
    <NodeWrapper selected={selected}>
      <Handle type="target" position={Position.Left} className="!bg-gray-400 !w-3 !h-3" />
      <span className="material-symbols-outlined text-gray-500 dark:text-gray-400 text-3xl">settings</span>
      <span className="font-semibold mt-2 text-gray-900 dark:text-white">{data.label}</span>
      <Handle type="source" position={Position.Right} className="!bg-gray-400 !w-3 !h-3" />
    </NodeWrapper>
  );
});

export const ApprovalNode = memo(({ data, selected }: NodeProps) => {
  return (
    <div className={`relative flex flex-col items-center justify-center rounded-xl border-2 bg-primary/10 p-4 shadow-2xl min-w-[180px] transition-all duration-200 ${selected ? 'border-primary ring-4 ring-primary/20' : 'border-transparent'}`}>
      <Handle type="target" position={Position.Left} className="!bg-primary !w-3 !h-3" />
      <span className="material-symbols-outlined text-primary text-3xl">person</span>
      <span className="font-semibold mt-2 text-primary">{data.label}</span>
      <Handle type="source" position={Position.Right} className="!bg-primary !w-3 !h-3" />
      
      {/* SLA Badge */}
      {data.sla && (
         <div className="absolute -top-3 -right-3 flex items-center gap-1 rounded-full bg-white dark:bg-[#18232f] px-2 py-0.5 text-xs shadow border border-gray-200 dark:border-gray-700">
           <span className="material-symbols-outlined text-xs text-yellow-500">timer</span>
           <span className="font-medium text-gray-500 dark:text-gray-400">{data.sla}</span>
         </div>
      )}
    </div>
  );
});

export const DecisionNode = memo(({ data, selected }: NodeProps) => {
  return (
    <NodeWrapper selected={selected}>
      <Handle type="target" position={Position.Left} className="!bg-gray-400 !w-3 !h-3" />
      <span className="material-symbols-outlined text-gray-500 dark:text-gray-400 text-3xl">call_split</span>
      <span className="font-semibold mt-2 text-gray-900 dark:text-white">{data.label}</span>
      <Handle type="source" position={Position.Right} className="!bg-gray-400 !w-3 !h-3" />
      {/* Decision handles for True/False could be added here or just generic source */}
    </NodeWrapper>
  );
});

export const EventNode = memo(({ data, selected }: NodeProps) => {
  return (
    <NodeWrapper selected={selected}>
      <Handle type="target" position={Position.Left} className="!bg-gray-400 !w-3 !h-3" />
      <span className="material-symbols-outlined text-gray-500 dark:text-gray-400 text-3xl">notifications</span>
      <span className="font-semibold mt-2 text-gray-900 dark:text-white">{data.label}</span>
      <Handle type="source" position={Position.Right} className="!bg-gray-400 !w-3 !h-3" />
    </NodeWrapper>
  );
});
