import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  TableContainer,
  Paper,
  IconButton,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Chip,
  Stack,
  Alert,
  CircularProgress,
  Tooltip,
  Autocomplete,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  Star as DefaultIcon,
  StarOutline as NotDefaultIcon,
  Link as LinkIcon,
} from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext';

// ============================================================================
// Types
// ============================================================================

interface PhysicalMapping {
  mapping_id?: string;
  context_type: 'table' | 'business_object' | 'datasource' | 'tenant';
  context_id: string;
  table_id: string;
  table_name?: string;
  column_id: string;
  column_name?: string;
  expression?: string;
  priority: number;
  is_default?: boolean;
  status?: string;
}

interface TieBreakerConfig {
  strategy: 'precedence' | 'latest_timestamp' | 'custom' | 'priority';
  precedence?: string[];
  timestamp_column?: string;
  custom_expression?: string;
  description?: string;
}

interface SemanticTermMappingData {
  term_id: string;
  term_name?: string;
  semantic_type: string;
  mappings: PhysicalMapping[];
  default_mapping_id?: string;
  tie_breaker?: TieBreakerConfig;
}

interface SemanticTermPhysicalMappingEditorProps {
  termId: string;
  termName?: string;
}

// ============================================================================
// Component
// ============================================================================

export const SemanticTermPhysicalMappingEditor: React.FC<SemanticTermPhysicalMappingEditorProps> = ({
  termId,
  termName,
}) => {
  const { tenant, datasource } = useTenant();
  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || '';

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<SemanticTermMappingData | null>(null);

  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [editMapping, setEditMapping] = useState<PhysicalMapping | null>(null);

  const [availableTables, setAvailableTables] = useState<any[]>([]);
  const [availableColumns, setAvailableColumns] = useState<any[]>([]);

  const [newMapping, setNewMapping] = useState<PhysicalMapping>({
    context_type: 'table',
    context_id: '',
    table_id: '',
    column_id: '',
    priority: 0,
    is_default: false,
  });

  // Load mappings
  const loadMappings = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/semantic-term/${termId}/mappings`, {
        headers: { 'X-Tenant-ID': tenantId },
      });

      if (!response.ok) {
        throw new Error('Failed to load mappings');
      }

      const result = await response.json();
      setData(result);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Load available tables
  const loadTables = async () => {
    try {
      const response = await fetch(`/api/catalog/nodes?type=table&datasource_id=${datasourceId}`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (response.ok) {
        const tables = await response.json();
        setAvailableTables(tables || []);
      }
    } catch (err) {
      console.error('Failed to load tables:', err);
    }
  };

  // Load columns for selected table
  const loadColumns = async (tableId: string) => {
    if (!tableId) {
      setAvailableColumns([]);
      return;
    }

    try {
      const response = await fetch(`/api/catalog/nodes/${tableId}/columns`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (response.ok) {
        const columns = await response.json();
        setAvailableColumns(columns || []);
      }
    } catch (err) {
      console.error('Failed to load columns:', err);
    }
  };

  useEffect(() => {
    if (termId && tenantId) {
      loadMappings();
      loadTables();
    }
  }, [termId, tenantId]);

  const handleTableChange = (tableId: string) => {
    setNewMapping((prev) => ({
      ...prev,
      table_id: tableId,
      context_id: tableId,
      column_id: '',
    }));
    loadColumns(tableId);
  };

  const handleSaveMapping = async () => {
    try {
      const method = editMapping ? 'PUT' : 'POST';
      const url = editMapping
        ? `/api/semantic-term/${termId}/mappings/${editMapping.mapping_id}`
        : `/api/semantic-term/${termId}/mappings`;

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify(newMapping),
      });

      if (!response.ok) {
        throw new Error('Failed to save mapping');
      }

      setAddDialogOpen(false);
      setEditMapping(null);
      setNewMapping({
        context_type: 'table',
        context_id: '',
        table_id: '',
        column_id: '',
        priority: 0,
        is_default: false,
      });
      loadMappings();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleDeleteMapping = async (mappingId: string) => {
    if (!confirm('Delete this mapping?')) return;

    try {
      const response = await fetch(`/api/semantic-term/${termId}/mappings/${mappingId}`, {
        method: 'DELETE',
        headers: { 'X-Tenant-ID': tenantId },
      });

      if (!response.ok) {
        throw new Error('Failed to delete mapping');
      }

      loadMappings();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleSetDefault = async (mappingId: string) => {
    try {
      const response = await fetch(`/api/semantic-term/${termId}/mappings/${mappingId}/default`, {
        method: 'PUT',
        headers: { 'X-Tenant-ID': tenantId },
      });

      if (!response.ok) {
        throw new Error('Failed to set default');
      }

      loadMappings();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleEdit = (mapping: PhysicalMapping) => {
    setEditMapping(mapping);
    setNewMapping({ ...mapping });
    loadColumns(mapping.table_id);
    setAddDialogOpen(true);
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Box>
          <Typography variant="h6">
            <LinkIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
            Physical Mappings
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {termName || termId}
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => {
            setEditMapping(null);
            setNewMapping({
              context_type: 'table',
              context_id: '',
              table_id: '',
              column_id: '',
              priority: 0,
              is_default: data?.mappings.length === 0,
            });
            setAddDialogOpen(true);
          }}
        >
          Add Mapping
        </Button>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {/* Tie-Breaker Info */}
      {data?.mappings && data.mappings.length > 1 && (
        <Alert severity="info" sx={{ mb: 2 }}>
          Multiple mappings exist. Using{' '}
          <strong>{data.tie_breaker?.strategy || 'priority'}</strong> strategy for resolution.
          {data.tie_breaker?.description && (
            <Typography variant="caption" sx={{ display: 'block' }}>
              {data.tie_breaker.description}
            </Typography>
          )}
        </Alert>
      )}

      {/* Mappings Table */}
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell sx={{ fontWeight: 600 }}>Context</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Table</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Column</TableCell>
              <TableCell sx={{ fontWeight: 600 }} align="center">Priority</TableCell>
              <TableCell sx={{ fontWeight: 600 }} align="center">Default</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {data?.mappings.map((m) => (
              <TableRow
                key={m.mapping_id}
                sx={{ '&:hover': { bgcolor: 'action.hover' } }}
              >
                <TableCell>
                  <Chip
                    label={m.context_type}
                    size="small"
                    variant="outlined"
                    color={m.context_type === 'business_object' ? 'primary' : 'default'}
                  />
                </TableCell>
                <TableCell>
                  <Typography variant="body2">{m.table_name || m.table_id}</Typography>
                </TableCell>
                <TableCell>
                  <Typography variant="body2">{m.column_name || m.column_id}</Typography>
                  {m.expression && (
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                      expr: {m.expression}
                    </Typography>
                  )}
                </TableCell>
                <TableCell align="center">
                  <Chip label={m.priority} size="small" variant="outlined" />
                </TableCell>
                <TableCell align="center">
                  <Tooltip title={m.is_default ? 'Default mapping' : 'Set as default'}>
                    <IconButton
                      size="small"
                      color={m.is_default ? 'primary' : 'default'}
                      onClick={() => !m.is_default && handleSetDefault(m.mapping_id!)}
                    >
                      {m.is_default ? <DefaultIcon /> : <NotDefaultIcon />}
                    </IconButton>
                  </Tooltip>
                </TableCell>
                <TableCell align="right">
                  <Tooltip title="Edit">
                    <IconButton size="small" onClick={() => handleEdit(m)}>
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Delete">
                    <IconButton
                      size="small"
                      color="error"
                      onClick={() => handleDeleteMapping(m.mapping_id!)}
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </TableCell>
              </TableRow>
            ))}

            {(!data?.mappings || data.mappings.length === 0) && (
              <TableRow>
                <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                  <Typography color="text.secondary">
                    No physical mappings defined. Add a mapping to connect this term to a column.
                  </Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Add/Edit Dialog */}
      <Dialog open={addDialogOpen} onClose={() => setAddDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editMapping ? 'Edit Mapping' : 'Add Physical Mapping'}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <FormControl fullWidth>
              <InputLabel>Context Type</InputLabel>
              <Select
                value={newMapping.context_type}
                label="Context Type"
                onChange={(e) =>
                  setNewMapping((prev) => ({
                    ...prev,
                    context_type: e.target.value as any,
                  }))
                }
              >
                <MenuItem value="table">Table</MenuItem>
                <MenuItem value="business_object">Business Object</MenuItem>
                <MenuItem value="datasource">Datasource</MenuItem>
                <MenuItem value="tenant">Tenant</MenuItem>
              </Select>
            </FormControl>

            <Autocomplete
              options={availableTables}
              getOptionLabel={(opt) => opt.node_name || opt.id}
              value={availableTables.find((t) => t.id === newMapping.table_id) || null}
              onChange={(_, value) => handleTableChange(value?.id || '')}
              renderInput={(params) => <TextField {...params} label="Table" />}
            />

            <Autocomplete
              options={availableColumns}
              getOptionLabel={(opt) => opt.node_name || opt.column_name || opt.id}
              value={availableColumns.find((c) => c.id === newMapping.column_id) || null}
              onChange={(_, value) =>
                setNewMapping((prev) => ({ ...prev, column_id: value?.id || '' }))
              }
              renderInput={(params) => <TextField {...params} label="Column" />}
              disabled={!newMapping.table_id}
            />

            <TextField
              label="Expression (optional)"
              value={newMapping.expression || ''}
              onChange={(e) =>
                setNewMapping((prev) => ({ ...prev, expression: e.target.value }))
              }
              fullWidth
              helperText="For derived values, e.g., COALESCE(col1, col2)"
            />

            <TextField
              label="Priority"
              type="number"
              value={newMapping.priority}
              onChange={(e) =>
                setNewMapping((prev) => ({ ...prev, priority: parseInt(e.target.value) || 0 }))
              }
              fullWidth
              helperText="Lower = higher priority for 1:M resolution"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAddDialogOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleSaveMapping}
            disabled={!newMapping.table_id || !newMapping.column_id}
          >
            Save
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default SemanticTermPhysicalMappingEditor;
