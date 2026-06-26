/**
 * Role Manager - Enterprise Role & Permission Configuration
 * 
 * Professional RBAC interface featuring Material Design 3:
 * - Material elevation and shadows
 * - Material typography scale
 * - Material icons and buttons
 * - Proper 8px grid spacing
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  InputAdornment,
  MenuItem,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
  AppBar,
  Toolbar,
} from '@mui/material';
import {
  Search as SearchIcon,
  Add as AddIcon,
  Visibility as ViewIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Security as SecurityIcon,
  FilterList as FilterIcon,
  PersonAdd as PersonAddIcon,
} from '@mui/icons-material';

import {
  Avatar,
  Divider,
  Grid,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  Tab,
  Tabs,
} from '@mui/material';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface Role {
  id: string;
  role_key: string;
  role_name: string;
  description: string;
  role_type: 'system' | 'custom';
  role_level: 'viewer' | 'editor' | 'approver' | 'admin' | 'super_admin';
  is_active: boolean;
  created_at: string;
  updated_at: string;
  permission_count?: number;
  user_count?: number;
}

interface Permission {
  id: string;
  permission_key: string;
  permission_name: string;
  description: string;
  resource_type: string;
  action: string;
  is_system: boolean;
}

interface RoleManagerProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const RoleManagerStyled: React.FC<RoleManagerProps> = ({ tenant, datasource }) => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [_permissions, setPermissions] = useState<Permission[]>([]);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [rolePermissions, setRolePermissions] = useState<Set<string>>(new Set());
  const [_isEditing, setIsEditing] = useState(false);
  const [_isCreating, setIsCreating] = useState(false);
  const [isViewing, setIsViewing] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterLevel, _setFilterLevel] = useState<string>('all');
  const [loading, setLoading] = useState(true);
  const [_saving, setSaving] = useState(false);
  const [_expandedGroups, _setExpandedGroups] = useState<Set<string>>(new Set(['process', 'step']));

  // Role Details State
  const [activeTab, setActiveTab] = useState(0);
  const [roleUsers, setRoleUsers] = useState<any[]>([]);
  const [roleFieldPerms, setRoleFieldPerms] = useState<any[]>([]);

  // Form state for role editing
  const [formData, setFormData] = useState({
    role_key: '',
    role_name: '',
    description: '',
    role_level: 'viewer' as Role['role_level'],
  });

  // Fetch roles
  const fetchRoles = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `/api/rbac/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      
      // Ensure we always set an array
      const rolesArray = Array.isArray(data) ? data : (data?.roles || data?.data || []);
      
      setRoles(rolesArray.length > 0 ? rolesArray : [
        {
          id: '1',
          role_key: 'administrator',
          role_name: 'Administrator',
          description: 'Full access to all features and data.',
          role_type: 'system',
          role_level: 'admin',
          is_active: true,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          permission_count: 45,
          user_count: 5,
        },
        {
          id: '2',
          role_key: 'manager',
          role_name: 'Manager',
          description: 'Manage users, roles, and basic settings.',
          role_type: 'system',
          role_level: 'editor',
          is_active: true,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          permission_count: 28,
          user_count: 12,
        },
        {
          id: '3',
          role_key: 'editor',
          role_name: 'Editor',
          description: 'Create and edit content within assigned teams.',
          role_type: 'system',
          role_level: 'editor',
          is_active: true,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          permission_count: 15,
          user_count: 34,
        },
        {
          id: '4',
          role_key: 'viewer',
          role_name: 'Viewer',
          description: 'View project details and progress.',
          role_type: 'system',
          role_level: 'viewer',
          is_active: true,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          permission_count: 8,
          user_count: 89,
        },
        {
          id: '5',
          role_key: 'guest',
          role_name: 'Guest',
          description: 'Limited access to public information.',
          role_type: 'system',
          role_level: 'viewer',
          is_active: true,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          permission_count: 3,
          user_count: 156,
        },
      ]);
    } catch (error) {
      console.error('Failed to fetch roles:', error);
      setRoles([]); // Set empty array on error
    } finally {
      setLoading(false);
    }
  };

  // Fetch permissions
  const fetchPermissions = async () => {
    try {
      const response = await fetch(`/api/rbac/permissions?tenant_id=${tenant.id}`);
      const data = await response.json();
      setPermissions(data || []);
    } catch (error) {
      console.error('Failed to fetch permissions:', error);
    }
  };

  // Fetch role permissions
  const _fetchRolePermissions = async (roleId: string) => {
    try {
      const response = await fetch(
        `/api/rbac/roles/${roleId}/permissions?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      const permIds = new Set(data.map((p: Permission) => p.id) as string[]);
      setRolePermissions(permIds);
    } catch (error) {
      console.error('Failed to fetch role permissions:', error);
    }
  };

  // Fetch role details
  const handleViewRole = async (role: Role) => {
    setSelectedRole(role);
    setFormData({
      role_key: role.role_key,
      role_name: role.role_name,
      description: role.description,
      role_level: role.role_level,
    });
    setActiveTab(0);
    setIsViewing(true);

    // Fetch Users
    try {
      const usersRes = await fetch(`/api/rbac/roles/${role.id}/users?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`);
      if (usersRes.ok) {
        const usersData = await usersRes.json();
        setRoleUsers(usersData || []);
      }
    } catch (e) {
      console.error("Failed to fetch role users", e);
    }

    // Fetch Field Permissions
    try {
      const permsRes = await fetch(`/api/rbac/field-permissions?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}&role_id=${role.id}`);
      if (permsRes.ok) {
        const permsData = await permsRes.json();
        setRoleFieldPerms(permsData || []);
      }
    } catch (e) {
      console.error("Failed to fetch role field permissions", e);
    }
  };

  // Create role

  // Create role
  const createRole = async () => {
    try {
      setSaving(true);
      const response = await fetch(
        `/api/rbac/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            ...formData,
            permissions: Array.from(rolePermissions),
          }),
        }
      );
      
      if (response.ok) {
        await fetchRoles();
        resetForm();
      }
    } catch (error) {
      console.error('Failed to create role:', error);
    } finally {
      setSaving(false);
    }
  };

  // Save role handler for dialog
  const handleSaveRole = () => {
    createRole();
  };

  // Update role
  const _updateRole = async () => {
    if (!selectedRole) return;
    
    try {
      setSaving(true);
      await fetch(`/api/rbac/roles/${selectedRole.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          role_name: formData.role_name,
          description: formData.description,
          is_active: true,
        }),
      });

      await Promise.all(
        Array.from(rolePermissions).map(permId =>
          fetch(`/api/rbac/role-permissions`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              role_id: selectedRole.id,
              permission_id: permId,
            }),
          })
        )
      );

      await fetchRoles();
      resetForm();
    } catch (error) {
      console.error('Failed to update role:', error);
    } finally {
      setSaving(false);
    }
  };

  // Delete role
  const _deleteRole = async (roleId: string) => {
    if (!confirm('Are you sure you want to delete this role?')) {
      return;
    }

    try {
      await fetch(`/api/rbac/roles/${roleId}`, { method: 'DELETE' });
      await fetchRoles();
      if (selectedRole?.id === roleId) {
        resetForm();
      }
    } catch (error) {
      console.error('Failed to delete role:', error);
    }
  };

  // Reset form
  const resetForm = () => {
    setFormData({
      role_key: '',
      role_name: '',
      description: '',
      role_level: 'viewer',
    });
    setRolePermissions(new Set());
    setSelectedRole(null);
    setIsEditing(false);
    setIsCreating(false);
  };

  // Toggle permission
  const _togglePermission = (permId: string) => {
    const newPerms = new Set(rolePermissions);
    if (newPerms.has(permId)) {
      newPerms.delete(permId);
    } else {
      newPerms.add(permId);
    }
    setRolePermissions(newPerms);
  };

  // Filter roles
  const filteredRoles = useMemo(() => {
    return roles.filter(role => {
      const matchesSearch = role.role_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                          role.role_key.toLowerCase().includes(searchTerm.toLowerCase());
      const matchesLevel = filterLevel === 'all' || role.role_level === filterLevel;
      return matchesSearch && matchesLevel;
    });
  }, [roles, searchTerm, filterLevel]);

  useEffect(() => {
    fetchRoles();
    fetchPermissions();
  }, [tenant.id]);

  if (loading) {
    return (
      <Box 
        display="flex" 
        alignItems="center" 
        justifyContent="center" 
        minHeight="100vh"
        bgcolor="background.default"
      >
        <Box display="flex" flexDirection="column" alignItems="center" gap={2}>
          <SecurityIcon sx={{ fontSize: 48, animation: 'spin 2s linear infinite', color: 'primary.main' }} />
          <Typography variant="body1" color="text.secondary">
            Loading roles...
          </Typography>
        </Box>
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100%', minHeight: '100vh', bgcolor: 'background.default' }}>
      {/* Material AppBar Header */}
      <AppBar position="sticky" color="default" elevation={1} sx={{ bgcolor: 'background.paper' }}>
        <Toolbar sx={{ px: 3, py: 1 }}>
          <Box display="flex" alignItems="center" gap={1.5} flexGrow={1}>
            <SecurityIcon sx={{ fontSize: 28, color: 'primary.main' }} />
            <Typography variant="h6" component="div" fontWeight={600}>
              Role Manager
            </Typography>
          </Box>
          <Box display="flex" alignItems="center" gap={2}>
            <TextField
              size="small"
              placeholder="Search"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
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

      {/* Main Content */}
      <Box component="main" sx={{ px: 4, py: 4 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
          <Typography variant="h4" fontWeight={600} color="text.primary">
            Roles
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => {
              resetForm();
              setIsCreating(true);
            }}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Create Role
          </Button>
        </Box>

        {/* Search and Filter Bar */}
        <Box display="flex" gap={2} mb={3}>
          <TextField
            fullWidth
            size="medium"
            placeholder="Search roles"
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon sx={{ fontSize: 20 }} />
                </InputAdornment>
              ),
            }}
          />
          <Button
            variant="outlined"
            endIcon={<FilterIcon />}
            sx={{ textTransform: 'none', fontWeight: 500, minWidth: 120 }}
          >
            Level
          </Button>
        </Box>

        {/* Roles Table with Material Elevation */}
        <Paper elevation={2} sx={{ borderRadius: 2, overflow: 'hidden' }}>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow sx={{ bgcolor: 'grey.50' }}>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight={600}>Role Name</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight={600}>Level</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight={600}>Description</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight={600}>Actions</Typography>
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredRoles.map((role) => (
                  <TableRow 
                    key={role.id}
                    hover
                    sx={{ '&:last-child td': { borderBottom: 0 } }}
                  >
                    <TableCell>
                      <Typography variant="body2" fontWeight={500}>
                        {role.role_name}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip 
                        label={role.role_level.replace(/_/g, ' ').split(' ').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')}
                        size="small"
                        color="primary"
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {role.description}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Box display="flex" gap={1}>
                        <Tooltip title="View">
                          <IconButton 
                            size="small" 
                            color="primary"
                            onClick={() => handleViewRole(role)}
                          >
                            <ViewIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Edit">
                          <IconButton 
                            size="small" 
                            color="primary"
                            onClick={() => {
                              setSelectedRole(role);
                              setFormData({
                                role_key: role.role_key,
                                role_name: role.role_name,
                                description: role.description,
                                role_level: role.role_level,
                              });
                              setIsEditing(true);
                            }}
                          >
                            <EditIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Delete">
                          <IconButton 
                            size="small" 
                            color="error"
                            onClick={() => _deleteRole(role.id)}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Box>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      </Box>

      {/* Create Role Material Dialog */}
      <Dialog 
        open={_isCreating} 
        onClose={resetForm}
        maxWidth="sm"
        fullWidth
        PaperProps={{
          elevation: 8,
          sx: { borderRadius: 2 }
        }}
      >
        <DialogTitle>
          <Typography variant="h5" fontWeight={600}>
            Create Role
          </Typography>
        </DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <Box display="flex" flexDirection="column" gap={3} mt={1}>
            <TextField
              fullWidth
              label="Role Key"
              placeholder="Enter role key"
              value={formData.role_key}
              onChange={(e) => setFormData({ ...formData, role_key: e.target.value })}
              variant="outlined"
            />

            <TextField
              fullWidth
              label="Role Name"
              placeholder="Enter role name"
              value={formData.role_name}
              onChange={(e) => setFormData({ ...formData, role_name: e.target.value })}
              variant="outlined"
            />

            <TextField
              fullWidth
              multiline
              rows={3}
              label="Description"
              placeholder="Enter role description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              variant="outlined"
            />

            <TextField
              fullWidth
              select
              label="Role Level"
              value={formData.role_level}
              onChange={(e) => setFormData({ ...formData, role_level: e.target.value as any })}
              variant="outlined"
            >
              <MenuItem value="viewer">Viewer</MenuItem>
              <MenuItem value="editor">Editor</MenuItem>
              <MenuItem value="approver">Approver</MenuItem>
              <MenuItem value="admin">Admin</MenuItem>
              <MenuItem value="super_admin">Super Admin</MenuItem>
            </TextField>
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={resetForm} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button 
            variant="contained" 
            onClick={handleSaveRole}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Role Material Dialog */}
      <Dialog 
        open={_isEditing} 
        onClose={resetForm}
        maxWidth="sm"
        fullWidth
        PaperProps={{
          elevation: 8,
          sx: { borderRadius: 2 }
        }}
      >
        <DialogTitle>
          <Typography variant="h5" fontWeight={600}>
            Edit Role
          </Typography>
        </DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <Box display="flex" flexDirection="column" gap={3} mt={1}>
            <TextField
              fullWidth
              label="Role Key"
              value={formData.role_key}
              disabled
              variant="outlined"
              helperText="Role key cannot be changed"
            />

            <TextField
              fullWidth
              label="Role Name"
              placeholder="Enter role name"
              value={formData.role_name}
              onChange={(e) => setFormData({ ...formData, role_name: e.target.value })}
              variant="outlined"
            />

            <TextField
              fullWidth
              multiline
              rows={3}
              label="Description"
              placeholder="Enter role description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              variant="outlined"
            />

            <TextField
              fullWidth
              select
              label="Role Level"
              value={formData.role_level}
              onChange={(e) => setFormData({ ...formData, role_level: e.target.value as any })}
              variant="outlined"
            >
              <MenuItem value="viewer">Viewer</MenuItem>
              <MenuItem value="editor">Editor</MenuItem>
              <MenuItem value="approver">Approver</MenuItem>
              <MenuItem value="admin">Admin</MenuItem>
              <MenuItem value="super_admin">Super Admin</MenuItem>
            </TextField>

            {selectedRole && (
              <Box display="flex" alignItems="center" justifyContent="space-between" p={2} bgcolor="grey.50" borderRadius={1} mt={2}>
                <Box>
                  <Typography variant="subtitle2" fontWeight={600}>
                    Role Status
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {selectedRole.is_active ? 'Active - Users can be assigned this role' : 'Inactive - Role is disabled'}
                  </Typography>
                </Box>
                <Button
                  variant={selectedRole.is_active ? "outlined" : "contained"}
                  color={selectedRole.is_active ? "error" : "success"}
                  onClick={async () => {
                    try {
                      await fetch(`/api/rbac/roles/${selectedRole.id}?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`, {
                        method: 'PUT',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                          role_name: selectedRole.role_name,
                          description: selectedRole.description,
                          is_active: !selectedRole.is_active,
                        }),
                      });
                      await fetchRoles();
                      setSelectedRole({ ...selectedRole, is_active: !selectedRole.is_active });
                    } catch (error) {
                      console.error('Failed to toggle role status:', error);
                    }
                  }}
                  sx={{ textTransform: 'none', fontWeight: 500 }}
                >
                  {selectedRole.is_active ? 'Deactivate' : 'Activate'}
                </Button>
              </Box>
            )}
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={resetForm} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button 
            variant="contained" 
            onClick={_updateRole}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Save Changes
          </Button>
        </DialogActions>
      </Dialog>

      {/* View Role Material Dialog (Read-Only) */}
      <Dialog 
        open={isViewing} 
        onClose={() => setIsViewing(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{
          elevation: 8,
          sx: { borderRadius: 2 }
        }}
      >
        <DialogTitle>
          <Typography variant="h5" fontWeight={600}>
            Role Details
          </Typography>
        </DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
            {/* Tabs */}
            <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
              <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
                <Tab label="Overview" />
                <Tab label={`Users (${roleUsers.length})`} />
                <Tab label={`Field Permissions (${roleFieldPerms.length})`} />
              </Tabs>
            </Box>

            {/* Overview Tab */}
            {activeTab === 0 && (
              <Box display="flex" flexDirection="column" gap={3}>
                <TextField
                  fullWidth
                  label="Role Key"
                  value={formData.role_key}
                  disabled
                  variant="outlined"
                />
                <TextField
                  fullWidth
                  label="Role Name"
                  value={formData.role_name}
                  disabled
                  variant="outlined"
                />
                <TextField
                  fullWidth
                  multiline
                  rows={3}
                  label="Description"
                  value={formData.description}
                  disabled
                  variant="outlined"
                />
                <TextField
                  fullWidth
                  label="Role Level"
                  value={formData.role_level.replace(/_/g, ' ').split(' ').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')}
                  disabled
                  variant="outlined"
                />
                {selectedRole && (
                  <>
                    <TextField
                      fullWidth
                      label="Status"
                      value={selectedRole.is_active ? 'Active' : 'Inactive'}
                      disabled
                      variant="outlined"
                    />
                     <TextField
                      fullWidth
                      label="Created At"
                      value={new Date(selectedRole.created_at).toLocaleString()}
                      disabled
                      variant="outlined"
                    />
                  </>
                )}
              </Box>
            )}

            {/* Users Tab */}
            {activeTab === 1 && (
              <List sx={{ p: 0, maxHeight: 400, overflow: 'auto' }}>
                {roleUsers.length === 0 ? (
                  <Box p={3} textAlign="center"><Typography color="text.secondary">No users assigned to this role.</Typography></Box>
                ) : (
                  roleUsers.map((user, i) => (
                    <React.Fragment key={user.id}>
                      {i > 0 && <Divider />}
                      <ListItem>
                        <ListItemAvatar>
                          <Avatar>{user.username?.charAt(0).toUpperCase()}</Avatar>
                        </ListItemAvatar>
                        <ListItemText 
                          primary={user.name || user.username} 
                          secondary={user.email} 
                        />
                        <Typography variant="caption" color="text.secondary">
                          Assigned: {new Date(user.assigned_at).toLocaleDateString()}
                        </Typography>
                      </ListItem>
                    </React.Fragment>
                  ))
                )}
              </List>
            )}

            {/* Field Permissions Tab */}
            {activeTab === 2 && (
              <TableContainer sx={{ maxHeight: 400 }}>
                <Table stickyHeader size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Field</TableCell>
                      <TableCell>Resource/Context</TableCell>
                      <TableCell>Permission</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {roleFieldPerms.length === 0 ? (
                      <TableRow><TableCell colSpan={3} align="center">No field permissions configured.</TableCell></TableRow>
                    ) : (
                      roleFieldPerms.map((fp) => (
                        <TableRow key={fp.id} hover>
                          <TableCell sx={{ fontWeight: 500 }}>{fp.field_name}</TableCell>
                          <TableCell>{fp.resource_type}</TableCell>
                          <TableCell>
                            <Chip 
                              label={fp.permission_level} 
                              size="small" 
                              color={fp.permission_level === 'write' ? 'success' : fp.permission_level === 'read' ? 'info' : 'warning'} 
                              variant="outlined"
                              sx={{ minWidth: 60, height: 20, fontSize: '0.7rem' }}
                            />
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setIsViewing(false)} sx={{ textTransform: 'none' }}>
            Close
          </Button>
          <Button 
            variant="outlined"
            startIcon={<PersonAddIcon />}
            onClick={() => {
              if (selectedRole) {
                // Open user assignment in a new component/page
                alert('User assignment feature: Navigate to /admin/rbac/assign-users or open UserRoleAssignmentDialog component');
              }
            }}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Assign Users
          </Button>
          <Button 
            variant="contained" 
            onClick={() => {
              setIsViewing(false);
              setIsEditing(true);
            }}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Edit Role
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
