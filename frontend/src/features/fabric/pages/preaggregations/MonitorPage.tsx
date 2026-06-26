import React, { useEffect, useState } from 'react';
import { Box, Typography, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, CircularProgress, Alert, Chip } from '@mui/material';
import { useLocation } from 'react-router-dom';
import { devLog } from '../../../../utils/devLogger';
import { PreAggLog } from '../types';

// Mock API call - typed
const fetchLogs = async (params: { tenant_instance_id?: string | null; model_name?: string | null }): Promise<{ logs: PreAggLog[] }> => {
  devLog('Fetching logs with params:', params);
  // In a real app, this would be an API call, e.g., using axios or fetch.
  return {
    logs: [
      { id: '1', executed_at: new Date().toISOString(), model_name: 'orders', measures: ['count'], dimensions: ['status'], time_dimension: 'created_at', hit_preaggregation: true, preaggregation_name: 'orders_daily' },
      { id: '2', executed_at: new Date().toISOString(), model_name: 'orders', measures: ['total_amount'], dimensions: ['region'], time_dimension: 'created_at', hit_preaggregation: false, preaggregation_name: null },
    ]
  };
};

const MonitorPage: React.FC = () => {
  const [logs, setLogs] = useState<PreAggLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const location = useLocation();

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const datasourceId = params.get('tenant_instance_id');
    const modelName = params.get('model_name');

    const loadData = async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetchLogs({ tenant_instance_id: datasourceId, model_name: modelName });
        setLogs(res.logs);
      } catch (e: unknown) {
        setError((e as Error)?.message || 'Failed to fetch logs.');
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [location.search]);

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>Pre-Aggregation Monitor</Typography>
      {/* TODO: Add FilterBar component here */}
      {loading && <CircularProgress />}
      {error && <Alert severity="error">{error}</Alert>}
      {!loading && !error && (
        <Paper>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Time</TableCell>
                  <TableCell>Model</TableCell>
                  <TableCell>Measures</TableCell>
                  <TableCell>Dimensions</TableCell>
                  <TableCell>Hit Pre-Agg</TableCell>
                  <TableCell>Pre-Agg Name</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {logs.map((log: PreAggLog) => (
                  <TableRow key={log.id}>
                    <TableCell>{new Date(log.executed_at).toLocaleString()}</TableCell>
                    <TableCell>{log.model_name}</TableCell>
                    <TableCell>{log.measures.join(', ')}</TableCell>
                    <TableCell>{log.dimensions.join(', ')}</TableCell>
                    <TableCell>
                      <Chip label={log.hit_preaggregation ? 'Yes' : 'No'} color={log.hit_preaggregation ? 'success' : 'error'} size="small" />
                    </TableCell>
                    <TableCell>{log.preaggregation_name || 'N/A'}</TableCell>
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

export default MonitorPage;