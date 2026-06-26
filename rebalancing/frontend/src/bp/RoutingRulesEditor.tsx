import React from "react";
import { ConditionBuilder } from "./ConditionBuilder";

export interface RoutingRule {
  condition: string;
  actorRole: string;
}

export interface RoutingRules {
  routes: RoutingRule[];
  fallbackRole: string;
}

interface Props {
  value: RoutingRules | null;
  onChange: (val: RoutingRules) => void;
}

export const RoutingRulesEditor: React.FC<Props> = ({ value, onChange }) => {
  const routes = value?.routes ?? [];
  const fallbackRole = value?.fallbackRole ?? "";

  const updateRoute = (idx: number, patch: Partial<RoutingRule>) => {
    const nextRoutes = [...routes];
    nextRoutes[idx] = { ...nextRoutes[idx], ...patch };
    onChange({ routes: nextRoutes, fallbackRole });
  };

  const addRoute = () => {
    onChange({
      routes: [...routes, { condition: "", actorRole: "" }],
      fallbackRole,
    });
  };

  const removeRoute = (idx: number) => {
    const nextRoutes = routes.filter((_, i) => i !== idx);
    onChange({ routes: nextRoutes, fallbackRole });
  };

  return (
    <div className="space-y-4">
      {routes.map((route, idx) => (
        <div key={idx} className="border p-3 rounded bg-white shadow-sm flex flex-col gap-2">
           <div className="flex justify-between items-center">
            <span className="text-xs font-bold text-gray-500">Route #{idx + 1}</span>
            <button onClick={() => removeRoute(idx)} className="text-red-500 text-xs">Remove</button>
          </div>
          <div className="flex flex-col gap-1">
             <label className="text-xs">Condition</label>
             <ConditionBuilder
                value={route.condition}
                onChange={(val) => updateRoute(idx, { condition: val })}
             />
          </div>
          <div className="flex flex-col gap-1">
             <label className="text-xs">Target Role</label>
             <select
                className="border p-1 text-sm bg-white"
                value={route.actorRole}
                onChange={(e) => updateRoute(idx, { actorRole: e.target.value })}
             >
                <option value="">Select a role...</option>
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
                <optgroup label="System">
                  <option value="PROJECT_OWNER">Project Owner</option>
                </optgroup>
             </select>
          </div>
        </div>
      ))}
      <button 
        onClick={addRoute}
        className="w-full py-1 bg-gray-50 border border-dashed border-gray-300 text-gray-600 text-sm"
      >
        + Add Route
      </button>

      <div className="border-t pt-2 mt-2">
        <label className="text-xs font-bold block mb-1">Fallback Role</label>
        <select
          className="w-full border p-1 text-sm bg-white"
          value={fallbackRole}
          onChange={(e) => onChange({ routes, fallbackRole: e.target.value })}
        >
                <option value="">Select a fallback role...</option>
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
  );
};
