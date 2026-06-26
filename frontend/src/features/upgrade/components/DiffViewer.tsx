import { useState, useEffect, useCallback } from 'react';
import { getViewIdentifier } from '../../../types/views';
import { devError } from '../../../utils/devLogger';
import type { DiffReport, Change } from '../../../types/upgrade-generated';
import { getDiffReport } from '../api';

interface DiffViewerProps {
  fromVersion: string;
  toVersion: string;
  onClose?: () => void;
}

export default function DiffViewer({ fromVersion, toVersion, onClose }: DiffViewerProps) {
  const [diffReport, setDiffReport] = useState<DiffReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'cubes' | 'views' | 'governance' | 'preAggs'>('cubes');

  const loadDiffReport = useCallback(async () => {
    try {
      setLoading(true);
      const report = await getDiffReport(fromVersion, toVersion);
      setDiffReport(report);
      setError(null);
    } catch (err) {
      setError('Failed to load diff report');
      devError('Diff report error:', err);
    } finally {
      setLoading(false);
    }
  }, [fromVersion, toVersion]);

  useEffect(() => {
    loadDiffReport();
  }, [loadDiffReport]);

  const renderChanges = (changes: Change[]) => (
    <ul className="space-y-1">
      {changes.map((c, idx) => (
        <li key={idx} className="text-sm">
          <span className="font-mono">{c.type}</span>{' '}
          {c.name && <strong>{c.name}</strong>}
          {c.join_path && <em> ({c.join_path})</em>}
        </li>
      ))}
    </ul>
  );

  if (loading) {
    return (
      <div className="diff-viewer">
        <p>Loading diff report...</p>
      </div>
    );
  }

  if (error || !diffReport) {
    return (
      <div className="diff-viewer">
        <p className="text-red-500">{error || 'No diff report available'}</p>
      </div>
    );
  }

  return (
    <div className="diff-viewer">
      <header className="flex gap-4 border-b pb-2 mb-4">
        {['cubes', 'views', 'governance', 'preAggs'].map((tab) => (
          <button
            key={tab}
            className={`px-3 py-1 rounded ${activeTab === tab ? 'bg-blue-500 text-white' : 'bg-gray-200'}`}
            onClick={() => setActiveTab(tab as 'cubes' | 'views' | 'governance' | 'preAggs')}
          >
            {tab.toUpperCase()}
          </button>
        ))}
      </header>

      <section className="mt-4">
        {activeTab === 'cubes' &&
          diffReport.cubes.map((cube) => (
            <div key={(cube as any).name || (cube as any).id} className="mb-4 p-4 border rounded">
              <h3 className="font-semibold text-lg">{(cube as any).title || (cube as any).name || (cube as any).id} ({(cube as any).status})</h3>
              {renderChanges(cube.changes)}
            </div>
          ))}

        {activeTab === 'views' &&
          diffReport.views.map((view) => (
            <div key={(view as any).name || getViewIdentifier(view)} className="mb-4 p-4 border rounded">
              <h3 className="font-semibold text-lg">{(view as any).title || (view as any).name || getViewIdentifier(view)} ({(view as any).status})</h3>
              {renderChanges(view.changes)}
            </div>
          ))}

        {activeTab === 'governance' &&
          diffReport.governance.map((g) => (
            <div key={g.name} className="mb-4 p-4 border rounded">
              <h3 className="font-semibold text-lg">{g.name}</h3>
              {renderChanges(g.changes)}
            </div>
          ))}

        {activeTab === 'preAggs' &&
          diffReport.pre_aggregations.map((p) => (
            <div key={p.name} className="mb-4 p-4 border rounded">
              <h3 className="font-semibold text-lg">{p.cube} / {p.name}</h3>
              <p className="text-sm text-gray-600">{p.status} — {p.reason}</p>
            </div>
          ))}
      </section>

      {onClose && (
        <div className="mt-4 flex justify-end">
          <button
            className="px-4 py-2 bg-gray-500 text-white rounded"
            onClick={onClose}
          >
            Close
          </button>
        </div>
      )}
    </div>
  );
}
