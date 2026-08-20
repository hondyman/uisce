import React, { useState, useMemo } from 'react';
import { 
  Box, Typography, Grid, Paper, TextField, InputAdornment, 
  Card, CardContent, Chip, IconButton, useTheme, alpha, Skeleton,
  Button, Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Dialog, DialogTitle, DialogContent, DialogActions, FormControlLabel, Switch, Autocomplete,
  Tooltip
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
import { CoreIcon, CustomIcon } from '../../components/common/CoreCustomIcons';

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
      .map(type => type.config!.color) || [];
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

          {/* View Toggles & Actions */}
          <Box sx={{ display: 'flex', gap: 0.5, border: `1px solid ${C.border}`, borderRadius: 2, p: 0.5, bgcolor: C.panel }}>
            <IconButton 
              size="small"
              onClick={() => setViewMode('tiles')}
              sx={{ 
                bgcolor: viewMode === 'tiles' ? C.accentDim : 'transparent', 
                color: viewMode === 'tiles' ? C.accent : C.textMuted,
                borderRadius: 1.5,
                border: viewMode === 'tiles' ? `1px solid ${C.accent}66` : '1px solid transparent',
              }}
            >
              <ViewComfyIcon fontSize="small" />
            </IconButton>
            <IconButton 
              size="small"
              onClick={() => setViewMode('table')}
              sx={{ 
                bgcolor: viewMode === 'table' ? C.accentDim : 'transparent', 
                color: viewMode === 'table' ? C.accent : C.textMuted,
                borderRadius: 1.5,
                border: viewMode === 'table' ? `1px solid ${C.accent}66` : '1px solid transparent',
              }}
            >
              <ViewAgendaIcon fontSize="small" />
            </IconButton>
          </Box>
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

      {/* Search and Filter */}
      <Box 
        sx={{ 
          p: 1.5, 
          mb: 3, 
          borderRadius: 2.5, 
          border: `1px solid ${C.border}`,
          bgcolor: C.panel,
          display: 'flex',
          alignItems: 'center',
          gap: 1.5
        }}
      >
        <TextField
          fullWidth
          variant="outlined"
          placeholder="Search edge types by predicate or description..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: C.textMuted }} />
              </InputAdornment>
            ),
            sx: { 
              borderRadius: 2,
              bgcolor: isDark ? '#0A0C12' : '#F8FAFC',
              color: C.text,
              '& fieldset': { borderColor: C.border },
              '&:hover fieldset': { borderColor: `${C.accent}88` },
              '&.Mui-focused fieldset': { borderColor: C.accent },
            }
          }}
          size="small"
        />
        <IconButton sx={{ border: `1px solid ${C.border}`, borderRadius: 2, color: C.textMuted, '&:hover': { color: C.text, bgcolor: C.accentDim } }}>
          <FilterListIcon fontSize="small" />
        </IconButton>
      </Box>

      {/* Edge Types Grid or Table */}
      {viewMode === 'tiles' ? (
        <Grid container spacing={2.5}>
          {isLoading ? (
            Array.from({ length: 8 }).map((_, i) => (
              <Grid key={i} size={{ xs: 12, sm: 6, md: 4, lg: 3 }}>
                <Skeleton 
                  variant="rectangular" 
                  height={190} 
                  sx={{ borderRadius: 3, bgcolor: isDark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.04)' }} 
                />
              </Grid>
            ))
          ) : filteredTypes?.map((type) => {
            const edgeColor = type.config?.color;
            const isCore = type.type === 'core';
            
            return (
              <Grid key={type.id} size={{ xs: 12, sm: 6, md: 4, lg: 3 }}>
                <Card 
                  elevation={0}
                  onClick={() => navigate(`/catalog/edge-types/${type.id}`)}
                  sx={{ 
                    height: '100%', 
                    borderRadius: 3,
                    bgcolor: C.panel,
                    border: `1px solid ${C.border}`,
                    transition: 'all 0.2s ease-in-out',
                    cursor: 'pointer',
                    position: 'relative',
                    overflow: 'hidden',
                    display: 'flex',
                    flexDirection: 'column',
                    '&:hover': {
                      transform: 'translateY(-3px)',
                      borderColor: edgeColor || C.accent,
                      boxShadow: isDark ? `0 8px 24px rgba(0,0,0,0.5)` : `0 8px 24px rgba(0,0,0,0.08)`,
                      bgcolor: C.panelHover,
                    }
                  }}
                >
                  {edgeColor && (
                    <Box sx={{ height: 3, bgcolor: edgeColor, width: '100%' }} />
                  )}
                  <CardContent sx={{ p: 2.5, flex: 1, display: 'flex', flexDirection: 'column' }}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1.5 }}>
                      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                        {type.type && (isCore ? <CoreIcon fontSize="small" /> : <CustomIcon fontSize="small" />)}
                      </Box>
                      {type.is_active && (
                        <Tooltip title="Active">
                          <Box 
                            sx={{ 
                              width: 8, 
                              height: 8, 
                              borderRadius: '50%', 
                              bgcolor: C.success,
                              boxShadow: `0 0 8px ${C.success}`
                            }} 
                          />
                        </Tooltip>
                      )}
                    </Box>
                    
                    <Typography variant="h6" fontWeight="bold" sx={{ color: C.text, fontSize: '1.05rem', mb: 1 }} noWrap>
                      {type.edge_type_name}
                    </Typography>
                    
                    {/* Subject -> Object Relationship Badge */}
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8, mb: 'auto', flexWrap: 'wrap' }}>
                      <span style={{
                        display: 'inline-flex', alignItems: 'center', padding: '2px 7px',
                        borderRadius: 6, fontSize: 11, fontWeight: 600,
                        color: C.blue, background: isDark ? 'rgba(96,165,250,0.12)' : 'rgba(96,165,250,0.08)',
                        border: `1px solid rgba(96,165,250,0.3)`,
                      }}>
                        {type.subject_node_type_name || 'Subject'}
                      </span>
                      <ArrowForwardIcon sx={{ fontSize: 13, color: C.textMuted }} />
                      <span style={{
                        display: 'inline-flex', alignItems: 'center', padding: '2px 7px',
                        borderRadius: 6, fontSize: 11, fontWeight: 600,
                        color: C.purple, background: isDark ? 'rgba(167,139,250,0.12)' : 'rgba(167,139,250,0.08)',
                        border: `1px solid rgba(167,139,250,0.3)`,
                      }}>
                        {type.object_node_type_name || 'Object'}
                      </span>
                    </Box>

                    {type.description && (
                      <Typography 
                        variant="body2" 
                        sx={{ 
                          color: C.textMuted,
                          fontSize: '0.82rem',
                          lineHeight: 1.4,
                          mt: 1.5,
                          mb: 'auto',
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden'
                        }}
                      >
                        {type.description}
                      </Typography>
                    )}

                    <Box sx={{ mt: 2.5, pt: 1.5, borderTop: `1px solid ${C.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 1 }}>
                      <Typography variant="caption" sx={{ color: C.textMuted, fontFamily: 'monospace', fontSize: '0.75rem' }}>
                        {new Date(type.created_at).toLocaleDateString()}
                      </Typography>
                      <Box sx={{ display: 'flex', gap: 0.5 }} onClick={(e) => e.stopPropagation()}>
                        <IconButton 
                          size="small"
                          onClick={() => handleEditOpen(type)}
                          sx={{ color: C.textMuted, '&:hover': { color: C.accent, bgcolor: C.accentDim } }}
                          title="Edit"
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                        <IconButton 
                          size="small"
                          onClick={() => navigate(`/catalog/edge-types/${type.id}`)}
                          sx={{ color: C.textMuted, '&:hover': { color: C.text, bgcolor: C.accentDim } }}
                          title="View Details"
                        >
                          <ArrowForwardIcon fontSize="small" />
                        </IconButton>
                        <IconButton 
                          size="small"
                          onClick={() => handleDelete(type)}
                          sx={{ color: C.textMuted, '&:hover': { color: C.danger, bgcolor: `${C.danger}18` } }}
                          title="Delete"
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
        <TableContainer 
          component={Paper} 
          elevation={0} 
          sx={{ 
            border: `1px solid ${C.border}`, 
            borderRadius: 3, 
            bgcolor: C.panel,
            overflow: 'hidden' 
          }}
        >
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: isDark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)', borderBottom: `1px solid ${C.border}` }}>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Predicate</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Description</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Relationship</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Status</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Created</TableCell>
                <TableCell align="right" sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 6 }).map((_, j) => (
                      <TableCell key={j}>
                        <Skeleton variant="text" sx={{ bgcolor: isDark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.04)' }} />
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
                      hover
                      sx={{ 
                        '&:hover': { bgcolor: isDark ? 'rgba(255,255,255,0.02) !important' : 'rgba(0,0,0,0.01) !important' },
                        borderLeft: edgeColor ? `3px solid ${edgeColor}` : 'none',
                        borderBottom: `1px solid ${C.border}`,
                      }}
                    >
                      <TableCell sx={{ fontWeight: 600, color: C.text }}>{type.edge_type_name}</TableCell>
                      <TableCell sx={{ maxWidth: 300 }}>
                        <Typography variant="body2" sx={{ color: C.textMuted }} noWrap>
                          {type.description || '-'}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8 }}>
                          <span style={{
                            display: 'inline-flex', alignItems: 'center', padding: '2px 7px',
                            borderRadius: 6, fontSize: 11, fontWeight: 600,
                            color: C.blue, background: isDark ? 'rgba(96,165,250,0.12)' : 'rgba(96,165,250,0.08)',
                            border: `1px solid rgba(96,165,250,0.3)`,
                          }}>
                            {type.subject_node_type_name || 'Unknown'}
                          </span>
                          <ArrowForwardIcon sx={{ fontSize: 12, color: C.textMuted }} />
                          <span style={{
                            display: 'inline-flex', alignItems: 'center', padding: '2px 7px',
                            borderRadius: 6, fontSize: 11, fontWeight: 600,
                            color: C.purple, background: isDark ? 'rgba(167,139,250,0.12)' : 'rgba(167,139,250,0.08)',
                            border: `1px solid rgba(167,139,250,0.3)`,
                          }}>
                            {type.object_node_type_name || 'Unknown'}
                          </span>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
                          <span style={{
                            display: 'inline-flex', alignItems: 'center', padding: '2px 8px',
                            borderRadius: 9999, fontSize: 10, fontWeight: 700, letterSpacing: '0.04em',
                            color: type.is_active ? C.success : C.textMuted,
                            background: type.is_active ? (isDark ? `${C.success}18` : `${C.success}12`) : 'transparent',
                            border: `1px solid ${type.is_active ? C.success : C.border}44`,
                            fontFamily: 'monospace', textTransform: 'uppercase',
                          }}>
                            {type.is_active ? 'Active' : 'Inactive'}
                          </span>
                          {type.type && (
                            <span style={{
                              display: 'inline-flex', alignItems: 'center', padding: '2px 8px',
                              borderRadius: 9999, fontSize: 10, fontWeight: 700, letterSpacing: '0.04em',
                              color: type.type === 'core' ? C.accent : C.warning,
                              background: type.type === 'core' ? (isDark ? `${C.accent}18` : `${C.accent}12`) : (isDark ? `${C.warning}18` : `${C.warning}12`),
                              border: `1px solid ${type.type === 'core' ? C.accent : C.warning}44`,
                              fontFamily: 'monospace', textTransform: 'uppercase',
                            }}>
                              {type.type}
                            </span>
                          )}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={{ color: C.textMuted, fontFamily: 'monospace', fontSize: '0.8rem' }}>
                          {new Date(type.created_at).toLocaleDateString()}
                        </Typography>
                      </TableCell>
                      <TableCell align="right">
                        <Box sx={{ display: 'flex', gap: 0.5, justifyContent: 'flex-end' }}>
                          <IconButton 
                            size="small"
                            onClick={() => handleEditOpen(type)}
                            sx={{ color: C.textMuted, '&:hover': { color: C.accent, bgcolor: C.accentDim } }}
                          >
                            <EditIcon fontSize="small" />
                          </IconButton>
                          <IconButton 
                            size="small"
                            onClick={() => navigate(`/catalog/edge-types/${type.id}`)}
                            sx={{ color: C.textMuted, '&:hover': { color: C.text, bgcolor: C.accentDim } }}
                          >
                            <ArrowForwardIcon fontSize="small" />
                          </IconButton>
                          <IconButton 
                            size="small"
                            onClick={() => handleDelete(type)}
                            sx={{ color: C.textMuted, '&:hover': { color: C.danger, bgcolor: `${C.danger}18` } }}
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

