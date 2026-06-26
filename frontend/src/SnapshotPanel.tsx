import { useState, useEffect } from 'react';
import { listDashboardSnapshots, createDashboardSnapshot } from './api';
import type { DashboardSnapshot } from './types';

interface SnapshotPanelProps {
  dashboardId: string;
  onCompare: (snapshotId: string) => void;
}

export default function SnapshotPanel({ dashboardId, onCompare }: SnapshotPanelProps) {
  const [snapshots, setSnapshots] = useState<DashboardSnapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [newSnapshotName, setNewSnapshotName] = useState('');

  const fetchSnapshots = () => {
    setLoading(true);
    listDashboardSnapshots(dashboardId).then(setSnapshots).finally(() => setLoading(false));
  };

  useEffect(fetchSnapshots, [dashboardId]);

  const handleCreateSnapshot = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newSnapshotName.trim()) return;
    await createDashboardSnapshot(dashboardId, newSnapshotName);
    setNewSnapshotName('');
    fetchSnapshots(); // Refresh list
  };

  if (loading) {
    return <div>Loading snapshots...</div>;
  }

  return (
    <div className="snapshot-panel">
      <h4>Snapshots</h4>
      <div className="snapshot-list">
        {snapshots.map(s => (
          <div key={s.id} className="snapshot-item">
            <div>
              <strong>{s.name}</strong>
              <small>by {s.created_by} on {new Date(s.timestamp).toLocaleDateString()}</small>
            </div>
            <button onClick={() => onCompare(s.id)}>Compare</button>
          </div>
        ))}
      </div>
      <form onSubmit={handleCreateSnapshot} className="snapshot-form">
        <input
          type="text"
          value={newSnapshotName}
          onChange={e => setNewSnapshotName(e.target.value)}
          placeholder="New snapshot name..."
        />
        <button type="submit" disabled={!newSnapshotName.trim()}>Create</button>
      </form>
    </div>
  );
}