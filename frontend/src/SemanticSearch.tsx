import { useState, useMemo, useEffect } from 'react';
import { useNotification } from './hooks/useNotification';
import { devError } from './utils/devLogger';
import { semanticSearch, getSavedQuery, logSearchFeedback } from './api';
import { useDebounce } from './hooks/useDebounce';
import type { SemanticSearchResult, SearchFilters, FullSavedQuery } from './types';
import SearchFiltersPanel from './SearchFiltersPanel';
import ExplainMatchPanel from './ExplainMatchPanel';
import SearchResultCard from './SearchResultCard';


interface SemanticSearchProps {
  onOpenQuery: (q: FullSavedQuery) => void;
  // onOpenWorkbook: (id: string) => void;
}

export default function SemanticSearchContainer({ onOpenQuery }: SemanticSearchProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SemanticSearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [filters, setFilters] = useState<SearchFilters>({ type: ['query', 'workbook'], scope: 'all', tags: [] });
  const [showOnlyAccessible, setShowOnlyAccessible] = useState(true);
  const [explainResult, setExplainResult] = useState<SemanticSearchResult | null>(null);
  const debouncedQuery = useDebounce(query, 300);
  const notification = useNotification();

  useEffect(() => {
    if (debouncedQuery.length < 3) {
      setResults([]);
      return;
    }
    setLoading(true);
    semanticSearch({ query: debouncedQuery, filters })
      .then(setResults)
        .catch((e) => { devError(e); })
        .finally(() => setLoading(false));
  }, [debouncedQuery, filters]);

  const handleFeedback = (result: SemanticSearchResult, action: 'favorited' | 'ignored') => {
    logSearchFeedback({
      query: debouncedQuery,
      result_id: result.id,
      result_type: result.type,
      action,
    });
  };

  const handleOpen = async (result: SemanticSearchResult) => {
    logSearchFeedback({
      query: debouncedQuery,
      result_id: result.id,
      result_type: result.type,
      action: 'clicked',
    });
    if (result.type === 'query') {
      // The search result is a summary. We need to fetch the full query to open it.
      const fullQuery = await getSavedQuery(result.id);
      onOpenQuery(fullQuery);
    } else {
      // onOpenWorkbook(result.id);
      const notification = useNotification();
      notification.info(`Opening workbook ${result.name} is not implemented yet.`);
    }
    setQuery(''); // Clear search after opening
    setResults([]);
  };

  const displayedResults = useMemo(() => {
    return showOnlyAccessible ? results.filter(r => r.has_access) : results;
  }, [results, showOnlyAccessible]);

  return (
    <div className="semantic-search-container">
      <div className="semantic-search-bar">
        <input
          type="search"
          placeholder="🔍 Search by meaning… e.g. 'churn trends in APAC'"
          value={query}
          onChange={e => setQuery(e.target.value)}
        />
      </div>
      <div className="search-controls">
        <SearchFiltersPanel filters={filters} onChange={setFilters} />
        <label className="access-toggle">
          <input type="checkbox" checked={showOnlyAccessible} onChange={e => setShowOnlyAccessible(e.target.checked)} />
          Show only assets I can access
        </label>
      </div>

      {loading && <div className="search-results-popup loading">Loading...</div>}
      {!loading && displayedResults.length > 0 && (
        <div className="search-results-grid">
          {displayedResults.map(res => (
            <SearchResultCard key={res.id} result={res} onOpen={handleOpen} onExplain={setExplainResult} onFeedback={handleFeedback} />
          ))}
        </div>
      )}
      {explainResult && (
        <ExplainMatchPanel result={explainResult} onClose={() => setExplainResult(null)} />
      )}
    </div>
  );
}