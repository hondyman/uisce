import { useState, useEffect } from 'react';
import { devError } from './utils/devLogger';
import { compareSemanticViewVersions } from './api';
import type { SemanticDiff, MemberDiffItem } from './types';

interface SemanticDiffViewerProps {
  viewName: string;
  fromVersion: number;
  toVersion: number;
}

function DiffSection({ title, items }: { title: string, items?: MemberDiffItem[] }) {
  if (!items || items.length === 0) return null;
  return (
    <div className="diff-section">
      <h5>{title}</h5>
      <table>
        <thead>
          <tr>
            <th>Member</th>
            <th>Change</th>
            <th>Details</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item, i) => (
            <tr key={i} className={`change-${item.change_type}`}>
              <td>{item.name}</td>
              <td>{item.change_type}</td>
              <td>
                {item.change_type === 'modified' && (
                  <>
                    <span className="before">{item.before}</span> → <span className="after">{item.after}</span>
                  </>
                )}
                {item.change_type === 'added' && <span className="after">{item.after}</span>}
                {item.change_type === 'removed' && <span className="before">{item.before}</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function SemanticDiffViewer({ viewName, fromVersion, toVersion }: SemanticDiffViewerProps) {
  const [diff, setDiff] = useState<SemanticDiff | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!viewName || !fromVersion || !toVersion) return;
    setLoading(true);
    compareSemanticViewVersions(viewName, fromVersion, toVersion)
    .then(setDiff)
    .catch((e) => { devError(e); })
      .finally(() => setLoading(false));
  }, [viewName, fromVersion, toVersion]);

  if (loading) return <div>Loading semantic diff...</div>;
  if (!diff) return <div>Could not load diff.</div>;

  return (
    <div className="semantic-diff-viewer">
      <h4>Semantic View Diff: v{diff.from_version} vs v{diff.to_version}</h4>
      <DiffSection title="Dimension Changes" items={diff.dimensions} />
      <DiffSection title="Metric Changes" items={diff.metrics} />
    </div>
  );
}