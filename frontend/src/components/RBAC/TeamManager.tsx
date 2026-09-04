/**
 * Team Manager - Enterprise Team & Department Management
 *
 * Features:
 * - Team creation and configuration
 * - Member roster management
 * - Role-in-team assignments
 * - Team permission configuration
 * - Cross-functional team support
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Typography,
  Button,
  TextField,
  InputAdornment,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Chip,
  Avatar,
  Paper,
  IconButton,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  ListItemSecondaryAction,
  CircularProgress,
  Tooltip,
  Stack,
  Divider,
} from '@mui/material';
import {
  Group as GroupIcon,
  Add as AddIcon,
  Close as CloseIcon,
  Save as SaveIcon,
  Delete as DeleteIcon,
  PersonAdd as PersonAddIcon,
  Search as SearchIcon,
  FilterList as FilterIcon,
  Security as SecurityIcon,
  TrackChanges as TargetIcon,
  Hub as NetworkIcon,
  WorkspacePremium as CrownIcon,
  CheckCircle as CheckCircleIcon,
  PersonRemove as PersonRemoveIcon,
} from '@mui/icons-material';
import apiClient from '../../utils/apiClient';

interface Team {
  id: string;
  team_key: string;
  team_name: string;
  description: string;
  team_type: 'functional' | 'project' | 'cross_functional';
  manager_user_id: string;
  manager_name?: string;
  member_count?: number;
  created_at: string;
}

interface TeamMember {
  id: string;
  team_id: string;
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
  department: string;
}

interface TeamManagerProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

export const TeamManager: React.FC<TeamManagerProps> = ({ tenant, datasource }) => {
  const [teams, setTeams] = useState<Team[]>([]);
  const [selectedTeam, setSelectedTeam] = useState<Team | null>(null);
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showAddMemberModal, setShowAddMemberModal] = useState(false);
  const [saving, setSaving] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [typeFilter, setTypeFilter] = useState<string>('all');

  const [teamForm, setTeamForm] = useState({
    team_key: '',
    team_name: '',
    description: '',
    team_type: 'functional' as 'functional' | 'project' | 'cross_functional',
    manager_user_id: '',
  });

  const [memberForm, setMemberForm] = useState({
    user_id: '',
    role_in_team: 'member' as 'member' | 'lead' | 'admin',
  });

  const fetchTeams = async () => {
    try {
      setLoading(true);
      const data = await apiClient<Team[]>(`/api/rbac/teams?tenant_id=${tenant.id}`);
      setTeams(data || []);
    } catch (error) {
      console.error('Failed to fetch teams:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchTeamMembers = async (teamId: string) => {
    try {
      const data = await apiClient<TeamMember[]>(`/api/rbac/teams/${teamId}/members?tenant_id=${tenant.id}`);
      setTeamMembers(data || []);
    } catch (error) {
      console.error('Failed to fetch team members:', error);
    }
  };

  const fetchUsers = async () => {
    try {
      const data = await apiClient<User[]>(`/api/rbac/users?tenant_id=${tenant.id}`);
      setUsers(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch users:', error);
    }
  };

  const createTeam = async () => {
    try {
      setSaving(true);
      await apiClient(`/api/rbac/teams`, {
        method: 'POST',
        body: JSON.stringify({ ...teamForm }),
      });
      await fetchTeams();
      setShowCreateModal(false);
      resetTeamForm();
    } catch (error) {
      console.error('Failed to create team:', error);
    } finally {
      setSaving(false);
    }
  };

  const deleteTeam = async (teamId: string) => {
    if (!confirm('Are you sure you want to delete this team?')) return;

    try {
      setSaving(true);
      await apiClient(`/api/rbac/teams/${teamId}`, { method: 'DELETE' });
      await fetchTeams();
      if (selectedTeam?.id === teamId) {
        setSelectedTeam(null);
        setTeamMembers([]);
      }
    } catch (error) {
      console.error('Failed to delete team:', error);
    } finally {
      setSaving(false);
    }
  };

  const addTeamMember = async () => {
    if (!selectedTeam) return;

    try {
      setSaving(true);
      await apiClient(`/api/rbac/teams/${selectedTeam.id}/members`, {
        method: 'POST',
        body: JSON.stringify({ ...memberForm }),
      });
      await fetchTeamMembers(selectedTeam.id);
      await fetchTeams();
      setShowAddMemberModal(false);
      resetMemberForm();
    } catch (error) {
      console.error('Failed to add team member:', error);
    } finally {
      setSaving(false);
    }
  };

  const removeTeamMember = async (memberId: string) => {
    if (!selectedTeam || !confirm('Remove this member from the team?')) return;

    try {
      setSaving(true);
      await apiClient(`/api/rbac/teams/${selectedTeam.id}/members/${memberId}`, {
        method: 'DELETE',
      });
      await fetchTeamMembers(selectedTeam.id);
      await fetchTeams();
    } catch (error) {
      console.error('Failed to remove team member:', error);
    } finally {
      setSaving(false);
    }
  };

  const resetTeamForm = () => {
    setTeamForm({
      team_key: '',
      team_name: '',
      description: '',
      team_type: 'functional',
      manager_user_id: '',
    });
  };

  const resetMemberForm = () => {
    setMemberForm({
      user_id: '',
      role_in_team: 'member',
    });
  };

  const filteredTeams = useMemo(() => {
    return teams.filter((team) => {
      const searchLower = searchTerm.toLowerCase();
      const matchesSearch =
        (team.team_name ?? '').toLowerCase().includes(searchLower) ||
        (team.team_key ?? '').toLowerCase().includes(searchLower) ||
        (team.description ?? '').toLowerCase().includes(searchLower);
      const matchesType = typeFilter === 'all' || team.team_type === typeFilter;
      return matchesSearch && matchesType;
    });
  }, [teams, searchTerm, typeFilter]);

  useEffect(() => {
    fetchTeams();
    fetchUsers();
  }, [tenant.id]);

  useEffect(() => {
    if (selectedTeam) {
      fetchTeamMembers(selectedTeam.id);
    }
  }, [selectedTeam]);

  const getTeamTypeColor = (type: string): 'primary' | 'success' | 'secondary' => {
    const colors: Record<string, 'primary' | 'success' | 'secondary'> = {
      functional: 'primary',
      project: 'success',
      cross_functional: 'secondary',
    };
    return colors[type] || 'primary';
  };

  const getRoleColor = (role: string): 'error' | 'warning' | 'primary' => {
    const colors: Record<string, 'error' | 'warning' | 'primary'> = {
      admin: 'error',
      lead: 'warning',
      member: 'primary',
    };
    return colors[role] || 'primary';
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Paper sx={{ p: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <GroupIcon sx={{ fontSize: 32, color: 'primary.main' }} />
            <Box>
              <Typography variant="h5" fontWeight={700}>
                Team Manager
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Manage teams and departments for {tenant.display_name}
              </Typography>
            </Box>
          </Box>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => {
              resetTeamForm();
              setShowCreateModal(true);
            }}
          >
            Create Team
          </Button>
        </Box>

        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 3 }}>
          <TextField
            size="small"
            placeholder="Search teams..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
            sx={{ flex: 1 }}
          />
          <FormControl size="small" sx={{ minWidth: 180 }}>
            <InputLabel>Team Type</InputLabel>
            <Select
              value={typeFilter}
              label="Team Type"
              onChange={(e) => setTypeFilter(e.target.value)}
            >
              <MenuItem value="all">All Types</MenuItem>
              <MenuItem value="functional">Functional</MenuItem>
              <MenuItem value="project">Project</MenuItem>
              <MenuItem value="cross_functional">Cross-Functional</MenuItem>
            </Select>
          </FormControl>
        </Stack>

        <Stack direction={{ xs: 'column', lg: 'row' }} spacing={3}>
          <Paper variant="outlined" sx={{ flex: 1, p: 2 }}>
            <Typography variant="h6" fontWeight={600} sx={{ mb: 2 }}>
              Teams ({filteredTeams.length})
            </Typography>
            <List sx={{ maxHeight: 'calc(100vh - 20rem)', overflow: 'auto' }}>
              {filteredTeams.map((team) => (
                <ListItem
                  key={team.id}
                  component="div"
                  onClick={() => setSelectedTeam(team)}
                  selected={selectedTeam?.id === team.id}
                  sx={{
                    mb: 1,
                    borderRadius: 1,
                    border: '1px solid',
                    borderColor: selectedTeam?.id === team.id ? 'primary.main' : 'divider',
                    bgcolor: selectedTeam?.id === team.id ? 'action.selected' : 'background.paper',
                    cursor: 'pointer',
                    '&:hover': { bgcolor: 'action.hover' },
                  }}
                  secondaryAction={
                    <IconButton edge="end" onClick={(e) => { e.stopPropagation(); deleteTeam(team.id); }}>
                      <DeleteIcon color="error" />
                    </IconButton>
                  }
                >
                  <ListItemAvatar>
                    <Avatar sx={{ bgcolor: `${getTeamTypeColor(team.team_type)}.main` }}>
                      {team.team_type === 'project' ? <TargetIcon /> : team.team_type === 'cross_functional' ? <NetworkIcon /> : <GroupIcon />}
                    </Avatar>
                  </ListItemAvatar>
                  <ListItemText
                    primary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography fontWeight={600}>{team.team_name}</Typography>
                        <Chip
                          label={team.team_type.replace('_', ' ').toUpperCase()}
                          size="small"
                          color={getTeamTypeColor(team.team_type)}
                        />
                      </Box>
                    }
                    secondary={
                      <Box sx={{ mt: 0.5 }}>
                        <Typography variant="caption" color="text.secondary">{team.description}</Typography>
                        <Typography variant="caption" display="block" color="text.secondary">
                          {team.member_count || 0} Members
                        </Typography>
                      </Box>
                    }
                  />
                </ListItem>
              ))}
            </List>
          </Paper>

          <Paper variant="outlined" sx={{ flex: 1, p: 2 }}>
            {selectedTeam ? (
              <>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                  <Typography variant="h6" fontWeight={600}>Team Members</Typography>
                  <Button
                    variant="contained"
                    color="success"
                    size="small"
                    startIcon={<PersonAddIcon />}
                    onClick={() => {
                      resetMemberForm();
                      setShowAddMemberModal(true);
                    }}
                  >
                    Add Member
                  </Button>
                </Box>
                <List sx={{ maxHeight: 'calc(100vh - 24rem)', overflow: 'auto' }}>
                  {teamMembers.map((member) => (
                    <ListItem
                      key={member.id}
                      sx={{
                        mb: 1,
                        borderRadius: 1,
                        border: '1px solid',
                        borderColor: 'divider',
                      }}
                      secondaryAction={
                        <IconButton edge="end" onClick={() => removeTeamMember(member.id)}>
                          <PersonRemoveIcon color="error" />
                        </IconButton>
                      }
                    >
                      <ListItemAvatar>
                        <Avatar sx={{ bgcolor: 'primary.main' }}>
                          {member.user_name?.charAt(0).toUpperCase() || '?'}
                        </Avatar>
                      </ListItemAvatar>
                      <ListItemText
                        primary={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Typography fontWeight={500}>{member.user_name}</Typography>
                            <Chip
                              icon={member.role_in_team === 'admin' ? <CrownIcon /> : member.role_in_team === 'lead' ? <SecurityIcon /> : <CheckCircleIcon />}
                              label={member.role_in_team.toUpperCase()}
                              size="small"
                              color={getRoleColor(member.role_in_team)}
                            />
                          </Box>
                        }
                        secondary={
                          <Typography variant="caption" color="text.secondary">
                            {member.user_email} • Joined {new Date(member.joined_at).toLocaleDateString()}
                          </Typography>
                        }
                      />
                    </ListItem>
                  ))}
                  {teamMembers.length === 0 && (
                    <Typography variant="body2" color="text.secondary" sx={{ textAlign: 'center', py: 4 }}>
                      No members in this team
                    </Typography>
                  )}
                </List>
              </>
            ) : (
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: 300 }}>
                <GroupIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
                <Typography color="text.secondary">Select a team to view members</Typography>
              </Box>
            )}
          </Paper>
        </Stack>
      </Paper>

      <Dialog open={showCreateModal} onClose={() => setShowCreateModal(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          Create Team
          <IconButton onClick={() => setShowCreateModal(false)}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <Divider />
        <DialogContent sx={{ pt: 2 }}>
          <Stack spacing={2}>
            <TextField
              label="Team Key *"
              value={teamForm.team_key}
              onChange={(e) => setTeamForm({ ...teamForm, team_key: e.target.value })}
              placeholder="e.g., engineering_team"
              size="small"
              fullWidth
            />
            <TextField
              label="Team Name *"
              value={teamForm.team_name}
              onChange={(e) => setTeamForm({ ...teamForm, team_name: e.target.value })}
              placeholder="e.g., Engineering Team"
              size="small"
              fullWidth
            />
            <TextField
              label="Description"
              value={teamForm.description}
              onChange={(e) => setTeamForm({ ...teamForm, description: e.target.value })}
              placeholder="Team purpose and responsibilities..."
              multiline
              rows={3}
              size="small"
              fullWidth
            />
            <FormControl size="small" fullWidth>
              <InputLabel>Team Type *</InputLabel>
              <Select
                value={teamForm.team_type}
                label="Team Type *"
                onChange={(e) =>
                  setTeamForm({
                    ...teamForm,
                    team_type: e.target.value as 'functional' | 'project' | 'cross_functional',
                  })
                }
              >
                <MenuItem value="functional">Functional (Department-based)</MenuItem>
                <MenuItem value="project">Project (Temporary team)</MenuItem>
                <MenuItem value="cross_functional">Cross-Functional (Multi-department)</MenuItem>
              </Select>
            </FormControl>
            <FormControl size="small" fullWidth>
              <InputLabel>Team Manager *</InputLabel>
              <Select
                value={teamForm.manager_user_id}
                label="Team Manager *"
                onChange={(e) => setTeamForm({ ...teamForm, manager_user_id: e.target.value })}
              >
                <MenuItem value="">Select a manager</MenuItem>
                {users.map((user) => (
                  <MenuItem key={user.id} value={user.id}>
                    {user.full_name} ({user.email})
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2, pt: 1 }}>
          <Button onClick={() => setShowCreateModal(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={createTeam}
            disabled={!teamForm.team_key || !teamForm.team_name || !teamForm.manager_user_id || saving}
            startIcon={<SaveIcon />}
          >
            {saving ? 'Creating...' : 'Create Team'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={showAddMemberModal && selectedTeam !== null} onClose={() => setShowAddMemberModal(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          Add Team Member
          <IconButton onClick={() => setShowAddMemberModal(false)}>
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <Divider />
        <DialogContent sx={{ pt: 2 }}>
          <Stack spacing={2}>
            <FormControl size="small" fullWidth>
              <InputLabel>User *</InputLabel>
              <Select
                value={memberForm.user_id}
                label="User *"
                onChange={(e) => setMemberForm({ ...memberForm, user_id: e.target.value })}
              >
                <MenuItem value="">Select a user</MenuItem>
                {users.map((user) => (
                  <MenuItem key={user.id} value={user.id}>
                    {user.full_name} ({user.email})
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControl size="small" fullWidth>
              <InputLabel>Role in Team *</InputLabel>
              <Select
                value={memberForm.role_in_team}
                label="Role in Team *"
                onChange={(e) =>
                  setMemberForm({
                    ...memberForm,
                    role_in_team: e.target.value as 'member' | 'lead' | 'admin',
                  })
                }
              >
                <MenuItem value="member">Member</MenuItem>
                <MenuItem value="lead">Team Lead</MenuItem>
                <MenuItem value="admin">Team Admin</MenuItem>
              </Select>
            </FormControl>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ p: 2, pt: 1 }}>
          <Button onClick={() => setShowAddMemberModal(false)}>Cancel</Button>
          <Button
            variant="contained"
            color="success"
            onClick={addTeamMember}
            disabled={!memberForm.user_id || saving}
            startIcon={<PersonAddIcon />}
          >
            {saving ? 'Adding...' : 'Add Member'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
