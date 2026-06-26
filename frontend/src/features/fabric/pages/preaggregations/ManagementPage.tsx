import React, { useState } from 'react';
import { Box, Typography, Paper, TextField, Button, Stack, Select, MenuItem, FormControl, InputLabel, CircularProgress, Alert, TableContainer, Table, TableHead, TableRow, TableCell, TableBody } from '@mui/material';
import axios from 'axios';
import yaml from 'js-yaml';
import { devLog } from '../../../../utils/devLogger';
import getErrorMessage from '../../../../utils/errors';
import { PreAggregationConfig, ScheduledPreAggregation, JobRun } from '../types';

interface ManagementPageProps {
  tenantId?: string;
  datasourceId?: string;
}

const ManagementPage: React.FC<ManagementPageProps> = (_props: ManagementPageProps) => {
  const [cube, setCube] = useState('');
  const [preYaml, setPreYaml] = useState(`# Example pre-aggregation\nname: sales_rollup\ntype: rollup\ndimensions: [store_id, product_id]\nmeasures: [sales]\nstorage: materialized_view\n`);
  const [sql, setSql] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [executed, setExecuted] = useState<boolean | null>(null);
  const [storage, setStorage] = useState('materialized_view');

  // use shared getErrorMessage from utils/errors

  const parsePre = (): PreAggregationConfig => {
    try {
      const obj = yaml.load(preYaml) as unknown;
      // support both a root pre_aggregation object or a single map describing it
      if (obj && typeof obj === 'object') {
        const map = obj as Record<string, unknown>;
        if ('pre_aggregation' in map && map.pre_aggregation && typeof map.pre_aggregation === 'object') return map.pre_aggregation as PreAggregationConfig;
        if ('preAggregations' in map) {
          const preAggs = (map as { preAggregations?: unknown }).preAggregations;
          if (Array.isArray(preAggs) && preAggs.length > 0) {
            return preAggs[0] as PreAggregationConfig;
          }
        }
        return map as PreAggregationConfig;
      }
      throw new Error('Parsed YAML is not an object');
    } catch (e: unknown) {
      throw new Error('Invalid YAML: ' + getErrorMessage(e));
    }
  };

  const handleGenerate = async () => {
    setError(null);
    setSql(null);
    setExecuted(null);
    let pre: PreAggregationConfig;
    try {
      pre = parsePre();
    } catch (e: unknown) {
      setError(getErrorMessage(e));
      return;
    }

    // normalize storage selection into pre object
    pre.storage = storage;

    setLoading(true);
    try {
      const res = await axios.post('/pre_aggregations/generate', { cube, pre });
      devLog('generate response', res.data);
      setSql(res.data.sql || null);
    } catch (e: unknown) {
      setError(getErrorMessage(e) || 'Request failed');
    } finally {
      setLoading(false);
    }
  };

  const handleRefresh = async (execute = false) => {
    setError(null);
    setSql(null);
    setExecuted(null);
    let pre: PreAggregationConfig;
    try {
      pre = parsePre();
    } catch (e: unknown) {
      setError((e as Error)?.message || getErrorMessage(e));
      return;
    }
    pre.storage = storage;

    setLoading(true);
    try {
      const res = await axios.post('/pre_aggregations/refresh', { cube, pre, execute });
      devLog('refresh response', res.data);
      setSql(res.data.sql || null);
      setExecuted(!!res.data.executed);
      if (res.data.note) setError(String(res.data.note));
    } catch (e: unknown) {
      setError(getErrorMessage(e) || 'Request failed');
    } finally {
      setLoading(false);
    }
  };

  // fetch scheduled pre-aggregations from DB-driven admin endpoint
  const [scheduled, setScheduled] = useState<ScheduledPreAggregation[] | null>(null);
  const [history, setHistory] = useState<JobRun[] | null>(null);

  const fetchScheduled = async () => {
    try {
      const { data } = await axios.get('/admin/pre_aggregations');
      setScheduled(data.pre_aggregations as ScheduledPreAggregation[]);
    } catch (err: unknown) {
  devLog('failed to fetch scheduled pre-aggregations', getErrorMessage(err));
    }
  };

  React.useEffect(() => { fetchScheduled(); }, []);

  const forceRun = async (item: ScheduledPreAggregation) => {
    try {
      setLoading(true);
      const res = await axios.post('/admin/pre_aggregations/force', { cube: item.cube, name: item.name, execute: true });
      devLog('force run', res.data);
      setSql(res.data.sql || null);
      setExecuted(!!res.data.executed);
      await fetchScheduled();
    } catch (e: unknown) {
  setError(getErrorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const removeScheduled = async (item: ScheduledPreAggregation) => {
    try {
      setLoading(true);
      await axios.post('/admin/pre_aggregations/remove', { cube: item.cube, name: item.name });
      await fetchScheduled();
    } catch (e: unknown) {
  setError(getErrorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  const fetchHistory = async (item: ScheduledPreAggregation) => {
    try {
      setLoading(true);
      const jobID = `${item.cube}::${item.name}`;
      const res = await axios.post('/admin/pre_aggregations/history', { job_id: jobID });
      setHistory(res.data.runs as JobRun[]);
    } catch (e: unknown) {
  setError(getErrorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>Pre-Aggregation Management</Typography>

      <Paper sx={{ p: 2, mb: 2 }}>
        <Stack spacing={2}>
          <TextField label="Cube / Model Name" value={cube} onChange={(e) => setCube(e.target.value)} fullWidth />

          <FormControl fullWidth>
            <InputLabel id="storage-label">Storage</InputLabel>
            <Select labelId="storage-label" value={storage} label="Storage" onChange={(e) => setStorage(String(e.target.value))}>
              <MenuItem value="materialized_view">Materialized View</MenuItem>
              <MenuItem value="table">Table</MenuItem>
            </Select>
          </FormControl>

          <TextField
            label="Pre-Aggregation (YAML)"
            value={preYaml}
            onChange={(e) => setPreYaml(e.target.value)}
            multiline
            minRows={8}
            fullWidth
          />

          <Stack direction="row" spacing={2}>
            <Button variant="contained" onClick={handleGenerate} disabled={loading}>Generate DDL</Button>
            <Button variant="outlined" onClick={() => handleRefresh(false)} disabled={loading}>Refresh (no exec)</Button>
            <Button variant="contained" color="secondary" onClick={() => handleRefresh(true)} disabled={loading}>Refresh & Execute</Button>
            {loading && <CircularProgress size={24} />}
          </Stack>

          {error && <Alert severity="error">{error}</Alert>}

          {sql && (
            <Paper sx={{ p: 2, backgroundColor: '#f7f7f7' }}>
              <Typography variant="subtitle1">Generated SQL</Typography>
              <Box component="pre" sx={{ whiteSpace: 'pre-wrap', m: 0 }}>{sql}</Box>
              {executed !== null && (
                <Typography variant="body2" sx={{ mt: 1 }}>{executed ? 'Execution requested (see server logs).' : 'Execution not performed.'}</Typography>
              )}
            </Paper>
          )}
        </Stack>
      </Paper>

      <Paper sx={{ p: 2 }}>
        <Typography variant="h6">Scheduled Pre-Aggregations</Typography>
        {!scheduled && <Typography variant="body2">No scheduled pre-aggregations found.</Typography>}
        {scheduled && (
          <Box sx={{ mt: 1 }}>
            {scheduled.length === 0 && <Typography variant="body2">No scheduled pre-aggregations found.</Typography>}
            {scheduled.length > 0 && (
              <Box>
                {scheduled.map((s) => (
                  <Paper key={`${s.cube}::${s.name}`} sx={{ p: 1, mb: 1 }}>
                    <Stack direction="row" spacing={2} alignItems="center">
                      <Box sx={{ flex: 1 }}>
                        <Typography><strong>{s.cube} :: {s.name}</strong></Typography>
                        <Typography variant="caption">Scheduled: {s.scheduled || '—'} · Storage: {s.storage || '—'} · Last run: {s.last_run || '—'}</Typography>
                      </Box>
                      <Stack direction="row" spacing={1}>
                        <Button size="small" onClick={() => forceRun(s)}>Force Run</Button>
                        <Button size="small" onClick={() => removeScheduled(s)}>Remove</Button>
                        <Button size="small" onClick={() => fetchHistory(s)}>History</Button>
                      </Stack>
                    </Stack>
                  </Paper>
                ))}
              </Box>
            )}
          </Box>
        )}

        {history && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle1">Run History</Typography>
            {history.length === 0 && <Typography variant="body2">No runs found for selected job.</Typography>}
            {history.length > 0 && (
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Started</TableCell>
                      <TableCell>Finished</TableCell>
                      <TableCell>Success</TableCell>
                      <TableCell>Message</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {history.map((r) => (
                      <TableRow key={r.id}>
                        <TableCell>{new Date(r.started_at).toLocaleString()}</TableCell>
                        <TableCell>{r.finished_at ? new Date(r.finished_at).toLocaleString() : '—'}</TableCell>
                        <TableCell>{r.success ? 'Yes' : 'No'}</TableCell>
                        <TableCell>{r.message || ''}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </Box>
        )}
      </Paper>
    </Box>
  );
};

export default ManagementPage;
