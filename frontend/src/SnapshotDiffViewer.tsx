import { useState, useEffect } from 'react';
import { devError } from './utils/devLogger';
import { compareSnapshots } from './api';
import type { SnapshotDiff, SnapshotDiffItem } from './types';

interface SnapshotDiffViewerProps {
  snapshotId: string;
  compareToId: string; // e.g., 'current' or another snapshot ID
}

function DiffSection({ title, items }: { title: string, items?: SnapshotDiffItem[] }) {
  if (!items || items.length === 0) return null;
  return (
    <div className="diff-section">
      <h5>{title}</h5>
      <table>
        <thead>
          <tr>
            <th>Field</th>
            <th>Before</th>
            <th>After</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item, i) => (
            <tr key={i} className={`change-${item.change_type}`}>
              <td>{item.field}</td>
              <td>{item.before}</td>
              <td>{item.after}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function SnapshotDiffViewer({ snapshotId, compareToId }: SnapshotDiffViewerProps) {
  const [diff, setDiff] = useState<SnapshotDiff | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!snapshotId || !compareToId) return;
    setLoading(true);
    compareSnapshots(snapshotId, compareToId)
      .then(setDiff)
      .catch((e) => { devError(e); })
      .finally(() => setLoading(false));
  }, [snapshotId, compareToId]);

  if (loading) return <div>Loading diff...</div>;
  if (!diff) return <div>Could not load diff.</div>;

  return (
    <div className="snapshot-diff-viewer">
      <h4>Snapshot Comparison</h4>
  <DiffSection title="Filter Changes" items={diff.filters_diff} />
  <DiffSection title="Metric Changes" items={diff.metrics_diff} />
  <DiffSection title="Layout Changes" items={diff.layout_diff} />
  <DiffSection title="Semantic Context Changes" items={diff.semantic_diff} />
    </div>
  );
}