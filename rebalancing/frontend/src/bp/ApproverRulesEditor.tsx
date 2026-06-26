import React from "react";
import { ConditionBuilder } from "./ConditionBuilder";
import { ApproverRule, ApprovalConfig } from "./approvers";

interface Props {
  value: ApprovalConfig;
  onChange: (v: ApprovalConfig) => void;
}

export const ApproverRulesEditor: React.FC<Props> = ({ value, onChange }) => {
  const rules = value.rules || [];

  const updateRule = (id: string, patch: Partial<ApproverRule>) => {
    const nextRules = rules.map((r) => (r.id === id ? { ...r, ...patch } : r));
    onChange({ ...value, rules: nextRules });
  };

  const addRule = () => {
    const rule: ApproverRule = {
      id: crypto.randomUUID(),
      label: `Rule ${rules.length + 1}`,
      actorRole: "",
      condition: null,
    };
    onChange({ ...value, rules: [...rules, rule] });
  };

  const removeRule = (id: string) => {
    onChange({ ...value, rules: rules.filter((r) => r.id !== id) });
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center mb-2">
         <h4 className="text-sm font-bold text-gray-700">Role-Based Approver Rules</h4>
         <button onClick={addRule} className="text-xs px-2 py-1 bg-blue-50 text-blue-600 rounded border border-blue-200">
           + Add Rule
         </button>
      </div>
      
      {rules.map((r) => (
        <div key={r.id} className="approver-rule border rounded p-3 bg-white shadow-sm space-y-3">
          <div className="flex justify-between">
             <input
               className="font-bold text-xs border-b border-gray-300 w-1/2 focus:outline-none focus:border-blue-500"
               value={r.label}
               placeholder="Rule Label (e.g. US High Value)"
               onChange={(e) => updateRule(r.id, { label: e.target.value })}
             />
             <button onClick={() => removeRule(r.id)} className="text-red-500 text-xs">Delete</button>
          </div>

          <div>
             <label className="block text-[10px] uppercase text-gray-400 font-bold mb-1">If condition matches</label>
             <ConditionBuilder
                value={r.condition}
                onChange={(json) => updateRule(r.id, { condition: json })}
             />
          </div>

          <div>
            <label className="block text-[10px] uppercase text-gray-400 font-bold mb-1">Assign to Role</label>
            <select
                className="w-full border rounded p-1 text-sm bg-gray-50"
                value={r.actorRole}
                onChange={(e) => updateRule(r.id, { actorRole: e.target.value })}
              >
                <option value="">Select role...</option>
                <optgroup label="Management">
                  <option value="MANAGER">Manager</option>
                  <option value="DIRECTOR">Director</option>
                  <option value="VP">VP</option>
                </optgroup>
                <optgroup label="Departments">
                  <option value="HR_BP">HR Partner</option>
                  <option value="FINANCE_ANALYST">Finance</option>
                  <option value="LEGAL_COUNSEL">Legal</option>
                </optgroup>
            </select>
          </div>
        </div>
      ))}
      
      {rules.length === 0 && (
        <div className="text-center p-4 border border-dashed rounded text-gray-400 text-xs">
           No specific rules defined. Fallback role will be used.
        </div>
      )}

      <div className="border-t pt-3 mt-2">
        <label className="block text-xs font-bold text-gray-700 mb-1">
          Fallback Role
        </label>
        <select
          className="w-full border rounded p-1 text-sm"
          value={value.fallbackRole ?? ""}
          onChange={(e) => onChange({ ...value, fallbackRole: e.target.value })}
        >
          <option value="">Select fallback role...</option>
           <optgroup label="Management">
                  <option value="MANAGER">Manager</option>
                  <option value="DIRECTOR">Director</option>
                  <option value="VP">VP</option>
                </optgroup>
                <optgroup label="Departments">
                  <option value="HR_BP">HR Partner</option>
                  <option value="FINANCE_ANALYST">Finance</option>
                  <option value="LEGAL_COUNSEL">Legal</option>
                </optgroup>
        </select>
        <p className="text-[10px] text-gray-400 mt-1">Used if no specific rules match.</p>
      </div>
    </div>
  );
};
