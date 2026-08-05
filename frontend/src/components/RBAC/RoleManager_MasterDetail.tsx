/**
 * RoleManager_MasterDetail - Enterprise Role & Permission Management
 * 
 * Master-Detail layout with contextual tabs:
 * - Pages & Permissions Tab: Route access control with toggles (stubbed)
 * - Users Tab: Users assigned to this role
 * - Groups/Teams Tab: AD Groups and Teams mapped to this role
 * 
 * Core (Gold Copy) roles are read-only - cannot modify permissions
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Box, Paper, Typography, Button, TextField, InputAdornment, List, ListItemButton,
  ListItemText, ListItemIcon, Chip, IconButton, Tabs, Tab, Stack, Avatar,
  Divider, useTheme, alpha, Switch, ListSubheader, Tooltip, ListItemAvatar,
  CircularProgress, MenuItem, Autocomplete
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import LockIcon from '@mui/icons-material/Lock';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import ShieldIcon from '@mui/icons-material/Shield';
import WebAssetIcon from '@mui/icons-material/WebAsset';
import GroupIcon from '@mui/icons-material/Group';
import SyncIcon from '@mui/icons-material/Sync';
import CloseIcon from '@mui/icons-material/Close';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import { apiFetch } from '../../lib/apiClient';

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
  is_template: boolean;
  is_active: boolean;
  permission_count?: number;
  user_count?: number;
  created_at: string;
}

interface AppPage {
  id: string;
  name: string;
  path: string;
  hasAccess: boolean;
  category: string;
}

interface AssignedUser {
  id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  tenant_name: string;
  assigned_at: string;
}

interface AssignedGroup {
  id: string;
  group_id: string;
  group_name: string;
  source: 'ad' | 'local';
  type: 'ad_group' | 'team';
  assigned_at: string;
}

interface RoleManagerProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const RoleManagerMasterDetail: React.FC<RoleManagerProps> = ({ tenant, datasource }) => {
  const theme = useTheme();
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [assignedUsers, setAssignedUsers] = useState<AssignedUser[]>([]);
  const [assignedGroups, setAssignedGroups] = useState<AssignedGroup[]>([]);
  const [pages, setPages] = useState<AppPage[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [tabIndex, setTabIndex] = useState(0);
  const [openCreateRoleModal, setOpenCreateRoleModal] = useState(false);
  const [openAssignUserModal, setOpenAssignUserModal] = useState(false);
  const [openMapGroupModal, setOpenMapGroupModal] = useState(false);

  const [newRoleForm, setNewRoleForm] = useState({
    role_key: '',
    role_name: '',
    description: '',
    role_level: 'viewer' as 'viewer' | 'editor' | 'approver' | 'admin' | 'super_admin',
  });

  const [assignUserForm, setAssignUserForm] = useState({
    user_id: '',
  });
  const [userSearchQuery, setUserSearchQuery] = useState('');
  const [userSearchResults, setUserSearchResults] = useState<Array<{ id: string; full_name: string; email: string; department?: string }>>([]);
  const [searchingUsers, setSearchingUsers] = useState(false);

  // Dynamic Policies state
  const [dynamicPolicies, setDynamicPolicies] = useState<Array<{
    id: string;
    role_key: string;
    resource_type: string;
    user_attribute: string;
    resource_attribute: string;
    action: string;
    description: string;
    is_active: boolean;
  }>>([]);
  const [openPolicyModal, setOpenPolicyModal] = useState(false);
  const [newPolicyForm, setNewPolicyForm] = useState({
    resource_type: '',
    user_attribute: '',
    resource_attribute: 'id',
    action: 'read',
    description: '',
  });

  // Fetch roles
  const fetchRoles = async () => {
    try {
      setLoading(true);
      const response = await apiFetch(
        `/api/rbac/roles?tenant_instance_id=${datasource.id}`,
        {
          headers: {
            'X-Tenant-ID': tenant.id,
            'X-Tenant-Datasource-ID': datasource.id,
          },
        }
      );
      const data = await response.json();
      const rolesArray = Array.isArray(data)
        ? data
        : Array.isArray(data?.roles)
        ? data.roles
        : Array.isArray(data?.data)
        ? data.data
        : [];
      setRoles(rolesArray);
      if (rolesArray.length > 0 && !selectedRole) {
        setSelectedRole(rolesArray[0]);
      }
    } catch (error) {
      console.error('Failed to fetch roles:', error);
      setRoles([]);
    } finally {
      setLoading(false);
    }
  };

  // Fetch role users
  const fetchRoleUsers = async (roleId: string) => {
    try {
      const response = await fetch(
        `/api/rbac/roles/${roleId}/users?tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      const usersArray = Array.isArray(data)
        ? data
        : Array.isArray(data?.users)
        ? data.users
        : [];
      setAssignedUsers(usersArray);
    } catch (error) {
      console.error('Failed to fetch role users:', error);
      setAssignedUsers([]);
    }
  };

  // Fetch role groups
  const fetchRoleGroups = async (roleId: string) => {
    try {
      const response = await fetch(
        `/api/rbac/roles/${roleId}/groups?tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      const groupsArray = Array.isArray(data)
        ? data
        : Array.isArray(data?.groups)
        ? data.groups
        : [];
      setAssignedGroups(groupsArray);
    } catch (error) {
      console.error('Failed to fetch role groups:', error);
      setAssignedGroups([]);
    }
  };

  // Fetch pages assigned to role
  const fetchPages = async (roleId: string) => {
    try {
      const response = await fetch(`/api/rbac/roles/${roleId}/pages`);
      if (response.ok) {
        const data: Array<{ page_id: string; page_name: string; page_path: string; category: string }> = await response.json();
        setPages(data.map(p => ({
          id: p.page_id,
          name: p.page_name,
          path: p.page_path,
          category: p.category,
          hasAccess: true,
        })));
      } else {
        setPages([]);
      }
    } catch (error) {
      console.error('Failed to fetch pages:', error);
      setPages([]);
    }
  };

  // Fetch dynamic policies for role
  const fetchPolicies = async (roleKey: string) => {
    try {
      const response = await apiFetch(
        `/api/rbac/roles/${encodeURIComponent(roleKey)}/policies`,
        {
          headers: {
            'X-Tenant-ID': tenant.id,
            'X-Tenant-Datasource-ID': datasource.id,
          },
        }
      );
      if (response.ok) {
        const data = await response.json();
        setDynamicPolicies(Array.isArray(data) ? data : []);
      } else {
        setDynamicPolicies([]);
      }
    } catch (error) {
      console.error('Failed to fetch policies:', error);
      setDynamicPolicies([]);
    }
  };

  // Create dynamic policy
  const createPolicy = async () => {
    if (!selectedRole) return;
    try {
      setSaving(true);
      const response = await apiFetch(
        `/api/rbac/roles/${encodeURIComponent(selectedRole.role_key)}/policies`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenant.id,
            'X-Tenant-Datasource-ID': datasource.id,
          },
          body: JSON.stringify(newPolicyForm),
        }
      );
      if (response.ok) {
        await fetchPolicies(selectedRole.role_key);
        setOpenPolicyModal(false);
        setNewPolicyForm({ resource_type: '', user_attribute: '', resource_attribute: 'id', action: 'read', description: '' });
      }
    } catch (error) {
      console.error('Failed to create policy:', error);
    } finally {
      setSaving(false);
    }
  };

  // Delete dynamic policy
  const deletePolicy = async (policyId: string) => {
    if (!selectedRole || !confirm('Delete this policy?')) return;
    try {
      setSaving(true);
      await apiFetch(
        `/api/rbac/roles/${encodeURIComponent(selectedRole.role_key)}/policies/${policyId}`,
        {
          method: 'DELETE',
          headers: {
            'X-Tenant-ID': tenant.id,
            'X-Tenant-Datasource-ID': datasource.id,
          },
        }
      );
      await fetchPolicies(selectedRole.role_key);
    } catch (error) {
      console.error('Failed to delete policy:', error);
    } finally {
      setSaving(false);
    }
  };

  // Create role
  const createRole = async () => {
    try {
      setSaving(true);
      const response = await apiFetch(
        `/api/rbac/roles?tenant_instance_id=${datasource.id}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenant.id,
            'X-Tenant-Datasource-ID': datasource.id,
          },
          body: JSON.stringify(newRoleForm),
        }
      );
      if (response.ok) {
        await fetchRoles();
        setOpenCreateRoleModal(false);
        setNewRoleForm({ role_key: '', role_name: '', description: '', role_level: 'viewer' });
      }
    } catch (error) {
      console.error('Failed to create role:', error);
    } finally {
      setSaving(false);
    }
  };

  // Assign user to role
  const assignUser = async () => {
    if (!selectedRole || !assignUserForm.user_id) return;
    try {
      setSaving(true);
      await apiFetch(`/api/rbac/roles/${selectedRole.id}/assign`, {
        method: 'POST',
        body: JSON.stringify({
          user_id: assignUserForm.user_id,
        }),
      });
      await fetchRoleUsers(selectedRole.id);
      setOpenAssignUserModal(false);
      setAssignUserForm({ user_id: '' });
    } catch (error) {
      console.error('Failed to assign user:', error);
    } finally {
      setSaving(false);
    }
  };

  // Search users for role assignment
  const searchUsers = async (query: string) => {
    setUserSearchQuery(query);
    if (query.length < 2) {
      setUserSearchResults([]);
      return;
    }
    try {
      setSearchingUsers(true);
      const response = await apiFetch(`/api/rbac/users/search?q=${encodeURIComponent(query)}`);
      if (response.ok) {
        const results = await response.json();
        setUserSearchResults(results);
      }
    } catch (error) {
      console.error('Failed to search users:', error);
    } finally {
      setSearchingUsers(false);
    }
  };

  // Unassign user from role
  const unassignUser = async (userId: string) => {
    if (!selectedRole || !confirm('Remove this user from the role?')) return;
    try {
      setSaving(true);
      await apiFetch(`/api/rbac/roles/${selectedRole.id}/unassign/${userId}`, {
        method: 'DELETE',
      });
      await fetchRoleUsers(selectedRole.id);
    } catch (error) {
      console.error('Failed to unassign user:', error);
    } finally {
      setSaving(false);
    }
  };

  // Toggle page access (for non-core roles)
  const togglePageAccess = async (pageId: string, hasAccess: boolean) => {
    if (!selectedRole || selectedRole.is_template) return;
    // TODO: Wire up to POST /api/rbac/roles/{roleId}/pages when backend is ready
    console.log('Toggle page access:', pageId, hasAccess);
  };

  // Group roles for the sidebar
  const coreRoles = useMemo(() => {
    return roles.filter(r => r.is_template && 
      (r.role_name?.toLowerCase().includes(searchTerm.toLowerCase()) || 
       r.role_key?.toLowerCase().includes(searchTerm.toLowerCase()))
    );
  }, [roles, searchTerm]);

  const customRoles = useMemo(() => {
    return roles.filter(r => !r.is_template && 
      (r.role_name?.toLowerCase().includes(searchTerm.toLowerCase()) || 
       r.role_key?.toLowerCase().includes(searchTerm.toLowerCase()))
    );
  }, [roles, searchTerm]);

  const getRoleLevelColor = (level: string) => {
    const colors: Record<string, 'error' | 'warning' | 'primary' | 'default'> = {
      super_admin: 'error',
      admin: 'error',
      approver: 'warning',
      editor: 'primary',
      viewer: 'default',
    };
    return colors[level] || 'default';
  };

  // Load data on mount
  useEffect(() => {
    fetchRoles();
  }, [tenant.id]);

  // Fetch role details when selection changes
  useEffect(() => {
    if (selectedRole) {
      fetchRoleUsers(selectedRole.id);
      fetchRoleGroups(selectedRole.id);
      fetchPages(selectedRole.id);
      fetchPolicies(selectedRole.role_key);
    } else {
      setAssignedUsers([]);
      setAssignedGroups([]);
      setPages([]);
      setDynamicPolicies([]);
    }
  }, [selectedRole]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 64px)', bgcolor: 'background.default' }}>
      
      {/* Left Sidebar: Role List */}
      <Paper 
        elevation={0} 
        sx={{ width: 320, borderRight: `1px solid ${theme.palette.divider}`, display: 'flex', flexDirection: 'column' }}
      >
        <Box sx={{ p: 2, borderBottom: `1px solid ${theme.palette.divider}` }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
            <Typography variant="h6" fontWeight={700}>Roles</Typography>
            <IconButton 
              size="small" 
              color="primary" 
              onClick={() => setOpenCreateRoleModal(true)}
              sx={{ bgcolor: alpha(theme.palette.primary.main, 0.1) }}
            >
              <AddIcon fontSize="small" />
            </IconButton>
          </Stack>
          
          <TextField
            fullWidth
            size="small"
            placeholder="Search roles..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" sx={{ color: 'text.secondary' }} />
                </InputAdornment>
              ),
              sx: { borderRadius: 2, bgcolor: alpha(theme.palette.text.primary, 0.04), border: 'none' }
            }}
          />
        </Box>

        <Box sx={{ flex: 1, overflowY: 'auto' }}>
          <List dense>
            {coreRoles.length > 0 && (
              <ListSubheader sx={{ bgcolor: 'transparent', fontWeight: 700, typography: 'overline', color: 'text.secondary' }}>
                Core (Gold Copy)
              </ListSubheader>
            )}
            {coreRoles.map((role) => (
              <ListItemButton
                key={role.id}
                selected={selectedRole?.id === role.id}
                onClick={() => setSelectedRole(role)}
                sx={{ 
                  py: 1.5, 
                  borderLeft: selectedRole?.id === role.id ? `3px solid ${theme.palette.primary.main}` : '3px solid transparent',
                  '&:hover': { bgcolor: alpha(theme.palette.primary.main, 0.05) }
                }}
              >
                <ListItemIcon sx={{ minWidth: 40 }}>
                  <Avatar 
                    sx={{ 
                      width: 32, 
                      height: 32, 
                      bgcolor: alpha(theme.palette.text.primary, 0.08), 
                      color: 'text.secondary' 
                    }}
                  >
                    <LockIcon fontSize="small" />
                  </Avatar>
                </ListItemIcon>
                <ListItemText 
                  primary={<Typography variant="body2" fontWeight={600}>{role.role_name}</Typography>} 
                  secondary={<Typography variant="caption" color="text.secondary">{role.role_key}</Typography>} 
                />
              </ListItemButton>
            ))}

            {customRoles.length > 0 && (
              <ListSubheader sx={{ bgcolor: 'transparent', fontWeight: 700, typography: 'overline', color: 'text.secondary', mt: 1 }}>
                Custom (Tenant)
              </ListSubheader>
            )}
            {customRoles.map((role) => (
              <ListItemButton
                key={role.id}
                selected={selectedRole?.id === role.id}
                onClick={() => setSelectedRole(role)}
                sx={{ 
                  py: 1.5, 
                  borderLeft: selectedRole?.id === role.id ? `3px solid ${theme.palette.primary.main}` : '3px solid transparent',
                  '&:hover': { bgcolor: alpha(theme.palette.primary.main, 0.05) }
                }}
              >
                <ListItemIcon sx={{ minWidth: 40 }}>
                  <Avatar 
                    sx={{ 
                      width: 32, 
                      height: 32, 
                      bgcolor: alpha(theme.palette.primary.main, 0.1), 
                      color: 'primary.main' 
                    }}
                  >
                    <ShieldIcon fontSize="small" />
                  </Avatar>
                </ListItemIcon>
                <ListItemText 
                  primary={<Typography variant="body2" fontWeight={600}>{role.role_name}</Typography>} 
                  secondary={<Typography variant="caption" color="text.secondary">{role.role_key}</Typography>} 
                />
              </ListItemButton>
            ))}
          </List>
        </Box>
      </Paper>

      {/* Right Pane: Role Detail View */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {!selectedRole ? (
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
            <Avatar sx={{ width: 80, height: 80, mb: 3, bgcolor: alpha(theme.palette.primary.main, 0.1), color: 'primary.main' }}>
              <ShieldIcon sx={{ fontSize: 40 }} />
            </Avatar>
            <Typography variant="h5" fontWeight={700} gutterBottom>No Role Selected</Typography>
            <Typography variant="body1" color="text.secondary" align="center" sx={{ maxWidth: 400 }}>
              Select a role from the left to manage its page permissions, assigned users, and groups.
            </Typography>
          </Box>
        ) : (
          <>
            {/* Header */}
            <Box sx={{ p: 3, borderBottom: `1px solid ${theme.palette.divider}` }}>
              <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                <Box>
                  <Stack direction="row" spacing={1.5} alignItems="center">
                    <Typography variant="h5" fontWeight={700}>{selectedRole.role_name}</Typography>
                    <Chip 
                      label={selectedRole.is_template ? "Core (Read-Only)" : "Custom"} 
                      size="small" 
                      color={selectedRole.is_template ? "default" : "primary"}
                      variant="outlined"
                      icon={selectedRole.is_template ? <LockIcon fontSize="small" /> : undefined}
                    />
                    <Chip 
                      label={selectedRole.role_level?.replace('_', ' ')} 
                      size="small" 
                      color={getRoleLevelColor(selectedRole.role_level)}
                      sx={{ textTransform: 'capitalize' }}
                    />
                  </Stack>
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                    {selectedRole.description}
                  </Typography>
                  <Typography variant="caption" color="text.disabled" sx={{ mt: 0.5, display: 'block' }}>
                    Key: <code>{selectedRole.role_key}</code> • {selectedRole.user_count || 0} users • {selectedRole.permission_count || 0} permissions
                  </Typography>
                </Box>
                <IconButton>
                  <MoreVertIcon />
                </IconButton>
              </Stack>
            </Box>

            {/* Tabs */}
            <Tabs
              value={tabIndex}
              onChange={(e, val) => setTabIndex(val)}
              sx={{ px: 3, borderBottom: `1px solid ${theme.palette.divider}` }}
            >
              <Tab label="Pages & Permissions" />
              <Tab label="Users" />
              <Tab label="Groups / Teams" />
              <Tab label="Dynamic Policies" />
            </Tabs>

            {/* Tab Content */}
            <Box sx={{ flex: 1, p: 3, overflowY: 'auto' }}>
              
              {/* TAB 1: Pages & Permissions */}
              {tabIndex === 0 && (
                <Box>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="subtitle1" fontWeight={600}>
                      Accessible Pages ({pages.filter(p => p.hasAccess).length}/{pages.length})
                    </Typography>
                    <Tooltip title={selectedRole.is_template ? "Core roles are read-only" : ""}>
                      <span>
                        <Button 
                          variant="contained" 
                          startIcon={<AddIcon />} 
                          disableElevation 
                          sx={{ borderRadius: 2 }}
                          disabled={selectedRole.is_template}
                          onClick={() => {/* TODO: Open add page modal */}}
                        >
                          Add Page
                        </Button>
                      </span>
                    </Tooltip>
                  </Stack>

                  {selectedRole.is_template && (
                    <Box sx={{ mb: 2, p: 1.5, bgcolor: alpha(theme.palette.warning.main, 0.1), borderRadius: 1, display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LockIcon fontSize="small" color="warning" />
                      <Typography variant="caption" color="warning.dark">
                        Core roles are read-only. Page access is managed by the system administrator.
                      </Typography>
                    </Box>
                  )}

                  {pages.length === 0 ? (
                    <Paper variant="outlined" sx={{ borderRadius: 2, p: 4, textAlign: 'center' }}>
                      <WebAssetIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                      <Typography variant="body2" color="text.secondary">
                        Page permissions are not yet configured for this role.
                      </Typography>
                    </Paper>
                  ) : (
                    <Paper variant="outlined" sx={{ borderRadius: 2 }}>
                      <List dense>
                        {pages.map((page, index) => (
                          <React.Fragment key={page.id}>
                            <ListItemButton sx={{ py: 1.5 }}>
                              <ListItemIcon sx={{ minWidth: 40 }}>
                                <WebAssetIcon fontSize="small" color="action" />
                              </ListItemIcon>
                              <ListItemText 
                                primary={
                                  <Stack direction="row" spacing={1} alignItems="center">
                                    <Typography variant="body2" fontWeight={500}>{page.name}</Typography>
                                    <Chip label={page.category} size="small" sx={{ height: 18, fontSize: 10, bgcolor: alpha(theme.palette.text.primary, 0.06) }} />
                                  </Stack>
                                } 
                                secondary={<Typography variant="caption" color="text.secondary">{page.path}</Typography>} 
                              />
                              <Switch 
                                edge="end" 
                                checked={page.hasAccess} 
                                disabled={selectedRole.is_template}
                                onChange={() => togglePageAccess(page.id, !page.hasAccess)}
                                color="primary"
                              />
                            </ListItemButton>
                            {index < pages.length - 1 && <Divider />}
                          </React.Fragment>
                        ))}
                      </List>
                    </Paper>
                  )}
                </Box>
              )}

              {/* TAB 2: Users */}
              {tabIndex === 1 && (
                <Box>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="subtitle1" fontWeight={600}>Assigned Users ({assignedUsers.length})</Typography>
                    <Button 
                      variant="contained" 
                      startIcon={<AddIcon />} 
                      disableElevation 
                      sx={{ borderRadius: 2 }}
                      onClick={() => setOpenAssignUserModal(true)}
                    >
                      Assign User
                    </Button>
                  </Stack>

                  {assignedUsers.length === 0 ? (
                    <Paper variant="outlined" sx={{ borderRadius: 2, p: 4, textAlign: 'center' }}>
                      <GroupIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                      <Typography variant="body2" color="text.secondary">
                        No users assigned to this role.
                      </Typography>
                    </Paper>
                  ) : (
                    <Paper variant="outlined" sx={{ borderRadius: 2 }}>
                      <List dense>
                        {assignedUsers.map((user, index) => (
                          <React.Fragment key={user.id}>
                            <ListItemButton sx={{ py: 1.5 }}>
                              <ListItemAvatar>
                                <Avatar sx={{ width: 32, height: 32, bgcolor: alpha(theme.palette.primary.main, 0.1), color: 'primary.main', fontWeight: 600, fontSize: 14 }}>
                                  {user.user_name?.charAt(0)}
                                </Avatar>
                              </ListItemAvatar>
                              <ListItemText 
                                primary={<Typography variant="body2" fontWeight={600}>{user.user_name}</Typography>} 
                                secondary={
                                  <Typography variant="caption" color="text.secondary">
                                    {user.user_email} • Tenant: {user.tenant_name}
                                  </Typography>
                                } 
                              />
                              <Typography variant="caption" color="text.disabled">
                                {new Date(user.assigned_at).toLocaleDateString()}
                              </Typography>
                              <IconButton size="small" edge="end" color="error" sx={{ ml: 1 }} onClick={() => unassignUser(user.user_id)}>
                                <DeleteOutlineIcon fontSize="small" />
                              </IconButton>
                            </ListItemButton>
                            {index < assignedUsers.length - 1 && <Divider />}
                          </React.Fragment>
                        ))}
                      </List>
                    </Paper>
                  )}
                </Box>
              )}

              {/* TAB 3: Groups / Teams */}
              {tabIndex === 2 && (
                <Box>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="subtitle1" fontWeight={600}>Mapped Groups ({assignedGroups.length})</Typography>
                    <Button 
                      variant="contained" 
                      startIcon={<AddIcon />} 
                      disableElevation 
                      sx={{ borderRadius: 2 }}
                      onClick={() => setOpenMapGroupModal(true)}
                    >
                      Map Group
                    </Button>
                  </Stack>

                  {assignedGroups.length === 0 ? (
                    <Paper variant="outlined" sx={{ borderRadius: 2, p: 4, textAlign: 'center' }}>
                      <GroupIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                      <Typography variant="body2" color="text.secondary">
                        No groups or teams mapped to this role.
                      </Typography>
                    </Paper>
                  ) : (
                    <Paper variant="outlined" sx={{ borderRadius: 2 }}>
                      <List dense>
                        {assignedGroups.map((group, index) => (
                          <React.Fragment key={group.id}>
                            <ListItemButton sx={{ py: 1.5 }}>
                              <ListItemAvatar>
                                <Avatar sx={{ width: 32, height: 32, bgcolor: alpha(theme.palette.secondary.main, 0.1), color: 'secondary.main' }}>
                                  {group.source === 'ad' ? <SyncIcon fontSize="small" /> : <GroupIcon fontSize="small" />}
                                </Avatar>
                              </ListItemAvatar>
                              <ListItemText 
                                primary={
                                  <Stack direction="row" spacing={1} alignItems="center">
                                    <Typography variant="body2" fontWeight={600}>{group.group_name}</Typography>
                                    {group.source === 'ad' && (
                                      <Chip 
                                        icon={<SyncIcon sx={{ fontSize: 10 }} />} 
                                        label="AD" 
                                        size="small" 
                                        sx={{ height: 18, fontSize: 10 }} 
                                      />
                                    )}
                                  </Stack>
                                } 
                                secondary={
                                  <Typography variant="caption" color="text.secondary">
                                    Source: {group.source === 'ad' ? 'Active Directory Sync' : 'Local Team'} • 
                                    Type: {group.type === 'ad_group' ? 'AD Group' : 'Team'}
                                  </Typography>
                                } 
                              />
                              <Typography variant="caption" color="text.disabled">
                                {new Date(group.assigned_at).toLocaleDateString()}
                              </Typography>
                              <IconButton size="small" edge="end" color="error" sx={{ ml: 1 }}>
                                <DeleteOutlineIcon fontSize="small" />
                              </IconButton>
                            </ListItemButton>
                            {index < assignedGroups.length - 1 && <Divider />}
                          </React.Fragment>
                        ))}
                      </List>
                    </Paper>
                  )}
                </Box>
              )}

              {/* TAB 4: Dynamic Policies */}
              {tabIndex === 3 && (
                <Box>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                    <Box>
                      <Typography variant="subtitle1" fontWeight={600}>Dynamic Access Policies</Typography>
                      <Typography variant="caption" color="text.secondary">
                        Rules that grant access to resources based on user attributes.
                      </Typography>
                    </Box>
                    <Tooltip title={selectedRole?.is_template ? "Core roles are read-only" : ""}>
                      <span>
                        <Button
                          variant="contained"
                          startIcon={<AddIcon />}
                          disableElevation
                          sx={{ borderRadius: 2 }}
                          onClick={() => setOpenPolicyModal(true)}
                          disabled={selectedRole?.is_template}
                        >
                          Add Policy
                        </Button>
                      </span>
                    </Tooltip>
                  </Stack>

                  {selectedRole?.is_template && (
                    <Box sx={{ mb: 2, p: 1.5, bgcolor: alpha(theme.palette.warning.main, 0.1), borderRadius: 1, display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LockIcon fontSize="small" color="warning" />
                      <Typography variant="caption" color="warning.dark">
                        Core roles are read-only. Dynamic policies are managed by the system administrator.
                      </Typography>
                    </Box>
                  )}

                  {dynamicPolicies.length === 0 ? (
                    <Paper variant="outlined" sx={{ borderRadius: 2, p: 4, textAlign: 'center' }}>
                      <Typography variant="body2" color="text.secondary">
                        No dynamic policies configured for this role.
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Add a rule to automatically grant this role access to resources based on user attributes.
                      </Typography>
                    </Paper>
                  ) : (
                    <Paper variant="outlined" sx={{ borderRadius: 2 }}>
                      <List dense>
                        {dynamicPolicies.map((policy, index) => (
                          <React.Fragment key={policy.id}>
                            <ListItemButton sx={{ py: 1.5 }}>
                              <ListItemIcon sx={{ minWidth: 40 }}>
                                <ShieldIcon fontSize="small" color="action" />
                              </ListItemIcon>
                              <ListItemText
                                primary={
                                  <Stack direction="row" spacing={1} alignItems="center">
                                    <Typography variant="body2" fontWeight={600}>{policy.resource_type}</Typography>
                                    <Chip label={policy.action} size="small" color="primary" sx={{ height: 18, fontSize: '0.65rem' }} />
                                    <Chip label="ABAC" size="small" sx={{ height: 18, fontSize: '0.65rem', bgcolor: alpha(theme.palette.info.main, 0.1), color: 'info.main' }} />
                                  </Stack>
                                }
                                secondary={
                                  <Typography variant="caption" color="text.secondary" component="div" sx={{ mt: 0.5 }}>
                                    If user.{policy.user_attribute} == {policy.resource_type}.{policy.resource_attribute}
                                    {policy.description && <span> — {policy.description}</span>}
                                  </Typography>
                                }
                              />
                              <IconButton size="small" edge="end" color="error" onClick={() => deletePolicy(policy.id)} disabled={selectedRole?.is_template}>
                                <DeleteOutlineIcon fontSize="small" />
                              </IconButton>
                            </ListItemButton>
                            {index < dynamicPolicies.length - 1 && <Divider />}
                          </React.Fragment>
                        ))}
                      </List>
                    </Paper>
                  )}
                </Box>
              )}
            </Box>
          </>
        )}
      </Box>

      {/* Create Role Modal */}
      <Dialog 
        open={openCreateRoleModal} 
        onClose={() => setOpenCreateRoleModal(false)} 
        maxWidth="sm" 
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Create New Role</Typography>
              <Typography variant="body2" color="text.secondary">Define a new role for your organization.</Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenCreateRoleModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Stack spacing={2.5}>
            <TextField
              autoFocus
              label="Role Key"
              placeholder="e.g., tenant.support_agent"
              fullWidth
              variant="outlined"
              value={newRoleForm.role_key}
              onChange={(e) => setNewRoleForm({ ...newRoleForm, role_key: e.target.value })}
              helperText="Unique identifier for the role."
            />
            <TextField
              label="Role Name"
              placeholder="e.g., Support Agent"
              fullWidth
              variant="outlined"
              value={newRoleForm.role_name}
              onChange={(e) => setNewRoleForm({ ...newRoleForm, role_name: e.target.value })}
            />
            <TextField
              label="Description"
              placeholder="Brief description of this role's purpose"
              fullWidth
              multiline
              rows={3}
              variant="outlined"
              value={newRoleForm.description}
              onChange={(e) => setNewRoleForm({ ...newRoleForm, description: e.target.value })}
            />
            <TextField
              select
              label="Role Level"
              fullWidth
              value={newRoleForm.role_level}
              onChange={(e) => setNewRoleForm({ 
                ...newRoleForm, 
                role_level: e.target.value as 'viewer' | 'editor' | 'approver' | 'admin' | 'super_admin' 
              })}
              variant="outlined"
            >
              <MenuItem value="viewer">Viewer</MenuItem>
              <MenuItem value="editor">Editor</MenuItem>
              <MenuItem value="approver">Approver</MenuItem>
              <MenuItem value="admin">Admin</MenuItem>
              <MenuItem value="super_admin">Super Admin</MenuItem>
            </TextField>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setOpenCreateRoleModal(false)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button 
            onClick={createRole} 
            variant="contained" 
            color="primary" 
            disableElevation 
            sx={{ borderRadius: 2, px: 3 }}
            disabled={!newRoleForm.role_key || !newRoleForm.role_name || saving}
          >
            {saving ? 'Creating...' : 'Create Role'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Assign User Modal (stubbed - needs user search) */}
      <Dialog 
        open={openAssignUserModal} 
        onClose={() => setOpenAssignUserModal(false)} 
        maxWidth="xs" 
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Assign User</Typography>
              <Typography variant="body2" color="text.secondary">
                Add a user to {selectedRole?.role_name}
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenAssignUserModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Autocomplete
            options={userSearchResults}
            getOptionLabel={(option) => `${option.full_name} (${option.email})`}
            loading={searchingUsers}
            inputValue={userSearchQuery}
            onInputChange={(_, value) => searchUsers(value)}
            onChange={(_, value) => {
              if (value) {
                setAssignUserForm({ user_id: value.id });
              }
            }}
            renderOption={(props, option) => (
              <Box component="li" {...props}>
                <Box>
                  <Typography variant="body2">{option.full_name}</Typography>
                  <Typography variant="caption" color="text.secondary">{option.email}</Typography>
                  {option.department && (
                    <Typography variant="caption" color="text.secondary"> — {option.department}</Typography>
                  )}
                </Box>
              </Box>
            )}
            renderInput={(params) => (
              <TextField
                {...params}
                placeholder="Search by name or email..."
                size="small"
                InputProps={{
                  ...params.InputProps,
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon fontSize="small" />
                    </InputAdornment>
                  ),
                  endAdornment: searchingUsers ? (
                    <InputAdornment position="end">
                      <CircularProgress size={16} />
                    </InputAdornment>
                  ) : null,
                }}
              />
            )}
            noOptionsText={userSearchQuery.length < 2 ? "Type at least 2 characters to search" : "No users found"}
          />
          {assignUserForm.user_id && (
            <Chip
              size="small"
              label="User selected"
              color="success"
              sx={{ mt: 1 }}
              onDelete={() => setAssignUserForm({ user_id: '' })}
            />
          )}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => { setOpenAssignUserModal(false); setAssignUserForm({ user_id: '' }); setUserSearchQuery(''); setUserSearchResults([]); }} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button
            onClick={assignUser}
            variant="contained"
            disabled={!assignUserForm.user_id || saving}
            sx={{ borderRadius: 2, px: 2 }}
          >
            Assign
          </Button>
        </DialogActions>
      </Dialog>

      {/* Map Group Modal (stubbed) */}
      <Dialog 
        open={openMapGroupModal} 
        onClose={() => setOpenMapGroupModal(false)} 
        maxWidth="xs" 
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Map Group</Typography>
              <Typography variant="body2" color="text.secondary">
                Map an AD group or team to {selectedRole?.role_name}
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenMapGroupModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Typography variant="body2" color="text.secondary">
            Group mapping will be implemented here.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setOpenMapGroupModal(false)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>

      {/* Create Dynamic Policy Modal */}
      <Dialog
        open={openPolicyModal}
        onClose={() => setOpenPolicyModal(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Create Dynamic Policy</Typography>
              <Typography variant="body2" color="text.secondary">
                Grant access to a resource if the user's attribute matches the resource's attribute.
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenPolicyModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Stack spacing={2.5}>
            <Box sx={{ p: 2, border: 1, borderColor: 'divider', borderRadius: 2, bgcolor: alpha(theme.palette.primary.main, 0.03) }}>
              <Typography variant="caption" color="text.secondary" fontWeight={700}>
                RULE LOGIC:
              </Typography>
              <Typography variant="body2" sx={{ mt: 1 }}>
                If User Attribute <strong>[{newPolicyForm.user_attribute || 'user_attribute'}]</strong> equals Resource Attribute <strong>[{newPolicyForm.resource_type || 'resource_type'}.{newPolicyForm.resource_attribute}]</strong>, grant <strong>[{newPolicyForm.action.toUpperCase()}]</strong> access.
              </Typography>
            </Box>

            <TextField
              label="Resource Type"
              placeholder="e.g., portfolios"
              fullWidth
              variant="outlined"
              value={newPolicyForm.resource_type}
              onChange={(e) => setNewPolicyForm({...newPolicyForm, resource_type: e.target.value})}
              helperText="The table or entity type (e.g., portfolios, documents)."
            />
            <TextField
              label="User Attribute"
              placeholder="e.g., assigned_portfolio_id"
              fullWidth
              variant="outlined"
              value={newPolicyForm.user_attribute}
              onChange={(e) => setNewPolicyForm({...newPolicyForm, user_attribute: e.target.value})}
              helperText="The key stored on the user's attributes."
            />
            <TextField
              label="Resource Attribute"
              placeholder="e.g., id"
              fullWidth
              variant="outlined"
              value={newPolicyForm.resource_attribute}
              onChange={(e) => setNewPolicyForm({...newPolicyForm, resource_attribute: e.target.value})}
              helperText="The column on the resource to match against (usually 'id')."
            />
            <TextField
              select
              label="Action"
              fullWidth
              variant="outlined"
              value={newPolicyForm.action}
              onChange={(e) => setNewPolicyForm({...newPolicyForm, action: e.target.value})}
            >
              <MenuItem value="read">Read</MenuItem>
              <MenuItem value="write">Write</MenuItem>
              <MenuItem value="delete">Delete</MenuItem>
            </TextField>
            <TextField
              label="Description (optional)"
              placeholder="e.g., Grant access to assigned portfolio"
              fullWidth
              variant="outlined"
              value={newPolicyForm.description}
              onChange={(e) => setNewPolicyForm({...newPolicyForm, description: e.target.value})}
            />
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setOpenPolicyModal(false)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button
            onClick={createPolicy}
            variant="contained"
            color="primary"
            disableElevation
            sx={{ borderRadius: 2, px: 3 }}
            disabled={!newPolicyForm.resource_type || !newPolicyForm.user_attribute || !newPolicyForm.resource_attribute || saving}
          >
            {saving ? 'Creating...' : 'Save Policy'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
