import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  LinearProgress as _LinearProgress,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField as _TextField,
  Typography,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import _ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import _ExpandLessIcon from '@mui/icons-material/ExpandLess';
import FieldAutocomplete from '../common/FieldAutocomplete';

const useStyles = makeStyles({
  container: {
    padding: '20px',
  },
  expandableRow: {
    cursor: 'pointer',
    backgroundColor: '#f5f5f5',
  },
  expandedRow: {
    backgroundColor: '#e3f2fd',
  },
  statusSuccess: {
    color: '#4caf50',
  },
  statusFailed: {
    color: '#f44336',
  },
  statusWarning: {
    color: '#ff9800',
  },
});

interface ValidationResultRecord {
  id: string;
  bp_name: string;
  step_name: string;
  passed: boolean;
  error_count: number;
  warning_count: number;
  execution_time_ms: number;
  executed_at: string;
  user_id: string;
  errors: string[];
  warnings: string[];
  actions: string[];
}

const ValidationResultsPanel: React.FC = () => {
  const classes = useStyles();
  const [results, setResults] = useState<ValidationResultRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [selectedResult, setSelectedResult] = useState<ValidationResultRecord | null>(null);
  const [filterBP, setFilterBP] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'passed' | 'failed' | 'warning'>('all');

  const getTenantContext = () => {
    const tenantId = localStorage.getItem('selected_tenant')
      ? JSON.parse(localStorage.getItem('selected_tenant') || '{}').id
      : null;
    const datasourceId = localStorage.getItem('selected_datasource')
      ? JSON.parse(localStorage.getItem('selected_datasource') || '{}').id
      : null;
    return { tenantId, datasourceId };
  };

  const fetchResults = useCallback(async () => {
    try {
      setLoading(true);
      const { tenantId, datasourceId } = getTenantContext();

      if (!tenantId || !datasourceId) {
        setError('Please select a tenant and datasource first');
        return;
      }

      const params = new URLSearchParams();
      if (filterBP) params.append('bp_name', filterBP);
      if (filterStatus !== 'all') {
        if (filterStatus === 'passed') {
          params.append('passed', 'true');
        } else if (filterStatus === 'failed') {
          params.append('passed', 'false');
        }
      }

      const response = await fetch(
        `/api/validations/results?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}&${params.toString()}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch results: ${response.statusText}`);
      }

      const data = await response.json();
      setResults(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch results');
    } finally {
      setLoading(false);
    }
  }, [filterBP, filterStatus]);

  useEffect(() => {
    fetchResults();
  }, [fetchResults]);

  const getStatusColor = (result: ValidationResultRecord) => {
    if (result.passed) return 'success';
    if (result.error_count > 0) return 'error';
    return 'warning';
  };

  const getStatusLabel = (result: ValidationResultRecord) => {
    if (result.passed) return 'Passed';
    if (result.error_count > 0) return 'Failed';
    return 'Warning';
  };

  const handleViewDetails = (result: ValidationResultRecord) => {
    setSelectedResult(result);
    setDetailsOpen(true);
  };

  const filteredResults = results.filter((r) => {
    if (filterBP && !r.bp_name.includes(filterBP)) return false;
    if (filterStatus === 'passed' && !r.passed) return false;
    if (filterStatus === 'failed' && (r.passed || r.error_count === 0)) return false;
    if (filterStatus === 'warning' && (r.passed || r.error_count > 0)) return false;
    return true;
  });

  return (
    <Box className={classes.container}>
      <Typography variant="h6" sx={{ mb: 3 }}>
        Validation Results
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6}>
              <FieldAutocomplete
                value={filterBP}
                onChange={(value) => setFilterBP(value)}
                entityName="BusinessProcess"
                label="Filter by Business Process"
                placeholder="Search for a business process..."
                required={false}
                showRecentFields={true}
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth size="small">
                <InputLabel id="filter-status-select-label">Filter by Status</InputLabel>
                <Select
                  labelId="filter-status-select-label"
                  id="filter-status-select"
                  value={filterStatus}
                  label="Filter by Status"
                  onChange={(e) => {
                    const v = String((e.target as HTMLSelectElement).value);
                    if (v === 'passed' || v === 'failed' || v === 'warning') {
                      setFilterStatus(v as 'passed' | 'failed' | 'warning');
                    } else {
                      setFilterStatus('all');
                    }
                  }}
                  aria-label="Filter by Status"
                >
                  <MenuItem value="all">All</MenuItem>
                  <MenuItem value="passed">Passed</MenuItem>
                  <MenuItem value="failed">Failed</MenuItem>
                  <MenuItem value="warning">Warning</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12}>
              <Button variant="outlined" onClick={fetchResults} disabled={loading}>
                {loading ? <CircularProgress size={20} sx={{ mr: 1 }} /> : null}
                Refresh Results
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
              <TableCell>BP / Step</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Errors</TableCell>
              <TableCell>Warnings</TableCell>
              <TableCell>Time (ms)</TableCell>
              <TableCell>Executed</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredResults.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} sx={{ textAlign: 'center', py: 4 }}>
                  {loading ? <CircularProgress size={24} /> : 'No results found'}
                </TableCell>
              </TableRow>
            ) : (
              filteredResults.map((result) => (
                <TableRow
                  key={result.id}
                  className={classes.expandableRow}
                  onClick={() => setExpandedId(expandedId === result.id ? null : result.id)}
                >
                  <TableCell>
                    {result.bp_name} / {result.step_name}
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={getStatusLabel(result)}
                      color={getStatusColor(result) as 'success' | 'error' | 'warning'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell sx={{ color: result.error_count > 0 ? '#f44336' : 'inherit' }}>
                    {result.error_count}
                  </TableCell>
                  <TableCell sx={{ color: result.warning_count > 0 ? '#ff9800' : 'inherit' }}>
                    {result.warning_count}
                  </TableCell>
                  <TableCell>{result.execution_time_ms}</TableCell>
                  <TableCell sx={{ fontSize: '0.85rem' }}>
                    {new Date(result.executed_at).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <Button
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleViewDetails(result);
                      }}
                    >
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
        <DialogTitle>Validation Result Details</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {selectedResult && (
            <Box>
              <Grid container spacing={2} sx={{ mb: 2 }}>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Business Process
                  </Typography>
                  <Typography variant="body2">{selectedResult.bp_name}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Step
                  </Typography>
                  <Typography variant="body2">{selectedResult.step_name}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Status
                  </Typography>
                  <Chip
                    label={getStatusLabel(selectedResult)}
                    color={getStatusColor(selectedResult) as 'success' | 'error' | 'warning'}
                    size="small"
                  />
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="textSecondary">
                    Execution Time
                  </Typography>
                  <Typography variant="body2">{selectedResult.execution_time_ms}ms</Typography>
                </Grid>
              </Grid>

              {selectedResult.errors.length > 0 && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="subtitle2" sx={{ color: '#f44336', mb: 1 }}>
                    Errors ({selectedResult.errors.length})
                  </Typography>
                  {selectedResult.errors.map((err, idx) => (
                    <Typography key={idx} variant="body2" sx={{ ml: 2, mb: 1 }}>
                      {idx + 1}. {err}
                    </Typography>
                  ))}
                </Box>
              )}

              {selectedResult.warnings.length > 0 && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="subtitle2" sx={{ color: '#ff9800', mb: 1 }}>
                    Warnings ({selectedResult.warnings.length})
                  </Typography>
                  {selectedResult.warnings.map((warn, idx) => (
                    <Typography key={idx} variant="body2" sx={{ ml: 2, mb: 1 }}>
                      {idx + 1}. {warn}
                    </Typography>
                  ))}
                </Box>
              )}

              {selectedResult.actions.length > 0 && (
                <Box>
                  <Typography variant="subtitle2" sx={{ mb: 1 }}>
                    Actions to Take
                  </Typography>
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                    {selectedResult.actions.map((action, idx) => (
                      <Chip key={idx} label={action} size="small" variant="outlined" />
                    ))}
                  </Box>
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

export default ValidationResultsPanel;
