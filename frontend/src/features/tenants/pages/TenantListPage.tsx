import React, { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation } from '@apollo/client';
import {
  Box,
  Button,
  Card,
  CircularProgress,
  Alert,
  Typography,
  TextField,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  InputAdornment,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TablePagination,
  Stack
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Visibility as VisibilityIcon,
  Search as SearchIcon,
  FilterList as FilterIcon,
  Sort as SortIcon,
} from '@mui/icons-material';
import { GET_TENANTS } from '../../../graphql/queries/tenantQueries';
import { DELETE_TENANT, CREATE_TENANT, UPDATE_TENANT } from '../../../graphql/mutations/tenantMutations';
import type { Tenant } from '../../../types';
import TenantDialog from '../components/TenantDialog';

export const TenantListPage: React.FC = () => {
  const navigate = useNavigate();
  const { loading, error, data, refetch } = useQuery(GET_TENANTS);
  
  // Mutations
  const [deleteTenant] = useMutation(DELETE_TENANT, { 
    onCompleted: () => refetch() 
  });
  const [createTenant] = useMutation(CREATE_TENANT, { 
    onCompleted: () => refetch() 
  });
  const [updateTenant] = useMutation(UPDATE_TENANT, { 
    onCompleted: () => refetch() 
  });

  // State management
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [tenantDialog, setTenantDialog] = useState<{ open: boolean; tenant: Tenant | null }>({
    open: false,
    tenant: null,
  });
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [deleteConfirmTenant, setDeleteConfirmTenant] = useState<Tenant | null>(null);

  const tenants: Tenant[] = data?.tenants ?? [];

  // Filter and search
  const filteredTenants = useMemo(() => {
    return tenants.filter((tenant) => {
      const searchLower = searchQuery.toLowerCase();
      const matchesSearch =
        tenant.display_name?.toLowerCase().includes(searchLower) ||
        (tenant as any).id?.toLowerCase().includes(searchLower);

      const matchesStatus =
        statusFilter === 'all' ||
        (statusFilter === 'active' && tenant.is_active) ||
        (statusFilter === 'inactive' && !tenant.is_active);

      return matchesSearch && matchesStatus;
    });
  }, [tenants, searchQuery, statusFilter]);

  // Pagination
  const paginatedTenants = useMemo(() => {
    return filteredTenants.slice(
      page * rowsPerPage,
      page * rowsPerPage + rowsPerPage
    );
  }, [filteredTenants, page, rowsPerPage]);

  const handleAddTenant = () => {
    setTenantDialog({ open: true, tenant: null });
  };

  const handleEditTenant = (tenant: Tenant) => {
    setTenantDialog({ open: true, tenant });
  };

  const handleSaveTenant = async (tenantData: Partial<Tenant>) => {
    try {
      if (tenantData.id) {
        await updateTenant({ variables: tenantData });
      } else {
        await createTenant({ variables: tenantData });
      }
      setTenantDialog({ open: false, tenant: null });
    } catch (err) {
      console.error('Error saving tenant:', err);
    }
  };

  const handleDeleteClick = (tenant: Tenant) => {
    setDeleteConfirmTenant(tenant);
    setDeleteConfirmOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (deleteConfirmTenant) {
      try {
        await deleteTenant({ variables: { id: deleteConfirmTenant.id } });
      } catch (err) {
        console.error('Error deleting tenant:', err);
      }
    }
    setDeleteConfirmOpen(false);
    setDeleteConfirmTenant(null);
  };

  const handleViewDetails = (tenantId: string) => {
    navigate(`/tenants/${tenantId}`);
  };

  const getStatusColor = (isActive: boolean) => {
    return isActive ? 'success' : 'default';
  };

  const getStatusLabel = (isActive: boolean) => {
    return isActive ? 'Active' : 'Inactive';
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '400px' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return <Alert severity="error">Failed to load tenants: {error.message}</Alert>;
  }

  return (
    <Box sx={{ p: { xs: 2, md: 3 } }}>
      {/* Header */}
      <Box sx={{ mb: 4 }}>
        <Typography variant="h4" sx={{ fontWeight: 'bold', mb: 1 }}>
          Tenants
        </Typography>
        <Typography variant="body1" color="textSecondary">
          Manage your organization's tenants, configurations, and instance hierarchy.
        </Typography>
      </Box>

      {/* Top Action Bar */}
      <Box
        sx={{
          display: 'flex',
          gap: 2,
          mb: 3,
          flexDirection: { xs: 'column', md: 'row' },
          alignItems: { xs: 'stretch', md: 'center' },
          justifyContent: 'space-between',
        }}
      >
        <Button
          variant="contained"
          color="primary"
          startIcon={<AddIcon />}
          onClick={handleAddTenant}
          sx={{ alignSelf: { xs: 'flex-start', md: 'auto' } }}
        >
          New Tenant
        </Button>
      </Box>

      {/* Filter and Search Card */}
      <Card sx={{ mb: 3, p: 2 }}>
        <Stack spacing={2}>
          <TextField
            placeholder="Filter by name, ID, or region..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            fullWidth
            variant="outlined"
            size="small"
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon color="action" />
                </InputAdornment>
              ),
            }}
          />
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<FilterIcon />}
              onClick={() => setStatusFilter(statusFilter === 'all' ? 'active' : 'all')}
            >
              Status: {statusFilter === 'all' ? 'All' : statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1)}
            </Button>
            <Button
              variant="outlined"
              size="small"
              startIcon={<FilterIcon />}
            >
              Region: All
            </Button>
            <Button
              variant="outlined"
              size="small"
              startIcon={<SortIcon />}
            >
              Sort
            </Button>
          </Box>
        </Stack>
      </Card>

      {/* Tenants Table */}
      <Card>
        <TableContainer>
          <Table>
            <TableHead sx={{ backgroundColor: '#f5f5f5' }}>
              <TableRow>
                <TableCell sx={{ fontWeight: 'bold' }}>Tenant Name / ID</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Status</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Instances</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Region</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Created</TableCell>
                <TableCell align="right" sx={{ fontWeight: 'bold' }}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {paginatedTenants.length > 0 ? (
                paginatedTenants.map((tenant) => (
                  <TableRow
                    key={tenant.id}
                    hover
                    sx={{
                      '&:hover .actions': {
                        opacity: 1,
                      },
                    }}
                  >
                    <TableCell>
                      <Box>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography
                            variant="subtitle2"
                            sx={{
                              fontWeight: 'bold',
                              cursor: 'pointer',
                              color: 'primary.main',
                              '&:hover': { textDecoration: 'underline' },
                            }}
                            onClick={() => handleViewDetails(tenant.id)}
                          >
                            {tenant.display_name || tenant.name || 'Unnamed'}
                          </Typography>
                          {tenant.gold_copy && (
                            <Chip
                              label="Gold Copy"
                              size="small"
                              color="warning"
                              sx={{ height: 20, fontSize: '0.7rem', fontWeight: 'bold' }}
                            />
                          )}
                        </Box>
                        <Typography
                          variant="caption"
                          color="textSecondary"
                          sx={{ display: 'block', mt: 0.5, fontFamily: 'monospace' }}
                        >
                          {tenant.id}
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={getStatusLabel(tenant.is_active)}
                        color={getStatusColor(tenant.is_active)}
                        size="small"
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Chip
                          label={
                            (tenant as any).tenant_instances?.length ?? 0
                          }
                          size="small"
                          variant="filled"
                        />
                        <Typography variant="body2" color="textSecondary">
                          {((tenant as any).tenant_instances?.length ?? 0) === 1
                            ? 'Instance'
                            : 'Instances'}
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {(tenant as any).region || 'N/A'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="textSecondary">
                        {(tenant as any).created_at && typeof (tenant as any).created_at === 'string'
                          ? new Date((tenant as any).created_at).toLocaleDateString()
                          : 'N/A'}
                      </Typography>
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
                          onClick={() => handleViewDetails(tenant.id)}
                          title="View Details"
                        >
                          <VisibilityIcon fontSize="small" />
                        </IconButton>
                        <IconButton
                          size="small"
                          onClick={() => handleEditTenant(tenant)}
                          title="Edit"
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                        <span title={tenant.gold_copy ? "Gold Copy tenants cannot be deleted" : "Delete tenant"}>
                          <IconButton
                            size="small"
                            color="error"
                            onClick={() => handleDeleteClick(tenant)}
                            title="Delete"
                            disabled={tenant.gold_copy}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </span>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                    <Typography color="textSecondary">
                      No tenants found
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
        <TablePagination
          component="div"
          count={filteredTenants.length}
          page={page}
          onPageChange={(_, newPage) => setPage(newPage)}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={(event) => {
            setRowsPerPage(parseInt(event.target.value, 10));
            setPage(0);
          }}
        />
      </Card>

      {/* Tenant Dialog */}
      <TenantDialog
        open={tenantDialog.open}
        tenant={tenantDialog.tenant}
        onClose={() => setTenantDialog({ open: false, tenant: null })}
        onSave={handleSaveTenant}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)}>
        <DialogTitle>Delete Tenant</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete "{deleteConfirmTenant?.display_name || deleteConfirmTenant?.name}"?
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
    </Box>
  );
};

export default TenantListPage;
