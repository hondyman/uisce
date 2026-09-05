import React, { useState, useEffect, useCallback } from 'react';
import type { BOField } from './BOGovernanceStudio';
import './BOGovernanceStudio.css';

// ─── Types ────────────────────────────────────────────────────────────────────

type TriggerEvent = 'ON_SAVE' | 'ON_SUBMIT' | 'ON_READ' | 'ON_DELETE' | 'ON_FIELD_CHANGE';
type ActionType = 'BLOCK' | 'REQUIRE_APPROVAL' | 'NOTIFY_ROLE' | 'ESCALATE' | 'COMPUTE_FIELD';

interface PolicyRule {
  policy_id: string;
  bo_key: string;
  policy_name: string;
  description: string;
  trigger_event: TriggerEvent;
  condition_expr: string;
  action_type: ActionType;
  action_config: Record<string, unknown>;
  priority: number;
  is_active: boolean;
  is_core: boolean;
}

interface SimResult {
  triggered: boolean;
  error?: string;
}

interface PolicyRuleBuilderProps {
  tenantId: string;
  boKey: string;
  fields: BOField[];
  onCountChange?: (count: number) => void;
}

const TRIGGER_LABELS: Record<TriggerEvent, string> = {
  ON_SAVE: '💾 On Save',
  ON_SUBMIT: '📤 On Submit',
  ON_READ: '👁 On Read',
  ON_DELETE: '🗑 On Delete',
  ON_FIELD_CHANGE: '✏️ On Field Change',
};

const ACTION_META: Record<ActionType, { label: string; color: string; desc: string }> = {
  BLOCK:            { label: '🚫 Block',            color: '#f85149', desc: 'Immediately reject the operation' },
  REQUIRE_APPROVAL: { label: '✅ Require Approval', color: '#d29922', desc: 'Hold for human review before proceeding' },
  NOTIFY_ROLE:      { label: '🔔 Notify Role',      color: '#2f81f7', desc: 'Send a notification to a role group' },
  ESCALATE:         { label: '⬆ Escalate',          color: '#bc8cff', desc: 'Escalate to a senior approver' },
  COMPUTE_FIELD:    { label: '⚡ Compute Field',    color: '#3fb950', desc: 'Derive a field value from an expression' },
};

const EMPTY_POLICY: Omit<PolicyRule, 'policy_id'> = {
  bo_key: '',
  policy_name: '',
  description: '',
  trigger_event: 'ON_SAVE',
  condition_expr: 'record.status == "PENDING" && record.amount > 10000',
  action_type: 'REQUIRE_APPROVAL',
  action_config: {},
  priority: 100,
  is_active: true,
  is_core: false,
};

// ─── PolicyRuleBuilder Component ──────────────────────────────────────────────

const PolicyRuleBuilder: React.FC<PolicyRuleBuilderProps> = ({
  tenantId, boKey, fields, onCountChange,
}) => {
  const [policies, setPolicies] = useState<PolicyRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Partial<PolicyRule> | null>(null);
  const [isNew, setIsNew] = useState(false);
  const [activeEvent, setActiveEvent] = useState<TriggerEvent>('ON_SAVE');
  const [simRecord, setSimRecord] = useState('{\n  "status": "PENDING",\n  "amount": 50000\n}');
  const [simActor, setSimActor] = useState('{\n  "id": "user-123",\n  "roles": ["ANALYST"]\n}');
  const [simResult, setSimResult] = useState<SimResult | null>(null);
  const [simLoading, setSimLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const headers = useCallback(() => ({
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
  }), [tenantId]);

  const load = useCallback(async (event: TriggerEvent = activeEvent) => {
    setLoading(true);
    try {
      const res = await fetch(
        `/api/rule-fabric/bo/${boKey}/policies?event=${event}`,
        { headers: headers() }
      );
      if (res.ok) {
        const data = await res.json();
        const list = Array.isArray(data) ? data : [];
        setPolicies(list);
        onCountChange?.(list.length);
      }
    } finally {
      setLoading(false);
    }
  }, [boKey, headers, activeEvent, onCountChange]);

  useEffect(() => { load(activeEvent); }, [activeEvent, load]);

  const handleNew = () => {
    setEditing({ ...EMPTY_POLICY, trigger_event: activeEvent });
    setIsNew(true);
    setSimResult(null);
  };

  const handleSave = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      const url = isNew
        ? `/api/rule-fabric/bo/${boKey}/policies`
        : `/api/rule-fabric/bo/${boKey}/policies/${editing.policy_id}`;
      const res = await fetch(url, {
        method: isNew ? 'POST' : 'PUT',
        headers: headers(),
        body: JSON.stringify({ ...editing, bo_key: boKey }),
      });
      if (res.ok) { setEditing(null); await load(); }
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (policyId: string) => {
    if (!confirm('Delete this policy?')) return;
    await fetch(`/api/rule-fabric/bo/${boKey}/policies/${policyId}`, {
      method: 'DELETE', headers: headers(),
    });
    await load();
  };

  const handleSimulate = async () => {
    if (!editing?.condition_expr) return;
    setSimLoading(true);
    setSimResult(null);
    try {
      let record: unknown, actor: unknown;
      try { record = JSON.parse(simRecord); actor = JSON.parse(simActor); }
      catch { setSimResult({ triggered: false, error: 'Invalid JSON' }); return; }
      const res = await fetch(`/api/rule-fabric/bo/${boKey}/policies/simulate`, {
        method: 'POST', headers: headers(),
        body: JSON.stringify({ condition_expr: editing.condition_expr, record, actor }),
      });
      if (res.ok) setSimResult(await res.json());
    } finally {
      setSimLoading(false);
    }
  };

  const events = Object.keys(TRIGGER_LABELS) as TriggerEvent[];

  return (
    <div>
      <div className="bog-section-header">
        <div>
          <div className="bog-section-title">⚖️ Policy Rules</div>
          <div className="bog-section-desc">
            WHEN/THEN declarative policies: condition expressions trigger workflow actions.
          </div>
        </div>
        <button className="bog-btn bog-btn-primary" onClick={handleNew}>+ New Policy</button>
      </div>

      {/* ── Trigger Event Filter ── */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 20, flexWrap: 'wrap' }}>
        {events.map(ev => (
          <button key={ev}
            onClick={() => setActiveEvent(ev)}
            className="bog-btn"
            style={{
              background: activeEvent === ev ? 'rgba(47,129,247,0.2)' : 'var(--bog-surface)',
              border: `1px solid ${activeEvent === ev ? 'var(--bog-accent)' : 'var(--bog-border)'}`,
              color: activeEvent === ev ? 'var(--bog-accent)' : 'var(--bog-text-muted)',
              fontSize: 12,
            }}>
            {TRIGGER_LABELS[ev]}
          </button>
        ))}
      </div>

      {/* ── Edit Panel ── */}
      {editing && (
        <div className="bog-card" style={{ marginBottom: 20, borderColor: 'var(--bog-border-glow)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <span style={{ fontWeight: 700, fontSize: 15 }}>{isNew ? '+ New Policy Rule' : 'Edit Policy'}</span>
            <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={() => setEditing(null)}>✕</button>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
            <div className="bog-input-group">
              <label className="bog-input-label">Policy Name</label>
              <input className="bog-input" value={editing.policy_name ?? ''} placeholder="e.g. Large Trade Approval"
                onChange={e => setEditing(p => ({ ...p!, policy_name: e.target.value }))} />
            </div>
            <div className="bog-input-group">
              <label className="bog-input-label">Trigger Event</label>
              <select className="bog-select" value={editing.trigger_event}
                onChange={e => setEditing(p => ({ ...p!, trigger_event: e.target.value as TriggerEvent }))}>
                {events.map(ev => <option key={ev} value={ev}>{TRIGGER_LABELS[ev]}</option>)}
              </select>
            </div>
          </div>

          {/* ── WHEN Condition ── */}
          <div className="bog-condition-group">
            <div style={{ fontWeight: 700, fontSize: 12, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--bog-text-muted)', marginBottom: 8 }}>
              WHEN — Condition (CEL)
            </div>
            <textarea className="bog-textarea"
              value={editing.condition_expr ?? ''}
              placeholder={'record.amount > 50000 && actor.roles.exists(r, r == "ANALYST")'}
              style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
              onChange={e => setEditing(p => ({ ...p!, condition_expr: e.target.value }))} />
            <div style={{ fontSize: 11, color: 'var(--bog-text-muted)', marginTop: 4 }}>
              Variables: <code style={{ color: 'var(--bog-accent)' }}>record</code> · <code style={{ color: 'var(--bog-accent)' }}>actor</code> · <code style={{ color: 'var(--bog-accent)' }}>changes</code>
            </div>
          </div>

          {/* ── THEN Action ── */}
          <div className="bog-action-row">
            <span className="bog-action-label">THEN</span>
            <select className="bog-select" style={{ flex: '0 0 200px' }} value={editing.action_type}
              onChange={e => setEditing(p => ({ ...p!, action_type: e.target.value as ActionType }))}>
              {(Object.keys(ACTION_META) as ActionType[]).map(at => (
                <option key={at} value={at}>{ACTION_META[at].label}</option>
              ))}
            </select>
            <span style={{ fontSize: 12, color: 'var(--bog-text-muted)', flex: 1 }}>
              {ACTION_META[editing.action_type ?? 'BLOCK']?.desc}
            </span>
          </div>

          {/* NOTIFY_ROLE / ESCALATE config */}
          {(editing.action_type === 'NOTIFY_ROLE' || editing.action_type === 'ESCALATE') && (
            <div className="bog-input-group" style={{ marginTop: 12 }}>
              <label className="bog-input-label">Target Role</label>
              <input className="bog-input" value={(editing.action_config as Record<string, string>)?.role ?? ''}
                placeholder="COMPLIANCE_OFFICER"
                onChange={e => setEditing(p => ({ ...p!, action_config: { ...p!.action_config, role: e.target.value } }))} />
            </div>
          )}

          {/* ── Simulation Panel ── */}
          <div className="bog-test-panel">
            <div style={{ fontWeight: 700, fontSize: 12, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--bog-text-muted)', marginBottom: 10 }}>
              ⚡ Policy Simulation
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div className="bog-input-group">
                <label className="bog-input-label">Record</label>
                <textarea className="bog-textarea" value={simRecord} onChange={e => setSimRecord(e.target.value)} />
              </div>
              <div className="bog-input-group">
                <label className="bog-input-label">Actor</label>
                <textarea className="bog-textarea" value={simActor} onChange={e => setSimActor(e.target.value)} />
              </div>
            </div>
            <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={handleSimulate} disabled={simLoading}>
              {simLoading ? '⏳ Simulating…' : '▶ Simulate'}
            </button>
            {simResult && (
              <div className={`bog-test-result ${simResult.error ? 'error' : simResult.triggered ? 'fail' : 'pass'}`}>
                {simResult.error
                  ? `⚠ Error: ${simResult.error}`
                  : simResult.triggered
                    ? `⚡ TRIGGERED — Policy fires (${ACTION_META[editing.action_type ?? 'BLOCK']?.label})`
                    : '○ NOT triggered — Condition evaluated to false'}
              </div>
            )}
          </div>

          <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
            <button className="bog-btn bog-btn-primary" onClick={handleSave} disabled={saving}>
              {saving ? '⏳ Saving…' : (isNew ? '✓ Create Policy' : '✓ Save')}
            </button>
            <button className="bog-btn bog-btn-secondary" onClick={() => setEditing(null)}>Cancel</button>
          </div>
        </div>
      )}

      {/* ── Policy List ── */}
      {loading ? (
        <div className="bog-loading" style={{ padding: 40 }}><div className="bog-spinner" /></div>
      ) : policies.length === 0 && !editing ? (
        <div className="bog-empty">
          <div className="bog-empty-icon">⚖️</div>
          <div className="bog-empty-title">No policies for {TRIGGER_LABELS[activeEvent]}</div>
          <div className="bog-empty-desc">Create WHEN/THEN policies to enforce business rules, require approvals, or trigger notifications.</div>
          <button className="bog-btn bog-btn-primary" onClick={handleNew} style={{ marginTop: 8 }}>+ New Policy</button>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {policies.map(p => {
            const meta = ACTION_META[p.action_type];
            return (
              <div key={p.policy_id} className="bog-card" style={{ borderLeft: `3px solid ${meta.color}` }}>
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14 }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 6 }}>
                      <span style={{ fontWeight: 700, fontSize: 14 }}>{p.policy_name}</span>
                      <span style={{ background: `${meta.color}22`, color: meta.color, border: `1px solid ${meta.color}44`, borderRadius: 4, padding: '1px 8px', fontSize: 11, fontWeight: 700 }}>
                        {meta.label}
                      </span>
                      {p.is_core && <span className="bog-badge-core">CORE</span>}
                    </div>
                    <div style={{ fontSize: 12, fontFamily: 'monospace', color: 'var(--bog-text-muted)', marginBottom: 4 }}>
                      WHEN: {p.condition_expr}
                    </div>
                    {p.description && <div style={{ fontSize: 12, color: 'var(--bog-text-muted)' }}>{p.description}</div>}
                  </div>
                  {!p.is_core && (
                    <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
                      <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={() => { setEditing(p); setIsNew(false); }}>Edit</button>
                      <button className="bog-btn bog-btn-danger bog-btn-sm" onClick={() => handleDelete(p.policy_id)}>Delete</button>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default PolicyRuleBuilder;
