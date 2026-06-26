import { useState, useEffect } from 'react';
import { getDuplicateQueries } from './api';
import type { DuplicateQueryCluster, SavedQuery } from './types';

interface DuplicateQueriesPanelProps {
  datasourceId: string;
}

export default function DuplicateQueriesPanel({ datasourceId }: DuplicateQueriesPanelProps) {
  const [clusters, setClusters] = useState<DuplicateQueryCluster[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getDuplicateQueries()
      .then(setClusters)
  .catch((e) => { import('./utils/devLogger').then(({ devError }) => devError(e)).catch(() => {}); })
      .finally(() => setLoading(false));
  }, [datasourceId]);

  if (loading) return <div>Scanning for duplicates...</div>;

  return (
    <div className="duplicate-queries-panel">
      <h4>Duplicates in {datasourceId}</h4>
      {clusters.length === 0 && <p>No duplicate queries found.</p>}
      {clusters.map(cluster => (
        <div key={cluster.fingerprint} className="duplicate-cluster">
          <h4>{cluster.queries.length} queries seem identical:</h4>
          <ul>
            {(cluster.queries as SavedQuery[]).map(q => (
              <li key={q.id}>
                <span>{q.name}</span>
                <small>Owner: {q.owner_user_id}</small>
              </li>
            ))}
          </ul>
          <button>Review & Merge</button>
        </div>
      ))}
    </div>
  );
}