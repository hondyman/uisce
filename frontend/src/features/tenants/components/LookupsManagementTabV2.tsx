import React, { useState, useRef, useMemo } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  TextField,
  CircularProgress,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  FormHelperText,
  IconButton,
  Tooltip,
  Typography,
  Chip,
  Alert,
  Card,
  Grid,
  Paper,
  TableSortLabel,
  Drawer,
  Divider,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material';
import {
  Visibility as VisibilityIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Download as DownloadIcon,
  Upload as UploadIcon,
  Add as AddIcon,
  ControlPoint as ControlPointIcon,
  ViewAgenda as TableViewIcon,
  ViewWeek as CardViewIcon,
} from '@mui/icons-material';
import { useLookups, useLookupValues, useInfiniteLookupValues, createLookup, deleteLookup, createLookupValue, deleteLookupValue, updateLookup, updateLookupValue } from '../../../api/lookups';
import { useNotification } from '../../../hooks/useNotification';
import { useQueryClient } from '@tanstack/react-query';
import { FlatLookupValuesPanel } from './FlatLookupValuesPanel';

type SortField = 'name' | 'description' | 'type';
type SortOrder = 'asc' | 'desc';
type ViewMode = 'table' | 'card';
type FilterType = 'all' | 'core' | 'custom';

export default function LookupsManagementTab({ tenantId, instanceFilter }: { tenantId: string; instanceFilter?: string | null }) {
  const [search, setSearch] = useState('');
  const [selectedLookup, setSelectedLookup] = useState<string | null>(null);
  const [valuesOpen, setValuesOpen] = useState(false);
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [filterType, setFilterType] = useState<FilterType>('all');
  const [sortField, setSortField] = useState<SortField>('name');
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc');
  const [sidebarOpen, setSidebarOpen] = useState(true);

  const { data: lookups, isLoading } = useLookups(tenantId, search, 500);
  const qc = useQueryClient();
  const notification = useNotification();
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Export/Import dialog state
  const [exportDialogOpen, setExportDialogOpen] = useState(false);
  const [exportLookupId, setExportLookupId] = useState<string | null>(null);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [importLookupId, setImportLookupId] = useState<string | null>(null);

  // Dialog state for creating/editing lookups
  const [createLookupOpen, setCreateLookupOpen] = useState(false);
  const [createLookupForm, setCreateLookupForm] = useState<{ name: string; description?: string }>({ name: '', description: '' });
  const [editLookup, setEditLookup] = useState<{ id: string; name: string; description?: string } | null>(null);
  const [lookupFormError, setLookupFormError] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<{ open: boolean; type: 'lookup' | 'value' | null; id?: string }>(() => ({ open: false, type: null }));

  // Calculate facet counts
  const facetCounts = useMemo(() => {
    if (!lookups) return { core: 0, custom: 0, all: 0 };
    const core = lookups.filter((l: any) => l.is_core).length;
    const custom = lookups.filter((l: any) => !l.is_core).length;
    return { core, custom, all: lookups.length };
  }, [lookups]);

  // Filter and sort lookups
  const filteredAndSorted = useMemo(() => {
    let filtered = lookups || [];

    // Apply type filter
    if (filterType === 'core') {
      filtered = filtered.filter((l: any) => l.is_core);
    } else if (filterType === 'custom') {
      filtered = filtered.filter((l: any) => !l.is_core);
    }

    // Sort
    const sorted = [...filtered].sort((a: any, b: any) => {
      let aVal = a[sortField];
      let bVal = b[sortField];

      if (sortField === 'type') {
        aVal = a.is_core ? 'core' : 'custom';
        bVal = b.is_core ? 'core' : 'custom';
      }

      if (typeof aVal === 'string') aVal = aVal.toLowerCase();
      if (typeof bVal === 'string') bVal = bVal.toLowerCase();

      if (aVal < bVal) return sortOrder === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortOrder === 'asc' ? 1 : -1;
      return 0;
    });

    return sorted;
  }, [lookups, filterType, sortField, sortOrder]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('asc');
    }
  };

  const openValues = (id?: string) => {
    if (!id) return;
    setSelectedLookup(id);
    setValuesOpen(true);
  };

  const handleCreateLookup = async (name: string, description?: string) => {
    try {
      setLookupFormError(null);
      if (!name) throw new Error('A name is required');
      if (!tenantId) throw new Error('Missing tenant');
      await createLookup(tenantId, { name, description });
      notification.success('Lookup created');
      qc.invalidateQueries({ queryKey: ['lookups', tenantId] });
      setCreateLookupOpen(false);
      setCreateLookupForm({ name: '', description: '' });
    } catch (err: any) {
      setLookupFormError(err.message || String(err));
    }
  };

  const handleDeleteLookup = async (id: string) => {
    if (!tenantId) return;
    await deleteLookup(tenantId, id);
    notification.success('Lookup deleted');
    qc.invalidateQueries({ queryKey: ['lookups', tenantId] });
    setConfirmDelete({ open: false, type: null });
  };

  const handleOpenEditLookup = (lk: any) => {
    setEditLookup({ id: lk.id, name: lk.name, description: lk.description });
  };

  const handleUpdateLookup = async (id: string, name: string, description?: string) => {
    if (!tenantId) return;
    await updateLookup(tenantId, id, { name, description });
    notification.success('Lookup updated');
    qc.invalidateQueries({ queryKey: ['lookups', tenantId] });
    setEditLookup(null);
  };

  const handleOpenExportDialog = (lookupId: string) => {
    setExportLookupId(lookupId);
    setExportDialogOpen(true);
  };

  const handleOpenImportDialog = (lookupId: string) => {
    setImportLookupId(lookupId);
    setImportDialogOpen(true);
  };

  const handleDownloadTemplate = (format: 'excel' | 'csv' = 'excel') => {
    if (!exportLookupId) return;

    if (format === 'excel') {
      const excelContent = `
        <html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel">
          <head>
            <meta charset="UTF-8">
          </head>
          <body>
            <table>
              <thead>
                <tr>
                  <th>value</th>
                  <th>label</th>
                  <th>parent_id</th>
                  <th>metadata</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>example_value</td>
                  <td>Example Label</td>
                  <td></td>
                  <td>{}</td>
                </tr>
              </tbody>
            </table>
          </body>
        </html>
      `;

      const blob = new Blob([excelContent], { type: 'application/vnd.ms-excel' });
      const link = document.createElement('a');
      link.href = URL.createObjectURL(blob);
      link.download = `lookup_values_template.xls`;
      link.click();
      URL.revokeObjectURL(link.href);
    } else {
      const csvContent = 'value,label,parent_id,metadata\nexample_value,Example Label,,{}';
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
      const link = document.createElement('a');
      link.href = URL.createObjectURL(blob);
      link.download = `lookup_values_template.csv`;
      link.click();
      URL.revokeObjectURL(link.href);
    }
  };

  const handleExportValues = async (format: 'json' | 'csv') => {
    if (!exportLookupId || !tenantId) return;

    setExportDialogOpen(false);

    try {
      const url = `/api/lookups/${exportLookupId}/values?tenant_id=${tenantId}`;
      const res = await fetch(url, { credentials: 'include' });

      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || 'Failed to fetch lookup values');
      }

      const raw = await res.json();
      const allValues = (raw.items || []).map((r: any) => ({
        id: r.id,
        name: r.label || r.value || r.name,
        parent_id: r.parent_id || r.parentId || null,
        metadata: r.metadata,
        label: r.label,
        value: r.value,
      }));

      if (!allValues || allValues.length === 0) {
        notification.error('No values to export');
        return;
      }

      if (format === 'json') {
        const jsonContent = JSON.stringify(allValues, null, 2);
        const blob = new Blob([jsonContent], { type: 'application/json' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = `lookup_values_export.json`;
        link.click();
        URL.revokeObjectURL(link.href);
      } else {
        const headers = 'value,label,parent_id,metadata\n';
        const rows = allValues.map((v: any) => {
          const metadata = v.metadata ? JSON.stringify(v.metadata).replace(/"/g, '""') : '{}';
          return `"${v.name || v.value}","${v.label || v.name}","${v.parent_id || ''}","${metadata}"`;
        }).join('\n');

        const csvContent = headers + rows;
        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = `lookup_values_export.csv`;
        link.click();
        URL.revokeObjectURL(link.href);
      }

      notification.success(`Exported ${allValues.length} values as ${format.toUpperCase()}`);
    } catch (err: any) {
      notification.error('Failed to export values: ' + err.message);
    }
  };

  const handleUploadFile = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file || !importLookupId) return;

    setImportDialogOpen(false);

    try {
      const text = await file.text();
      const isExcel = file.name.endsWith('.xls') || file.name.endsWith('.xlsx');

      let lines: string[] = [];

      if (isExcel) {
        const parser = new DOMParser();
        const doc = parser.parseFromString(text, 'text/html');
        const rows = doc.querySelectorAll('tr');

        rows.forEach((row, index) => {
          if (index === 0) return;
          const cells = row.querySelectorAll('td');
          if (cells.length >= 4) {
            const line = Array.from(cells).map(cell => cell.textContent?.trim() || '').join(',');
            if (line) lines.push(line);
          }
        });
      } else {
        lines = text.split('\n').filter(line => line.trim());
        lines = lines.slice(1);
      }

      if (lines.length === 0) {
        notification.error('File is empty or invalid');
        return;
      }

      let successCount = 0;
      let errorCount = 0;

      for (const line of lines) {
        const [value, label, parent_id, metadataStr] = line.split(',').map(s => s.trim().replace(/^"|"$/g, ''));

        if (!value) continue;

        try {
          let metadata = {};
          if (metadataStr && metadataStr !== '{}') {
            try {
              metadata = JSON.parse(metadataStr.replace(/""/g, '"'));
            } catch {
              // If metadata is not valid JSON, skip it
            }
          }

          await createLookupValue(tenantId, importLookupId, {
            value,
            label: label || value,
            parent_id: parent_id || null,
            metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
          });
          successCount++;
        } catch (err) {
          errorCount++;
        }
      }

      qc.invalidateQueries({ queryKey: ['lookup-values', tenantId, importLookupId] });
      qc.invalidateQueries({ queryKey: ['lookup-values/infinite', tenantId, importLookupId] });

      if (successCount > 0) {
        notification.success(`Imported ${successCount} values${errorCount > 0 ? ` (${errorCount} failed)` : ''}`);
      } else {
        notification.error('Failed to import any values');
      }
    } catch (err: any) {
      notification.error('Failed to read file: ' + err.message);
    }

    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  // Render table view
  const renderTableView = () => (
    <Box sx={{ overflowX: 'auto' }}>
      <Table size="small">
        <TableHead sx={{ backgroundColor: '#fafafa' }}>
          <TableRow>
            <TableCell>
              <TableSortLabel active={sortField === 'name'} direction={sortField === 'name' ? sortOrder : 'asc'} onClick={() => handleSort('name')}>
                Name
              </TableSortLabel>
            </TableCell>
            <TableCell>
              <TableSortLabel active={sortField === 'description'} direction={sortField === 'description' ? sortOrder : 'asc'} onClick={() => handleSort('description')}>
                Description
              </TableSortLabel>
            </TableCell>
            <TableCell width={100}>
              <TableSortLabel active={sortField === 'type'} direction={sortField === 'type' ? sortOrder : 'asc'} onClick={() => handleSort('type')}>
                Type
              </TableSortLabel>
            </TableCell>
            <TableCell width={300} align="right">
              Actions
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {filteredAndSorted.map((lk: any) => (
            <TableRow key={lk.id} hover>
              <TableCell sx={{ fontWeight: 500 }}>{lk.name}</TableCell>
              <TableCell sx={{ color: 'text.secondary' }}>{lk.description || '—'}</TableCell>
              <TableCell>
                <Chip label={lk.is_core ? 'CORE' : 'CUSTOM'} size="small" color={lk.is_core ? 'primary' : 'default'} variant={lk.is_core ? 'filled' : 'outlined'} />
              </TableCell>
              <TableCell align="right">
                <Tooltip title="View Values">
                  <IconButton size="small" onClick={() => openValues(lk.id)} color="primary">
                    <VisibilityIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Export Values">
                  <IconButton size="small" onClick={() => handleOpenExportDialog(lk.id)} color="primary">
                    <DownloadIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Import Values">
                  <IconButton size="small" onClick={() => handleOpenImportDialog(lk.id)} color="primary">
                    <UploadIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Tooltip title={lk.tenant_id === tenantId ? 'Edit Lookup' : 'Read Only'}>
                  <span>
                    <IconButton size="small" onClick={() => handleOpenEditLookup(lk)} color="primary" disabled={lk.tenant_id !== tenantId}>
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
                <Tooltip title={lk.tenant_id === tenantId ? 'Delete Lookup' : 'Read Only'}>
                  <span>
                    <IconButton size="small" onClick={() => setConfirmDelete({ open: true, type: 'lookup', id: lk.id })} color="error" disabled={lk.tenant_id !== tenantId}>
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  );

  // Render card view
  const renderCardView = () => (
    <Grid container spacing={2}>
      {filteredAndSorted.map((lk: any) => (
        <Grid item xs={12} sm={6} md={4} key={lk.id}>
          <Card sx={{ display: 'flex', flexDirection: 'column', height: '100%' }} elevation={2}>
            <Box sx={{ p: 2, pb: 1 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', mb: 1 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, flex: 1 }}>
                  {lk.name}
                </Typography>
                <Chip label={lk.is_core ? 'CORE' : 'CUSTOM'} size="small" color={lk.is_core ? 'primary' : 'default'} variant={lk.is_core ? 'filled' : 'outlined'} />
              </Box>
              {lk.description && <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>{lk.description}</Typography>}
            </Box>
            <Divider />
            <Box sx={{ p: 2, display: 'flex', gap: 0.5, justifyContent: 'flex-end', flexWrap: 'wrap' }}>
              <Tooltip title="View Values">
                <IconButton size="small" onClick={() => openValues(lk.id)} color="primary">
                  <VisibilityIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Tooltip title="Export Values">
                <IconButton size="small" onClick={() => handleOpenExportDialog(lk.id)} color="primary">
                  <DownloadIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Tooltip title="Import Values">
                <IconButton size="small" onClick={() => handleOpenImportDialog(lk.id)} color="primary">
                  <UploadIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Tooltip title={lk.tenant_id === tenantId ? 'Edit' : 'Read Only'}>
                <span>
                  <IconButton size="small" onClick={() => handleOpenEditLookup(lk)} color="primary" disabled={lk.tenant_id !== tenantId}>
                    <EditIcon fontSize="small" />
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title={lk.tenant_id === tenantId ? 'Delete' : 'Read Only'}>
                <span>
                  <IconButton size="small" onClick={() => setConfirmDelete({ open: true, type: 'lookup', id: lk.id })} color="error" disabled={lk.tenant_id !== tenantId}>
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </span>
              </Tooltip>
            </Box>
          </Card>
        </Grid>
      ))}
    </Grid>
  );

  return (
    <Box sx={{ display: 'flex', height: '100%', bgcolor: '#fafafa' }}>
      {/* Sidebar with Facets */}
      <Drawer variant="permanent" sx={{ width: sidebarOpen ? 280 : 0, transition: 'all 0.3s', overflow: 'hidden' }} PaperProps={{ sx: { position: 'relative', width: 280, pt: 2 } }}>
        <Box sx={{ px: 2 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600, textTransform: 'uppercase', fontSize: '0.75rem', color: 'text.secondary', mb: 2 }}>
            Type
          </Typography>
          <Stack spacing={1}>
            {[
              { key: 'all', label: 'All', count: facetCounts.all },
              { key: 'core', label: 'Core', count: facetCounts.core },
              { key: 'custom', label: 'Custom', count: facetCounts.custom },
            ].map((facet) => (
              <Button
                key={facet.key}
                variant={filterType === facet.key ? 'contained' : 'text'}
                fullWidth
                onClick={() => setFilterType(facet.key as FilterType)}
                sx={{
                  justifyContent: 'space-between',
                  textTransform: 'none',
                  fontWeight: filterType === facet.key ? 600 : 400,
                }}
              >
                <span>{facet.label}</span>
                <Chip label={facet.count} size="small" variant="outlined" sx={{ ml: 1 }} />
              </Button>
            ))}
          </Stack>
        </Box>
      </Drawer>

      {/* Main Content */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', p: 3 }}>
        {/* Header */}
        <Box sx={{ mb: 3 }}>
          <Typography variant="h5" sx={{ fontWeight: 600, mb: 3 }}>
            Lookups
          </Typography>

          {/* Search and Controls */}
          <Paper sx={{ p: 2, mb: 2 }}>
            <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', justifyContent: 'space-between' }}>
              <TextField
                placeholder="Search lookups..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                size="small"
                sx={{ flex: 1, maxWidth: 400 }}
              />
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                  {filteredAndSorted.length}
                </Typography>
                <ToggleButtonGroup value={viewMode} exclusive onChange={(e, newVal) => newVal && setViewMode(newVal)} size="small">
                  <ToggleButton value="table" title="Table view">
                    <TableViewIcon fontSize="small" />
                  </ToggleButton>
                  <ToggleButton value="card" title="Card view">
                    <CardViewIcon fontSize="small" />
                  </ToggleButton>
                </ToggleButtonGroup>
                <Button variant="contained" size="small" startIcon={<AddIcon />} onClick={() => setCreateLookupOpen(true)}>
                  New
                </Button>
              </Box>
            </Box>
          </Paper>

          {instanceFilter && (
            <Alert severity="info" sx={{ mb: 2 }}>
              Lookups are tenant-wide and apply to all instances. Instance filtering does not apply here.
            </Alert>
          )}
        </Box>

        {/* Content */}
        {isLoading ? <CircularProgress sx={{ mx: 'auto' }} /> : viewMode === 'table' ? renderTableView() : renderCardView()}

        {filteredAndSorted.length === 0 && !isLoading && (
          <Box sx={{ textAlign: 'center', py: 6 }}>
            <Typography color="text.secondary">No lookups found. Create one to get started.</Typography>
          </Box>
        )}
      </Box>

      {/* Dialogs */}
      <Dialog open={valuesOpen} onClose={() => setValuesOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Lookup Values</DialogTitle>
        <DialogContent>{selectedLookup && <LookupValuesPanelRouter tenantId={tenantId} lookupId={selectedLookup} onRequestDelete={(id) => setConfirmDelete({ open: true, type: 'value', id })} />}</DialogContent>
        <DialogActions>
          <Button onClick={() => setValuesOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={createLookupOpen} onClose={() => setCreateLookupOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>New Lookup</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Name"
            fullWidth
            value={createLookupForm.name}
            onChange={(e) => setCreateLookupForm((s) => ({ ...s, name: e.target.value }))}
          />
          <TextField
            margin="dense"
            label="Description"
            fullWidth
            value={createLookupForm.description}
            onChange={(e) => setCreateLookupForm((s) => ({ ...s, description: e.target.value }))}
          />
          {lookupFormError && <FormHelperText error>{lookupFormError}</FormHelperText>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateLookupOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => handleCreateLookup(createLookupForm.name, createLookupForm.description)}>
            Create
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!editLookup} onClose={() => setEditLookup(null)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Lookup</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Name"
            fullWidth
            value={editLookup?.name || ''}
            onChange={(e) => setEditLookup((prev) => (prev ? { ...prev, name: e.target.value } : prev))}
          />
          <TextField
            margin="dense"
            label="Description"
            fullWidth
            value={editLookup?.description || ''}
            onChange={(e) => setEditLookup((prev) => (prev ? { ...prev, description: e.target.value } : prev))}
          />
          {lookupFormError && <FormHelperText error>{lookupFormError}</FormHelperText>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditLookup(null)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={() => editLookup && handleUpdateLookup(editLookup.id, editLookup.name, editLookup.description)}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={confirmDelete.open} onClose={() => setConfirmDelete({ open: false, type: null })}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>Are you sure you want to delete this {confirmDelete.type}?</DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDelete({ open: false, type: null })}>Cancel</Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => {
              if (confirmDelete.type === 'lookup' && confirmDelete.id) handleDeleteLookup(confirmDelete.id);
              if (confirmDelete.type === 'value' && confirmDelete.id && selectedLookup) {
                deleteLookupValue(tenantId, selectedLookup, confirmDelete.id).then(() => {
                  qc.invalidateQueries({ queryKey: ['lookup-values', tenantId, selectedLookup] });
                  setConfirmDelete({ open: false, type: null });
                });
              }
            }}
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={exportDialogOpen} onClose={() => setExportDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <DownloadIcon color="primary" />
          Export Lookup Values
        </DialogTitle>
        <DialogContent sx={{ pt: 3 }}>
          <Stack spacing={2}>
            <Button variant="outlined" fullWidth onClick={() => handleExportValues('json')}>
              JSON Format
            </Button>
            <Button variant="outlined" fullWidth onClick={() => handleExportValues('csv')}>
              CSV Format
            </Button>
            <Button
              variant="outlined"
              fullWidth
              onClick={() => {
                handleDownloadTemplate('excel');
                setExportDialogOpen(false);
              }}
            >
              Download Template
            </Button>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setExportDialogOpen(false)}>Cancel</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={importDialogOpen} onClose={() => setImportDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <UploadIcon color="primary" />
          Import Lookup Values
        </DialogTitle>
        <DialogContent sx={{ pt: 3 }}>
          <input
            ref={fileInputRef}
            type="file"
            accept=".csv,.xls,.xlsx"
            style={{ display: 'none' }}
            onChange={handleUploadFile}
          />
          <Box
            sx={{
              border: '2px dashed',
              borderColor: 'primary.main',
              borderRadius: 2,
              p: 4,
              textAlign: 'center',
              cursor: 'pointer',
              '&:hover': { backgroundColor: 'primary.50' },
            }}
            onClick={() => fileInputRef.current?.click()}
          >
            <UploadIcon sx={{ fontSize: 48, color: 'primary.main', mb: 2 }} />
            <Typography variant="h6">Select file to import</Typography>
            <Typography variant="caption" color="text.secondary">
              CSV, XLS, or XLSX
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setImportDialogOpen(false)}>Cancel</Button>
        </DialogActions>
      </Dialog>

      <input ref={fileInputRef} type="file" accept=".csv,.xls,.xlsx" style={{ display: 'none' }} onChange={handleUploadFile} />
    </Box>
  );
}

function LookupValuesPanelRouter({ tenantId, lookupId, onRequestDelete }: { tenantId: string; lookupId: string; onRequestDelete?: (id: string) => void }) {
  const { data: values } = useLookupValues(tenantId, lookupId);

  const isFlatLookup = values && values.length > 0 && values.every((v: any) => v.metadata && !v.parent_id);

  if (isFlatLookup) {
    return <FlatLookupValuesPanel tenantId={tenantId} lookupId={lookupId} onRequestDelete={onRequestDelete} />;
  }

  return <LookupValuesPanel tenantId={tenantId} lookupId={lookupId} onRequestDelete={onRequestDelete} />;
}

function LookupValuesPanel({ tenantId, lookupId, onRequestDelete }: { tenantId: string; lookupId: string; onRequestDelete?: (id: string) => void }) {
  const notification = useNotification();
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useInfiniteLookupValues(tenantId, lookupId, null, null, 20);
  const { data: allValues } = useLookupValues(tenantId, lookupId);
  const qc = useQueryClient();
  const [createValueOpen, setCreateValueOpen] = useState(false);
  const [createValueForm, setCreateValueForm] = useState<{ value: string; label?: string; parent_id?: string | null }>({ value: '', label: '', parent_id: null });
  const [editValue, setEditValue] = useState<{ id: string; value: string; label?: string; parent_id?: string | null } | null>(null);
  const [valueFormError, setValueFormError] = useState<string | null>(null);
  const [searchValues, setSearchValues] = useState('');
  const [sortBy, setSortBy] = useState<'name' | 'type'>('name');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');

  const displayValues = data?.pages.flatMap((page: any) => page.items) || [];

  const filteredValues = displayValues.filter((v: any) => v.name.toLowerCase().includes(searchValues.toLowerCase()));

  const sortedValues = useMemo(() => {
    const sorted = [...filteredValues];
    sorted.sort((a: any, b: any) => {
      let compareA, compareB;
      if (sortBy === 'name') {
        compareA = a.name.toLowerCase();
        compareB = b.name.toLowerCase();
      } else {
        compareA = a.is_core ? 0 : 1;
        compareB = b.is_core ? 0 : 1;
      }
      return sortOrder === 'asc' ? compareA.localeCompare(compareB) : compareB.localeCompare(compareA);
    });
    return sorted;
  }, [filteredValues, sortBy, sortOrder]);

  const handleSort = (field: 'name' | 'type') => {
    if (sortBy === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(field);
      setSortOrder('asc');
    }
  };

  const handleCreateValue = async (value: string, label?: string, parent_id?: string | null) => {
    if (!tenantId) return;
    await createLookupValue(tenantId, lookupId, { value, label: label || value, parent_id });
    notification.success('Lookup value created');
    qc.invalidateQueries({ queryKey: ['lookup-values', tenantId, lookupId] });
    qc.invalidateQueries({ queryKey: ['lookup-values/infinite', tenantId, lookupId] });
    setCreateValueOpen(false);
  };

  const handleUpdateValue = async (id: string, value: string, label?: string, parent_id?: string | null) => {
    if (!tenantId) return;
    await updateLookupValue(tenantId, lookupId, id, { value, label, parent_id });
    notification.success('Lookup value updated');
    qc.invalidateQueries({ queryKey: ['lookup-values', tenantId, lookupId] });
    qc.invalidateQueries({ queryKey: ['lookup-values/infinite', tenantId, lookupId] });
    setEditValue(null);
  };

  if (isLoading) return <CircularProgress size={18} />;

  return (
    <>
      <Box sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
        <TextField label="Search values" value={searchValues} onChange={(e) => setSearchValues(e.target.value)} size="small" fullWidth placeholder="Search by name or label..." />
        <Button variant="contained" size="small" onClick={() => setCreateValueOpen(true)} sx={{ whiteSpace: 'nowrap' }}>
          New Value
        </Button>
      </Box>
      <Box sx={{ overflowX: 'auto' }}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>
                <TableSortLabel
                  active={sortBy === 'name'}
                  direction={sortOrder}
                  onClick={() => handleSort('name')}
                >
                  Name
                </TableSortLabel>
              </TableCell>
              <TableCell>
                <TableSortLabel
                  active={sortBy === 'type'}
                  direction={sortOrder}
                  onClick={() => handleSort('type')}
                >
                  Type
                </TableSortLabel>
              </TableCell>
              <TableCell>Parent</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {sortedValues.map((v: any) => (
              <TableRow key={v.id}>
                <TableCell sx={{ fontWeight: 500 }}>{v.name}</TableCell>
                <TableCell>
                  <Chip label={v.is_core ? 'CORE' : 'CUSTOM'} size="small" color={v.is_core ? 'primary' : 'default'} variant={v.is_core ? 'filled' : 'outlined'} />
                </TableCell>
                <TableCell>{v.parent_id || '—'}</TableCell>
                <TableCell align="right">
                  <Tooltip title={v.tenant_id === tenantId ? 'Edit' : 'Read Only'}>
                    <span>
                      <IconButton
                        size="small"
                        onClick={() => setEditValue({ id: v.id, value: v.name, label: v.label || v.name, parent_id: v.parent_id || null })}
                        disabled={v.tenant_id !== tenantId}
                        color="primary"
                      >
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </span>
                  </Tooltip>
                  <Tooltip title={v.tenant_id === tenantId ? 'Delete' : 'Read Only'}>
                    <span>
                      <IconButton
                        size="small"
                        onClick={() => onRequestDelete?.(v.id)}
                        disabled={v.tenant_id !== tenantId}
                        color="error"
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </span>
                  </Tooltip>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Box>
      {hasNextPage && (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
          <Button variant="outlined" size="small" onClick={() => fetchNextPage()} disabled={isFetchingNextPage}>
            {isFetchingNextPage ? <CircularProgress size={16} /> : 'Load More'}
          </Button>
        </Box>
      )}
      <Dialog open={createValueOpen} onClose={() => setCreateValueOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>New Lookup Value</DialogTitle>
        <DialogContent>
          <TextField autoFocus margin="dense" label="Value" fullWidth value={createValueForm.value} onChange={(e) => setCreateValueForm((s) => ({ ...s, value: e.target.value }))} />
          <TextField margin="dense" label="Label" fullWidth value={createValueForm.label} onChange={(e) => setCreateValueForm((s) => ({ ...s, label: e.target.value }))} />
          <FormControl fullWidth sx={{ mt: 1 }}>
            <InputLabel id="parent-select-label">Parent</InputLabel>
            <Select
              labelId="parent-select-label"
              value={createValueForm.parent_id || ''}
              label="Parent"
              onChange={(e) => setCreateValueForm((s) => ({ ...s, parent_id: e.target.value || null }))}
            >
              <MenuItem value="">None</MenuItem>
              {(allValues || []).map((opt: any) => (
                <MenuItem key={opt.id} value={opt.id}>
                  {opt.name}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          {valueFormError && <FormHelperText error>{valueFormError}</FormHelperText>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateValueOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={() => {
              setValueFormError(null);
              if (!createValueForm.value) {
                setValueFormError('Value required');
                return;
              }
              handleCreateValue(createValueForm.value, createValueForm.label, createValueForm.parent_id);
            }}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!editValue} onClose={() => setEditValue(null)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Lookup Value</DialogTitle>
        <DialogContent>
          <TextField autoFocus margin="dense" label="Value" fullWidth value={editValue?.value || ''} onChange={(e) => setEditValue((s) => (s ? { ...s, value: e.target.value } : s))} />
          <TextField margin="dense" label="Label" fullWidth value={editValue?.label || ''} onChange={(e) => setEditValue((s) => (s ? { ...s, label: e.target.value } : s))} />
          <FormControl fullWidth sx={{ mt: 1 }}>
            <InputLabel id="parent-select-label">Parent</InputLabel>
            <Select
              labelId="parent-select-label"
              value={editValue?.parent_id || ''}
              label="Parent"
              onChange={(e) => setEditValue((s) => (s ? { ...s, parent_id: e.target.value || null } : s))}
            >
              <MenuItem value="">None</MenuItem>
              {(allValues || []).map((opt: any) => opt.id !== editValue?.id && <MenuItem key={opt.id} value={opt.id}>{opt.name}</MenuItem>)}
            </Select>
          </FormControl>
          {valueFormError && <FormHelperText error>{valueFormError}</FormHelperText>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditValue(null)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={() => {
              setValueFormError(null);
              if (!editValue || !editValue.value) {
                setValueFormError('Value required');
                return;
              }
              handleUpdateValue(editValue.id, editValue.value, editValue.label, editValue.parent_id);
            }}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
