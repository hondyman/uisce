import React, { useState } from "react";

interface Props {
  delayExpr: string;
  slaExpr: string;
  onChange: (timing: { delayExpr: string; slaExpr: string }) => void;
}

export const TimingEditor: React.FC<Props> = ({ delayExpr, slaExpr, onChange }) => {
  // Heuristic: If expressions look like integers, default to Simple mode. Else Advanced.
  const isSimple = (s: string) => !s || /^\d+$/.test(s);
  const [mode, setMode] = useState<"simple" | "advanced">(
    (isSimple(delayExpr) && isSimple(slaExpr)) ? "simple" : "advanced"
  );

  return (
    <div className="border p-3 rounded bg-white shadow-sm space-y-3">
      <div className="flex items-center justify-between">
        <label className="text-sm font-bold text-gray-700">Mode</label>
        <select 
          className="border rounded text-sm p-1"
          value={mode} 
          onChange={(e) => setMode(e.target.value as any)}
        >
          <option value="simple">Simple (Hours)</option>
          <option value="advanced">Advanced (Starlark)</option>
        </select>
      </div>

      {mode === "simple" ? (
        <>
          <div className="flex flex-col gap-1">
            <label className="text-xs font-semibold">Delay (hours)</label>
            <input
              type="number"
              min={0}
              className="border p-1 text-sm"
              value={delayExpr}
              placeholder="0"
              onChange={(e) => onChange({ delayExpr: e.target.value, slaExpr })}
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs font-semibold">SLA (hours)</label>
            <input
              type="number"
              min={0}
              className="border p-1 text-sm"
              value={slaExpr}
              placeholder="0"
              onChange={(e) => onChange({ delayExpr, slaExpr: e.target.value })}
            />
          </div>
        </>
      ) : (
        <>
          <div className="flex flex-col gap-1">
            <label className="text-xs font-semibold">Delay Expression (Starlark)</label>
            <textarea
              className="border p-1 text-sm h-16 font-mono"
              value={delayExpr}
              onChange={(e) => onChange({ delayExpr: e.target.value, slaExpr })}
              placeholder="hours(24) if num_field('req','risk') > 80 else hours(1)"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs font-semibold">SLA Expression (Starlark)</label>
            <textarea
              className="border p-1 text-sm h-16 font-mono"
              value={slaExpr}
              onChange={(e) => onChange({ delayExpr, slaExpr: e.target.value })}
              placeholder="days(3)"
            />
          </div>
        </>
      )}
    </div>
  );
};
