import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
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
  Paper,
  Chip,
  Grid,
  Alert,
  CircularProgress,
  Typography,
} from '@mui/material';
import DownloadIcon from '@mui/icons-material/Download';
import RefreshIcon from '@mui/icons-material/Refresh';
import { API_BASE_URL, DEFAULT_TENANT_ID, EXPORT_CONFIG, getApiHeaders } from '../api/config';

interface Export {
  id: string;
  job_id: string;
  export_format: 'csv' | 'json' | 'parquet';
  status: 'queued' | 'processing' | 'completed' | 'failed';
  file_size: number;
  record_count: number;
  created_at: string;
  completed_at?: string;
  expires_at?: string;
  presigned_url?: string;
}

export const EDM_ExportsManager: React.FC = () => {
  const [exports, setExports] = useState<Export[]>([]);
  const [openDialog, setOpenDialog] = useState(false);
  const [jobId, setJobId] = useState('');
  const [format, setFormat] = useState<'csv' | 'json' | 'parquet'>(EXPORT_CONFIG.DEFAULT_FORMAT);
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch exports on mount
  useEffect(() => {
    fetchExports();
  }, []);

  const fetchExports = async () => {
    try {
      setInitialLoading(true);
      setError(null);
      // Note: This would need a job ID to list exports for that job
      // For now, we'll just show the UI ready to create exports
      setExports([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch exports');
    } finally {
      setInitialLoading(false);
    }
  };

  const handleOpenDialog = () => {
    setOpenDialog(true);
  };

  const handleCloseDialog = () => {
    setOpenDialog(false);
    setJobId('');
    setFormat('csv');
  };

  const handleCreateExport = async () => {
    if (!jobId) {
      setError('Job ID is required');
      return;
    }
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`${API_BASE_URL}/jobs/${jobId}/exports`, {
        method: 'POST',
        headers: getApiHeaders(DEFAULT_TENANT_ID),
        body: JSON.stringify({
          export_format: format,
          filter_criteria: {},
          include_errors: false,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to create export: ${response.statusText}`);
      }

      const newExport: Export = await response.json();
      setExports([newExport, ...exports]);
      handleCloseDialog();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create export');
    } finally {
      setLoading(false);
    }
  };

  const handleDownload = async (exportId: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/exports/${exportId}/download`, {
        headers: getApiHeaders(DEFAULT_TENANT_ID),
      });

      if (!response.ok) {
        throw new Error('Failed to download export');
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `export-${exportId}.${format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to download export');
    }
  };

  const getStatusColor = (status: string): 'default' | 'primary' | 'success' | 'error' | 'warning' => {
    switch (status) {
      case 'completed': return 'success';
      case 'processing': return 'warning';
      case 'failed': return 'error';
      default: return 'default';
    }
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  return (
    <Box sx={{ p: 3 }}>
      <Card>
        <CardHeader
          title="Job Exports"
          subheader="Export and download job results in multiple formats"
          action={
            <Button
              variant="contained"
              startIcon={<RefreshIcon />}
              onClick={() => console.log('Refresh')}
            >
              Refresh
            </Button>
          }
        />
        <CardContent>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          
          {initialLoading && (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
              <CircularProgress />
            </Box>
          )}

          {!initialLoading && (
            <>
              <Grid container spacing={2} sx={{ mb: 3 }}>
                <Grid item xs={12}>
                  <Button variant="contained" color="primary" onClick={handleOpenDialog}>
                    Create New Export
                  </Button>
                  <Button
                    variant="outlined"
                    startIcon={<RefreshIcon />}
                    onClick={fetchExports}
                    sx={{ ml: 1 }}
                  >
                    Refresh
                  </Button>
                </Grid>
              </Grid>

              <TableContainer component={Paper}>
                <Table>
                  <TableHead sx={{ bgcolor: '#f3f4f6' }}>
                    <TableRow>
                      <TableCell><strong>Export ID</strong></TableCell>
                      <TableCell><strong>Job ID</strong></TableCell>
                      <TableCell><strong>Format</strong></TableCell>
                      <TableCell><strong>Status</strong></TableCell>
                      <TableCell align="right"><strong>Records</strong></TableCell>
                      <TableCell align="right"><strong>File Size</strong></TableCell>
                      <TableCell><strong>Created</strong></TableCell>
                      <TableCell align="center"><strong>Action</strong></TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {exports.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={8} align="center" sx={{ py: 3 }}>
                          <Typography color="textSecondary">No exports yet</Typography>
                        </TableCell>
                      </TableRow>
                    ) : (
                      exports.map((exp) => (
                        <TableRow key={exp.id} hover>
                          <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}>
                            {exp.id}
                          </TableCell>
                          <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}>
                            {exp.job_id}
                          </TableCell>
                          <TableCell>
                            <Chip label={exp.export_format.toUpperCase()} size="small" variant="outlined" />
                          </TableCell>
                          <TableCell>
                            <Chip label={exp.status} size="small" color={getStatusColor(exp.status)} />
                          </TableCell>
                          <TableCell align="right">{exp.record_count.toLocaleString()}</TableCell>
                          <TableCell align="right">{formatFileSize(exp.file_size)}</TableCell>
                          <TableCell sx={{ fontSize: '0.85rem' }}>
                            {new Date(exp.created_at).toLocaleString()}
                          </TableCell>
                          <TableCell align="center">
                            {exp.status === 'completed' && (
                              <Button
                                size="small"
                                startIcon={<DownloadIcon />}
                                onClick={() => handleDownload(exp.id)}
                              >
                                Download
                              </Button>
                            )}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>

              <Alert severity="info" sx={{ mt: 2 }}>
                💡 Exports are retained for 7 days by default. Presigned URLs expire after 24 hours.
              </Alert>
            </>
          )}
        </CardContent>
      </Card>

      {/* Create Export Dialog */}
      <Dialog open={openDialog} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Export</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <TextField
            label="Job ID"
            fullWidth
            value={jobId}
            onChange={(e) => setJobId(e.target.value)}
            placeholder="Enter Job ID to export results"
            sx={{ mb: 2 }}
          />
          <FormControl fullWidth>
            <InputLabel>Export Format</InputLabel>
            <Select value={format} onChange={(e) => setFormat(e.target.value as any)} label="Export Format">
              <MenuItem value="csv">CSV - Comma-separated values</MenuItem>
              <MenuItem value="json">JSON - JavaScript Object Notation</MenuItem>
              <MenuItem value="parquet">Parquet - Columnar format</MenuItem>
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button
            onClick={handleCreateExport}
            variant="contained"
            disabled={!jobId || loading}
          >
            {loading ? <CircularProgress size={24} /> : 'Create Export'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
