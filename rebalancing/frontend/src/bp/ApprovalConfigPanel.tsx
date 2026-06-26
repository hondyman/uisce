import React from "react";
import { ConditionBuilder } from "./ConditionBuilder";
import { ApprovalNodeData } from "./ApprovalNode";
import { EscalationConfigPanel } from "./EscalationConfigPanel";
import { EscalationStep } from "./escalationTypes"; // Import types

interface Props {
  node: any; // React Flow node
  onUpdate: (nodeId: string, data: ApprovalNodeData) => void;
}

export const ApprovalConfigPanel: React.FC<Props> = ({ node, onUpdate }) => {
  // Safe cast and default initialization
  const data = node.data as ApprovalNodeData;
  const chain = data.approvalChain || { rules: [], fallbackRole: "" };

  const updateApprovalChain = (patch: any) => {
    const newData: ApprovalNodeData = {
      ...data,
      approvalChain: { ...chain, ...patch },
    };
    onUpdate(node.id, newData);
  };
  
  const updateData = (patch: Partial<ApprovalNodeData>) => {
      onUpdate(node.id, { ...data, ...patch });
  }

  const updateEscalations = (escalations: EscalationStep[]) => {
      onUpdate(node.id, { ...data, escalations });
  };

  const addRule = () => {
    const newRule = {
      id: crypto.randomUUID(),
      label: `Rule ${(chain.rules?.length ?? 0) + 1}`,
      condition: null,
      actorRole: "",
    };
    const rules = [...(chain.rules ?? []), newRule];
    updateApprovalChain({ rules });
  };

  const updateRule = (ruleId: string, patch: any) => {
    const rules = (chain.rules ?? []).map((r) =>
      r.id === ruleId ? { ...r, ...patch } : r
    );
    updateApprovalChain({ rules });
  };

  const removeRule = (ruleId: string) => {
    const rules = (chain.rules ?? []).filter((r) => r.id !== ruleId);
    updateApprovalChain({ rules });
  };

  return (
    <div className="approval-config-panel" style={{ padding: '16px' }}>
      <h3 style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '16px' }}>Approval Configuration: {data.stepKey}</h3>

      <div className="config-section" style={{ marginBottom: '24px' }}>
        <h4 style={{ fontSize: '0.9rem', fontWeight: 600, marginBottom: '12px', color: '#475569' }}>Approver Rules</h4>
        {(chain.rules ?? []).map((rule) => (
          <div key={rule.id} className="rule-card" style={{ background: 'white', padding: '12px', border: '1px solid #e2e8f0', borderRadius: '6px', marginBottom: '12px' }}>
            <div className="rule-row" style={{ marginBottom: '8px' }}>
              <label style={{ display: 'block', fontSize: '12px', color: '#64748b', marginBottom: '4px' }}>Rule Label</label>
              <input
                value={rule.label}
                onChange={(e) => updateRule(rule.id, { label: e.target.value })}
                placeholder="e.g., US High Value"
                style={{ width: '100%', padding: '6px', fontSize: '12px', border: '1px solid #cbd5e1', borderRadius: '4px' }}
              />
            </div>

            <div className="rule-row" style={{ marginBottom: '8px' }}>
              <label style={{ display: 'block', fontSize: '12px', color: '#64748b', marginBottom: '4px' }}>Actor Role</label>
              <input
                value={rule.actorRole}
                onChange={(e) => updateRule(rule.id, { actorRole: e.target.value })}
                placeholder="e.g., Manager, Director, Compliance"
                style={{ width: '100%', padding: '6px', fontSize: '12px', border: '1px solid #cbd5e1', borderRadius: '4px' }}
              />
            </div>

            <div className="rule-row" style={{ marginBottom: '12px' }}>
              <label style={{ display: 'block', fontSize: '12px', color: '#64748b', marginBottom: '4px' }}>Condition</label>
              <ConditionBuilder
                value={rule.condition ?? { operator: "AND", conditions: [] }} // Provide default if null
                onChange={(cond) => updateRule(rule.id, { condition: cond })}
              />
            </div>

            <button
              onClick={() => removeRule(rule.id)}
              className="btn-delete"
              style={{ padding: '4px 8px', fontSize: '11px', color: '#ef4444', background: 'none', border: '1px solid #ef4444', borderRadius: '4px', cursor: 'pointer' }}
            >
              Delete Rule
            </button>
          </div>
        ))}

        <button onClick={addRule} className="btn-primary" style={{ marginTop: '8px', padding: '8px 12px', background: '#3b82f6', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer', fontSize: '12px' }}>
          Add Approver Rule
        </button>
      </div>

      <div className="config-section" style={{ marginBottom: '24px' }}>
        <h4 style={{ fontSize: '0.9rem', fontWeight: 600, marginBottom: '8px', color: '#475569' }}>Settings</h4>
        <div style={{ marginBottom: '12px' }}>
             <label style={{ display: 'block', fontSize: '12px', color: '#64748b', marginBottom: '4px' }}>Fallback Role</label>
            <input
            value={chain.fallbackRole ?? ""}
            onChange={(e) =>
                updateApprovalChain({ fallbackRole: e.target.value })
            }
            placeholder="e.g., Compliance"
            style={{ width: '100%', padding: '6px', fontSize: '12px', border: '1px solid #cbd5e1', borderRadius: '4px' }}
            />
        </div>
      </div>
      
      {/* New Escalation Section */}
      <EscalationConfigPanel 
         escalations={data.escalations || []}
         onChange={updateEscalations}
      />

      <div className="config-section" style={{ marginTop: '20px' }}>
        <h4 style={{ fontSize: '0.9rem', fontWeight: 600, marginBottom: '8px', color: '#475569' }}>Overall SLA (optional)</h4>
        <label style={{ display: 'block', fontSize: '12px', color: '#64748b', marginBottom: '4px' }}>Last Chance SLA Expression (Starlark)</label>
        <textarea
          value={data.slaExpr || ""}
          onChange={(e) => updateData({ slaExpr: e.target.value })}
          placeholder='def sla_seconds(ctx): return hours(72)'
          style={{ minHeight: "60px", width: "100%", fontFamily: "monospace", fontSize: '12px', padding: '8px', border: '1px solid #cbd5e1', borderRadius: '4px' }}
        />
      </div>
    </div>
  );
};
