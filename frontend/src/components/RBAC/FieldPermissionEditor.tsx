/**
 * Field Permission Editor - Enterprise Field-Level Security
 *
 * Configures bp_field_permissions rows for a specific business object, the
 * same table/mechanism security.EntitlementsService reads to enforce read
 * access, write access, and PII masking on BO records. Permissions here are
 * always scoped to one resource_type="business_object" + resource_id (the
 * selected BO's id) — EntitlementsService only ever queries
 * resource_type='business_object', and enforcement is keyed per BO
 * instance, so a permission row without a concrete resource_id is inert.
 */

import React, { useState, useEffect, useMemo } from 'react';
import { Box, Typography } from '@mui/material';
import apiClient from '../../utils/apiClient';
import {
  Lock as LockIcon,
  Visibility as VisibilityIcon,
  VisibilityOff as VisibilityOffIcon,
  Edit as EditIcon,
  Security as SecurityIcon,
  Save as SaveIcon,
  Close as CloseIcon,
  Search as SearchIcon,
} from '@mui/icons-material';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface FieldPermission {
  id: string;
  role_id: string;
  role_key: string;
  role_name: string;
  resource_type?: string;
  resource_id?: string;
  field_name: string;
  permission_level: 'none' | 'read' | 'write' | 'mask';
  masking_pattern?: string;
}

interface Role {
  id: string;
  role_key: string;
  role_name: string;
  role_level: string;
}

interface BusinessObjectSummary {
  id: string;
  name: string;
  displayName: string;
}

interface BOField {
  id: string;
  name: string;
  technicalName: string;
}

interface FieldPermissionEditorProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const FieldPermissionEditor: React.FC<FieldPermissionEditorProps> = ({
  tenant,
}) => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [businessObjects, setBusinessObjects] = useState<BusinessObjectSummary[]>([]);
  const [selectedBOId, setSelectedBOId] = useState<string>('');
  const [boFields, setBoFields] = useState<BOField[]>([]);
  const [fieldPermissions, setFieldPermissions] = useState<FieldPermission[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadingFields, setLoadingFields] = useState(false);
  const [saving, setSaving] = useState(false);

  const [maskModal, setMaskModal] = useState<{ roleId: string; fieldName: string } | null>(null);
  const [maskingPattern, setMaskingPattern] = useState('XXX-XX-####');

  // Fetch roles + business objects once
  useEffect(() => {
    (async () => {
      try {
        setLoading(true);
        const [roleData, boData] = await Promise.all([
          apiClient<Role[]>('/api/rbac/roles'),
          apiClient<BusinessObjectSummary[]>('/api/business-objects'),
        ]);
        setRoles(roleData || []);
        const bos = boData || [];
        setBusinessObjects(bos);
        if (bos.length > 0) {
          setSelectedBOId(bos[0].id);
        }
      } catch (error) {
        console.error('Failed to load roles/business objects:', error);
      } finally {
        setLoading(false);
      }
    })();
  }, [tenant.id]);

  // Fetch this BO's fields + its existing field permissions whenever the selected BO changes
  useEffect(() => {
    if (!selectedBOId) {
      setBoFields([]);
      setFieldPermissions([]);
      return;
    }
    (async () => {
      try {
        setLoadingFields(true);
        const [fields, perms] = await Promise.all([
          apiClient<BOField[]>(`/api/business-objects/${selectedBOId}/fields`),
          apiClient<FieldPermission[]>('/api/rbac/field-permissions'),
        ]);
        setBoFields(fields || []);
        setFieldPermissions(
          (perms || []).filter(
            p => p.resource_type === 'business_object' && p.resource_id === selectedBOId
          )
        );
      } catch (error) {
        console.error('Failed to load BO fields/permissions:', error);
      } finally {
        setLoadingFields(false);
      }
    })();
  }, [selectedBOId]);

  const refetchPermissions = async () => {
    try {
      const perms = await apiClient<FieldPermission[]>('/api/rbac/field-permissions');
      setFieldPermissions(
        (perms || []).filter(
          p => p.resource_type === 'business_object' && p.resource_id === selectedBOId
        )
      );
    } catch (error) {
      console.error('Failed to refresh field permissions:', error);
    }
  };

  // Set field permission
  const setFieldPermission = async (
    roleId: string,
    fieldName: string,
    level: 'none' | 'read' | 'write' | 'mask',
    pattern?: string
  ) => {
    if (!selectedBOId) return;
    try {
      setSaving(true);
      await apiClient(`/api/rbac/field-permissions`, {
        method: 'POST',
        body: JSON.stringify({
          role_id: roleId,
          resource_type: 'business_object',
          resource_id: selectedBOId,
          field_name: fieldName,
          permission_level: level,
          masking_pattern: pattern,
        }),
      });
      await refetchPermissions();
    } catch (error) {
      console.error('Failed to set field permission:', error);
    } finally {
      setSaving(false);
    }
  };

  const handleLevelClick = (roleId: string, fieldName: string, level: 'none' | 'read' | 'write' | 'mask') => {
    if (level === 'mask') {
      const existing = fieldPermissions.find(
        p => p.role_id === roleId && p.field_name === fieldName
      );
      setMaskingPattern(existing?.masking_pattern || 'XXX-XX-####');
      setMaskModal({ roleId, fieldName });
      return;
    }
    setFieldPermission(roleId, fieldName, level);
  };

  // Get permission for role and field
  const getPermission = (roleId: string, fieldName: string): 'none' | 'read' | 'write' | 'mask' => {
    const perm = fieldPermissions.find(
      p => p.role_id === roleId && p.field_name === fieldName
    );
    return perm?.permission_level || 'none';
  };

  // Filter fields by search
  const filteredFields = useMemo(() => {
    const q = searchTerm.toLowerCase();
    return boFields.filter(field =>
      (field.name || '').toLowerCase().includes(q) ||
      (field.technicalName || '').toLowerCase().includes(q)
    );
  }, [boFields, searchTerm]);

  const getPermissionColor = (level: string) => {
    const colors = {
      none: 'bg-red-100 text-red-700 border-red-300',
      read: 'bg-green-100 text-green-700 border-green-300',
      write: 'bg-blue-100 text-blue-700 border-blue-300',
      mask: 'bg-yellow-100 text-yellow-700 border-yellow-300',
    };
    return colors[level as keyof typeof colors] || 'bg-gray-100 text-gray-700 border-gray-300';
  };

  const getPermissionIcon = (level: string) => {
    const icons: Record<string, React.ReactNode> = {
      none: <VisibilityOffIcon sx={{ width: 16, height: 16 }} />,
      read: <VisibilityIcon sx={{ width: 16, height: 16 }} />,
      write: <EditIcon sx={{ width: 16, height: 16 }} />,
      mask: <LockIcon sx={{ width: 16, height: 16 }} />,
    };
    return icons[level] || <VisibilityOffIcon sx={{ width: 16, height: 16 }} />;
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '60vh' }}>
        <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4 }}>
          <SecurityIcon sx={{ width: 48, height: 48, color: 'primary.main' }} />
          <Typography color="text.secondary">Loading field permissions...</Typography>
        </Box>
      </Box>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 p-6">
      {/* Header */}
      <div className="mb-8">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 flex items-center gap-3">
            <LockIcon sx={{ width: 32, height: 32, color: 'primary.main' }} />
            Field Permission Editor
          </h1>
          <p className="text-gray-600 mt-2">
            Configure per-field read/write/mask access for {tenant.display_name}. Changes apply to the
            selected business object only.
          </p>
        </div>

        {/* Business Object & Search */}
        <div className="flex gap-4 mt-6">
          <select
            value={selectedBOId}
            onChange={(e) => setSelectedBOId(e.target.value)}
            className="px-4 py-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500 bg-white min-w-[220px]"
          >
            {businessObjects.length === 0 && <option value="">No business objects found</option>}
            {businessObjects.map(bo => (
              <option key={bo.id} value={bo.id}>
                {bo.displayName || bo.name}
              </option>
            ))}
          </select>
          <div className="flex-1 relative">
            <SearchIcon sx={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: 'text.secondary', width: 20, height: 20 }} />
            <input
              type="text"
              placeholder="Search fields..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
            />
          </div>
        </div>
      </div>

      {/* Permission Matrix */}
      <div className="bg-white rounded-2xl shadow-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-100">
              <tr>
                <th className="text-left py-4 px-6 font-bold text-gray-700 sticky left-0 bg-gray-100 z-10">
                  Field
                </th>
                {roles.map(role => (
                  <th key={role.id} className="text-center py-4 px-4 font-bold text-gray-700 min-w-[150px]">
                    <div className="flex flex-col items-center gap-1">
                      <span>{role.role_name}</span>
                      <span className="text-xs text-gray-500 font-normal">{role.role_level}</span>
                    </div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loadingFields ? (
                <tr>
                  <td colSpan={roles.length + 1} className="py-8 text-center text-gray-500">
                    Loading fields...
                  </td>
                </tr>
              ) : filteredFields.length === 0 ? (
                <tr>
                  <td colSpan={roles.length + 1} className="py-8 text-center text-gray-500">
                    {selectedBOId ? 'No fields found for this business object.' : 'Select a business object.'}
                  </td>
                </tr>
              ) : (
                filteredFields.map((field, idx) => {
                  const fieldName = field.technicalName || field.name;
                  return (
                    <tr key={field.id} className={idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
                      <td className="py-4 px-6 font-medium text-gray-900 sticky left-0 bg-inherit z-10">
                        <div>
                          <div className="font-bold">{field.name}</div>
                          <div className="text-xs text-gray-500">{fieldName}</div>
                        </div>
                      </td>
                      {roles.map(role => {
                        const currentLevel = getPermission(role.id, fieldName);
                        return (
                          <td key={role.id} className="py-4 px-4">
                            <div className="flex flex-col gap-1">
                              {(['none', 'read', 'write', 'mask'] as const).map(level => (
                                <button
                                  key={level}
                                  onClick={() => handleLevelClick(role.id, fieldName, level)}
                                  disabled={saving}
                                  className={`px-3 py-2 rounded-lg text-xs font-bold border-2 transition-all flex items-center justify-center gap-2 ${
                                    currentLevel === level
                                      ? getPermissionColor(level)
                                      : 'bg-white text-gray-400 border-gray-200 hover:border-gray-300'
                                  }`}
                                >
                                  {getPermissionIcon(level)}
                                  {level.toUpperCase()}
                                </button>
                              ))}
                            </div>
                          </td>
                        );
                      })}
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Legend */}
      <div className="mt-6 bg-white rounded-2xl shadow-xl p-6">
        <h3 className="text-lg font-bold text-gray-900 mb-4">Permission Levels</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="flex items-center gap-3">
            <div className={`px-4 py-2 rounded-lg border-2 ${getPermissionColor('none')}`}>
              {getPermissionIcon('none')}
            </div>
            <div>
              <div className="font-medium text-gray-900">None</div>
              <div className="text-xs text-gray-600">Field hidden entirely</div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className={`px-4 py-2 rounded-lg border-2 ${getPermissionColor('read')}`}>
              {getPermissionIcon('read')}
            </div>
            <div>
              <div className="font-medium text-gray-900">Read</div>
              <div className="text-xs text-gray-600">Visible, not editable</div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className={`px-4 py-2 rounded-lg border-2 ${getPermissionColor('write')}`}>
              {getPermissionIcon('write')}
            </div>
            <div>
              <div className="font-medium text-gray-900">Write</div>
              <div className="text-xs text-gray-600">Visible and editable</div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className={`px-4 py-2 rounded-lg border-2 ${getPermissionColor('mask')}`}>
              {getPermissionIcon('mask')}
            </div>
            <div>
              <div className="font-medium text-gray-900">Mask</div>
              <div className="text-xs text-gray-600">Visible, value replaced</div>
            </div>
          </div>
        </div>
      </div>

      {/* Masking pattern modal — shown when a cell is set to "mask" */}
      {maskModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-md w-full">
            <div className="p-6 border-b-2 border-gray-200">
              <div className="flex items-center justify-between">
                <h3 className="text-xl font-bold text-gray-900">Masking Pattern</h3>
                <button
                  onClick={() => setMaskModal(null)}
                  className="p-2 hover:bg-gray-100 rounded-lg transition-all"
                >
                  <CloseIcon sx={{ width: 24, height: 24, color: 'text.secondary' }} />
                </button>
              </div>
              <p className="text-sm text-gray-600 mt-2">
                Field <span className="font-mono font-bold">{maskModal.fieldName}</span>
              </p>
            </div>

            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Pattern
                </label>
                <input
                  type="text"
                  value={maskingPattern}
                  onChange={(e) => setMaskingPattern(e.target.value)}
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500 font-mono"
                  placeholder="e.g., XXX-XX-####"
                />
                <p className="text-xs text-gray-500 mt-2">
                  X = masked character, # = visible character kept from the end of the value.
                </p>
              </div>
            </div>

            <div className="p-6 border-t-2 border-gray-200 flex justify-end gap-3">
              <button
                onClick={() => setMaskModal(null)}
                className="px-6 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium hover:bg-gray-200 transition-all"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  if (!maskModal) return;
                  await setFieldPermission(maskModal.roleId, maskModal.fieldName, 'mask', maskingPattern);
                  setMaskModal(null);
                }}
                disabled={!maskingPattern || saving}
                className="px-6 py-3 bg-purple-600 text-white rounded-lg font-medium hover:bg-purple-700 transition-all flex items-center gap-2 disabled:opacity-50"
              >
                <SaveIcon sx={{ width: 20, height: 20 }} />
                {saving ? 'Saving...' : 'Apply Mask'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
