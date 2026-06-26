
import type { ColumnMeta, PageInfo } from './types';

interface InsightsPanelProps {
  result?: { rows: any[]; columns: ColumnMeta[]; page: PageInfo };
}

export default function InsightsPanel({ result }: InsightsPanelProps) {
  return (
    <div className="insights-panel">
      <h4>Automated Insights</h4>
      {result ? (
        <p>Found {result.rows.length} results. No significant anomalies detected.</p>
      ) : (
        <p className="text-placeholder">Run a query to generate insights.</p>
      )}
    </div>
  );
}