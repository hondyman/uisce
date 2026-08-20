import { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Button,
  TextField,
  InputAdornment,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  CircularProgress,
  Alert,
  Chip,
  FormControlLabel,
  Switch,
} from '@mui/material';
import {
  Search as SearchIcon,
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  History as HistoryIcon,
  Storage as StorageIcon,
  Speed as SpeedIcon,
} from '@mui/icons-material';
import { useNotification } from '../../../../hooks/useNotification';
import { useTenant } from '../../../../contexts/TenantContext';
import apiClient from '../../../../utils/apiClient';

interface RecordsCrudTabProps {
  businessObject: any;
}

export function RecordsCrudTab({ businessObject }: RecordsCrudTabProps) {
  const notification = useNotification();
  const { tenant } = useTenant();
  const tenantId = tenant?.id || '';

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [rows, setRows] = useState<any[]>([]);
  const [columns, setColumns] = useState<string[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [search, setSearch] = useState('');
  const [execTimeMs, setExecTimeMs] = useState<number | null>(null);
  const [driverTable, setDriverTable] = useState('');

  // Time Travel
  const [isTimeTravelEnabled, setIsTimeTravelEnabled] = useState(false);
  const [asOfDate, setAsOfDate] = useState<string>('');

  // CRUD Modals
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<any>(null);
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [submitting, setSubmitting] = useState(false);

  // Available Fields
  const fields = [
    ...(businessObject?.coreFields || []),
    ...(businessObject?.customFields || []),
    ...(businessObject?.config?.fields || []),
  ];

  const fetchRecords = useCallback(async () => {
    if (!businessObject?.id) return;
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({
        page: String(page + 1),
        limit: String(rowsPerPage),
      });
      if (search.trim()) params.append('search', search.trim());
      if (isTimeTravelEnabled && asOfDate) {
        params.append('asOfValidTime', new Date(asOfDate).toISOString());
      }

      const res = await apiClient<any>(
        `/business-objects/${encodeURIComponent(businessObject.id)}/data?${params.toString()}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
          },
        }
      );

      setRows(res.rows || []);
      setColumns(res.columns || []);
      setTotal(res.total || 0);
      setExecTimeMs(res.executionTimeMs ?? null);
      setDriverTable(res.driverTable || '');
    } catch (err: any) {
      setError(err?.message || 'Failed to load records from datasource');
    } finally {
      setLoading(false);
    }
  }, [businessObject?.id, tenantId, page, rowsPerPage, search, isTimeTravelEnabled, asOfDate]);

  useEffect(() => {
    fetchRecords();
  }, [fetchRecords]);

  const handleOpenCreate = () => {
    const initial: Record<string, any> = {};
    fields.forEach((f: any) => {
      const name = f.technicalName || f.name;
      if (name) initial[name] = '';
    });
    setFormData(initial);
    setCreateModalOpen(true);
  };

  const handleOpenEdit = (record: any) => {
    setSelectedRecord(record);
    setFormData({ ...record });
    setEditModalOpen(true);
  };

  const handleOpenDelete = (record: any) => {
    setSelectedRecord(record);
    setDeleteModalOpen(true);
  };

  const handleSaveCreate = async () => {
    setSubmitting(true);
    try {
      await apiClient(`/business-objects/${encodeURIComponent(businessObject.id)}/data`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({ record: formData }),
      });
      notification.success('Record created successfully');
      setCreateModalOpen(false);
      fetchRecords();
    } catch (err: any) {
      notification.error(err?.message || 'Failed to create record');
    } finally {
      setSubmitting(false);
    }
  };

  const handleSaveEdit = async () => {
    if (!selectedRecord) return;
    setSubmitting(true);
    try {
      const recordId = selectedRecord.id || selectedRecord[columns[0]];
      await apiClient(
        `/business-objects/${encodeURIComponent(businessObject.id)}/data/${encodeURIComponent(recordId)}`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
          },
          body: JSON.stringify({ record: formData }),
        }
      );
      notification.success('Record updated successfully');
      setEditModalOpen(false);
      fetchRecords();
    } catch (err: any) {
      notification.error(err?.message || 'Failed to update record');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!selectedRecord) return;
    setSubmitting(true);
    try {
      const recordId = selectedRecord.id || selectedRecord[columns[0]];
      await apiClient(
        `/business-objects/${encodeURIComponent(businessObject.id)}/data/${encodeURIComponent(recordId)}`,
        {
          method: 'DELETE',
          headers: {
            'X-Tenant-ID': tenantId,
          },
        }
      );
      notification.success('Record deleted successfully');
      setDeleteModalOpen(false);
      fetchRecords();
    } catch (err: any) {
      notification.error(err?.message || 'Failed to delete record');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      {/* Header Bar */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Box>
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="h6" sx={{ fontWeight: 700 }}>
              Physical Records & ORM Data
            </Typography>
            {driverTable && (
              <Chip
                icon={<StorageIcon sx={{ fontSize: '1rem !important' }} />}
                label={driverTable}
                size="small"
                variant="outlined"
                color="primary"
              />
            )}
          </Stack>
          <Typography variant="body2" color="text.secondary">
            Query, create, and manage live entity records directly through the Business Object ORM interface.
          </Typography>
        </Box>
        <Stack direction="row" spacing={1.5} alignItems="center">
          {execTimeMs !== null && (
            <Chip
              icon={<SpeedIcon sx={{ fontSize: '1rem !important' }} />}
              label={`${execTimeMs}ms`}
              size="small"
              variant="outlined"
            />
          )}
          <Button
            variant="contained"
            color="primary"
            startIcon={<AddIcon />}
            onClick={handleOpenCreate}
            size="small"
          >
            Add Record
          </Button>
        </Stack>
      </Stack>

      {/* Filter & Time Travel Toolbar */}
      <Paper variant="outlined" sx={{ p: 1.5, mb: 2 }}>
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems="center" justifyContent="space-between">
          <TextField
            size="small"
            placeholder="Search records..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(0);
            }}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" />
                </InputAdornment>
              ),
            }}
            sx={{ width: { xs: '100%', md: 320 } }}
          />

          <Stack direction="row" spacing={2} alignItems="center">
            {businessObject?.enableHistory && (
              <Stack direction="row" spacing={1} alignItems="center">
                <FormControlLabel
                  control={
                    <Switch
                      size="small"
                      checked={isTimeTravelEnabled}
                      onChange={(e) => setIsTimeTravelEnabled(e.target.checked)}
                    />
                  }
                  label={
                    <Stack direction="row" spacing={0.5} alignItems="center">
                      <HistoryIcon fontSize="small" color={isTimeTravelEnabled ? 'primary' : 'action'} />
                      <Typography variant="caption" sx={{ fontWeight: 600 }}>
                        Time Travel
                      </Typography>
                    </Stack>
                  }
                />
                {isTimeTravelEnabled && (
                  <TextField
                    type="datetime-local"
                    size="small"
                    value={asOfDate}
                    onChange={(e) => setAsOfDate(e.target.value)}
                    InputLabelProps={{ shrink: true }}
                    sx={{ width: 220 }}
                  />
                )}
              </Stack>
            )}

            <IconButton size="small" onClick={fetchRecords} disabled={loading} title="Refresh Records">
              <RefreshIcon fontSize="small" />
            </IconButton>
          </Stack>
        </Stack>
      </Paper>

      {/* Error Alert */}
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Records Table */}
      <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
        {loading && (
          <Box sx={{ p: 6, textAlign: 'center' }}>
            <CircularProgress size={32} />
            <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
              Loading physical records...
            </Typography>
          </Box>
        )}

        {!loading && rows.length === 0 && (
          <Box sx={{ p: 6, textAlign: 'center' }}>
            <StorageIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 1 }} />
            <Typography variant="h6" color="text.secondary">
              No Records Found
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5, mb: 2 }}>
              No rows returned from the physical driver table.
            </Typography>
            <Button variant="outlined" startIcon={<AddIcon />} onClick={handleOpenCreate} size="small">
              Create First Record
            </Button>
          </Box>
        )}

        {!loading && rows.length > 0 && (
          <>
            <TableContainer sx={{ maxHeight: 520 }}>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    {columns.map((col) => (
                      <TableCell key={col} sx={{ fontWeight: 700, bgcolor: 'background.paper' }}>
                        {col}
                      </TableCell>
                    ))}
                    <TableCell align="right" sx={{ fontWeight: 700, bgcolor: 'background.paper', width: 100 }}>
                      Actions
                    </TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {rows.map((row, rIdx) => (
                    <TableRow key={row.id || rIdx} hover>
                      {columns.map((col) => (
                        <TableCell key={col} sx={{ maxWidth: 240, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {row[col] === null || row[col] === undefined ? (
                            <Typography variant="caption" color="text.disabled">null</Typography>
                          ) : typeof row[col] === 'boolean' ? (
                            <Chip label={row[col] ? 'true' : 'false'} size="small" variant="outlined" color={row[col] ? 'success' : 'default'} />
                          ) : (
                            String(row[col])
                          )}
                        </TableCell>
                      ))}
                      <TableCell align="right">
                        <Stack direction="row" spacing={0.5} justifyContent="flex-end">
                          <Tooltip title="Edit Record">
                            <IconButton size="small" onClick={() => handleOpenEdit(row)}>
                              <EditIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="Delete Record">
                            <IconButton size="small" color="error" onClick={() => handleOpenDelete(row)}>
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </Stack>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
            <TablePagination
              rowsPerPageOptions={[10, 25, 50, 100]}
              component="div"
              count={total}
              rowsPerPage={rowsPerPage}
              page={page}
              onPageChange={(_, newPage) => setPage(newPage)}
              onRowsPerPageChange={(e) => {
                setRowsPerPage(parseInt(e.target.value, 10));
                setPage(0);
              }}
            />
          </>
        )}
      </Paper>

      {/* Create Modal */}
      <Dialog open={createModalOpen} onClose={() => setCreateModalOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 700 }}>Add New {businessObject?.displayName || 'Record'}</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {columns.filter(col => !['created_at', 'updated_at', 'last_modified_at'].includes(col)).map((col) => (
              <TextField
                key={col}
                label={col}
                size="small"
                fullWidth
                value={formData[col] ?? ''}
                onChange={(e) => setFormData({ ...formData, [col]: e.target.value })}
              />
            ))}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateModalOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSaveCreate} disabled={submitting}>
            {submitting ? <CircularProgress size={20} /> : 'Save Record'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Modal */}
      <Dialog open={editModalOpen} onClose={() => setEditModalOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 700 }}>Edit Record</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {columns.filter(col => !['id', 'created_at', 'updated_at'].includes(col)).map((col) => (
              <TextField
                key={col}
                label={col}
                size="small"
                fullWidth
                value={formData[col] ?? ''}
                onChange={(e) => setFormData({ ...formData, [col]: e.target.value })}
              />
            ))}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditModalOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSaveEdit} disabled={submitting}>
            {submitting ? <CircularProgress size={20} /> : 'Update Record'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Modal */}
      <Dialog open={deleteModalOpen} onClose={() => setDeleteModalOpen(false)}>
        <DialogTitle sx={{ fontWeight: 700 }}>Confirm Record Deletion</DialogTitle>
        <DialogContent>
          <Typography variant="body2">
            Are you sure you want to delete this record from <strong>{driverTable || 'driver table'}</strong>? This operation cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteModalOpen(false)}>Cancel</Button>
          <Button variant="contained" color="error" onClick={handleDelete} disabled={submitting}>
            {submitting ? <CircularProgress size={20} /> : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
