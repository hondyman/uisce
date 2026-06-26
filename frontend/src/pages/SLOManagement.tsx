import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Stack,
  Alert,
  CircularProgress,
  Slider,
  FormHelperText,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  CheckCircle as HealthyIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
  Visibility as ViewIcon,
} from '@mui/icons-material';

interface SLO {
  id: string;
  name: string;
  description: string;
  target: number;
  window: string;
  metric_query: string;
  created_at: string;
  alert_rules?: AlertRule[];
}

interface AlertRule {
  id: string;
  name: string;
  condition: string;
  severity: 'critical' | 'warning' | 'info';
  channels: string[];
  enabled: boolean;
}

interface SLOStatus {
  slo_id: string;
  slo_name: string;
  current_value: number;
  target: number;
  budget_remaining: number;
  status: 'healthy' | 'degraded' | 'breached';
}

const statusColors = {
  healthy: '#4caf50',
  degraded: '#ff9800',
  breached: '#f44336',
};

const metricQueryOptions = [
  { value: 'availability', label: 'Availability (Success Rate)' },
  { value: 'error_rate', label: 'Error Rate' },
  { value: 'latency_p95', label: 'P95 Latency' },
  { value: 'latency_p99', label: 'P99 Latency' },
];

const windowOptions = [
  { value: '24h', label: '24 Hours' },
  { value: '7d', label: '7 Days' },
  { value: '30d', label: '30 Days' },
  { value: '90d', label: '90 Days' },
];

export const SLOManagement: React.FC = () => {
  const [slos, setSLOs] = useState<SLO[]>([]);
  const [statuses, setStatuses] = useState<Map<string, SLOStatus>>(new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingSLO, setEditingSLO] = useState<SLO | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    target: 99.9,
    window: '7d',
    metric_query: 'availability',
  });

  const fetchSLOs = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/observability/slos');
      if (!response.ok) throw new Error('Failed to fetch SLOs');
      const result = await response.json();
      setSLOs(result || []);
      
      // Fetch status for each SLO
      const statusMap = new Map<string, SLOStatus>();
      for (const slo of result || []) {
        const statusResp = await fetch(`/api/observability/slos/${slo.id}/status`);
        if (statusResp.ok) {
          const status = await statusResp.json();
          statusMap.set(slo.id, status);
        }
      }
      setStatuses(statusMap);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSLOs();
  }, []);

  const handleOpenDialog = (slo?: SLO) => {
    if (slo) {
      setEditingSLO(slo);
      setFormData({
        name: slo.name,
        description: slo.description,
        target: slo.target,
        window: slo.window,
        metric_query: slo.metric_query,
      });
    } else {
      setEditingSLO(null);
      setFormData({
        name: '',
        description: '',
        target: 99.9,
        window: '7d',
        metric_query: 'availability',
      });
    }
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingSLO(null);
  };

  const handleSubmit = async () => {
    try {
      const method = editingSLO ? 'PUT' : 'POST';
      const url = editingSLO 
        ? `/api/observability/slos/${editingSLO.id}`
        : '/api/observability/slos';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });

      if (!response.ok) throw new Error('Failed to save SLO');

      handleCloseDialog();
      fetchSLOs();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this SLO?')) return;

    try {
      const response = await fetch(`/api/observability/slos/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) throw new Error('Failed to delete SLO');
      fetchSLOs();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    }
  };

  const getStatusIcon = (status?: SLOStatus) => {
    if (!status) return null;
    switch (status.status) {
      case 'healthy':
        return <HealthyIcon sx={{ color: statusColors.healthy }} />;
      case 'degraded':
        return <WarningIcon sx={{ color: statusColors.degraded }} />;
      case 'breached':
        return <ErrorIcon sx={{ color: statusColors.breached }} />;
      default:
        return null;
    }
  };

  if (loading && slos.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            SLO Management
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Define and track Service Level Objectives
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => handleOpenDialog()}
          sx={{
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          }}
        >
          Create SLO
        </Button>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* SLO Table */}
      <Paper sx={{ overflow: 'hidden' }}>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow sx={{ backgroundColor: 'grey.50' }}>
                <TableCell sx={{ fontWeight: 600 }}>Status</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Name</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Target</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Window</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Current</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Budget</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Alert Rules</TableCell>
                <TableCell sx={{ fontWeight: 600 }} align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {slos.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">
                      No SLOs defined yet. Create your first SLO to start tracking.
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                slos.map((slo) => {
                  const status = statuses.get(slo.id);
                  return (
                    <TableRow key={slo.id} hover>
                      <TableCell>
                        {getStatusIcon(status)}
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={{ fontWeight: 500 }}>
                          {slo.name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {slo.description}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2">{slo.target}%</Typography>
                      </TableCell>
                      <TableCell>
                        <Chip size="small" label={slo.window} variant="outlined" />
                      </TableCell>
                      <TableCell>
                        <Typography
                          variant="body2"
                          sx={{
                            fontWeight: 600,
                            color: status ? statusColors[status.status] : 'text.primary',
                          }}
                        >
                          {status?.current_value.toFixed(2)}%
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2">
                          {status?.budget_remaining.toFixed(2)}%
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip
                          size="small"
                          label={`${slo.alert_rules?.length || 0} rules`}
                          color={slo.alert_rules?.length ? 'primary' : 'default'}
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell align="right">
                        <Tooltip title="View Details">
                          <IconButton size="small">
                            <ViewIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Edit">
                          <IconButton size="small" onClick={() => handleOpenDialog(slo)}>
                            <EditIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Delete">
                          <IconButton size="small" color="error" onClick={() => handleDelete(slo.id)}>
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {/* Create/Edit SLO Dialog */}
      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>
          {editingSLO ? 'Edit SLO' : 'Create New SLO'}
        </DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <TextField
              label="Name"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              fullWidth
              required
              placeholder="e.g., Query Availability"
            />
            <TextField
              label="Description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              fullWidth
              multiline
              rows={2}
              placeholder="Describe what this SLO measures"
            />
            <FormControl fullWidth>
              <InputLabel>Metric</InputLabel>
              <Select
                value={formData.metric_query}
                label="Metric"
                onChange={(e) => setFormData({ ...formData, metric_query: e.target.value })}
              >
                {metricQueryOptions.map((opt) => (
                  <MenuItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>The metric to track for this SLO</FormHelperText>
            </FormControl>
            <Box>
              <Typography gutterBottom>Target: {formData.target}%</Typography>
              <Slider
                value={formData.target}
                onChange={(_, value) => setFormData({ ...formData, target: value as number })}
                min={90}
                max={99.99}
                step={0.1}
                marks={[
                  { value: 99, label: '99%' },
                  { value: 99.9, label: '99.9%' },
                  { value: 99.99, label: '99.99%' },
                ]}
              />
              <FormHelperText>
                Error Budget: {(100 - formData.target).toFixed(2)}%
              </FormHelperText>
            </Box>
            <FormControl fullWidth>
              <InputLabel>Window</InputLabel>
              <Select
                value={formData.window}
                label="Window"
                onChange={(e) => setFormData({ ...formData, window: e.target.value })}
              >
                {windowOptions.map((opt) => (
                  <MenuItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </MenuItem>
                ))}
              </Select>
              <FormHelperText>Time window for SLO calculation</FormHelperText>
            </FormControl>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={!formData.name}
          >
            {editingSLO ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default SLOManagement;
