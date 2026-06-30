/**
 * User Role Assignment - Material Design 3 Edition
 * 
 * Features Material Design:
 * - Material elevation and shadows
 * - Material typography scale
 * - Material icons and buttons with ripple effects
 * - Proper 8px grid spacing
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  IconButton,
  InputAdornment,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Paper,
  Tab,
  Tabs,
  TextField,
  Tooltip,
  Typography,
  AppBar,
  Toolbar,
  Divider,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material';
import {
  Search as SearchIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  Person as PersonIcon,
  Security as SecurityIcon,
  Group as GroupIcon,
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

export const UserRoleAssignmentStyled: React.FC<UserRoleAssignmentProps> = ({
  tenant,
  datasource,
}) => {
  const [users, _setUsers] = useState<User[]>([
    {
      id: '1',
      email: 'sophia.clark@example.com',
      full_name: 'Sophia Clark',
      department: 'Engineering',
      title: 'Senior Engineer',
      is_active: true,
    },
    {
      id: '2',
      email: 'liam.walker@example.com',
      full_name: 'Liam Walker',
      department: 'Sales',
      title: 'Sales Manager',
      is_active: true,
    },
    {
      id: '3',
      email: 'olivia.davis@example.com',
      full_name: 'Olivia Davis',
      department: 'Marketing',
      title: 'Marketing Lead',
      is_active: true,
    },
    {
      id: '4',
      email: 'noah.rodriguez@example.com',
      full_name: 'Noah Rodriguez',
      department: 'Product',
      title: 'Product Manager',
      is_active: true,
    },
    {
      id: '5',
      email: 'emma.wilson@example.com',
      full_name: 'Emma Wilson',
      department: 'Operations',
      title: 'Operations Manager',
      is_active: true,
    },
  ]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [userRoles, setUserRoles] = useState<UserRole[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [_loading, _setLoading] = useState(false);
  const [_saving, setSaving] = useState(false);
  const [isAssigning, setIsAssigning] = useState(false);

  // Assignment form state
  const [assignmentForm, setAssignmentForm] = useState({
    role_id: '',
    scope_type: 'global' as 'global' | 'process' | 'step' | 'team',
    scope_id: '',
    expires_at: '',
  });

  // Fetch roles
  const fetchRoles = async () => {
    try {
      const response = await fetch(
        `/api/rbac/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}&datasource_id=${datasource.id}`
      );
      const data = await response.json();
      setRoles(data || []);
    } catch (error) {
      console.error('Failed to fetch roles:', error);
    }
  };

  // Fetch user roles
  const fetchUserRoles = async (userId: string) => {
    try {
      const response = await fetch(
        `/api/rbac/users/${userId}/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}&datasource_id=${datasource.id}`
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
          datasource_id: datasource.id,
          scope_type: assignmentForm.scope_type,
          scope_id: assignmentForm.scope_id || null,
          expires_at: assignmentForm.expires_at || null,
        }),
      });

      await fetchUserRoles(selectedUser.id);
      resetAssignmentForm();
      setIsAssigning(false);
    } catch (error) {
      console.error('Failed to assign role:', error);
    } finally {
      setSaving(false);
    }
  };

  // Unassign role from user
  const unassignRole = async (roleId: string) => {
    if (!selectedUser) return;
    if (!confirm('Are you sure you want to remove this role assignment?')) return;

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
      return matchesSearch;
    });
  }, [users, searchTerm]);

  useEffect(() => {
    fetchRoles();
  }, [tenant.id]);

  useEffect(() => {
    if (selectedUser) {
      fetchUserRoles(selectedUser.id);
    }
  }, [selectedUser]);

  return (
    <Box sx={{ width: '100%', minHeight: '100vh', bgcolor: 'background.default' }}>
      {/* Material AppBar Header */}
      <AppBar position="sticky" color="default" elevation={1} sx={{ bgcolor: 'background.paper' }}>
        <Toolbar sx={{ px: 3, py: 1 }}>
          <Box display="flex" alignItems="center" gap={1.5} flexGrow={1}>
            <GroupIcon sx={{ fontSize: 28, color: 'primary.main' }} />
            <Typography variant="h6" component="div" fontWeight={600}>
              User Roles
            </Typography>
          </Box>
          <Box display="flex" alignItems="center" gap={2}>
            <TextField
              size="small"
              placeholder="Search"
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon sx={{ fontSize: 18 }} />
                  </InputAdornment>
                ),
              }}
              sx={{ width: 240 }}
            />
          </Box>
        </Toolbar>
      </AppBar>

      {/* Main Content - Two Panel Layout */}
      <Box display="flex" height="calc(100vh - 64px)">
        {/* Left Sidebar - User List */}
        <Paper 
          elevation={2} 
          sx={{ 
            width: 320, 
            borderRadius: 0,
            overflow: 'auto',
            borderRight: 1,
            borderColor: 'divider'
          }}
        >
          <Box sx={{ p: 2 }}>
            <TextField
              fullWidth
              size="small"
              placeholder="Search users"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon sx={{ fontSize: 18 }} />
                  </InputAdornment>
                ),
              }}
              sx={{ mb: 2 }}
            />

            {/* Tabs */}
            <Tabs value={0} sx={{ mb: 2, minHeight: 40 }}>
              <Tab label="All" sx={{ minHeight: 40, textTransform: 'none', fontSize: '0.875rem' }} />
              <Tab label="Active" sx={{ minHeight: 40, textTransform: 'none', fontSize: '0.875rem' }} />
              <Tab label="Inactive" sx={{ minHeight: 40, textTransform: 'none', fontSize: '0.875rem' }} />
            </Tabs>

            {/* Users List */}
            <List sx={{ p: 0 }}>
              {filteredUsers.map((user) => (
                <ListItem 
                  key={user.id} 
                  disablePadding
                  sx={{ mb: 0.5 }}
                >
                  <ListItemButton
                    selected={selectedUser?.id === user.id}
                    onClick={() => setSelectedUser(user)}
                    sx={{
                      borderRadius: 1,
                      '&.Mui-selected': {
                        bgcolor: 'primary.lighter',
                        borderLeft: 4,
                        borderColor: 'primary.main',
                      },
                    }}
                  >
                    <ListItemText
                      primary={
                        <Typography variant="body2" fontWeight={500}>
                          {user.full_name}
                        </Typography>
                      }
                      secondary={
                        <Typography variant="caption" color="text.secondary">
                          {user.email} {user.department && `• ${user.department}`}
                        </Typography>
                      }
                    />
                  </ListItemButton>
                </ListItem>
              ))}
            </List>
          </Box>
        </Paper>

        {/* Right Panel - Role Details */}
        <Box sx={{ flex: 1, overflow: 'auto', p: 4 }}>
          {selectedUser ? (
            <Box sx={{ maxWidth: 800 }}>
              <Box display="flex" alignItems="center" justifyContent="space-between" mb={4}>
                <Box>
                  <Typography variant="h4" fontWeight={600} gutterBottom>
                    {selectedUser.full_name}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {selectedUser.email}
                  </Typography>
                </Box>
                <Button
                  variant="contained"
                  startIcon={<AddIcon />}
                  onClick={() => setIsAssigning(true)}
                  sx={{ textTransform: 'none', fontWeight: 500 }}
                >
                  Assign Role
                </Button>
              </Box>

              <Divider sx={{ mb: 3 }} />

              <Typography variant="h6" fontWeight={600} mb={2}>
                Current Roles
              </Typography>

              {/* Current Roles */}
              {userRoles.length > 0 ? (
                <Box display="flex" flexDirection="column" gap={2}>
                  {userRoles.map((role) => (
                    <Card key={role.id} elevation={1}>
                      <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                        <Box display="flex" alignItems="center" justifyContent="space-between">
                          <Box display="flex" alignItems="center" gap={2}>
                            <SecurityIcon color="primary" />
                            <Box>
                              <Typography variant="body1" fontWeight={500}>
                                {role.role_name}
                              </Typography>
                              <Chip 
                                label={`${role.scope_type} scope`} 
                                size="small" 
                                variant="outlined"
                                sx={{ mt: 0.5 }}
                              />
                            </Box>
                          </Box>
                          <Tooltip title="Remove role">
                            <IconButton
                              color="error"
                              onClick={() => unassignRole(role.role_id)}
                              size="small"
                            >
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </Box>
                      </CardContent>
                    </Card>
                  ))}
                </Box>
              ) : (
                <Paper elevation={0} sx={{ p: 4, textAlign: 'center', bgcolor: 'grey.50' }}>
                  <SecurityIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 2 }} />
                  <Typography variant="body1" color="text.secondary">
                    No roles assigned to this user
                  </Typography>
                </Paper>
              )}
            </Box>
          ) : (
            <Box 
              display="flex" 
              flexDirection="column" 
              alignItems="center" 
              justifyContent="center"
              height="100%"
              textAlign="center"
            >
              <PersonIcon sx={{ fontSize: 80, color: 'text.disabled', mb: 2 }} />
              <Typography variant="h6" fontWeight={600} gutterBottom>
                No User Selected
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Select a user from the left panel to view and manage their roles.
              </Typography>
            </Box>
          )}
        </Box>
      </Box>

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
            Assign Role to {selectedUser?.full_name}
          </Typography>
        </DialogTitle>
        <DialogContent>
          <Box mt={2} display="flex" flexDirection="column" gap={2}>
            <FormControl fullWidth>
              <InputLabel id="role-select-label">Select Role</InputLabel>
              <Select
                labelId="role-select-label"
                value={assignmentForm.role_id}
                label="Select Role"
                onChange={(e) => setAssignmentForm({ ...assignmentForm, role_id: e.target.value as string })}
              >
                {roles.map((r) => (
                  <MenuItem key={r.id} value={r.id}>
                    {r.role_name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <FormControl fullWidth>
              <InputLabel id="scope-select-label">Scope Type</InputLabel>
              <Select
                labelId="scope-select-label"
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
              />
            )}

            <TextField
              fullWidth
              label="Expires At"
              type="date"
              InputLabelProps={{ shrink: true }}
              value={assignmentForm.expires_at}
              onChange={(e) => setAssignmentForm({ ...assignmentForm, expires_at: e.target.value })}
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
            disabled={!assignmentForm.role_id}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Assign
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
