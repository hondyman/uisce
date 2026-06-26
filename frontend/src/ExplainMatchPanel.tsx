import type { SemanticSearchResult } from './types';

export default function ExplainMatchPanel({ result, onClose }: { result: SemanticSearchResult; onClose: () => void }) {
  return (
    <div className="explain-panel" role="dialog" aria-modal="false">
      <div className="explain-panel-header">
        <h3>Why this matched</h3>
        <button onClick={onClose} aria-label="Close">×</button>
      </div>
      <div className="explain-panel-body">
        <p><strong>Matched Concepts:</strong> {result.matched_concepts?.join(', ') || 'N/A'}</p>
        <p><strong>Similarity Score:</strong> {(result.score * 100).toFixed(1)}%</p>
        <p><strong>Source:</strong> {result.source_summary || 'N/A'}</p>
      </div>
    </div>
  );
}
