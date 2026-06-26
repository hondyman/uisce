import React, { useState } from 'react';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import { Plus, Trash2, Save, X, Edit2, Check, AlertCircle } from 'lucide-react';
import ParameterBuilder from './ParameterBuilder';
import { getParameterSchema, getAvailableRuleTypes, validateParameters } from '../lib/parameterSchemas';

interface Rule {
  id?: string;
  name: string;
  description: string;
  ruleType: string;
  parameters: Record<string, any>;
  enabled: boolean;
  createdAt?: string;
  updatedAt?: string;
}

interface RuleBuilderProps {
  onSave?: (rule: Rule) => void;
  onDelete?: (id: string) => void;
  onUpdate?: (rule: Rule) => void;
  rules?: Rule[];
  initialRule?: Rule;
}

/**
 * RuleBuilder
 * 
 * Schema-driven business rule configuration builder using unified ParameterBuilder component.
 * Allows users to:
 * - Select rule type (determines available parameters)
 * - Configure parameters via ParameterBuilder (8 field types, validation)
 * - Create, edit, delete rules
 * - Enable/disable rules
 * - View all configured rules
 * 
 * Integration: Uses ParameterBuilder for consistent parameter UI/UX across app
 */
export const RuleBuilder: React.FC<RuleBuilderProps> = ({
  onSave,
  onDelete,
  onUpdate,
  rules: externalRules = [],
  initialRule,
}) => {
  const [rules, setRules] = useState<Rule[]>(externalRules);
  const [formData, setFormData] = useState<Rule>(
    initialRule || {
      name: '',
      description: '',
      ruleType: 'CONCENTRATION',
      parameters: {},
      enabled: true,
    }
  );

  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});
  const [showValidation, setShowValidation] = useState(false);
  const [isEditingNew, setIsEditingNew] = useState(!initialRule);
  const [editingId, setEditingId] = useState<string | undefined>(initialRule?.id);
  const confirm = useConfirm();
  const notification = useNotification();

  // Get available rule types
  const ruleTypes = getAvailableRuleTypes();

  // Handle rule type change - reset parameters when type changes
  const handleRuleTypeChange = (newType: string) => {
    setFormData({
      ...formData,
      ruleType: newType,
      parameters: {}, // Reset parameters for new type
    });
    setValidationErrors({});
  };

  // Handle parameter changes from ParameterBuilder
  const handleParametersChange = (newParams: Record<string, any>) => {
    setFormData({
      ...formData,
      parameters: newParams,
    });
    // Clear validation errors when user changes parameters
    setValidationErrors({});
  };

  // Validate and save new rule
  const handleSaveRule = () => {
    const schema = getParameterSchema(formData.ruleType);
    if (!schema) {
      setValidationErrors({ base: 'Invalid rule type selected' });
      return;
    }

    // Validate parameters
    const errors = validateParameters(formData.ruleType, formData.parameters);
    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors);
      setShowValidation(true);
      return;
    }

    // Validate basic fields
    if (!formData.name.trim()) {
      setValidationErrors({ name: 'Rule name is required' });
      setShowValidation(true);
      return;
    }

    // All validations passed
    setShowValidation(false);

    const newRule: Rule = {
      ...formData,
      id: formData.id || `rule_${Date.now()}`,
      createdAt: formData.createdAt || new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    setRules([...rules, newRule]);

    // Call external callback
    onSave?.(newRule);

    // Reset form
    setFormData({
      name: '',
      description: '',
      ruleType: 'CONCENTRATION',
      parameters: {},
      enabled: true,
    });
    setValidationErrors({});
    setIsEditingNew(false);
  };

  // Update existing rule
  const handleUpdateRule = (updatedRule: Rule) => {
    setRules(
      rules.map((r) => (r.id === updatedRule.id ? updatedRule : r))
    );
    onUpdate?.(updatedRule);
    setEditingId(undefined);
  };

  // Delete rule
  const handleDeleteRule = async (id: string | undefined) => {
    if (!id) return;
    if (!(await confirm({ title: 'Delete rule', description: 'Delete this rule?' }))) return;
    setRules(rules.filter((r) => r.id !== id));
    onDelete?.(id);
    notification.success('Rule deleted');
  };

  // Edit rule
  const handleEditRule = (rule: Rule) => {
    setFormData(rule);
    setEditingId(rule.id);
    setIsEditingNew(true);
  };

  // Cancel edit
  const handleCancelEdit = () => {
    setFormData({
      name: '',
      description: '',
      ruleType: 'CONCENTRATION',
      parameters: {},
      enabled: true,
    });
    setValidationErrors({});
    setIsEditingNew(false);
    setEditingId(undefined);
  };

  // Toggle rule enabled state
  const handleToggleRule = (id: string | undefined) => {
    if (!id) return;
    const updated = rules.map((r) => {
      if (r.id === id) {
        const updatedRule = { ...r, enabled: !r.enabled };
        onUpdate?.(updatedRule);
        return updatedRule;
      }
      return r;
    });
    setRules(updated);
  };

  const schema = getParameterSchema(formData.ruleType);

  return (
    <div className="bg-white dark:bg-slate-900 rounded-lg p-6 shadow-lg border dark:border-slate-700">
      <h2 className="text-2xl font-bold mb-6 text-gray-900 dark:text-white">Business Rules Engine</h2>

      {/* Rule Creation Form */}
      {isEditingNew && (
        <div className="mb-6 p-6 bg-blue-50 dark:bg-slate-800 border-2 border-blue-200 dark:border-slate-700 rounded-lg">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            {editingId ? 'Edit Rule' : 'Create New Rule'}
          </h3>

          <div className="space-y-4">
            {/* Rule Name */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                Rule Name *
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="e.g., Maximum Position Concentration"
                className="w-full px-4 py-2 border dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 outline-none"
              />
              {validationErrors.name && showValidation && (
                <p className="mt-1 text-sm text-red-600 dark:text-red-400">{validationErrors.name}</p>
              )}
            </div>

            {/* Rule Description */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                Description
              </label>
              <textarea
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                placeholder="Optional: Describe what this rule validates"
                rows={2}
                className="w-full px-4 py-2 border dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 outline-none"
              />
            </div>

            {/* Rule Type */}
            <div className="flex items-end gap-4">
              <div className="flex-1">
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                  Rule Type *
                </label>
                <select
                  value={formData.ruleType}
                  onChange={(e) => handleRuleTypeChange(e.target.value)}
                  aria-label="Rule Type"
                  className="w-full px-4 py-2 border dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 outline-none"
                >
                  {ruleTypes.map((type) => (
                    <option key={type.value} value={type.value}>
                      {type.label}
                    </option>
                  ))}
                </select>
              </div>

              <label className="flex items-center gap-2 cursor-pointer pb-2">
                <input
                  type="checkbox"
                  checked={formData.enabled}
                  onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                  className="w-4 h-4 rounded border dark:border-slate-600 accent-blue-600"
                />
                <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">Enabled</span>
              </label>
            </div>

            {/* Parameter Configuration */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
                Rule Parameters *
              </label>
              <div className="p-4 bg-white dark:bg-slate-900 border dark:border-slate-700 rounded-lg">
                {schema ? (
                  <ParameterBuilder
                    schema={schema}
                    parameters={formData.parameters}
                    onChange={handleParametersChange}
                    errors={validationErrors}
                    showValidation={showValidation}
                  />
                ) : (
                  <p className="text-gray-600 dark:text-gray-400 text-sm">
                    Select a rule type to configure parameters
                  </p>
                )}
              </div>
            </div>

            {/* Error Display */}
            {validationErrors.base && showValidation && (
              <div className="p-3 bg-red-100 dark:bg-red-900 border border-red-400 dark:border-red-700 rounded-lg flex items-center gap-2">
                <AlertCircle size={18} className="text-red-600 dark:text-red-300 flex-shrink-0" />
                <p className="text-sm text-red-800 dark:text-red-200">{validationErrors.base}</p>
              </div>
            )}

            {/* Form Actions */}
            <div className="flex gap-3 justify-end pt-2">
              <button
                onClick={handleCancelEdit}
                className="flex items-center gap-2 px-4 py-2 bg-gray-400 hover:bg-gray-500 text-white rounded-lg font-semibold transition-colors"
              >
                <X size={18} />
                Cancel
              </button>
              <button
                onClick={editingId ? () => handleUpdateRule(formData) : handleSaveRule}
                className="flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold transition-colors"
              >
                <Check size={18} />
                {editingId ? 'Update Rule' : 'Create Rule'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create New Button (when not editing) */}
      {!isEditingNew && (
        <button
          onClick={() => setIsEditingNew(true)}
          className="mb-6 flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-semibold transition-colors"
        >
          <Plus size={18} />
          Create New Rule
        </button>
      )}

      {/* Rules List */}
      <div>
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Configured Rules ({rules.length})
        </h3>

        {rules.length > 0 ? (
          <div className="space-y-3">
            {rules.map((rule) => (
              <div
                key={rule.id}
                className={`p-4 rounded-lg border-2 transition-colors ${
                  rule.enabled
                    ? 'bg-white dark:bg-slate-800 border-gray-200 dark:border-slate-700'
                    : 'bg-gray-50 dark:bg-slate-800 border-gray-300 dark:border-slate-700 opacity-60'
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3">
                      <p className="font-semibold text-gray-900 dark:text-white">{rule.name}</p>
                      <span
                        className={`px-2 py-1 rounded text-xs font-semibold ${
                          rule.enabled
                            ? 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200'
                            : 'bg-gray-300 dark:bg-gray-600 text-gray-800 dark:text-gray-200'
                        }`}
                      >
                        {rule.enabled ? 'Active' : 'Inactive'}
                      </span>
                    </div>

                    {rule.description && (
                      <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{rule.description}</p>
                    )}

                    <div className="flex gap-4 mt-2 text-xs text-gray-500 dark:text-gray-500">
                      <span>Type: <span className="font-mono font-semibold">{rule.ruleType}</span></span>
                      {rule.createdAt && (
                        <span>Created: {new Date(rule.createdAt).toLocaleDateString()}</span>
                      )}
                    </div>

                    {/* Parameter Summary */}
                    {Object.keys(rule.parameters).length > 0 && (
                      <div className="mt-2 p-2 bg-gray-100 dark:bg-slate-700 rounded text-xs">
                        <p className="font-semibold text-gray-700 dark:text-gray-300 mb-1">Parameters:</p>
                        <ul className="space-y-1 text-gray-600 dark:text-gray-400">
                          {Object.entries(rule.parameters).map(([key, value]) => (
                            <li key={key}>
                              <span className="font-mono">{key}</span>: {String(value)}
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>

                  <div className="flex gap-2 ml-4">
                    <button
                      onClick={() => handleToggleRule(rule.id)}
                      className={`p-2 rounded transition-colors ${
                        rule.enabled
                          ? 'text-blue-600 hover:bg-blue-100 dark:text-blue-400 dark:hover:bg-slate-700'
                          : 'text-orange-600 hover:bg-orange-100 dark:text-orange-400 dark:hover:bg-slate-700'
                      }`}
                      title={rule.enabled ? 'Disable rule' : 'Enable rule'}
                    >
                      <Check size={18} />
                    </button>

                    <button
                      onClick={() => handleEditRule(rule)}
                      className="p-2 text-blue-600 hover:bg-blue-100 dark:text-blue-400 dark:hover:bg-slate-700 rounded transition-colors"
                      title="Edit rule"
                    >
                      <Edit2 size={18} />
                    </button>

                    <button
                      onClick={() => handleDeleteRule(rule.id)}
                      className="p-2 text-red-600 hover:bg-red-100 dark:text-red-400 dark:hover:bg-slate-700 rounded transition-colors"
                      title="Delete rule"
                    >
                      <Trash2 size={18} />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            <p>No rules configured yet.</p>
            <p className="text-sm mt-1">Create your first rule to get started.</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default RuleBuilder;
