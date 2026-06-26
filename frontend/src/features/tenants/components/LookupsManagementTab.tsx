import React, { useState, useRef } from 'react';
import { Box, Button, Dialog, DialogTitle, DialogContent, DialogActions, Table, TableHead, TableRow, TableCell, TableBody, TextField, CircularProgress, MenuItem, Select, FormControl, InputLabel, FormHelperText, IconButton, Tooltip, Typography, Chip, Alert } from '@mui/material';
import { Visibility as VisibilityIcon, Edit as EditIcon, Delete as DeleteIcon, Download as DownloadIcon, Upload as UploadIcon, Add as AddIcon, ControlPoint as ControlPointIcon } from '@mui/icons-material';
import { useLookups, useLookupValues, useInfiniteLookupValues, createLookup, deleteLookup, createLookupValue, deleteLookupValue, updateLookup, updateLookupValue } from '../../../api/lookups';
import { useNotification } from '../../../hooks/useNotification';
import { useQueryClient } from '@tanstack/react-query';
import { FlatLookupValuesPanel } from './FlatLookupValuesPanel';
// import { useTenant } from '../../../contexts/TenantContext';

export default function LookupsManagementTab({ tenantId, instanceFilter }: { tenantId: string; instanceFilter?: string | null }) {
  const [search, setSearch] = useState('');
  const [selectedLookup, setSelectedLookup] = useState<string | null>(null);
  const [valuesOpen, setValuesOpen] = useState(false);
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
  // Confirmation dialog state
  const [confirmDelete, setConfirmDelete] = useState<{ open: boolean; type: 'lookup' | 'value' | null; id?: string }>(() => ({ open: false, type: null }));

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
      // Create Excel-compatible HTML table
      const excelContent = `
        <html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel">
          <head>
            <meta charset="UTF-8">
            <!--[if gte mso 9]><xml><x:ExcelWorkbook><x:ExcelWorksheets><x:ExcelWorksheet>
            <x:Name>Lookup Values</x:Name><x:WorksheetOptions><x:DisplayGridlines/></x:WorksheetOptions>
            </x:ExcelWorksheet></x:ExcelWorksheets></x:ExcelWorkbook></xml><![endif]-->
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
                <tr>
                  <td>child_value</td>
                  <td>Child Example</td>
                  <td>example_value</td>
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
      // Create CSV content
      const csvContent = 'value,label,parent_id,metadata\n' +
        'example_value,Example Label,,{}\n' +
        'child_value,Child Example,example_value,{}';
      
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
      // Fetch all values for the selected lookup directly using fetch API
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
        value: r.value 
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
        // CSV format
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
        // Parse HTML table from Excel file
        const parser = new DOMParser();
        const doc = parser.parseFromString(text, 'text/html');
        const rows = doc.querySelectorAll('tr');
        
        rows.forEach((row, index) => {
          if (index === 0) return; // Skip header
          const cells = row.querySelectorAll('td');
          if (cells.length >= 4) {
            const line = Array.from(cells).map(cell => cell.textContent?.trim() || '').join(',');
            if (line) lines.push(line);
          }
        });
      } else {
        // Parse CSV
        lines = text.split('\n').filter(line => line.trim());
        lines = lines.slice(1); // Skip header
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
            metadata: Object.keys(metadata).length > 0 ? metadata : undefined
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

    // Reset file input
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };



  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
        <TextField label="Search lookups" value={search} onChange={(e) => setSearch(e.target.value)} size="small" />
        
        <Box sx={{ flex: 1 }} />
        
        <Tooltip title="New Lookup">
          <IconButton 
            onClick={() => setCreateLookupOpen(true)} 
            color="primary"
            sx={{ fontSize: '2rem' }}
          >
            <ControlPointIcon sx={{ fontSize: '2rem' }} />
          </IconButton>
        </Tooltip>
      </Box>

      {instanceFilter && (
        <Alert severity="info" sx={{ mb: 2 }}>
          Lookups are tenant-wide and apply to all instances. Instance filtering does not apply here.
        </Alert>
      )}

      {isLoading ? (
        <CircularProgress />
      ) : (
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Description</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {(lookups || []).map((lk: any) => (
              <TableRow key={lk.id} hover>
                <TableCell>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    {lk.name}
                    {lk.is_core ? (
                      <Chip 
                        label="CORE" 
                        size="small" 
                        color="info" 
                        title="Cloned from Gold Copy"
                        sx={{ height: 20, fontSize: '0.65rem', fontWeight: 'bold' }} 
                      />
                    ) : (
                      <Chip 
                        label="CUSTOM" 
                        size="small" 
                        variant="outlined"
                        sx={{ height: 20, fontSize: '0.65rem', fontWeight: 'bold' }} 
                      />
                    )}
                  </Box>
                </TableCell>
                <TableCell>{lk.description}</TableCell>
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
                  <Tooltip title={lk.tenant_id === tenantId ? "Edit Lookup" : "Read Only"}>
                    <span>
                      <IconButton size="small" onClick={() => handleOpenEditLookup(lk)} color="primary" disabled={lk.tenant_id !== tenantId}>
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </span>
                  </Tooltip>
                  <Tooltip title={lk.tenant_id === tenantId ? "Delete Lookup" : "Read Only"}>
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
      )}

        <Dialog open={valuesOpen} onClose={() => setValuesOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Lookup Values</DialogTitle>
        <DialogContent>
          {selectedLookup && <LookupValuesPanelRouter tenantId={tenantId} lookupId={selectedLookup} onRequestDelete={(id) => setConfirmDelete({ open: true, type: 'value', id })} />}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setValuesOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={createLookupOpen} onClose={() => setCreateLookupOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>New Lookup</DialogTitle>
        <DialogContent>
          <TextField autoFocus margin="dense" label="Name" fullWidth value={createLookupForm.name} onChange={(e) => setCreateLookupForm((s) => ({ ...s, name: e.target.value }))} />
          <TextField margin="dense" label="Description" fullWidth value={createLookupForm.description} onChange={(e) => setCreateLookupForm((s) => ({ ...s, description: e.target.value }))} />
          {lookupFormError && <FormHelperText error>{lookupFormError}</FormHelperText>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateLookupOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => handleCreateLookup(createLookupForm.name, createLookupForm.description)}>Create</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!editLookup} onClose={() => setEditLookup(null)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Lookup</DialogTitle>
        <DialogContent>
          <TextField autoFocus margin="dense" label="Name" fullWidth value={editLookup?.name || ''} onChange={(e) => setEditLookup((prev) => prev ? ({ ...prev, name: e.target.value }) : prev)} />
          <TextField margin="dense" label="Description" fullWidth value={editLookup?.description || ''} onChange={(e) => setEditLookup((prev) => prev ? ({ ...prev, description: e.target.value }) : prev)} />
          {lookupFormError && <FormHelperText error>{lookupFormError}</FormHelperText>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditLookup(null)}>Cancel</Button>
          <Button variant="contained" onClick={() => editLookup && handleUpdateLookup(editLookup.id, editLookup.name, editLookup.description)}>Save</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={confirmDelete.open} onClose={() => setConfirmDelete({ open: false, type: null })}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>Are you sure you want to delete this {confirmDelete.type}?</DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDelete({ open: false, type: null })}>Cancel</Button>
          <Button color="error" variant="contained" onClick={() => {
            if (confirmDelete.type === 'lookup' && confirmDelete.id) handleDeleteLookup(confirmDelete.id);
            if (confirmDelete.type === 'value' && confirmDelete.id && selectedLookup) { deleteLookupValue(tenantId, selectedLookup, confirmDelete.id).then(() => { qc.invalidateQueries({ queryKey: ['lookup-values', tenantId, selectedLookup] }); setConfirmDelete({ open: false, type: null }); }); }
          }}>Delete</Button>
        </DialogActions>
      </Dialog>

      {/* Export Format Selection Dialog */}
      <Dialog 
        open={exportDialogOpen} 
        onClose={() => setExportDialogOpen(false)} 
        maxWidth="sm" 
        fullWidth
        PaperProps={{
          sx: {
            borderRadius: 2,
            boxShadow: '0 8px 32px rgba(0,0,0,0.12)'
          }
        }}
      >
        <DialogTitle sx={{ 
          pb: 1, 
          display: 'flex', 
          alignItems: 'center', 
          gap: 1,
          borderBottom: '1px solid',
          borderColor: 'divider'
        }}>
          <DownloadIcon color="primary" />
          <Typography variant="h6">Export Lookup Values</Typography>
        </DialogTitle>
        <DialogContent sx={{ pt: 3 }}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Button
              variant="outlined"
              size="large"
              startIcon={<DownloadIcon />}
              onClick={() => handleExportValues('json')}
              fullWidth
              sx={{
                justifyContent: 'flex-start',
                py: 2,
                px: 3,
                borderRadius: 2,
                textAlign: 'left',
                '&:hover': {
                  backgroundColor: 'primary.50',
                  borderColor: 'primary.main'
                }
              }}
            >
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-start', flex: 1 }}>
                <Typography variant="subtitle1" fontWeight={600}>Export as JSON</Typography>
                <Typography variant="caption" color="text.secondary">
                  Structured data format, ideal for programmatic use
                </Typography>
              </Box>
            </Button>
            <Button
              variant="outlined"
              size="large"
              startIcon={<DownloadIcon />}
              onClick={() => handleExportValues('csv')}
              fullWidth
              sx={{
                justifyContent: 'flex-start',
                py: 2,
                px: 3,
                borderRadius: 2,
                textAlign: 'left',
                '&:hover': {
                  backgroundColor: 'primary.50',
                  borderColor: 'primary.main'
                }
              }}
            >
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-start', flex: 1 }}>
                <Typography variant="subtitle1" fontWeight={600}>Export as CSV</Typography>
                <Typography variant="caption" color="text.secondary">
                  Spreadsheet format, compatible with Excel and Google Sheets
                </Typography>
              </Box>
            </Button>
            <Button
              variant="outlined"
              size="large"
              startIcon={<DownloadIcon />}
              onClick={() => {
                handleDownloadTemplate('excel');
                setExportDialogOpen(false);
              }}
              fullWidth
              sx={{
                justifyContent: 'flex-start',
                py: 2,
                px: 3,
                borderRadius: 2,
                textAlign: 'left',
                '&:hover': {
                  backgroundColor: 'primary.50',
                  borderColor: 'primary.main'
                }
              }}
            >
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-start', flex: 1 }}>
                <Typography variant="subtitle1" fontWeight={600}>Download Excel Template</Typography>
                <Typography variant="caption" color="text.secondary">
                  Empty template with example data for bulk import
                </Typography>
              </Box>
            </Button>
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setExportDialogOpen(false)} size="large">Cancel</Button>
        </DialogActions>
      </Dialog>

      {/* Import File Upload Dialog */}
      <Dialog 
        open={importDialogOpen} 
        onClose={() => setImportDialogOpen(false)} 
        maxWidth="sm" 
        fullWidth
        PaperProps={{
          sx: {
            borderRadius: 2,
            boxShadow: '0 8px 32px rgba(0,0,0,0.12)'
          }
        }}
      >
        <DialogTitle sx={{ 
          pb: 1, 
          display: 'flex', 
          alignItems: 'center', 
          gap: 1,
          borderBottom: '1px solid',
          borderColor: 'divider'
        }}>
          <UploadIcon color="primary" />
          <Typography variant="h6">Import Lookup Values</Typography>
        </DialogTitle>
        <DialogContent sx={{ pt: 3 }}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, alignItems: 'center' }}>
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv,.xls,.xlsx"
              style={{ display: 'none' }}
              onChange={handleUploadFile}
            />
            <Box
              sx={{
                width: '100%',
                border: '2px dashed',
                borderColor: 'primary.main',
                borderRadius: 2,
                p: 4,
                textAlign: 'center',
                backgroundColor: 'primary.50',
                cursor: 'pointer',
                transition: 'all 0.2s',
                '&:hover': {
                  backgroundColor: 'primary.100',
                  borderColor: 'primary.dark'
                }
              }}
              onClick={() => fileInputRef.current?.click()}
            >
              <UploadIcon sx={{ fontSize: 48, color: 'primary.main', mb: 2 }} />
              <Typography variant="h6" gutterBottom>
                Click to select file
              </Typography>
              <Typography variant="body2" color="text.secondary">
                or drag and drop your file here
              </Typography>
            </Box>
            <Box sx={{ textAlign: 'center' }}>
              <Typography variant="caption" color="text.secondary" display="block" gutterBottom>
                Supported formats:
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, justifyContent: 'center', flexWrap: 'wrap' }}>
                <Chip label="CSV" size="small" variant="outlined" />
                <Chip label="Excel (.xls)" size="small" variant="outlined" />
                <Chip label="Excel (.xlsx)" size="small" variant="outlined" />
              </Box>
            </Box>
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setImportDialogOpen(false)} size="large">Cancel</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

function LookupValuesPanelRouter({ tenantId, lookupId, onRequestDelete }: { tenantId: string; lookupId: string; onRequestDelete?: (id: string) => void }) {
  // Detect if this is a flat lookup by checking if it has no parent values
  const { data: values } = useLookupValues(tenantId, lookupId);
  
  // If all values have metadata and no parent_id, use flat panel
  const isFlatLookup = values && values.length > 0 && values.every((v: any) => v.metadata && !v.parent_id);
  
  if (isFlatLookup) {
    return <FlatLookupValuesPanel tenantId={tenantId} lookupId={lookupId} onRequestDelete={onRequestDelete} />;
  }
  
  // Otherwise use standard hierarchical panel
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

  // Flatten paginated results for display
  const displayValues = data?.pages.flatMap((page: any) => page.items) || [];
  
  const filteredValues = displayValues.filter((v: any) => 
    v.name.toLowerCase().includes(searchValues.toLowerCase())
  );

  const handleCreateValue = async (value: string, label?: string, parent_id?: string | null) => {
    if (!tenantId) return;
    await createLookupValue(tenantId, lookupId, { value, label: label || value, parent_id });
    notification.success('Lookup value created');
    qc.invalidateQueries({ queryKey: ['lookup-values', tenantId, lookupId] });
    qc.invalidateQueries({ queryKey: ['lookup-values/infinite', tenantId, lookupId] });
    setCreateValueOpen(false);
  };

  const handleOpenEditValue = (v: any) => setEditValue({ id: v.id, value: v.name, label: v.label || v.name, parent_id: v.parent_id || null });

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
        <TextField 
          label="Search values" 
          value={searchValues} 
          onChange={(e) => setSearchValues(e.target.value)} 
          size="small" 
          fullWidth
          placeholder="Search by name or label..."
        />
        <Button variant="contained" size="small" onClick={() => setCreateValueOpen(true)} sx={{ whiteSpace: 'nowrap' }}>New Value</Button>
      </Box>
      <Table size="small">
      <TableHead>
        <TableRow>
          <TableCell>Name</TableCell>
          <TableCell>Parent</TableCell>
          <TableCell>Actions</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {filteredValues.map((v: any) => (
          <TableRow key={v.id}>
            <TableCell>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                {v.name}
                {v.is_core ? (
                  <Chip 
                    label="CORE" 
                    size="small" 
                    color="info" 
                    title="Cloned from Gold Copy"
                    sx={{ height: 20, fontSize: '0.65rem', fontWeight: 'bold' }} 
                  />
                ) : (
                  <Chip 
                    label="CUSTOM" 
                    size="small" 
                    variant="outlined"
                    sx={{ height: 20, fontSize: '0.65rem', fontWeight: 'bold' }} 
                  />
                )}
              </Box>
            </TableCell>
            <TableCell>{v.parent_id || '—'}</TableCell>
            <TableCell>
              <Tooltip title={v.tenant_id === tenantId ? "Edit" : "Read Only"}>
                <span>
                  <Button size="small" onClick={() => setEditValue({ id: v.id, value: v.name, label: v.label || v.name, parent_id: v.parent_id || null })} disabled={v.tenant_id !== tenantId}>Edit</Button>
                </span>
              </Tooltip>
              <Tooltip title={v.tenant_id === tenantId ? "Delete" : "Read Only"}>
                <span>
                  <Button size="small" color="error" onClick={() => onRequestDelete?.(v.id)} disabled={v.tenant_id !== tenantId}>Delete</Button>
                </span>
              </Tooltip>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
      {hasNextPage && (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
          <Button 
            variant="outlined" 
            size="small" 
            onClick={() => fetchNextPage()} 
            disabled={isFetchingNextPage}
          >
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
                <MenuItem key={opt.id} value={opt.id}>{opt.name}</MenuItem>
              ))}
            </Select>
          </FormControl>
          {valueFormError && <FormHelperText error>{valueFormError}</FormHelperText>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateValueOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => { setValueFormError(null); if (!createValueForm.value) { setValueFormError('Value required'); return; } handleCreateValue(createValueForm.value, createValueForm.label, createValueForm.parent_id); }}>Create</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={!!editValue} onClose={() => setEditValue(null)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Lookup Value</DialogTitle>
        <DialogContent>
          <TextField autoFocus margin="dense" label="Value" fullWidth value={editValue?.value || ''} onChange={(e) => setEditValue((s) => s ? ({ ...s, value: e.target.value }) : s)} />
          <TextField margin="dense" label="Label" fullWidth value={editValue?.label || ''} onChange={(e) => setEditValue((s) => s ? ({ ...s, label: e.target.value }) : s)} />
          <FormControl fullWidth sx={{ mt: 1 }}>
            <InputLabel id="parent-select-label">Parent</InputLabel>
            <Select
              labelId="parent-select-label"
              value={editValue?.parent_id || ''}
              label="Parent"
              onChange={(e) => setEditValue((s) => s ? ({ ...s, parent_id: e.target.value || null }) : s)}
            >
              <MenuItem value="">None</MenuItem>
              {(allValues || []).map((opt: any) => opt.id !== editValue?.id && (<MenuItem key={opt.id} value={opt.id}>{opt.name}</MenuItem>))}
            </Select>
          </FormControl>
          {valueFormError && <FormHelperText error>{valueFormError}</FormHelperText>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditValue(null)}>Cancel</Button>
          <Button variant="contained" onClick={() => { setValueFormError(null); if (!editValue || !editValue.value) { setValueFormError('Value required'); return; } handleUpdateValue(editValue.id, editValue.value, editValue.label, editValue.parent_id); }}>Save</Button>
        </DialogActions>
      </Dialog>
    </>
  );
}