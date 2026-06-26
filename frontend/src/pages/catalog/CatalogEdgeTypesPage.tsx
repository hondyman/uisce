import React, { useState, useMemo } from 'react';
import { 
  Box, Typography, Grid, Paper, TextField, InputAdornment, 
  Card, CardContent, Chip, IconButton, useTheme, alpha, Skeleton,
  Button, Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Dialog, DialogTitle, DialogContent, DialogActions, FormControlLabel, Switch, Autocomplete
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import FilterListIcon from '@mui/icons-material/FilterList';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import AddIcon from '@mui/icons-material/Add';
import ViewAgendaIcon from '@mui/icons-material/ViewAgenda';
import ViewComfyIcon from '@mui/icons-material/ViewComfy';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import PaletteIcon from '@mui/icons-material/Palette';
import { useNavigate } from 'react-router-dom';
import { useEdgeTypes, useDeleteEdgeType, useCreateEdgeType, type EdgeType } from '../../api/edgeTypes';
import { useUpdateEdgeType } from '../../api/edgeTypes';
import { useNodeTypes } from '../../api/nodeTypes';
import { useTenant } from '../../contexts/TenantContext';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import { ColorPaletteEditor } from '../../components/ColorPaletteEditor';

export const CatalogEdgeTypesPage: React.FC = () => {
  const theme = useTheme();
  const navigate = useNavigate();
  const { tenant } = useTenant();
  const confirm = useConfirm();
  const notification = useNotification();
  const [search, setSearch] = useState('');
  const [viewMode, setViewMode] = useState<'tiles' | 'table'>('tiles');
  const [editingType, setEditingType] = useState<EdgeType | null>(null);
  const [editDescription, setEditDescription] = useState('');
  const [editColor, setEditColor] = useState('');
  const [colorPaletteOpen, setColorPaletteOpen] = useState(false);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ edge_type_name: '', description: '', subjectNodeTypeId: '', objectNodeTypeId: '', isActive: true });
  const { data: edgeTypes, isLoading } = useEdgeTypes(tenant?.id || '');
  const { data: nodeTypes } = useNodeTypes('');
  const updateMutation = useUpdateEdgeType();
  const deleteMutation = useDeleteEdgeType();
  const createMutation = useCreateEdgeType();

  // Get all used colors to avoid conflicts
  const usedColors = useMemo(() => {
    return edgeTypes
      ?.filter(type => type.config?.color)
      .map(type => type.config.color) || [];
  }, [edgeTypes]);

  const filteredTypes = edgeTypes?.filter(t => 
    t.edge_type_name.toLowerCase().includes(search.toLowerCase()) || 
    t.description?.toLowerCase().includes(search.toLowerCase())
  );

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

  return (
    <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
        <Box>
          <Typography variant="h4" fontWeight="bold" gutterBottom>
            Edge Types
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Browse and manage the relationship definitions of your data catalog.
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Box sx={{ display: 'flex', gap: 0.5, border: `1px solid ${theme.palette.divider}`, borderRadius: 2, p: 0.5 }}>
            <IconButton 
              size="small"
              onClick={() => setViewMode('tiles')}
              sx={{ bgcolor: viewMode === 'tiles' ? 'primary.main' : 'transparent', color: viewMode === 'tiles' ? 'white' : 'inherit' }}
            >
              <ViewComfyIcon fontSize="small" />
            </IconButton>
            <IconButton 
              size="small"
              onClick={() => setViewMode('table')}
              sx={{ bgcolor: viewMode === 'table' ? 'primary.main' : 'transparent', color: viewMode === 'table' ? 'white' : 'inherit' }}
            >
              <ViewAgendaIcon fontSize="small" />
            </IconButton>
          </Box>
          <Button 
            variant="contained" 
            startIcon={<AddIcon />}
            onClick={() => setIsCreateModalOpen(true)}
            sx={{ borderRadius: 2, px: 3, py: 1 }}
          >
            Create Type
          </Button>
        </Box>
      </Box>

      {/* Stats/Overview (Mock data for visual depth) */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        {[
          { label: 'Total Types', value: edgeTypes?.length || 0, color: theme.palette.primary.main },
          { label: 'CDM Relations', value: edgeTypes?.filter(e => e.edge_type_name.startsWith('CDM')).length || 0, color: theme.palette.info.main },
          { label: 'Active Edges', value: '1.2k', color: theme.palette.success.main }, // Placeholder
        ].map((stat, i) => (
          <Grid item xs={12} md={4} key={i}>
            <Paper 
              elevation={0}
              sx={{ 
                p: 3, 
                borderRadius: 4, 
                border: `1px solid ${theme.palette.divider}`,
                background: `linear-gradient(135deg, ${alpha(stat.color, 0.05)} 0%, ${alpha(theme.palette.background.paper, 1)} 100%)`
              }}
            >
              <Typography variant="overline" color="text.secondary" fontWeight="bold">
                {stat.label}
              </Typography>
              <Typography variant="h3" fontWeight="bold" sx={{ color: stat.color }}>
                {stat.value}
              </Typography>
            </Paper>
          </Grid>
        ))}
      </Grid>

      {/* Search and Filter */}
      <Paper 
        elevation={0} 
        sx={{ 
          p: 2, 
          mb: 4, 
          borderRadius: 3, 
          border: `1px solid ${theme.palette.divider}`,
          display: 'flex',
          alignItems: 'center',
          gap: 2
        }}
      >
        <TextField
          fullWidth
          variant="outlined"
          placeholder="Search edge types..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon color="action" />
              </InputAdornment>
            ),
            sx: { borderRadius: 2 }
          }}
          size="small"
        />
        <IconButton sx={{ border: `1px solid ${theme.palette.divider}`, borderRadius: 2 }}>
          <FilterListIcon />
        </IconButton>
      </Paper>

      {/* Edge Types Grid or Table */}
      {viewMode === 'tiles' ? (
        <Grid container spacing={3}>
          {isLoading ? (
            Array.from({ length: 6 }).map((_, i) => (
              <Grid item xs={12} sm={6} md={4} lg={3} key={i}>
                <Skeleton variant="rectangular" height={200} sx={{ borderRadius: 4 }} />
              </Grid>
            ))
          ) : filteredTypes?.map((type) => {
            const edgeColor = type.config?.color;
            
            return (
              <Grid item xs={12} sm={6} md={4} lg={3} key={type.id}>
                <Card 
                  elevation={0}
                  sx={{ 
                    height: '100%', 
                    borderRadius: 4,
                    border: `1px solid ${theme.palette.divider}`,
                    transition: 'transform 0.2s, box-shadow 0.2s',
                    overflow: 'hidden',
                    '&:hover': {
                      transform: 'translateY(-4px)',
                      boxShadow: theme.shadows[4],
                      borderColor: theme.palette.primary.main
                    }
                  }}
                >
                  {edgeColor && (
                    <Box sx={{ height: 4, bgcolor: edgeColor }} />
                  )}
                  <CardContent sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column' }}>
                    <Box sx={{ display: 'flex', justifyContent: 'flex-start', alignItems: 'center', mb: 2, gap: 0.5 }}>
                      {type.is_active && (
                        <Box 
                          sx={{ 
                            width: 8, 
                            height: 8, 
                            borderRadius: '50%', 
                            bgcolor: theme.palette.success.main,
                            boxShadow: `0 0 8px ${theme.palette.success.main}`
                          }} 
                        />
                      )}
                      {type.type && (
                        <Chip 
                          label={type.type === 'core' ? 'Core' : 'Custom'}
                          size="small"
                          color={type.type === 'core' ? 'primary' : 'warning'}
                          variant={type.type === 'core' ? 'filled' : 'outlined'}
                          sx={{ fontWeight: 600 }}
                        />
                      )}
                    </Box>
                    
                    <Typography variant="h6" fontWeight="bold" gutterBottom noWrap>
                      {type.edge_type_name}
                    </Typography>
                    
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 'auto' }}>
                      <Chip
                        label={type.subject_node_type_name || 'Unknown'}
                        size="small"
                        color="primary"
                        variant="outlined"
                        sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                      />
                      <ArrowForwardIcon sx={{ fontSize: 14, color: 'text.secondary' }} />
                      <Chip
                        label={type.object_node_type_name || 'Unknown'}
                        size="small"
                        color="secondary"
                        variant="outlined"
                        sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                      />
                    </Box>

                    <Box sx={{ mt: 3, pt: 2, borderTop: `1px solid ${theme.palette.divider}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 1 }}>
                      <Typography variant="caption" color="text.secondary">
                        {new Date(type.created_at).toLocaleDateString()}
                      </Typography>
                      <Box sx={{ display: 'flex', gap: 1 }}>
                        <IconButton 
                          size="small"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleEditOpen(type);
                          }}
                          sx={{ color: 'primary.main' }}
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                        <IconButton 
                          size="small"
                          onClick={() => navigate(`/catalog/edge-types/${type.id}`)}
                        >
                          <ArrowForwardIcon fontSize="small" />
                        </IconButton>
                        <IconButton 
                          size="small"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDelete(type);
                          }}
                          sx={{ color: 'error.main' }}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Box>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
            );
          })}
        </Grid>
      ) : (
        <TableContainer component={Paper} elevation={0} sx={{ border: `1px solid ${theme.palette.divider}`, borderRadius: 2 }}>
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: alpha(theme.palette.primary.main, 0.05) }}>
                <TableCell sx={{ fontWeight: 'bold' }}>Predicate</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Description</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Node Types</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Status</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Created</TableCell>
                <TableCell align="right" sx={{ fontWeight: 'bold' }}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 6 }).map((_, j) => (
                      <TableCell key={j}>
                        <Skeleton variant="text" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                filteredTypes?.map((type) => {
                  const edgeColor = type.config?.color;
                  
                  return (
                    <TableRow 
                      key={type.id}
                      sx={{ 
                        '&:hover': { bgcolor: alpha(theme.palette.primary.main, 0.02) },
                        borderLeft: edgeColor ? `4px solid ${edgeColor}` : 'none'
                      }}
                    >
                      <TableCell sx={{ fontWeight: 500 }}>{type.edge_type_name}</TableCell>
                      <TableCell sx={{ maxWidth: 300 }}>
                        <Typography variant="body2" noWrap>
                          {type.description || '-'}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          <Chip
                            label={type.subject_node_type_name || 'Unknown'}
                            size="small"
                            color="primary"
                            variant="outlined"
                            sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                          />
                          <ArrowForwardIcon sx={{ fontSize: 14, color: 'text.secondary' }} />
                          <Chip
                            label={type.object_node_type_name || 'Unknown'}
                            size="small"
                            color="secondary"
                            variant="outlined"
                            sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                          />
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
                          <Chip 
                            label={type.is_active ? 'Active' : 'Inactive'}
                            size="small"
                            color={type.is_active ? 'success' : 'default'}
                          />
                          {type.type && (
                            <Chip 
                              label={type.type === 'core' ? 'Core' : 'Custom'}
                              size="small"
                              color={type.type === 'core' ? 'primary' : 'warning'}
                              variant={type.type === 'core' ? 'filled' : 'outlined'}
                            />
                          )}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">
                          {new Date(type.created_at).toLocaleDateString()}
                        </Typography>
                      </TableCell>
                      <TableCell align="right">
                        <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end' }}>
                          <IconButton 
                            size="small"
                            onClick={() => handleEditOpen(type)}
                            sx={{ color: 'primary.main' }}
                          >
                            <EditIcon fontSize="small" />
                          </IconButton>
                          <IconButton 
                            size="small"
                            onClick={() => navigate(`/catalog/edge-types/${type.id}`)}
                          >
                            <ArrowForwardIcon fontSize="small" />
                          </IconButton>
                          <IconButton 
                            size="small"
                            onClick={() => handleDelete(type)}
                            sx={{ color: 'error.main' }}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Box>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Edit Dialog */}
      <Dialog open={!!editingType} onClose={() => setEditingType(null)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Edge Type</DialogTitle>
        <DialogContent sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Box>
            <Typography variant="subtitle2" fontWeight="bold" gutterBottom>
              Predicate
            </Typography>
            <Typography variant="body2" color="text.secondary">
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
          />
          <Box>
            <Typography variant="subtitle2" fontWeight="bold" gutterBottom>
              Color
            </Typography>
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', mb: 1 }}>
              <Box
                sx={{
                  width: 40,
                  height: 40,
                  borderRadius: 1,
                  bgcolor: editColor || '#ccc',
                  border: `2px solid ${theme.palette.divider}`,
                }}
              />
              <TextField
                type="text"
                placeholder="#FF5733"
                value={editColor}
                onChange={(e) => setEditColor(e.target.value)}
                size="small"
                sx={{ flex: 1 }}
              />
              <IconButton
                size="small"
                onClick={() => setColorPaletteOpen(true)}
                sx={{ border: `1px solid ${theme.palette.divider}`, borderRadius: 1 }}
              >
                <PaletteIcon fontSize="small" />
              </IconButton>
            </Box>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
              Click the palette icon to choose from suggested colors, or enter a hex code (e.g., #FF5733).
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditingType(null)}>Cancel</Button>
          <Button 
            onClick={handleEditSave} 
            variant="contained"
            disabled={updateMutation.isPending}
          >
            Save
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
      <Dialog open={isCreateModalOpen} onClose={() => setIsCreateModalOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create Edge Type</DialogTitle>
        <DialogContent sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label="Predicate"
            placeholder="e.g., has_semantic_meaning"
            value={createForm.edge_type_name}
            onChange={(e) => setCreateForm({ ...createForm, edge_type_name: e.target.value })}
            fullWidth
            required
          />
          <TextField
            label="Description"
            placeholder="e.g., Semantic relationship between entities"
            multiline
            rows={3}
            value={createForm.description}
            onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
            fullWidth
          />
          <Autocomplete
            options={nodeTypes || []}
            getOptionLabel={(option) => `${option.catalog_type_name} (${option.id.substring(0, 8)}...)`}
            value={nodeTypes?.find(nt => nt.id === createForm.subjectNodeTypeId) || null}
            onChange={(_, newValue) => setCreateForm({ ...createForm, subjectNodeTypeId: newValue?.id || '' })}
            renderInput={(params) => <TextField {...params} label="Subject Node Type" placeholder="Search node types..." required />}
            fullWidth
            isOptionEqualToValue={(option, value) => option.id === value.id}
          />
          <Autocomplete
            options={nodeTypes || []}
            getOptionLabel={(option) => `${option.catalog_type_name} (${option.id.substring(0, 8)}...)`}
            value={nodeTypes?.find(nt => nt.id === createForm.objectNodeTypeId) || null}
            onChange={(_, newValue) => setCreateForm({ ...createForm, objectNodeTypeId: newValue?.id || '' })}
            renderInput={(params) => <TextField {...params} label="Object Node Type" placeholder="Search node types..." required />}
            fullWidth
            isOptionEqualToValue={(option, value) => option.id === value.id}
          />
          <FormControlLabel
            control={
              <Switch
                checked={createForm.isActive}
                onChange={(e) => setCreateForm({ ...createForm, isActive: e.target.checked })}
              />
            }
            label="Active"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setIsCreateModalOpen(false)}>Cancel</Button>
          <Button 
            onClick={handleCreateSubmit}
            variant="contained"
            disabled={createMutation.isPending}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

