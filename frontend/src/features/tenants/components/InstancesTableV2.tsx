import React, { useState } from 'react';
import { useMutation } from '@apollo/client';
import { DELETE_CONNECTION } from '../../../graphql/mutations/tenantMutations';
import {
  Box,
  Button,
  Card,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Stack,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Link,
  TableSortLabel,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  FilterList as FilterIcon,
  Link as LinkIcon,
} from '@mui/icons-material';
import type { TenantInstance } from '../../../types';
import InstanceResourcesDialog from './InstanceResourcesDialog';

// Extended interface with linked resources info
export interface EnrichedTenantInstance extends TenantInstance {
  linkedResources: {
    productCount: number;
    connectionCount: number;
    details: any[]; // ResourceStats[]
  };
}

interface InstancesTableProps {
  instances: EnrichedTenantInstance[]; // Updated type
  onAddInstance?: () => void;
  onEditInstance?: (instance: TenantInstance) => void;
  onDeleteInstance?: (instanceId: string) => void;
  onReload?: () => void;
}

export const InstancesTable: React.FC<InstancesTableProps> = ({
  instances,
  onAddInstance,
  onEditInstance,
  onDeleteInstance,
  onReload,
}) => {
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [deleteConfirmInstance, setDeleteConfirmInstance] = useState<TenantInstance | null>(null);
  
  // Sorting State
  const [sortBy, setSortBy] = useState('name');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');
  
  const handleSort = (property: string) => {
    const isAsc = sortBy === property && sortOrder === 'asc';
    setSortOrder(isAsc ? 'desc' : 'asc');
    setSortBy(property);
  };

  const sortedInstances = [...instances].sort((a, b) => {
    let aValue: any = '';
    let bValue: any = '';

    switch (sortBy) {
      case 'name':
        aValue = (a.display_name || a.instance_name || '').toLowerCase();
        bValue = (b.display_name || b.instance_name || '').toLowerCase();
        break;
      case 'environment':
        aValue = (a as any).environment?.toLowerCase() || '';
        bValue = (b as any).environment?.toLowerCase() || '';
        break;
      case 'status':
        aValue = a.is_active ? 1 : 0;
        bValue = b.is_active ? 1 : 0;
        break;
      default:
        return 0;
    }

    if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1;
    if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1;
    return 0;
  });
  
  // Dialog State
  const [resourcesDialogOpen, setResourcesDialogOpen] = useState(false);
  const [selectedInstanceResources, setSelectedInstanceResources] = useState<{
    name: string;
    resources: any[];
  } | null>(null);

  const handleDeleteClick = (instance: TenantInstance) => {
    setDeleteConfirmInstance(instance);
    setDeleteConfirmOpen(true);
  };

  const handleConfirmDelete = () => {
    if (deleteConfirmInstance && onDeleteInstance) {
      onDeleteInstance(deleteConfirmInstance.id);
    }
    setDeleteConfirmOpen(false);
    setDeleteConfirmInstance(null);
  };

  const [deleteConnection] = useMutation(DELETE_CONNECTION);

  const handleDeleteConnection = async (connectionId: string) => {
    try {
      if (confirm('Are you sure you want to delete this connection?')) {
        await deleteConnection({ variables: { id: connectionId } });
        
        // Update local state to remove the connection immediately from view
        if (selectedInstanceResources) {
          const updatedResources = selectedInstanceResources.resources.map(res => ({
            ...res,
            connections: res.connections.filter((c: any) => c.id !== connectionId)
          }));
          
          setSelectedInstanceResources({
            ...selectedInstanceResources,
            resources: updatedResources
          });
        }
        
        // Refresh table data
        onReload?.();
      }
    } catch (error) {
      console.error('Failed to delete connection:', error);
      alert('Failed to delete connection');
    }
  };

  const handleOpenResources = (instance: EnrichedTenantInstance) => {
    setSelectedInstanceResources({
      name: instance.display_name || instance.instance_name,
      resources: instance.linkedResources.details,
    });
    setResourcesDialogOpen(true);
  };

  return (
    <>
      <Card>
        {/* Header */}
        <Box
          sx={{
            px: 3,
            py: 2,
            borderBottom: '1px solid',
            borderColor: 'divider',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 'bold', mb: 0.5 }}>
              Associated Instances
            </Typography>
            <Typography variant="caption" color="textSecondary">
              ({instances.length} {instances.length === 1 ? 'Active' : 'Active'})
            </Typography>
          </Box>
          <Stack direction="row" spacing={1}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<FilterIcon />}
            >
              Filter
            </Button>
            <Button
              variant="contained"
              size="small"
              startIcon={<AddIcon />}
              onClick={onAddInstance}
            >
              Add Instance
            </Button>
          </Stack>
        </Box>

        {/* Table */}
        <TableContainer>
          <Table>
            <TableHead sx={{ backgroundColor: '#fafafa' }}>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold', fontSize: '0.75rem', textTransform: 'uppercase' }}>
                  <TableSortLabel
                    active={sortBy === 'name'}
                    direction={sortBy === 'name' ? sortOrder : 'asc'}
                    onClick={() => handleSort('name')}
                  >
                    Instance Name
                  </TableSortLabel>
                </TableCell>
                {/* Removed Product Column */}
                <TableCell sx={{ fontWeight: 'bold', fontSize: '0.75rem', textTransform: 'uppercase' }}>
                  <TableSortLabel
                    active={sortBy === 'environment'}
                    direction={sortBy === 'environment' ? sortOrder : 'asc'}
                    onClick={() => handleSort('environment')}
                  >
                    Environment
                  </TableSortLabel>
                </TableCell>
                <TableCell sx={{ fontWeight: 'bold', fontSize: '0.75rem', textTransform: 'uppercase' }}>
                  Linked Resources
                </TableCell>
                <TableCell sx={{ fontWeight: 'bold', fontSize: '0.75rem', textTransform: 'uppercase' }}>
                  <TableSortLabel
                    active={sortBy === 'status'}
                    direction={sortBy === 'status' ? sortOrder : 'asc'}
                    onClick={() => handleSort('status')}
                  >
                    Status
                  </TableSortLabel>
                </TableCell>
                <TableCell align="right" sx={{ fontWeight: 'bold', fontSize: '0.75rem', textTransform: 'uppercase' }}>
                  Actions
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {sortedInstances.length > 0 ? (
                sortedInstances.map((instance) => (
                  <TableRow
                    key={instance.id}
                    hover
                    sx={{
                      '&:hover .actions': {
                        opacity: 1,
                      },
                    }}
                  >
                    <TableCell>
                      <Box>
                        <Typography
                          variant="subtitle2"
                          sx={{
                            fontWeight: 500,
                            cursor: 'pointer',
                            color: 'primary.main',
                            '&:hover': { textDecoration: 'underline' },
                          }}
                        >
                          {instance.display_name || instance.instance_name || 'Unnamed'}
                        </Typography>
                        <Typography
                          variant="caption"
                          color="textSecondary"
                          sx={{ display: 'block', mt: 0.5, fontFamily: 'monospace' }}
                        >
                          {instance.id}
                        </Typography>
                      </Box>
                    </TableCell>
                    {/* Removed Product Cell */}
                    <TableCell>
                      <Chip
                        label={(instance as any).environment || 'Production'}
                        size="small"
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell>
                      <Box 
                        sx={{ display: 'flex', alignItems: 'center', gap: 1, cursor: 'pointer' }}
                        onClick={() => handleOpenResources(instance)}
                      >
                         <LinkIcon fontSize="small" color="action" />
                         <Typography variant="body2" sx={{ textDecoration: 'underline', color: 'primary.main' }}>
                           {instance.linkedResources.productCount} Products, {instance.linkedResources.connectionCount} Connections
                         </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Box
                          sx={{
                            width: 10,
                            height: 10,
                            borderRadius: '50%',
                            backgroundColor: instance.is_active ? '#4caf50' : '#bdbdbd',
                          }}
                        />
                        <Typography variant="body2">
                          {instance.is_active ? 'Active' : 'Inactive'}
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell align="right">
                      <Stack
                        direction="row"
                        spacing={0.5}
                        justifyContent="flex-end"
                        className="actions"
                        sx={{ opacity: { xs: 1, md: 0 }, transition: 'opacity 0.2s' }}
                      >
                        <IconButton
                          size="small"
                          onClick={() => onEditInstance?.(instance)}
                          title="Edit"
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                        <IconButton
                          size="small"
                          color="error"
                          onClick={() => handleDeleteClick(instance)}
                          title="Delete"
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                    <Typography color="textSecondary">
                      No instances found
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Footer */}
        <Box
          sx={{
            px: 3,
            py: 2,
            borderTop: '1px solid',
            borderColor: 'divider',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            backgroundColor: '#fafafa',
          }}
        >
          <Typography variant="caption" color="textSecondary">
            Showing {Math.min(instances.length, 10)} of {instances.length} instances
          </Typography>
          <Stack direction="row" spacing={1}>
            <Button size="small" variant="outlined" disabled>
              Previous
            </Button>
            <Button size="small" variant="outlined">
              Next
            </Button>
          </Stack>
        </Box>
      </Card>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)}>
        <DialogTitle>Delete Instance</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete "{deleteConfirmInstance?.display_name || deleteConfirmInstance?.instance_name}"?
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteConfirmOpen(false)}>Cancel</Button>
          <Button
            onClick={handleConfirmDelete}
            color="error"
            variant="contained"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
      
      {/* Resources Dialog */}
      {selectedInstanceResources && (
        <InstanceResourcesDialog
          open={resourcesDialogOpen}
          onClose={() => setResourcesDialogOpen(false)}
          instanceName={selectedInstanceResources.name}
          resources={selectedInstanceResources.resources}
          onDeleteConnection={handleDeleteConnection}
        />
      )}
    </>
  );
};

export default InstancesTable;
