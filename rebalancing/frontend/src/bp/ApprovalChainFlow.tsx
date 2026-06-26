import React, { useCallback, useMemo, useState } from "react";
import ReactFlow, {
  Background,
  Controls,
  Edge,
  Node,
  Position,
  ReactFlowProvider,
} from "reactflow";
import "reactflow/dist/style.css";
import { ConditionBuilder } from "./ConditionBuilder";

export type ApprovalLevelId = string;

// Adapted to use string for conditions (Starlark/JSON string) to match backend model
export interface ApprovalLevel {
  id: ApprovalLevelId;
  name: string;
  actorRole: string;
  entryCondition: string; // was entryConditionJson
  skipIf?: string;        // was skipIfJson
  stopCriteria?: string;  // was stopCriteriaJson
}

export interface ApprovalChain {
  levels: ApprovalLevel[];
}

interface Props {
  value: ApprovalChain;
  onChange: (v: ApprovalChain) => void;
}

const nodeWidth = 220;
const nodeHeight = 100;
const verticalGap = 80;

export const ApprovalChainFlow: React.FC<Props> = ({ value, onChange }) => {
  const levels = value?.levels ?? [];
  const [selectedLevelId, setSelectedLevelId] = useState<string | null>(
    levels[0]?.id ?? null
  );

  const { nodes, edges } = useMemo(() => {
    const nodes: Node[] = levels.map((level, index) => ({
      id: level.id,
      data: {
        label: level.name || `Level ${index + 1}`,
        role: level.actorRole,
      },
      position: { x: 0, y: index * (nodeHeight + verticalGap) },
      style: {
        width: nodeWidth,
        height: nodeHeight,
        borderRadius: 8,
        border: "1px solid #ccc",
        padding: 8,
        background: "#fff",
        fontSize: "12px"
      },
      sourcePosition: Position.Bottom,
      targetPosition: Position.Top,
    }));

    const edges: Edge[] = levels.slice(1).map((level, index) => ({
      id: `e-${levels[index].id}-${level.id}`,
      source: levels[index].id,
      target: level.id,
      type: "smoothstep",
      animated: true,
    }));

    return { nodes, edges };
  }, [levels]);

  const updateLevel = useCallback(
    (id: string, patch: Partial<ApprovalLevel>) => {
      const next = levels.map((lvl) => (lvl.id === id ? { ...lvl, ...patch } : lvl));
      onChange({ levels: next });
    },
    [levels, onChange]
  );

  const addLevel = () => {
    const id = `lvl-${Date.now()}`;
    const newLevel: ApprovalLevel = {
      id,
      name: `Level ${levels.length + 1}`,
      actorRole: "",
      entryCondition: "",
    };
    onChange({ levels: [...levels, newLevel] });
    setSelectedLevelId(id);
  };

  const removeLevel = () => {
    if (!selectedLevelId) return;
    const idx = levels.findIndex((l) => l.id === selectedLevelId);
    const next = levels.filter((l) => l.id !== selectedLevelId);
    onChange({ levels: next });
    if (next.length === 0) setSelectedLevelId(null);
    else if (idx > 0) setSelectedLevelId(next[idx - 1].id);
    else setSelectedLevelId(next[0].id);
  };

  const selectedLevel = levels.find((l) => l.id === selectedLevelId) || null;

  return (
    <ReactFlowProvider>
      <div style={{ display: "flex", height: 400, border: "1px solid #eee", borderRadius: 4 }}>
        <div style={{ flex: 1, borderRight: "1px solid #ddd" }}>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodeClick={(_, node) => setSelectedLevelId(node.id)}
            fitView
          >
            <Background />
            <Controls />
          </ReactFlow>
        </div>
        <div style={{ width: 360, padding: 12, overflowY: "auto", background: "#f9fafb" }}>
          <div style={{ marginBottom: 12, display: "flex", gap: 8 }}>
            <button onClick={addLevel} className="px-3 py-1 bg-blue-600 text-white text-xs rounded hover:bg-blue-700">Add Level</button>
            <button onClick={removeLevel} disabled={!selectedLevel} className="px-3 py-1 bg-white border text-red-600 text-xs rounded hover:bg-gray-50 disabled:opacity-50">
              Remove Selected
            </button>
          </div>
          {selectedLevel ? (
            <ApprovalLevelInspector
              level={selectedLevel}
              onChange={(patch) =>
                updateLevel(selectedLevel.id, patch as Partial<ApprovalLevel>)
              }
            />
          ) : (
            <div className="text-gray-400 text-sm text-center mt-10">Select a level to edit settings</div>
          )}
        </div>
      </div>
    </ReactFlowProvider>
  );
};

interface LevelInspectorProps {
  level: ApprovalLevel;
  onChange: (patch: Partial<ApprovalLevel>) => void;
}

const ApprovalLevelInspector: React.FC<LevelInspectorProps> = ({ level, onChange }) => (
  <div className="space-y-4">
    <div>
      <h4 className="font-bold text-sm mb-2 text-gray-800">{level.name}</h4>
      <label className="block text-xs font-semibold text-gray-600 mb-1">
        Level name
      </label>
      <input
        className="w-full border rounded p-1 text-sm"
        value={level.name}
        onChange={(e) => onChange({ name: e.target.value })}
      />
    </div>
    
    <div>
      <label className="block text-xs font-semibold text-gray-600 mb-1">
        Approver Role
      </label>
      <select
        className="w-full border rounded p-1 text-sm bg-white"
        value={level.actorRole}
        onChange={(e) => onChange({ actorRole: e.target.value })}
      >
        <option value="">Select a role...</option>
        <optgroup label="Management">
          <option value="MANAGER">Manager (Direct Supervisor)</option>
          <option value="DIRECTOR">Director</option>
          <option value="VP">VP</option>
          <option value="CFO">CFO</option>
        </optgroup>
        <optgroup label="Departments">
          <option value="HR_BP">HR Business Partner</option>
          <option value="FINANCE_ANALYST">Finance Analyst</option>
          <option value="LEGAL_COUNSEL">Legal Counsel</option>
          <option value="COMPLIANCE_OFFICER">Compliance Officer</option>
        </optgroup>
        <optgroup label="Dynamic/System">
          <option value="PROJECT_OWNER">Project Owner</option>
          <option value="COST_CENTER_OWNER">Cost Center Owner</option>
        </optgroup>
      </select>
      <div className="text-[10px] text-gray-400 mt-1">
        * Role resolution is dynamic based on requester's hierarchy.
      </div>
    </div>

    <section>
      <h5 className="text-xs font-bold text-gray-700 mb-1 mt-2">Entry condition</h5>
      <ConditionBuilder
        value={level.entryCondition || ""}
        onChange={(val) => onChange({ entryCondition: val })}
      />
    </section>

    <section>
      <h5 className="text-xs font-bold text-gray-700 mb-1 mt-2">Skip if</h5>
      <ConditionBuilder
        value={level.skipIf || ""}
        onChange={(val) => onChange({ skipIf: val })}
      />
    </section>

    <section>
      <h5 className="text-xs font-bold text-gray-700 mb-1 mt-2">Stop approval chain when</h5>
      <ConditionBuilder
        value={level.stopCriteria || ""}
        onChange={(val) => onChange({ stopCriteria: val })}
      />
    </section>
  </div>
);
