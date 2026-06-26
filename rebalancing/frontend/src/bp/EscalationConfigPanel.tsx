import React from "react";
import { ConditionBuilder } from "./ConditionBuilder";
import { EscalationStep } from "./escalationTypes";

interface Props {
  escalations: EscalationStep[];
  onChange: (escalations: EscalationStep[]) => void;
}

export const EscalationConfigPanel: React.FC<Props> = ({ escalations, onChange }) => {
  const addEscalation = () => {
    const newStep: EscalationStep = {
      id: crypto.randomUUID(),
      stepNumber: (escalations?.length ?? 0) + 1,
      delayAfterPreviousExpr: "",
      targetActorRole: "",
      notificationTemplate: "default",
    };
    onChange([...(escalations ?? []), newStep]);
  };

  const updateEscalation = (id: string, patch: Partial<EscalationStep>) => {
    onChange(
      (escalations ?? []).map((e) => (e.id === id ? { ...e, ...patch } : e))
    );
  };

  const removeEscalation = (id: string) => {
    onChange((escalations ?? []).filter((e) => e.id !== id));
  };

  return (
    <div className="escalation-config" style={{ marginTop: '20px', borderTop: '1px solid #eee', paddingTop: '10px' }}>
      <h4 style={{ fontSize: '0.9rem', fontWeight: 600, marginBottom: '12px', color: '#475569' }}>Escalation Chain</h4>
      {(escalations ?? []).map((step, idx) => (
        <div key={step.id} className="escalation-step-card" style={{ background: '#fff', border: '1px solid #cbd5e1', borderRadius: '6px', padding: '12px', marginBottom: '10px' }}>
          <div className="step-header" style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px', fontSize: '12px', fontWeight: 'bold' }}>
            <span className="step-number">Step {idx + 1}</span>
          </div>

          <div className="step-form">
            <div className="form-row" style={{ marginBottom: '8px' }}>
              <label style={{ display: 'block', fontSize: '11px', color: '#64748b' }}>Delay after previous (Starlark)</label>
              <textarea
                value={step.delayAfterPreviousExpr}
                onChange={(e) =>
                  updateEscalation(step.id, { delayAfterPreviousExpr: e.target.value })
                }
                placeholder='e.g., "def delay_seconds(ctx): return hours(24)"'
                style={{ width: "100%", minHeight: "50px", fontFamily: "monospace", fontSize: "11px", border: '1px solid #cbd5e1', borderRadius: '4px' }}
              />
            </div>

            <div className="form-row" style={{ marginBottom: '8px' }}>
              <label style={{ display: 'block', fontSize: '11px', color: '#64748b' }}>Escalate to role</label>
              <input
                value={step.targetActorRole}
                onChange={(e) =>
                  updateEscalation(step.id, { targetActorRole: e.target.value })
                }
                placeholder="e.g., Director, C-Suite"
                style={{ width: "100%", fontSize: "12px", border: '1px solid #cbd5e1', borderRadius: '4px', padding: '4px' }}
              />
            </div>

            <div className="form-row" style={{ marginBottom: '8px' }}>
              <label style={{ display: 'block', fontSize: '11px', color: '#64748b' }}>Notification template</label>
              <select
                value={step.notificationTemplate ?? "default"}
                onChange={(e) =>
                  updateEscalation(step.id, { notificationTemplate: e.target.value })
                }
                style={{ width: "100%", fontSize: "12px", border: '1px solid #cbd5e1', borderRadius: '4px', padding: '4px' }}
              >
                <option value="default">Default (reminder + escalation)</option>
                <option value="urgent">Urgent (high priority)</option>
                <option value="exec">Executive summary</option>
              </select>
            </div>

            <div className="form-row" style={{ marginBottom: '8px' }}>
              <label style={{ display: 'block', fontSize: '11px', color: '#64748b' }}>Condition (optional)</label>
              <ConditionBuilder
                value={step.condition ?? { operator: "AND", conditions: [] }}
                onChange={(cond) => updateEscalation(step.id, { condition: cond })}
              />
            </div>

            <button
              onClick={() => removeEscalation(step.id)}
              className="btn-delete"
              style={{ fontSize: "11px", color: "red", background: "none", border: "none", cursor: "pointer", padding: 0 }}
            >
              Remove step
            </button>
          </div>
        </div>
      ))}

      <button onClick={addEscalation} className="btn-secondary" style={{ width: "100%", padding: "6px", fontSize: "12px", background: "#f1f5f9", border: "1px solid #cbd5e1", borderRadius: "4px", cursor: "pointer" }}>
        + Add escalation step
      </button>
    </div>
  );
};
