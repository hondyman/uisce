import { useState, useEffect } from 'react';
import { useNotification } from './hooks/useNotification';
import { devError } from './utils/devLogger';
import { getSuggestions, getSavedQuery, logSearchFeedback } from './api';
import type { SemanticSearchResult, FullSavedQuery } from './types';
import SearchResultCard from './SearchResultCard';
import ExplainMatchPanel from './ExplainMatchPanel';

interface SuggestedForYouPanelProps {
  datasourceId: string;
  onOpenQuery: (q: FullSavedQuery) => void;
}

export default function SuggestedForYouPanel({ datasourceId, onOpenQuery }: SuggestedForYouPanelProps) {
  const [suggestions, setSuggestions] = useState<SemanticSearchResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [explainResult, setExplainResult] = useState<SemanticSearchResult | null>(null);
  const notification = useNotification();

  useEffect(() => {
    if (!datasourceId) return;
    setLoading(true);
    // In a real app, you'd also pass the user ID.
    getSuggestions("user-123", datasourceId)
    .then(setSuggestions)
  .catch((e) => { devError(e); })
      .finally(() => setLoading(false));
  }, [datasourceId]);

  const handleOpen = async (result: SemanticSearchResult) => {
    logSearchFeedback({ query: 'suggestion', result_id: result.id, result_type: result.type, action: 'clicked' });
    if (result.type === 'query') {
      const fullQuery = await getSavedQuery(result.id);
      onOpenQuery(fullQuery);
    } else {
      notification.info(`Opening workbook ${result.name} is not implemented yet.`);
    }
  };

  const handleFeedback = (result: SemanticSearchResult, action: 'favorited' | 'ignored') => {
    logSearchFeedback({ query: 'suggestion', result_id: result.id, result_type: result.type, action });
    // Optionally, remove the item from suggestions if ignored
    if (action === 'ignored') {
      setSuggestions(s => s.filter(sugg => sugg.id !== result.id));
    }
  };

  if (loading || !suggestions.length) return null;

  return (
    <div className="suggested-panel">
      <h4>Suggested for You</h4>
      {suggestions.map(s => (
        <SearchResultCard key={s.id} result={s} onOpen={handleOpen} onExplain={setExplainResult} onFeedback={handleFeedback} />
      ))}
      {explainResult && (
        <ExplainMatchPanel result={explainResult} onClose={() => setExplainResult(null)} />
      )}
    </div>
  );
}