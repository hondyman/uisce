import React, { useState, useMemo } from 'react';
import { Search, ChevronDown, Eye, Edit2, Trash2, Filter } from 'lucide-react';

interface Role {
  id: string;
  name: string;
  key: string;
  level: string;
  description: string;
  permissionCount: number;
  userCount: number;
  createdAt: string;
}

interface RoleSummaryProps {
  roles?: Role[];
  onCreateRole?: () => void;
  onViewRole?: (role: Role) => void;
  onEditRole?: (role: Role) => void;
  onDeleteRole?: (role: Role) => void;
}

const LEVEL_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  system: { bg: 'bg-red-50', text: 'text-red-700', border: 'border-red-200' },
  organization: { bg: 'bg-blue-50', text: 'text-blue-700', border: 'border-blue-200' },
  team: { bg: 'bg-green-50', text: 'text-green-700', border: 'border-green-200' },
  project: { bg: 'bg-purple-50', text: 'text-purple-700', border: 'border-purple-200' },
  external: { bg: 'bg-amber-50', text: 'text-amber-700', border: 'border-amber-200' },
};

const DEFAULT_ROLES: Role[] = [
  {
    id: '1',
    name: 'Administrator',
    key: 'admin',
    level: 'system',
    description: 'Full access to all features and data.',
    permissionCount: 24,
    userCount: 3,
    createdAt: '2024-01-15',
  },
  {
    id: '2',
    name: 'Manager',
    key: 'manager',
    level: 'organization',
    description: 'Manage users, roles, and basic settings.',
    permissionCount: 18,
    userCount: 12,
    createdAt: '2024-01-20',
  },
  {
    id: '3',
    name: 'Editor',
    key: 'editor',
    level: 'team',
    description: 'Create and edit content within assigned teams.',
    permissionCount: 12,
    userCount: 45,
    createdAt: '2024-02-01',
  },
  {
    id: '4',
    name: 'Viewer',
    key: 'viewer',
    level: 'project',
    description: 'View project details and progress.',
    permissionCount: 5,
    userCount: 120,
    createdAt: '2024-02-05',
  },
  {
    id: '5',
    name: 'Guest',
    key: 'guest',
    level: 'external',
    description: 'Limited access to public information.',
    permissionCount: 2,
    userCount: 200,
    createdAt: '2024-02-10',
  },
];

export const RoleSummary: React.FC<RoleSummaryProps> = ({
  roles = DEFAULT_ROLES,
  onCreateRole,
  onViewRole,
  onEditRole,
  onDeleteRole,
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedLevel, setSelectedLevel] = useState<string>('');
  const [sortBy, setSortBy] = useState<'name' | 'level' | 'users'>('name');

  const filteredAndSortedRoles = useMemo(() => {
    let filtered = roles.filter(role => {
      const matchesSearch = role.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           role.key.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           role.description.toLowerCase().includes(searchTerm.toLowerCase());
      const matchesLevel = !selectedLevel || role.level === selectedLevel;
      return matchesSearch && matchesLevel;
    });

    return filtered.sort((a, b) => {
      switch (sortBy) {
        case 'level':
          return a.level.localeCompare(b.level);
        case 'users':
          return b.userCount - a.userCount;
        case 'name':
        default:
          return a.name.localeCompare(b.name);
      }
    });
  }, [roles, searchTerm, selectedLevel, sortBy]);

  const uniqueLevels = Array.from(new Set(roles.map(r => r.level)));

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <header className="flex items-center justify-between whitespace-nowrap border-b border-[#e7ecf4] px-10 py-3 bg-white">
        <div className="flex items-center gap-4">
          <div className="w-8 h-8 bg-gradient-to-br from-[#3d87f5] to-[#2d6fd4] rounded-lg flex items-center justify-center">
            <span className="text-white font-bold text-lg">🔐</span>
          </div>
          <h1 className="text-lg font-bold text-[#0d131c]">CloudRBAC</h1>
        </div>
        <div className="flex items-center gap-9">
          <a href="#" className="text-[#0d131c] text-sm font-medium hover:text-[#3d87f5] transition-colors">Dashboard</a>
          <a href="#" className="text-[#0d131c] text-sm font-medium hover:text-[#3d87f5] transition-colors">Roles</a>
          <a href="#" className="text-[#0d131c] text-sm font-medium hover:text-[#3d87f5] transition-colors">Users</a>
          <a href="#" className="text-[#0d131c] text-sm font-medium hover:text-[#3d87f5] transition-colors">Groups</a>
        </div>
        <div className="w-10 h-10 bg-gradient-to-br from-blue-400 to-blue-600 rounded-full" />
      </header>

      {/* Main Content */}
      <div className="px-10 py-8">
        <div className="max-w-7xl mx-auto">
          {/* Title and Create Button */}
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-3xl font-bold text-[#0d131c]">Roles</h2>
            <button
              onClick={onCreateRole}
              className="px-4 py-2 bg-[#3d87f5] text-white text-sm font-semibold rounded-lg hover:bg-[#2d6fd4] transition-colors shadow-sm"
            >
              + Create Role
            </button>
          </div>

          {/* Filters and Search */}
          <div className="bg-white rounded-lg border border-[#e7ecf4] mb-6 p-4">
            <div className="space-y-4">
              {/* Search Bar */}
              <div className="flex gap-4">
                <div className="flex-1 relative">
                  <Search className="absolute left-3 top-3 w-5 h-5 text-[#496a9c]" />
                  <input
                    type="text"
                    placeholder="Search roles by name, key, or description..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="w-full pl-10 pr-4 py-2 border border-[#ced9e8] rounded-lg bg-slate-50 focus:outline-none focus:ring-2 focus:ring-[#3d87f5] text-sm"
                  />
                </div>
              </div>

              {/* Filter Controls */}
              <div className="flex gap-3 flex-wrap">
                <div className="flex items-center gap-2">
                  <Filter className="w-4 h-4 text-[#496a9c]" />
                  <select
                    value={selectedLevel}
                    onChange={(e) => setSelectedLevel(e.target.value)}
                    className="px-3 py-2 border border-[#ced9e8] rounded-lg bg-white text-sm focus:outline-none focus:ring-2 focus:ring-[#3d87f5]"
                  >
                    <option value="">All Levels</option>
                    {uniqueLevels.map(level => (
                      <option key={level} value={level}>
                        {level.charAt(0).toUpperCase() + level.slice(1)}
                      </option>
                    ))}
                  </select>
                </div>

                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as 'name' | 'level' | 'users')}
                  className="px-3 py-2 border border-[#ced9e8] rounded-lg bg-white text-sm focus:outline-none focus:ring-2 focus:ring-[#3d87f5] ml-auto"
                >
                  <option value="name">Sort by Name</option>
                  <option value="level">Sort by Level</option>
                  <option value="users">Sort by Users</option>
                </select>
              </div>
            </div>
          </div>

          {/* Results Count */}
          <div className="mb-4 text-sm text-[#496a9c]">
            Showing {filteredAndSortedRoles.length} of {roles.length} roles
          </div>

          {/* Roles Table/Grid */}
          {filteredAndSortedRoles.length > 0 ? (
            <div className="bg-white rounded-lg border border-[#e7ecf4] overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-[#e7ecf4] bg-slate-50">
                      <th className="px-6 py-4 text-left text-xs font-semibold text-[#0d131c] uppercase tracking-wider">Role</th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-[#0d131c] uppercase tracking-wider">Key</th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-[#0d131c] uppercase tracking-wider">Level</th>
                      <th className="px-6 py-4 text-left text-xs font-semibold text-[#0d131c] uppercase tracking-wider">Description</th>
                      <th className="px-6 py-4 text-center text-xs font-semibold text-[#0d131c] uppercase tracking-wider">Permissions</th>
                      <th className="px-6 py-4 text-center text-xs font-semibold text-[#0d131c] uppercase tracking-wider">Users</th>
                      <th className="px-6 py-4 text-center text-xs font-semibold text-[#0d131c] uppercase tracking-wider">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredAndSortedRoles.map((role) => {
                      const levelColor = LEVEL_COLORS[role.level] || LEVEL_COLORS.project;
                      return (
                        <tr key={role.id} className="border-b border-[#e7ecf4] hover:bg-slate-50 transition-colors">
                          <td className="px-6 py-4">
                            <div className="font-medium text-[#0d131c]">{role.name}</div>
                            <div className="text-xs text-[#496a9c]">Created {role.createdAt}</div>
                          </td>
                          <td className="px-6 py-4">
                            <code className="text-sm bg-slate-100 px-2 py-1 rounded text-[#0d131c] font-mono">
                              {role.key}
                            </code>
                          </td>
                          <td className="px-6 py-4">
                            <span
                              className={`inline-block px-3 py-1 rounded-full text-xs font-semibold ${levelColor.bg} ${levelColor.text} border ${levelColor.border}`}
                            >
                              {role.level.charAt(0).toUpperCase() + role.level.slice(1)}
                            </span>
                          </td>
                          <td className="px-6 py-4 max-w-xs">
                            <p className="text-sm text-[#496a9c] truncate">{role.description}</p>
                          </td>
                          <td className="px-6 py-4 text-center">
                            <span className="inline-block bg-blue-50 text-blue-700 px-3 py-1 rounded-full text-sm font-medium">
                              {role.permissionCount}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-center">
                            <span className="text-sm font-medium text-[#0d131c]">{role.userCount}</span>
                          </td>
                          <td className="px-6 py-4">
                            <div className="flex items-center justify-center gap-2">
                              {/* View Button */}
                              <button
                                onClick={() => onViewRole?.(role)}
                                className="p-2 hover:bg-blue-50 rounded-lg transition-colors group relative"
                                title="View role details"
                              >
                                <Eye className="w-4 h-4 text-[#3d87f5]" />
                                <span className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-2 py-1 bg-slate-900 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none">
                                  View
                                </span>
                              </button>

                              {/* Edit Button */}
                              <button
                                onClick={() => onEditRole?.(role)}
                                className="p-2 hover:bg-green-50 rounded-lg transition-colors group relative"
                                title="Edit role"
                              >
                                <Edit2 className="w-4 h-4 text-[#16a34a]" />
                                <span className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-2 py-1 bg-slate-900 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none">
                                  Edit
                                </span>
                              </button>

                              {/* Delete Button */}
                              <button
                                onClick={() => onDeleteRole?.(role)}
                                className="p-2 hover:bg-red-50 rounded-lg transition-colors group relative"
                                title="Delete role"
                              >
                                <Trash2 className="w-4 h-4 text-[#dc2626]" />
                                <span className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-2 py-1 bg-slate-900 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none">
                                  Delete
                                </span>
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <div className="text-center py-12 bg-white rounded-lg border border-[#e7ecf4]">
              <p className="text-[#496a9c] text-sm">No roles found matching your search criteria</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default RoleSummary;
