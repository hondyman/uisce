import React, { useState, useEffect, useCallback } from 'react';
import './BOGovernanceStudio.css';

// ─── Types ────────────────────────────────────────────────────────────────────

type Operation = 'READ' | 'WRITE' | 'DELETE' | 'EXECUTE' | 'ADMIN';

interface AccessPolicy {
  access_id?: string;
  bo_key: string;
  role_key: string;
  operation: Operation;
  is_allowed: boolean;
  condition_expr?: string;
  row_filter_expr?: string;
  is_core: boolean;
}

interface AccessControlMatrixProps {
  tenantId: string;
  boKey: string;
}

const OPERATIONS: Operation[] = ['READ', 'WRITE', 'DELETE', 'EXECUTE', 'ADMIN'];

// Built-in common roles (can be extended/loaded from backend)
const DEFAULT_ROLES = [
  'ADMIN',
  'COMPLIANCE_OFFICER',
  'PORTFOLIO_MANAGER',
  'ANALYST',
  'AUDITOR',
  'DATA_STEWARD',
  'READ_ONLY',
];

// ─── AccessControlMatrix Component ───────────────────────────────────────────

const AccessControlMatrix: React.FC<AccessControlMatrixProps> = ({ tenantId, boKey }) => {
  const [policies, setPolicies] = useState<AccessPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [roles, setRoles] = useState<string[]>(DEFAULT_ROLES);
  const [newRole, setNewRole] = useState('');
  const [selected, setSelected] = useState<AccessPolicy | null>(null);
  const [saving, setSaving] = useState<string>('');

  const headers = useCallback(() => ({
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
  }), [tenantId]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/bo/${boKey}/governance/access`, { headers: headers() });
      if (res.ok) {
        const data: AccessPolicy[] = await res.json();
        setPolicies(Array.isArray(data) ? data : []);
        // Discover any additional roles from existing policies
        const discovered = Array.from(new Set(data.map(p => p.role_key)));
        setRoles(prev => Array.from(new Set([...prev, ...discovered])));
      }
    } finally {
      setLoading(false);
    }
  }, [boKey, headers]);

  useEffect(() => { load(); }, [load]);

  // Build an index: role_key + operation → AccessPolicy
  const policyIndex = useCallback((role: string, op: Operation): AccessPolicy | undefined => {
    return policies.find(p => p.role_key === role && p.operation === op);
  }, [policies]);

  const toggle = async (role: string, op: Operation) => {
    const existing = policyIndex(role, op);
    const key = `${role}:${op}`;
    setSaving(key);
    try {
      const payload: AccessPolicy = {
        access_id: existing?.access_id,
        bo_key: boKey,
        role_key: role,
        operation: op,
        is_allowed: existing ? !existing.is_allowed : true,
        condition_expr: existing?.condition_expr,
        row_filter_expr: existing?.row_filter_expr,
        is_core: false,
      };
      const res = await fetch(`/api/v1/bo/${boKey}/governance/access`, {
        method: 'POST', headers: headers(), body: JSON.stringify(payload),
      });
      if (res.ok) await load();
    } finally {
      setSaving('');
    }
  };

  const addRole = () => {
    const r = newRole.trim().toUpperCase().replace(/\s+/g, '_');
    if (r && !roles.includes(r)) {
      setRoles(prev => [...prev, r]);
    }
    setNewRole('');
  };

  const CellContent: React.FC<{ role: string; op: Operation }> = ({ role, op }) => {
    const p = policyIndex(role, op);
    const key = `${role}:${op}`;
    const isLoading = saving === key;

    if (isLoading) return <span style={{ color: 'var(--bog-text-muted)', fontSize: 11 }}>…</span>;
    if (!p) {
      return (
        <button onClick={() => toggle(role, op)} className="bog-matrix-cell-denied"
          title="Click to grant access" style={{ cursor: 'pointer', background: 'transparent', border: '1px dashed rgba(248,81,73,0.3)', color: 'var(--bog-text-muted)' }}>
          —
        </button>
      );
    }
    if (p.is_allowed && !p.condition_expr) {
      return (
        <button onClick={() => toggle(role, op)} className="bog-matrix-cell-allowed"
          title={p.is_core ? 'Core policy (read-only)' : 'Click to revoke'}
          disabled={p.is_core}>
          ✓ ALLOW
        </button>
      );
    }
    if (p.is_allowed && p.condition_expr) {
      return (
        <button onClick={() => setSelected(p)} className="bog-matrix-cell-conditional"
          title="Conditional — click to edit">
          ⚡ COND
        </button>
      );
    }
    return (
      <button onClick={() => toggle(role, op)} className="bog-matrix-cell-denied"
        title={p.is_core ? 'Core policy (read-only)' : 'Click to allow'}
        disabled={p.is_core}>
        ✗ DENY
      </button>
    );
  };

  return (
    <div>
      <div className="bog-section-header">
        <div>
          <div className="bog-section-title">🔐 Access Control Matrix</div>
          <div className="bog-section-desc">
            RBAC + ABAC: click cells to toggle READ/WRITE/DELETE/EXECUTE/ADMIN per role.
            Amber = conditional (ABAC expression required).
          </div>
        </div>
      </div>

      {/* ── Add Role ── */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 20 }}>
        <input className="bog-input" style={{ maxWidth: 240 }} placeholder="Add role (e.g. RISK_MANAGER)"
          value={newRole} onChange={e => setNewRole(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && addRole()} />
        <button className="bog-btn bog-btn-secondary" onClick={addRole}>+ Add Role</button>
      </div>

      {/* ── Matrix Table ── */}
      {loading ? (
        <div className="bog-loading" style={{ padding: 40 }}><div className="bog-spinner" /></div>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table className="bog-matrix-table">
            <thead>
              <tr>
                <th style={{ textAlign: 'left', width: 200 }}>Role</th>
                {OPERATIONS.map(op => (
                  <th key={op}>{op}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {roles.map(role => (
                <tr key={role}>
                  <td style={{ fontFamily: 'monospace', fontSize: 12, color: 'var(--bog-text)' }}>
                    {role}
                  </td>
                  {OPERATIONS.map(op => (
                    <td key={op}>
                      <CellContent role={role} op={op} />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* ── Row-Level Filter Editor ── */}
      {selected && (
        <div className="bog-card" style={{ marginTop: 20, borderColor: 'var(--bog-border-glow)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
            <span style={{ fontWeight: 700 }}>
              Edit Conditional Policy — {selected.role_key} / {selected.operation}
            </span>
            <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={() => setSelected(null)}>✕</button>
          </div>
          <div className="bog-input-group">
            <label className="bog-input-label">ABAC Condition Expression (CEL)</label>
            <textarea className="bog-textarea"
              value={selected.condition_expr ?? ''}
              placeholder={'principal.department == record.owner_department'}
              onChange={e => setSelected(p => p ? { ...p, condition_expr: e.target.value } : null)} />
          </div>
          <div className="bog-input-group">
            <label className="bog-input-label">Row-Level Filter SQL Fragment</label>
            <textarea className="bog-textarea"
              value={selected.row_filter_expr ?? ''}
              placeholder={'tenant_id = :tenant_id AND owner_id = :principal_id'}
              onChange={e => setSelected(p => p ? { ...p, row_filter_expr: e.target.value } : null)} />
          </div>
          <div style={{ display: 'flex', gap: 10 }}>
            <button className="bog-btn bog-btn-primary" onClick={async () => {
              if (!selected) return;
              await fetch(`/api/v1/bo/${boKey}/governance/access`, {
                method: 'POST', headers: headers(), body: JSON.stringify(selected),
              });
              setSelected(null);
              await load();
            }}>✓ Save</button>
            <button className="bog-btn bog-btn-secondary" onClick={() => setSelected(null)}>Cancel</button>
          </div>
        </div>
      )}

      {/* ── Legend ── */}
      <div style={{ display: 'flex', gap: 16, marginTop: 24, padding: 14, background: 'var(--bog-surface)', borderRadius: 8, flexWrap: 'wrap' }}>
        <span style={{ fontSize: 12, color: 'var(--bog-text-muted)', fontWeight: 700 }}>Legend:</span>
        <span className="bog-matrix-cell-allowed" style={{ fontSize: 11 }}>✓ ALLOW</span>
        <span className="bog-matrix-cell-denied" style={{ fontSize: 11 }}>✗ DENY</span>
        <span className="bog-matrix-cell-conditional" style={{ fontSize: 11 }}>⚡ COND</span>
        <span style={{ fontSize: 12, color: 'var(--bog-text-muted)' }}>— = No policy (implicit deny) · Click any cell to toggle</span>
      </div>
    </div>
  );
};

export default AccessControlMatrix;
