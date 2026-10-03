import React, { useState, useMemo } from 'react';
import { Box, Typography, TextField, Chip, IconButton, useTheme, Button, Dialog, DialogTitle, DialogContent, DialogActions, FormControlLabel, Switch, Autocomplete, Menu, MenuItem, Stack } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import PaletteIcon from '@mui/icons-material/Palette';
import { useNavigate } from 'react-router-dom';
import { useEdgeTypes, useDeleteEdgeType, useCreateEdgeType, type EdgeType } from '../../api/edgeTypes';
import { useUpdateEdgeType } from '../../api/edgeTypes';
import { useNodeTypes } from '../../api/nodeTypes';
import { useTenant } from '../../contexts/TenantContext';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import { ColorPaletteEditor } from '../../components/ColorPaletteEditor';
import CatalogList from '../../components/common/CatalogList';

export const CatalogEdgeTypesPage: React.FC = () => {
  const theme = useTheme();
  const navigate = useNavigate();
  const { tenant } = useTenant();
  const confirm = useConfirm();
  const notification = useNotification();
  const [actionsMenu, setActionsMenu] = useState<{ el: HTMLElement; type: EdgeType } | null>(null);
  const [editingType, setEditingType] = useState<EdgeType | null>(null);
  const [editDescription, setEditDescription] = useState('');
  const [editColor, setEditColor] = useState('');
  const [colorPaletteOpen, setColorPaletteOpen] = useState(false);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ edge_type_name: '', description: '', subjectNodeTypeId: '', objectNodeTypeId: '', isActive: true });
  const { data: edgeTypes, isLoading } = useEdgeTypes();
  const { data: nodeTypes } = useNodeTypes('');
  const updateMutation = useUpdateEdgeType();
  const deleteMutation = useDeleteEdgeType();
  const createMutation = useCreateEdgeType();

  // Get all used colors to avoid conflicts
  const usedColors = useMemo(() => {
    return edgeTypes
      ?.filter(type => type.config?.color)
      .map(type => type.config!.color) || [];
  }, [edgeTypes]);

  const handleEditOpen = (type: EdgeType) => {
    setEditingType(type);
    setEditDescription(type.description || '');
    setEditColor(type.config?.color || '');
  };

  const handleEditSave = async () => {
    if (!editingType || !tenant?.id) return;
    try {
      await updateMutation.mutateAsync({
        id: editingType.id,
        tenantId: tenant.id,
        data: {
          description: editDescription,
          config: {
            ...editingType.config,
            color: editColor,
          },
        },
      });
      setEditingType(null);
    } catch (error) {
      console.error('Failed to update edge type:', error);
    }
  };

  const handleDelete = async (type: EdgeType) => {
    if (!tenant?.id) return;
    const confirmed = await confirm({
      title: 'Delete Edge Type',
      description: `Are you sure you want to delete "${type.edge_type_name}"? This action cannot be undone.`,
    });
    if (!confirmed) return;

    try {
      await deleteMutation.mutateAsync({
        id: type.id,
        tenantId: tenant.id,
      });
      notification.success(`Edge type "${type.edge_type_name}" deleted successfully`);
    } catch (error) {
      notification.error(`Failed to delete edge type: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleCreateSubmit = async () => {
    if (!tenant?.id) return;
    if (!createForm.edge_type_name.trim() || !createForm.subjectNodeTypeId || !createForm.objectNodeTypeId) {
      notification.error('Please fill in all required fields');
      return;
    }

    try {
      await createMutation.mutateAsync({
        tenant_id: tenant.id,
        edge_type_name: createForm.edge_type_name,
        description: createForm.description,
        subject_node_type_id: createForm.subjectNodeTypeId,
        object_node_type_id: createForm.objectNodeTypeId,
        is_active: createForm.isActive,
      });
      notification.success(`Edge type "${createForm.edge_type_name}" created successfully`);
      setIsCreateModalOpen(false);
      setCreateForm({ edge_type_name: '', description: '', subjectNodeTypeId: '', objectNodeTypeId: '', isActive: true });
    } catch (error) {
      notification.error(`Failed to create edge type: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const isDark = theme.palette.mode === 'dark';
  const C = useMemo(() => ({
    bg: isDark ? '#0A0C12' : '#F8FAFC',
    sidebar: isDark ? '#0F1117' : '#F1F5F9',
    panel: isDark ? '#13161E' : '#FFFFFF',
    panelHover: isDark ? '#1A1E2A' : '#F1F5F9',
    border: isDark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.08)',
    borderStrong: isDark ? 'rgba(255,255,255,0.12)' : 'rgba(0,0,0,0.14)',
    accent: '#6366F1',
    accentDim: isDark ? 'rgba(99,102,241,0.15)' : 'rgba(99,102,241,0.08)',
    accentGlow: '0 0 20px rgba(99,102,241,0.4)',
    text: isDark ? '#E2E8F0' : '#0F172A',
    textMuted: isDark ? '#8892A4' : '#64748B',
    success: '#10B981',
    warning: '#F59E0B',
    danger: '#EF4444',
    purple: '#A78BFA',
    teal: '#2DD4BF',
    blue: '#60A5FA',
    orange: '#FB923C',
  }), [isDark]);

  const totalTypesCount = edgeTypes?.length || 0;
  const cdmCount = edgeTypes?.filter(e => e.edge_type_name.startsWith('CDM')).length || 0;
  const activeCount = edgeTypes?.filter(e => e.is_active).length || 0;

  return (
    <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto', minHeight: '100vh', color: C.text, bgcolor: C.bg }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3, flexWrap: 'wrap', gap: 2 }}>
        <Box>
          <Typography variant="h4" fontWeight="bold" sx={{ color: C.text, letterSpacing: '-0.02em', mb: 0.5 }}>
            Edge Types
          </Typography>
          <Typography variant="body2" sx={{ color: C.textMuted }}>
            Browse and manage the relationship definitions of your data catalog.
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
          {/* Outlined Summary Badges */}
          {!isLoading && (
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '4px 10px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.accent, background: isDark ? 'rgba(99,102,241,0.12)' : 'rgba(99,102,241,0.08)',
                border: `1px solid ${C.accent}44`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                {totalTypesCount} Types
              </span>
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '4px 10px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.purple, background: isDark ? 'rgba(167,139,250,0.12)' : 'rgba(167,139,250,0.08)',
                border: `1px solid ${C.purple}44`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                {cdmCount} CDM
              </span>
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '4px 10px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.success, background: isDark ? 'rgba(16,185,129,0.12)' : 'rgba(16,185,129,0.08)',
                border: `1px solid ${C.success}44`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                {activeCount} Active
              </span>
            </Box>
          )}

          <Button 
            variant="contained" 
            startIcon={<AddIcon />}
            onClick={() => setIsCreateModalOpen(true)}
            sx={{ 
              borderRadius: 2, px: 2.5, py: 0.8,
              bgcolor: C.accent,
              color: '#FFFFFF',
              boxShadow: isDark ? C.accentGlow : 'none',
              '&:hover': { bgcolor: '#4F46E5' }
            }}
          >
            Create Type
          </Button>
        </Box>
      </Box>

      <CatalogList<EdgeType>
        items={edgeTypes ?? []}
        getId={(t) => t.id}
        getTitle={(t) => t.edge_type_name}
        getSubtitle={(t) => `${t.subject_node_type_name || 'Subject'} → ${t.object_node_type_name || 'Object'}`}
        getDescription={(t) => t.description || undefined}
        isCore={(t) => t.type === 'core'}
        storageKey="edge-types-view"
        searchPlaceholder="Search edge types by predicate or description..."
        loading={isLoading}
        emptyMessage="No edge types yet."
        noMatchMessage="No edge types match your search."
        onOpen={(t) => navigate(`/catalog/edge-types/${t.id}`)}
        onActions={(t, el) => setActionsMenu({ el, type: t })}
        createLabel="Create Type"
        onCreate={() => setIsCreateModalOpen(true)}
        renderTileMeta={(t) => (
          <Stack direction="row" spacing={0.5} useFlexGap flexWrap="wrap" alignItems="center">
            <Chip size="small" color={t.is_active ? 'success' : 'default'} variant={t.is_active ? 'filled' : 'outlined'} label={t.is_active ? 'Active' : 'Inactive'} />
            {t.config?.color && <Box sx={{ width: 14, height: 14, borderRadius: '50%', bgcolor: t.config.color, border: `1px solid ${C.border}` }} />}
            <Typography variant="caption" color="text.secondary">{new Date(t.created_at).toLocaleDateString()}</Typography>
          </Stack>
        )}
        columns={[
          { key: 'description', header: 'Description', render: (t) => t.description || '-' },
          { key: 'relationship', header: 'Relationship', render: (t) => `${t.subject_node_type_name || 'Unknown'} → ${t.object_node_type_name || 'Unknown'}` },
          { key: 'status', header: 'Status', render: (t) => <Chip size="small" color={t.is_active ? 'success' : 'default'} variant={t.is_active ? 'filled' : 'outlined'} label={t.is_active ? 'Active' : 'Inactive'} /> },
          { key: 'created', header: 'Created', render: (t) => new Date(t.created_at).toLocaleDateString() },
        ]}
      />
      <Menu anchorEl={actionsMenu?.el} open={!!actionsMenu} onClose={() => setActionsMenu(null)}>
        <MenuItem onClick={() => { const t = actionsMenu!.type; setActionsMenu(null); navigate(`/catalog/edge-types/${t.id}`); }}>View Details</MenuItem>
        <MenuItem onClick={() => { const t = actionsMenu!.type; setActionsMenu(null); handleEditOpen(t); }}>Edit</MenuItem>
        <MenuItem sx={{ color: 'error.main' }} onClick={() => { const t = actionsMenu!.type; setActionsMenu(null); void handleDelete(t); }}>Delete</MenuItem>
      </Menu>

      {/* Edit Dialog */}
      <Dialog 
        open={!!editingType} 
        onClose={() => setEditingType(null)} 
        maxWidth="sm" 
        fullWidth
        PaperProps={{
          sx: {
            bgcolor: C.panel,
            color: C.text,
            border: `1px solid ${C.border}`,
            borderRadius: 3,
          }
        }}
      >
        <DialogTitle sx={{ borderBottom: `1px solid ${C.border}`, fontWeight: 700 }}>Edit Edge Type</DialogTitle>
        <DialogContent sx={{ pt: 3, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Box sx={{ mt: 1 }}>
            <Typography variant="subtitle2" fontWeight="bold" sx={{ color: C.textMuted, mb: 0.5 }}>
              Predicate
            </Typography>
            <Typography variant="body1" sx={{ color: C.text, fontWeight: 600, fontFamily: 'monospace' }}>
              {editingType?.edge_type_name}
            </Typography>
          </Box>
          <TextField
            label="Description"
            multiline
            rows={3}
            fullWidth
            value={editDescription}
            onChange={(e) => setEditDescription(e.target.value)}
            placeholder="Add a description for this edge type..."
            InputProps={{
              sx: {
                bgcolor: isDark ? '#0A0C12' : '#F8FAFC',
                color: C.text,
                '& fieldset': { borderColor: C.border },
              }
            }}
          />
          <Box>
            <Typography variant="subtitle2" fontWeight="bold" sx={{ color: C.textMuted, mb: 0.5 }}>
              Color Accent
            </Typography>
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', mb: 1 }}>
              <Box
                sx={{
                  width: 38,
                  height: 38,
                  borderRadius: 1.5,
                  bgcolor: editColor || '#ccc',
                  border: `2px solid ${C.border}`,
                }}
              />
              <TextField
                type="text"
                placeholder="#6366F1"
                value={editColor}
                onChange={(e) => setEditColor(e.target.value)}
                size="small"
                sx={{ flex: 1 }}
                InputProps={{
                  sx: {
                    bgcolor: isDark ? '#0A0C12' : '#F8FAFC',
                    color: C.text,
                    fontFamily: 'monospace',
                    '& fieldset': { borderColor: C.border },
                  }
                }}
              />
              <IconButton
                size="small"
                onClick={() => setColorPaletteOpen(true)}
                sx={{ border: `1px solid ${C.border}`, borderRadius: 1.5, color: C.accent, bgcolor: C.accentDim }}
              >
                <PaletteIcon fontSize="small" />
              </IconButton>
            </Box>
            <Typography variant="caption" sx={{ color: C.textMuted, display: 'block' }}>
              Select a color to distinguish this relationship type across graph views.
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2, borderTop: `1px solid ${C.border}` }}>
          <Button onClick={() => setEditingType(null)} sx={{ color: C.textMuted }}>Cancel</Button>
          <Button 
            onClick={handleEditSave} 
            variant="contained"
            disabled={updateMutation.isPending}
            sx={{ bgcolor: C.accent, color: '#fff', '&:hover': { bgcolor: '#4F46E5' } }}
          >
            Save Changes
          </Button>
        </DialogActions>
      </Dialog>

      {/* Color Palette Editor */}
      <ColorPaletteEditor
        open={colorPaletteOpen}
        onClose={() => setColorPaletteOpen(false)}
        usedColors={usedColors.filter(c => c !== editColor)}
        onColorSelect={(color) => setEditColor(color)}
      />

      {/* Create Edge Type Dialog */}
      <Dialog 
        open={isCreateModalOpen} 
        onClose={() => setIsCreateModalOpen(false)} 
        maxWidth="sm" 
        fullWidth
        PaperProps={{
          sx: {
            bgcolor: C.panel,
            color: C.text,
            border: `1px solid ${C.border}`,
            borderRadius: 3,
          }
        }}
      >
        <DialogTitle sx={{ borderBottom: `1px solid ${C.border}`, fontWeight: 700 }}>Create Edge Type</DialogTitle>
        <DialogContent sx={{ pt: 3, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label="Predicate"
            placeholder="e.g., has_semantic_meaning"
            value={createForm.edge_type_name}
            onChange={(e) => setCreateForm({ ...createForm, edge_type_name: e.target.value })}
            fullWidth
            required
            sx={{ mt: 1 }}
            InputProps={{
              sx: {
                bgcolor: isDark ? '#0A0C12' : '#F8FAFC',
                color: C.text,
                '& fieldset': { borderColor: C.border },
              }
            }}
          />
          <TextField
            label="Description"
            placeholder="e.g., Semantic relationship between entities"
            multiline
            rows={3}
            value={createForm.description}
            onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
            fullWidth
            InputProps={{
              sx: {
                bgcolor: isDark ? '#0A0C12' : '#F8FAFC',
                color: C.text,
                '& fieldset': { borderColor: C.border },
              }
            }}
          />
          <Autocomplete
            options={nodeTypes || []}
            getOptionLabel={(option) => `${option.catalog_type_name} (${option.id.substring(0, 8)}...)`}
            value={nodeTypes?.find(nt => nt.id === createForm.subjectNodeTypeId) || null}
            onChange={(_, newValue) => setCreateForm({ ...createForm, subjectNodeTypeId: newValue?.id || '' })}
            renderInput={(params) => (
              <TextField 
                {...params} 
                label="Subject Node Type" 
                placeholder="Search node types..." 
                required 
                InputProps={{
                  ...params.InputProps,
                  sx: {
                    bgcolor: isDark ? '#0A0C12' : '#F8FAFC',
                    color: C.text,
                    '& fieldset': { borderColor: C.border },
                  }
                }}
              />
            )}
            fullWidth
            isOptionEqualToValue={(option, value) => option.id === value.id}
          />
          <Autocomplete
            options={nodeTypes || []}
            getOptionLabel={(option) => `${option.catalog_type_name} (${option.id.substring(0, 8)}...)`}
            value={nodeTypes?.find(nt => nt.id === createForm.objectNodeTypeId) || null}
            onChange={(_, newValue) => setCreateForm({ ...createForm, objectNodeTypeId: newValue?.id || '' })}
            renderInput={(params) => (
              <TextField 
                {...params} 
                label="Object Node Type" 
                placeholder="Search node types..." 
                required 
                InputProps={{
                  ...params.InputProps,
                  sx: {
                    bgcolor: isDark ? '#0A0C12' : '#F8FAFC',
                    color: C.text,
                    '& fieldset': { borderColor: C.border },
                  }
                }}
              />
            )}
            fullWidth
            isOptionEqualToValue={(option, value) => option.id === value.id}
          />
          <FormControlLabel
            control={
              <Switch
                checked={createForm.isActive}
                onChange={(e) => setCreateForm({ ...createForm, isActive: e.target.checked })}
                sx={{
                  '& .MuiSwitch-switchBase.Mui-checked': {
                    color: C.accent,
                  },
                  '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': {
                    backgroundColor: C.accent,
                  },
                }}
              />
            }
            label={<Typography sx={{ color: C.text, fontSize: '0.9rem' }}>Active</Typography>}
          />
        </DialogContent>
        <DialogActions sx={{ p: 2, borderTop: `1px solid ${C.border}` }}>
          <Button onClick={() => setIsCreateModalOpen(false)} sx={{ color: C.textMuted }}>Cancel</Button>
          <Button 
            onClick={handleCreateSubmit}
            variant="contained"
            disabled={createMutation.isPending}
            sx={{ bgcolor: C.accent, color: '#fff', '&:hover': { bgcolor: '#4F46E5' } }}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

