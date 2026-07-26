import React, { useState, useEffect, useCallback } from 'react';
import type { BOField } from './BOGovernanceStudio';
import './BOGovernanceStudio.css';

// ─── Types ────────────────────────────────────────────────────────────────────

type Classification = 'PUBLIC' | 'MASKED' | 'ENCRYPTED' | 'RESTRICTED';

interface FieldSecurityConfig {
  security_id?: string;
  bo_key: string;
  field_key: string;
  classification: Classification;
  mask_pattern?: string;
  visible_to_roles: string[];
  mask_for_roles: string[];
  deny_to_roles: string[];
  is_core: boolean;
}

interface MaskPreview {
  sanitised: Record<string, unknown>;
  mask_results: Array<{
    field_key: string;
    visibility: string;
    masked_value?: string;
  }>;
}

interface FieldSecurityConfiguratorProps {
  tenantId: string;
  boKey: string;
  fields: BOField[];
}

const CLASS_META: Record<Classification, { icon: string; label: string; desc: string; css: string }> = {
  PUBLIC:     { icon: '🔓', label: 'Public',     desc: 'All authorised users see full value', css: 'bog-class-public' },
  MASKED:     { icon: '🔒', label: 'Masked',     desc: 'Pattern-masked for most roles (e.g. ***-**-1234)', css: 'bog-class-masked' },
  ENCRYPTED:  { icon: '🔐', label: 'Encrypted',  desc: 'Only specific roles can decrypt', css: 'bog-class-encrypted' },
  RESTRICTED: { icon: '🚫', label: 'Restricted', desc: 'Hidden completely from response', css: 'bog-class-restricted' },
};

const CLASSIFICATIONS: Classification[] = ['PUBLIC', 'MASKED', 'ENCRYPTED', 'RESTRICTED'];

// ─── FieldSecurityConfigurator Component ──────────────────────────────────────

const FieldSecurityConfigurator: React.FC<FieldSecurityConfiguratorProps> = ({
  tenantId, boKey, fields,
}) => {
  const [configs, setConfigs] = useState<FieldSecurityConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<FieldSecurityConfig | null>(null);
  const [previewRoles, setPreviewRoles] = useState('ADMIN,ANALYST');
  const [previewRecord, setPreviewRecord] = useState('');
  const [previewResult, setPreviewResult] = useState<MaskPreview | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const headers = useCallback(() => ({
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
  }), [tenantId]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/bo/${boKey}/governance/field-security`, { headers: headers() });
      if (res.ok) setConfigs(await res.json() ?? []);
    } finally {
      setLoading(false);
    }
  }, [boKey, headers]);

  useEffect(() => {
    load();
    // Build default preview record from fields
    const sample: Record<string, string> = {};
    fields.forEach(f => { sample[f.key] = `sample_${f.key}`; });
    setPreviewRecord(JSON.stringify(sample, null, 2));
  }, [load, fields]);

  const configForField = (key: string): FieldSecurityConfig | undefined =>
    configs.find(c => c.field_key === key);

  const openEdit = (field: BOField) => {
    const existing = configForField(field.key);
    setEditing(existing ?? {
      bo_key: boKey,
      field_key: field.key,
      classification: 'PUBLIC',
      mask_pattern: undefined,
      visible_to_roles: ['ADMIN'],
      mask_for_roles: [],
      deny_to_roles: [],
      is_core: false,
    });
  };

  const handleSave = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      const res = await fetch(`/api/v1/bo/${boKey}/governance/field-security`, {
        method: 'POST', headers: headers(), body: JSON.stringify({ ...editing, bo_key: boKey }),
      });
      if (res.ok) { setEditing(null); await load(); }
    } finally {
      setSaving(false);
    }
  };

  const handlePreview = async () => {
    setPreviewLoading(true);
    setPreviewResult(null);
    try {
      let record: unknown;
      try { record = JSON.parse(previewRecord); }
      catch { return; }
      const roles = previewRoles.split(',').map(r => r.trim()).filter(Boolean);
      const res = await fetch(`/api/v1/bo/${boKey}/governance/field-security/preview`, {
        method: 'POST', headers: headers(),
        body: JSON.stringify({ record, roles }),
      });
      if (res.ok) setPreviewResult(await res.json());
    } finally {
      setPreviewLoading(false);
    }
  };

  const parseRoles = (raw: string) => raw.split(',').map(r => r.trim()).filter(Boolean);
  const formatRoles = (roles: string[]) => roles.join(', ');

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 380px', gap: 24 }}>
      {/* ── Left: Field List ── */}
      <div>
        <div className="bog-section-header">
          <div>
            <div className="bog-section-title">🔒 Field Security</div>
            <div className="bog-section-desc">
              Classify each field: Public · Masked · Encrypted · Restricted.
              Controls what each role can see in API responses.
            </div>
          </div>
        </div>

        {loading ? (
          <div className="bog-loading" style={{ padding: 40 }}><div className="bog-spinner" /></div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {fields.map(field => {
              const cfg = configForField(field.key);
              const cls: Classification = cfg?.classification ?? 'PUBLIC';
              const meta = CLASS_META[cls];
              return (
                <div key={field.key} className="bog-card"
                  style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '12px 16px' }}>
                  <span style={{ fontSize: 20 }}>{meta.icon}</span>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
                      <span style={{ fontWeight: 600, fontSize: 13 }}>{field.display_name}</span>
                      <span className={`bog-pill ${meta.css}`}>{meta.label}</span>
                      {cfg?.is_core && <span className="bog-badge-core">CORE</span>}
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--bog-text-muted)', fontFamily: 'monospace' }}>
                      {field.key}
                      {cfg?.mask_pattern && <span style={{ marginLeft: 8 }}>pattern: {cfg.mask_pattern}</span>}
                    </div>
                    {cfg && (
                      <div style={{ fontSize: 11, color: 'var(--bog-text-muted)', marginTop: 2 }}>
                        {cfg.visible_to_roles.length > 0 && <>👁 {cfg.visible_to_roles.join(', ')} &nbsp;</>}
                        {cfg.mask_for_roles.length > 0 && <>🔒 {cfg.mask_for_roles.join(', ')} &nbsp;</>}
                        {cfg.deny_to_roles.length > 0 && <>🚫 {cfg.deny_to_roles.join(', ')}</>}
                      </div>
                    )}
                  </div>
                  {!cfg?.is_core && (
                    <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={() => openEdit(field)}>
                      Configure
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* ── Right: Edit Panel + Preview ── */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* Edit Panel */}
        {editing ? (
          <div className="bog-card" style={{ borderColor: 'var(--bog-border-glow)' }}>
            <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 14 }}>
              Configure: <span style={{ color: 'var(--bog-accent)', fontFamily: 'monospace' }}>{editing.field_key}</span>
            </div>

            <div className="bog-input-group">
              <label className="bog-input-label">Classification</label>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 6 }}>
                {CLASSIFICATIONS.map(c => {
                  const m = CLASS_META[c];
                  return (
                    <button key={c}
                      onClick={() => setEditing(p => p ? { ...p, classification: c } : null)}
                      style={{
                        background: editing.classification === c ? `rgba(47,129,247,0.1)` : 'var(--bog-surface-2)',
                        border: `1px solid ${editing.classification === c ? 'var(--bog-accent)' : 'var(--bog-border)'}`,
                        borderRadius: 8, padding: '8px 10px', cursor: 'pointer', textAlign: 'left',
                        color: editing.classification === c ? 'var(--bog-text)' : 'var(--bog-text-muted)',
                      }}>
                      <div style={{ fontSize: 15 }}>{m.icon}</div>
                      <div style={{ fontSize: 11, fontWeight: 700, marginTop: 2 }}>{m.label}</div>
                    </button>
                  );
                })}
              </div>
            </div>

            {editing.classification === 'MASKED' && (
              <div className="bog-input-group">
                <label className="bog-input-label">Mask Pattern (# = shown, * = masked)</label>
                <input className="bog-input" placeholder="***-**-####" value={editing.mask_pattern ?? ''}
                  onChange={e => setEditing(p => p ? { ...p, mask_pattern: e.target.value } : null)} />
              </div>
            )}

            <div className="bog-input-group">
              <label className="bog-input-label">Visible to Roles (comma-separated)</label>
              <input className="bog-input" value={formatRoles(editing.visible_to_roles)}
                placeholder="ADMIN, COMPLIANCE_OFFICER"
                onChange={e => setEditing(p => p ? { ...p, visible_to_roles: parseRoles(e.target.value) } : null)} />
            </div>
            <div className="bog-input-group">
              <label className="bog-input-label">Masked for Roles</label>
              <input className="bog-input" value={formatRoles(editing.mask_for_roles)}
                placeholder="ANALYST, AUDITOR"
                onChange={e => setEditing(p => p ? { ...p, mask_for_roles: parseRoles(e.target.value) } : null)} />
            </div>
            <div className="bog-input-group">
              <label className="bog-input-label">Denied to Roles (hidden)</label>
              <input className="bog-input" value={formatRoles(editing.deny_to_roles)}
                placeholder="READ_ONLY"
                onChange={e => setEditing(p => p ? { ...p, deny_to_roles: parseRoles(e.target.value) } : null)} />
            </div>

            <div style={{ display: 'flex', gap: 8 }}>
              <button className="bog-btn bog-btn-primary" onClick={handleSave} disabled={saving}>
                {saving ? '⏳' : '✓ Save'}
              </button>
              <button className="bog-btn bog-btn-secondary" onClick={() => setEditing(null)}>Cancel</button>
            </div>
          </div>
        ) : (
          <div className="bog-empty" style={{ padding: 30 }}>
            <div className="bog-empty-icon" style={{ fontSize: 32 }}>🔒</div>
            <div className="bog-empty-desc">Click "Configure" on any field to set its security classification.</div>
          </div>
        )}

        {/* ── Masking Preview Panel ── */}
        <div className="bog-card">
          <div style={{ fontWeight: 700, fontSize: 13, marginBottom: 12 }}>👁 Masking Preview</div>
          <div className="bog-input-group">
            <label className="bog-input-label">Roles (comma-separated)</label>
            <input className="bog-input" value={previewRoles} onChange={e => setPreviewRoles(e.target.value)} />
          </div>
          <div className="bog-input-group">
            <label className="bog-input-label">Sample Record (JSON)</label>
            <textarea className="bog-textarea" style={{ minHeight: 100 }} value={previewRecord}
              onChange={e => setPreviewRecord(e.target.value)} />
          </div>
          <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={handlePreview} disabled={previewLoading}>
            {previewLoading ? '⏳' : '▶ Preview'}
          </button>
          {previewResult && (
            <div style={{ marginTop: 12 }}>
              <div style={{ fontSize: 11, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.5px', color: 'var(--bog-text-muted)', marginBottom: 8 }}>
                Output
              </div>
              {Object.entries(previewResult.sanitised).map(([k, v]) => {
                const mr = previewResult.mask_results.find(r => r.field_key === k);
                return (
                  <div key={k} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', borderBottom: '1px solid var(--bog-border)', fontSize: 12 }}>
                    <span style={{ color: 'var(--bog-text-muted)', fontFamily: 'monospace', flex: '0 0 140px', overflow: 'hidden', textOverflow: 'ellipsis' }}>{k}</span>
                    <span style={{ fontFamily: 'monospace', color: mr?.visibility === 'HIDDEN' ? 'var(--bog-text-muted)' : mr?.visibility === 'MASKED' ? 'var(--bog-amber)' : 'var(--bog-text)', flex: 1 }}>
                      {mr?.visibility === 'HIDDEN' ? '(hidden)' : String(v)}
                    </span>
                    {mr && <span className={`bog-pill ${mr.visibility === 'FULL' ? 'bog-pill-success' : mr.visibility === 'MASKED' ? 'bog-pill-warning' : 'bog-pill-error'}`} style={{ fontSize: 9 }}>{mr.visibility}</span>}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default FieldSecurityConfigurator;
