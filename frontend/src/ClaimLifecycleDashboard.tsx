import { useState, useEffect, useCallback } from 'react';
import { getClaimLifecycleSnapshot } from './api';
import { ClaimLifecycleSnapshot, ClaimLifecycleEvent } from './types';
import './IndexMonitorDashboard.css'; // Re-use styles
import ClaimsTable from './ClaimsTable';

// Re-usable components
const MetricCard = ({ title, value, warning = false }: { title: string; value: number | string; warning?: boolean }) => (
  <div className={`metric-card ${warning ? 'warning' : ''}`}>
    <div className="metric-value">{value}</div>
    <div className="metric-title">{title}</div>
  </div>
);

const LifecycleTimeline = ({ events }: { events: ClaimLifecycleEvent[] }) => (
  <div className="job-timeline"> {/* Re-use class name */}
    <h4>Recent Claim Events</h4>
    <table className="governance-table">
      <thead>
        <tr>
          <th>Timestamp</th>
          <th>Event</th>
          <th>Actor</th>
          <th>Notes</th>
        </tr>
      </thead>
      <tbody>
        {events.map(event => (
          <tr key={event.id}>
            <td>{new Date(event.timestamp).toLocaleString()}</td>
            <td><span className={`status-badge status-${event.event_type}`}>{event.event_type.replace(/_/g, ' ')}</span></td>
            <td>{event.actor_user_id}</td>
            <td>{event.notes || '-'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

export default function ClaimLifecycleDashboard() {
  const [snapshot, setSnapshot] = useState<ClaimLifecycleSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>('active');

  const fetchSnapshot = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getClaimLifecycleSnapshot();
      setSnapshot(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch lifecycle data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSnapshot();
  }, [fetchSnapshot]);

  if (loading) return <div>Loading claim lifecycle dashboard...</div>;
  if (error) return <div className="error-text">Error: {error}</div>;
  if (!snapshot) return <div>No claim data available.</div>;

  const injectedClaimLifecycle = `
    .claim-table-container{ margin-top: 2rem }
    .dashboard-tabs{ border-bottom: none; margin-bottom: 1rem }
  `;

  return (
    <>
      <style dangerouslySetInnerHTML={{ __html: injectedClaimLifecycle }} />
      <div className="index-monitor-dashboard"> {/* Re-use class name */}
        <h2>Claim Lifecycle Overview</h2>
        <div className="metrics-grid">
          <MetricCard title="Active Claims" value={snapshot.active_count} />
          <MetricCard title="Expiring Soon" value={snapshot.expiring_soon_count} warning={snapshot.expiring_soon_count > 0} />
          <MetricCard title="Renewal Requests" value={snapshot.renewal_requested_count} />
          <MetricCard title="Expired Claims" value={snapshot.expired_count} />
          <MetricCard title="Revoked Claims" value={snapshot.revoked_count} />
        </div>
        <LifecycleTimeline events={snapshot.recent_events} />

        <div className="claim-table-container">
          <h3>Claims List</h3>
          <div className="dashboard-tabs">
            <button onClick={() => setFilterStatus('active')} className={filterStatus === 'active' ? 'active' : ''}>Active</button>
            <button onClick={() => setFilterStatus('expiring')} className={filterStatus === 'expiring' ? 'active' : ''}>Expiring</button>
            <button onClick={() => setFilterStatus('renewal_requested')} className={filterStatus === 'renewal_requested' ? 'active' : ''}>Pending Renewal</button>
            <button onClick={() => setFilterStatus('expired')} className={filterStatus === 'expired' ? 'active' : ''}>Expired</button>
            <button onClick={() => setFilterStatus('revoked')} className={filterStatus === 'revoked' ? 'active' : ''}>Revoked</button>
          </div>
          <ClaimsTable statusFilter={filterStatus} />
        </div>
      </div>
    </>
  );
}