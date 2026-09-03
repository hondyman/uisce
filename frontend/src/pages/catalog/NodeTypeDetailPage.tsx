import React, { useState, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  Box, Typography, Paper, Grid, Chip, Button, IconButton, 
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  TextField, useTheme, alpha, Skeleton, Breadcrumbs, Link, Card,
  Dialog, DialogTitle, DialogContent, DialogActions, Stack, Divider,
  FormControl, FormLabel, InputLabel, Select, MenuItem
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import CloseIcon from '@mui/icons-material/Close';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import DescriptionOutlinedIcon from '@mui/icons-material/DescriptionOutlined';
import SettingsInputComponentIcon from '@mui/icons-material/SettingsInputComponent';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { useNodeType, useNodesByType, useCreateNode, useUpdateNode, useDeleteNode, CatalogNode } from '../../api/nodeTypes';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import { useEdgeTypes } from '../../api/edgeTypes';
import { useTenant } from '../../contexts/TenantContext';
import { ProfessionalColorPicker } from '../../components/ProfessionalColorPicker';
import { PropertyEditor, PropertyDefinition } from '../../components/PropertyEditor';
import { CoreIcon, CustomIcon } from '../../components/common/CoreCustomIcons';

export const NodeTypeDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const theme = useTheme();
  const { tenant } = useTenant();
  const confirm = useConfirm();
  const notification = useNotification();
  const [selectedNode, setSelectedNode] = useState<any>(null);

  // Edit state
  const [editOpen, setEditOpen] = useState(false);
  const [editDescription, setEditDescription] = useState('');
  const [editColor, setEditColor] = useState('');
  const [editIsActive, setEditIsActive] = useState(false);
  const { data: edgeTypes } = useEdgeTypes(tenant?.id || '');
  const [editIsSaving, setEditIsSaving] = useState(false);
  const [editPropsOpen, setEditPropsOpen] = useState(false);
  const [editProps, setEditProps] = useState<PropertyDefinition[]>([]);
  const [editPropsSaving, setEditPropsSaving] = useState(false);

  // Asset (node instance) state
  const [assetDialogOpen, setAssetDialogOpen] = useState(false);
  const [editingAsset, setEditingAsset] = useState<CatalogNode | null>(null);
  const [assetName, setAssetName] = useState('');
  const [assetDescription, setAssetDescription] = useState('');
  const [assetIsActive, setAssetIsActive] = useState(true);
  const [assetFieldValues, setAssetFieldValues] = useState<Record<string, string>>({});

  const { data: nodeType, isLoading: typeLoading, refetch: refetchNodeType } = useNodeType(id || '');
  const { data: assetNodes, isLoading: assetsLoading } = useNodesByType(id || '');
  const createNodeMutation = useCreateNode();
  const updateNodeMutation = useUpdateNode();
  const deleteNodeMutation = useDeleteNode();

  // The node type's own schema (defined via the Properties editor) drives the form
  // fields shown when creating/editing an asset of this type.
  const typeProperties: PropertyDefinition[] = useMemo(() => {
    if (!nodeType?.config?.properties) return [];
    if (Array.isArray(nodeType.config.properties)) return nodeType.config.properties;
    return Object.entries(nodeType.config.properties).map(([k, v]: [string, any]) => ({
      name: k,
      label: v.label || k,
      data_type: v.data_type || 'string',
      nullable: v.nullable ?? true,
      description: v.description || '',
      properties: v.properties || [],
    }));
  }, [nodeType]);

  const handleAssetCreateOpen = () => {
    setEditingAsset(null);
    setAssetName('');
    setAssetDescription('');
    setAssetIsActive(true);
    setAssetFieldValues({});
    setAssetDialogOpen(true);
  };

  const handleAssetEditOpen = (node: CatalogNode) => {
    setEditingAsset(node);
    setAssetName(node.node_name || '');
    setAssetDescription(node.description || '');
    setAssetIsActive(node.is_active ?? true);
    const values: Record<string, string> = {};
    typeProperties.forEach((p) => {
      const v = node.properties?.[p.name];
      values[p.name] = v === undefined || v === null ? '' : String(v);
    });
    setAssetFieldValues(values);
    setAssetDialogOpen(true);
  };

  const handleAssetSave = async () => {
    if (!id || !assetName.trim()) {
      notification.error('Name is required');
      return;
    }
    const properties: Record<string, any> = {};
    typeProperties.forEach((p) => {
      const raw = assetFieldValues[p.name];
      if (raw === undefined || raw === '') return;
      if (p.data_type === 'integer' || p.data_type === 'float') {
        const num = Number(raw);
        properties[p.name] = Number.isNaN(num) ? raw : num;
      } else if (p.data_type === 'boolean') {
        properties[p.name] = raw === 'true';
      } else {
        properties[p.name] = raw;
      }
    });

    try {
      if (editingAsset) {
        await updateNodeMutation.mutateAsync({
          id: editingAsset.id,
          nodeTypeId: id,
          node_name: assetName,
          description: assetDescription,
          is_active: assetIsActive,
          properties,
        });
        notification.success(`"${assetName}" updated successfully`);
      } else {
        await createNodeMutation.mutateAsync({
          nodeTypeId: id,
          node_name: assetName,
          description: assetDescription,
          is_active: assetIsActive,
          properties,
        });
        notification.success(`"${assetName}" created successfully`);
      }
      setAssetDialogOpen(false);
    } catch (error) {
      notification.error(`Failed to save asset: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleAssetDelete = async (node: CatalogNode) => {
    if (!id) return;
    const confirmed = await confirm({
      title: 'Delete Asset',
      description: `Are you sure you want to delete "${node.node_name}"? This action cannot be undone.`,
    });
    if (!confirmed) return;

    try {
      await deleteNodeMutation.mutateAsync({ id: node.id, nodeTypeId: id });
      notification.success(`"${node.node_name}" deleted successfully`);
    } catch (error) {
      notification.error(`Failed to delete asset: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  // Filter edge types where this node type is subject or object
  const associatedEdgeTypes = useMemo(() => {
    if (!edgeTypes || !id) return [];
    return edgeTypes.filter(et => 
      et.subject_node_type_id === id || et.object_node_type_id === id
    );
  }, [edgeTypes, id]);

  // Initialize edit form when dialog opens
  const handleEditOpen = () => {
    if (nodeType) {
      setEditDescription(nodeType.description || '');
      setEditColor(nodeType.config?.color || '#3B82F6');
      setEditIsActive(nodeType.is_active ?? true);
      setEditOpen(true);
    }
  };

  const handleDeleteProperty = (name: string) => {
    setEditProps(editProps.filter(p => p.name !== name));
  };

  // Open properties editor and map existing properties to PropertyDefinition[]
  const handleEditPropsOpen = () => {
    if (!nodeType) return;
    const existingProps: PropertyDefinition[] = [];
    if (Array.isArray(nodeType.config?.properties)) {
      nodeType.config.properties.forEach((p: any) => {
        existingProps.push({
          name: p.name || '',
          label: p.label || '',
          data_type: p.data_type || 'string',
          nullable: p.nullable ?? true,
          description: p.description || '',
          properties: p.properties || [],
        });
      });
    } else if (nodeType.config?.properties && typeof nodeType.config.properties === 'object') {
      Object.entries(nodeType.config.properties).forEach(([k, v]: [string, any]) => {
        existingProps.push({
          name: k,
          label: v.label || '',
          data_type: v.data_type || 'string',
          nullable: v.nullable ?? true,
          description: v.description || '',
          properties: v.properties || [],
        });
      });
    }
    setEditProps(existingProps);
    setEditPropsOpen(true);
  };

  const handleSaveProperties = async () => {
    if (!nodeType || !id || !tenant?.id) return;
    setEditPropsSaving(true);
    try {
      const response = await fetch(`/api/node-types/${id}?tenant_id=${tenant.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          description: nodeType.description,
          config: nodeType.config || {},
          properties: editProps,
        }),
      });

      if (response.ok) {
        setEditPropsOpen(false);
        refetchNodeType();
      }
    } catch (error) {
      console.error('Failed to save properties:', error);
    } finally {
      setEditPropsSaving(false);
    }
  };

  // Save edit changes
  const handleEditSave = async () => {
    if (!nodeType || !id || !tenant?.id) return;
    setEditIsSaving(true);
    try {
      const response = await fetch(`/api/node-types/${id}?tenant_id=${tenant.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          description: editDescription,
          is_active: editIsActive,
          config: {
            ...(nodeType.config || {}),
            color: editColor,
          },
        }),
      });

      if (response.ok) {
        setEditOpen(false);
        refetchNodeType();
      }
    } catch (error) {
      console.error('Failed to save node type:', error);
    } finally {
      setEditIsSaving(false);
    }
  };

  // Removed - addProperty and removeProperty are no longer used, using PropertyEditor component instead

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

  if (typeLoading) {
    return (
      <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto' }}>
        <Skeleton variant="rectangular" height={260} sx={{ borderRadius: 3, bgcolor: isDark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.04)' }} />
      </Box>
    );
  }

  if (!nodeType) {
    return (
      <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto', color: C.text }}>
        <Typography>Node Type not found</Typography>
      </Box>
    );
  }

  const nodeColor = nodeType.config?.color;
  const isCore = nodeType.type === 'core' || nodeType.core;

  return (
    <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto', minHeight: '100vh', color: C.text, bgcolor: C.bg }}>
      {/* Breadcrumbs */}
      <Breadcrumbs sx={{ mb: 3, '& .MuiBreadcrumbs-separator': { color: C.textMuted } }}>
        <Link 
          color="inherit" 
          component="button" 
          onClick={() => navigate('/catalog/node-types')}
          underline="hover"
          sx={{ color: C.textMuted, '&:hover': { color: C.accent }, cursor: 'pointer', fontSize: '0.9rem' }}
        >
          Node Types
        </Link>
        <Typography sx={{ color: C.text, fontSize: '0.9rem', fontWeight: 600 }}>{nodeType.catalog_type_name}</Typography>
      </Breadcrumbs>

      {/* Header Card */}
      <Paper 
        elevation={0}
        sx={{ 
          p: 3.5, 
          mb: 4, 
          borderRadius: 3, 
          bgcolor: C.panel,
          border: `1px solid ${C.border}`,
          borderTop: nodeColor ? `3px solid ${nodeColor}` : `1px solid ${C.border}`,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <IconButton 
              onClick={() => navigate('/catalog/node-types')} 
              sx={{ 
                bgcolor: isDark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.04)', 
                border: `1px solid ${C.border}`,
                color: C.textMuted,
                '&:hover': { color: C.text, bgcolor: C.accentDim }
              }}
            >
              <ArrowBackIcon fontSize="small" />
            </IconButton>
            <Box>
              <Typography variant="h4" fontWeight="bold" sx={{ color: C.text, letterSpacing: '-0.02em', mb: 1 }}>
                {nodeType.catalog_type_name}
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
                {nodeColor && (
                  <Box sx={{ width: 14, height: 14, borderRadius: '50%', bgcolor: nodeColor, border: `1px solid ${C.border}` }} title={`Color: ${nodeColor}`} />
                )}
                <span style={{
                  display: 'inline-flex', alignItems: 'center', padding: '2px 8px',
                  borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                  color: nodeType.is_active ? C.success : C.textMuted,
                  background: nodeType.is_active ? (isDark ? `${C.success}18` : `${C.success}12`) : 'transparent',
                  border: `1px solid ${nodeType.is_active ? C.success : C.border}44`,
                  fontFamily: 'monospace', textTransform: 'uppercase',
                }}>
                  {nodeType.is_active ? 'Active' : 'Inactive'}
                </span>
                {isCore ? <CoreIcon fontSize="small" /> : <CustomIcon fontSize="small" />}
                <Typography variant="caption" sx={{ color: C.textMuted, fontFamily: 'monospace', ml: 0.5 }}>
                  ID: {nodeType.id}
                </Typography>
              </Box>
            </Box>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Button 
              variant="contained" 
              startIcon={<EditIcon />}
              onClick={handleEditOpen}
              sx={{ 
                borderRadius: 2, px: 2.5, py: 0.8,
                bgcolor: C.accent,
                color: '#FFFFFF',
                boxShadow: isDark ? C.accentGlow : 'none',
                '&:hover': { bgcolor: '#4F46E5' }
              }}
            >
              Edit Type
            </Button>
          </Box>
        </Box>
        
        <Grid container spacing={3} sx={{ mt: 2 }}>
            <Grid size={{ xs: 12, md: 7 }}>
                 <Typography variant="subtitle2" fontWeight="bold" sx={{ display: 'flex', alignItems: 'center', gap: 0.8, color: C.textMuted, mb: 1, textTransform: 'uppercase', letterSpacing: '0.04em' }}>
                    <DescriptionOutlinedIcon fontSize="small"/> Description
                 </Typography>
                 <Typography variant="body1" sx={{ color: C.text, lineHeight: 1.6 }}>
                    {nodeType.description || 'No description provided for this node type.'}
                 </Typography>
            </Grid>
            
            <Grid size={{ xs: 12, md: 5 }}>
                {/* Properties Card */}
                <Card 
                  elevation={0} 
                  sx={{ 
                    borderRadius: 2.5, 
                    height: '100%', 
                    minHeight: 280,
                    bgcolor: isDark ? '#0F1117' : '#F8FAFC',
                    border: `1px solid ${C.border}`,
                  }}
                >
                  <Box sx={{ 
                    p: 2, 
                    borderBottom: `1px solid ${C.border}`, 
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center'
                  }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <SettingsInputComponentIcon fontSize="small" sx={{ color: C.accent }} />
                      <Typography variant="subtitle2" fontWeight="bold" sx={{ color: C.text }}>Properties</Typography>
                    </Box>
                    <IconButton size="small" sx={{ color: C.accent, bgcolor: C.accentDim }} onClick={handleEditPropsOpen}>
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </Box>
                  <Box sx={{ p: 0 }}>
                    {nodeType.config?.properties && (Array.isArray(nodeType.config.properties) ? nodeType.config.properties.length > 0 : Object.keys(nodeType.config.properties).length > 0) ? (
                      <Stack divider={<Divider sx={{ borderColor: C.border }} />}>
                        {(Array.isArray(nodeType.config.properties) ? nodeType.config.properties : Object.entries(nodeType.config.properties).map(([key, val]: [string, any]) => ({ name: key, ...val }))).map((prop: any) => (
                          <Box key={prop.name} sx={{ p: 1.5, '&:hover': { bgcolor: isDark ? 'rgba(255,255,255,0.02)' : 'rgba(0,0,0,0.02)' } }}>
                            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                              <Typography variant="body2" fontWeight="bold" sx={{ color: C.text }}>{prop.label || prop.name}</Typography>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <span style={{
                                  display: 'inline-flex', alignItems: 'center', padding: '1px 6px',
                                  borderRadius: 4, fontSize: 10, fontWeight: 700,
                                  color: C.blue, background: isDark ? 'rgba(96,165,250,0.12)' : 'rgba(96,165,250,0.08)',
                                  border: `1px solid rgba(96,165,250,0.3)`, fontFamily: 'monospace'
                                }}>
                                  {prop.data_type || 'string'}
                                </span>
                                <IconButton size="small" sx={{ color: C.textMuted, '&:hover': { color: C.danger } }} onClick={() => handleDeleteProperty(prop.name)}>
                                  <DeleteIcon fontSize="small" />
                                </IconButton>
                              </Box>
                            </Box>
                            {prop.description && (
                              <Typography variant="caption" sx={{ color: C.textMuted, display: 'block', mt: 0.5 }}>
                                {prop.description}
                              </Typography>
                            )}
                            {prop.properties && prop.properties.length > 0 && (
                              <Box sx={{ mt: 1, pl: 2, borderLeft: `2px solid ${C.border}` }}>
                                <Typography variant="caption" sx={{ color: C.textMuted }}>
                                  {prop.properties.length} nested fields
                                </Typography>
                              </Box>
                            )}
                          </Box>
                        ))}
                      </Stack>
                    ) : (
                      <Box sx={{ p: 3, textAlign: 'center' }}>
                        <InfoOutlinedIcon sx={{ fontSize: 32, color: C.textMuted, mb: 1, opacity: 0.5 }} />
                        <Typography variant="body2" sx={{ color: C.textMuted }}>No properties defined.</Typography>
                        <Button size="small" sx={{ mt: 1.5, color: C.accent }} onClick={handleEditPropsOpen}>Add Property</Button>
                      </Box>
                    )}
                  </Box>
                </Card>
            </Grid>
        </Grid>
      </Paper>

      {/* Assets (node instances) Section */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6" fontWeight="bold" sx={{ color: C.text }}>
          Assets ({assetNodes?.length || 0})
        </Typography>
        <Button
          variant="contained"
          size="small"
          startIcon={<AddIcon />}
          onClick={handleAssetCreateOpen}
          sx={{
            borderRadius: 2,
            bgcolor: C.accent,
            color: '#FFFFFF',
            '&:hover': { bgcolor: '#4F46E5' },
          }}
        >
          Add Asset
        </Button>
      </Box>

      {assetsLoading ? (
        <Skeleton variant="rectangular" height={120} sx={{ borderRadius: 3, mb: 4, bgcolor: isDark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.04)' }} />
      ) : assetNodes && assetNodes.length > 0 ? (
        <TableContainer
          component={Paper}
          elevation={0}
          sx={{
            border: `1px solid ${C.border}`,
            borderRadius: 3,
            bgcolor: C.panel,
            overflow: 'hidden',
            mb: 4,
          }}
        >
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: isDark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)', borderBottom: `1px solid ${C.border}` }}>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Name</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Description</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Status</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Created</TableCell>
                <TableCell align="right" sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {assetNodes.map((node) => (
                <TableRow
                  key={node.id}
                  hover
                  onClick={() => setSelectedNode(node)}
                  sx={{
                    cursor: 'pointer',
                    '&:hover': { bgcolor: isDark ? 'rgba(255,255,255,0.02) !important' : 'rgba(0,0,0,0.01) !important' },
                    borderBottom: `1px solid ${C.border}`,
                  }}
                >
                  <TableCell sx={{ fontWeight: 600, color: C.text }}>{node.node_name}</TableCell>
                  <TableCell sx={{ maxWidth: 320 }}>
                    <Typography variant="body2" sx={{ color: C.textMuted }} noWrap>
                      {node.description || '-'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <span style={{
                      display: 'inline-flex', alignItems: 'center', padding: '2px 8px',
                      borderRadius: 9999, fontSize: 10, fontWeight: 700, letterSpacing: '0.04em',
                      color: node.is_active !== false ? C.success : C.textMuted,
                      background: node.is_active !== false ? (isDark ? `${C.success}18` : `${C.success}12`) : 'transparent',
                      border: `1px solid ${node.is_active !== false ? C.success : C.border}44`,
                      fontFamily: 'monospace', textTransform: 'uppercase',
                    }}>
                      {node.is_active !== false ? 'Active' : 'Inactive'}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" sx={{ color: C.textMuted, fontFamily: 'monospace', fontSize: '0.8rem' }}>
                      {new Date(node.created_at).toLocaleDateString()}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Box sx={{ display: 'flex', gap: 0.5, justifyContent: 'flex-end' }} onClick={(e) => e.stopPropagation()}>
                      <IconButton size="small" onClick={() => handleAssetEditOpen(node)} sx={{ color: C.textMuted, '&:hover': { color: C.accent, bgcolor: C.accentDim } }}>
                        <EditIcon fontSize="small" />
                      </IconButton>
                      <IconButton size="small" onClick={() => handleAssetDelete(node)} sx={{ color: C.textMuted, '&:hover': { color: C.danger, bgcolor: `${C.danger}18` } }}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Box>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      ) : (
        <Paper
          elevation={0}
          sx={{
            p: 4,
            mb: 4,
            borderRadius: 3,
            border: `1px solid ${C.border}`,
            bgcolor: C.panel,
            textAlign: 'center',
          }}
        >
          <Typography sx={{ color: C.textMuted, mb: 1.5 }}>No assets of this type yet.</Typography>
          <Button size="small" startIcon={<AddIcon />} sx={{ color: C.accent }} onClick={handleAssetCreateOpen}>
            Add the first asset
          </Button>
        </Paper>
      )}

      {/* Associated Edge Types Section */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6" fontWeight="bold" sx={{ color: C.text }}>
          Associated Edge Types ({associatedEdgeTypes?.length || 0})
        </Typography>
      </Box>

      {associatedEdgeTypes && associatedEdgeTypes.length > 0 ? (
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
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Subject</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Predicate</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Object</TableCell>
                <TableCell sx={{ fontWeight: 700, color: C.textMuted, fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>Type</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {associatedEdgeTypes.map((edgeType) => (
                <TableRow 
                  key={edgeType.id} 
                  hover
                  sx={{ 
                    '&:hover': { bgcolor: isDark ? 'rgba(255,255,255,0.02) !important' : 'rgba(0,0,0,0.01) !important' },
                    borderBottom: `1px solid ${C.border}`,
                  }}
                >
                  <TableCell>
                    <span style={{
                      display: 'inline-flex', alignItems: 'center', padding: '2px 7px',
                      borderRadius: 6, fontSize: 11, fontWeight: 600,
                      color: C.blue, background: isDark ? 'rgba(96,165,250,0.12)' : 'rgba(96,165,250,0.08)',
                      border: `1px solid rgba(96,165,250,0.3)`,
                    }}>
                      {edgeType.subject_node_type_name || 'Unknown'}
                    </span>
                  </TableCell>
                  <TableCell sx={{ fontWeight: 600, color: C.text }}>
                    {edgeType.edge_type_name}
                  </TableCell>
                  <TableCell>
                    <span style={{
                      display: 'inline-flex', alignItems: 'center', padding: '2px 7px',
                      borderRadius: 6, fontSize: 11, fontWeight: 600,
                      color: C.purple, background: isDark ? 'rgba(167,139,250,0.12)' : 'rgba(167,139,250,0.08)',
                      border: `1px solid rgba(167,139,250,0.3)`,
                    }}>
                      {edgeType.object_node_type_name || 'Unknown'}
                    </span>
                  </TableCell>
                  <TableCell>
                    {edgeType.type === 'core' ? <CoreIcon fontSize="small" /> : <CustomIcon fontSize="small" />}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      ) : (
        <Paper 
          elevation={0}
          sx={{ 
            p: 4, 
            borderRadius: 3, 
            border: `1px solid ${C.border}`,
            bgcolor: C.panel,
            textAlign: 'center'
          }}
        >
          <Typography sx={{ color: C.textMuted }}>No edge types associated with this node type.</Typography>
        </Paper>
      )}

      {/* Node Details Dialog */}
      {selectedNode && (
        <Dialog 
          open={!!selectedNode} 
          onClose={() => setSelectedNode(null)}
          maxWidth="md"
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
          <DialogTitle sx={{ borderBottom: `1px solid ${C.border}`, fontWeight: 700 }}>
            {selectedNode.node_name}
            <IconButton
              aria-label="close"
              onClick={() => setSelectedNode(null)}
              sx={{
                position: 'absolute',
                right: 8,
                top: 8,
                color: C.textMuted,
              }}
            >
              <CloseIcon />
            </IconButton>
          </DialogTitle>
          <DialogContent dividers sx={{ borderColor: C.border }}>
             <Typography variant="subtitle2" sx={{ color: C.textMuted, mb: 0.5 }}>Qualified Path</Typography>
             <Paper variant="outlined" sx={{ p: 1.5, bgcolor: isDark ? '#0A0C12' : '#F8FAFC', border: `1px solid ${C.border}`, mb: 2, fontFamily: 'monospace', color: C.text }}>
                {selectedNode.qualified_path}
             </Paper>

             {selectedNode.description && (
               <>
                 <Typography variant="subtitle2" sx={{ color: C.textMuted, mb: 0.5 }}>Description</Typography>
                 <Typography variant="body2" sx={{ color: C.text, mb: 2 }}>{selectedNode.description}</Typography>
               </>
             )}

             <Typography variant="subtitle2" sx={{ color: C.textMuted, mb: 0.5 }}>Properties</Typography>
             <Paper variant="outlined" sx={{ p: 2, bgcolor: isDark ? '#0A0C12' : '#F8FAFC', border: `1px solid ${C.border}`, maxHeight: 400, overflow: 'auto' }}>
                <Box component="pre" sx={{ margin: 0, fontSize: '0.85rem', fontFamily: 'monospace', color: C.text, whiteSpace: 'pre-wrap', wordWrap: 'break-word' }}>
                  {JSON.stringify(selectedNode.properties, null, 2)}
                </Box>
             </Paper>
          </DialogContent>
          <DialogActions sx={{ p: 2, borderTop: `1px solid ${C.border}` }}>
            <Button onClick={() => setSelectedNode(null)} sx={{ color: C.textMuted }}>Close</Button>
          </DialogActions>
        </Dialog>
      )}

      {/* Edit Node Type Dialog */}
      <Dialog 
        open={editOpen} 
        onClose={() => setEditOpen(false)}
        maxWidth="md"
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
        <DialogTitle sx={{ borderBottom: `1px solid ${C.border}`, fontWeight: 700 }}>
          Edit Node Type: {nodeType?.catalog_type_name}
          <IconButton
            aria-label="close"
            onClick={() => setEditOpen(false)}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
              color: C.textMuted,
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers sx={{ borderColor: C.border, pt: 3 }}>
          <Stack spacing={3} sx={{ mt: 1 }}>
            {/* Active Status */}
            <FormControl fullWidth>
              <FormLabel sx={{ color: C.textMuted, fontWeight: 600, fontSize: '0.85rem' }}>Status</FormLabel>
              <Box sx={{ display: 'flex', gap: 2, mt: 1 }}>
                <Button
                  variant={editIsActive ? 'contained' : 'outlined'}
                  onClick={() => setEditIsActive(true)}
                  sx={{ 
                    flex: 1, 
                    bgcolor: editIsActive ? C.success : 'transparent',
                    color: editIsActive ? '#fff' : C.textMuted,
                    borderColor: editIsActive ? C.success : C.border,
                    '&:hover': { bgcolor: editIsActive ? '#059669' : `${C.success}18` }
                  }}
                >
                  Active
                </Button>
                <Button
                  variant={!editIsActive ? 'contained' : 'outlined'}
                  onClick={() => setEditIsActive(false)}
                  sx={{ 
                    flex: 1, 
                    bgcolor: !editIsActive ? C.danger : 'transparent',
                    color: !editIsActive ? '#fff' : C.textMuted,
                    borderColor: !editIsActive ? C.danger : C.border,
                    '&:hover': { bgcolor: !editIsActive ? '#DC2626' : `${C.danger}18` }
                  }}
                >
                  Inactive
                </Button>
              </Box>
            </FormControl>

            {/* Description */}
            <TextField
              fullWidth
              label="Description"
              value={editDescription}
              onChange={(e) => setEditDescription(e.target.value)}
              multiline
              rows={3}
              placeholder="Enter node type description..."
              InputProps={{
                sx: {
                  bgcolor: isDark ? '#0A0C12' : '#F8FAFC',
                  color: C.text,
                  '& fieldset': { borderColor: C.border },
                }
              }}
            />

            {/* Color Picker */}
            <Box>
              <ProfessionalColorPicker
                color={editColor}
                onChange={setEditColor}
                label="Node Type Color"
                showRecent
              />
            </Box>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2, borderTop: `1px solid ${C.border}` }}>
          <Button onClick={() => setEditOpen(false)} sx={{ color: C.textMuted }}>Cancel</Button>
          <Button 
            onClick={handleEditSave} 
            variant="contained" 
            disabled={editIsSaving}
            sx={{ bgcolor: C.accent, color: '#fff', '&:hover': { bgcolor: '#4F46E5' } }}
          >
            {editIsSaving ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Properties Editor Dialog */}
      <Dialog 
        open={editPropsOpen} 
        onClose={() => setEditPropsOpen(false)} 
        maxWidth="md" 
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
        <DialogTitle sx={{ borderBottom: `1px solid ${C.border}` }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <SettingsInputComponentIcon sx={{ color: C.accent }} />
            <Typography variant="h6" fontWeight="bold">Manage Node Type Properties</Typography>
          </Box>
          <IconButton
            aria-label="close"
            onClick={() => setEditPropsOpen(false)}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
              color: C.textMuted,
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers sx={{ borderColor: C.border, bgcolor: isDark ? '#0F1117' : '#F8FAFC' }}>
          <Box sx={{ py: 2 }}>
            <PropertyEditor 
              properties={editProps} 
              onChange={setEditProps} 
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2, borderTop: `1px solid ${C.border}` }}>
          <Button onClick={() => setEditPropsOpen(false)} sx={{ color: C.textMuted }}>Cancel</Button>
          <Button 
            onClick={handleSaveProperties} 
            variant="contained" 
            disabled={editPropsSaving}
            sx={{ bgcolor: C.accent, color: '#fff', '&:hover': { bgcolor: '#4F46E5' } }}
          >
            {editPropsSaving ? 'Saving...' : 'Save All Properties'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Create/Edit Asset Dialog */}
      <Dialog
        open={assetDialogOpen}
        onClose={() => setAssetDialogOpen(false)}
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
        <DialogTitle sx={{ borderBottom: `1px solid ${C.border}`, fontWeight: 700 }}>
          {editingAsset ? `Edit Asset: ${editingAsset.node_name}` : `New ${nodeType.catalog_type_name} Asset`}
        </DialogTitle>
        <DialogContent sx={{ pt: 3, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label="Name"
            value={assetName}
            onChange={(e) => setAssetName(e.target.value)}
            fullWidth
            required
            sx={{ mt: 1 }}
            InputProps={{
              sx: { bgcolor: isDark ? '#0A0C12' : '#F8FAFC', color: C.text, '& fieldset': { borderColor: C.border } }
            }}
          />
          <TextField
            label="Description"
            multiline
            rows={2}
            value={assetDescription}
            onChange={(e) => setAssetDescription(e.target.value)}
            fullWidth
            InputProps={{
              sx: { bgcolor: isDark ? '#0A0C12' : '#F8FAFC', color: C.text, '& fieldset': { borderColor: C.border } }
            }}
          />

          {typeProperties.length > 0 && (
            <>
              <Typography variant="subtitle2" fontWeight="bold" sx={{ color: C.textMuted, mt: 1 }}>
                Properties
              </Typography>
              {typeProperties.map((p) => (
                p.data_type === 'boolean' ? (
                  <FormControl key={p.name} fullWidth size="small">
                    <InputLabel sx={{ color: C.textMuted }}>{p.label || p.name}</InputLabel>
                    <Select
                      value={assetFieldValues[p.name] ?? ''}
                      label={p.label || p.name}
                      onChange={(e) => setAssetFieldValues({ ...assetFieldValues, [p.name]: e.target.value as string })}
                      sx={{ color: C.text, '& fieldset': { borderColor: C.border } }}
                    >
                      <MenuItem value="">-</MenuItem>
                      <MenuItem value="true">True</MenuItem>
                      <MenuItem value="false">False</MenuItem>
                    </Select>
                  </FormControl>
                ) : (
                  <TextField
                    key={p.name}
                    label={p.label || p.name}
                    type={p.data_type === 'integer' || p.data_type === 'float' ? 'number' : p.data_type === 'date' ? 'date' : 'text'}
                    value={assetFieldValues[p.name] ?? ''}
                    onChange={(e) => setAssetFieldValues({ ...assetFieldValues, [p.name]: e.target.value })}
                    fullWidth
                    size="small"
                    helperText={p.description}
                    InputLabelProps={p.data_type === 'date' ? { shrink: true } : undefined}
                    InputProps={{
                      sx: { bgcolor: isDark ? '#0A0C12' : '#F8FAFC', color: C.text, '& fieldset': { borderColor: C.border } }
                    }}
                  />
                )
              ))}
            </>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 2, borderTop: `1px solid ${C.border}` }}>
          <Button onClick={() => setAssetDialogOpen(false)} sx={{ color: C.textMuted }}>Cancel</Button>
          <Button
            onClick={handleAssetSave}
            variant="contained"
            disabled={createNodeMutation.isPending || updateNodeMutation.isPending}
            sx={{ bgcolor: C.accent, color: '#fff', '&:hover': { bgcolor: '#4F46E5' } }}
          >
            {editingAsset ? 'Save Changes' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
