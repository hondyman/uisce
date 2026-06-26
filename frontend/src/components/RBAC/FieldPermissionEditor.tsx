/**
 * Field Permission Editor - Enterprise Field-Level Security
 * 
 * Features:
 * - Field-level access control configuration
 * - PII masking pattern management
 * - Role-to-field permission matrix
 * - Visual masking preview
 * - Compliance-ready security rules
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Lock,
  Eye,
  EyeOff,
  Edit2,
  Shield,
  Save,
  X,
  Plus,
  Trash2,
  AlertTriangle,
  CheckCircle2,
  Search,
} from 'lucide-react';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface FieldPermission {
  id: string;
  role_id: string;
  role_key: string;
  role_name: string;
  resource_type: string;
  field_name: string;
  permission_level: 'none' | 'read' | 'write' | 'mask';
}

interface MaskingRule {
  id: string;
  resource_type: string;
  field_name: string;
  masking_type: 'full' | 'partial' | 'hash' | 'tokenize';
  masking_pattern: string;
  unmasked_roles: string[];
}

interface Role {
  id: string;
  role_key: string;
  role_name: string;
  role_level: string;
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
  datasource,
}) => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [fieldPermissions, setFieldPermissions] = useState<FieldPermission[]>([]);
  const [maskingRules, setMaskingRules] = useState<MaskingRule[]>([]);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [selectedResource, setSelectedResource] = useState<string>('process');
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(true);
  const [showMaskingModal, setShowMaskingModal] = useState(false);
  const [saving, setSaving] = useState(false);

  // Common sensitive fields
  const sensitiveFields = [
    { name: 'ssn', label: 'Social Security Number', category: 'PII' },
    { name: 'tax_id', label: 'Tax ID', category: 'PII' },
    { name: 'bank_account', label: 'Bank Account', category: 'Financial' },
    { name: 'credit_card', label: 'Credit Card', category: 'Financial' },
    { name: 'email', label: 'Email Address', category: 'PII' },
    { name: 'phone', label: 'Phone Number', category: 'PII' },
    { name: 'salary', label: 'Salary', category: 'Financial' },
    { name: 'account_balance', label: 'Account Balance', category: 'Financial' },
  ];

  // Masking form state
  const [maskingForm, setMaskingForm] = useState({
    resource_type: 'process',
    field_name: '',
    masking_type: 'partial' as 'full' | 'partial' | 'hash' | 'tokenize',
    masking_pattern: '',
    unmasked_roles: [] as string[],
  });

  // Fetch roles
  const fetchRoles = async () => {
    try {
      const response = await fetch(
        `/api/rbac/roles?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      setRoles(data || []);
    } catch (error) {
      console.error('Failed to fetch roles:', error);
    }
  };

  // Fetch field permissions
  const fetchFieldPermissions = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `/api/rbac/field-permissions?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
      );
      const data = await response.json();
      setFieldPermissions(data || []);
    } catch (error) {
      console.error('Failed to fetch field permissions:', error);
    } finally {
      setLoading(false);
    }
  };

  // Set field permission
  const setFieldPermission = async (
    roleId: string,
    fieldName: string,
    level: 'none' | 'read' | 'write' | 'mask'
  ) => {
    try {
      setSaving(true);
      await fetch(`/api/rbac/field-permissions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tenant_id: tenant.id,
          tenant_instance_id: datasource.id,
          role_id: roleId,
          resource_type: selectedResource,
          field_name: fieldName,
          permission_level: level,
        }),
      });
      await fetchFieldPermissions();
    } catch (error) {
      console.error('Failed to set field permission:', error);
    } finally {
      setSaving(false);
    }
  };

  // Create masking rule
  const createMaskingRule = async () => {
    try {
      setSaving(true);
      await fetch(`/api/rbac/field-masking-rules`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...maskingForm,
          tenant_id: tenant.id,
          tenant_instance_id: datasource.id,
        }),
      });
      await fetchFieldPermissions();
      setShowMaskingModal(false);
      resetMaskingForm();
    } catch (error) {
      console.error('Failed to create masking rule:', error);
    } finally {
      setSaving(false);
    }
  };

  // Reset masking form
  const resetMaskingForm = () => {
    setMaskingForm({
      resource_type: 'process',
      field_name: '',
      masking_type: 'partial',
      masking_pattern: '',
      unmasked_roles: [],
    });
  };

  // Get permission for role and field
  const getPermission = (roleId: string, fieldName: string): 'none' | 'read' | 'write' | 'mask' => {
    const perm = fieldPermissions.find(
      p => p.role_id === roleId && p.field_name === fieldName && p.resource_type === selectedResource
    );
    return perm?.permission_level || 'none';
  };

  // Filter fields by search
  const filteredFields = useMemo(() => {
    return sensitiveFields.filter(field =>
      field.label.toLowerCase().includes(searchTerm.toLowerCase()) ||
      field.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      field.category.toLowerCase().includes(searchTerm.toLowerCase())
    );
  }, [searchTerm]);

  useEffect(() => {
    fetchRoles();
    fetchFieldPermissions();
  }, [tenant.id]);

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
    const icons = {
      none: <EyeOff className="w-4 h-4" />,
      read: <Eye className="w-4 h-4" />,
      write: <Edit2 className="w-4 h-4" />,
      mask: <Lock className="w-4 h-4" />,
    };
    return icons[level as keyof typeof icons] || <EyeOff className="w-4 h-4" />;
  };

  const maskingPatterns = {
    ssn: 'XXX-XX-####',
    tax_id: 'XX-XXXXXXX',
    bank_account: 'XXXX-####',
    credit_card: 'XXXX-XXXX-XXXX-####',
    email: 'X***@domain.com',
    phone: '(XXX) XXX-####',
  };

  const getExampleMasked = (fieldName: string, pattern: string) => {
    const examples = {
      ssn: { original: '123-45-6789', masked: 'XXX-XX-6789' },
      tax_id: { original: '12-3456789', masked: 'XX-XXXXXX9' },
      bank_account: { original: '1234-5678', masked: 'XXXX-5678' },
      credit_card: { original: '1234-5678-9012-3456', masked: 'XXXX-XXXX-XXXX-3456' },
      email: { original: 'john.doe@example.com', masked: 'j***@example.com' },
      phone: { original: '(555) 123-4567', masked: '(XXX) XXX-4567' },
    };
    return examples[fieldName as keyof typeof examples] || { original: 'Sample', masked: 'XXXX' };
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="flex flex-col items-center gap-4">
          <Shield className="w-12 h-12 animate-pulse text-blue-500" />
          <p className="text-gray-600">Loading field permissions...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 p-6">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 flex items-center gap-3">
              <Lock className="w-8 h-8 text-blue-600" />
              Field Permission Editor
            </h1>
            <p className="text-gray-600 mt-2">
              Configure field-level security and masking rules for {tenant.display_name}
            </p>
          </div>
          <button
            onClick={() => {
              resetMaskingForm();
              setShowMaskingModal(true);
            }}
            className="px-6 py-3 bg-purple-600 text-white rounded-lg font-medium hover:bg-purple-700 transition-all flex items-center gap-2 shadow-lg"
          >
            <Plus className="w-5 h-5" />
            Add Masking Rule
          </button>
        </div>

        {/* Resource Type & Search */}
        <div className="flex gap-4 mt-6">
          <select
            value={selectedResource}
            onChange={(e) => setSelectedResource(e.target.value)}
            className="px-4 py-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500 bg-white"
          >
            <option value="process">Process</option>
            <option value="step">Step</option>
            <option value="document">Document</option>
          </select>
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
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
                  Field Name
                </th>
                <th className="text-left py-4 px-6 font-bold text-gray-700">Category</th>
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
              {filteredFields.map((field, idx) => (
                <tr key={field.name} className={idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
                  <td className="py-4 px-6 font-medium text-gray-900 sticky left-0 bg-inherit z-10">
                    <div>
                      <div className="font-bold">{field.label}</div>
                      <div className="text-xs text-gray-500">{field.name}</div>
                    </div>
                  </td>
                  <td className="py-4 px-6">
                    <span className={`px-3 py-1 rounded-full text-xs font-bold ${
                      field.category === 'PII'
                        ? 'bg-purple-100 text-purple-700 border border-purple-300'
                        : 'bg-green-100 text-green-700 border border-green-300'
                    }`}>
                      {field.category}
                    </span>
                  </td>
                  {roles.map(role => {
                    const currentLevel = getPermission(role.id, field.name);
                    return (
                      <td key={role.id} className="py-4 px-4">
                        <div className="flex flex-col gap-1">
                          {(['none', 'read', 'write', 'mask'] as const).map(level => (
                            <button
                              key={level}
                              onClick={() => setFieldPermission(role.id, field.name, level)}
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
              ))}
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
              <div className="text-xs text-gray-600">No access (hidden)</div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className={`px-4 py-2 rounded-lg border-2 ${getPermissionColor('read')}`}>
              {getPermissionIcon('read')}
            </div>
            <div>
              <div className="font-medium text-gray-900">Read</div>
              <div className="text-xs text-gray-600">Full visibility</div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className={`px-4 py-2 rounded-lg border-2 ${getPermissionColor('write')}`}>
              {getPermissionIcon('write')}
            </div>
            <div>
              <div className="font-medium text-gray-900">Write</div>
              <div className="text-xs text-gray-600">Full access</div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className={`px-4 py-2 rounded-lg border-2 ${getPermissionColor('mask')}`}>
              {getPermissionIcon('mask')}
            </div>
            <div>
              <div className="font-medium text-gray-900">Mask</div>
              <div className="text-xs text-gray-600">Partial visibility</div>
            </div>
          </div>
        </div>
      </div>

      {/* Masking Modal */}
      {showMaskingModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b-2 border-gray-200">
              <div className="flex items-center justify-between">
                <h3 className="text-2xl font-bold text-gray-900">Add Masking Rule</h3>
                <button
                  onClick={() => setShowMaskingModal(false)}
                  className="p-2 hover:bg-gray-100 rounded-lg transition-all"
                >
                  <X className="w-6 h-6 text-gray-600" />
                </button>
              </div>
            </div>

            <div className="p-6 space-y-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Field Name *
                </label>
                <select
                  value={maskingForm.field_name}
                  onChange={(e) => {
                    const fieldName = e.target.value;
                    setMaskingForm({
                      ...maskingForm,
                      field_name: fieldName,
                      masking_pattern: maskingPatterns[fieldName as keyof typeof maskingPatterns] || '',
                    });
                  }}
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                >
                  <option value="">Select a field</option>
                  {sensitiveFields.map(field => (
                    <option key={field.name} value={field.name}>
                      {field.label} ({field.name})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Masking Type *
                </label>
                <select
                  value={maskingForm.masking_type}
                  onChange={(e) =>
                    setMaskingForm({
                      ...maskingForm,
                      masking_type: e.target.value as 'full' | 'partial' | 'hash' | 'tokenize',
                    })
                  }
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                >
                  <option value="full">Full Masking (****)</option>
                  <option value="partial">Partial Masking (XXX-XX-1234)</option>
                  <option value="hash">Hash (SHA-256)</option>
                  <option value="tokenize">Tokenize (Replace with token)</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Masking Pattern *
                </label>
                <input
                  type="text"
                  value={maskingForm.masking_pattern}
                  onChange={(e) =>
                    setMaskingForm({ ...maskingForm, masking_pattern: e.target.value })
                  }
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  placeholder="e.g., XXX-XX-####"
                />
                <p className="text-xs text-gray-500 mt-2">
                  Use X for masked characters, # for visible characters from the end
                </p>
              </div>

              {/* Preview */}
              {maskingForm.field_name && maskingForm.masking_pattern && (
                <div className="p-4 bg-blue-50 border-2 border-blue-200 rounded-xl">
                  <h4 className="font-bold text-gray-900 mb-3">Masking Preview</h4>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm text-gray-600">Original</p>
                      <p className="font-mono text-lg text-gray-900">
                        {getExampleMasked(maskingForm.field_name, maskingForm.masking_pattern).original}
                      </p>
                    </div>
                    <div className="text-2xl text-gray-400">→</div>
                    <div>
                      <p className="text-sm text-gray-600">Masked</p>
                      <p className="font-mono text-lg font-bold text-blue-600">
                        {maskingForm.masking_pattern}
                      </p>
                    </div>
                  </div>
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Unmasked Roles (Full Access)
                </label>
                <div className="space-y-2 max-h-48 overflow-y-auto border-2 border-gray-200 rounded-lg p-4">
                  {roles.map(role => (
                    <label key={role.id} className="flex items-center gap-3 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={maskingForm.unmasked_roles.includes(role.id)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setMaskingForm({
                              ...maskingForm,
                              unmasked_roles: [...maskingForm.unmasked_roles, role.id],
                            });
                          } else {
                            setMaskingForm({
                              ...maskingForm,
                              unmasked_roles: maskingForm.unmasked_roles.filter(id => id !== role.id),
                            });
                          }
                        }}
                        className="w-5 h-5 text-blue-600 border-2 border-gray-300 rounded"
                      />
                      <span className="font-medium text-gray-900">{role.role_name}</span>
                      <span className="text-xs text-gray-500">({role.role_level})</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>

            <div className="p-6 border-t-2 border-gray-200 flex justify-end gap-3">
              <button
                onClick={() => setShowMaskingModal(false)}
                className="px-6 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium hover:bg-gray-200 transition-all"
              >
                Cancel
              </button>
              <button
                onClick={createMaskingRule}
                disabled={!maskingForm.field_name || !maskingForm.masking_pattern || saving}
                className="px-6 py-3 bg-purple-600 text-white rounded-lg font-medium hover:bg-purple-700 transition-all flex items-center gap-2 disabled:opacity-50"
              >
                <Save className="w-5 h-5" />
                {saving ? 'Saving...' : 'Save Rule'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
