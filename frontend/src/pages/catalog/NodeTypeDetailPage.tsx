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
import { useNodeType, useNodesByType } from '../../api/nodeTypes';
import { useEdgeTypes } from '../../api/edgeTypes';
import { useTenant } from '../../contexts/TenantContext';
import { ProfessionalColorPicker } from '../../components/ProfessionalColorPicker';
import { PropertyEditor, PropertyDefinition } from '../../components/PropertyEditor';

export const NodeTypeDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const theme = useTheme();
  const { tenant } = useTenant();
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
  
  const { data: nodeType, isLoading: typeLoading, refetch: refetchNodeType } = useNodeType(id || '');

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

  if (typeLoading) {
    return <Box sx={{ p: 4 }}><Skeleton variant="rectangular" height={200} /></Box>;
  }

  if (!nodeType) {
    return <Box sx={{ p: 4 }}><Typography>Node Type not found</Typography></Box>;
  }

  const nodeColor = nodeType.config?.color;

  return (
    <Box sx={{ p: 4, maxWidth: 1600, mx: 'auto' }}>
      {/* Breadcrumbs */}
      <Breadcrumbs sx={{ mb: 3 }}>
        <Link 
          color="inherit" 
          component="button" 
          onClick={() => navigate('/catalog/node-types')}
          underline="hover"
        >
          Node Types
        </Link>
        <Typography color="text.primary">{nodeType.catalog_type_name}</Typography>
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
          borderTop: nodeColor ? `4px solid ${nodeColor}` : 'none'
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <IconButton onClick={() => navigate('/catalog/node-types')} sx={{ bgcolor: 'background.paper', border: `1px solid ${theme.palette.divider}` }}>
              <ArrowBackIcon />
            </IconButton>
            <Box>
              <Typography variant="h3" fontWeight="bold" gutterBottom>
                {nodeType.catalog_type_name}
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                {nodeColor && (
                  <Box sx={{ width: 18, height: 18, borderRadius: '50%', bgcolor: nodeColor, border: '1px solid rgba(0,0,0,0.12)' }} title={`Color: ${nodeColor}`} />
                )}
                <Chip label={nodeType.is_active ? 'Active' : 'Inactive'} color={nodeType.is_active ? 'success' : 'default'} size="small" />
                <Chip
                  label={nodeType.type === 'core' || nodeType.core ? 'Core' : 'Custom'}
                  color={nodeType.type === 'core' || nodeType.core ? 'primary' : 'secondary'}
                  size="small"
                />
                <Typography variant="body2" color="text.secondary">
                  ID: {nodeType.id}
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
          </Box>
        </Box>
        
        <Grid container spacing={4} sx={{ mt: 2 }}>
            <Grid item xs={12} md={8}>
                 <Typography variant="subtitle1" fontWeight="bold" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <DescriptionOutlinedIcon fontSize="small"/> Description
                 </Typography>
                 <Typography variant="body1" color="text.secondary" paragraph>
                    {nodeType.description || 'No description provided for this node type.'}
                 </Typography>
            </Grid>
            
            <Grid item xs={12} md={4}>
                {/* Properties Card */}
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
                    {nodeType.config?.properties && (Array.isArray(nodeType.config.properties) ? nodeType.config.properties.length > 0 : Object.keys(nodeType.config.properties).length > 0) ? (
                      <Stack divider={<Divider />}>
                        {(Array.isArray(nodeType.config.properties) ? nodeType.config.properties : Object.entries(nodeType.config.properties).map(([key, val]: [string, any]) => ({ name: key, ...val }))).map((prop: any) => (
                          <Box key={prop.name} sx={{ p: 2, '&:hover': { bgcolor: alpha(theme.palette.action.hover, 0.5) } }}>
                            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                              <Typography variant="body2" fontWeight="bold">{prop.label || prop.name}</Typography>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <Chip label={prop.data_type || 'string'} size="small" variant="outlined" sx={{ height: 20, fontSize: '0.65rem' }} />
                                <IconButton size="small" color="error" onClick={() => handleDeleteProperty(prop.name)}>
                                  <DeleteIcon fontSize="small" />
                                </IconButton>
                              </Box>
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
            </Grid>
        </Grid>
      </Paper>

      {/* Associated Edge Types Section */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h5" fontWeight="bold">
          Associated Edge Types ({associatedEdgeTypes?.length || 0})
        </Typography>
      </Box>

      {associatedEdgeTypes && associatedEdgeTypes.length > 0 ? (
        <TableContainer component={Paper} elevation={0} sx={{ border: `1px solid ${theme.palette.divider}`, borderRadius: 2 }}>
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: alpha(theme.palette.primary.main, 0.05) }}>
                <TableCell sx={{ fontWeight: 'bold' }}>Subject</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Predicate</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Object</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Type</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {associatedEdgeTypes.map((edgeType) => (
                <TableRow key={edgeType.id} sx={{ '&:hover': { bgcolor: 'grey.100' } }}>
                  <TableCell>
                    <Chip
                      label={edgeType.subject_node_type_name || 'Unknown'}
                      size="small"
                      color="primary"
                      variant="outlined"
                      sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                    />
                  </TableCell>
                  <TableCell sx={{ fontWeight: 500 }}>
                    {edgeType.edge_type_name}
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={edgeType.object_node_type_name || 'Unknown'}
                      size="small"
                      color="secondary"
                      variant="outlined"
                      sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={edgeType.type === 'core' ? 'Core' : 'Custom'}
                      size="small"
                      color={edgeType.type === 'core' ? 'primary' : 'warning'}
                      variant={edgeType.type === 'core' ? 'filled' : 'outlined'}
                      sx={{ fontWeight: 600 }}
                    />
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
            borderRadius: 2, 
            border: `1px solid ${theme.palette.divider}`,
            textAlign: 'center'
          }}
        >
          <Typography color="text.secondary">No edge types associated with this node type.</Typography>
        </Paper>
      )}

      {/* Node Details Dialog */}
      {selectedNode && (
        <Dialog 
          open={!!selectedNode} 
          onClose={() => setSelectedNode(null)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>
            {selectedNode.node_name}
            <IconButton
              aria-label="close"
              onClick={() => setSelectedNode(null)}
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
             <Typography variant="subtitle2" gutterBottom>Qualified Path</Typography>
             <Paper variant="outlined" sx={{ p: 1, bgcolor: 'grey.50', mb: 2, fontFamily: 'monospace' }}>
                {selectedNode.qualified_path}
             </Paper>

             {selectedNode.description && (
               <>
                 <Typography variant="subtitle2" gutterBottom>Description</Typography>
                 <Typography variant="body2" paragraph>{selectedNode.description}</Typography>
               </>
             )}

             <Typography variant="subtitle2" gutterBottom>Properties</Typography>
             <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50', maxHeight: 400, overflow: 'auto' }}>
                <Box component="pre" sx={{ margin: 0, fontSize: '0.875rem', whiteSpace: 'pre-wrap', wordWrap: 'break-word' }}>
                  {JSON.stringify(selectedNode.properties, null, 2)}
                </Box>
             </Paper>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setSelectedNode(null)}>Close</Button>
          </DialogActions>
        </Dialog>
      )}

      {/* Edit Node Type Dialog */}
      <Dialog 
        open={editOpen} 
        onClose={() => setEditOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          Edit Node Type: {nodeType?.catalog_type_name}
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
              placeholder="Enter node type description..."
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

      {/* Properties Editor Dialog */}
      <Dialog open={editPropsOpen} onClose={() => setEditPropsOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <SettingsInputComponentIcon color="primary" />
            <Typography variant="h6">Manage Node Type Properties</Typography>
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
          >
            {editPropsSaving ? 'Saving...' : 'Save All Properties'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
