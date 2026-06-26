/**
 * User Management Page
 * Manage users and their role assignments within a tenant
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  InputAdornment,
  AppBar,
  Toolbar,
  Autocomplete,
  Grid,
  Card,
  CardContent,
  CardActions,
  Avatar,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material';
import {
  Search as SearchIcon,
  PersonAdd as PersonAddIcon,
  Security as SecurityIcon,
  Edit as EditIcon,
  CheckCircle as CheckCircleIcon,
  Cancel as CancelIcon,
  Email as EmailIcon,
  ViewList as ViewListIcon,
  ViewModule as ViewModuleIcon,
} from '@mui/icons-material';
import { useTenant } from '../../../contexts/TenantContext';

interface User {
  id: string;
  username: string;
  email: string;
  name: string;
  first_name?: string;
  last_name?: string;
  status: string;
  is_active: boolean;
  created_at: string;
}

interface Role {
  id: string;
  role_key: string;
  role_name: string;
  role_level: string;
  is_active: boolean;
}

interface UserRole {
  id: string;
  role_id: string;
  role_key: string;
  role_name: string;
  role_level: string;
  assigned_at: string;
}

export const UserManagementPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [userRoles, setUserRoles] = useState<UserRole[]>([]);
  const [availableRoles, setAvailableRoles] = useState<Role[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isAssigning, setIsAssigning] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [selectedRoleId, setSelectedRoleId] = useState('');
  const [viewMode, setViewMode] = useState<'grid' | 'table'>('table');
  
  // New User Form State
  const [newUser, setNewUser] = useState({
    username: '',
    email: '',
    name: '',
    password: 'password123', // Default for now
  });

  // Fetch users
  const fetchUsers = async () => {
    if (!tenant?.id) {
      setError('No tenant context available');
      setLoading(false);
      return;
    }
    
    try {
      setLoading(true);
      setError(null);
      const response = await fetch(`/api/rbac/users?tenant_id=${tenant.id}`);
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      
      const data = await response.json();
      setUsers(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch users:', error);
      setError(error instanceof Error ? error.message : 'Failed to fetch users');
      setUsers([]);
    } finally {
      setLoading(false);
    }
  };

  // Fetch all roles
  const fetchRoles = async () => {
    if (!tenant?.id || !datasource?.id) return;
    
    try {
      const response = await fetch(
        `/api/rbac/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      setRoles(Array.isArray(data) ? data.filter((r: Role) => r.is_active) : []);
    } catch (error) {
      console.error('Failed to fetch roles:', error);
      setRoles([]);
    }
  };

  // Fetch user's assigned roles
  const fetchUserRoles = async (userId: string) => {
    if (!tenant?.id || !datasource?.id) return;

    try {
      const response = await fetch(
        `/api/rbac/users/${userId}/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      setUserRoles(Array.isArray(data) ? data : []);
      
      // Calculate available roles (not yet assigned)
      const assignedRoleIds = new Set((Array.isArray(data) ? data : []).map((ur: UserRole) => ur.role_id));
      setAvailableRoles(roles.filter(r => !assignedRoleIds.has(r.id)));
    } catch (error) {
      console.error('Failed to fetch user roles:', error);
      setUserRoles([]);
      setAvailableRoles(roles);
    }
  };

  // Assign role to user
  const assignRole = async () => {
    if (!selectedUser || !selectedRoleId || !tenant?.id || !datasource?.id) return;

    try {
      const response = await fetch(
        `/api/rbac/roles/${selectedRoleId}/assign?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            user_id: selectedUser.id,
            scope_type: 'global',
          }),
        }
      );

      if (response.ok) {
        await fetchUserRoles(selectedUser.id);
        setIsAssigning(false);
        setSelectedRoleId('');
      }
    } catch (error) {
      console.error('Failed to assign role:', error);
    }
  };

  // Unassign role from user
  const unassignRole = async (roleId: string) => {
    if (!selectedUser || !tenant?.id || !datasource?.id) return;

    if (!confirm('Are you sure you want to remove this role from the user?')) {
      return;
    }

    try {
      await fetch(
        `/api/rbac/roles/${roleId}/unassign/${selectedUser.id}?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`,
        { method: 'DELETE' }
      );
      await fetchUserRoles(selectedUser.id);
    } catch (error) {
      console.error('Failed to unassign role:', error);
    }
  };

  // Create new user
  const createUser = async () => {
    if (!tenant?.id) return;

    try {
      const response = await fetch(`/api/rbac/users?tenant_id=${tenant.id}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...newUser,
          tenant_id: tenant.id,
          first_name: newUser.name.split(' ')[0],
          last_name: newUser.name.split(' ').slice(1).join(' '),
        }),
      });

      if (response.ok) {
        setIsCreating(false);
        setNewUser({ username: '', email: '', name: '', password: 'password123' });
        fetchUsers();
      } else {
        const err = await response.text();
        alert(`Failed to create user: ${err}`);
      }
    } catch (error) {
      console.error('Failed to create user:', error);
      alert('Failed to create user');
    }
  };

  useEffect(() => {
    if (tenant?.id && datasource?.id) {
      fetchUsers();
      fetchRoles();
    }
  }, [tenant?.id, datasource?.id]);

  const filteredUsers = users.filter(user =>
    user.username?.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.email?.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.name?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (loading) {
    return (
      <Box display="flex" alignItems="center" justifyContent="center" minHeight="100vh">
        <Typography>Loading users...</Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" minHeight="100vh" gap={2}>
        <Typography color="error" variant="h6">Error Loading Users</Typography>
        <Typography color="text.secondary">{error}</Typography>
        <Button variant="contained" onClick={() => {
          setError(null);
          fetchUsers();
        }}>Retry</Button>
      </Box>
    );
  }

  if (!tenant || !datasource) {
    return (
      <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" minHeight="100vh" gap={2}>
        <SecurityIcon sx={{ fontSize: 64, color: 'text.disabled' }} />
        <Typography variant="h6" color="text.secondary">No Tenant Context</Typography>
        <Typography color="text.secondary">Please select a tenant to manage users</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100%', minHeight: '100vh', bgcolor: 'background.default' }}>
      {/* Header */}
      <AppBar position="sticky" color="default" elevation={1} sx={{ bgcolor: 'background.paper' }}>
        <Toolbar sx={{ px: 3, py: 1 }}>
          <Box display="flex" alignItems="center" gap={1.5} flexGrow={1}>
            <SecurityIcon sx={{ fontSize: 28, color: 'primary.main' }} />
            <Typography variant="h6" component="div" fontWeight={600}>
              User Management
            </Typography>
          </Box>
          <Box display="flex" alignItems="center" gap={2}>
            <Autocomplete
              freeSolo
              options={users.map(u => u.name || u.username)}
              value={searchTerm}
              onInputChange={(_, newValue) => setSearchTerm(newValue || '')}
              renderInput={(params) => (
                <TextField
                  {...params}
                  size="small"
                  placeholder="Search users..."
                  InputProps={{
                    ...params.InputProps,
                    startAdornment: (
                      <InputAdornment position="start">
                        <SearchIcon sx={{ fontSize: 18 }} />
                      </InputAdornment>
                    ),
                  }}
                />
              )}
              sx={{ width: 300 }}
            />
          </Box>
        </Toolbar>
      </AppBar>

      {/* Main Content */}
      <Box component="main" sx={{ px: 4, py: 4 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
          <Typography variant="h4" fontWeight={600} color="text.primary">
            Users ({filteredUsers.length})
          </Typography>
          <Box display="flex" gap={2}>
             <ToggleButtonGroup
              value={viewMode}
              exclusive
              onChange={(_, value) => value && setViewMode(value)}
              size="small"
            >
              <ToggleButton value="table" aria-label="table view">
                <ViewListIcon fontSize="small" />
              </ToggleButton>
              <ToggleButton value="grid" aria-label="grid view">
                <ViewModuleIcon fontSize="small" />
              </ToggleButton>
            </ToggleButtonGroup>
            <Button
                variant="contained"
                startIcon={<PersonAddIcon />}
                onClick={() => setIsCreating(true)}
                sx={{ textTransform: 'none', fontWeight: 600 }}
            >
                Add User
            </Button>
          </Box>
        </Box>

        {/* Users Content */}
        {viewMode === 'table' ? (
            <Paper elevation={2} sx={{ borderRadius: 2, overflow: 'hidden' }}>
            <TableContainer>
                <Table>
                <TableHead>
                    <TableRow sx={{ bgcolor: 'grey.50' }}>
                    <TableCell>
                        <Typography variant="subtitle2" fontWeight={600}>User</Typography>
                    </TableCell>
                    <TableCell>
                        <Typography variant="subtitle2" fontWeight={600}>Email</Typography>
                    </TableCell>
                    <TableCell>
                        <Typography variant="subtitle2" fontWeight={600}>Username</Typography>
                    </TableCell>
                    <TableCell>
                        <Typography variant="subtitle2" fontWeight={600}>Status</Typography>
                    </TableCell>
                    <TableCell>
                        <Typography variant="subtitle2" fontWeight={600}>Actions</Typography>
                    </TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {filteredUsers.length === 0 ? (
                    <TableRow>
                        <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                        <Typography color="text.secondary">
                            No users found
                        </Typography>
                        </TableCell>
                    </TableRow>
                    ) : (
                    filteredUsers.map((user) => (
                        <TableRow key={user.id} hover>
                        <TableCell>
                            <Box display="flex" alignItems="center" gap={1.5}>
                                <Avatar sx={{ width: 32, height: 32, bgcolor: 'primary.main', fontSize: '0.875rem' }}>
                                    {(user.name || user.username)[0].toUpperCase()}
                                </Avatar>
                                <Typography variant="body2" fontWeight={500}>
                                {user.name || `${user.first_name || ''} ${user.last_name || ''}`.trim() || 'N/A'}
                                </Typography>
                            </Box>
                        </TableCell>
                        <TableCell>
                            <Typography variant="body2" color="text.secondary">
                            {user.email || 'N/A'}
                            </Typography>
                        </TableCell>
                        <TableCell>
                            <Typography variant="body2" color="text.secondary">
                            {user.username}
                            </Typography>
                        </TableCell>
                        <TableCell>
                            <Chip
                            icon={user.is_active ? <CheckCircleIcon /> : <CancelIcon />}
                            label={user.is_active ? 'Active' : 'Inactive'}
                            size="small"
                            color={user.is_active ? 'success' : 'default'}
                            variant="outlined"
                            />
                        </TableCell>
                        <TableCell>
                            <Button
                            size="small"
                            variant="outlined"
                            startIcon={<EditIcon />}
                            onClick={() => {
                                setSelectedUser(user);
                                fetchUserRoles(user.id);
                            }}
                            sx={{ textTransform: 'none', fontWeight: 500 }}
                            >
                            Manage Roles
                            </Button>
                        </TableCell>
                        </TableRow>
                    ))
                    )}
                </TableBody>
                </Table>
            </TableContainer>
            </Paper>
        ) : (
            <Grid container spacing={3}>
                {filteredUsers.map((user) => (
                    <Grid item xs={12} sm={6} md={4} key={user.id}>
                        <Card elevation={1} sx={{ borderRadius: 2, height: '100%', display: 'flex', flexDirection: 'column' }}>
                            <CardContent sx={{ flexGrow: 1 }}>
                                <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={2}>
                                     <Box display="flex" alignItems="center" gap={1.5}>
                                        <Avatar sx={{ width: 48, height: 48, bgcolor: 'primary.main' }}>
                                            {(user.name || user.username)[0].toUpperCase()}
                                        </Avatar>
                                        <Box>
                                            <Typography variant="subtitle1" fontWeight={600}>
                                                {user.name || `${user.first_name || ''} ${user.last_name || ''}`.trim() || user.username}
                                            </Typography>
                                            <Typography variant="body2" color="text.secondary">
                                                {user.username}
                                            </Typography>
                                        </Box>
                                    </Box>
                                    <Chip
                                        label={user.is_active ? 'Active' : 'Inactive'}
                                        size="small"
                                        color={user.is_active ? 'success' : 'default'}
                                        variant="outlined"
                                    />
                                </Box>
                                
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                                    <EmailIcon fontSize="small" /> {user.email || 'No email'}
                                </Typography>
                            </CardContent>
                            <CardActions sx={{ px: 2, pb: 2, pt: 0 }}>
                                <Button
                                    fullWidth
                                    variant="outlined"
                                    startIcon={<EditIcon />}
                                    onClick={() => {
                                        setSelectedUser(user);
                                        fetchUserRoles(user.id);
                                    }}
                                    sx={{ textTransform: 'none' }}
                                >
                                    Manage Roles
                                </Button>
                            </CardActions>
                        </Card>
                    </Grid>
                ))}
            </Grid>
        )}
      </Box>

      {/* User Roles Dialog */}
      <Dialog
        open={!!selectedUser}
        onClose={() => setSelectedUser(null)}
        maxWidth="md"
        fullWidth
        PaperProps={{ elevation: 8, sx: { borderRadius: 2 } }}
      >
        <DialogTitle>
          <Box display="flex" justifyContent="space-between" alignItems="center">
            <Box>
              <Typography variant="h6" fontWeight={600}>
                Manage Roles
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {selectedUser?.name || selectedUser?.username}
              </Typography>
            </Box>
            <Button
              variant="contained"
              startIcon={<PersonAddIcon />}
              onClick={() => setIsAssigning(true)}
              disabled={availableRoles.length === 0}
              sx={{ textTransform: 'none', fontWeight: 500 }}
            >
              Assign Role
            </Button>
          </Box>
        </DialogTitle>
        <DialogContent>
          <Box mt={2}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
              <Typography variant="subtitle2" fontWeight={600}>
                Assigned Roles ({userRoles.length})
              </Typography>
            </Box>
            {userRoles.length === 0 ? (
              <Box py={4} textAlign="center">
                <Typography color="text.secondary">
                  No roles assigned yet
                </Typography>
              </Box>
            ) : (
              <Box display="flex" flexDirection="column" gap={1} sx={{ maxHeight: 400, overflow: 'auto' }}>
                {userRoles.map((userRole) => (
                  <Paper key={userRole.id} elevation={1} sx={{ p: 2 }}>
                    <Box display="flex" justifyContent="space-between" alignItems="center">
                      <Box>
                        <Typography variant="body2" fontWeight={600}>
                          {userRole.role_name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          Level: {userRole.role_level} • Assigned: {new Date(userRole.assigned_at).toLocaleDateString()}
                        </Typography>
                      </Box>
                      <Button
                        size="small"
                        color="error"
                        onClick={() => unassignRole(userRole.role_id)}
                        sx={{ textTransform: 'none' }}
                      >
                        Remove
                      </Button>
                    </Box>
                  </Paper>
                ))}
              </Box>
            )}
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setSelectedUser(null)} sx={{ textTransform: 'none' }}>
            Close
          </Button>
        </DialogActions>
      </Dialog>

      {/* Assign Role Dialog */}
      <Dialog
        open={isAssigning}
        onClose={() => setIsAssigning(false)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ elevation: 8, sx: { borderRadius: 2 } }}
      >
        <DialogTitle>
          <Typography variant="h6" fontWeight={600}>
            Assign Role
          </Typography>
        </DialogTitle>
        <DialogContent>
          <Box mt={2}>
            <Autocomplete
              options={availableRoles}
              getOptionLabel={(option) => option.role_name}
              value={availableRoles.find(r => r.id === selectedRoleId) || null}
              onChange={(_, newValue) => setSelectedRoleId(newValue?.id || '')}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Select Role"
                  placeholder="Search roles..."
                />
              )}
              renderOption={(props, option) => (
                <li {...props} key={option.id}>
                  <Box>
                    <Typography variant="body2" fontWeight={500}>
                      {option.role_name}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Level: {option.role_level}
                    </Typography>
                  </Box>
                </li>
              )}
              fullWidth
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setIsAssigning(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={assignRole}
            disabled={!selectedRoleId}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Assign
          </Button>
        </DialogActions>
      </Dialog>
      {/* Create User Dialog */}
      <Dialog
        open={isCreating}
        onClose={() => setIsCreating(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ elevation: 8, sx: { borderRadius: 2 } }}
      >
        <DialogTitle>
          <Typography variant="h6" fontWeight={600}>
            Add New User
          </Typography>
        </DialogTitle>
        <DialogContent>
          <Box mt={2} display="flex" flexDirection="column" gap={2}>
            <TextField
              label="Full Name"
              fullWidth
              value={newUser.name}
              onChange={(e) => setNewUser({ ...newUser, name: e.target.value })}
            />
            <TextField
              label="Username"
              fullWidth
              value={newUser.username}
              onChange={(e) => setNewUser({ ...newUser, username: e.target.value })}
            />
            <TextField
              label="Email"
              fullWidth
              type="email"
              value={newUser.email}
              onChange={(e) => setNewUser({ ...newUser, email: e.target.value })}
            />
            <TextField
              label="Password"
              fullWidth
              type="password"
              value={newUser.password}
              onChange={(e) => setNewUser({ ...newUser, password: e.target.value })}
              helperText="Default: password123"
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setIsCreating(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={createUser}
            disabled={!newUser.username || !newUser.email || !newUser.name}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Create User
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
