import { useState, useMemo, useEffect } from 'react';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';
import FormControlLabel from '@mui/material/FormControlLabel';
import Checkbox from '@mui/material/Checkbox';
import CircularProgress from '@mui/material/CircularProgress';
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
}

export default function SemanticSearchContainer({ onOpenQuery }: SemanticSearchProps) {
  const theme = useTheme();
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
      const fullQuery = await getSavedQuery(result.id);
      onOpenQuery(fullQuery);
    } else {
      notification.info(`Opening workbook ${result.name} is not implemented yet.`);
    }
    setQuery('');
    setResults([]);
  };

  const displayedResults = useMemo(() => {
    return showOnlyAccessible ? results.filter(r => r.has_access) : results;
  }, [results, showOnlyAccessible]);

  const isDark = theme.palette.mode === 'dark';

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ p: 2, borderRadius: 1, border: 1, borderColor: 'divider', bgcolor: isDark ? '#1f2937' : '#fff' }}>
        <TextField
          fullWidth
          variant="outlined"
          placeholder="🔍 Search by meaning… e.g. 'churn trends in APAC'"
          value={query}
          onChange={e => setQuery(e.target.value)}
          size="small"
          InputProps={{
            startAdornment: <Typography sx={{ mr: 1 }}>🔍</Typography>,
          }}
        />
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
        <SearchFiltersPanel filters={filters} onChange={setFilters} />
        <FormControlLabel
          control={
            <Checkbox
              checked={showOnlyAccessible}
              onChange={e => setShowOnlyAccessible(e.target.checked)}
              size="small"
            />
          }
          label="Show only assets I can access"
        />
      </Box>

      {loading && (
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
          <CircularProgress size="small" sx={{ mr: 1 }} />
          <Typography variant="body2">Loading...</Typography>
        </Box>
      )}
      {!loading && displayedResults.length > 0 && (
        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 2 }}>
          {displayedResults.map(res => (
            <SearchResultCard key={res.id} result={res} onOpen={handleOpen} onExplain={setExplainResult} onFeedback={handleFeedback} />
          ))}
        </Box>
      )}
      {explainResult && (
        <ExplainMatchPanel result={explainResult} onClose={() => setExplainResult(null)} />
      )}
    </Box>
  );
}
