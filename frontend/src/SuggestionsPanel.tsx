import { useEffect, useState } from 'react';
import { devError } from './utils/devLogger';
import { getSuggestedQueries } from './api';
import type { SuggestedQuery } from './types';

interface SuggestionsPanelProps {
  viewName: string;
  onOpen: (id: string) => void;
}

export default function SuggestionsPanel({ viewName, onOpen }: SuggestionsPanelProps) {
  const [suggestions, setSuggestions] = useState<SuggestedQuery[]>([]);

  useEffect(() => {
    if (viewName) {
  getSuggestedQueries(viewName).then(setSuggestions).catch((e) => { devError(e); });
    }
  }, [viewName]);

  if (!suggestions.length) {
    return null;
  }

  return (
    <div className="suggestions-panel">
      <h4>Suggested Queries</h4>
      <ul>
        {suggestions.map(s => (
          <li key={s.saved_query_id}>
            <button onClick={() => onOpen(s.saved_query_id)} title={`Reason: ${s.reason}`}>{s.name}</button>
          </li>
        ))}
      </ul>
    </div>
  );
}