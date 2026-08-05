/**
 * UserManager_MasterDetail - Enterprise User Management
 * 
 * Master-Detail layout with contextual tabs:
 * - Roles Tab: User's assigned roles with scope
 * - Teams Tab: User's team memberships
 * - Details Tab: User profile information
 */

import React, { useState, useEffect, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { apiClient } from '../../utils/apiClient';
import {
  Box, Paper, Typography, Button, TextField, InputAdornment, List, ListItemButton,
  ListItemText, ListItemAvatar, Chip, IconButton, Tabs, Tab, Stack, Avatar, 
  Divider, useTheme, alpha, Grid, CircularProgress, Checkbox
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import PersonIcon from '@mui/icons-material/Person';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import BusinessCenterIcon from '@mui/icons-material/BusinessCenter';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import SecurityIcon from '@mui/icons-material/Security';
import SyncIcon from '@mui/icons-material/Sync';
import CloseIcon from '@mui/icons-material/Close';
import EditIcon from '@mui/icons-material/Edit';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import MenuItem from '@mui/material/MenuItem';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface User {
  id: string;
  username: string;
  email: string;
  full_name: string;
  department?: string;
  title?: string;
  is_active: boolean;
  last_login?: string;
  created_at: string;
}

interface UserRole {
  id: string;
  role_id: string;
  role_key: string;
  role_name: string;
  role_level: string;
  scope_type: 'global' | 'process' | 'step' | 'team';
  scope_id?: string;
  scope_name?: string;
  tenant_name?: string;
  assigned_by?: string;
  assigned_at: string;
}

interface UserTeam {
  id: string;
  team_id: string;
  team_name: string;
  team_type: string;
  source: 'local' | 'ad';
  role_in_team: string;
  joined_at: string;
}

interface Team {
  id: string;
  team_key: string;
  team_name: string;
  team_type: string;
  source: 'local' | 'ad';
}

interface Role {
  id: string;
  role_key: string;
  role_name: string;
  role_level: string;
}

interface UserManagerProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const UserManagerMasterDetail: React.FC<UserManagerProps> = ({ tenant, datasource }) => {
  const theme = useTheme();
  const [searchParams] = useSearchParams();
  const selectedParam = searchParams.get('selected') || searchParams.get('userId');

  const [users, setUsers] = useState<User[]>([]);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [userRoles, setUserRoles] = useState<UserRole[]>([]);
  const [userTeams, setUserTeams] = useState<UserTeam[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [tabIndex, setTabIndex] = useState(0);
  const [openAssignRoleModal, setOpenAssignRoleModal] = useState(false);
  const [openAddToTeamModal, setOpenAddToTeamModal] = useState(false);
  const [openCreateUserModal, setOpenCreateUserModal] = useState(false);
  const [editingRole, setEditingRole] = useState<UserRole | null>(null);
  const [editingTeam, setEditingTeam] = useState<UserTeam | null>(null);

  const [assignRoleForm, setAssignRoleForm] = useState({
    role_id: '',
    scope_type: 'global' as 'global' | 'process' | 'step' | 'team',
    scope_id: '',
  });

  const [addToTeamForm, setAddToTeamForm] = useState({
    team_id: '',
    role_in_team: 'member' as 'member' | 'lead' | 'admin',
  });

  const [editRoleForm, setEditRoleForm] = useState({
    scope_type: 'global' as 'global' | 'process' | 'step' | 'team',
    scope_id: '',
  });

  const [editTeamForm, setEditTeamForm] = useState({
    role_in_team: 'member' as 'member' | 'lead' | 'admin',
  });

  const [createUserForm, setCreateUserForm] = useState({
    username: '',
    email: '',
    full_name: '',
    password: '',
    department: '',
  });

  // User Attributes state
  const [userAttributes, setUserAttributes] = useState<Record<string, string>>({});
  const [newAttrKey, setNewAttrKey] = useState('');
  const [newAttrValue, setNewAttrValue] = useState('');

  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedUserIds(filteredUsers.map(u => u.id));
    } else {
      setSelectedUserIds([]);
    }
  };

  const handleSelectOne = (userId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setSelectedUserIds(prev =>
      prev.includes(userId) ? prev.filter(id => id !== userId) : [...prev, userId]
    );
  };

  const deleteSelectedUsers = async () => {
    if (selectedUserIds.length === 0) return;
    const count = selectedUserIds.length;
    if (!confirm(`Are you sure you want to delete ${count} user(s)?`)) return;

    try {
      setSaving(true);
      await Promise.all(
        selectedUserIds.map(id =>
          apiClient(`/rbac/users/${id}`, { method: 'DELETE' }, { tenantId: tenant.id })
        )
      );
      setSelectedUserIds([]);
      await fetchUsers();
    } catch (error) {
      console.error('Failed to delete selected users:', error);
      alert('Failed to delete selected users');
      await fetchUsers();
    } finally {
      setSaving(false);
    }
  };

  // Create new user
  const createUser = async () => {
    if (!createUserForm.username || !createUserForm.email || !createUserForm.full_name) {
      alert('Please fill in all required fields');
      return;
    }
    try {
      setSaving(true);
      await apiClient(
        '/rbac/users',
        {
          method: 'POST',
          body: JSON.stringify({
            username: createUserForm.username,
            email: createUserForm.email,
            full_name: createUserForm.full_name,
            password: createUserForm.password || 'password123',
            department: createUserForm.department,
            tenant_id: tenant.id,
          }),
        },
        { tenantId: tenant.id }
      );
      setOpenCreateUserModal(false);
      setCreateUserForm({ username: '', email: '', full_name: '', password: '', department: '' });
      await fetchUsers();
    } catch (error) {
      console.error('Failed to create user:', error);
      alert('Failed to create user');
    } finally {
      setSaving(false);
    }
  };

  // Fetch users
  const fetchUsers = async () => {
    try {
      setLoading(true);
      const data = await apiClient<User[]>('/rbac/users', {}, { tenantId: tenant.id });
      const usersArray = Array.isArray(data)
        ? data
        : Array.isArray((data as any)?.users)
        ? (data as any).users
        : Array.isArray((data as any)?.data)
        ? (data as any).data
        : [];
      setUsers(usersArray);
      if (usersArray.length > 0) {
        if (selectedParam) {
          const found = usersArray.find(u => u.id === selectedParam);
          if (found) setSelectedUser(found);
          else if (!selectedUser) setSelectedUser(usersArray[0]);
        } else if (!selectedUser) {
          setSelectedUser(usersArray[0]);
        }
      }
    } catch (error) {
      console.error('Failed to fetch users:', error);
      setUsers([]);
    } finally {
      setLoading(false);
    }
  };

  // Fetch user roles
  const fetchUserRoles = async (userId: string) => {
    try {
      const data = await apiClient<UserRole[]>(`/rbac/users/${userId}/access`, {}, { tenantId: tenant.id });
      const rolesArray = Array.isArray(data) ? data : [];
      setUserRoles(rolesArray);
    } catch (error) {
      console.error('Failed to fetch user roles:', error);
      setUserRoles([]);
    }
  };

  // Fetch user teams
  const fetchUserTeams = async (userId: string) => {
    try {
      const data = await apiClient<UserTeam[]>(`/rbac/users/${userId}/teams`, {}, { tenantId: tenant.id });
      const teamsArray = Array.isArray(data) ? data : [];
      setUserTeams(teamsArray);
    } catch (error) {
      console.error('Failed to fetch user teams:', error);
      setUserTeams([]);
    }
  };

  // Fetch teams for add to team modal
  const fetchTeams = async () => {
    try {
      const data = await apiClient<Team[]>('/rbac/teams', {}, { tenantId: tenant.id });
      const teamsArray = Array.isArray(data) ? data : [];
      setTeams(teamsArray);
    } catch (error) {
      console.error('Failed to fetch teams:', error);
      setTeams([]);
    }
  };

  // Fetch roles for assign role modal
  const fetchRoles = async () => {
    try {
      const data = await apiClient<Role[]>('/rbac/roles', {}, { tenantId: tenant.id });
      const rolesArray = Array.isArray(data) ? data : [];
      setRoles(rolesArray);
    } catch (error) {
      console.error('Failed to fetch roles:', error);
      setRoles([]);
    }
  };

  // Assign role to user
  const assignRole = async () => {
    if (!selectedUser || !assignRoleForm.role_id) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/roles/${assignRoleForm.role_id}/assign`,
        {
          method: 'POST',
          body: JSON.stringify({
            user_id: selectedUser.id,
            scope_type: assignRoleForm.scope_type,
            scope_id: assignRoleForm.scope_id || null,
          }),
        },
        { tenantId: tenant.id, datasourceId: datasource.id }
      );
      await fetchUserRoles(selectedUser.id);
      setOpenAssignRoleModal(false);
      setAssignRoleForm({ role_id: '', scope_type: 'global', scope_id: '' });
    } catch (error) {
      console.error('Failed to assign role:', error);
    } finally {
      setSaving(false);
    }
  };

  // Unassign role from user
  const unassignRole = async (roleId: string) => {
    if (!selectedUser || !confirm('Remove this role assignment?')) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/roles/${roleId}/unassign/${selectedUser.id}`,
        {
          method: 'DELETE',
        },
        { tenantId: tenant.id, datasourceId: datasource.id }
      );
      await fetchUserRoles(selectedUser.id);
    } catch (error) {
      console.error('Failed to unassign role:', error);
    } finally {
      setSaving(false);
    }
  };

  // Add user to team
  const addToTeam = async () => {
    if (!selectedUser || !addToTeamForm.team_id) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/teams/${addToTeamForm.team_id}/members`,
        {
          method: 'POST',
          body: JSON.stringify({
            user_id: selectedUser.id,
            role_in_team: addToTeamForm.role_in_team,
          }),
        },
        { tenantId: tenant.id, datasourceId: datasource.id }
      );
      await fetchUserTeams(selectedUser.id);
      setOpenAddToTeamModal(false);
      setAddToTeamForm({ team_id: '', role_in_team: 'member' });
    } catch (error) {
      console.error('Failed to add to team:', error);
      alert('Failed to add user to team');
    } finally {
      setSaving(false);
    }
  };

  // Remove user from team
  const removeFromTeam = async (teamId: string) => {
    if (!selectedUser || !confirm('Remove user from this team?')) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/teams/${teamId}/members/${selectedUser.id}`,
        { method: 'DELETE' },
        { tenantId: tenant.id, datasourceId: datasource.id }
      );
      await fetchUserTeams(selectedUser.id);
    } catch (error) {
      console.error('Failed to remove from team:', error);
      alert('Failed to remove user from team');
    } finally {
      setSaving(false);
    }
  };

  // Update role assignment
  const updateRoleAssignment = async () => {
    if (!selectedUser || !editingRole) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/roles/${editingRole.role_id}/unassign/${selectedUser.id}`,
        {
          method: 'PATCH',
          body: JSON.stringify({
            scope_type: editRoleForm.scope_type,
            scope_id: editRoleForm.scope_id || null,
          }),
        },
        { tenantId: tenant.id, datasourceId: datasource.id }
      );
      await fetchUserRoles(selectedUser.id);
      setEditingRole(null);
      setEditRoleForm({ scope_type: 'global', scope_id: '' });
    } catch (error) {
      console.error('Failed to update role assignment:', error);
      alert('Failed to update role assignment');
    } finally {
      setSaving(false);
    }
  };

  // Update team member role
  const updateTeamMemberRole = async () => {
    if (!selectedUser || !editingTeam) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/teams/${editingTeam.team_id}/members/${selectedUser.id}`,
        {
          method: 'PATCH',
          body: JSON.stringify({
            role_in_team: editTeamForm.role_in_team,
          }),
        },
        { tenantId: tenant.id, datasourceId: datasource.id }
      );
      await fetchUserTeams(selectedUser.id);
      setEditingTeam(null);
      setEditTeamForm({ role_in_team: 'member' });
    } catch (error) {
      console.error('Failed to update team member:', error);
      alert('Failed to update team member');
    } finally {
      setSaving(false);
    }
  };

  // Fetch user attributes
  const fetchUserAttributes = async (userId: string) => {
    try {
      const response = await apiClient(
        `/rbac/users/${userId}/attributes`,
        {},
        { tenantId: tenant.id }
      );
      const data = await response.json();
      if (data.jsonb_attributes) {
        setUserAttributes(data.jsonb_attributes);
      }
    } catch (error) {
      console.error('Failed to fetch user attributes:', error);
    }
  };

  // Add attribute
  const addAttribute = async () => {
    if (!selectedUser || !newAttrKey || !newAttrValue) return;
    try {
      setSaving(true);
      const updatedAttrs = { ...userAttributes, [newAttrKey]: newAttrValue };
      await apiClient(
        `/rbac/users/${selectedUser.id}/attributes`,
        {
          method: 'PUT',
          body: JSON.stringify({ attributes: updatedAttrs }),
        },
        { tenantId: tenant.id }
      );
      setUserAttributes(updatedAttrs);
      setNewAttrKey('');
      setNewAttrValue('');
    } catch (error) {
      console.error('Failed to add attribute:', error);
      alert('Failed to add attribute');
    } finally {
      setSaving(false);
    }
  };

  // Delete attribute
  const deleteAttribute = async (key: string) => {
    if (!selectedUser) return;
    try {
      setSaving(true);
      const updatedAttrs = { ...userAttributes };
      delete updatedAttrs[key];
      await apiClient(
        `/rbac/users/${selectedUser.id}/attributes`,
        {
          method: 'PUT',
          body: JSON.stringify({ attributes: updatedAttrs }),
        },
        { tenantId: tenant.id }
      );
      setUserAttributes(updatedAttrs);
    } catch (error) {
      console.error('Failed to delete attribute:', error);
      alert('Failed to delete attribute');
    } finally {
      setSaving(false);
    }
  };

  // Filter users
  const filteredUsers = useMemo(() => {
    return users.filter(user => 
      user.full_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      user.email?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (user.department?.toLowerCase().includes(searchTerm.toLowerCase()) ?? false)
    );
  }, [users, searchTerm]);

  // Load data on mount
  useEffect(() => {
    fetchUsers();
    fetchTeams();
    fetchRoles();
  }, [tenant.id]);

  // Fetch user data when selection changes
  useEffect(() => {
    if (selectedUser) {
      fetchUserRoles(selectedUser.id);
      fetchUserTeams(selectedUser.id);
      fetchUserAttributes(selectedUser.id);
    } else {
      setUserRoles([]);
      setUserTeams([]);
      setUserAttributes({});
    }
  }, [selectedUser]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 64px)', bgcolor: 'background.default' }}>
      
      {/* Left Sidebar: User List */}
      <Paper 
        elevation={0} 
        sx={{ width: 320, borderRight: `1px solid ${theme.palette.divider}`, display: 'flex', flexDirection: 'column' }}
      >
        <Box sx={{ p: 2, borderBottom: `1px solid ${theme.palette.divider}` }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
            <Typography variant="h6" fontWeight={700}>Users</Typography>
            <Stack direction="row" spacing={1} alignItems="center">
              {selectedUserIds.length > 0 && (
                <Button
                  size="small"
                  variant="contained"
                  color="error"
                  startIcon={<DeleteOutlineIcon fontSize="small" />}
                  onClick={deleteSelectedUsers}
                  sx={{ textTransform: 'none', fontWeight: 600, px: 1.5 }}
                >
                  Delete ({selectedUserIds.length})
                </Button>
              )}
              <IconButton size="small" color="primary" sx={{ bgcolor: alpha(theme.palette.primary.main, 0.1) }} onClick={() => setOpenCreateUserModal(true)}>
                <AddIcon fontSize="small" />
              </IconButton>
            </Stack>
          </Stack>
          
          <TextField
            fullWidth
            size="small"
            placeholder="Search by name or email..."
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
          {filteredUsers.length === 0 ? (
            <Box sx={{ textAlign: 'center', mt: 4, p: 2 }}>
              <Typography variant="body2" color="text.secondary">
                No users found.
              </Typography>
            </Box>
          ) : (
            <List dense disablePadding>
              <Box sx={{ px: 2, py: 1, display: 'flex', alignItems: 'center', borderBottom: `1px solid ${theme.palette.divider}`, bgcolor: alpha(theme.palette.text.primary, 0.02) }}>
                <Checkbox
                  size="small"
                  indeterminate={selectedUserIds.length > 0 && selectedUserIds.length < filteredUsers.length}
                  checked={filteredUsers.length > 0 && selectedUserIds.length === filteredUsers.length}
                  onChange={(e) => handleSelectAll(e.target.checked)}
                />
                <Typography variant="caption" fontWeight={600} color="text.secondary" sx={{ ml: 1 }}>
                  Select All ({filteredUsers.length})
                </Typography>
              </Box>
              {filteredUsers.map((user) => (
                <ListItemButton
                  key={user.id}
                  selected={selectedUser?.id === user.id}
                  onClick={() => setSelectedUser(user)}
                  sx={{ 
                    py: 1.5, 
                    borderLeft: selectedUser?.id === user.id ? `3px solid ${theme.palette.primary.main}` : '3px solid transparent',
                    '&:hover': { bgcolor: alpha(theme.palette.primary.main, 0.05) }
                  }}
                >
                  <Checkbox
                    size="small"
                    checked={selectedUserIds.includes(user.id)}
                    onChange={(e) => handleSelectOne(user.id, e as any)}
                    onClick={(e) => e.stopPropagation()}
                    sx={{ mr: 1 }}
                  />
                  <ListItemAvatar>
                    <Avatar 
                      sx={{ 
                        width: 36, 
                        height: 36, 
                        bgcolor: alpha(theme.palette.primary.main, 0.1), 
                        color: 'primary.main', 
                        fontWeight: 600 
                      }}
                    >
                      {(user.full_name || user.username || user.email || 'U').charAt(0).toUpperCase()}
                    </Avatar>
                  </ListItemAvatar>
                  <ListItemText 
                    primary={<Typography variant="body2" fontWeight={600} sx={{ textDecoration: !user.is_active ? 'line-through' : 'none', color: !user.is_active ? 'text.secondary' : 'text.primary' }}>{user.full_name || user.username}</Typography>} 
                    secondary={
                      <Typography variant="caption" color="text.secondary" component="div" sx={{ textDecoration: !user.is_active ? 'line-through' : 'none' }}>
                        {user.email}
                        {user.department && ` • ${user.department}`}
                      </Typography>
                    }
                  />
                  {!user.is_active && (
                    <Chip label="Inactive" size="small" color="error" sx={{ height: 18, fontSize: '0.65rem' }} />
                  )}
                </ListItemButton>
              ))}
            </List>
          )}
        </Box>
      </Paper>

      {/* Right Pane: User Detail View */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {!selectedUser ? (
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
            <Avatar sx={{ width: 80, height: 80, mb: 3, bgcolor: alpha(theme.palette.primary.main, 0.1), color: 'primary.main' }}>
              <PersonIcon sx={{ fontSize: 40 }} />
            </Avatar>
            <Typography variant="h5" fontWeight={700} gutterBottom>No User Selected</Typography>
            <Typography variant="body1" color="text.secondary" align="center" sx={{ maxWidth: 400 }}>
              Select a user from the left to manage their roles, teams, and profile details.
            </Typography>
          </Box>
        ) : (
          <>
            {/* Header */}
            <Box sx={{ p: 3, borderBottom: `1px solid ${theme.palette.divider}` }}>
              <Stack direction="row" spacing={2} alignItems="center">
                <Avatar 
                  sx={{ 
                    width: 56, 
                    height: 56, 
                    bgcolor: alpha(theme.palette.primary.main, 0.1), 
                    color: 'primary.main', 
                    fontWeight: 700 
                  }}
                >
                  {selectedUser.full_name?.charAt(0).toUpperCase()}
                </Avatar>
                <Box sx={{ flex: 1 }}>
                  <Stack direction="row" spacing={1.5} alignItems="center">
                    <Typography variant="h5" fontWeight={700}>{selectedUser.full_name}</Typography>
                    <Chip 
                      label={selectedUser.is_active ? 'Active' : 'Inactive'} 
                      size="small" 
                      color={selectedUser.is_active ? 'success' : 'error'} 
                      variant="outlined"
                      sx={{ textTransform: 'capitalize', borderRadius: 1 }}
                    />
                  </Stack>
                  <Typography variant="body2" color="text.secondary">{selectedUser.email}</Typography>
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
              <Tab label="Roles" />
              <Tab label="Teams" />
              <Tab label="Details" />
              <Tab label="Attributes" />
            </Tabs>

            {/* Tab Content */}
            <Box sx={{ flex: 1, p: 3, overflowY: 'auto' }}>
              
              {/* TAB 1: Roles & Access */}
              {tabIndex === 0 && (
                <Box>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="subtitle1" fontWeight={600}>Assigned Roles ({userRoles.length})</Typography>
                    <Button 
                      variant="contained" 
                      startIcon={<AddIcon />} 
                      disableElevation 
                      sx={{ borderRadius: 2 }}
                      onClick={() => setOpenAssignRoleModal(true)}
                    >
                      Assign Role
                    </Button>
                  </Stack>

                  {userRoles.length === 0 ? (
                    <Paper variant="outlined" sx={{ borderRadius: 2, p: 4, textAlign: 'center' }}>
                      <SecurityIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                      <Typography variant="body2" color="text.secondary">
                        No roles assigned to this user.
                      </Typography>
                    </Paper>
                  ) : (
                    <Paper variant="outlined" sx={{ borderRadius: 2 }}>
                      <List dense>
                        {userRoles.map((role, index) => (
                          <React.Fragment key={role.id}>
                            <ListItemButton sx={{ py: 1.5 }}>
                              <ListItemText 
                                primary={
                                  <Stack direction="row" spacing={1} alignItems="center">
                                    <Typography variant="body2" fontWeight={600}>{role.role_name}</Typography>
                                    <Chip 
                                      label={role.role_key} 
                                      size="small" 
                                      sx={{ height: 20, fontSize: 10, bgcolor: alpha(theme.palette.text.primary, 0.08) }} 
                                    />
                                  </Stack>
                                } 
                                secondary={
                                  <Typography variant="caption" color="text.secondary" component="div" sx={{ mt: 0.5 }}>
                                    <Stack direction="row" spacing={2}>
                                      <span>Tenant: <strong>{role.tenant_name || 'All'}</strong></span>
                                      {role.scope_type !== 'global' && (
                                        <span>Scope: <strong>{role.scope_name || role.scope_type}</strong></span>
                                      )}
                                      {role.assigned_by && <span>Assigned by: {role.assigned_by}</span>}
                                    </Stack>
                                  </Typography>
                                } 
                              />
                              <IconButton size="small" edge="end" onClick={() => { setEditingRole(role); setEditRoleForm({ scope_type: role.scope_type as any, scope_id: role.scope_id || '' }); }}>
                                <EditIcon fontSize="small" />
                              </IconButton>
                              <IconButton size="small" edge="end" color="error" onClick={() => unassignRole(role.role_id)}>
                                <DeleteOutlineIcon fontSize="small" />
                              </IconButton>
                            </ListItemButton>
                            {index < userRoles.length - 1 && <Divider />}
                          </React.Fragment>
                        ))}
                      </List>
                    </Paper>
                  )}
                </Box>
              )}

              {/* TAB 2: Teams */}
              {tabIndex === 1 && (
                <Box>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="subtitle1" fontWeight={600}>Team Memberships ({userTeams.length})</Typography>
                    <Button 
                      variant="contained" 
                      startIcon={<AddIcon />} 
                      disableElevation 
                      sx={{ borderRadius: 2 }}
                      onClick={() => setOpenAddToTeamModal(true)}
                    >
                      Add to Team
                    </Button>
                  </Stack>

                  {userTeams.length === 0 ? (
                    <Paper variant="outlined" sx={{ borderRadius: 2, p: 4, textAlign: 'center' }}>
                      <BusinessCenterIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                      <Typography variant="body2" color="text.secondary">
                        User is not a member of any teams.
                      </Typography>
                    </Paper>
                  ) : (
                    <Paper variant="outlined" sx={{ borderRadius: 2 }}>
                      <List dense>
                        {userTeams.map((team, index) => (
                          <React.Fragment key={team.id}>
                            <ListItemButton sx={{ py: 1.5 }}>
                              <ListItemAvatar>
                                <Avatar sx={{ width: 32, height: 32, bgcolor: alpha(theme.palette.secondary.main, 0.1), color: 'secondary.main' }}>
                                  <BusinessCenterIcon fontSize="small" />
                                </Avatar>
                              </ListItemAvatar>
                              <ListItemText 
                                primary={
                                  <Stack direction="row" spacing={1} alignItems="center">
                                    <Typography variant="body2" fontWeight={600}>{team.team_name}</Typography>
                                    {team.source === 'ad' && (
                                      <SyncIcon sx={{ fontSize: 12, color: 'text.secondary' }} />
                                    )}
                                  </Stack>
                                } 
                                secondary={
                                  <Typography variant="caption" color="text.secondary">
                                    {team.team_type} Team • Role: {team.role_in_team}
                                  </Typography>
                                } 
                              />
                              {team.source === 'local' && (
                                <>
                                  <IconButton size="small" edge="end" onClick={() => { setEditingTeam(team); setEditTeamForm({ role_in_team: team.role_in_team as any }); }}>
                                    <EditIcon fontSize="small" />
                                  </IconButton>
                                  <IconButton size="small" edge="end" color="error" onClick={() => removeFromTeam(team.team_id)}>
                                    <DeleteOutlineIcon fontSize="small" />
                                  </IconButton>
                                </>
                              )}
                            </ListItemButton>
                            {index < userTeams.length - 1 && <Divider />}
                          </React.Fragment>
                        ))}
                      </List>
                    </Paper>
                  )}
                </Box>
              )}

              {/* TAB 3: Details */}
              {tabIndex === 2 && (
                <Box>
                  <Paper variant="outlined" sx={{ borderRadius: 2, p: 3 }}>
                    <Stack direction="row" spacing={1.5} alignItems="center" mb={3}>
                      <PersonIcon color="action" />
                      <Typography variant="subtitle1" fontWeight={600}>Profile Information</Typography>
                    </Stack>
                    <Grid container spacing={3}>
                      <Grid item xs={6}>
                        <Typography variant="overline" color="text.secondary" display="block">Full Name</Typography>
                        <Typography variant="body2">{selectedUser.full_name}</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="overline" color="text.secondary" display="block">Username</Typography>
                        <Typography variant="body2">{selectedUser.username}</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="overline" color="text.secondary" display="block">Email Address</Typography>
                        <Typography variant="body2">{selectedUser.email}</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="overline" color="text.secondary" display="block">Department</Typography>
                        <Typography variant="body2">{selectedUser.department || 'Not set'}</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="overline" color="text.secondary" display="block">Title</Typography>
                        <Typography variant="body2">{selectedUser.title || 'Not set'}</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="overline" color="text.secondary" display="block">Status</Typography>
                        <Chip
                          label={selectedUser.is_active ? 'Active' : 'Inactive'}
                          size="small"
                          color={selectedUser.is_active ? 'success' : 'error'}
                          sx={{ textTransform: 'capitalize', borderRadius: 1, mt: 0.5 }}
                        />
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="overline" color="text.secondary" display="block">Last Login</Typography>
                        <Typography variant="body2">{selectedUser.last_login || 'Never'}</Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="overline" color="text.secondary" display="block">Member Since</Typography>
                        <Typography variant="body2">{new Date(selectedUser.created_at).toLocaleDateString()}</Typography>
                      </Grid>
                    </Grid>
                  </Paper>
                </Box>
              )}

              {/* TAB 4: Attributes */}
              {tabIndex === 3 && (
                <Box>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                    <Box>
                      <Typography variant="subtitle1" fontWeight={600}>User Attributes</Typography>
                      <Typography variant="caption" color="text.secondary">
                        Key-value pairs used for dynamic ABAC policy evaluation.
                      </Typography>
                    </Box>
                  </Stack>

                  {/* Add Attribute Form */}
                  <Paper variant="outlined" sx={{ borderRadius: 2, p: 2, mb: 3 }}>
                    <Typography variant="body2" fontWeight={600} mb={1.5}>Add New Attribute</Typography>
                    <Stack direction="row" spacing={2} alignItems="flex-end">
                      <TextField
                        label="Attribute Key"
                        placeholder="e.g., assigned_portfolio_id"
                        size="small"
                        value={newAttrKey}
                        onChange={(e) => setNewAttrKey(e.target.value)}
                        sx={{ flex: 1 }}
                      />
                      <TextField
                        label="Attribute Value"
                        placeholder="e.g., port_999"
                        size="small"
                        value={newAttrValue}
                        onChange={(e) => setNewAttrValue(e.target.value)}
                        sx={{ flex: 1 }}
                      />
                      <Button
                        variant="contained"
                        startIcon={<AddIcon />}
                        disabled={!newAttrKey || !newAttrValue || saving}
                        onClick={addAttribute}
                        sx={{ borderRadius: 2 }}
                      >
                        Add
                      </Button>
                    </Stack>
                  </Paper>

                  {/* Existing Attributes */}
                  {Object.keys(userAttributes).length === 0 ? (
                    <Paper variant="outlined" sx={{ borderRadius: 2, p: 4, textAlign: 'center' }}>
                      <Typography variant="body2" color="text.secondary">
                        No attributes set for this user.
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Add attributes above to enable dynamic access control.
                      </Typography>
                    </Paper>
                  ) : (
                    <Paper variant="outlined" sx={{ borderRadius: 2 }}>
                      <List dense>
                        {Object.entries(userAttributes).map(([key, value], index) => (
                          <React.Fragment key={key}>
                            <ListItemButton sx={{ py: 1.5 }}>
                              <ListItemText
                                primary={
                                  <Stack direction="row" spacing={1} alignItems="center">
                                    <Typography variant="body2" fontWeight={600}>{key}</Typography>
                                    <Chip
                                      label="JSONB"
                                      size="small"
                                      sx={{ height: 18, fontSize: '0.65rem', bgcolor: alpha(theme.palette.info.main, 0.1), color: 'info.main' }}
                                    />
                                  </Stack>
                                }
                                secondary={
                                  <Typography variant="caption" color="text.secondary" component="div" sx={{ mt: 0.5 }}>
                                    {value}
                                  </Typography>
                                }
                              />
                              <IconButton size="small" edge="end" color="error" onClick={() => deleteAttribute(key)}>
                                <DeleteOutlineIcon fontSize="small" />
                              </IconButton>
                            </ListItemButton>
                            {index < Object.entries(userAttributes).length - 1 && <Divider />}
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

      {/* Assign Role Modal */}
      <Dialog 
        open={openAssignRoleModal} 
        onClose={() => setOpenAssignRoleModal(false)} 
        maxWidth="xs" 
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Assign Role</Typography>
              <Typography variant="body2" color="text.secondary">
                Assign a role to {selectedUser?.full_name}
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenAssignRoleModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Stack spacing={2.5}>
            <TextField
              select
              label="Role"
              fullWidth
              value={assignRoleForm.role_id}
              onChange={(e) => setAssignRoleForm({ ...assignRoleForm, role_id: e.target.value })}
              variant="outlined"
            >
              <MenuItem value="" disabled>
                <Typography variant="body2" color="text.secondary">Select a role...</Typography>
              </MenuItem>
              {roles.map(role => (
                <MenuItem key={role.id} value={role.id}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <SecurityIcon fontSize="small" color="primary" />
                    <Typography variant="body2">{role.role_name}</Typography>
                    <Typography variant="caption" color="text.secondary">({role.role_key})</Typography>
                  </Stack>
                </MenuItem>
              ))}
            </TextField>
            <TextField
              select
              label="Scope Type"
              fullWidth
              value={assignRoleForm.scope_type}
              onChange={(e) => setAssignRoleForm({ 
                ...assignRoleForm, 
                scope_type: e.target.value as 'global' | 'process' | 'step' | 'team' 
              })}
              variant="outlined"
            >
              <MenuItem value="global">Global (All resources)</MenuItem>
              <MenuItem value="team">Team (Within their team)</MenuItem>
              <MenuItem value="process">Process (Specific workflow)</MenuItem>
              <MenuItem value="step">Step (Specific step in a process)</MenuItem>
            </TextField>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setOpenAssignRoleModal(false)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button 
            onClick={assignRole} 
            variant="contained" 
            color="primary" 
            disableElevation 
            sx={{ borderRadius: 2, px: 3 }}
            disabled={!assignRoleForm.role_id || saving}
          >
            {saving ? 'Assigning...' : 'Assign Role'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Add to Team Modal */}
      <Dialog 
        open={openAddToTeamModal} 
        onClose={() => setOpenAddToTeamModal(false)} 
        maxWidth="xs" 
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Add to Team</Typography>
              <Typography variant="body2" color="text.secondary">
                Add {selectedUser?.full_name} to a team
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenAddToTeamModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Stack spacing={2.5}>
            <TextField
              select
              label="Team"
              fullWidth
              value={addToTeamForm.team_id}
              onChange={(e) => setAddToTeamForm({ ...addToTeamForm, team_id: e.target.value })}
              variant="outlined"
            >
              <MenuItem value="" disabled>
                <Typography variant="body2" color="text.secondary">Select a team...</Typography>
              </MenuItem>
              {teams
                .filter(team => !userTeams.some(ut => ut.team_id === team.id))
                .map(team => (
                  <MenuItem key={team.id} value={team.id}>
                    <Stack direction="row" spacing={1} alignItems="center">
                      <BusinessCenterIcon fontSize="small" color="secondary" />
                      <Typography variant="body2">{team.team_name}</Typography>
                      {team.source === 'ad' && <SyncIcon fontSize="small" color="action" />}
                    </Stack>
                  </MenuItem>
                ))}
            </TextField>
            <TextField
              select
              label="Role in Team"
              fullWidth
              value={addToTeamForm.role_in_team}
              onChange={(e) => setAddToTeamForm({ 
                ...addToTeamForm, 
                role_in_team: e.target.value as 'member' | 'lead' | 'admin' 
              })}
              variant="outlined"
            >
              <MenuItem value="member">Member</MenuItem>
              <MenuItem value="lead">Team Lead</MenuItem>
              <MenuItem value="admin">Team Admin</MenuItem>
            </TextField>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setOpenAddToTeamModal(false)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button 
            onClick={addToTeam} 
            variant="contained" 
            color="primary" 
            disableElevation 
            sx={{ borderRadius: 2, px: 3 }}
            disabled={!addToTeamForm.team_id || saving}
          >
            {saving ? 'Adding...' : 'Add to Team'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Role Modal */}
      <Dialog
        open={!!editingRole}
        onClose={() => setEditingRole(null)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Edit Role Assignment</Typography>
              <Typography variant="body2" color="text.secondary">
                Update scope for {editingRole?.role_name}
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setEditingRole(null)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Stack spacing={2.5}>
            <TextField
              select
              label="Scope Type"
              fullWidth
              value={editRoleForm.scope_type}
              onChange={(e) => setEditRoleForm({
                ...editRoleForm,
                scope_type: e.target.value as 'global' | 'process' | 'step' | 'team'
              })}
              variant="outlined"
            >
              <MenuItem value="global">Global (All resources)</MenuItem>
              <MenuItem value="team">Team (Within their team)</MenuItem>
              <MenuItem value="process">Process (Specific workflow)</MenuItem>
              <MenuItem value="step">Step (Specific step in a process)</MenuItem>
            </TextField>
            {editRoleForm.scope_type !== 'global' && (
              <TextField
                label="Scope ID"
                fullWidth
                value={editRoleForm.scope_id}
                onChange={(e) => setEditRoleForm({ ...editRoleForm, scope_id: e.target.value })}
                variant="outlined"
                placeholder="Enter scope ID..."
              />
            )}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setEditingRole(null)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button
            onClick={updateRoleAssignment}
            variant="contained"
            color="primary"
            disableElevation
            sx={{ borderRadius: 2, px: 3 }}
            disabled={saving}
          >
            {saving ? 'Updating...' : 'Update'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Team Member Modal */}
      <Dialog
        open={!!editingTeam}
        onClose={() => setEditingTeam(null)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Edit Team Role</Typography>
              <Typography variant="body2" color="text.secondary">
                Update role for {editingTeam?.team_name}
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setEditingTeam(null)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <TextField
            select
            label="Role in Team"
            fullWidth
            value={editTeamForm.role_in_team}
            onChange={(e) => setEditTeamForm({
              ...editTeamForm,
              role_in_team: e.target.value as 'member' | 'lead' | 'admin'
            })}
            variant="outlined"
          >
            <MenuItem value="member">Member</MenuItem>
            <MenuItem value="lead">Team Lead</MenuItem>
            <MenuItem value="admin">Team Admin</MenuItem>
          </TextField>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setEditingTeam(null)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button
            onClick={updateTeamMemberRole}
            variant="contained"
            color="primary"
            disableElevation
            sx={{ borderRadius: 2, px: 3 }}
            disabled={saving}
          >
            {saving ? 'Updating...' : 'Update'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Create User Modal */}
      <Dialog
        open={openCreateUserModal}
        onClose={() => setOpenCreateUserModal(false)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Create New User</Typography>
              <Typography variant="body2" color="text.secondary">
                Add a new user to the system
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenCreateUserModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Stack spacing={2.5}>
            <TextField
              label="Username"
              fullWidth
              value={createUserForm.username}
              onChange={(e) => setCreateUserForm({ ...createUserForm, username: e.target.value })}
              variant="outlined"
              required
            />
            <TextField
              label="Email"
              fullWidth
              type="email"
              value={createUserForm.email}
              onChange={(e) => setCreateUserForm({ ...createUserForm, email: e.target.value })}
              variant="outlined"
              required
            />
            <TextField
              label="Full Name"
              fullWidth
              value={createUserForm.full_name}
              onChange={(e) => setCreateUserForm({ ...createUserForm, full_name: e.target.value })}
              variant="outlined"
              required
            />
            <TextField
              label="Department"
              fullWidth
              value={createUserForm.department}
              onChange={(e) => setCreateUserForm({ ...createUserForm, department: e.target.value })}
              variant="outlined"
            />
            <TextField
              label="Password"
              fullWidth
              type="password"
              value={createUserForm.password}
              onChange={(e) => setCreateUserForm({ ...createUserForm, password: e.target.value })}
              variant="outlined"
              helperText="Leave empty for default password"
            />
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setOpenCreateUserModal(false)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button
            onClick={createUser}
            variant="contained"
            color="primary"
            disableElevation
            sx={{ borderRadius: 2, px: 3 }}
            disabled={saving || !createUserForm.username || !createUserForm.email || !createUserForm.full_name}
          >
            {saving ? 'Creating...' : 'Create User'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
