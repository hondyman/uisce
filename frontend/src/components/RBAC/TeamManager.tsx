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
import { Box, Typography } from '@mui/material';
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
} from '@mui/icons-material';
import apiClient from '../../utils/apiClient';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

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

// ============================================================================
// MAIN COMPONENT
// ============================================================================

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

  // Create team form
  const [teamForm, setTeamForm] = useState({
    team_key: '',
    team_name: '',
    description: '',
    team_type: 'functional' as 'functional' | 'project' | 'cross_functional',
    manager_user_id: '',
  });

  // Add member form
  const [memberForm, setMemberForm] = useState({
    user_id: '',
    role_in_team: 'member' as 'member' | 'lead' | 'admin',
  });

  // Fetch teams
  const fetchTeams = async () => {
    try {
      setLoading(true);
      const data = await apiClient<Team[]>('/api/rbac/teams');
      setTeams(data || []);
    } catch (error) {
      console.error('Failed to fetch teams:', error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch team members
  const fetchTeamMembers = async (teamId: string) => {
    try {
      const data = await apiClient<TeamMember[]>(`/api/rbac/teams/${teamId}/members`);
      setTeamMembers(data || []);
    } catch (error) {
      console.error('Failed to fetch team members:', error);
    }
  };

  // Fetch users
  const fetchUsers = async () => {
    try {
      const data = await apiClient<User[]>('/api/users');
      setUsers(data || []);
    } catch (error) {
      console.error('Failed to fetch users:', error);
    }
  };

  // Create team
  const createTeam = async () => {
    try {
      setSaving(true);
      await apiClient(`/api/rbac/teams`, {
        method: 'POST',
        body: JSON.stringify({
          ...teamForm,
        }),
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

  // Delete team
  const deleteTeam = async (teamId: string) => {
    if (!confirm('Are you sure you want to delete this team?')) return;

    try {
      setSaving(true);
      await apiClient(`/api/rbac/teams/${teamId}`, {
        method: 'DELETE',
      });
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

  // Add team member
  const addTeamMember = async () => {
    if (!selectedTeam) return;

    try {
      setSaving(true);
      await apiClient(`/api/rbac/teams/${selectedTeam.id}/members`, {
        method: 'POST',
        body: JSON.stringify({
          ...memberForm,
        }),
      });
      await fetchTeamMembers(selectedTeam.id);
      await fetchTeams(); // Refresh member count
      setShowAddMemberModal(false);
      resetMemberForm();
    } catch (error) {
      console.error('Failed to add team member:', error);
    } finally {
      setSaving(false);
    }
  };

  // Remove team member
  const removeTeamMember = async (memberId: string) => {
    if (!selectedTeam || !confirm('Remove this member from the team?')) return;

    try {
      setSaving(true);
      await apiClient(`/api/rbac/teams/${selectedTeam.id}/members/${memberId}`, {
        method: 'DELETE',
      });
      await fetchTeamMembers(selectedTeam.id);
      await fetchTeams(); // Refresh member count
    } catch (error) {
      console.error('Failed to remove team member:', error);
    } finally {
      setSaving(false);
    }
  };

  // Reset team form
  const resetTeamForm = () => {
    setTeamForm({
      team_key: '',
      team_name: '',
      description: '',
      team_type: 'functional',
      manager_user_id: '',
    });
  };

  // Reset member form
  const resetMemberForm = () => {
    setMemberForm({
      user_id: '',
      role_in_team: 'member',
    });
  };

  // Filter teams
  const filteredTeams = useMemo(() => {
    return teams.filter(team => {
      const matchesSearch =
        team.team_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        team.team_key.toLowerCase().includes(searchTerm.toLowerCase()) ||
        team.description?.toLowerCase().includes(searchTerm.toLowerCase());
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

  const getTeamTypeIcon = (type: string) => {
    const icons: Record<string, React.ReactNode> = {
      functional: <GroupIcon sx={{ width: 20, height: 20 }} />,
      project: <TargetIcon sx={{ width: 20, height: 20 }} />,
      cross_functional: <NetworkIcon sx={{ width: 20, height: 20 }} />,
    };
    return icons[type] || <GroupIcon sx={{ width: 20, height: 20 }} />;
  };

  const getTeamTypeColor = (type: string) => {
    const colors = {
      functional: 'bg-blue-100 text-blue-700 border-blue-300',
      project: 'bg-green-100 text-green-700 border-green-300',
      cross_functional: 'bg-purple-100 text-purple-700 border-purple-300',
    };
    return colors[type as keyof typeof colors] || 'bg-gray-100 text-gray-700 border-gray-300';
  };

  const getRoleIcon = (role: string) => {
    const icons: Record<string, React.ReactNode> = {
      admin: <CrownIcon sx={{ width: 16, height: 16 }} />,
      lead: <SecurityIcon sx={{ width: 16, height: 16 }} />,
      member: <CheckCircleIcon sx={{ width: 16, height: 16 }} />,
    };
    return icons[role] || <CheckCircleIcon sx={{ width: 16, height: 16 }} />;
  };

  const getRoleColor = (role: string) => {
    const colors = {
      admin: 'bg-red-100 text-red-700 border-red-300',
      lead: 'bg-orange-100 text-orange-700 border-orange-300',
      member: 'bg-blue-100 text-blue-700 border-blue-300',
    };
    return colors[role as keyof typeof colors] || 'bg-gray-100 text-gray-700 border-gray-300';
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4 }}>
          <GroupIcon sx={{ width: 48, height: 48, color: 'primary.main', animation: 'pulse 2s infinite' }} />
          <Typography color="text.secondary">Loading teams...</Typography>
        </Box>
      </Box>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 p-6">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 flex items-center gap-3">
              <GroupIcon sx={{ width: 32, height: 32, color: 'primary.main' }} />
              Team Manager
            </h1>
            <p className="text-gray-600 mt-2">
              Manage teams and departments for {tenant.display_name}
            </p>
          </div>
          <button
            onClick={() => {
              resetTeamForm();
              setShowCreateModal(true);
            }}
            className="px-6 py-3 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-all flex items-center gap-2 shadow-lg"
          >
            <AddIcon sx={{ width: 20, height: 20 }} />
            Create Team
          </button>
        </div>

        {/* Search and Filters */}
        <div className="flex gap-4 mt-6">
          <div className="flex-1 relative">
            <SearchIcon sx={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: 'text.secondary', width: 20, height: 20 }} />
            <input
              type="text"
              placeholder="Search teams..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
            />
          </div>
          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
            className="px-4 py-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500 bg-white"
          >
            <option value="all">All Types</option>
            <option value="functional">Functional</option>
            <option value="project">Project</option>
            <option value="cross_functional">Cross-Functional</option>
          </select>
        </div>
      </div>

      {/* Two-Column Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Team List */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          <h2 className="text-xl font-bold text-gray-900 mb-4">
            Teams ({filteredTeams.length})
          </h2>
          <div className="space-y-3 max-h-[calc(100vh-20rem)] overflow-y-auto">
            {filteredTeams.map(team => (
              <div
                key={team.id}
                onClick={() => setSelectedTeam(team)}
                className={`p-4 border-2 rounded-xl cursor-pointer transition-all ${
                  selectedTeam?.id === team.id
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <div className={`p-2 rounded-lg border-2 ${getTeamTypeColor(team.team_type)}`}>
                        {getTeamTypeIcon(team.team_type)}
                      </div>
                      <div>
                        <h3 className="font-bold text-gray-900">{team.team_name}</h3>
                        <p className="text-xs text-gray-500">{team.team_key}</p>
                      </div>
                    </div>
                    <p className="text-sm text-gray-600 mb-3">{team.description}</p>
                    <div className="flex items-center gap-3 flex-wrap">
                      <span className={`px-3 py-1 rounded-full text-xs font-bold border-2 ${getTeamTypeColor(team.team_type)}`}>
                        {team.team_type.replace('_', ' ').toUpperCase()}
                      </span>
                      <span className="px-3 py-1 rounded-full text-xs font-bold bg-gray-100 text-gray-700 border-2 border-gray-300">
                        {team.member_count || 0} Members
                      </span>
                    </div>
                  </div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      deleteTeam(team.id);
                    }}
                    className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-all"
                  >
                    <DeleteIcon sx={{ width: 20, height: 20 }} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Team Details */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          {selectedTeam ? (
            <>
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-xl font-bold text-gray-900">Team Members</h2>
                <button
                  onClick={() => {
                    resetMemberForm();
                    setShowAddMemberModal(true);
                  }}
                  className="px-4 py-2 bg-green-600 text-white rounded-lg font-medium hover:bg-green-700 transition-all flex items-center gap-2"
                >
                  <UserPlus className="w-5 h-5" />
                  Add Member
                </button>
              </div>

              <div className="space-y-3 max-h-[calc(100vh-24rem)] overflow-y-auto">
                {teamMembers.map(member => (
                  <div
                    key={member.id}
                    className="p-4 border-2 border-gray-200 rounded-xl hover:border-gray-300 transition-all"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center text-white font-bold">
                            {member.user_name.charAt(0).toUpperCase()}
                          </div>
                          <div>
                            <h4 className="font-bold text-gray-900">{member.user_name}</h4>
                            <p className="text-xs text-gray-500">{member.user_email}</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className={`px-3 py-1 rounded-full text-xs font-bold border-2 flex items-center gap-2 ${getRoleColor(member.role_in_team)}`}>
                            {getRoleIcon(member.role_in_team)}
                            {member.role_in_team.toUpperCase()}
                          </span>
                          <span className="text-xs text-gray-500">
                            Joined {new Date(member.joined_at).toLocaleDateString()}
                          </span>
                        </div>
                      </div>
                      <button
                        onClick={() => removeTeamMember(member.id)}
                        className="p-2 text-red-600 hover:bg-red-50 rounded-lg transition-all"
                      >
                        <UserMinus className="w-5 h-5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-center">
              <GroupIcon sx={{ width: 64, height: 64, color: 'text.disabled', mb: 2 }} />
              <p className="text-gray-500 text-lg font-medium mb-2">
                No Team Selected
              </p>
              <p className="text-gray-400 text-sm">
                Select a team from the list to view members
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Create Team Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-2xl w-full">
            <div className="p-6 border-b-2 border-gray-200">
              <div className="flex items-center justify-between">
                <h3 className="text-2xl font-bold text-gray-900">Create Team</h3>
                <button
                  onClick={() => setShowCreateModal(false)}
                  className="p-2 hover:bg-gray-100 rounded-lg transition-all"
                >
                  <CloseIcon sx={{ width: 24, height: 24, color: 'text.secondary' }} />
                </button>
              </div>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Team Key *
                </label>
                <input
                  type="text"
                  value={teamForm.team_key}
                  onChange={(e) => setTeamForm({ ...teamForm, team_key: e.target.value })}
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  placeholder="e.g., engineering_team"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Team Name *
                </label>
                <input
                  type="text"
                  value={teamForm.team_name}
                  onChange={(e) => setTeamForm({ ...teamForm, team_name: e.target.value })}
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  placeholder="e.g., Engineering Team"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Description
                </label>
                <textarea
                  value={teamForm.description}
                  onChange={(e) => setTeamForm({ ...teamForm, description: e.target.value })}
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  rows={3}
                  placeholder="Team purpose and responsibilities..."
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Team Type *
                </label>
                <select
                  value={teamForm.team_type}
                  onChange={(e) =>
                    setTeamForm({
                      ...teamForm,
                      team_type: e.target.value as 'functional' | 'project' | 'cross_functional',
                    })
                  }
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                >
                  <option value="functional">Functional (Department-based)</option>
                  <option value="project">Project (Temporary team)</option>
                  <option value="cross_functional">Cross-Functional (Multi-department)</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Team Manager *
                </label>
                <select
                  value={teamForm.manager_user_id}
                  onChange={(e) =>
                    setTeamForm({ ...teamForm, manager_user_id: e.target.value })
                  }
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                >
                  <option value="">Select a manager</option>
                  {users.map(user => (
                    <option key={user.id} value={user.id}>
                      {user.full_name} ({user.email})
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div className="p-6 border-t-2 border-gray-200 flex justify-end gap-3">
              <button
                onClick={() => setShowCreateModal(false)}
                className="px-6 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium hover:bg-gray-200 transition-all"
              >
                Cancel
              </button>
              <button
                onClick={createTeam}
                disabled={!teamForm.team_key || !teamForm.team_name || !teamForm.manager_user_id || saving}
                className="px-6 py-3 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-all flex items-center gap-2 disabled:opacity-50"
              >
                <SaveIcon sx={{ width: 20, height: 20 }} />
                {saving ? 'Creating...' : 'Create Team'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Member Modal */}
      {showAddMemberModal && selectedTeam && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-xl w-full">
            <div className="p-6 border-b-2 border-gray-200">
              <div className="flex items-center justify-between">
                <h3 className="text-2xl font-bold text-gray-900">Add Team Member</h3>
                <button
                  onClick={() => setShowAddMemberModal(false)}
                  className="p-2 hover:bg-gray-100 rounded-lg transition-all"
                >
                  <CloseIcon sx={{ width: 24, height: 24, color: 'text.secondary' }} />
                </button>
              </div>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  User *
                </label>
                <select
                  value={memberForm.user_id}
                  onChange={(e) => setMemberForm({ ...memberForm, user_id: e.target.value })}
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                >
                  <option value="">Select a user</option>
                  {users.map(user => (
                    <option key={user.id} value={user.id}>
                      {user.full_name} ({user.email})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Role in Team *
                </label>
                <select
                  value={memberForm.role_in_team}
                  onChange={(e) =>
                    setMemberForm({
                      ...memberForm,
                      role_in_team: e.target.value as 'member' | 'lead' | 'admin',
                    })
                  }
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                >
                  <option value="member">Member</option>
                  <option value="lead">Team Lead</option>
                  <option value="admin">Team Admin</option>
                </select>
              </div>
            </div>

            <div className="p-6 border-t-2 border-gray-200 flex justify-end gap-3">
              <button
                onClick={() => setShowAddMemberModal(false)}
                className="px-6 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium hover:bg-gray-200 transition-all"
              >
                Cancel
              </button>
              <button
                onClick={addTeamMember}
                disabled={!memberForm.user_id || saving}
                className="px-6 py-3 bg-green-600 text-white rounded-lg font-medium hover:bg-green-700 transition-all flex items-center gap-2 disabled:opacity-50"
              >
                <PersonAddIcon sx={{ width: 20, height: 20 }} />
                {saving ? 'Adding...' : 'Add Member'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
