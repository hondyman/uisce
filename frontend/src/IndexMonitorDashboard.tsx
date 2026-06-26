import { useState, useEffect, useCallback } from 'react';
import { getIndexMonitorSnapshot } from './api';
import { IndexMonitorSnapshot, IndexJob, AssetFreshness } from './types';
import './IndexMonitorDashboard.css';
import MetricBreakdown from './MetricBreakdown';

// Re-usable components from GovernanceOverview
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

// New sub-components for this dashboard
const JobTimeline = ({ jobs }: { jobs: IndexJob[] }) => (
  <div className="job-timeline">
    <h4>Recent Indexing Jobs</h4>
    <table className="governance-table">
      <thead>
        <tr>
          <th>Status</th>
          <th>Job Type</th>
          <th>Triggered By</th>
          <th>Assets Affected</th>
          <th>Timestamp</th>
        </tr>
      </thead>
      <tbody>
        {jobs.map(job => (
          <tr key={job.id}>
            <td><span className={`status-badge status-${job.status}`}>{job.status}</span></td>
            <td>{job.job_type}</td>
            <td>{job.triggered_by}</td>
            <td>{job.affected_assets > 0 ? job.affected_assets : '-'}</td>
            <td>{new Date(job.started_at).toLocaleString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const StaleAssetList = ({ assets }: { assets: AssetFreshness[] }) => (
  <div className="stale-asset-list">
    <h4>Stale Assets (Not Indexed in &gt;7 Days)</h4>
    {assets.length === 0 ? (
      <p>No stale assets found. ✅</p>
    ) : (
      <table className="governance-table">
        <thead>
          <tr>
            <th>Asset Name</th>
            <th>Type</th>
            <th>Last Indexed</th>
            <th>Certified</th>
          </tr>
        </thead>
        <tbody>
          {assets.map(asset => (
            <tr key={asset.asset_id}>
              <td>{asset.asset_name}</td>
              <td>{asset.asset_type}</td>
              <td>{new Date(asset.last_indexed_at).toLocaleDateString()}</td>
              <td>{asset.certified ? '✅' : '❌'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    )}
  </div>
);


export default function IndexMonitorDashboard() {
  const [snapshot, setSnapshot] = useState<IndexMonitorSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSnapshot = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getIndexMonitorSnapshot();
      setSnapshot(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch index monitor data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSnapshot();
  }, [fetchSnapshot]);

  if (loading) return <div>Loading index monitor...</div>;
  if (error) return <div className="error-text">Error: {error}</div>;
  if (!snapshot) return <div>No index data available.</div>;

  const timeAgo = (dateString: string) => {
    const date = new Date(dateString);
    const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
    let interval = seconds / 31536000;
    if (interval > 1) return Math.floor(interval) + " years ago";
    interval = seconds / 2592000;
    if (interval > 1) return Math.floor(interval) + " months ago";
    interval = seconds / 86400;
    if (interval > 1) return Math.floor(interval) + " days ago";
    interval = seconds / 3600;
    if (interval > 1) return Math.floor(interval) + " hours ago";
    interval = seconds / 60;
    if (interval > 1) return Math.floor(interval) + " minutes ago";
    return Math.floor(seconds) + " seconds ago";
  };

  return (
    <div className="index-monitor-dashboard">
      <ProgressBar title="Semantic Health Score" percent={snapshot.semantic_health_score} />
      <MetricBreakdown
        certified={snapshot.certified_coverage}
        claims={snapshot.claim_alignment}
        usage={snapshot.usage_coverage}
        audit={snapshot.audit_completeness}
        risk={snapshot.risk_exposure}
      />
      <div className="metrics-grid">
        <MetricCard title="Last Full Refresh" value={timeAgo(snapshot.last_full_refresh)} />
        <MetricCard title="Certified Coverage" value={`${snapshot.certified_coverage.toFixed(1)}%`} />
        <MetricCard title="Unindexed Assets" value={snapshot.unindexed_asset_count} warning={snapshot.unindexed_asset_count > 0} />
      </div>
      <JobTimeline jobs={snapshot.recent_jobs} />
      <StaleAssetList assets={snapshot.stale_assets} />
    </div>
  );
}