import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Grid,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  Alert,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import RefreshIcon from '@mui/icons-material/Refresh';

const useStyles = makeStyles({
  container: {
    padding: '20px',
  },
  auditTable: {
    marginTop: '20px',
  },
  preformatted: {
    fontSize: '0.75rem',
    margin: '8px 0 0 0',
    overflow: 'auto',
    maxHeight: '200px',
  },
});

interface AuditRecord {
  id: string;
  bp_name: string;
  step_name: string;
  rule_name: string;
  passed: boolean;
  error_message: string | null;
  executed_by: string;
  executed_at: string;
  execution_time_ms: number;
  request_data: Record<string, any>;
}

const ValidationHistoryPanel: React.FC = () => {
  const classes = useStyles();
  const [history, setHistory] = useState<AuditRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filterBP, setFilterBP] = useState('');
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<AuditRecord | null>(null);

  const getTenantContext = () => {
    const tenantId = localStorage.getItem('selected_tenant')
      ? JSON.parse(localStorage.getItem('selected_tenant') || '{}').id
      : null;
    const datasourceId = localStorage.getItem('selected_datasource')
      ? JSON.parse(localStorage.getItem('selected_datasource') || '{}').id
      : null;
    return { tenantId, datasourceId };
  };

  const fetchHistory = useCallback(async () => {
    try {
      setLoading(true);
      const { tenantId, datasourceId } = getTenantContext();

      if (!tenantId || !datasourceId) {
        setError('Please select a tenant and datasource first');
        return;
      }

      const params = new URLSearchParams();
      params.append('limit', '100');
      if (filterBP) params.append('bp_name', filterBP);

      const response = await fetch(
        `/api/validations/history?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}&${params.toString()}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch history: ${response.statusText}`);
      }

      const data = await response.json();
      setHistory(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch history');
    } finally {
      setLoading(false);
    }
  }, [filterBP]);

  useEffect(() => {
    fetchHistory();
  }, [fetchHistory]);

  const handleViewDetails = (record: AuditRecord) => {
    setSelectedRecord(record);
    setDetailsOpen(true);
  };

  const calculateStats = () => {
    if (history.length === 0) return { total: 0, passed: 0, failed: 0, successRate: 0 };
    const passed = history.filter((r) => r.passed).length;
    const failed = history.length - passed;
    return {
      total: history.length,
      passed,
      failed,
      successRate: ((passed / history.length) * 100).toFixed(1),
    };
  };

  const stats = calculateStats();

  return (
    <Box className={classes.container}>
      <Typography variant="h6" sx={{ mb: 3 }}>
        Validation Audit History
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={3}>
          <Card>
            <CardContent sx={{ textAlign: 'center' }}>
              <Typography color="textSecondary" gutterBottom>
                Total Validations
              </Typography>
              <Typography variant="h5">{stats.total}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Card>
            <CardContent sx={{ textAlign: 'center' }}>
              <Typography color="textSecondary" gutterBottom>
                Passed
              </Typography>
              <Typography variant="h5" sx={{ color: '#4caf50' }}>
                {stats.passed}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Card>
            <CardContent sx={{ textAlign: 'center' }}>
              <Typography color="textSecondary" gutterBottom>
                Failed
              </Typography>
              <Typography variant="h5" sx={{ color: '#f44336' }}>
                {stats.failed}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Card>
            <CardContent sx={{ textAlign: 'center' }}>
              <Typography color="textSecondary" gutterBottom>
                Success Rate
              </Typography>
              <Typography variant="h5" sx={{ color: '#1976d2' }}>
                {stats.successRate}%
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Grid container spacing={2}>
            <Grid item xs={12} sm={9}>
              <TextField
                fullWidth
                label="Filter by Business Process"
                size="small"
                value={filterBP}
                onChange={(e) => setFilterBP(e.target.value)}
                placeholder="e.g., ChangeMaritalStatus"
              />
            </Grid>
            <Grid item xs={12} sm={3}>
              <Button
                fullWidth
                variant="outlined"
                startIcon={<RefreshIcon />}
                onClick={fetchHistory}
                disabled={loading}
                sx={{ height: '40px' }}
              >
                Refresh
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      <TableContainer component={Paper} className={classes.auditTable}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
              <TableCell>BP / Step / Rule</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Executed By</TableCell>
              <TableCell>Time (ms)</TableCell>
              <TableCell>Date/Time</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {history.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} sx={{ textAlign: 'center', py: 4 }}>
                  {loading ? <CircularProgress size={24} /> : 'No audit records found'}
                </TableCell>
              </TableRow>
            ) : (
              history.map((record) => (
                <TableRow key={record.id}>
                  <TableCell sx={{ fontSize: '0.85rem' }}>
                    <Box>
                      <Typography variant="caption" display="block">
                        {record.bp_name} / {record.step_name}
                      </Typography>
                      <Typography variant="caption" color="textSecondary">
                        {record.rule_name}
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={record.passed ? 'Passed' : 'Failed'}
                      color={record.passed ? 'success' : 'error'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell sx={{ fontSize: '0.85rem' }}>{record.executed_by}</TableCell>
                  <TableCell>{record.execution_time_ms}</TableCell>
                  <TableCell sx={{ fontSize: '0.75rem' }}>
                    {new Date(record.executed_at).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <Button size="small" onClick={() => handleViewDetails(record)}>
                      Details
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={detailsOpen} onClose={() => setDetailsOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Audit Record Details</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {selectedRecord && (
            <Box>
              <Grid container spacing={2} sx={{ mb: 2 }}>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Business Process
                  </Typography>
                  <Typography variant="body2">{selectedRecord.bp_name}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Step
                  </Typography>
                  <Typography variant="body2">{selectedRecord.step_name}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Rule
                  </Typography>
                  <Typography variant="body2">{selectedRecord.rule_name}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Status
                  </Typography>
                  <Chip
                    label={selectedRecord.passed ? 'Passed' : 'Failed'}
                    color={selectedRecord.passed ? 'success' : 'error'}
                    size="small"
                  />
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Executed By
                  </Typography>
                  <Typography variant="body2">{selectedRecord.executed_by}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Execution Time
                  </Typography>
                  <Typography variant="body2">{selectedRecord.execution_time_ms}ms</Typography>
                </Grid>
              </Grid>

              {selectedRecord.error_message && (
                <Box sx={{ mb: 2, p: 1, backgroundColor: '#ffebee', borderRadius: 1 }}>
                  <Typography variant="caption" color="error">
                    Error
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#f44336' }}>
                    {selectedRecord.error_message}
                  </Typography>
                </Box>
              )}

              {selectedRecord.request_data && Object.keys(selectedRecord.request_data).length > 0 && (
                <Box sx={{ p: 1, backgroundColor: '#f5f5f5', borderRadius: 1 }}>
                  <Typography variant="caption" color="textSecondary">
                    Request Data
                  </Typography>
                  <pre className={classes.preformatted}>
                    {JSON.stringify(selectedRecord.request_data, null, 2)}
                  </pre>
                </Box>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailsOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ValidationHistoryPanel;
