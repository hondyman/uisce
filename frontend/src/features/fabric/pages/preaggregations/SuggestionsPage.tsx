import React, { useEffect, useState, Suspense } from 'react';
import { Box, Typography, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, CircularProgress, Alert, Button } from '@mui/material';
import { useLocation } from 'react-router-dom';
import LazySyntaxHighlighter from '../../../../components/LazySyntaxHighlighter';
import yaml from 'js-yaml';
import { devLog } from '../../../../utils/devLogger';
import { PreAggSuggestion } from '../types';
import { useNotification } from '../../../../hooks/useNotification';

// Mock API call - typed
const fetchSuggestions = async (params: { tenant_instance_id?: string | null; model_name?: string | null }): Promise<{ suggestions: PreAggSuggestion[] }> => {
  devLog('Fetching suggestions with params:', params);
  return {
    suggestions: [
      { id: 'sug1', model_name: 'orders', hit_count: 15, suggested: { name: 'orders_daily_rollup', type: 'rollup', measures: ['count'], dimensions: ['status'], timeDimensionReference: 'created_at', granularity: 'day' } },
    ]
  };
};

const applySuggestion = async (suggestion: PreAggSuggestion) => {
  devLog('Applying suggestion:', suggestion);
  // In a real app, this would be a POST request.
  return { status: 'ok' };
};

const SuggestionsPage: React.FC = () => {
  const [suggestions, setSuggestions] = useState<PreAggSuggestion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const notification = useNotification();
  const location = useLocation();

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const datasourceId = params.get('tenant_instance_id');
    const modelName = params.get('model_name');

    const loadData = async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetchSuggestions({ tenant_instance_id: datasourceId, model_name: modelName });
        setSuggestions(res.suggestions);
      } catch (e: unknown) {
        setError((e as Error)?.message || 'Failed to fetch suggestions.');
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [location.search]);

  const handleAccept = async (suggestion: PreAggSuggestion) => {
    try {
      await applySuggestion(suggestion);
      setSuggestions(prev => prev.filter(s => s.id !== suggestion.id));
    } catch (e) {
      notification.error('Failed to apply suggestion.');
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>Pre-Aggregation Suggestions</Typography>
      {loading && <CircularProgress />}
      {error && <Alert severity="error">{error}</Alert>}
      {!loading && !error && (
        <Paper>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Model</TableCell>
                  <TableCell>Hit Count</TableCell>
                  <TableCell>Suggested YAML</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {suggestions.map((sug) => (
                  <TableRow key={sug.id}>
                    <TableCell>{sug.model_name}</TableCell>
                    <TableCell>{sug.hit_count}</TableCell>
                    <TableCell>
                      <Suspense fallback={<div>Loading code...</div>}>
                        <div className="lazy-syntax-wrapper">
                          <LazySyntaxHighlighter language="yaml">
                            {yaml.dump({ preAggregations: [sug.suggested] })}
                          </LazySyntaxHighlighter>
                        </div>
                      </Suspense>
                    </TableCell>
                    <TableCell>
                      <Button variant="contained" onClick={() => handleAccept(sug)}>Accept</Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      )}
    </Box>
  );
};

export default SuggestionsPage;