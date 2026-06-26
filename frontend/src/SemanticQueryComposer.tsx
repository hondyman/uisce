import { useState, useMemo } from 'react';
import type { SemanticViewMeta, ViewMeta, SemanticQuery, SemanticMember, SemanticModelClaim } from './types';
import { getViewIdentifier } from './types/views';

// A simple Lock icon component
const LockIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="lock-icon">
    <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
    <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
  </svg>
);

function AccessDeniedOverlay({ assetType, onRequestAccess }: { assetType: string, onRequestAccess: () => void }) {
  return (
    <div className="access-denied-overlay">
      <LockIcon />
      <h4>Access Denied</h4>
      <p>You don’t have permission to view these <strong>{assetType}</strong>.</p>
      <button onClick={onRequestAccess}>Request Access</button>
    </div>
  );
}

function MemberList({ title, members, selected, onToggle, disabled = false, onRequestAccess }: { title: string, members: SemanticMember[], selected: string[], onToggle: (name: string) => void, disabled?: boolean, onRequestAccess: () => void }) {
  if (disabled) {
    return (
      <div className="member-list disabled">
        <AccessDeniedOverlay assetType={title} onRequestAccess={onRequestAccess} />
      </div>
    );
  }

  return (
    <div className="member-list">
      <h5>{title}</h5>
      {members.map(m => (
  <label key={m.name} title={m.description || m.label}>
          <input
            type="checkbox"
            checked={selected.includes(m.name)}
            onChange={() => onToggle(m.name)}
          />
          {m.label}
        </label>
      ))}
    </div>
  );
}

interface SemanticQueryComposerProps {
  view: SemanticViewMeta | ViewMeta;
  claims: SemanticModelClaim[];
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  onExecute: (_query: SemanticQuery) => void;
  onRequestAccess: (assetType: 'dimensions' | 'metrics') => void;
}

export default function SemanticQueryComposer({ view, claims, onExecute, onRequestAccess }: SemanticQueryComposerProps) {
  const [query, setQuery] = useState<SemanticQuery>({
    dimensions: [],
    metrics: [],
    filters: [],
    order: [],
    limit: 100,
  });

  const { visibleDimensions, visibleMetrics, isRestricted, restrictionType } = useMemo(() => {
    // Normalize members from either SemanticViewMeta or ViewMeta
    const allDimensions: SemanticMember[] = (view as SemanticViewMeta).dimensions || (view as ViewMeta).dimensions || [];
    const allMetrics: SemanticMember[] = (view as SemanticViewMeta).metrics || (view as ViewMeta).measures || [];
    const readClaim = claims.find(c => c.permission === 'read');

    // Default: if no claim or no scope on the claim, user has full access to this view.
    if (!readClaim || !readClaim.scope || readClaim.scope.length === 0) {
  return { visibleDimensions: allDimensions, visibleMetrics: allMetrics, isRestricted: false, restrictionType: null };
    }

    const scope = readClaim.scope;
    const allowedDimensions = new Set<string>();
    const allowedMetrics = new Set<string>();

    // Check for broad, category-level access first.
    const hasFullDimensionAccess = scope.includes('dimensions');
    const hasFullMetricAccess = scope.includes('metrics');

    if (hasFullDimensionAccess && allDimensions) {
      allDimensions.forEach(d => allowedDimensions.add(d.name));
    }
    if (hasFullMetricAccess && allMetrics) {
      allMetrics.forEach(m => allowedMetrics.add(m.name));
    }

    // Then, add specific, object-level access.
    scope.forEach(item => {
      const [type, name] = item.split(':');
      if (name) { // Ensure it's an object-level scope item
        if (type === 'dimension') allowedDimensions.add(name);
        if (type === 'metric') allowedMetrics.add(name);
      }
    });

  const finalVisibleDimensions = (allDimensions || []).filter(d => allowedDimensions.has(d.name));
  const finalVisibleMetrics = (allMetrics || []).filter(m => allowedMetrics.has(m.name));

  const restricted = finalVisibleDimensions.length < (allDimensions || []).length || finalVisibleMetrics.length < (allMetrics || []).length;
  const restrictionMessage = restricted ? `You have access to ${finalVisibleDimensions.length} of ${(allDimensions || []).length} dimensions and ${finalVisibleMetrics.length} of ${(allMetrics || []).length} metrics.` : null;

    return {
  visibleDimensions: finalVisibleDimensions,
  visibleMetrics: finalVisibleMetrics,
      isRestricted: restricted,
      restrictionType: restrictionMessage,
    };
  }, [view, claims]);

  const toggleMember = (type: 'dimensions' | 'metrics', name: string) => {
    setQuery(q => {
      const current = q[type];
      const updated = current.includes(name) ? current.filter(m => m !== name) : [...current, name];
      return { ...q, [type]: updated };
    });
  };

  // replaced unsafe casts with helper below
  const getViewDisplayTitle = (v: SemanticViewMeta | ViewMeta): string => {
    const r = v as unknown as Record<string, unknown>;
    if (typeof r.title === 'string' && r.title.length) return String(r.title);
    if (typeof r.name === 'string' && r.name.length) return String(r.name);
    const id = getViewIdentifier(v);
    return id ? String(id).slice(0, 8) : 'Unnamed View';
  };
  const displayTitle = getViewDisplayTitle(view);

  return (
    <div className="semantic-composer">
      <div className="composer-header">
        <h3>{displayTitle}</h3>
        {isRestricted && (
          <span className="restricted-badge" title={restrictionType || undefined}>
            🔒 Restricted View
          </span>
        )}
      </div>
      <div className="composer-body">
        <MemberList
          title="Dimensions"
          members={visibleDimensions}
          selected={query.dimensions}
          onToggle={(name) => toggleMember('dimensions', name)}
          disabled={visibleDimensions.length === 0 && Array.isArray(view.dimensions) && view.dimensions.length > 0}
          onRequestAccess={() => onRequestAccess('dimensions')}
        />
        <MemberList
          title="Metrics"
          members={visibleMetrics}
          selected={query.metrics}
          onToggle={(name) => toggleMember('metrics', name)}
          disabled={visibleMetrics.length === 0 && Array.isArray((view as SemanticViewMeta).metrics || (view as ViewMeta).measures) && ((view as SemanticViewMeta).metrics || (view as ViewMeta).measures || []).length > 0}
          onRequestAccess={() => onRequestAccess('metrics')}
        />
      </div>
      <button onClick={() => onExecute(query)} disabled={query.metrics.length === 0}>Run Query</button>
    </div>
  );
}