import React, { useState } from 'react';
import { X, Plus, Trash2 } from 'lucide-react';

interface Permission {
  id: string;
  name: string;
  description: string;
  category: string;
}

interface RoleFormData {
  roleKey: string;
  roleName: string;
  description: string;
  roleLevel: string;
  permissions: string[];
}

const PERMISSION_OPTIONS: Permission[] = [
  { id: 'view_dashboard', name: 'View Dashboard', description: 'Access to dashboard analytics', category: 'Dashboard' },
  { id: 'manage_users', name: 'Manage Users', description: 'Create, update, and delete users', category: 'User Management' },
  { id: 'edit_settings', name: 'Edit Settings', description: 'Modify system and role settings', category: 'Settings' },
  { id: 'create_reports', name: 'Create Reports', description: 'Generate and manage reports', category: 'Reports' },
  { id: 'export_data', name: 'Export Data', description: 'Export system data to files', category: 'Data' },
  { id: 'manage_roles', name: 'Manage Roles', description: 'Create and modify roles', category: 'Role Management' },
  { id: 'view_audit_logs', name: 'View Audit Logs', description: 'Access system audit trail', category: 'Audit' },
  { id: 'manage_permissions', name: 'Manage Permissions', description: 'Configure role permissions', category: 'Permissions' },
];

const ROLE_LEVELS = [
  { value: 'system', label: 'System' },
  { value: 'organization', label: 'Organization' },
  { value: 'team', label: 'Team' },
  { value: 'project', label: 'Project' },
  { value: 'external', label: 'External' },
];

export const RoleEntryForm: React.FC<{
  onClose?: () => void;
  onSave?: (data: RoleFormData) => void;
  initialData?: RoleFormData;
}> = ({ onClose, onSave, initialData }) => {
  const [formData, setFormData] = useState<RoleFormData>(
    initialData || {
      roleKey: '',
      roleName: '',
      description: '',
      roleLevel: '',
      permissions: [],
    }
  );

  const [errors, setErrors] = useState<Record<string, string>>({});

  const validateForm = () => {
    const newErrors: Record<string, string> = {};
    if (!formData.roleKey.trim()) newErrors.roleKey = 'Role key is required';
    if (!formData.roleName.trim()) newErrors.roleName = 'Role name is required';
    if (!formData.roleLevel) newErrors.roleLevel = 'Role level is required';
    if (formData.permissions.length === 0) newErrors.permissions = 'At least one permission is required';
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSave = () => {
    if (validateForm() && onSave) {
      onSave(formData);
    }
  };

  const handlePermissionToggle = (permissionId: string) => {
    setFormData(prev => ({
      ...prev,
      permissions: prev.permissions.includes(permissionId)
        ? prev.permissions.filter(p => p !== permissionId)
        : [...prev.permissions, permissionId],
    }));
    if (errors.permissions) {
      setErrors(prev => ({ ...prev, permissions: '' }));
    }
  };

  const groupedPermissions = PERMISSION_OPTIONS.reduce((acc, perm) => {
    if (!acc[perm.category]) acc[perm.category] = [];
    acc[perm.category].push(perm);
    return acc;
  }, {} as Record<string, Permission[]>);

  return (
    <div className="flex items-center justify-center min-h-screen bg-slate-50 p-4">
      <div className="w-full max-w-2xl bg-white rounded-lg shadow-lg">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-[#e7ecf4] px-6 py-4">
          <h1 className="text-2xl font-bold text-[#0d131c]">
            {initialData ? 'Edit Role' : 'Create New Role'}
          </h1>
          {onClose && (
            <button
              onClick={onClose}
              className="p-1 hover:bg-slate-100 rounded-lg transition-colors"
              aria-label="Close"
            >
              <X className="w-6 h-6 text-[#496a9c]" />
            </button>
          )}
        </div>

        {/* Content */}
        <div className="p-6 space-y-6 max-h-[calc(100vh-200px)] overflow-y-auto">
          {/* Basic Information Section */}
          <div>
            <h2 className="text-lg font-semibold text-[#0d131c] mb-4">Basic Information</h2>
            <div className="space-y-4">
              {/* Role Key */}
              <div>
                <label className="block text-sm font-medium text-[#0d131c] mb-2">
                  Role Key <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={formData.roleKey}
                  onChange={(e) => {
                    setFormData(prev => ({ ...prev, roleKey: e.target.value }));
                    if (errors.roleKey) setErrors(prev => ({ ...prev, roleKey: '' }));
                  }}
                  placeholder="e.g., admin, editor, viewer"
                  className={`w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-[#3d87f5] ${
                    errors.roleKey ? 'border-red-500 bg-red-50' : 'border-[#ced9e8] bg-slate-50'
                  }`}
                />
                {errors.roleKey && <p className="text-sm text-red-500 mt-1">{errors.roleKey}</p>}
              </div>

              {/* Role Name */}
              <div>
                <label className="block text-sm font-medium text-[#0d131c] mb-2">
                  Role Name <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={formData.roleName}
                  onChange={(e) => {
                    setFormData(prev => ({ ...prev, roleName: e.target.value }));
                    if (errors.roleName) setErrors(prev => ({ ...prev, roleName: '' }));
                  }}
                  placeholder="e.g., Administrator, Editor"
                  className={`w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-[#3d87f5] ${
                    errors.roleName ? 'border-red-500 bg-red-50' : 'border-[#ced9e8] bg-slate-50'
                  }`}
                />
                {errors.roleName && <p className="text-sm text-red-500 mt-1">{errors.roleName}</p>}
              </div>

              {/* Description */}
              <div>
                <label className="block text-sm font-medium text-[#0d131c] mb-2">Description</label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                  placeholder="Describe the purpose and responsibilities of this role"
                  rows={3}
                  className="w-full px-4 py-2 border border-[#ced9e8] rounded-lg focus:outline-none focus:ring-2 focus:ring-[#3d87f5] bg-slate-50"
                />
              </div>

              {/* Role Level */}
              <div>
                <label className="block text-sm font-medium text-[#0d131c] mb-2">
                  Role Level <span className="text-red-500">*</span>
                </label>
                <select
                  value={formData.roleLevel}
                  onChange={(e) => {
                    setFormData(prev => ({ ...prev, roleLevel: e.target.value }));
                    if (errors.roleLevel) setErrors(prev => ({ ...prev, roleLevel: '' }));
                  }}
                  className={`w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-[#3d87f5] ${
                    errors.roleLevel ? 'border-red-500 bg-red-50' : 'border-[#ced9e8] bg-slate-50'
                  }`}
                >
                  <option value="">Select role level</option>
                  {ROLE_LEVELS.map(level => (
                    <option key={level.value} value={level.value}>
                      {level.label}
                    </option>
                  ))}
                </select>
                {errors.roleLevel && <p className="text-sm text-red-500 mt-1">{errors.roleLevel}</p>}
              </div>
            </div>
          </div>

          {/* Permissions Section */}
          <div>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-[#0d131c]">Permissions</h2>
              <span className="text-sm text-[#496a9c]">{formData.permissions.length} selected</span>
            </div>
            
            {errors.permissions && (
              <p className="text-sm text-red-500 mb-4">{errors.permissions}</p>
            )}

            <div className="space-y-6">
              {Object.entries(groupedPermissions).map(([category, permissions]) => (
                <div key={category}>
                  <h3 className="text-sm font-semibold text-[#0d131c] mb-3">{category}</h3>
                  <div className="space-y-2 ml-4">
                    {permissions.map(permission => (
                      <label
                        key={permission.id}
                        className="flex items-start gap-3 p-3 rounded-lg hover:bg-slate-50 cursor-pointer transition-colors"
                      >
                        <input
                          type="checkbox"
                          checked={formData.permissions.includes(permission.id)}
                          onChange={() => handlePermissionToggle(permission.id)}
                          className="w-5 h-5 rounded border-[#ced9e8] border-2 bg-transparent text-[#3d87f5] checked:bg-[#3d87f5] checked:border-[#3d87f5] mt-0.5 focus:ring-0 focus:ring-offset-0"
                        />
                        <div>
                          <p className="text-sm font-medium text-[#0d131c]">{permission.name}</p>
                          <p className="text-xs text-[#496a9c]">{permission.description}</p>
                        </div>
                      </label>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-3 border-t border-[#e7ecf4] px-6 py-4 bg-slate-50">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg border border-[#ced9e8] bg-white text-[#0d131c] font-medium hover:bg-slate-100 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-2 rounded-lg bg-[#3d87f5] text-white font-medium hover:bg-[#2d6fd4] transition-colors flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            {initialData ? 'Update Role' : 'Create Role'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default RoleEntryForm;
