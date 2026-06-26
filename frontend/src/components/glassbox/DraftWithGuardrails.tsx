import React from 'react';

interface Violation {
  ruleId: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  message: string;
  start: number;
  end: number;
}

interface DraftProps {
  content: string;
  violations: Violation[];
}

export const DraftWithGuardrails: React.FC<DraftProps> = ({ content, violations }) => {
  // Simple rendering logic for demo - in prod use a proper text overlay lib
  return (
    <div className="bg-gray-900 p-4 rounded-lg mt-4">
      <h3 className="text-lg font-bold text-white mb-2">Draft Content & Guardrails</h3>
      <div className="grid grid-cols-3 gap-4">
        <div className="col-span-2 bg-black p-4 rounded font-mono text-sm text-gray-300 whitespace-pre-wrap relative">
          {content}
          {/* Overlay simulation */}
          {violations.length > 0 && (
            <div className="absolute top-4 right-4">
               <span className="text-red-500 text-xs">⚠ {violations.length} Violations Detected</span>
            </div>
          )}
        </div>
        <div className="col-span-1 space-y-2">
          {violations.map((v, idx) => (
            <div key={idx} className={`p-3 rounded border ${
              v.severity === 'critical' ? 'bg-red-900/20 border-red-500' : 'bg-yellow-900/20 border-yellow-500'
            }`}>
              <div className="flex justify-between items-center mb-1">
                <span className="font-bold text-xs uppercase">{v.ruleId}</span>
                <span className="text-[10px] px-1.5 py-0.5 bg-black/50 rounded">{v.severity}</span>
              </div>
              <p className="text-xs text-gray-400">{v.message}</p>
              <button className="mt-2 text-xs text-blue-400 hover:text-blue-300 underline">
                Apply Fix
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
