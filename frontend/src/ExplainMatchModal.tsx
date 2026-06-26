import type { SemanticSearchResult } from './types';

export default function ExplainMatchModal({ result, onClose }: { result: SemanticSearchResult; onClose: () => void }) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={e => e.stopPropagation()}>
        <h2>Why this matched</h2>
        <p><strong>Matched Concepts:</strong> {result.matched_concepts?.join(', ') || 'N/A'}</p>
        <p><strong>Similarity Score:</strong> {(result.score * 100).toFixed(1)}%</p>
        <p><strong>Source:</strong> {result.source_summary || 'N/A'}</p>
        <div className="modal-actions">
          <button onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}