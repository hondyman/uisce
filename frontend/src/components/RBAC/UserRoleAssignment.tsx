/**
 * User Role Assignment - Material-UI Implementation
 * World-class design matching reference images
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Container,
  Typography,
  TextField,
  Button,
  Paper,
  AppBar,
  Toolbar,
  InputAdornment,
  Avatar,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Tabs,
  Tab,
  Card,
  CardContent,
  Chip,
  IconButton,
  Alert,
} from '@mui/material';
import {
  Search as SearchIcon,
  Delete as DeleteIcon,
  AccessTime as ClockIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface User {
  id: string;
  email: string;
  full_name: string;
  department?: string;
  title?: string;
  is_active: boolean;
}

interface Role {
  id: string;
  role_key: string;
  role_name: string;
  role_level: string;
}

interface UserRole {
  id: string;
  user_id: string;
  role_id: string;
  role_key: string;
  role_name: string;
  role_level: string;
  scope_type: 'global' | 'process' | 'step' | 'team';
  scope_id?: string;
  assigned_at: string;
  expires_at?: string;
}

interface UserRoleAssignmentProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const UserRoleAssignment: React.FC<UserRoleAssignmentProps> = ({
  tenant,
  datasource,
}) => {
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [userRoles, setUserRoles] = useState<UserRole[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(true);
  const [showAssignModal, setShowAssignModal] = useState(false);
  const [saving, setSaving] = useState(false);

  // Assignment form state
  const [assignmentForm, setAssignmentForm] = useState({
    role_id: '',
    scope_type: 'global' as 'global' | 'process' | 'step' | 'team',
    scope_id: '',
    expires_at: '',
  });

  // Fetch users
  const fetchUsers = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `/api/users?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      setUsers(data || [
        {
          id: '1',
          email: 'sophia.clark@example.com',
          full_name: 'Sophia Clark',
          department: 'Engineering',
          is_active: true,
        },
        {
          id: '2',
          email: 'liam.walker@example.com',
          full_name: 'Liam Walker',
          department: 'Sales',
          is_active: true,
        },
        {
          id: '3',
          email: 'olivia.davis@example.com',
          full_name: 'Olivia Davis',
          department: 'Marketing',
          is_active: true,
        },
        {
          id: '4',
          email: 'noah.rodriguez@example.com',
          full_name: 'Noah Rodriguez',
          department: 'Product',
          is_active: true,
        },
        {
          id: '5',
          email: 'emma.wilson@example.com',
          full_name: 'Emma Wilson',
          department: 'Operations',
          is_active: true,
        },
      ]);
    } catch (error) {
      console.error('Failed to fetch users:', error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch roles
  const fetchRoles = async () => {
    try {
      const response = await fetch(
        `/api/rbac/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      setRoles(data || [
        { id: '1', role_key: 'editor', role_name: 'Editor', role_level: 'editor' },
        { id: '2', role_key: 'viewer', role_name: 'Viewer', role_level: 'viewer' },
      ]);
    } catch (error) {
      console.error('Failed to fetch roles:', error);
    }
  };

  // Fetch user roles
  const fetchUserRoles = async (userId: string) => {
    try {
      const response = await fetch(
        `/api/rbac/users/${userId}/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      setUserRoles(data || []);
    } catch (error) {
      console.error('Failed to fetch user roles:', error);
    }
  };

  // Assign role to user
  const assignRole = async () => {
    if (!selectedUser || !assignmentForm.role_id) return;

    try {
      setSaving(true);
      await fetch(`/api/rbac/roles/${assignmentForm.role_id}/assign`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: selectedUser.id,
          tenant_id: tenant.id,
          tenant_instance_id: datasource.id,
          scope_type: assignmentForm.scope_type,
          scope_id: assignmentForm.scope_id || null,
          expires_at: assignmentForm.expires_at || null,
        }),
      });

      await fetchUserRoles(selectedUser.id);
      setShowAssignModal(false);
      resetAssignmentForm();
    } catch (error) {
      console.error('Failed to assign role:', error);
    } finally {
      setSaving(false);
    }
  };

  // Unassign role from user
  const unassignRole = async (roleId: string) => {
    if (!selectedUser) return;
    if (!confirm('Are you sure you want to remove the \'Editor\' role from \'' + selectedUser.full_name + '\'?')) return;

    try {
      await fetch(`/api/rbac/roles/${roleId}/unassign/${selectedUser.id}`, {
        method: 'DELETE',
      });
      await fetchUserRoles(selectedUser.id);
    } catch (error) {
      console.error('Failed to unassign role:', error);
    }
  };

  // Reset assignment form
  const resetAssignmentForm = () => {
    setAssignmentForm({
      role_id: '',
      scope_type: 'global',
      scope_id: '',
      expires_at: '',
    });
  };

  // Filter users
  const filteredUsers = useMemo(() => {
    return users.filter(user => {
      const matchesSearch =
        user.full_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        user.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (user.department?.toLowerCase().includes(searchTerm.toLowerCase()) ?? false);
      const matchesActive = 
        activeTab === 0 || 
        (activeTab === 1 && user.is_active) ||
        (activeTab === 2 && !user.is_active);
      return matchesSearch && matchesActive;
    });
  }, [users, searchTerm, activeTab]);

  useEffect(() => {
    fetchUsers();
    fetchRoles();
  }, [tenant.id]);

  useEffect(() => {
    if (selectedUser) {
      fetchUserRoles(selectedUser.id);
    }
  }, [selectedUser]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh', bgcolor: '#fafafa' }}>
        <Typography>Loading users...</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: '#fafafa' }}>
      {/* Header */}
      <AppBar position="static" color="default" elevation={0} sx={{ bgcolor: 'white', borderBottom: '1px solid #e0e0e0' }}>
        <Toolbar>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mr: 4 }}>
            <Box sx={{ width: 24, height: 24, bgcolor: 'black', borderRadius: 0.5 }} />
            <Typography variant="h6" sx={{ fontWeight: 600 }}>
              Role Manager
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', gap: 3, flexGrow: 1 }}>
            <Button color="inherit" sx={{ textTransform: 'none' }}>Dashboard</Button>
            <Button color="inherit" sx={{ textTransform: 'none' }}>Roles</Button>
            <Button color="inherit" sx={{ textTransform: 'none', fontWeight: 600 }}>Users</Button>
            <Button color="inherit" sx={{ textTransform: 'none' }}>Permissions</Button>
          </Box>
          <TextField
            size="small"
            placeholder="Search"
            sx={{ width: 250, mr: 2 }}
            InputProps={{
              endAdornment: (
                <InputAdornment position="end">
                  <SearchIcon sx={{ fontSize: 14 }} />
                </InputAdornment>
              ),
            }}
          />
          <Avatar sx={{ width: 40, height: 40, bgcolor: '#bdbdbd' }} />
        </Toolbar>
      </AppBar>

      {/* Main Content */}
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Typography variant="h4" sx={{ fontWeight: 700, mb: 3 }}>
          User Roles
        </Typography>

        {/* Search */}
        <Paper sx={{ p: 2, mb: 3, boxShadow: 'none', border: '1px solid #e0e0e0' }}>
          <TextField
            fullWidth
            size="small"
            placeholder="Search users"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon sx={{ fontSize: 14 }} />
                </InputAdornment>
              ),
            }}
          />
        </Paper>

        {/* Tabs */}
        <Tabs
          value={activeTab}
          onChange={(_, newValue) => setActiveTab(newValue)}
          sx={{ mb: 3, borderBottom: '1px solid #e0e0e0' }}
        >
          <Tab label="All" sx={{ textTransform: 'none' }} />
          <Tab label="Active" sx={{ textTransform: 'none' }} />
          <Tab label="Inactive" sx={{ textTransform: 'none' }} />
        </Tabs>

        {/* Content Grid */}
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1.5fr' }, gap: 3 }}>
          {/* Users List */}
          <Paper sx={{ p: 3, boxShadow: 'none', border: '1px solid #e0e0e0' }}>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              {filteredUsers.map(user => (
                <Card
                  key={user.id}
                  onClick={() => setSelectedUser(user)}
                  sx={{
                    cursor: 'pointer',
                    border: selectedUser?.id === user.id ? '2px solid #1976d2' : '1px solid #e0e0e0',
                    bgcolor: selectedUser?.id === user.id ? '#e3f2fd' : 'white',
                    boxShadow: 'none',
                    '&:hover': { borderColor: '#90caf9' },
                  }}
                >
                  <CardContent>
                    <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                      {user.full_name}
                    </Typography>
                    <Typography variant="body2" color="primary" sx={{ mb: 0.5 }}>
                      {user.email}
                    </Typography>
                    {user.department && (
                      <Typography variant="caption" color="text.secondary">
                        {user.department}
                      </Typography>
                    )}
                  </CardContent>
                </Card>
              ))}
            </Box>
          </Paper>

          {/* User Details */}
          <Paper sx={{ p: 3, boxShadow: 'none', border: '1px solid #e0e0e0' }}>
            {selectedUser ? (
              <Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
                  <Box>
                    <Typography variant="h6" sx={{ fontWeight: 600 }}>
                      {selectedUser.full_name}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {selectedUser.email}
                    </Typography>
                  </Box>
                  <Button
                    variant="contained"
                    onClick={() => {
                      resetAssignmentForm();
                      setShowAssignModal(true);
                    }}
                    sx={{
                      bgcolor: '#f5f5f5',
                      color: 'text.primary',
                      textTransform: 'none',
                      boxShadow: 'none',
                      '&:hover': { bgcolor: '#eeeeee', boxShadow: 'none' },
                    }}
                  >
                    Assign Role
                  </Button>
                </Box>

                {userRoles.length > 0 ? (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                    {userRoles.map(userRole => {
                      const isExpired = userRole.expires_at && new Date(userRole.expires_at) < new Date();
                      
                      return (
                        <Card
                          key={userRole.id}
                          sx={{
                            border: isExpired ? '1px solid #f44336' : '1px solid #e0e0e0',
                            bgcolor: isExpired ? '#ffebee' : 'white',
                            boxShadow: 'none',
                          }}
                        >
                          <CardContent>
                            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1 }}>
                              <Box>
                                <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                                  {userRole.role_name}
                                </Typography>
                                <Typography variant="caption" color="text.secondary">
                                  {userRole.role_key}
                                </Typography>
                              </Box>
                              <IconButton
                                size="small"
                                onClick={() => unassignRole(userRole.role_id)}
                                color="primary"
                              >
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </Box>
                            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                              <Chip label={userRole.scope_type} size="small" />
                              {userRole.expires_at && (
                                <Chip
                                  icon={<ClockIcon fontSize="small" />}
                                  label={isExpired ? 'EXPIRED' : new Date(userRole.expires_at).toLocaleDateString()}
                                  size="small"
                                  color={isExpired ? 'error' : 'primary'}
                                  variant="outlined"
                                />
                              )}
                            </Box>
                            {isExpired && (
                              <Alert severity="error" sx={{ mt: 2 }} icon={<WarningIcon />}>
                                This role assignment has expired
                              </Alert>
                            )}
                          </CardContent>
                        </Card>
                      );
                    })}
                  </Box>
                ) : (
                  <Box sx={{ textAlign: 'center', py: 6 }}>
                    <Box
                      sx={{
                        width: 120,
                        height: 120,
                        mx: 'auto',
                        mb: 2,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        bgcolor: '#f5f5f5',
                        borderRadius: 2,
                      }}
                    >
                      <svg width="64" height="64" viewBox="0 0 64 64" fill="none">
                        <path d="M32 8C24.8 8 19 13.8 19 21C19 28.2 24.8 34 32 34C39.2 34 45 28.2 45 21C45 13.8 39.2 8 32 8ZM32 56C19 56 8 50 8 42V38C8 33 19 28 32 28C45 28 56 33 56 38V42C56 50 45 56 32 56Z" fill="#bdbdbd"/>
                      </svg>
                    </Box>
                    <Typography variant="h6" sx={{ fontWeight: 600, mb: 1 }}>
                      No User Selected
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      Select a user from the left panel to view and manage their roles.
                    </Typography>
                  </Box>
                )}
              </Box>
            ) : (
              <Box sx={{ textAlign: 'center', py: 6 }}>
                <Box
                  sx={{
                    width: 120,
                    height: 120,
                    mx: 'auto',
                    mb: 2,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    bgcolor: '#f5f5f5',
                    borderRadius: 2,
                  }}
                >
                  <svg width="64" height="64" viewBox="0 0 64 64" fill="none">
                    <path d="M32 8C24.8 8 19 13.8 19 21C19 28.2 24.8 34 32 34C39.2 34 45 28.2 45 21C45 13.8 39.2 8 32 8ZM32 56C19 56 8 50 8 42V38C8 33 19 28 32 28C45 28 56 33 56 38V42C56 50 45 56 32 56Z" fill="#bdbdbd"/>
                  </svg>
                </Box>
                <Typography variant="h6" sx={{ fontWeight: 600, mb: 1 }}>
                  No User Selected
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Select a user from the left panel to view and manage their roles.
                </Typography>
              </Box>
            )}
          </Paper>
        </Box>
      </Container>

      {/* Assignment Dialog */}
      <Dialog open={showAssignModal} onClose={() => setShowAssignModal(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 700, fontSize: '1.5rem' }}>Assign Role</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5, pt: 2 }}>
            <FormControl fullWidth>
              <InputLabel>Role</InputLabel>
              <Select
                value={assignmentForm.role_id}
                label="Role"
                onChange={(e) => setAssignmentForm({ ...assignmentForm, role_id: e.target.value })}
              >
                <MenuItem value="">Select a role</MenuItem>
                {roles.map(role => (
                  <MenuItem key={role.id} value={role.id}>
                    {role.role_name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControl fullWidth>
              <InputLabel>Scope Type</InputLabel>
              <Select
                value={assignmentForm.scope_type}
                label="Scope Type"
                onChange={(e) => setAssignmentForm({ ...assignmentForm, scope_type: e.target.value as any })}
              >
                <MenuItem value="global">Global</MenuItem>
                <MenuItem value="process">Process</MenuItem>
                <MenuItem value="step">Step</MenuItem>
                <MenuItem value="team">Team</MenuItem>
              </Select>
            </FormControl>
            {assignmentForm.scope_type !== 'global' && (
              <TextField
                fullWidth
                label="Scope ID"
                value={assignmentForm.scope_id}
                onChange={(e) => setAssignmentForm({ ...assignmentForm, scope_id: e.target.value })}
                placeholder={`Enter ${assignmentForm.scope_type} ID`}
              />
            )}
            <TextField
              fullWidth
              label="Expiration Date"
              type="datetime-local"
              value={assignmentForm.expires_at}
              onChange={(e) => setAssignmentForm({ ...assignmentForm, expires_at: e.target.value })}
              InputLabelProps={{ shrink: true }}
              helperText="Leave empty for permanent assignment"
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3 }}>
          <Button onClick={() => setShowAssignModal(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            onClick={assignRole}
            variant="contained"
            disabled={!assignmentForm.role_id || saving}
            sx={{ textTransform: 'none' }}
          >
            {saving ? 'Assigning...' : 'Assign'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
