// frontend/src/components/catalog/SuggestionPreviewModal.tsx
import { useState } from "react";

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

export default function SuggestionPreviewModal({ suggestion, onApply, onDismiss, onClose }: Props) {
  const [dismissReason, setDismissReason] = useState("");

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
        <h2 className="text-xl font-semibold mb-4">{suggestion.title}</h2>
        <p className="text-gray-700 mb-2">{suggestion.description}</p>
        <p className="text-sm text-gray-600 mb-2">
          <strong>Confidence:</strong> {(suggestion.confidence * 100).toFixed(1)}%
        </p>
        <p className="text-sm text-gray-600 mb-4">
          <strong>Reasoning:</strong> {suggestion.reasoning}
        </p>
        <div className="flex space-x-2">
          <button
            onClick={() => onApply(suggestion)}
            className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
          >
            Apply
          </button>
          {suggestion.dismissible && (
            <>
              <input
                type="text"
                placeholder="Reason for dismissal"
                value={dismissReason}
                onChange={(e) => setDismissReason(e.target.value)}
                className="flex-1 px-2 py-1 border rounded"
              />
              <button
                onClick={() => onDismiss(suggestion, dismissReason)}
                className="bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600"
              >
                Dismiss
              </button>
            </>
          )}
          <button
            onClick={onClose}
            className="bg-gray-500 text-white px-4 py-2 rounded hover:bg-gray-600"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}