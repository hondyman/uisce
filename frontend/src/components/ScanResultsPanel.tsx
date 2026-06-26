 
import './ScanResultsModal.css';
import type { ScanResultItem } from './ScanResultsModal';

interface Props {
  opened: boolean;
  onClose: () => void;
  results: ScanResultItem[];
  onRetry: (datasourceId: string) => Promise<void> | void;
}

export default function ScanResultsPanel({ opened, onClose, results, onRetry }: Props) {
  if (!opened) return null;

  const successes = results.filter((r) => r.success).length;
  const failures = results.length - successes;

  return (
    <div className="scan-results-panel" role="region" aria-label={`Scan Results (${successes} ok, ${failures} failed)`}>
      <div className="scan-results-list">
        <h3>Scan Results ({successes} ok, {failures} failed)</h3>
        {results.map((r) => (
          <div key={r.tenant_instance_id} className="scan-results-item">
            <div className="scan-results-item-main">
              <div className="scan-results-name">{r.name || r.tenant_instance_id}</div>
              {!r.success && (<div className="scan-results-error">{r.error || 'Unknown error'}</div>)}
            </div>
            <div className="scan-results-actions">
              <div className={`scan-results-badge ${r.success ? 'success' : 'failed'}`}>{r.success ? 'Success' : 'Failed'}</div>
              {!r.success && (
                <button className="scan-retry-btn" onClick={() => onRetry(r.tenant_instance_id)}>Retry</button>
              )}
            </div>
          </div>
        ))}

        <div className="scan-results-footer">
          <button className="scan-close-btn" onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}
