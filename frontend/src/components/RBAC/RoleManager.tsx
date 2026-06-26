/**
 * Role Manager - Material-UI Implementation
 * World-class design matching reference images
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Container,
  Typography,
  TextField,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  AppBar,
  Toolbar,
  InputAdornment,
  IconButton,
  Avatar,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Checkbox,
  FormControlLabel,
  FormGroup,
  Chip,
  Grid,
  Card,
  CardContent,
  CardActions,
  ToggleButton,
  ToggleButtonGroup,
  Autocomplete,
  Tabs,
  Tab,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemAvatar,
} from '@mui/material';
import {
  Search as SearchIcon,
  Visibility as EyeIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  ExpandMore as ExpandMoreIcon,
  ViewList as ViewListIcon,
  ViewModule as ViewModuleIcon,
} from '@mui/icons-material';

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

export const RoleManager: React.FC<RoleManagerProps> = ({ tenant, datasource }) => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [rolePermissions, setRolePermissions] = useState<Set<string>>(new Set());
  const [isCreating, setIsCreating] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterLevel, setFilterLevel] = useState<string>('all');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [viewMode, setViewMode] = useState<'grid' | 'table'>('table');
  
  // Role Details Dialog State
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [activeTab, setActiveTab] = useState(0);
  const [roleUsers, setRoleUsers] = useState<any[]>([]);
  const [roleFieldPerms, setRoleFieldPerms] = useState<any[]>([]);

  // Form state for role editing

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
      setRoles(data || [
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
        },
      ]);
    } catch (error) {
      console.error('Failed to fetch roles:', error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch permissions
  const fetchPermissions = async () => {
    try {
      const response = await fetch(`/api/rbac/permissions?tenant_id=${tenant.id}`);
      const data = await response.json();
      setPermissions(data || [
        { id: '1', permission_key: 'view_dashboard', permission_name: 'View Dashboard', description: '', resource_type: 'dashboard', action: 'view', is_system: true },
        { id: '2', permission_key: 'manage_users', permission_name: 'Manage Users', description: '', resource_type: 'users', action: 'manage', is_system: true },
        { id: '3', permission_key: 'edit_settings', permission_name: 'Edit Settings', description: '', resource_type: 'settings', action: 'edit', is_system: true },
        { id: '4', permission_key: 'create_reports', permission_name: 'Create Reports', description: '', resource_type: 'reports', action: 'create', is_system: true },
        { id: '5', permission_key: 'export_data', permission_name: 'Export Data', description: '', resource_type: 'data', action: 'export', is_system: true },
      ]);
    } catch (error) {
      console.error('Failed to fetch permissions:', error);
    }
  };

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

  // Delete role
  const deleteRole = async (roleId: string) => {
    if (!confirm('Are you sure you want to delete this role?')) return;

    try {
      await fetch(`/api/rbac/roles/${roleId}`, { method: 'DELETE' });
      await fetchRoles();
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
    setIsCreating(false);
  };

  // Toggle permission
  const togglePermission = (permId: string) => {
    const newPerms = new Set(rolePermissions);
    if (newPerms.has(permId)) {
      newPerms.delete(permId);
    } else {
      newPerms.add(permId);
    }
    setRolePermissions(newPerms);
  };

  // View role details
  const handleViewRole = (role: Role) => {
    setSelectedRole(role);
    setRoleUsers([]); // Placeholder - would fetch real users
    setRoleFieldPerms([]); // Placeholder - would fetch real permissions
    setDetailsOpen(true);
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
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh', bgcolor: '#fafafa' }}>
        <Typography>Loading roles...</Typography>
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
            <Button color="inherit" sx={{ textTransform: 'none', fontWeight: 600 }}>Roles</Button>
            <Button color="inherit" sx={{ textTransform: 'none' }}>Users</Button>
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
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Roles
          </Typography>
          <Button
            variant="contained"
            onClick={() => setIsCreating(true)}
            sx={{
              bgcolor: '#f5f5f5',
              color: 'text.primary',
              textTransform: 'none',
              boxShadow: 'none',
              '&:hover': { bgcolor: '#eeeeee', boxShadow: 'none' },
            }}
          >
            Create Role
          </Button>
        </Box>

        {/* Search and Filter */}
        <Paper sx={{ p: 2, mb: 3, boxShadow: 'none', border: '1px solid #e0e0e0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Box display="flex" gap={2} flexGrow={1}>
            <Autocomplete
                freeSolo
                options={roles.map(r => r.role_name)}
                value={searchTerm}
                onInputChange={(_, newValue) => setSearchTerm(newValue || '')}
                renderInput={(params) => (
                    <TextField
                        {...params}
                        size="small"
                        placeholder="Search roles"
                        InputProps={{
                        ...params.InputProps,
                        startAdornment: (
                            <InputAdornment position="start">
                            <SearchIcon sx={{ fontSize: 14 }} />
                            </InputAdornment>
                        ),
                        }}
                    />
                )}
                sx={{ width: 300 }}
            />
            <Button
                variant="outlined"
                size="small"
                endIcon={<ExpandMoreIcon />}
                sx={{ textTransform: 'none' }}
            >
                Level
            </Button>
          </Box>
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
        </Paper>

        {/* Roles Content */}
        {viewMode === 'table' ? (
            <TableContainer component={Paper} sx={{ boxShadow: 'none', border: '1px solid #e0e0e0' }}>
            <Table>
                <TableHead sx={{ bgcolor: '#fafafa' }}>
                <TableRow>
                    <TableCell sx={{ fontWeight: 600 }}>Role Name</TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>Level</TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>Description</TableCell>
                    <TableCell sx={{ fontWeight: 600 }}>Actions</TableCell>
                </TableRow>
                </TableHead>
                <TableBody>
                {filteredRoles.map((role) => (
                    <TableRow key={role.id} hover>
                    <TableCell sx={{ fontWeight: 500 }}>{role.role_name}</TableCell>
                    <TableCell>
                        <Chip label={role.role_level} size="small" color="primary" variant="outlined" />
                    </TableCell>
                    <TableCell sx={{ color: 'text.secondary' }}>{role.description}</TableCell>
                    <TableCell>
                        <Box sx={{ display: 'flex', gap: 1 }}>
                        <Button
                            size="small"
                            startIcon={<EyeIcon fontSize="small" />}
                            sx={{ textTransform: 'none', minWidth: 'auto' }}
                            onClick={() => handleViewRole(role)}
                        >
                            Eye
                        </Button>
                        <Button
                            size="small"
                            startIcon={<EditIcon fontSize="small" />}
                            sx={{ textTransform: 'none', minWidth: 'auto' }}
                        >
                            Pencil
                        </Button>
                        <Button
                            size="small"
                            startIcon={<DeleteIcon fontSize="small" />}
                            onClick={() => deleteRole(role.id)}
                            sx={{ textTransform: 'none', minWidth: 'auto' }}
                        >
                            Trash
                        </Button>
                        </Box>
                    </TableCell>
                    </TableRow>
                ))}
                </TableBody>
            </Table>
            </TableContainer>
        ) : (
            <Grid container spacing={3}>
                {filteredRoles.map((role) => (
                    <Grid item xs={12} sm={6} md={4} key={role.id}>
                         <Card elevation={0} sx={{ border: '1px solid #e0e0e0', borderRadius: 2, height: '100%', display: 'flex', flexDirection: 'column' }}>
                            <CardContent sx={{ flexGrow: 1 }}>
                                <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={2}>
                                    <Typography variant="h6" fontWeight={600}>
                                        {role.role_name}
                                    </Typography>
                                    <Chip label={role.role_level} size="small" color="primary" variant="outlined" />
                                </Box>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2, minHeight: 40 }}>
                                    {role.description || 'No description provided.'}
                                </Typography>
                                 <Typography variant="caption" color="text.secondary" display="block">
                                    Key: {role.role_key}
                                </Typography>
                            </CardContent>
                            <CardActions sx={{ px: 2, pb: 2, pt: 0, justifyContent: 'flex-end' }}>
                                 <Button
                                    size="small"
                                    startIcon={<EyeIcon />}
                                    sx={{ textTransform: 'none' }}
                                    onClick={() => handleViewRole(role)}
                                >
                                    Details
                                </Button>
                                <Button
                                    size="small"
                                    startIcon={<EditIcon />}
                                    sx={{ textTransform: 'none' }}
                                >
                                    Edit
                                </Button>
                                <Button
                                    size="small"
                                    startIcon={<DeleteIcon />}
                                    onClick={() => deleteRole(role.id)}
                                    color="error"
                                    sx={{ textTransform: 'none' }}
                                >
                                    Delete
                                </Button>
                            </CardActions>
                         </Card>
                    </Grid>
                ))}
            </Grid>
        )}
      </Container>

      {/* Create Role Dialog */}
      <Dialog open={isCreating} onClose={resetForm} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 700, fontSize: '1.5rem' }}>Create Role</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5, pt: 2 }}>
            <TextField
              fullWidth
              label="Role Key"
              placeholder="Enter role key"
              value={formData.role_key}
              onChange={(e) => setFormData({ ...formData, role_key: e.target.value })}
            />
            <TextField
              fullWidth
              label="Role Name"
              placeholder="Enter role name"
              value={formData.role_name}
              onChange={(e) => setFormData({ ...formData, role_name: e.target.value })}
            />
            <TextField
              fullWidth
              label="Description"
              placeholder="Enter role description"
              multiline
              rows={3}
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            />
            <FormControl fullWidth>
              <InputLabel>Role Level</InputLabel>
              <Select
                value={formData.role_level}
                label="Role Level"
                onChange={(e) => setFormData({ ...formData, role_level: e.target.value as any })}
              >
                <MenuItem value="viewer">Viewer</MenuItem>
                <MenuItem value="editor">Editor</MenuItem>
                <MenuItem value="approver">Approver</MenuItem>
                <MenuItem value="admin">Admin</MenuItem>
              </Select>
            </FormControl>
            <Box>
              <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
                Permissions
              </Typography>
              <FormGroup>
                {permissions.map((perm) => (
                  <FormControlLabel
                    key={perm.id}
                    control={
                      <Checkbox
                        checked={rolePermissions.has(perm.id)}
                        onChange={() => togglePermission(perm.id)}
                      />
                    }
                    label={perm.permission_name}
                  />
                ))}
              </FormGroup>
            </Box>
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3 }}>
          <Button onClick={resetForm} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            onClick={createRole}
            variant="contained"
            disabled={saving}
            sx={{ textTransform: 'none' }}
          >
            {saving ? 'Saving...' : 'Save'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Role Details Dialog */}
      <Dialog open={detailsOpen} onClose={() => setDetailsOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 2, pb: 0 }}>
            <Box>
                <Typography variant="h6" fontWeight={700}>{selectedRole?.role_name || 'Role Details'}</Typography>
                <Typography variant="caption" color="text.secondary">{selectedRole?.role_key}</Typography>
            </Box>
            {selectedRole && <Chip label={selectedRole.role_level} size="small" color="primary" variant="outlined" sx={{ ml: 'auto' }} />}
        </DialogTitle>
        <DialogContent sx={{ p: 0 }}>
            <Box sx={{ borderBottom: 1, borderColor: 'divider', px: 3 }}>
                <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
                    <Tab label="Overview" />
                    <Tab label={`Users (${roleUsers.length})`} />
                    <Tab label={`Field Permissions (${roleFieldPerms.length})`} />
                </Tabs>
            </Box>
            
            {/* Overview Tab */}
            {activeTab === 0 && (
                <Box p={3}>
                    <Typography variant="subtitle2" gutterBottom>Description</Typography>
                    <Typography variant="body2" color="text.secondary" paragraph>
                        {selectedRole?.description || 'No description provided.'}
                    </Typography>
                    <Grid container spacing={2} mt={1}>
                        <Grid item xs={6}>
                            <Paper variant="outlined" sx={{ p: 2, textAlign: 'center' }}>
                                <Typography variant="h4" color="primary">{roleUsers.length}</Typography>
                                <Typography variant="caption" color="text.secondary">Assigned Users</Typography>
                            </Paper>
                        </Grid>
                        <Grid item xs={6}>
                            <Paper variant="outlined" sx={{ p: 2, textAlign: 'center' }}>
                                <Typography variant="h4" color="secondary">{roleFieldPerms.length}</Typography>
                                <Typography variant="caption" color="text.secondary">Field Policies</Typography>
                            </Paper>
                        </Grid>
                    </Grid>
                </Box>
            )}

            {/* Users Tab */}
            {activeTab === 1 && (
                <List sx={{ p: 0 }}>
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
        <DialogActions>
            <Button onClick={() => setDetailsOpen(false)}>Close</Button>
            <Button variant="contained" onClick={() => { setDetailsOpen(false); setIsCreating(true); /* TODO Edit logic */ }}>Edit Role</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
