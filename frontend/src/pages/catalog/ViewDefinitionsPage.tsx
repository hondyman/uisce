import React, { useMemo, useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  TextField,
  InputAdornment,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControlLabel,
  Switch,
  Autocomplete,
  Select,
  MenuItem,
  InputLabel,
  FormControl,
  Divider,
  Stack,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { CoreIcon, CustomIcon } from '../../components/common/CoreCustomIcons';

import {
  useViewDefinitions,
  useCreateViewDefinition,
  useUpdateViewDefinition,
  useDeleteViewDefinition,
  ViewDefinition,
  ViewDefinitionConfig,
  ViewGroupingRule,
} from '../../api/viewDefinitions';
import { useNodeTypes } from '../../api/nodeTypes';
import { useEdgeTypes } from '../../api/edgeTypes';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';

interface FormState {
  view_key: string;
  display_name: string;
  description: string;
  is_active: boolean;
  defaultInclude: boolean;
  includedNodeTypes: string[];
  excludedNodeTypes: string[];
  includedEdgeTypes: string[];
  excludedEdgeTypes: string[];
  grouping: ViewGroupingRule[];
  layoutAlgorithm: 'dagre' | 'bfs-level';
  layoutDirection: 'LR' | 'TB';
  assignedCatalogNodeTypes: string[];
  assignedBoSubtypes: string[];
}

const emptyForm: FormState = {
  view_key: '',
  display_name: '',
  description: '',
  is_active: true,
  defaultInclude: true,
  includedNodeTypes: [],
  excludedNodeTypes: [],
  includedEdgeTypes: [],
  excludedEdgeTypes: [],
  grouping: [],
  layoutAlgorithm: 'dagre',
  layoutDirection: 'LR',
  assignedCatalogNodeTypes: [],
  assignedBoSubtypes: [],
};

function configToForm(vd: ViewDefinition): FormState {
  const config = vd.config || {};
  const nodeTypes = config.typePolicy?.nodeTypes || {};
  const edgeTypes = config.typePolicy?.edgeTypes || {};
  return {
    view_key: vd.view_key,
    display_name: vd.display_name,
    description: vd.description || '',
    is_active: vd.is_active,
    defaultInclude: config.typePolicy?.defaultInclude ?? true,
    includedNodeTypes: Object.entries(nodeTypes).filter(([, v]) => v === 'include').map(([k]) => k),
    excludedNodeTypes: Object.entries(nodeTypes).filter(([, v]) => v === 'exclude').map(([k]) => k),
    includedEdgeTypes: Object.entries(edgeTypes).filter(([, v]) => v === 'include').map(([k]) => k),
    excludedEdgeTypes: Object.entries(edgeTypes).filter(([, v]) => v === 'exclude').map(([k]) => k),
    grouping: config.grouping || [],
    layoutAlgorithm: config.layout?.algorithm || 'dagre',
    layoutDirection: (config.layout?.direction as 'LR' | 'TB') || 'LR',
    assignedCatalogNodeTypes: config.assignedAssetTypes?.catalogNodeTypes || [],
    assignedBoSubtypes: config.assignedAssetTypes?.boSubtypes || [],
  };
}

function formToConfig(form: FormState): ViewDefinitionConfig {
  const nodeTypes: Record<string, 'include' | 'exclude'> = {};
  form.includedNodeTypes.forEach((t) => (nodeTypes[t] = 'include'));
  form.excludedNodeTypes.forEach((t) => (nodeTypes[t] = 'exclude'));

  const edgeTypes: Record<string, 'include' | 'exclude'> = {};
  form.includedEdgeTypes.forEach((t) => (edgeTypes[t] = 'include'));
  form.excludedEdgeTypes.forEach((t) => (edgeTypes[t] = 'exclude'));

  return {
    typePolicy: { defaultInclude: form.defaultInclude, nodeTypes, edgeTypes },
    grouping: form.grouping,
    layout: { algorithm: form.layoutAlgorithm, direction: form.layoutDirection },
    assignedAssetTypes: {
      catalogNodeTypes: form.assignedCatalogNodeTypes,
      boSubtypes: form.assignedBoSubtypes,
    },
  };
}

export const ViewDefinitionsPage: React.FC = () => {
  const [search, setSearch] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);

  const { data: views, isLoading } = useViewDefinitions();
  const { data: nodeTypes } = useNodeTypes();
  const { data: edgeTypes } = useEdgeTypes();
  const createMutation = useCreateViewDefinition();
  const updateMutation = useUpdateViewDefinition();
  const deleteMutation = useDeleteViewDefinition();
  const confirm = useConfirm();
  const notification = useNotification();

  const nodeTypeNames = useMemo(() => (nodeTypes || []).map((t) => t.catalog_type_name), [nodeTypes]);
  const edgeTypeNames = useMemo(() => (edgeTypes || []).map((t) => t.edge_type_name), [edgeTypes]);

  const filteredViews = (views || []).filter(
    (v) =>
      v.display_name.toLowerCase().includes(search.toLowerCase()) ||
      v.view_key.toLowerCase().includes(search.toLowerCase())
  );

  const openCreate = () => {
    setEditingId(null);
    setForm(emptyForm);
    setDialogOpen(true);
  };

  const openEdit = (vd: ViewDefinition) => {
    setEditingId(vd.id);
    setForm(configToForm(vd));
    setDialogOpen(true);
  };

  const handleSave = async () => {
    if (!form.view_key || !form.display_name) {
      notification.error('View key and display name are required');
      return;
    }
    setSaving(true);
    try {
      const config = formToConfig(form);
      if (editingId) {
        await updateMutation.mutateAsync({
          id: editingId,
          display_name: form.display_name,
          description: form.description,
          is_active: form.is_active,
          config,
        });
      } else {
        await createMutation.mutateAsync({
          view_key: form.view_key,
          display_name: form.display_name,
          description: form.description,
          config,
        });
      }
      setDialogOpen(false);
    } catch (err) {
      console.error('Failed to save view definition', err);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (vd: ViewDefinition) => {
    const ok = await confirm({
      title: 'Delete view',
      description: `Delete the view "${vd.display_name}"? This cannot be undone.`,
    });
    if (!ok) return;
    await deleteMutation.mutateAsync(vd.id);
  };

  const addGroupingRule = () => {
    setForm((f) => ({
      ...f,
      grouping: [...f.grouping, { childNodeType: '', clusterLabel: '', defaultCollapsed: true, collapseThreshold: 15 }],
    }));
  };

  const updateGroupingRule = (index: number, patch: Partial<ViewGroupingRule>) => {
    setForm((f) => ({
      ...f,
      grouping: f.grouping.map((r, i) => (i === index ? { ...r, ...patch } : r)),
    }));
  };

  const removeGroupingRule = (index: number) => {
    setForm((f) => ({ ...f, grouping: f.grouping.filter((_, i) => i !== index) }));
  };

  return (
    <Box sx={{ p: 3 }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 2 }}>
        <Typography variant="h5" fontWeight={700}>
          Graph Views
        </Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={openCreate}>
          New View
        </Button>
      </Stack>

      <TextField
        placeholder="Search views..."
        size="small"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        sx={{ mb: 2, width: 320 }}
        InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }}
      />

      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>View</TableCell>
              <TableCell>Key</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Status</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {!isLoading &&
              filteredViews.map((vd) => (
                <TableRow key={vd.id} hover>
                  <TableCell>
                    <Typography variant="body2" fontWeight={600}>
                      {vd.display_name}
                    </Typography>
                    {vd.description && (
                      <Typography variant="caption" color="text.secondary">
                        {vd.description}
                      </Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    <code>{vd.view_key}</code>
                  </TableCell>
                  <TableCell>
                    {vd.is_core ? (
                      <Chip icon={<CoreIcon />} label="Core" size="small" variant="outlined" />
                    ) : (
                      <Chip icon={<CustomIcon />} label="Custom" size="small" variant="outlined" color="secondary" />
                    )}
                  </TableCell>
                  <TableCell>
                    <Chip label={vd.is_active ? 'Active' : 'Inactive'} size="small" color={vd.is_active ? 'success' : 'default'} />
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title={vd.is_core ? 'Core views are read-only' : 'Edit'}>
                      <span>
                        <IconButton size="small" onClick={() => openEdit(vd)} disabled={vd.is_core}>
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </span>
                    </Tooltip>
                    <Tooltip title={vd.is_core ? 'Core views cannot be deleted' : 'Delete'}>
                      <span>
                        <IconButton size="small" onClick={() => handleDelete(vd)} disabled={vd.is_core}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </span>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            {!isLoading && filteredViews.length === 0 && (
              <TableRow>
                <TableCell colSpan={5}>
                  <Typography variant="body2" color="text.secondary" sx={{ py: 3, textAlign: 'center' }}>
                    No views yet. Create one to define a custom graph visualization.
                  </Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>{editingId ? 'Edit View' : 'New View'}</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2.5} sx={{ mt: 1 }}>
            <Stack direction="row" spacing={2}>
              <TextField
                label="View key"
                fullWidth
                value={form.view_key}
                onChange={(e) => setForm({ ...form, view_key: e.target.value })}
                disabled={!!editingId}
                helperText="Stable identifier, e.g. 'semantic_lineage'"
              />
              <TextField
                label="Display name"
                fullWidth
                value={form.display_name}
                onChange={(e) => setForm({ ...form, display_name: e.target.value })}
              />
            </Stack>
            <TextField
              label="Description"
              fullWidth
              multiline
              minRows={2}
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
            />

            <Divider textAlign="left">Type visibility</Divider>
            <FormControlLabel
              control={
                <Switch
                  checked={form.defaultInclude}
                  onChange={(e) => setForm({ ...form, defaultInclude: e.target.checked })}
                />
              }
              label="Include new/unlisted types by default"
            />
            <Stack direction="row" spacing={2}>
              <Autocomplete
                multiple
                fullWidth
                size="small"
                options={nodeTypeNames}
                value={form.includedNodeTypes}
                onChange={(_, value) => setForm({ ...form, includedNodeTypes: value })}
                renderInput={(params) => <TextField {...params} label="Node types to include" />}
              />
              <Autocomplete
                multiple
                fullWidth
                size="small"
                options={nodeTypeNames}
                value={form.excludedNodeTypes}
                onChange={(_, value) => setForm({ ...form, excludedNodeTypes: value })}
                renderInput={(params) => <TextField {...params} label="Node types to exclude" />}
              />
            </Stack>
            <Stack direction="row" spacing={2}>
              <Autocomplete
                multiple
                fullWidth
                size="small"
                options={edgeTypeNames}
                value={form.includedEdgeTypes}
                onChange={(_, value) => setForm({ ...form, includedEdgeTypes: value })}
                renderInput={(params) => <TextField {...params} label="Edge types to include" />}
              />
              <Autocomplete
                multiple
                fullWidth
                size="small"
                options={edgeTypeNames}
                value={form.excludedEdgeTypes}
                onChange={(_, value) => setForm({ ...form, excludedEdgeTypes: value })}
                renderInput={(params) => <TextField {...params} label="Edge types to exclude" />}
              />
            </Stack>

            <Divider textAlign="left">Grouping (for high-fan-out node types)</Divider>
            {form.grouping.map((rule, idx) => (
              <Paper key={idx} variant="outlined" sx={{ p: 1.5 }}>
                <Stack direction="row" spacing={1.5} alignItems="center">
                  <Autocomplete
                    size="small"
                    sx={{ minWidth: 160 }}
                    options={nodeTypeNames}
                    value={rule.childNodeType || null}
                    onChange={(_, value) => updateGroupingRule(idx, { childNodeType: value || '' })}
                    renderInput={(params) => <TextField {...params} label="Child type" />}
                  />
                  <Autocomplete
                    size="small"
                    sx={{ minWidth: 160 }}
                    options={edgeTypeNames}
                    value={rule.parentRelation || null}
                    onChange={(_, value) => updateGroupingRule(idx, { parentRelation: value || '' })}
                    renderInput={(params) => <TextField {...params} label="Parent relation" />}
                  />
                  <TextField
                    size="small"
                    label="Cluster label"
                    value={rule.clusterLabel || ''}
                    onChange={(e) => updateGroupingRule(idx, { clusterLabel: e.target.value })}
                  />
                  <TextField
                    size="small"
                    label="Collapse threshold"
                    type="number"
                    sx={{ width: 140 }}
                    value={rule.collapseThreshold ?? 15}
                    onChange={(e) => updateGroupingRule(idx, { collapseThreshold: Number(e.target.value) })}
                  />
                  <FormControlLabel
                    control={
                      <Switch
                        checked={rule.defaultCollapsed ?? true}
                        onChange={(e) => updateGroupingRule(idx, { defaultCollapsed: e.target.checked })}
                      />
                    }
                    label="Collapsed by default"
                  />
                  <IconButton size="small" onClick={() => removeGroupingRule(idx)}>
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Stack>
              </Paper>
            ))}
            <Button size="small" startIcon={<AddIcon />} onClick={addGroupingRule} sx={{ alignSelf: 'flex-start' }}>
              Add grouping rule
            </Button>

            <Divider textAlign="left">Layout</Divider>
            <Stack direction="row" spacing={2}>
              <FormControl size="small" fullWidth>
                <InputLabel>Algorithm</InputLabel>
                <Select
                  label="Algorithm"
                  value={form.layoutAlgorithm}
                  onChange={(e) => setForm({ ...form, layoutAlgorithm: e.target.value as 'dagre' | 'bfs-level' })}
                >
                  <MenuItem value="dagre">Dagre (layered)</MenuItem>
                  <MenuItem value="bfs-level">BFS levels (upstream/downstream)</MenuItem>
                </Select>
              </FormControl>
              <FormControl size="small" fullWidth>
                <InputLabel>Direction</InputLabel>
                <Select
                  label="Direction"
                  value={form.layoutDirection}
                  onChange={(e) => setForm({ ...form, layoutDirection: e.target.value as 'LR' | 'TB' })}
                >
                  <MenuItem value="LR">Left to right</MenuItem>
                  <MenuItem value="TB">Top to bottom</MenuItem>
                </Select>
              </FormControl>
            </Stack>

            <Divider textAlign="left">Assign to asset types</Divider>
            <Autocomplete
              multiple
              size="small"
              options={nodeTypeNames}
              value={form.assignedCatalogNodeTypes}
              onChange={(_, value) => setForm({ ...form, assignedCatalogNodeTypes: value })}
              renderInput={(params) => <TextField {...params} label="Catalog node types" />}
            />
            <Autocomplete
              multiple
              freeSolo
              size="small"
              options={[]}
              value={form.assignedBoSubtypes}
              onChange={(_, value) => setForm({ ...form, assignedBoSubtypes: value as string[] })}
              renderInput={(params) => (
                <TextField {...params} label="Business Object subtypes" placeholder="Type a subtype and press Enter" />
              )}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSave} disabled={saving}>
            {editingId ? 'Save Changes' : 'Create View'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
