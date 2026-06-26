import React from "react";
import { ConditionBuilder } from "./ConditionBuilder";

export interface ApprovalLevel {
  name: string;
  actorRole: string;
  entryCondition: string;
  skipIf?: string;
  stopCriteria?: string;
}

export interface ApprovalChain {
  levels: ApprovalLevel[];
}

interface Props {
  value: ApprovalChain | null;
  onChange: (chain: ApprovalChain) => void;
}

export const ApprovalChainEditor: React.FC<Props> = ({ value, onChange }) => {
  const levels = value?.levels ?? [];

  const updateLevel = (idx: number, patch: Partial<ApprovalLevel>) => {
    const next = [...levels];
    next[idx] = { ...next[idx], ...patch };
    onChange({ levels: next });
  };

  const addLevel = () => {
    onChange({
      levels: [...levels, { name: "", actorRole: "", entryCondition: "" }],
    });
  };

  const removeLevel = (idx: number) => {
    const next = levels.filter((_, i) => i !== idx);
    onChange({ levels: next });
  };

  return (
    <div className="space-y-4">
      {levels.map((level, idx) => (
        <div key={idx} className="border p-3 rounded bg-white shadow-sm">
          <div className="flex justify-between mb-2">
            <h5 className="font-bold">Level {idx + 1}</h5>
            <button onClick={() => removeLevel(idx)} className="text-red-500 text-xs">Remove</button>
          </div>
          
          <div className="grid grid-cols-2 gap-2 mb-2">
            <input
              className="border p-1 text-sm"
              value={level.name}
              placeholder="Level Name (e.g. Manager)"
              onChange={(e) => updateLevel(idx, { name: e.target.value })}
            />
            <input
              className="border p-1 text-sm"
              value={level.actorRole}
              placeholder="Actor Role (e.g. MANAGER)"
              onChange={(e) => updateLevel(idx, { actorRole: e.target.value })}
            />
          </div>

          <div className="space-y-2">
            <div>
              <label className="text-xs font-semibold">Entry Condition</label>
              <ConditionBuilder
                value={level.entryCondition}
                onChange={(val) => updateLevel(idx, { entryCondition: val })}
              />
            </div>
            <div>
              <label className="text-xs font-semibold">Skip If</label>
              <ConditionBuilder
                value={level.skipIf || ""}
                onChange={(val) => updateLevel(idx, { skipIf: val })}
              />
            </div>
            <div>
              <label className="text-xs font-semibold">Stop Criteria</label>
              <ConditionBuilder
                value={level.stopCriteria || ""}
                onChange={(val) => updateLevel(idx, { stopCriteria: val })}
              />
            </div>
          </div>
        </div>
      ))}
      <button 
        onClick={addLevel}
        className="w-full py-2 bg-blue-50 text-blue-600 border border-blue-200 rounded text-sm hover:bg-blue-100"
      >
        + Add Approval Level
      </button>
    </div>
  );
};
