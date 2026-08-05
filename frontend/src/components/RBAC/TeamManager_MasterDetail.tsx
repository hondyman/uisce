/**
 * TeamManager_MasterDetail - Enterprise Team & Department Management
 * 
 * Master-Detail layout with contextual tabs:
 * - Members Tab: Team members with AD sync read-only banner
 * - Role Mappings Tab: Roles assigned to the team
 * - Resources Tab: ABAC resource policies (stubbed)
 */

import React, { useState, useEffect, useMemo } from 'react';
import { apiClient } from '../../utils/apiClient';
import {
  Box, Paper, Typography, Button, TextField, InputAdornment, List, ListItemButton,
  ListItemIcon, ListItemText, Chip, IconButton, Tabs, Tab, Stack, Alert, AlertTitle,
  Divider, useTheme, alpha, Avatar, Menu, MenuItem, ListItemAvatar, CircularProgress,
  FormControlLabel, Switch
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import GroupsIcon from '@mui/icons-material/Groups';
import BusinessCenterIcon from '@mui/icons-material/BusinessCenter';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import LockIcon from '@mui/icons-material/Lock';
import SyncIcon from '@mui/icons-material/Sync';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import FolderSharedIcon from '@mui/icons-material/FolderShared';
import HubIcon from '@mui/icons-material/Hub';
import EditIcon from '@mui/icons-material/Edit';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import PersonRemoveIcon from '@mui/icons-material/PersonRemove';
import ShieldIcon from '@mui/icons-material/Shield';
import CrownIcon from '@mui/icons-material/EmojiEvents';
import CloseIcon from '@mui/icons-material/Close';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface Team {
  id: string;
  team_key: string;
  team_name: string;
  description: string;
  team_type: 'functional' | 'project' | 'cross_functional';
  source: 'local' | 'ad';
  is_active?: boolean;
  manager_user_id?: string;
  member_count?: number;
  created_at: string;
}

interface TeamMember {
  id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  role_in_team: 'member' | 'lead' | 'admin';
  joined_at: string;
}

interface User {
  id: string;
  username: string;
  email: string;
  full_name: string;
  department?: string;
}

interface TeamManagerProps {
  tenant: { id: string; display_name?: string };
  datasource?: { id: string; source_name?: string } | null;
}

// ============================================================================
// CONSTANTS
// ============================================================================

const teamTypes = [
  { value: 'functional', label: 'Functional', icon: <BusinessCenterIcon fontSize="small" /> },
  { value: 'project', label: 'Project', icon: <FolderSharedIcon fontSize="small" /> },
  { value: 'cross_functional', label: 'Cross-Functional', icon: <HubIcon fontSize="small" /> },
];

const roleConfig = {
  admin: { icon: <CrownIcon fontSize="small" />, color: 'error' as const },
  lead: { icon: <ShieldIcon fontSize="small" />, color: 'warning' as const },
  member: { icon: <BusinessCenterIcon fontSize="small" />, color: 'primary' as const },
};

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const TeamManagerMasterDetail: React.FC<TeamManagerProps> = ({ tenant, datasource }) => {
  const theme = useTheme();
  const [teams, setTeams] = useState<Team[]>([]);
  const [selectedTeam, setSelectedTeam] = useState<Team | null>(null);
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [tabIndex, setTabIndex] = useState(0);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [memberMenuAnchor, setMemberMenuAnchor] = useState<null | HTMLElement>(null);
  const [selectedMember, setSelectedMember] = useState<TeamMember | null>(null);
  const [openCreateModal, setOpenCreateModal] = useState(false);
  const [openEditModal, setOpenEditModal] = useState(false);
  const [openAddMemberModal, setOpenAddMemberModal] = useState(false);

  const [teamForm, setTeamForm] = useState({
    team_key: '',
    team_name: '',
    description: '',
    team_type: 'functional' as 'functional' | 'project' | 'cross_functional',
  });

  const [editTeamForm, setEditTeamForm] = useState({
    team_key: '',
    team_name: '',
    description: '',
    team_type: 'functional' as 'functional' | 'project' | 'cross_functional',
    manager_user_id: '',
    is_active: true,
  });

  const [memberForm, setMemberForm] = useState({
    user_id: '',
    role_in_team: 'member' as 'member' | 'lead' | 'admin',
  });

  // Fetch teams
  const fetchTeams = async () => {
    try {
      setLoading(true);
      const data = await apiClient<Team[]>('/rbac/teams', {}, { tenantId: tenant.id });
      const teamsArray = Array.isArray(data)
        ? data
        : Array.isArray((data as any)?.teams)
        ? (data as any)?.teams
        : Array.isArray((data as any)?.data)
        ? (data as any)?.data
        : [];
      const uniqueTeams = teamsArray.filter((team, index, self) => 
        index === self.findIndex(t => t.id === team.id)
      );
      setTeams(uniqueTeams);
      if (teamsArray.length > 0) {
        if (selectedTeam) {
          const updated = teamsArray.find(t => t.id === selectedTeam.id);
          if (updated) setSelectedTeam(updated);
          else setSelectedTeam(teamsArray[0]);
        } else {
          setSelectedTeam(teamsArray[0]);
        }
      } else {
        setSelectedTeam(null);
      }
    } catch (error) {
      console.error('Failed to fetch teams:', error);
      setTeams([]);
    } finally {
      setLoading(false);
    }
  };

  // Fetch team members
  const fetchTeamMembers = async (teamId: string) => {
    try {
      const data = await apiClient<TeamMember[]>(`/rbac/teams/${teamId}/members`, {}, { tenantId: tenant.id });
      const membersArray = Array.isArray(data)
        ? data
        : Array.isArray((data as any)?.members)
        ? (data as any).members
        : Array.isArray((data as any)?.data)
        ? (data as any).data
        : [];
      setTeamMembers(membersArray);
    } catch (error) {
      console.error('Failed to fetch team members:', error);
      setTeamMembers([]);
    }
  };

  // Fetch users for add member modal
  const fetchUsers = async () => {
    try {
      const data = await apiClient<User[]>('/rbac/users', {}, { tenantId: tenant.id });
      const usersArray = Array.isArray(data)
        ? data
        : Array.isArray((data as any)?.users)
        ? (data as any).users
        : Array.isArray((data as any)?.data)
        ? (data as any).data
        : [];
      setUsers(usersArray);
    } catch (error) {
      console.error('Failed to fetch users:', error);
      setUsers([]);
    }
  };

  // Create team
  const createTeam = async () => {
    try {
      setSaving(true);
      await apiClient(
        '/rbac/teams',
        {
          method: 'POST',
          body: JSON.stringify({
            team_key: teamForm.team_key,
            team_name: teamForm.team_name,
            description: teamForm.description,
            team_type: teamForm.team_type,
          }),
        },
        { tenantId: tenant.id, datasourceId: datasource?.id }
      );
      await fetchTeams();
      setOpenCreateModal(false);
      resetTeamForm();
    } catch (error) {
      console.error('Failed to create team:', error);
      alert('Failed to create team');
    } finally {
      setSaving(false);
    }
  };

  // Open Edit Team modal
  const handleOpenEditModal = () => {
    if (!selectedTeam) return;
    setEditTeamForm({
      team_key: selectedTeam.team_key || '',
      team_name: selectedTeam.team_name || '',
      description: selectedTeam.description || '',
      team_type: selectedTeam.team_type || 'functional',
      manager_user_id: selectedTeam.manager_user_id || '',
      is_active: selectedTeam.is_active ?? true,
    });
    setOpenEditModal(true);
  };

  // Update team details & status
  const updateTeam = async () => {
    if (!selectedTeam) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/teams/${selectedTeam.id}`,
        {
          method: 'PUT',
          body: JSON.stringify(editTeamForm),
        },
        { tenantId: tenant.id }
      );
      await fetchTeams();
      setOpenEditModal(false);
    } catch (error) {
      console.error('Failed to update team:', error);
      alert('Failed to update team details');
    } finally {
      setSaving(false);
    }
  };

  // Delete team
  const deleteTeam = async (teamId: string) => {
    if (!confirm('Are you sure you want to delete this team? This action cannot be undone.')) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/teams/${teamId}`,
        { method: 'DELETE' },
        { tenantId: tenant.id }
      );
      await fetchTeams();
      if (selectedTeam?.id === teamId) {
        setSelectedTeam(null);
        setTeamMembers([]);
      }
    } catch (error) {
      console.error('Failed to delete team:', error);
      alert('Failed to delete team');
    } finally {
      setSaving(false);
    }
  };

  // Add team member
  const addTeamMember = async () => {
    if (!selectedTeam) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/teams/${selectedTeam.id}/members`,
        {
          method: 'POST',
          body: JSON.stringify({
            user_id: memberForm.user_id,
            role_in_team: memberForm.role_in_team,
          }),
        },
        { tenantId: tenant.id, datasourceId: datasource?.id }
      );
      await fetchTeamMembers(selectedTeam.id);
      await fetchTeams();
      setOpenAddMemberModal(false);
      resetMemberForm();
    } catch (error) {
      console.error('Failed to add team member:', error);
      alert('Failed to add team member');
    } finally {
      setSaving(false);
    }
  };

  // Remove team member
  const removeTeamMember = async (memberId: string) => {
    if (!selectedTeam || !confirm('Remove this member from the team?')) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/teams/${selectedTeam.id}/members/${memberId}`,
        { method: 'DELETE' },
        { tenantId: tenant.id }
      );
      await fetchTeamMembers(selectedTeam.id);
      await fetchTeams();
    } catch (error) {
      console.error('Failed to remove team member:', error);
      alert('Failed to remove team member');
    } finally {
      setSaving(false);
    }
  };

  // Promote member to manager
  const promoteToManager = async (memberId: string) => {
    if (!selectedTeam) return;
    try {
      setSaving(true);
      await apiClient(
        `/rbac/teams/${selectedTeam.id}/members/${memberId}`,
        {
          method: 'PATCH',
          body: JSON.stringify({
            role_in_team: 'admin',
          }),
        },
        { tenantId: tenant.id, datasourceId: datasource?.id }
      );
      await fetchTeamMembers(selectedTeam.id);
      await fetchTeams();
    } catch (error) {
      console.error('Failed to promote member:', error);
    } finally {
      setSaving(false);
    }
  };

  // Reset forms
  const resetTeamForm = () => {
    setTeamForm({ team_key: '', team_name: '', description: '', team_type: 'functional' });
  };

  const resetMemberForm = () => {
    setMemberForm({ user_id: '', role_in_team: 'member' });
  };

  // Filter teams
  const filteredTeams = useMemo(() => {
    return teams.filter(team =>
      team.team_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      team.team_key?.toLowerCase().includes(searchTerm.toLowerCase())
    );
  }, [teams, searchTerm]);

  // Load data on mount
  useEffect(() => {
    fetchTeams();
    fetchUsers();
  }, [tenant.id]);

  // Fetch members when team selection changes
  useEffect(() => {
    if (selectedTeam) {
      fetchTeamMembers(selectedTeam.id);
    } else {
      setTeamMembers([]);
    }
  }, [selectedTeam]);

  const handleMenuClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => setAnchorEl(null);

  const handleMemberMenuClick = (event: React.MouseEvent<HTMLElement>, member: TeamMember) => {
    setMemberMenuAnchor(event.currentTarget);
    setSelectedMember(member);
  };

  const handleMemberMenuClose = () => {
    setMemberMenuAnchor(null);
    setSelectedMember(null);
  };

  const getTeamTypeConfig = (type: string) => {
    const configs = {
      functional: { icon: <BusinessCenterIcon />, color: 'primary' as const },
      project: { icon: <FolderSharedIcon />, color: 'success' as const },
      cross_functional: { icon: <HubIcon />, color: 'secondary' as const },
    };
    return configs[type as keyof typeof configs] || { icon: <GroupsIcon />, color: 'default' as const };
  };

  const getRoleConfig = (role: string) => {
    return roleConfig[role as keyof typeof roleConfig] || roleConfig.member;
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 64px)', bgcolor: 'background.default' }}>
      
      {/* Left Sidebar: Team List */}
      <Paper 
        elevation={0} 
        sx={{ width: 320, borderRight: `1px solid ${theme.palette.divider}`, display: 'flex', flexDirection: 'column' }}
      >
        <Box sx={{ p: 2, borderBottom: `1px solid ${theme.palette.divider}` }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
            <Typography variant="h6" fontWeight={700}>Teams</Typography>
            <IconButton 
              size="small" 
              color="primary" 
              onClick={() => { resetTeamForm(); setOpenCreateModal(true); }}
              sx={{ bgcolor: alpha(theme.palette.primary.main, 0.1) }}
            >
              <AddIcon fontSize="small" />
            </IconButton>
          </Stack>
          
          <TextField
            fullWidth
            size="small"
            placeholder="Search teams..."
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
          {filteredTeams.length === 0 ? (
            <Box sx={{ textAlign: 'center', mt: 4, p: 2 }}>
              <Typography variant="body2" color="text.secondary">
                No teams found.
              </Typography>
            </Box>
          ) : (
            <List dense>
              {filteredTeams.map((team) => {
                const typeConfig = getTeamTypeConfig(team.team_type);
                return (
                  <ListItemButton
                    key={team.id}
                    selected={selectedTeam?.id === team.id}
                    onClick={() => setSelectedTeam(team)}
                    sx={{ 
                      py: 1.5, 
                      borderLeft: selectedTeam?.id === team.id ? `3px solid ${theme.palette.primary.main}` : '3px solid transparent',
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
                        {typeConfig.icon}
                      </Avatar>
                    </ListItemIcon>
                    <ListItemText 
                      primary={
                        <Typography 
                          variant="body2" 
                          fontWeight={600} 
                          sx={{ textDecoration: team.is_active === false ? 'line-through' : 'none', color: team.is_active === false ? 'text.secondary' : 'text.primary' }}
                        >
                          {team.team_name}
                        </Typography>
                      } 
                      secondary={
                        <Stack direction="row" spacing={1} alignItems="center" component="span">
                          <Typography variant="caption" color="text.secondary">
                            {team.team_type?.replace('_', ' ')}
                          </Typography>
                          {team.source === 'ad' && <SyncIcon sx={{ fontSize: 12, color: 'text.secondary' }} />}
                        </Stack>
                      } 
                    />
                    <IconButton
                      size="small"
                      color="error"
                      title="Delete Team"
                      onClick={(e) => {
                        e.stopPropagation();
                        deleteTeam(team.id);
                      }}
                      sx={{ opacity: 0.7, '&:hover': { opacity: 1, bgcolor: alpha(theme.palette.error.main, 0.1) } }}
                    >
                      <DeleteOutlineIcon fontSize="small" />
                    </IconButton>
                  </ListItemButton>
                );
              })}
            </List>
          )}
        </Box>
      </Paper>

      {/* Right Pane: Detail View */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {!selectedTeam ? (
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
            <Avatar sx={{ width: 80, height: 80, mb: 3, bgcolor: alpha(theme.palette.primary.main, 0.1), color: 'primary.main' }}>
              <GroupsIcon sx={{ fontSize: 40 }} />
            </Avatar>
            <Typography variant="h5" fontWeight={700} gutterBottom>No Team Selected</Typography>
            <Typography variant="body1" color="text.secondary" align="center" sx={{ maxWidth: 400 }}>
              Select a team from the left to view its members, manage resources, or create a new team.
            </Typography>
          </Box>
        ) : (
          <>
            {/* Header */}
            <Box sx={{ p: 3, borderBottom: `1px solid ${theme.palette.divider}` }}>
              <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                <Box>
                  <Stack direction="row" spacing={1.5} alignItems="center">
                    <Avatar 
                      sx={{ 
                        width: 40, 
                        height: 40, 
                        bgcolor: alpha(theme.palette.primary.main, 0.1), 
                        color: 'primary.main' 
                      }}
                    >
                      {getTeamTypeConfig(selectedTeam.team_type).icon}
                    </Avatar>
                    <Box>
                      <Typography variant="h5" fontWeight={700} sx={{ textDecoration: selectedTeam.is_active === false ? 'line-through' : 'none' }}>
                        {selectedTeam.team_name}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">{selectedTeam.team_key}</Typography>
                    </Box>
                    <Chip 
                      label={selectedTeam.is_active === false ? 'Inactive' : 'Active'} 
                      size="small" 
                      color={selectedTeam.is_active === false ? 'error' : 'success'} 
                      variant="outlined"
                    />
                    <Chip 
                      label={selectedTeam.source === 'ad' ? 'AD Synced' : 'Local'} 
                      size="small" 
                      color={selectedTeam.source === 'ad' ? 'secondary' : 'default'}
                      variant="outlined"
                      icon={selectedTeam.source === 'ad' ? <SyncIcon fontSize="small" /> : undefined}
                    />
                  </Stack>
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                    {selectedTeam.description}
                  </Typography>
                </Box>
                <Stack direction="row" spacing={1} alignItems="center">
                  <Button
                    size="small"
                    variant="outlined"
                    startIcon={<EditIcon fontSize="small" />}
                    onClick={handleOpenEditModal}
                    sx={{ textTransform: 'none', borderRadius: 2 }}
                  >
                    Edit
                  </Button>
                  {selectedTeam.source === 'local' && (
                    <Button
                      size="small"
                      variant="outlined"
                      color="error"
                      startIcon={<DeleteOutlineIcon fontSize="small" />}
                      onClick={() => deleteTeam(selectedTeam.id)}
                      sx={{ textTransform: 'none', borderRadius: 2 }}
                    >
                      Delete
                    </Button>
                  )}
                  <IconButton onClick={handleMenuClick}>
                    <MoreVertIcon />
                  </IconButton>
                </Stack>
                <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={handleMenuClose}>
                  <MenuItem onClick={() => { handleMenuClose(); handleOpenEditModal(); }}>
                    <Stack direction="row" spacing={1.5} alignItems="center">
                      <EditIcon fontSize="small" />
                      <Typography variant="body2">Edit Details & Status</Typography>
                    </Stack>
                  </MenuItem>
                  {selectedTeam.source === 'local' && (
                    <MenuItem onClick={() => { handleMenuClose(); deleteTeam(selectedTeam.id); }} sx={{ color: 'error.main' }}>
                      <Stack direction="row" spacing={1.5} alignItems="center">
                        <DeleteOutlineIcon fontSize="small" />
                        <Typography variant="body2">Delete Team</Typography>
                      </Stack>
                    </MenuItem>
                  )}
                </Menu>
              </Stack>
            </Box>

            {/* Tabs */}
            <Tabs 
              value={tabIndex} 
              onChange={(e, val) => setTabIndex(val)}
              sx={{ px: 3, borderBottom: `1px solid ${theme.palette.divider}` }}
            >
              <Tab label="Members" />
              <Tab label="Role Mappings" />
              <Tab label="Resources" />
            </Tabs>

            {/* Tab Content */}
            <Box sx={{ flex: 1, p: 3, overflowY: 'auto' }}>
              
              {/* TAB 1: Members */}
              {tabIndex === 0 && (
                <Box>
                  {selectedTeam.source === 'ad' && (
                    <Alert severity="info" sx={{ mb: 3, borderRadius: 2 }}>
                      <AlertTitle>Active Directory Synced</AlertTitle>
                      Memberships for this team are automatically synced from your identity provider and are read-only.
                    </Alert>
                  )}
                  
                  <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="subtitle1" fontWeight={600}>Team Members ({teamMembers.length})</Typography>
                    {selectedTeam.source === 'local' && (
                      <Button 
                        variant="contained" 
                        startIcon={<PersonAddIcon />} 
                        disableElevation 
                        sx={{ borderRadius: 2 }}
                        onClick={() => { resetMemberForm(); setOpenAddMemberModal(true); }}
                      >
                        Add Member
                      </Button>
                    )}
                  </Stack>

                  {teamMembers.length === 0 ? (
                    <Paper variant="outlined" sx={{ borderRadius: 2, p: 4, textAlign: 'center' }}>
                      <GroupsIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                      <Typography variant="body2" color="text.secondary">
                        No members in this team yet.
                      </Typography>
                    </Paper>
                  ) : (
                    <Paper variant="outlined" sx={{ borderRadius: 2 }}>
                      <List dense>
                        {teamMembers.map((member, index) => {
                          const role = getRoleConfig(member.role_in_team);
                          return (
                            <React.Fragment key={member.id}>
                              <ListItemButton sx={{ py: 1.5 }}>
                                <ListItemAvatar>
                                  <Avatar sx={{ width: 36, height: 36 }}>
                                    {member.user_name?.charAt(0).toUpperCase()}
                                  </Avatar>
                                </ListItemAvatar>
                                <ListItemText 
                                  primary={member.user_name} 
                                  secondary={member.user_email} 
                                  primaryTypographyProps={{ variant: 'body2', fontWeight: 500 }}
                                  secondaryTypographyProps={{ variant: 'caption' }}
                                />
                                <Chip 
                                  icon={role.icon}
                                  label={member.role_in_team} 
                                  size="small" 
                                  color={role.color}
                                  variant="outlined"
                                  sx={{ mr: 1 }}
                                />
                                {selectedTeam.source === 'local' && (
                                  <IconButton 
                                    size="small" 
                                    onClick={(e) => handleMemberMenuClick(e, member)}
                                  >
                                    <MoreVertIcon fontSize="small" />
                                  </IconButton>
                                )}
                              </ListItemButton>
                              {index < teamMembers.length - 1 && <Divider />}
                            </React.Fragment>
                          );
                        })}
                      </List>
                    </Paper>
                  )}
                </Box>
              )}

              {/* TAB 2: Role Mappings */}
              {tabIndex === 1 && (
                <Box>
                  <Stack direction="row" justifyContent="flex-end" mb={2}>
                    <Button variant="contained" startIcon={<AddIcon />} disableElevation sx={{ borderRadius: 2 }}>
                      Map Role
                    </Button>
                  </Stack>
                  <Paper variant="outlined" sx={{ borderRadius: 2, p: 4, textAlign: 'center' }}>
                    <LockIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                    <Typography variant="body2" color="text.secondary" gutterBottom>
                      No roles mapped to this team yet.
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Mapping a role grants all team members the permissions of that role.
                    </Typography>
                  </Paper>
                </Box>
              )}

              {/* TAB 3: Resources (ABAC) */}
              {tabIndex === 2 && (
                <Box>
                  <Paper 
                    variant="outlined" 
                    sx={{ borderRadius: 2, p: 4, textAlign: 'center', bgcolor: alpha(theme.palette.text.primary, 0.02) }}
                  >
                    <GroupsIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                    <Typography variant="body2" color="text.secondary" gutterBottom>
                      No resource policies assigned.
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Resource mapping (ABAC) will be available here once fully implemented.
                    </Typography>
                  </Paper>
                </Box>
              )}
            </Box>
          </>
        )}
      </Box>

      {/* Member Actions Menu */}
      <Menu
        anchorEl={memberMenuAnchor}
        open={Boolean(memberMenuAnchor)}
        onClose={handleMemberMenuClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        {selectedMember && selectedMember.role_in_team !== 'admin' && (
          <MenuItem
            onClick={() => {
              if (selectedMember) {
                promoteToManager(selectedMember.id);
              }
              handleMemberMenuClose();
            }}
          >
            <Stack direction="row" spacing={1.5} alignItems="center">
              <CrownIcon fontSize="small" color="warning" />
              <Typography variant="body2">Make Team Manager</Typography>
            </Stack>
          </MenuItem>
        )}
        <MenuItem
          onClick={() => {
            if (selectedMember) {
              removeTeamMember(selectedMember.id);
            }
            handleMemberMenuClose();
          }}
          sx={{ color: 'error.main' }}
        >
          <Stack direction="row" spacing={1.5} alignItems="center">
            <PersonRemoveIcon fontSize="small" />
            <Typography variant="body2">Remove from Team</Typography>
          </Stack>
        </MenuItem>
      </Menu>

      {/* Create Team Modal */}
      <Dialog 
        open={openCreateModal} 
        onClose={() => setOpenCreateModal(false)} 
        maxWidth="sm" 
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Create New Team</Typography>
              <Typography variant="body2" color="text.secondary">Organize users into teams for resource mapping.</Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenCreateModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Stack spacing={2.5}>
            <TextField
              autoFocus
              label="Team Key"
              placeholder="e.g., na_sales_team"
              fullWidth
              variant="outlined"
              value={teamForm.team_key}
              onChange={(e) => setTeamForm({ ...teamForm, team_key: e.target.value })}
              helperText="Unique identifier used by the system."
            />
            <TextField
              label="Team Name"
              placeholder="e.g., North America Sales Team"
              fullWidth
              variant="outlined"
              value={teamForm.team_name}
              onChange={(e) => setTeamForm({ ...teamForm, team_name: e.target.value })}
            />
            <TextField
              label="Description"
              placeholder="Brief description of the team's purpose"
              fullWidth
              multiline
              rows={3}
              variant="outlined"
              value={teamForm.description}
              onChange={(e) => setTeamForm({ ...teamForm, description: e.target.value })}
            />
            <TextField
              select
              label="Team Type"
              fullWidth
              value={teamForm.team_type}
              onChange={(e) => setTeamForm({ 
                ...teamForm, 
                team_type: e.target.value as 'functional' | 'project' | 'cross_functional' 
              })}
              variant="outlined"
            >
              {teamTypes.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  <Stack direction="row" spacing={1.5} alignItems="center">
                    {option.icon}
                    <Typography variant="body2">{option.label}</Typography>
                  </Stack>
                </MenuItem>
              ))}
            </TextField>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setOpenCreateModal(false)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button 
            onClick={createTeam} 
            variant="contained" 
            color="primary" 
            disableElevation 
            sx={{ borderRadius: 2, px: 3 }}
            disabled={!teamForm.team_key || !teamForm.team_name || saving}
          >
            {saving ? 'Creating...' : 'Create Team'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Team Modal */}
      <Dialog 
        open={openEditModal} 
        onClose={() => setOpenEditModal(false)} 
        maxWidth="sm" 
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Edit Team Details & Status</Typography>
              <Typography variant="body2" color="text.secondary">
                Update team configuration and active status
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenEditModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Stack spacing={2.5} sx={{ mt: 1 }}>
            <TextField
              label="Team Name"
              fullWidth
              value={editTeamForm.team_name}
              onChange={(e) => setEditTeamForm({ ...editTeamForm, team_name: e.target.value })}
              variant="outlined"
            />
            <TextField
              label="Team Key"
              fullWidth
              value={editTeamForm.team_key}
              onChange={(e) => setEditTeamForm({ ...editTeamForm, team_key: e.target.value })}
              variant="outlined"
            />
            <TextField
              label="Description"
              fullWidth
              multiline
              rows={3}
              value={editTeamForm.description}
              onChange={(e) => setEditTeamForm({ ...editTeamForm, description: e.target.value })}
            />
            <TextField
              select
              label="Team Type"
              fullWidth
              value={editTeamForm.team_type}
              onChange={(e) => setEditTeamForm({ 
                ...editTeamForm, 
                team_type: e.target.value as 'functional' | 'project' | 'cross_functional' 
              })}
              variant="outlined"
            >
              {teamTypes.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  <Stack direction="row" spacing={1.5} alignItems="center">
                    {option.icon}
                    <Typography variant="body2">{option.label}</Typography>
                  </Stack>
                </MenuItem>
              ))}
            </TextField>

            <TextField
              select
              label="Team Manager"
              fullWidth
              value={editTeamForm.manager_user_id}
              onChange={(e) => setEditTeamForm({ ...editTeamForm, manager_user_id: e.target.value })}
              variant="outlined"
            >
              <MenuItem value="">
                <em>No Manager Assigned</em>
              </MenuItem>
              {users.map((user) => (
                <MenuItem key={user.id} value={user.id}>
                  {user.full_name} ({user.email})
                </MenuItem>
              ))}
            </TextField>

            <FormControlLabel
              control={
                <Switch
                  checked={editTeamForm.is_active}
                  onChange={(e) => setEditTeamForm({ ...editTeamForm, is_active: e.target.checked })}
                  color="primary"
                />
              }
              label={
                <Typography variant="body2" fontWeight={600}>
                  Team Status: {editTeamForm.is_active ? 'Active' : 'Inactive'}
                </Typography>
              }
            />
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setOpenEditModal(false)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button 
            onClick={updateTeam} 
            variant="contained" 
            color="primary" 
            disableElevation 
            sx={{ borderRadius: 2, px: 3 }}
            disabled={!editTeamForm.team_key || !editTeamForm.team_name || saving}
          >
            {saving ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Add Member Modal */}
      <Dialog 
        open={openAddMemberModal} 
        onClose={() => setOpenAddMemberModal(false)} 
        maxWidth="xs" 
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ pb: 1, pt: 3, px: 3 }}>
          <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" fontWeight={700}>Add Team Member</Typography>
              <Typography variant="body2" color="text.secondary">
                Add a user to {selectedTeam?.team_name}
              </Typography>
            </Box>
            <IconButton size="small" onClick={() => setOpenAddMemberModal(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent sx={{ px: 3, py: 2 }}>
          <Stack spacing={2.5}>
            <TextField
              select
              label="User"
              fullWidth
              value={memberForm.user_id}
              onChange={(e) => setMemberForm({ ...memberForm, user_id: e.target.value })}
              variant="outlined"
            >
              <MenuItem value="" disabled>
                <Typography variant="body2" color="text.secondary">Select a user...</Typography>
              </MenuItem>
              {users
                .filter(user => !teamMembers.some(m => m.user_id === user.id))
                .map(user => (
                  <MenuItem key={user.id} value={user.id}>
                    <Stack direction="row" spacing={1.5} alignItems="center">
                      <Avatar sx={{ width: 24, height: 24, fontSize: '0.75rem' }}>
                        {user.full_name?.charAt(0).toUpperCase()}
                      </Avatar>
                      <Typography variant="body2">{user.full_name}</Typography>
                      <Typography variant="caption" color="text.secondary">({user.email})</Typography>
                    </Stack>
                  </MenuItem>
                ))}
            </TextField>
            <TextField
              select
              label="Role in Team"
              fullWidth
              value={memberForm.role_in_team}
              onChange={(e) => setMemberForm({ 
                ...memberForm, 
                role_in_team: e.target.value as 'member' | 'lead' | 'admin' 
              })}
              variant="outlined"
            >
              <MenuItem value="member">
                <Stack direction="row" spacing={1} alignItems="center">
                  <BusinessCenterIcon fontSize="small" color="primary" />
                  <Typography variant="body2">Member</Typography>
                </Stack>
              </MenuItem>
              <MenuItem value="lead">
                <Stack direction="row" spacing={1} alignItems="center">
                  <ShieldIcon fontSize="small" color="warning" />
                  <Typography variant="body2">Team Lead</Typography>
                </Stack>
              </MenuItem>
              <MenuItem value="admin">
                <Stack direction="row" spacing={1} alignItems="center">
                  <CrownIcon fontSize="small" color="error" />
                  <Typography variant="body2">Team Admin</Typography>
                </Stack>
              </MenuItem>
            </TextField>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3, pt: 1 }}>
          <Button onClick={() => setOpenAddMemberModal(false)} color="inherit" sx={{ borderRadius: 2, px: 2 }}>
            Cancel
          </Button>
          <Button 
            onClick={addTeamMember} 
            variant="contained" 
            color="primary" 
            disableElevation 
            sx={{ borderRadius: 2, px: 3 }}
            disabled={!memberForm.user_id || saving}
          >
            {saving ? 'Adding...' : 'Add Member'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
