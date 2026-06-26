import { useState } from 'react';
import { Alert, Box, Button, Chip, CircularProgress, Stack, TextField, Typography } from '@mui/material';
import { runPreview } from '../api';
import getErrorMessage from '../../../utils/errors';

type Result = { query: string; old_rows: number; new_rows: number; diff_pct: number; totals: { old: number; new: number } };

export default function PreviewRunner({ fromVersion, toVersion }: { fromVersion: string; toVersion: string }) {
  const [queries, setQueries] = useState<string>('select 1;');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [results, setResults] = useState<Result[] | null>(null);

  const onRun = async () => {
    setLoading(true); setError(null); setResults(null);
    try {
      const qs = queries.split('\n').map(s => s.trim()).filter(Boolean);
      const data = await runPreview(fromVersion, toVersion, qs);
      setResults(data);
    } catch (e: unknown) {
      setError(getErrorMessage(e, 'Failed to run preview'));
    } finally { setLoading(false); }
  };

  return (
    <Box>
      <Stack spacing={1} sx={{ mb: 1 }}>
        <TextField
          label="Golden queries"
          placeholder="One query per line"
          size="small"
          multiline
          minRows={3}
          value={queries}
          onChange={(e) => setQueries(e.target.value)}
        />
        <Button variant="contained" size="small" onClick={onRun} disabled={!fromVersion || !toVersion || loading}>Run</Button>
      </Stack>
      {loading && <CircularProgress size={18} />}
      {error && <Alert severity="error">{error}</Alert>}
      {results && results.map((r) => (
        <Stack key={r.query} spacing={0.5} sx={{ mb: 1 }}>
          <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>{r.query}</Typography>
          <Stack direction="row" spacing={1}>
            <Chip size="small" label={`rows: ${r.old_rows} → ${r.new_rows}`} />
            <Chip size="small" color={r.diff_pct === 0 ? 'success' : 'warning'} label={`diff: ${(r.diff_pct*100).toFixed(2)}%`} />
            <Chip size="small" label={`totals: ${r.totals.old} → ${r.totals.new}`} />
          </Stack>
        </Stack>
      ))}
    </Box>
  );
}
