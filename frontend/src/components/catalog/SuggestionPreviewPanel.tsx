import { useState } from 'react';
import './SuggestionPreviewModal.css';

type Suggestion = {
  id: string;
  title: string;
  description: string;
  sourceEntity: string;
  targetEntity: string;
  edgeType: string;
  cardinality: string;
  fkColumn: string;
  confidence: number;
  reasoning: string;
  dismissible: boolean;
};

type Props = {
  suggestion: Suggestion;
  onApply: (suggestion: Suggestion) => void;
  onDismiss: (suggestion: Suggestion, reason: string) => void;
  onClose: () => void;
};

export default function SuggestionPreviewPanel({ suggestion, onApply, onDismiss, onClose }: Props) {
  const [dismissReason, setDismissReason] = useState('');

  return (
    <div className="suggestion-panel" role="region" aria-label="Suggestion Preview">
      <div className="suggestion-panel-inner">
        <div className="suggestion-panel-header">
          <h3>{suggestion.title}</h3>
          <button onClick={onClose} aria-label="Close">×</button>
        </div>
        <div className="suggestion-panel-body">
          <p className="text-gray-700 mb-2">{suggestion.description}</p>
          <p className="text-sm text-gray-600 mb-2"><strong>Confidence:</strong> {(suggestion.confidence * 100).toFixed(1)}%</p>
          <p className="text-sm text-gray-600 mb-4"><strong>Reasoning:</strong> {suggestion.reasoning}</p>

          <div className="suggestion-panel-actions">
            <button onClick={() => onApply(suggestion)} className="btn-primary">Apply</button>
            {suggestion.dismissible && (
              <>
                <input type="text" placeholder="Reason for dismissal" value={dismissReason} onChange={(e) => setDismissReason(e.target.value)} className="dismiss-input" />
                <button onClick={() => onDismiss(suggestion, dismissReason)} className="btn-danger">Dismiss</button>
              </>
            )}
            <button onClick={onClose} className="btn-cancel">Cancel</button>
          </div>
        </div>
      </div>
    </div>
  );
}
