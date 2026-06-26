import { FC } from 'react';
import * as TablerIcons from '@tabler/icons-react';
import { type ExtensionCompatibilityRow } from '../../services/extensions';

interface CompatibilityPanelProps {
  refreshCompatibility: () => Promise<void> | void;
  compatLoading: boolean;
  issueLevelFilter: 'all' | 'error' | 'warning';
  setIssueLevelFilter: (v: 'all' | 'error' | 'warning') => void;
  issueCodeFilter: string;
  setIssueCodeFilter: (v: string) => void;
  compatErr: string | null;
  filteredCompat: ExtensionCompatibilityRow[] | null;
  expandIssues: Record<string, boolean>;
  setExpandIssues: React.Dispatch<React.SetStateAction<Record<string, boolean>>>;
  expandChanges: Record<string, boolean>;
  setExpandChanges: React.Dispatch<React.SetStateAction<Record<string, boolean>>>;
}

const CompatibilityPanel: FC<CompatibilityPanelProps> = ({
  refreshCompatibility,
  compatLoading,
  issueLevelFilter,
  setIssueLevelFilter,
  issueCodeFilter,
  setIssueCodeFilter,
  compatErr,
  filteredCompat,
  expandIssues,
  setExpandIssues,
  expandChanges,
  setExpandChanges,
}) => {
  const countExtensionChanges = (changes: Record<string, any> | null | undefined): number => {
    if (!changes) return 0;
    let count = 0;
    const listKeys = ['dimensions_added', 'measures_added', 'joins_added', 'filters_added'];
    listKeys.forEach((key) => { if (Array.isArray((changes as any)[key])) count += (changes as any)[key].length; });
    const mapKeys = ['dimensions_overridden', 'measures_overridden', 'joins_overridden', 'filters_overridden'];
    mapKeys.forEach((key) => { if ((changes as any)[key] && typeof (changes as any)[key] === 'object') count += Object.keys((changes as any)[key]).length; });
    return count;
  };

  const renderExtensionChanges = (changes: Record<string, any>) => {
    const sections: Array<{ key: string; label: string; items?: string[]; map?: Record<string, string[]> }> = [];
    const listKeys = [
      { key: 'cube_fields', label: 'Cube Fields' },
      { key: 'dimensions_added', label: 'Dimensions Added' },
      { key: 'measures_added', label: 'Measures Added' },
      { key: 'joins_added', label: 'Joins Added' },
      { key: 'filters_added', label: 'Filters Added' },
    ];
    listKeys.forEach(({ key, label }) => {
      const v = (changes as any)[key];
      if (Array.isArray(v) && v.length > 0) sections.push({ key, label, items: v as string[] });
    });
    const mapKeys = [
      { key: 'dimensions_overridden', label: 'Dimensions Overridden' },
      { key: 'measures_overridden', label: 'Measures Overridden' },
      { key: 'joins_overridden', label: 'Joins Overridden' },
      { key: 'filters_overridden', label: 'Filters Overridden' },
    ];
    mapKeys.forEach(({ key, label }) => {
      const v = (changes as any)[key];
      if (v && typeof v === 'object') sections.push({ key, label, map: v as Record<string, string[]> });
    });

    if (sections.length === 0) {
      return <div className="changes-empty">No extension changes.</div>;
    }

    return (
      <>
        {sections.map((s) => (
          <div className="change-section" key={s.key}>
            <div className="change-title">{s.label}</div>
            {s.items && (
              <ul className="change-list">
                {s.items.map((it) => (
                  <li key={it}>
                    <TablerIcons.IconPlus size={14} /> {it}
                  </li>
                ))}
              </ul>
            )}
            {s.map && (
              <ul className="change-map">
                {Object.entries(s.map).map(([name, keys]) => (
                  <li key={name}>
                    <TablerIcons.IconSettings size={14} /> <strong>{name}</strong>
                    {Array.isArray(keys) && keys.length > 0 && (
                      <span className="change-keys"> — {keys.join(', ')}</span>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </div>
        ))}
      </>
    );
  };

  return (
    <section className="compat-panel">
      <div className="compat-header">
        <h4>Extension Compatibility</h4>
        <button className="btn btn-sm btn-outline" onClick={refreshCompatibility} disabled={compatLoading}>
          {compatLoading ? 'Refreshing…' : 'Refresh'}
        </button>
      </div>
      <div className="compat-filters">
        <label>
          Level:
          <select value={issueLevelFilter} onChange={(e) => setIssueLevelFilter(e.target.value as any)}>
            <option value="all">All</option>
            <option value="error">Error</option>
            <option value="warning">Warning</option>
          </select>
        </label>
        <label>
          Code:
          <input
            type="text"
            placeholder="Filter by code (e.g., TENANT_GUARD_MISSING)"
            value={issueCodeFilter}
            onChange={(e) => setIssueCodeFilter(e.target.value)}
          />
        </label>
      </div>
      {compatErr && <div className="error-text">{compatErr}</div>}
      {filteredCompat && filteredCompat.length > 0 ? (
        <div className="compat-list">
          {filteredCompat.map((row) => (
            <div key={row.extension_model_key} className={`compat-item ${row.version_mismatch ? 'warn' : ''}`}>
              <div className="compat-row-top">
                <span className="mono">{row.extension_model_key}</span>
                <span className="status">{row.status}</span>
              </div>
              <div className="compat-row-sub">
                <span>Base: {row.base_cube_name} v{row.base_version}</span>
                {row.extension_core_version_target != null && (
                  <span>Target: v{row.extension_core_version_target}</span>
                )}
                {row.version_mismatch && <span className="badge badge-warn">Version Mismatch</span>}
              </div>
              <div className="compat-sections">
                <div className="compat-section">
                  <button
                    className="btn btn-xs btn-outline"
                    onClick={() => setExpandIssues((s) => ({ ...s, [row.extension_model_key]: !s[row.extension_model_key] }))}
                  >
                    {expandIssues[row.extension_model_key] ? <TablerIcons.IconChevronDown size={14} /> : <TablerIcons.IconChevronRight size={14} />}
                    Issues ({row.issues?.length || 0})
                  </button>
                  {expandIssues[row.extension_model_key] && row.issues?.length > 0 && (
                    <ul className="issues">
                      {row.issues.map((is, idx) => (
                        <li key={idx} className={`issue ${is.level}`}>
                          <TablerIcons.IconAlertTriangle size={14} />
                          <strong>[{is.level}] {is.code}</strong> — {is.message}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>

                {row.extension_changes && (
                  <div className="compat-section">
                    <button
                      className="btn btn-xs btn-outline"
                      onClick={() => setExpandChanges((s) => ({ ...s, [row.extension_model_key]: !s[row.extension_model_key] }))}
                    >
                      {expandChanges[row.extension_model_key] ? <TablerIcons.IconChevronDown size={14} /> : <TablerIcons.IconChevronRight size={14} />}
                      Extension Changes ({countExtensionChanges(row.extension_changes)})
                    </button>
                    {expandChanges[row.extension_model_key] && (
                      <div className="changes-grid">
                        {renderExtensionChanges(row.extension_changes || {})}
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="compat-empty">No extensions found.</div>
      )}
    </section>
  );
};

export default CompatibilityPanel;
