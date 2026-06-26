import { useEffect, useState } from 'react';
import { getPreviewDiff } from './api';
import type { PreviewDiff } from './types';

interface PreviewDiffTabProps {
  savedId: string;
}

function formatPercentage(before: number, after: number): string {
  if (before === 0) return after > 0 ? "+∞%" : "0%";
  const change = ((after - before) / before) * 100;
  return `${change >= 0 ? '+' : ''}${change.toFixed(1)}%`;
}

export default function PreviewDiffTab({ savedId }: PreviewDiffTabProps) {
  const [diff, setDiff] = useState<PreviewDiff | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    getPreviewDiff(savedId)
      .then(setDiff)
      .catch(() => setError('No diff available for the last two runs.'))
      .finally(() => setLoading(false));
  }, [savedId]);

  if (loading) return <div>Loading diff...</div>;
  if (error) return <div>{error}</div>;
  if (!diff) return <div>No diff data found.</div>;

  const { row_count, columns, filters_changed } = diff;
  const safeRowCount: { before: number; after: number } = {
    before: typeof row_count?.before === 'number' ? row_count!.before! : 0,
    after: typeof row_count?.after === 'number' ? row_count!.after! : 0,
  };
  const safeColumns: { added: string[]; removed: string[] } = {
    added: Array.isArray(columns?.added) ? columns!.added! : [],
    removed: Array.isArray(columns?.removed) ? columns!.removed! : [],
  };

  return (
    <div className="preview-diff-tab">
      <h3>Changes Since Last Run</h3>
      <div className="diff-cards">
        <div className="diff-card">
          <h4>Row Count</h4>
          <p>
            {safeRowCount.before?.toLocaleString() ?? '0'} → {safeRowCount.after?.toLocaleString() ?? '0'}
            <span className={safeRowCount.after >= safeRowCount.before ? 'positive' : 'negative'}>
              ({formatPercentage(safeRowCount.before ?? 0, safeRowCount.after ?? 0)})
            </span>
          </p>
        </div>
        <div className="diff-card">
          <h4>Schema</h4>
          {safeColumns.added.length > 0 && <p className="positive">Added: {safeColumns.added.join(', ')}</p>}
          {safeColumns.removed.length > 0 && <p className="negative">Removed: {safeColumns.removed.join(', ')}</p>}
          {safeColumns.added.length === 0 && safeColumns.removed.length === 0 && <p>No changes</p>}
        </div>
        <div className="diff-card">
          <h4>Filters</h4>
          <p>{filters_changed ? 'Filters or query logic have changed.' : 'No changes'}</p>
        </div>
      </div>
    </div>
  );
}