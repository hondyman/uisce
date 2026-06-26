import { useState, useEffect } from 'react';
import { getImpactAnalysis } from './api';
import { devError } from './utils/devLogger';
import type { ImpactAnalysis } from './types';

interface ImpactPanelProps {
  assetId: string; // e.g., metric name like 'total_revenue'
}

export default function ImpactPanel({ assetId }: ImpactPanelProps) {
  const [impact, setImpact] = useState<ImpactAnalysis | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    getImpactAnalysis(assetId)
      .then(setImpact)
  .catch((e) => { devError(e); })
      .finally(() => setLoading(false));
  }, [assetId]);

  if (loading) {
    return <div>Loading impact...</div>;
  }

  if (!impact || (impact.queries.length === 0 && impact.workbooks.length === 0 && impact.dashboards.length === 0)) {
    return <div>No downstream dependencies found.</div>;
  }

  return (
    <div className="impact-panel">
      <h4>Impact Analysis</h4>
      <p>This asset is used in:</p>
  {impact.queries.length > 0 && (
        <>
          <h5>Queries</h5>
          <ul>
    {impact.queries.map((q: { id: string; name: string }) => <li key={q.id}>{q.name}</li>)}
          </ul>
        </>
      )}
      {impact.workbooks.length > 0 && (
        <>
          <h5>Workbooks</h5>
          <ul>
    {impact.workbooks.map((w: { id: string; name: string }) => <li key={w.id}>{w.name}</li>)}
          </ul>
        </>
      )}
      {impact.dashboards.length > 0 && (
        <>
          <h5>Dashboards</h5>
          <ul>
    {impact.dashboards.map((d: { id: string; name: string }) => <li key={d.id}>{d.name}</li>)}
          </ul>
        </>
      )}
    </div>
  );
}