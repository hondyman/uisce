import { useState, useEffect, useCallback } from 'react';
import { getLatestGovernanceSnapshot } from '../../api';
import { GovernanceSnapshot } from './types';
import './GovernanceOverview.css';

// Sub-components for the overview panel
const MetricCard = ({ title, value, warning = false }: { title: string; value: number | string; warning?: boolean }) => (
  <div className={`metric-card ${warning ? 'warning' : ''}`}>
    <div className="metric-value">{value}</div>
    <div className="metric-title">{title}</div>
  </div>
);

const ProgressBar = ({ title, percent }: { title: string; percent: number }) => (
  <div className="progress-bar-card">
    <div className="progress-bar-title">{title}</div>
    <div className="progress-bar-container">
      <div className="progress-bar-fill" data-progress-percent={percent}>
        {percent.toFixed(1)}%
      </div>
    </div>
  </div>
);

const AuditTimeline = ({ events }: { events: { actor: string; action: string }[] }) => (
  <div className="audit-timeline">
    <h4>Recent Governance Events</h4>
    <ul>
      {events.map((event, index) => (
        <li key={index}>
          <strong>{event.actor}</strong> {event.action}
        </li>
      ))}
    </ul>
  </div>
);

export default function GovernanceOverview() {
  const [snapshot, setSnapshot] = useState<GovernanceSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSnapshot = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getLatestGovernanceSnapshot();
      setSnapshot(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch governance overview');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSnapshot();
  }, [fetchSnapshot]);

  if (loading) return <div>Loading governance summary...</div>;
  if (error) return <div className="error-text">Error: {error}</div>;
  if (!snapshot) return <div>No governance data available.</div>;

  return (
    <div className="governance-overview-panel">
      <div className="metrics-grid">
        <MetricCard title="Certified Models" value={snapshot.certified_model_count ?? 0} />
        <MetricCard title="Unresolved Requests" value={snapshot.unresolved_request_count ?? 0} />
        <MetricCard
          title="Risky Claims"
          value={snapshot.risky_claim_count ?? 0}
          warning={(snapshot.risky_claim_count ?? 0) > 0}
        />
      </div>
      <ProgressBar title="Semantic Coverage" percent={snapshot.semantic_coverage_percent ?? 0} />
      <AuditTimeline events={(snapshot.recent_events as { actor: string; action: string }[]) ?? []} />
    </div>
  );
}