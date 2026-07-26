import React, { useState, useEffect, useCallback } from 'react';
import type { BOField } from './BOGovernanceStudio';
import './BOGovernanceStudio.css';

// ─── Types ───────────────────────────────────────────────────────────────────

interface ValidationRule {
  rule_id: string;
  bo_key: string;
  field_key?: string;
  rule_name: string;
  description: string;
  expression: string;
  error_message: string;
  severity: 'ERROR' | 'WARNING' | 'INFO';
  priority: number;
  is_active: boolean;
  is_core: boolean;
}

interface TestResult {
  passed: boolean;
  output: string;
  error?: string;
}

interface ValidationRuleBuilderProps {
  tenantId: string;
  boKey: string;
  fields: BOField[];
  onCountChange?: (count: number) => void;
}

const EMPTY_RULE: Omit<ValidationRule, 'rule_id'> = {
  bo_key: '',
  field_key: undefined,
  rule_name: '',
  description: '',
  expression: 'record.has("field_key")',
  error_message: 'Validation failed.',
  severity: 'ERROR',
  priority: 100,
  is_active: true,
  is_core: false,
};

// ─── ValidationRuleBuilder ────────────────────────────────────────────────────

const ValidationRuleBuilder: React.FC<ValidationRuleBuilderProps> = ({
  tenantId, boKey, fields, onCountChange,
}) => {
  const [rules, setRules] = useState<ValidationRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Partial<ValidationRule> | null>(null);
  const [isNew, setIsNew] = useState(false);
  const [testSample, setTestSample] = useState('{\n  "field_key": "value"\n}');
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [testLoading, setTestLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const headers = useCallback(() => ({
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
  }), [tenantId]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/bo/${boKey}/governance/validation-rules`, { headers: headers() });
      if (res.ok) {
        const data = await res.json();
        const list = Array.isArray(data) ? data : [];
        setRules(list);
        onCountChange?.(list.length);
      }
    } finally {
      setLoading(false);
    }
  }, [boKey, headers, onCountChange]);

  useEffect(() => { load(); }, [load]);

  const handleNew = () => {
    setEditing({ ...EMPTY_RULE });
    setIsNew(true);
    setTestResult(null);
  };

  const handleEdit = (rule: ValidationRule) => {
    setEditing({ ...rule });
    setIsNew(false);
    setTestResult(null);
  };

  const handleSave = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      const url = isNew
        ? `/api/v1/bo/${boKey}/governance/validation-rules`
        : `/api/v1/bo/${boKey}/governance/validation-rules/${editing.rule_id}`;
      const method = isNew ? 'POST' : 'PUT';
      const res = await fetch(url, {
        method,
        headers: headers(),
        body: JSON.stringify({ ...editing, bo_key: boKey }),
      });
      if (res.ok) {
        setEditing(null);
        await load();
      }
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (ruleId: string) => {
    if (!confirm('Delete this validation rule?')) return;
    await fetch(`/api/v1/bo/${boKey}/governance/validation-rules/${ruleId}`, {
      method: 'DELETE', headers: headers(),
    });
    await load();
  };

  const handleTest = async () => {
    if (!editing?.expression) return;
    setTestLoading(true);
    setTestResult(null);
    try {
      let sample: Record<string, unknown>;
      try { sample = JSON.parse(testSample); }
      catch { setTestResult({ passed: false, output: '', error: 'Invalid JSON in sample record' }); return; }

      const res = await fetch(`/api/v1/bo/${boKey}/governance/validation-rules/test`, {
        method: 'POST',
        headers: headers(),
        body: JSON.stringify({ expression: editing.expression, sample }),
      });
      if (res.ok) {
        setTestResult(await res.json());
      }
    } finally {
      setTestLoading(false);
    }
  };

  const SEVERITY_COLORS = { ERROR: 'bog-pill-error', WARNING: 'bog-pill-warning', INFO: 'bog-pill-info' };

  return (
    <div>
      <div className="bog-section-header">
        <div>
          <div className="bog-section-title">🛡️ Validation Rules</div>
          <div className="bog-section-desc">
            CEL-expression rules evaluated against every record save. ERROR blocks, WARNING alerts, INFO logs.
          </div>
        </div>
        <button className="bog-btn bog-btn-primary" onClick={handleNew}>+ New Rule</button>
      </div>

      {/* ── Edit / Create Panel ── */}
      {editing && (
        <div className="bog-card" style={{ marginBottom: 20, borderColor: 'var(--bog-border-glow)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <span style={{ fontWeight: 700, fontSize: 15 }}>{isNew ? '+ New Validation Rule' : 'Edit Rule'}</span>
            <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={() => setEditing(null)}>✕ Cancel</button>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
            <div className="bog-input-group">
              <label className="bog-input-label">Rule Name</label>
              <input className="bog-input" value={editing.rule_name ?? ''} placeholder="e.g. ISIN format check"
                onChange={e => setEditing(prev => ({ ...prev!, rule_name: e.target.value }))} />
            </div>
            <div className="bog-input-group">
              <label className="bog-input-label">Field (optional)</label>
              <select className="bog-select" value={editing.field_key ?? ''}
                onChange={e => setEditing(prev => ({ ...prev!, field_key: e.target.value || undefined }))}>
                <option value="">— Cross-field rule —</option>
                {fields.map(f => <option key={f.key} value={f.key}>{f.display_name}</option>)}
              </select>
            </div>
          </div>

          <div className="bog-input-group">
            <label className="bog-input-label">Description</label>
            <input className="bog-input" value={editing.description ?? ''} placeholder="What does this rule check?"
              onChange={e => setEditing(prev => ({ ...prev!, description: e.target.value }))} />
          </div>

          <div className="bog-input-group">
            <label className="bog-input-label">CEL Expression</label>
            <textarea className="bog-textarea" style={{ minHeight: 100, fontFamily: "'JetBrains Mono', monospace", fontSize: 13 }}
              value={editing.expression ?? ''}
              placeholder={'// record is a map[string]any. Return true = PASS.\nrecord.isin.size() == 12'}
              onChange={e => setEditing(prev => ({ ...prev!, expression: e.target.value }))} />
          </div>

          <div className="bog-input-group">
            <label className="bog-input-label">Error Message</label>
            <input className="bog-input" value={editing.error_message ?? ''} placeholder="Shown to the user on violation"
              onChange={e => setEditing(prev => ({ ...prev!, error_message: e.target.value }))} />
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14, marginBottom: 14 }}>
            <div className="bog-input-group">
              <label className="bog-input-label">Severity</label>
              <div className="bog-severity-row">
                {(['ERROR', 'WARNING', 'INFO'] as const).map(s => (
                  <button key={s}
                    className={`bog-severity-btn ${s.toLowerCase()} ${editing.severity === s ? 'active' : ''}`}
                    onClick={() => setEditing(prev => ({ ...prev!, severity: s }))}>
                    {s}
                  </button>
                ))}
              </div>
            </div>
            <div className="bog-input-group">
              <label className="bog-input-label">Priority (lower = first)</label>
              <input type="number" className="bog-input" value={editing.priority ?? 100}
                onChange={e => setEditing(prev => ({ ...prev!, priority: parseInt(e.target.value, 10) }))} />
            </div>
          </div>

          {/* ── Live Test Harness ── */}
          <div className="bog-test-panel">
            <div style={{ fontWeight: 700, fontSize: 12, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--bog-text-muted)', marginBottom: 10 }}>
              ⚡ Live Test Harness
            </div>
            <div className="bog-input-group">
              <label className="bog-input-label">Sample Record (JSON)</label>
              <textarea className="bog-textarea" value={testSample} onChange={e => setTestSample(e.target.value)} />
            </div>
            <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={handleTest} disabled={testLoading}>
              {testLoading ? '⏳ Testing…' : '▶ Run Test'}
            </button>
            {testResult && (
              <div className={`bog-test-result ${testResult.error ? 'error' : testResult.passed ? 'pass' : 'fail'}`}>
                {testResult.error
                  ? `⚠ Error: ${testResult.error}`
                  : testResult.passed
                    ? '✓ PASS — Expression returned true'
                    : '✗ FAIL — Expression returned false (rule violated)'}
              </div>
            )}
          </div>

          <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
            <button className="bog-btn bog-btn-primary" onClick={handleSave} disabled={saving}>
              {saving ? '⏳ Saving…' : (isNew ? '✓ Create Rule' : '✓ Save Changes')}
            </button>
            <button className="bog-btn bog-btn-secondary" onClick={() => setEditing(null)}>Cancel</button>
          </div>
        </div>
      )}

      {/* ── Rules List ── */}
      {loading ? (
        <div className="bog-loading" style={{ padding: 40 }}>
          <div className="bog-spinner" /><span>Loading rules…</span>
        </div>
      ) : rules.length === 0 && !editing ? (
        <div className="bog-empty">
          <div className="bog-empty-icon">🛡️</div>
          <div className="bog-empty-title">No validation rules yet</div>
          <div className="bog-empty-desc">Add CEL-expression rules to enforce field formats, cross-field consistency, and business logic.</div>
          <button className="bog-btn bog-btn-primary" onClick={handleNew} style={{ marginTop: 8 }}>+ New Rule</button>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {rules.map(rule => (
            <div key={rule.rule_id} className="bog-card" style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4 }}>
                  <span style={{ fontWeight: 700, fontSize: 14 }}>{rule.rule_name}</span>
                  <span className={`bog-pill ${SEVERITY_COLORS[rule.severity]}`}>{rule.severity}</span>
                  {rule.is_core && <span className="bog-badge-core">CORE</span>}
                  {!rule.is_active && <span className="bog-pill" style={{ background: 'rgba(125,133,144,0.15)', color: 'var(--bog-text-muted)' }}>INACTIVE</span>}
                </div>
                <div style={{ fontSize: 12, color: 'var(--bog-text-muted)', fontFamily: 'monospace' }}>
                  {rule.field_key && <span style={{ marginRight: 8, color: 'var(--bog-accent)' }}>[{rule.field_key}]</span>}
                  {rule.expression}
                </div>
                {rule.description && (
                  <div style={{ fontSize: 12, color: 'var(--bog-text-muted)', marginTop: 4 }}>{rule.description}</div>
                )}
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                {!rule.is_core && (
                  <>
                    <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={() => handleEdit(rule)}>Edit</button>
                    <button className="bog-btn bog-btn-danger bog-btn-sm" onClick={() => handleDelete(rule.rule_id)}>Delete</button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default ValidationRuleBuilder;
