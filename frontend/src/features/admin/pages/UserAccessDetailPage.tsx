import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Button,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Autocomplete,
  TextField,
  CircularProgress,
  Chip,
  AppBar,
  Toolbar,
  Grid,
  Card,
  CardContent,
  Avatar,
  Stack,
} from '@mui/material';
import {
  ArrowBack as ArrowBackIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  Person as PersonIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  AccessTime as AccessTimeIcon,
} from '@mui/icons-material';
import { apiClient } from '../../../utils/apiClient';
import { useTenant } from '../../../contexts/TenantContext';

interface AccessMapping {
  mapping_id: string;
  tenant_id: string;
  tenant_name: string;
  tenant_instance_id: string | null;
  tenant_instance_name: string;
  role_key: string;
  role_name: string;
  assigned_at: string;
  assigned_by: string;
}

interface User {
  id: string;
  username: string;
  email: string;
  name: string;
  first_name?: string;
  last_name?: string;
  status?: string;
  is_active?: boolean;
  last_login?: string;
  last_login_at?: string;
  created_at?: string;
}

interface RoleOption {
  role_key: string;
  role_name: string;
  role_level: string;
}

export const UserAccessDetailPage: React.FC = () => {
  const { userId } = useParams<{ userId: string }>();
  const navigate = useNavigate();
  const { tenant, datasource } = useTenant();

  const [user, setUser] = useState<User | null>(null);
  const [accessList, setAccessList] = useState<AccessMapping[]>([]);
  const [roles, setRoles] = useState<RoleOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Dialog States
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState<RoleOption | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Edit User State
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [editForm, setEditForm] = useState({
    name: '',
    username: '',
    email: '',
  });

  useEffect(() => {
    if (userId) {
      fetchData();
    }
  }, [userId]);

  const fetchData = async () => {
    if (!userId) return;

    try {
      setLoading(true);
      setError(null);

      // Fetch user info and access list in parallel
      const [userRes, accessRes] = await Promise.all([
        apiClient<User>(
          `/rbac/users/${userId}`,
          {},
          { tenantId: tenant?.id || '' }
        ),
        apiClient<AccessMapping[]>(
          `/rbac/users/${userId}/access`,
          {},
          { tenantId: tenant?.id || '' }
        ),
      ]);

      setUser(userRes);
      setEditForm({
        name: userRes.name || '',
        username: userRes.username || '',
        email: userRes.email || '',
      });
      setAccessList(Array.isArray(accessRes) ? accessRes : []);
    } catch (err) {
      console.error('Failed to fetch user access:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch user access');
    } finally {
      setLoading(false);
    }
  };

  const fetchRoles = async () => {
    try {
      const data = await apiClient<RoleOption[]>(
        '/rbac/roles',
        {},
        { tenantId: tenant?.id || '' }
      );
      setRoles(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error('Failed to fetch roles:', err);
      setRoles([]);
    }
  };

  const handleOpenAddDialog = async () => {
    await fetchRoles();
    setSelectedRole(null);
    setIsAddDialogOpen(true);
  };

  const handleAddAccess = async () => {
    if (!userId || !selectedRole || !tenant?.id) return;

    setSubmitting(true);
    try {
      await apiClient(
        `/rbac/users/${userId}/access`,
        {
          method: 'POST',
          body: JSON.stringify({
            tenant_id: tenant.id,
            tenant_instance_id: datasource?.id || null,
            role_key: selectedRole.role_key,
          }),
        },
        { tenantId: tenant.id }
      );
      setIsAddDialogOpen(false);
      fetchData();
    } catch (err) {
      console.error('Failed to add access:', err);
      alert('Failed to add access');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteAccess = async (mappingId: string) => {
    if (!userId) return;
    if (!confirm('Are you sure you want to revoke this access?')) return;

    try {
      await apiClient(
        `/rbac/users/${userId}/access/${mappingId}`,
        { method: 'DELETE' },
        { tenantId: tenant?.id || '' }
      );
      fetchData();
    } catch (err) {
      console.error('Failed to delete access:', err);
      alert('Failed to delete access');
    }
  };

  const handleSaveUser = async () => {
    if (!userId) return;

    try {
      setSubmitting(true);
      await apiClient(
        `/rbac/users/${userId}`,
        {
          method: 'PUT',
          body: JSON.stringify(editForm),
        },
        { tenantId: tenant?.id || '' }
      );
      setIsEditDialogOpen(false);
      fetchData();
    } catch (err) {
      console.error('Failed to update user:', err);
      alert('Failed to update user details');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="100vh">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" minHeight="100vh" gap={2}>
        <Typography color="error" variant="h6">{error}</Typography>
        <Button variant="contained" onClick={() => navigate('/admin/rbac/users')}>
          Back to Users
        </Button>
      </Box>
    );
  }

  const isInactive = user?.is_active === false || user?.status === 'inactive';
  const lastLoginText = (user?.last_login || user?.last_login_at)
    ? new Date(user.last_login || user.last_login_at!).toLocaleString()
    : 'Never logged in';

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'grey.100' }}>
      <AppBar position="static" elevation={1}>
        <Toolbar>
          <IconButton edge="start" color="inherit" onClick={() => navigate('/admin/rbac/users')}>
            <ArrowBackIcon />
          </IconButton>
          <Typography variant="h6" sx={{ ml: 2, flexGrow: 1 }}>
            User Management & Access Control
          </Typography>
        </Toolbar>
      </AppBar>

      <Box sx={{ p: 4 }}>
        {/* User Summary Card */}
        <Paper elevation={2} sx={{ p: 3, mb: 4, borderRadius: 2 }}>
          <Box display="flex" justifyContent="space-between" alignItems="flex-start">
            <Box display="flex" alignItems="center" gap={2}>
              <Avatar sx={{ width: 56, height: 56, bgcolor: isInactive ? 'grey.500' : 'primary.main', fontSize: '1.5rem' }}>
                {(user?.name || user?.username || 'U').charAt(0).toUpperCase()}
              </Avatar>
              <Box>
                <Box display="flex" alignItems="center" gap={1.5}>
                  <Typography
                    variant="h5"
                    fontWeight={600}
                    sx={{ textDecoration: isInactive ? 'line-through' : 'none', color: isInactive ? 'text.secondary' : 'text.primary' }}
                  >
                    {user?.name || `${user?.first_name || ''} ${user?.last_name || ''}`.trim() || user?.username}
                  </Typography>
                  <Chip
                    icon={!isInactive ? <CheckCircleIcon /> : <CancelIcon />}
                    label={!isInactive ? 'Active' : 'Inactive'}
                    size="small"
                    color={!isInactive ? 'success' : 'default'}
                    variant="outlined"
                  />
                </Box>
                <Typography
                  variant="body2"
                  color="text.secondary"
                  sx={{ textDecoration: isInactive ? 'line-through' : 'none' }}
                >
                  {user?.email} • Username: {user?.username}
                </Typography>
                <Stack direction="row" spacing={1} alignItems="center" mt={1}>
                  <AccessTimeIcon fontSize="small" sx={{ color: 'text.secondary', fontSize: 16 }} />
                  <Typography variant="caption" color="text.secondary" fontWeight={500}>
                    Last Login: {lastLoginText}
                  </Typography>
                </Stack>
              </Box>
            </Box>

            <Button
              variant="outlined"
              startIcon={<EditIcon />}
              onClick={() => setIsEditDialogOpen(true)}
              sx={{ textTransform: 'none', fontWeight: 600 }}
            >
              Edit User Details
            </Button>
          </Box>
        </Paper>

        {/* Access Mappings Table Header */}
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
          <Typography variant="h5" fontWeight={600}>
            Access Mappings ({accessList.length})
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleOpenAddDialog}
            sx={{ textTransform: 'none', fontWeight: 600 }}
          >
            Add Access Mapping
          </Button>
        </Box>

        <Paper elevation={2} sx={{ borderRadius: 2 }}>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow sx={{ bgcolor: 'black' }}>
                  <TableCell><Typography variant="subtitle2" fontWeight={600} color="white">Tenant</Typography></TableCell>
                  <TableCell><Typography variant="subtitle2" fontWeight={600} color="white">Instance</Typography></TableCell>
                  <TableCell><Typography variant="subtitle2" fontWeight={600} color="white">Role</Typography></TableCell>
                  <TableCell><Typography variant="subtitle2" fontWeight={600} color="white">Assigned By</Typography></TableCell>
                  <TableCell><Typography variant="subtitle2" fontWeight={600} color="white">Assigned At</Typography></TableCell>
                  <TableCell><Typography variant="subtitle2" fontWeight={600} color="white">Actions</Typography></TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {accessList.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                      <Typography color="text.secondary">No access mappings found</Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  accessList.map((access) => (
                    <TableRow key={access.mapping_id} hover>
                      <TableCell>
                        <Typography variant="body2" fontWeight={500}>
                          {access.tenant_name}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">
                          {access.tenant_instance_name}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip label={access.role_name} size="small" variant="outlined" />
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">
                          {access.assigned_by}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">
                          {new Date(access.assigned_at).toLocaleDateString()}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <IconButton
                          size="small"
                          color="error"
                          onClick={() => handleDeleteAccess(access.mapping_id)}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      </Box>

      {/* Add Access Mapping Dialog */}
      <Dialog open={isAddDialogOpen} onClose={() => setIsAddDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add Access Mapping</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
            <TextField
              label="Scoped Tenant"
              value={tenant?.name || tenant?.id || 'Master Tenant'}
              disabled
              fullWidth
              helperText="Tenant is automatically defaulted from your scoped session."
            />
            <TextField
              label="Scoped Instance"
              value={datasource?.source_name || datasource?.id || 'Default Instance'}
              disabled
              fullWidth
              helperText="Instance is automatically defaulted from your active datasource session."
            />
            <Autocomplete
              options={roles}
              getOptionLabel={(option) => `${option.role_name} (${option.role_level})`}
              value={selectedRole}
              onChange={(_, value) => setSelectedRole(value)}
              renderInput={(params) => (
                <TextField {...params} label="Select Role" placeholder="Choose a role to assign..." />
              )}
              fullWidth
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setIsAddDialogOpen(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleAddAccess}
            disabled={!selectedRole || submitting}
            sx={{ textTransform: 'none', fontWeight: 600 }}
          >
            {submitting ? 'Adding...' : 'Add Access'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit User Info Dialog */}
      <Dialog open={isEditDialogOpen} onClose={() => setIsEditDialogOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>Edit User Details</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
            <TextField
              label="Full Name"
              value={editForm.name}
              onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
              fullWidth
            />
            <TextField
              label="Username"
              value={editForm.username}
              onChange={(e) => setEditForm({ ...editForm, username: e.target.value })}
              fullWidth
            />
            <TextField
              label="Email"
              value={editForm.email}
              onChange={(e) => setEditForm({ ...editForm, email: e.target.value })}
              fullWidth
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setIsEditDialogOpen(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSaveUser}
            disabled={submitting}
            sx={{ textTransform: 'none', fontWeight: 600 }}
          >
            {submitting ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
