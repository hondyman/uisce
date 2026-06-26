import { useState, useEffect } from 'react';
import { getFolderAnalytics } from './api';
import { devError } from './utils/devLogger';
import type { FolderAnalytics } from './types';

function StatCard({ label, value }: { label: string; value: string | number | undefined }) {
  return (
    <div className="stat-card">
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  );
}

export default function FolderAnalyticsPanel({ folderId }: { folderId: string }) {
  const [stats, setStats] = useState<FolderAnalytics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    getFolderAnalytics(folderId)
      .then(setStats)
  .catch((e) => { devError(e); })
      .finally(() => setLoading(false));
  }, [folderId]);

  if (loading) {
    return <div className="analytics-panel loading">Loading analytics...</div>;
  }

  if (!stats) {
    return <div className="analytics-panel error">Could not load analytics.</div>;
  }

  return (
    <div className="analytics-panel">
      <h5>Analytics (30d)</h5>
      <div className="stats-grid">
        <StatCard label="Runs" value={stats.run_count_30d} />
        <StatCard label="Exports" value={stats.export_count_30d} />
        <StatCard label="Viewers" value={stats.viewer_count_30d} />
      </div>
      {/* Placeholder for Top Items and Change Log */}
    </div>
  );
}