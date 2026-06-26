/**
 * Report History Component
 * 
 * Displays execution history for reports with filtering,
 * status tracking, and ability to re-run or download outputs.
 */

import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  Chip,
  IconButton,
  Button,
  TextField,
  InputAdornment,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  CircularProgress,
  Alert,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  Search,
  Download,
  Refresh,
  Visibility,
  CheckCircle,
  Error as ErrorIcon,
  HourglassEmpty,
  Schedule,
} from '@mui/icons-material';
import { useReportInstances, useDownloadReport, useRenderReport } from '../../hooks/useSemanticReporting';
import { ReportInstance, RenderReportRequest } from '../../api/semanticReporting';
import { useTenant } from '../../contexts/TenantContext';
import { format, formatDistanceToNow } from 'date-fns';

interface ReportHistoryProps {
  reportId?: string; // Optional filter by specific report
  onViewReport?: (instanceId: string) => void;
}

const ReportHistory: React.FC<ReportHistoryProps> = ({ reportId, onViewReport: _onViewReport }) => {
  const { isSelected } = useTenant();
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [selectedInstance, setSelectedInstance] = useState<ReportInstance | null>(null);
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false);

  // Fetch instances
  const { data: instances, isLoading, error, refetch } = useReportInstances(100);
  const downloadMutation = useDownloadReport();
  const renderMutation = useRenderReport();

  // Filter instances
  const filteredInstances = React.useMemo(() => {
    if (!instances) return [];

    let filtered = instances;

    // Filter by report ID if specified
    if (reportId) {
      filtered = filtered.filter((i) => i.report_definition_id === reportId);
    }

    // Filter by status
    if (statusFilter !== 'all') {
      filtered = filtered.filter((i) => i.status === statusFilter);
    }

    // Filter by search term
    if (searchTerm) {
      const term = searchTerm.toLowerCase();
      filtered = filtered.filter(
        (i) =>
          i.context_name?.toLowerCase().includes(term) ||
          i.id.toLowerCase().includes(term)
      );
    }

    return filtered;
  }, [instances, reportId, statusFilter, searchTerm]);

  // Paginated instances
  const paginatedInstances = React.useMemo(() => {
    const start = page * rowsPerPage;
    return filteredInstances.slice(start, start + rowsPerPage);
  }, [filteredInstances, page, rowsPerPage]);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle color="success" fontSize="small" />;
      case 'failed':
        return <ErrorIcon color="error" fontSize="small" />;
      case 'generating':
        return <CircularProgress size={16} />;
      case 'pending':
        return <HourglassEmpty color="action" fontSize="small" />;
      default:
        return <Schedule color="action" fontSize="small" />;
    }
  };

  const getStatusColor = (status: string): 'success' | 'error' | 'warning' | 'default' => {
    switch (status) {
      case 'completed':
        return 'success';
      case 'failed':
        return 'error';
      case 'generating':
      case 'pending':
        return 'warning';
      default:
        return 'default';
    }
  };

  const handleDownload = (instanceId: string) => {
    downloadMutation.mutate(instanceId);
  };

  const handleRerun = async (instance: ReportInstance) => {
    const request: RenderReportRequest = {
      report_definition_id: instance.report_definition_id,
      report_extension_id: instance.report_extension_id,
      output_format: instance.output_format as 'pdf' | 'html' | 'excel',
      context_type: instance.context_type,
      context_id: instance.context_id,
      context_name: instance.context_name,
      parameters: instance.parameters,
    };

    try {
      await renderMutation.mutateAsync(request);
      refetch();
    } catch (err) {
      console.error('Failed to re-run report:', err);
    }
  };

  const handleViewDetails = (instance: ReportInstance) => {
    setSelectedInstance(instance);
    setDetailsDialogOpen(true);
  };

  if (!isSelected) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="warning">
          Please select a tenant and datasource to view report history.
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h5" component="h1">
          Report Execution History
        </Typography>
        <Button
          variant="outlined"
          startIcon={<Refresh />}
          onClick={() => refetch()}
          disabled={isLoading}
        >
          Refresh
        </Button>
      </Box>

      {/* Filters */}
      <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
        <TextField
          placeholder="Search by context or ID..."
          size="small"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <Search />
              </InputAdornment>
            ),
          }}
          sx={{ width: 300 }}
        />

        <FormControl size="small" sx={{ width: 150 }}>
          <InputLabel>Status</InputLabel>
          <Select
            value={statusFilter}
            label="Status"
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <MenuItem value="all">All Status</MenuItem>
            <MenuItem value="completed">Completed</MenuItem>
            <MenuItem value="failed">Failed</MenuItem>
            <MenuItem value="generating">Generating</MenuItem>
            <MenuItem value="pending">Pending</MenuItem>
          </Select>
        </FormControl>
      </Box>

      {/* Loading / Error */}
      {isLoading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      )}

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          Failed to load report history: {(error as Error).message}
        </Alert>
      )}

      {/* Table */}
      {!isLoading && (
        <Paper>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Status</TableCell>
                  <TableCell>Context</TableCell>
                  <TableCell>Format</TableCell>
                  <TableCell>Requested</TableCell>
                  <TableCell>Duration</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {paginatedInstances.map((instance) => (
                  <TableRow key={instance.id} hover>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        {getStatusIcon(instance.status)}
                        <Chip
                          label={instance.status}
                          size="small"
                          color={getStatusColor(instance.status)}
                          variant="outlined"
                        />
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {instance.context_name || instance.context_type || '-'}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {instance.id.slice(0, 8)}...
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip label={instance.output_format.toUpperCase()} size="small" />
                    </TableCell>
                    <TableCell>
                      <Tooltip title={format(new Date(instance.requested_at), 'PPpp')}>
                        <Typography variant="body2">
                          {formatDistanceToNow(new Date(instance.requested_at), { addSuffix: true })}
                        </Typography>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      {instance.generation_time_ms ? (
                        <Typography variant="body2">
                          {instance.generation_time_ms}ms
                        </Typography>
                      ) : (
                        '-'
                      )}
                    </TableCell>
                    <TableCell align="right">
                      <Tooltip title="View Details">
                        <IconButton
                          size="small"
                          onClick={() => handleViewDetails(instance)}
                        >
                          <Visibility />
                        </IconButton>
                      </Tooltip>
                      {instance.status === 'completed' && (
                        <Tooltip title="Download">
                          <IconButton
                            size="small"
                            onClick={() => handleDownload(instance.id)}
                            disabled={downloadMutation.isPending}
                          >
                            <Download />
                          </IconButton>
                        </Tooltip>
                      )}
                      <Tooltip title="Re-run">
                        <IconButton
                          size="small"
                          onClick={() => handleRerun(instance)}
                          disabled={renderMutation.isPending}
                        >
                          <Refresh />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}

                {paginatedInstances.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                      <Typography color="text.secondary">
                        No report executions found
                      </Typography>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>

          <TablePagination
            component="div"
            count={filteredInstances.length}
            page={page}
            onPageChange={(_, newPage) => setPage(newPage)}
            rowsPerPage={rowsPerPage}
            onRowsPerPageChange={(e) => {
              setRowsPerPage(parseInt(e.target.value, 10));
              setPage(0);
            }}
            rowsPerPageOptions={[10, 25, 50, 100]}
          />
        </Paper>
      )}

      {/* Details Dialog */}
      <Dialog
        open={detailsDialogOpen}
        onClose={() => setDetailsDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Execution Details</DialogTitle>
        <DialogContent>
          {selectedInstance && (
            <Box>
              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="text.secondary">
                  Instance ID
                </Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                  {selectedInstance.id}
                </Typography>
              </Box>

              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="text.secondary">
                  Status
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  {getStatusIcon(selectedInstance.status)}
                  <Chip
                    label={selectedInstance.status}
                    size="small"
                    color={getStatusColor(selectedInstance.status)}
                  />
                </Box>
              </Box>

              {selectedInstance.error_message && (
                <Alert severity="error" sx={{ mb: 2 }}>
                  {selectedInstance.error_message}
                </Alert>
              )}

              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="text.secondary">
                  Report Definition ID
                </Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                  {selectedInstance.report_definition_id}
                </Typography>
              </Box>

              {selectedInstance.report_extension_id && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="caption" color="text.secondary">
                    Extension ID
                  </Typography>
                  <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                    {selectedInstance.report_extension_id}
                  </Typography>
                </Box>
              )}

              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="text.secondary">
                  Output Format
                </Typography>
                <Typography variant="body2">
                  {selectedInstance.output_format.toUpperCase()}
                </Typography>
              </Box>

              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="text.secondary">
                  Requested At
                </Typography>
                <Typography variant="body2">
                  {format(new Date(selectedInstance.requested_at), 'PPpp')}
                </Typography>
              </Box>

              {selectedInstance.completed_at && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="caption" color="text.secondary">
                    Completed At
                  </Typography>
                  <Typography variant="body2">
                    {format(new Date(selectedInstance.completed_at), 'PPpp')}
                  </Typography>
                </Box>
              )}

              {selectedInstance.generation_time_ms && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="caption" color="text.secondary">
                    Generation Time
                  </Typography>
                  <Typography variant="body2">
                    {selectedInstance.generation_time_ms}ms
                  </Typography>
                </Box>
              )}

              {selectedInstance.parameters && Object.keys(selectedInstance.parameters).length > 0 && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="caption" color="text.secondary">
                    Parameters
                  </Typography>
                  <Paper variant="outlined" sx={{ p: 1, mt: 0.5 }}>
                    <Box component="pre" sx={{ m: 0, fontSize: 12, overflow: 'auto' }}>
                      {JSON.stringify(selectedInstance.parameters, null, 2)}
                    </Box>
                  </Paper>
                </Box>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          {selectedInstance?.status === 'completed' && (
            <Button
              startIcon={<Download />}
              onClick={() => {
                handleDownload(selectedInstance.id);
                setDetailsDialogOpen(false);
              }}
            >
              Download
            </Button>
          )}
          <Button onClick={() => setDetailsDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ReportHistory;
