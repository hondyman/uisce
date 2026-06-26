import React, { useState } from 'react';
import {
  Box,
  Button,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  IconButton,
  Chip,
  Tooltip,
  Switch,
  FormControlLabel,
  Typography,
} from '@mui/material';
import Editor from '@monaco-editor/react';
import AddIcon from '@mui/icons-material/Add';
import ControlPointIcon from '@mui/icons-material/ControlPoint';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { useMutation } from '@apollo/client';
import { useNotification } from '../../../hooks/useNotification';
import { CREATE_TENANT_INSTANCE, UPDATE_TENANT_INSTANCE, DELETE_TENANT_INSTANCE } from '../../../graphql/mutations/tenantMutations';

interface TenantInstance {
  id: string;
  instance_name: string;
  display_name: string;
  description?: string;
  url?: string;
  status?: string;
  is_active: boolean;
}

interface InstancesTabProps {
  tenantId: string;
  instances: TenantInstance[];
  onRefetch: () => void;
}

export default function InstancesTab({ tenantId, instances, onRefetch }: InstancesTabProps) {
  const notification = useNotification();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false);
  const [selectedInstance, setSelectedInstance] = useState<TenantInstance | null>(null);
  const [editingInstance, setEditingInstance] = useState<TenantInstance | null>(null);
  const [formData, setFormData] = useState({
    instance_name: '',
    display_name: '',
    description: '',
    url: '',
    is_active: true,
    config: '{}',
  });

  const [createInstance] = useMutation(CREATE_TENANT_INSTANCE);
  const [updateInstance] = useMutation(UPDATE_TENANT_INSTANCE);
  const [deleteInstance] = useMutation(DELETE_TENANT_INSTANCE);

  const handleOpenDialog = (instance?: TenantInstance) => {
    if (instance) {
      setEditingInstance(instance);
      setFormData({
        instance_name: instance.instance_name,
        display_name: instance.display_name,
        description: instance.description || '',
        url: instance.url || '',
        is_active: instance.is_active ?? true,
        config: typeof instance.config === 'string' ? instance.config : JSON.stringify(instance.config || {}, null, 2),
      });
    } else {
      setEditingInstance(null);
      setFormData({
        instance_name: '',
        display_name: '',
        description: '',
        url: '',
        is_active: true,
        config: '{}',
      });
    }
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setEditingInstance(null);
  };

  const handleViewDetails = (instance: TenantInstance) => {
    setSelectedInstance(instance);
    setDetailsDialogOpen(true);
  };

  const handleCloseDetailsDialog = () => {
    setDetailsDialogOpen(false);
    setSelectedInstance(null);
  };

  const handleSave = async () => {
    try {
      let configObj = {};
      try {
        configObj = JSON.parse(formData.config || '{}');
      } catch (e) {
        notification.error('Invalid JSON in config field');
        return;
      }

      const variables = {
        instance_name: formData.instance_name,
        display_name: formData.display_name,
        description: formData.description,
        url: formData.url,
        is_active: formData.is_active,
        config: configObj,
      };

      if (editingInstance) {
        await updateInstance({
          variables: {
            id: editingInstance.id,
            ...variables,
          },
        });
        notification.success('Instance updated successfully');
      } else {
        await createInstance({
          variables: {
            tenant_id: tenantId,
            ...variables,
          },
        });
        notification.success('Instance created successfully');
      }
      handleCloseDialog();
      onRefetch();
    } catch (error: any) {
      notification.error(error.message || 'Failed to save instance');
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this instance?')) return;
    
    try {
      await deleteInstance({ variables: { id } });
      notification.success('Instance deleted successfully');
      onRefetch();
    } catch (error: any) {
      notification.error(error.message || 'Failed to delete instance');
    }
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 2 }}>
        <Tooltip title="Add Instance">
          <IconButton
            color="primary"
            onClick={() => handleOpenDialog()}
            sx={{ fontSize: '2rem' }}
          >
            <ControlPointIcon sx={{ fontSize: '2rem' }} />
          </IconButton>
        </Tooltip>
      </Box>

      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>Display Name</TableCell>
            <TableCell>URL</TableCell>
            <TableCell>Active</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {instances.map((instance) => (
            <TableRow key={instance.id}>
              <TableCell>{instance.instance_name}</TableCell>
              <TableCell>{instance.display_name}</TableCell>
              <TableCell>{instance.url || '—'}</TableCell>
              <TableCell>
                {instance.is_active ? (
                  <CheckCircleIcon sx={{ color: '#4caf50', fontSize: '20px' }} />
                ) : (
                  <CancelIcon sx={{ color: '#f44336', fontSize: '20px' }} />
                )}
              </TableCell>
              <TableCell align="right">
                <IconButton size="small" onClick={() => handleViewDetails(instance)} color="primary" title="View details">
                  <VisibilityIcon fontSize="small" />
                </IconButton>
                <IconButton size="small" onClick={() => handleOpenDialog(instance)} color="primary" title="Edit">
                  <EditIcon fontSize="small" />
                </IconButton>
                <IconButton size="small" onClick={() => handleDelete(instance.id)} color="error" title="Delete">
                  <DeleteIcon fontSize="small" />
                </IconButton>
              </TableCell>
            </TableRow>
          ))}
          {instances.length === 0 && (
            <TableRow>
              <TableCell colSpan={5} align="center">
                No instances found. Click "Add Instance" to create one.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>{editingInstance ? 'Edit Instance' : 'Add Instance'}</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              label="Instance Name"
              value={formData.instance_name}
              onChange={(e) => setFormData({ ...formData, instance_name: e.target.value })}
              required
              helperText="Internal identifier (e.g., dev, staging, prod)"
            />
            <TextField
              label="Display Name"
              value={formData.display_name}
              onChange={(e) => setFormData({ ...formData, display_name: e.target.value })}
              required
            />
            <TextField
              label="Description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              multiline
              rows={2}
            />
            <TextField
              label="URL"
              value={formData.url}
              onChange={(e) => setFormData({ ...formData, url: e.target.value })}
              placeholder="https://..."
            />
            <FormControlLabel
              control={
                <Switch
                  checked={formData.is_active ?? true}
                  onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                />
              }
              label="Active"
              sx={{ 
                display: 'flex',
                alignItems: 'center',
                py: 1,
              }}
            />
            <Box>
              <Typography variant="subtitle2" sx={{ mb: 1 }}>Configuration (JSON)</Typography>
              <Editor
                height="250px"
                defaultLanguage="json"
                value={formData.config}
                onChange={(value) => setFormData({ ...formData, config: value || '{}' })}
                options={{
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  fontSize: 12,
                  tabSize: 2,
                  automaticLayout: true,
                }}
                theme="vs-dark"
              />
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSave} variant="contained" disabled={!formData.instance_name || !formData.display_name}>
            {editingInstance ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Details Dialog */}
      <Dialog open={detailsDialogOpen} onClose={handleCloseDetailsDialog} maxWidth="sm" fullWidth>
        <DialogTitle>Instance Details</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 2 }}>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>Instance Name</Typography>
              <Typography variant="body2">{selectedInstance?.instance_name || '—'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>Display Name</Typography>
              <Typography variant="body2">{selectedInstance?.display_name || '—'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>Description</Typography>
              <Typography variant="body2">{selectedInstance?.description || '—'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>URL</Typography>
              <Typography variant="body2">{selectedInstance?.url || '—'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>Active</Typography>
              <Typography variant="body2">
                {selectedInstance?.is_active ? (
                  <span style={{ color: '#4caf50' }}>✓ Active</span>
                ) : (
                  <span style={{ color: '#f44336' }}>✗ Inactive</span>
                )}
              </Typography>
            </Box>
            {selectedInstance?.config && typeof selectedInstance.config === 'object' && Object.keys(selectedInstance.config).length > 0 && (
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>Configuration</Typography>
                <Box sx={{ bgcolor: '#f5f5f5', p: 1, borderRadius: 1, fontSize: '12px', fontFamily: 'monospace', overflow: 'auto' }}>
                  <pre style={{ margin: 0 }}>{JSON.stringify(selectedInstance.config, null, 2)}</pre>
                </Box>
              </Box>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDetailsDialog}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
