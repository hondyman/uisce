import React, { useState, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  Box, Typography, Paper, Grid, Chip, Button, IconButton, 
  useTheme, alpha, Skeleton, Breadcrumbs, Link, Card,
  Dialog, DialogTitle, DialogContent, DialogActions, Stack, TextField, MenuItem, Divider,
  FormControl, FormLabel
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import EditIcon from '@mui/icons-material/Edit';
import CloseIcon from '@mui/icons-material/Close';
import DescriptionOutlinedIcon from '@mui/icons-material/DescriptionOutlined';
import ReactFlow, { 
  Node, Edge, Controls, Background,
  useNodesState, useEdgesState,
  NodeMouseHandler, MarkerType
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useEdgeType } from '../../api/edgeTypes';
import { useNodeType } from '../../api/nodeTypes';
import { useTenant } from '../../contexts/TenantContext';
import { ProfessionalColorPicker } from '../../components/ProfessionalColorPicker';
import { PropertyEditor, PropertyDefinition } from '../../components/PropertyEditor';
import { CoreIcon, CustomIcon } from '../../components/common/CoreCustomIcons';
import SettingsInputComponentIcon from '@mui/icons-material/SettingsInputComponent';
import LayersIcon from '@mui/icons-material/Layers';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';

export const EdgeTypeDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const theme = useTheme();
  const { tenant } = useTenant();
  
  // Edit state
  const [editOpen, setEditOpen] = useState(false);
  const [editDescription, setEditDescription] = useState('');
  const [editColor, setEditColor] = useState('');
  const [editIsActive, setEditIsActive] = useState(false);
  const [editIsSaving, setEditIsSaving] = useState(false);
  
  const { data: edgeType, isLoading: typeLoading, refetch: refetchEdgeType } = useEdgeType(id || '', tenant?.id || '');
  
  // Fetch subject and object node types for the relationship diagram
  const { data: subjectNodeType, isLoading: subjectLoading } = useNodeType(edgeType?.subject_node_type_id || '');
  const { data: objectNodeType, isLoading: objectLoading } = useNodeType(edgeType?.object_node_type_id || '');
  
  // React Flow state
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [editPropsOpen, setEditPropsOpen] = useState(false);
  const [editProps, setEditProps] = useState<PropertyDefinition[]>([]);
  const [editPropsSaving, setEditPropsSaving] = useState(false);

  // Initialize edit form
  const handleEditOpen = () => {
    if (edgeType) {
      setEditDescription(edgeType.description || '');
      setEditColor(edgeType.config?.color || '#1F2937');
      setEditIsActive(edgeType.is_active ?? true);
      setEditOpen(true);
    }
  };

  // Save edit changes
  const handleEditSave = async () => {
    if (!edgeType || !id || !tenant?.id) return;
    setEditIsSaving(true);
    try {
      const response = await fetch(`/api/edge-types/${id}?tenant_id=${tenant.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          description: editDescription,
          is_active: editIsActive,
          config: {
            ...(edgeType.config || {}),
            color: editColor,
          },
        }),
      });

      if (response.ok) {
        setEditOpen(false);
        refetchEdgeType();
      }
    } catch (error) {
      console.error('Failed to save edge type:', error);
    } finally {
      setEditIsSaving(false);
    }
  };

  // Initialize React Flow diagram whenever node types load
  useMemo(() => {
    if (!edgeType || !subjectNodeType || !objectNodeType) return;

    const newNodes: Node[] = [
      {
        id: 'subject',
        data: { 
          label: (
            <Box sx={{ textAlign: 'center', fontWeight: 'bold', fontSize: '0.9rem' }}>
              {subjectNodeType.catalog_type_name || 'Subject'}
            </Box>
          )
        },
        position: { x: 0, y: 0 },
        style: {
          background: theme.palette.primary.light,
          color: 'white',
          border: `2px solid ${theme.palette.primary.main}`,
          borderRadius: '8px',
          padding: '16px 24px',
          minWidth: '160px',
          fontWeight: 'bold',
        },
      },
      {
        id: 'object',
        data: { 
          label: (
            <Box sx={{ textAlign: 'center', fontWeight: 'bold', fontSize: '0.9rem' }}>
              {objectNodeType.catalog_type_name || 'Object'}
            </Box>
          )
        },
        position: { x: 350, y: 0 },
        style: {
          background: theme.palette.success.light,
          color: 'white',
          border: `2px solid ${theme.palette.success.main}`,
          borderRadius: '8px',
          padding: '16px 24px',
          minWidth: '160px',
          fontWeight: 'bold',
        },
      },
    ];

    const newEdges: Edge[] = [
      {
        id: `subject->object`,
        source: 'subject',
        target: 'object',
        label: edgeType.edge_type_name,
        animated: true,
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: edgeType.config?.color || theme.palette.grey[700],
        },
        style: {
          stroke: edgeType.config?.color || theme.palette.grey[700],
          strokeWidth: 3,
          strokeDasharray: 'none',
        },
        labelStyle: {
          background: theme.palette.background.paper,
          fontWeight: 'bold',
          fontSize: '12px',
          padding: '4px 8px',
          borderRadius: '4px',
          border: `1px solid ${theme.palette.divider}`,
        },
      },
    ];

    setNodes(newNodes);
    setEdges(newEdges);
  }, [edgeType, subjectNodeType, objectNodeType, setNodes, setEdges, theme]);

  // Get properties for the edge type
  const handleEditPropsOpen = () => {
    if (edgeType) {
      // Map EdgeProperty to PropertyDefinition
      const existingProps: PropertyDefinition[] = (edgeType.properties || []).map((p: any) => ({
        name: p.name || '',
        label: p.label || '',
        data_type: p.data_type || 'string',
        nullable: p.nullable ?? true,
        description: p.description || '',
        properties: p.properties || [], // Recursive support
      }));
      setEditProps(existingProps);
      setEditPropsOpen(true);
    }
  };

  const handleSaveProperties = async () => {
    if (!edgeType || !id || !tenant?.id) return;
    setEditPropsSaving(true);
    try {
      const response = await fetch(`/api/edge-types/${id}?tenant_id=${tenant.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          description: edgeType.description,
          config: edgeType.config || {},
          properties: editProps,
        }),
      });

      if (response.ok) {
        setEditPropsOpen(false);
        refetchEdgeType();
      }
    } catch (error) {
      console.error('Failed to save properties:', error);
    } finally {
      setEditPropsSaving(false);
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

  if (typeLoading || subjectLoading || objectLoading) {
    return (
      <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto' }}>
        <Skeleton variant="rectangular" height={260} sx={{ borderRadius: 3, bgcolor: isDark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.04)' }} />
      </Box>
    );
  }

  if (!edgeType) {
    return (
      <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto', color: C.text }}>
        <Typography>Edge Type not found</Typography>
      </Box>
    );
  }

  if (!subjectNodeType || !objectNodeType) {
    return (
      <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto', color: C.text }}>
        <Typography>Node type information not found</Typography>
      </Box>
    );
  }

  const edgeColor = edgeType.config?.color;
  const isCore = edgeType.type === 'core';

  return (
    <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto', minHeight: '100vh', color: C.text, bgcolor: C.bg }}>
      {/* Breadcrumbs */}
      <Breadcrumbs sx={{ mb: 3, '& .MuiBreadcrumbs-separator': { color: C.textMuted } }}>
        <Link 
          color="inherit" 
          component="button" 
          onClick={() => navigate('/catalog/edge-types')}
          underline="hover"
          sx={{ color: C.textMuted, '&:hover': { color: C.accent }, cursor: 'pointer', fontSize: '0.9rem' }}
        >
          Edge Types
        </Link>
        <Typography sx={{ color: C.text, fontSize: '0.9rem', fontWeight: 600 }}>{edgeType.edge_type_name}</Typography>
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
          borderTop: edgeColor ? `3px solid ${edgeColor}` : `1px solid ${C.border}`,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <IconButton 
              onClick={() => navigate('/catalog/edge-types')} 
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
                {edgeType.edge_type_name}
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
                <span style={{
                  display: 'inline-flex', alignItems: 'center', padding: '2px 8px',
                  borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                  color: edgeType.is_active ? C.success : C.textMuted,
                  background: edgeType.is_active ? (isDark ? `${C.success}18` : `${C.success}12`) : 'transparent',
                  border: `1px solid ${edgeType.is_active ? C.success : C.border}44`,
                  fontFamily: 'monospace', textTransform: 'uppercase',
                }}>
                  {edgeType.is_active ? 'Active' : 'Inactive'}
                </span>
                {edgeType.type && (isCore ? <CoreIcon fontSize="small" /> : <CustomIcon fontSize="small" />)}
                <Typography variant="caption" sx={{ color: C.textMuted, fontFamily: 'monospace', ml: 0.5 }}>
                  ID: {edgeType.id}
                </Typography>
              </Box>
            </Box>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            {edgeColor && (
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 1.5, py: 0.5, borderRadius: 2, bgcolor: isDark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.03)', border: `1px solid ${C.border}` }}>
                <Box sx={{ width: 14, height: 14, borderRadius: '50%', bgcolor: edgeColor }} />
                <Typography variant="caption" sx={{ color: C.textMuted, fontFamily: 'monospace' }}>{edgeColor}</Typography>
              </Box>
            )}
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
          <Grid size={{ xs: 12, lg: 7 }}>
            <Typography variant="subtitle2" fontWeight="bold" sx={{ display: 'flex', alignItems: 'center', gap: 0.8, color: C.textMuted, mb: 1, textTransform: 'uppercase', letterSpacing: '0.04em' }}>
              <DescriptionOutlinedIcon fontSize="small"/> Description
            </Typography>
            <Typography variant="body1" sx={{ color: C.text, lineHeight: 1.6 }} paragraph>
              {edgeType.description || 'No description provided for this edge type.'}
            </Typography>
            
            {/* Subject/Object Node Types */}
            <Box sx={{ mt: 2.5 }}>
              <Typography variant="subtitle2" sx={{ color: C.textMuted, mb: 1, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.04em', fontSize: '0.75rem' }}>
                Relationship Definition
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                <span style={{
                  display: 'inline-flex', alignItems: 'center', padding: '4px 12px',
                  borderRadius: 8, fontSize: 12, fontWeight: 700,
                  color: C.blue, background: isDark ? 'rgba(96,165,250,0.12)' : 'rgba(96,165,250,0.08)',
                  border: `1px solid rgba(96,165,250,0.3)`,
                }}>
                  {edgeType.subject_node_type_name || subjectNodeType.catalog_type_name}
                </span>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8, px: 1 }}>
                  <Typography variant="body2" fontWeight="bold" sx={{ color: edgeType.config?.color || C.accent, fontFamily: 'monospace' }}>
                    {edgeType.edge_type_name}
                  </Typography>
                  <ArrowForwardIcon sx={{ fontSize: 14, color: C.textMuted }} />
                </Box>
                <span style={{
                  display: 'inline-flex', alignItems: 'center', padding: '4px 12px',
                  borderRadius: 8, fontSize: 12, fontWeight: 700,
                  color: C.purple, background: isDark ? 'rgba(167,139,250,0.12)' : 'rgba(167,139,250,0.08)',
                  border: `1px solid rgba(167,139,250,0.3)`,
                }}>
                  {edgeType.object_node_type_name || objectNodeType.catalog_type_name}
                </span>
              </Box>
            </Box>

            {/* React Flow Diagram */}
            <Box sx={{ mt: 3 }}>
              <Typography variant="subtitle2" sx={{ color: C.textMuted, mb: 1, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.04em', fontSize: '0.75rem' }}>
                Relationship Flow
              </Typography>
              <Paper 
                elevation={0}
                sx={{ 
                  height: 380, 
                  borderRadius: 2.5, 
                  overflow: 'hidden', 
                  border: `1px solid ${C.border}`,
                  bgcolor: isDark ? '#0A0C12' : '#F8FAFC'
                }}
              >
                <ReactFlow 
                  nodes={nodes} 
                  edges={edges}
                  onNodesChange={onNodesChange}
                  onEdgesChange={onEdgesChange}
                  fitView
                >
                  <Background color={isDark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.08)'} gap={20} />
                  <Controls />
                </ReactFlow>
              </Paper>
            </Box>
          </Grid>
          
          <Grid size={{ xs: 12, lg: 5 }}>
            <Stack spacing={2.5}>
              {/* Properties Sidebar */}
              <Card 
                elevation={0} 
                sx={{ 
                  borderRadius: 2.5, 
                  bgcolor: isDark ? '#0F1117' : '#F8FAFC',
                  border: `1px solid ${C.border}`,
                  minHeight: 280
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
                  {edgeType.properties && edgeType.properties.length > 0 ? (
                    <Stack divider={<Divider sx={{ borderColor: C.border }} />}>
                      {edgeType.properties.map((prop: any) => (
                        <Box key={prop.name} sx={{ p: 1.5, '&:hover': { bgcolor: isDark ? 'rgba(255,255,255,0.02)' : 'rgba(0,0,0,0.02)' } }}>
                          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <Typography variant="body2" fontWeight="bold" sx={{ color: C.text }}>{prop.label || prop.name}</Typography>
                            <span style={{
                              display: 'inline-flex', alignItems: 'center', padding: '1px 6px',
                              borderRadius: 4, fontSize: 10, fontWeight: 700,
                              color: C.blue, background: isDark ? 'rgba(96,165,250,0.12)' : 'rgba(96,165,250,0.08)',
                              border: `1px solid rgba(96,165,250,0.3)`, fontFamily: 'monospace'
                            }}>
                              {prop.data_type || 'string'}
                            </span>
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

              {/* Info Card */}
              <Card 
                elevation={0} 
                sx={{ 
                  borderRadius: 2.5,
                  bgcolor: isDark ? '#0F1117' : '#F8FAFC',
                  border: `1px solid ${C.border}`,
                }}
              >
                <Box sx={{ p: 1.5, px: 2, borderBottom: `1px solid ${C.border}` }}>
                  <Typography variant="subtitle2" sx={{ color: C.textMuted, fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.04em' }}>
                    System Metadata
                  </Typography>
                </Box>
                <Box sx={{ p: 2 }}>
                  <Stack spacing={1.5}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Typography variant="body2" sx={{ color: C.textMuted }}>ID</Typography>
                      <Typography variant="caption" sx={{ color: C.text, fontFamily: 'monospace' }}>{edgeType.id}</Typography>
                    </Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Typography variant="body2" sx={{ color: C.textMuted }}>Status</Typography>
                      <span style={{
                        display: 'inline-flex', alignItems: 'center', padding: '1px 6px',
                        borderRadius: 4, fontSize: 10, fontWeight: 700,
                        color: edgeType.is_active ? C.success : C.textMuted,
                        background: edgeType.is_active ? (isDark ? `${C.success}18` : `${C.success}12`) : 'transparent',
                        border: `1px solid ${edgeType.is_active ? C.success : C.border}44`,
                        fontFamily: 'monospace', textTransform: 'uppercase',
                      }}>
                        {edgeType.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Typography variant="body2" sx={{ color: C.textMuted }}>Last Updated</Typography>
                      <Typography variant="caption" sx={{ color: C.textMuted, fontFamily: 'monospace' }}>
                        {new Date(edgeType.updated_at).toLocaleDateString()}
                      </Typography>
                    </Box>
                  </Stack>
                </Box>
              </Card>
            </Stack>
          </Grid>
        </Grid>
      </Paper>

      {/* Edit Edge Type Dialog */}
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
          Edit Edge Type: {edgeType?.edge_type_name}
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
              placeholder="Enter edge type description..."
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
                label="Edge Type Color (for visualization)"
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

      {/* Edit Properties Dialog */}
      <Dialog 
        open={editPropsOpen} 
        onClose={() => setEditPropsOpen(false)}
        maxWidth="lg"
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
            <Typography variant="h6" fontWeight="bold">Manage Edge Type Properties</Typography>
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
            startIcon={editPropsSaving ? <Skeleton variant="circular" width={20} height={20} /> : null}
          >
            {editPropsSaving ? 'Saving...' : 'Save All Properties'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
