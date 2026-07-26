import React, { useState, useEffect, useCallback } from 'react';
import ValidationRuleBuilder from './ValidationRuleBuilder';
import PolicyRuleBuilder from './PolicyRuleBuilder';
import AccessControlMatrix from './AccessControlMatrix';
import FieldSecurityConfigurator from './FieldSecurityConfigurator';
import BOAuditTimeline from './BOAuditTimeline';
import './BOGovernanceStudio.css';

// ─── Types ──────────────────────────────────────────────────────────────────

export interface BOField {
  key: string;
  display_name: string;
  type: string;
  is_required: boolean;
  is_readonly: boolean;
}

export interface BusinessObjectSummary {
  id: string;
  key: string;
  name: string;
  display_name: string;
  description: string;
  icon: string;
  is_core: boolean;
  fields: BOField[];
}

type StudioTab = 'fields' | 'validation' | 'policies' | 'access' | 'security' | 'audit';

interface TabConfig {
  id: StudioTab;
  label: string;
  icon: string;
  badge?: number;
}

// ─── BOGovernanceStudio Component ───────────────────────────────────────────

interface BOGovernanceStudioProps {
  tenantId: string;
  boKey: string;
}

const TABS: TabConfig[] = [
  { id: 'fields',     label: 'Fields',         icon: '⚡' },
  { id: 'validation', label: 'Validation',      icon: '🛡️' },
  { id: 'policies',   label: 'Policies',        icon: '⚖️' },
  { id: 'access',     label: 'Access Control',  icon: '🔐' },
  { id: 'security',   label: 'Field Security',  icon: '🔒' },
  { id: 'audit',      label: 'Audit Log',       icon: '📋' },
];

const BOGovernanceStudio: React.FC<BOGovernanceStudioProps> = ({ tenantId, boKey }) => {
  const [activeTab, setActiveTab] = useState<StudioTab>('fields');
  const [bo, setBo] = useState<BusinessObjectSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [validationCount, setValidationCount] = useState(0);
  const [policyCount, setPolicyCount] = useState(0);

  const headers = useCallback(() => ({
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
  }), [tenantId]);

  // Load BO summary + badge counts
  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const [boRes, rulesRes, policiesRes] = await Promise.all([
          fetch(`/api/v1/bo/${boKey}`, { headers: headers() }),
          fetch(`/api/v1/bo/${boKey}/governance/validation-rules`, { headers: headers() }),
          fetch(`/api/v1/bo/${boKey}/governance/policies`, { headers: headers() }),
        ]);
        if (boRes.ok) setBo(await boRes.json());
        if (rulesRes.ok) {
          const rules = await rulesRes.json();
          setValidationCount(Array.isArray(rules) ? rules.length : 0);
        }
        if (policiesRes.ok) {
          const pols = await policiesRes.json();
          setPolicyCount(Array.isArray(pols) ? pols.length : 0);
        }
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [boKey, headers]);

  const tabs: TabConfig[] = TABS.map(t => {
    if (t.id === 'validation' && validationCount > 0) return { ...t, badge: validationCount };
    if (t.id === 'policies' && policyCount > 0) return { ...t, badge: policyCount };
    return t;
  });

  if (loading) {
    return (
      <div className="bog-loading">
        <div className="bog-spinner" />
        <span>Loading BO Governance Studio…</span>
      </div>
    );
  }

  return (
    <div className="bog-root">
      {/* ── Header ── */}
      <div className="bog-header">
        <div className="bog-header-left">
          <span className="bog-bo-icon">{bo?.icon || '📦'}</span>
          <div className="bog-header-text">
            <h1 className="bog-title">{bo?.display_name || boKey}</h1>
            <p className="bog-subtitle">
              {bo?.is_core ? <span className="bog-badge-core">CORE</span> : <span className="bog-badge-custom">CUSTOM</span>}
              &nbsp;·&nbsp;{bo?.description || 'Business Object Governance Studio'}
            </p>
          </div>
        </div>
        <div className="bog-header-right">
          <span className="bog-header-field-count">{bo?.fields?.length ?? 0} fields</span>
        </div>
      </div>

      {/* ── Tab Bar ── */}
      <div className="bog-tabbar">
        {tabs.map(tab => (
          <button
            key={tab.id}
            className={`bog-tab ${activeTab === tab.id ? 'bog-tab--active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            <span className="bog-tab-icon">{tab.icon}</span>
            <span className="bog-tab-label">{tab.label}</span>
            {tab.badge !== undefined && (
              <span className="bog-tab-badge">{tab.badge}</span>
            )}
          </button>
        ))}
      </div>

      {/* ── Content Panel ── */}
      <div className="bog-content">
        {activeTab === 'fields' && bo && (
          <FieldsPanel bo={bo} />
        )}
        {activeTab === 'validation' && (
          <ValidationRuleBuilder
            tenantId={tenantId}
            boKey={boKey}
            fields={bo?.fields ?? []}
            onCountChange={setValidationCount}
          />
        )}
        {activeTab === 'policies' && (
          <PolicyRuleBuilder
            tenantId={tenantId}
            boKey={boKey}
            fields={bo?.fields ?? []}
            onCountChange={setPolicyCount}
          />
        )}
        {activeTab === 'access' && (
          <AccessControlMatrix tenantId={tenantId} boKey={boKey} />
        )}
        {activeTab === 'security' && (
          <FieldSecurityConfigurator
            tenantId={tenantId}
            boKey={boKey}
            fields={bo?.fields ?? []}
          />
        )}
        {activeTab === 'audit' && (
          <BOAuditTimeline tenantId={tenantId} boKey={boKey} />
        )}
      </div>
    </div>
  );
};

// ─── Fields Panel ────────────────────────────────────────────────────────────

const FIELD_TYPE_COLORS: Record<string, string> = {
  string: '#6c8ebf', text: '#6c8ebf', number: '#82b366', decimal: '#82b366',
  currency: '#d6b656', percentage: '#d6b656', date: '#9673a6', datetime: '#9673a6',
  boolean: '#ae4132', reference: '#23aae1', picklist: '#23aae1', json: '#7a7a7a',
  uuid: '#888', email: '#6c8ebf',
};

interface FieldsPanelProps {
  bo: BusinessObjectSummary;
}

const FieldsPanel: React.FC<FieldsPanelProps> = ({ bo }) => {
  const [search, setSearch] = useState('');

  const filtered = (bo.fields ?? []).filter(f =>
    f.display_name.toLowerCase().includes(search.toLowerCase()) ||
    f.key.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="bog-fields-panel">
      <div className="bog-fields-toolbar">
        <input
          className="bog-search-input"
          placeholder="Search fields…"
          value={search}
          onChange={e => setSearch(e.target.value)}
        />
        <span className="bog-field-count-badge">{filtered.length} fields</span>
      </div>
      <div className="bog-fields-grid">
        {filtered.map(field => (
          <div key={field.key} className="bog-field-card">
            <div className="bog-field-card-top">
              <span
                className="bog-field-type-pill"
                style={{ background: FIELD_TYPE_COLORS[field.type] ?? '#555' }}
              >
                {field.type}
              </span>
              {field.is_required && <span className="bog-req-badge">REQUIRED</span>}
              {field.is_readonly && <span className="bog-ro-badge">READ-ONLY</span>}
            </div>
            <div className="bog-field-name">{field.display_name}</div>
            <div className="bog-field-key">{field.key}</div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default BOGovernanceStudio;
