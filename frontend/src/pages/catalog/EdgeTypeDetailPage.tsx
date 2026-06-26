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

  if (typeLoading || subjectLoading || objectLoading) {
    return <Box sx={{ p: 4 }}><Skeleton variant="rectangular" height={400} /></Box>;
  }

  if (!edgeType) {
    return <Box sx={{ p: 4 }}><Typography>Edge Type not found</Typography></Box>;
  }

  if (!subjectNodeType || !objectNodeType) {
    return <Box sx={{ p: 4 }}><Typography>Node type information not found</Typography></Box>;
  }

  const edgeColor = edgeType.config?.color;

  return (
    <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto' }}>
      {/* Breadcrumbs */}
      <Breadcrumbs sx={{ mb: 3 }}>
        <Link 
          color="inherit" 
          component="button" 
          onClick={() => navigate('/catalog/edge-types')}
          underline="hover"
        >
          Edge Types
        </Link>
        <Typography color="text.primary">{edgeType.edge_type_name}</Typography>
      </Breadcrumbs>

      {/* Header Card */}
      <Paper 
        elevation={0}
        sx={{ 
          p: 4, 
          mb: 4, 
          borderRadius: 4, 
          background: `linear-gradient(135deg, ${alpha(theme.palette.primary.main, 0.05)} 0%, ${theme.palette.background.paper} 100%)`,
          border: `1px solid ${theme.palette.divider}`,
          borderTop: edgeColor ? `4px solid ${edgeColor}` : 'none'
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <IconButton onClick={() => navigate('/catalog/edge-types')} sx={{ bgcolor: 'background.paper', border: `1px solid ${theme.palette.divider}` }}>
              <ArrowBackIcon />
            </IconButton>
            <Box>
              <Typography variant="h3" fontWeight="bold" gutterBottom>
                {edgeType.edge_type_name}
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                <Chip label={edgeType.is_active ? 'Active' : 'Inactive'} color={edgeType.is_active ? 'success' : 'default'} size="small" />
                {edgeType.type && (
                  <Chip 
                    label={edgeType.type === 'core' ? 'Core' : 'Custom'} 
                    color={edgeType.type === 'core' ? 'primary' : 'warning'} 
                    size="small" 
                  />
                )}
                <Typography variant="body2" color="text.secondary">
                  ID: {edgeType.id}
                </Typography>
              </Box>
            </Box>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <Button 
              variant="contained" 
              startIcon={<EditIcon />}
              onClick={handleEditOpen}
              sx={{ textTransform: 'none' }}
            >
              Edit
            </Button>
            {edgeColor && (
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Box sx={{ width: 20, height: 20, borderRadius: 1, bgcolor: edgeColor, border: `1px solid ${theme.palette.divider}` }} />
                <Typography variant="caption" color="text.secondary">{edgeColor}</Typography>
              </Box>
            )}
          </Box>
        </Box>
        
        <Grid container spacing={4} sx={{ mt: 2 }}>
          <Grid item xs={12} lg={8}>
            <Typography variant="subtitle1" fontWeight="bold" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <DescriptionOutlinedIcon fontSize="small"/> Description
            </Typography>
            <Typography variant="body1" color="text.secondary" paragraph>
              {edgeType.description || 'No description provided for this edge type.'}
            </Typography>
            
            {/* Subject/Object Node Types */}
            <Box sx={{ mt: 3 }}>
              <Typography variant="subtitle2" gutterBottom fontWeight="bold">Relationship Definition</Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
                <Chip 
                  label={edgeType.subject_node_type_name || subjectNodeType.catalog_type_name} 
                  variant="filled"
                  color="primary"
                  sx={{ fontWeight: 'bold' }}
                />
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography variant="body2" fontWeight="bold" sx={{ color: edgeType.config?.color || theme.palette.grey[700] }}>
                    {edgeType.edge_type_name}
                  </Typography>
                  <Typography variant="body2">→</Typography>
                </Box>
                <Chip 
                  label={edgeType.object_node_type_name || objectNodeType.catalog_type_name} 
                  variant="filled"
                  color="success"
                  sx={{ fontWeight: 'bold' }}
                />
              </Box>
            </Box>

            {/* React Flow Diagram */}
            <Box sx={{ mt: 4, mb: 3 }}>
              <Typography variant="subtitle2" gutterBottom fontWeight="bold">Relationship Flow</Typography>
              <Paper 
                sx={{ 
                  height: 450, 
                  borderRadius: 2, 
                  overflow: 'hidden', 
                  border: `1px solid ${theme.palette.divider}`,
                  backgroundColor: alpha(theme.palette.background.default, 0.5)
                }}
              >
                <ReactFlow 
                  nodes={nodes} 
                  edges={edges}
                  onNodesChange={onNodesChange}
                  onEdgesChange={onEdgesChange}
                  fitView
                >
                  <Background color={theme.palette.divider} gap={16} />
                  <Controls />
                </ReactFlow>
              </Paper>
            </Box>
          </Grid>
          
          <Grid item xs={12} lg={4}>
            <Stack spacing={3}>
              {/* Properties Sidebar */}
              <Card variant="outlined" sx={{ borderRadius: 3, height: '100%', minHeight: 450 }}>
                <Box sx={{ 
                  p: 2, 
                  borderBottom: `1px solid ${theme.palette.divider}`, 
                  bgcolor: alpha(theme.palette.primary.main, 0.05),
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center'
                }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <SettingsInputComponentIcon fontSize="small" color="primary" />
                    <Typography variant="subtitle1" fontWeight="bold">Properties</Typography>
                  </Box>
                  <IconButton size="small" color="primary" onClick={handleEditPropsOpen}>
                    <EditIcon fontSize="small" />
                  </IconButton>
                </Box>
                <Box sx={{ p: 0 }}>
                  {edgeType.properties && edgeType.properties.length > 0 ? (
                    <Stack divider={<Divider />}>
                      {edgeType.properties.map((prop: any) => (
                        <Box key={prop.name} sx={{ p: 2, '&:hover': { bgcolor: alpha(theme.palette.action.hover, 0.5) } }}>
                          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <Typography variant="body2" fontWeight="bold">{prop.label || prop.name}</Typography>
                            <Chip label={prop.data_type || 'string'} size="small" variant="outlined" sx={{ height: 20, fontSize: '0.65rem' }} />
                          </Box>
                          {prop.description && (
                            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                              {prop.description}
                            </Typography>
                          )}
                          {prop.properties && prop.properties.length > 0 && (
                            <Box sx={{ mt: 1, pl: 2, borderLeft: `2px solid ${theme.palette.divider}` }}>
                              <Typography variant="caption" color="text.secondary">
                                {prop.properties.length} nested fields
                              </Typography>
                            </Box>
                          )}
                        </Box>
                      ))}
                    </Stack>
                  ) : (
                    <Box sx={{ p: 4, textAlign: 'center' }}>
                      <InfoOutlinedIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                      <Typography variant="body2" color="text.secondary">No properties defined.</Typography>
                      <Button size="small" sx={{ mt: 2 }} onClick={handleEditPropsOpen}>Add Property</Button>
                    </Box>
                  )}
                </Box>
              </Card>

              {/* Info Card */}
              <Card variant="outlined" sx={{ borderRadius: 3 }}>
                <Box sx={{ p: 2, borderBottom: `1px solid ${theme.palette.divider}`, bgcolor: alpha(theme.palette.grey[500], 0.05) }}>
                  <Typography variant="subtitle2">System Metadata</Typography>
                </Box>
                <Box sx={{ p: 2 }}>
                  <Stack spacing={1}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                      <Typography variant="body2" color="text.secondary">ID</Typography>
                      <Typography variant="caption" fontFamily="monospace">{edgeType.id}</Typography>
                    </Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                      <Typography variant="body2" color="text.secondary">Status</Typography>
                      <Chip label={edgeType.is_active ? 'Active' : 'Inactive'} size="small" color={edgeType.is_active ? 'success' : 'default'} />
                    </Box>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                      <Typography variant="body2" color="text.secondary">Last Updated</Typography>
                      <Typography variant="caption">{new Date(edgeType.updated_at).toLocaleDateString()}</Typography>
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
      >
        <DialogTitle>
          Edit Edge Type: {edgeType?.edge_type_name}
          <IconButton
            aria-label="close"
            onClick={() => setEditOpen(false)}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
              color: (theme) => theme.palette.grey[500],
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers>
          <Stack spacing={3} sx={{ mt: 2 }}>
            {/* Active Status */}
            <FormControl fullWidth>
              <FormLabel>Status</FormLabel>
              <Box sx={{ display: 'flex', gap: 2, mt: 1 }}>
                <Button
                  variant={editIsActive ? 'contained' : 'outlined'}
                  color="success"
                  onClick={() => setEditIsActive(true)}
                  sx={{ flex: 1, textTransform: 'none' }}
                >
                  Active
                </Button>
                <Button
                  variant={!editIsActive ? 'contained' : 'outlined'}
                  color="error"
                  onClick={() => setEditIsActive(false)}
                  sx={{ flex: 1, textTransform: 'none' }}
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
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button 
            onClick={handleEditSave} 
            variant="contained" 
            disabled={editIsSaving}
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
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <SettingsInputComponentIcon color="primary" />
            <Typography variant="h6">Manage Edge Type Properties</Typography>
          </Box>
          <IconButton
            aria-label="close"
            onClick={() => setEditPropsOpen(false)}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
              color: (theme) => theme.palette.grey[500],
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers sx={{ bgcolor: alpha(theme.palette.grey[500], 0.02) }}>
          <Box sx={{ py: 2 }}>
            <PropertyEditor 
              properties={editProps} 
              onChange={setEditProps} 
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setEditPropsOpen(false)}>Cancel</Button>
          <Button 
            onClick={handleSaveProperties} 
            variant="contained" 
            disabled={editPropsSaving}
            startIcon={editPropsSaving ? <Skeleton variant="circular" width={20} height={20} /> : null}
          >
            {editPropsSaving ? 'Saving...' : 'Save All Properties'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
