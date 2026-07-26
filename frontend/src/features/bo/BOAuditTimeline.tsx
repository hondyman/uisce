import React, { useState, useEffect, useCallback, useRef } from 'react';
import './BOGovernanceStudio.css';

// ─── Types ────────────────────────────────────────────────────────────────────

interface AuditEvent {
  event_id: string;
  bo_key: string;
  entity_id: string;
  operation: string;
  actor_id: string;
  actor_role: string;
  before_value?: Record<string, unknown>;
  after_value?: Record<string, unknown>;
  policy_triggered?: string;
  ip_address?: string;
  created_at: string;
}

interface BOAuditTimelineProps {
  tenantId: string;
  boKey: string;
}

const OP_META: Record<string, { color: string; cls: string }> = {
  CREATE:              { color: 'var(--bog-green)',  cls: 'op-create' },
  UPDATE:              { color: 'var(--bog-accent)', cls: 'op-update' },
  DELETE:              { color: 'var(--bog-red)',    cls: 'op-delete' },
  POLICY_TRIGGERED:    { color: 'var(--bog-amber)',  cls: 'op-policy' },
  ACCESS_DENIED:       { color: 'var(--bog-purple)', cls: 'op-access' },
  CREATE_VALIDATION_RULE: { color: 'var(--bog-accent)', cls: 'op-update' },
};

function formatTime(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString(undefined, { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
  } catch { return iso; }
}

function formatDate(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleDateString(undefined, { month: 'short', day: '2-digit' });
  } catch { return ''; }
}

// Simple JSON diff viewer — just shows before/after keys that changed
function DiffViewer({ before, after }: { before?: Record<string, unknown>; after?: Record<string, unknown> }) {
  if (!before && !after) return <div style={{ color: 'var(--bog-text-muted)', fontSize: 12 }}>No diff data</div>;
  const keys = Array.from(new Set([...Object.keys(before ?? {}), ...Object.keys(after ?? {})]));
  const changed = keys.filter(k => JSON.stringify((before ?? {})[k]) !== JSON.stringify((after ?? {})[k]));

  if (changed.length === 0) return <div style={{ color: 'var(--bog-text-muted)', fontSize: 12 }}>No field changes detected</div>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 12, fontFamily: 'monospace' }}>
      {changed.map(k => (
        <div key={k} style={{ display: 'grid', gridTemplateColumns: '120px 1fr 1fr', gap: 8, alignItems: 'start' }}>
          <span style={{ color: 'var(--bog-text-muted)' }}>{k}</span>
          <span style={{ color: 'var(--bog-red)', textDecoration: 'line-through', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {JSON.stringify((before ?? {})[k])}
          </span>
          <span style={{ color: 'var(--bog-green)', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {JSON.stringify((after ?? {})[k])}
          </span>
        </div>
      ))}
    </div>
  );
}

// ─── BOAuditTimeline Component ───────────────────────────────────────────────

const BOAuditTimeline: React.FC<BOAuditTimelineProps> = ({ tenantId, boKey }) => {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [filterOp, setFilterOp] = useState<string>('ALL');
  const [liveMode, setLiveMode] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval>>(null);

  const headers = useCallback(() => ({
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
  }), [tenantId]);

  const load = useCallback(async () => {
    try {
      const res = await fetch(`/api/v1/bo/${boKey}/governance/audit`, { headers: headers() });
      if (res.ok) {
        const data = await res.json();
        setEvents(Array.isArray(data) ? data : []);
      }
    } finally {
      setLoading(false);
    }
  }, [boKey, headers]);

  useEffect(() => { load(); }, [load]);

  // Live mode: poll every 5s
  useEffect(() => {
    if (liveMode) {
      intervalRef.current = setInterval(load, 5000);
    } else {
      if (intervalRef.current) clearInterval(intervalRef.current);
    }
    return () => { if (intervalRef.current) clearInterval(intervalRef.current); };
  }, [liveMode, load]);

  const toggleExpand = (id: string) => {
    setExpanded(prev => {
      const n = new Set(prev);
      n.has(id) ? n.delete(id) : n.add(id);
      return n;
    });
  };

  const allOps = Array.from(new Set(events.map(e => e.operation)));
  const filtered = filterOp === 'ALL' ? events : events.filter(e => e.operation === filterOp);

  return (
    <div>
      <div className="bog-section-header">
        <div>
          <div className="bog-section-title">📋 Audit Log</div>
          <div className="bog-section-desc">
            Immutable chronological record of all BO operations, policy triggers, and access denials.
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <button className="bog-btn bog-btn-secondary bog-btn-sm" onClick={load}>↻ Refresh</button>
          <button
            className="bog-btn bog-btn-sm"
            onClick={() => setLiveMode(l => !l)}
            style={{
              background: liveMode ? 'rgba(63,185,80,0.15)' : 'var(--bog-surface)',
              border: `1px solid ${liveMode ? 'var(--bog-green)' : 'var(--bog-border)'}`,
              color: liveMode ? 'var(--bog-green)' : 'var(--bog-text-muted)',
            }}>
            {liveMode ? '● LIVE' : '○ LIVE'}
          </button>
        </div>
      </div>

      {/* ── Filters ── */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 16, flexWrap: 'wrap' }}>
        <button
          className="bog-btn bog-btn-sm"
          onClick={() => setFilterOp('ALL')}
          style={{ background: filterOp === 'ALL' ? 'rgba(47,129,247,0.15)' : 'var(--bog-surface)', border: `1px solid ${filterOp === 'ALL' ? 'var(--bog-accent)' : 'var(--bog-border)'}`, color: filterOp === 'ALL' ? 'var(--bog-accent)' : 'var(--bog-text-muted)' }}>
          All
        </button>
        {allOps.map(op => {
          const m = OP_META[op] ?? { color: 'var(--bog-text-muted)', cls: '' };
          return (
            <button key={op} className="bog-btn bog-btn-sm"
              onClick={() => setFilterOp(op)}
              style={{ background: filterOp === op ? `${m.color}22` : 'var(--bog-surface)', border: `1px solid ${filterOp === op ? m.color : 'var(--bog-border)'}`, color: filterOp === op ? m.color : 'var(--bog-text-muted)' }}>
              {op}
            </button>
          );
        })}
      </div>

      {/* ── Event List ── */}
      {loading ? (
        <div className="bog-loading" style={{ padding: 40 }}><div className="bog-spinner" /></div>
      ) : filtered.length === 0 ? (
        <div className="bog-empty">
          <div className="bog-empty-icon">📋</div>
          <div className="bog-empty-title">No audit events yet</div>
          <div className="bog-empty-desc">All BO operations, policy triggers, and access denials will appear here.</div>
        </div>
      ) : (
        <div className="bog-audit-list">
          {filtered.map(evt => {
            const m = OP_META[evt.operation] ?? { color: 'var(--bog-text-muted)', cls: '' };
            const isExpanded = expanded.has(evt.event_id);
            const hasDiff = evt.before_value || evt.after_value;
            return (
              <div key={evt.event_id} className={`bog-audit-event ${m.cls}`}>
                {/* Time */}
                <div>
                  <div className="bog-audit-time">{formatDate(evt.created_at)}</div>
                  <div className="bog-audit-time">{formatTime(evt.created_at)}</div>
                </div>

                {/* Operation */}
                <div>
                  <span className="bog-audit-op" style={{ color: m.color }}>{evt.operation}</span>
                  {evt.policy_triggered && (
                    <div style={{ fontSize: 10, color: 'var(--bog-amber)', marginTop: 2 }}>
                      ⚖️ {evt.policy_triggered}
                    </div>
                  )}
                </div>

                {/* Actor + entity */}
                <div>
                  <div style={{ fontWeight: 600, fontSize: 12 }}>{evt.actor_id}</div>
                  <div className="bog-audit-actor">
                    {evt.actor_role && <span style={{ marginRight: 6 }}>[{evt.actor_role}]</span>}
                    {evt.entity_id && <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{evt.entity_id.substring(0, 12)}…</span>}
                  </div>
                </div>

                {/* Expand diff */}
                {hasDiff && (
                  <button className="bog-audit-diff-btn" onClick={() => toggleExpand(evt.event_id)}>
                    {isExpanded ? '▲ Hide diff' : '▼ View diff'}
                  </button>
                )}

                {/* Expanded diff */}
                {isExpanded && hasDiff && (
                  <div style={{ gridColumn: '1 / -1', borderTop: '1px solid var(--bog-border)', paddingTop: 12, marginTop: 4 }}>
                    <DiffViewer before={evt.before_value} after={evt.after_value} />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default BOAuditTimeline;
