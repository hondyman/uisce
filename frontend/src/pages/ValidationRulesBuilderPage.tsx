/**
 * Validation Rules Builder Page - REFACTORED
 * UI for creating, editing, and managing validation rules
 * 
 * Improvements:
 * - Uses custom hooks for API (useValidationRulesAPI) and forms (useValidationRuleForm)
 * - Reusable components (RuleCard, RulesList)
 * - Optimistic updates with rollback on error
 * - Better error handling and validation
 * - Performance optimized with memoization
 */

import React, { useEffect, useCallback, useMemo, useState } from 'react';
import { Plus, Save, X, AlertCircle, CheckCircle } from 'lucide-react';
import { WEALTH_VALIDATION_RULES } from '../data/wealthValidationRules';
import { useTenant } from '../contexts/TenantContext';
import {
  getRuleTypeOptions,
  getAccountTypeOptions,
} from '../lib/validationConstants';
import { devLog } from '../utils/devLogger';
import ParameterBuilder from '../components/ParameterBuilder';
import { getParameterSchema } from '../lib/parameterSchemas';
import RulesList from '../components/RulesList';
import { useValidationRulesAPI } from '../hooks/useValidationRulesAPI';
import { useValidationRuleForm } from '../hooks/useValidationRuleForm';
import { useConfirm } from '../components/ConfirmProvider';

export const ValidationRulesBuilderPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const confirm = useConfirm();

  // API management with optimistic updates
  const { rules, loading, saving, error, loadRules, createRule, updateRule, deleteRule, clearError } =
    useValidationRulesAPI({
      tenantId: tenant?.id,
      datasourceId: datasource?.id,
      onSuccess: (action, rule) => {
        devLog(`[ValidationRulesBuilder] ${action} successful`, { rule });
        showToast('success', `Rule ${action} successful`);
      },
      onError: (action, err) => {
        devLog(`[ValidationRulesBuilder] ${action} failed`, { error: err.message });
        showToast('error', `Failed to ${action} rule: ${err.message}`);
      },
    });

  // Form management with validation
  const form = useValidationRuleForm({
    onSubmit: async (formData) => {
      if (form.formData.id) {
        await updateRule(form.formData.id, formData);
      } else {
        await createRule(formData);
      }
      setShowForm(false);
      form.resetToBlank();
      await loadRules();
    },
  });

  // Local state
  const [showForm, setShowForm] = useState(false);
  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const [importing, setImporting] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<string>('ALL');
  const [sortBy, setSortBy] = useState<'name' | 'type' | 'severity' | 'order'>('name');

  // Track if initial load has happened
  const initialLoadRef = React.useRef(false);

  // Load rules on mount (only once per tenant/datasource change)
  useEffect(() => {
    if (tenant && datasource) {
      // Reset initial load flag when tenant/datasource changes
      initialLoadRef.current = false;
    }
  }, [tenant?.id, datasource?.id]);

  useEffect(() => {
    if (tenant && datasource && !initialLoadRef.current) {
      initialLoadRef.current = true;
      loadRules();
    }
  }, [tenant, datasource]); // Intentionally exclude loadRules to prevent infinite loop

  /**
   * Show toast notification
   */
  const showToast = useCallback((type: 'success' | 'error', message: string) => {
    setToast({ type, message });
    setTimeout(() => setToast(null), 3000);
  }, []);

  /**
   * Handle new rule button
   */
  const handleNewRule = useCallback(() => {
    form.resetToBlank();
    setShowForm(true);
  }, [form]);

  /**
   * Handle edit rule
   */
  const handleEditRule = useCallback(
    (rule: any) => {
      form.setFormData({
        id: rule.id,
        name: rule.name,
        description: rule.description || '',
        ruleType: rule.ruleType,
        accountTypes: rule.accountTypes || rule.scope || ['ALL_ACCOUNTS'],
        severity: rule.severity || 'BLOCK',
        isActive: rule.isActive !== false,
        evaluationOrder: rule.evaluationOrder || 100,
        allowOverride: rule.allowOverride || false,
        requiredAuthority: rule.requiredAuthority,
        parameters: rule.parameters || {},
      });
      setShowForm(true);
    },
    [form]
  );

  /**
   * Handle form submission
   */
  const handleFormSubmit = useCallback(
    async (e?: React.FormEvent) => {
      e?.preventDefault();
      await form.handleSubmit(e);
    },
    [form]
  );

  /**
   * Handle delete rule
   */
  const handleDeleteRule = useCallback(
    async (ruleId: string) => {
      try {
        await deleteRule(ruleId);
        await loadRules();
      } catch (error) {
        // Error already handled by API hook
      }
    },
    [deleteRule, loadRules]
  );

  /**
   * Handle import sample rules
   */
  const handleImportRules = useCallback(async () => {
    if (!tenant || !datasource) {
      showToast('error', 'Select a tenant and datasource before importing');
      return;
    }

    if (!(await confirm({ title: 'Import sample rules', description: 'Import sample wealth-management validation rules into the current tenant/datasource?' })))
      return;

    setImporting(true);
    try {
      let created = 0;
      for (const r of WEALTH_VALIDATION_RULES as any[]) {
        const payload: any = {
          tenantId: tenant.id,
          datasourceId: datasource.id,
          id: r.id,
          name: r.name,
          description: r.description,
          ruleType: r.ruleType,
          scope: r.scope,
          severity: r.severity,
          isActive: r.isActive,
          effectiveFrom: r.effectiveFrom,
          ...(r.effectiveTo ? { effectiveTo: r.effectiveTo } : {}),
          frequency: r.frequency,
          evaluationOrder: r.evaluationOrder,
          overrideConditions: r.overrideConditions,
          requiredAuthority: r.requiredAuthority,
          parameters: r.parameters,
        };

        try {
          await createRule(payload as any);
          created += 1;
        } catch (err) {
          devLog('Import: rule create failed, trying update', { ruleId: r.id });
          try {
            await updateRule(r.id, payload as any);
          } catch (e) {
            // Ignore
          }
        }
      }

      showToast('success', `Imported ${created} rules`);
      await loadRules();
    } catch (e) {
      console.error('Import failed', e);
      showToast('error', 'Failed to import rules');
    } finally {
      setImporting(false);
    }
  }, [tenant, datasource, createRule, updateRule, loadRules, showToast]);

  /**
   * Get available types for filtering
   */
  const availableTypes = useMemo(() => {
    const types = new Set(rules.map((r) => r.ruleType));
    return Array.from(types).sort();
  }, [rules]);

  // Check if tenant/datasource selected
  if (!tenant || !datasource) {
    return (
      <div className="p-8 bg-gradient-to-br from-blue-50 to-blue-50/50 dark:from-blue-950/20 dark:to-blue-950/10 rounded-lg">
        <AlertCircle className="w-6 h-6 text-yellow-600 dark:text-yellow-400 mb-2" />
        <p className="text-gray-700 dark:text-gray-300">
          Please select a tenant and datasource to manage validation rules.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6 bg-white dark:bg-gray-900 rounded-lg shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Validation Rules Builder</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">Create and manage investment validation rules</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={handleNewRule}
            aria-label="Create new validation rule"
            title="Create new validation rule"
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
          >
            <Plus className="w-5 h-5" />
            New Rule
          </button>
          <button
            onClick={handleImportRules}
            disabled={importing}
            aria-label="Import sample wealth management validation rules"
            title="Import wealth management validation rules"
            className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors disabled:opacity-50"
          >
            {importing ? 'Importing...' : 'Import Wealth Rules'}
          </button>
        </div>
      </div>

      {/* Toast */}
      {toast && (
        <div
          className={`p-4 rounded-lg flex items-center gap-3 ${
            toast.type === 'success'
              ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
              : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
          }`}
        >
          {toast.type === 'success' ? (
            <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400" />
          ) : (
            <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
          )}
          <span className={toast.type === 'success' ? 'text-green-800 dark:text-green-200' : 'text-red-800 dark:text-red-200'}>
            {toast.message}
          </span>
        </div>
      )}

      {/* Error banner (if API error persists) */}
      {error && (
        <div className="p-4 rounded-lg flex items-center justify-between bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
          <div className="flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
            <span className="text-red-800 dark:text-red-200">{error.message}</span>
          </div>
          <button
            onClick={clearError}
            className="text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Search and Filter Controls */}
      <div className="flex gap-4 flex-wrap">
        <div className="flex-1 min-w-64">
          <input
            type="text"
            placeholder="Search rules by name or type..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            aria-label="Search rules"
          />
        </div>

        <select
          value={filterType}
          onChange={(e) => setFilterType(e.target.value)}
          className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          aria-label="Filter by rule type"
        >
          <option value="ALL">All Types</option>
          {availableTypes.map((type) => (
            <option key={type} value={type}>
              {type}
            </option>
          ))}
        </select>

        <select
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value as any)}
          className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          aria-label="Sort rules"
        >
          <option value="name">Sort by Name</option>
          <option value="type">Sort by Type</option>
          <option value="severity">Sort by Severity</option>
          <option value="order">Sort by Order</option>
        </select>
      </div>

      {/* Rules List */}
      <RulesList
        rules={rules}
        loading={loading}
        onEdit={handleEditRule}
        onDelete={handleDeleteRule}
        onCreateNew={handleNewRule}
        filterType={filterType === 'ALL' ? undefined : filterType}
        searchTerm={searchTerm}
        sortBy={sortBy}
      />

      {/* Form Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/50 dark:bg-black/70 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
            {/* Form Header */}
            <div className="sticky top-0 flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
                {form.formData.id ? 'Edit Rule' : 'Create New Rule'}
              </h2>
              <button
                onClick={() => {
                  setShowForm(false);
                  form.resetToBlank();
                }}
                aria-label="Close form"
                title="Close"
                className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              >
                <X className="w-6 h-6 text-gray-600 dark:text-gray-400" />
              </button>
            </div>

            {/* Form Content */}
            <form onSubmit={handleFormSubmit} className="p-6 space-y-4">
              {/* Display form-level errors */}
              {form.getAllErrors().length > 0 && (
                <div className="p-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
                  <p className="text-sm font-medium text-red-800 dark:text-red-200 mb-2">Please fix the following errors:</p>
                  <ul className="space-y-1">
                    {form.getAllErrors().map((error, i) => (
                      <li key={i} className="text-sm text-red-700 dark:text-red-300">
                        • {error}
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Rule Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Rule Name *
                </label>
                <input
                  type="text"
                  value={form.formData.name}
                  onChange={(e) => form.updateField('name', e.target.value)}
                  onBlur={() => form.touchField('name')}
                  className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                    form.hasFieldError('name')
                      ? 'border-red-500 dark:border-red-400'
                      : 'border-gray-300 dark:border-gray-600'
                  }`}
                  placeholder="e.g., Max Position Concentration"
                />
                {form.getFieldError('name') && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{form.getFieldError('name')}</p>
                )}
              </div>

              {/* Rule Type */}
              <div>
                <label htmlFor="ruleType" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Rule Type *
                </label>
                <select
                  id="ruleType"
                  value={form.formData.ruleType}
                  onChange={(e) => form.updateField('ruleType', e.target.value)}
                  onBlur={() => form.touchField('ruleType')}
                  className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                    form.hasFieldError('ruleType')
                      ? 'border-red-500 dark:border-red-400'
                      : 'border-gray-300 dark:border-gray-600'
                  }`}
                >
                  {getRuleTypeOptions().map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>

              {/* Description */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
                <textarea
                  value={form.formData.description}
                  onChange={(e) => form.updateField('description', e.target.value)}
                  onBlur={() => form.touchField('description')}
                  className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                    form.hasFieldError('description')
                      ? 'border-red-500 dark:border-red-400'
                      : 'border-gray-300 dark:border-gray-600'
                  }`}
                  placeholder="Describe what this rule validates..."
                  rows={3}
                />
                {form.getFieldError('description') && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{form.getFieldError('description')}</p>
                )}
              </div>

              {/* Rule-specific parameters */}
              <div className="space-y-4 border-t border-gray-200 dark:border-gray-700 pt-4">
                <h3 className="text-lg font-medium text-gray-900 dark:text-white">Parameters</h3>
                {getParameterSchema(form.formData.ruleType) && (
                  <ParameterBuilder
                    schema={getParameterSchema(form.formData.ruleType)!}
                    parameters={form.formData.parameters}
                    onChange={(params) => form.updateField('parameters', params)}
                    showValidation={false}
                  />
                )}
              </div>

              {/* Account Types */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Account Types *</label>
                <div className="space-y-2">
                  {getAccountTypeOptions().map((option) => (
                    <label key={option.value} className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={form.formData.accountTypes.includes(option.value)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            form.updateField('accountTypes', [...form.formData.accountTypes, option.value]);
                          } else {
                            form.updateField(
                              'accountTypes',
                              form.formData.accountTypes.filter((at) => at !== option.value)
                            );
                          }
                          form.touchField('accountTypes');
                        }}
                        className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      <span className="text-sm text-gray-700 dark:text-gray-300">{option.label}</span>
                    </label>
                  ))}
                </div>
                {form.getFieldError('accountTypes') && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{form.getFieldError('accountTypes')}</p>
                )}
              </div>

              {/* Severity and Status */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label htmlFor="severity" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Severity
                  </label>
                  <select
                    id="severity"
                    value={form.formData.severity}
                    onChange={(e) => form.updateField('severity', e.target.value as any)}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="BLOCK">BLOCK</option>
                    <option value="WARNING">WARNING</option>
                    <option value="INFO">INFO</option>
                  </select>
                </div>
                <div>
                  <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                    <input
                      type="checkbox"
                      checked={form.formData.isActive}
                      onChange={(e) => form.updateField('isActive', e.target.checked)}
                      className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      aria-label="Active"
                    />
                    Active
                  </label>
                </div>
              </div>

              {/* Evaluation Order */}
              <div>
                <label htmlFor="evaluationOrder" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Evaluation Order (lower runs first) *
                </label>
                <input
                  id="evaluationOrder"
                  type="number"
                  value={form.formData.evaluationOrder}
                  onChange={(e) => form.updateField('evaluationOrder', parseInt(e.target.value) || 0)}
                  onBlur={() => form.touchField('evaluationOrder')}
                  className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                    form.hasFieldError('evaluationOrder')
                      ? 'border-red-500 dark:border-red-400'
                      : 'border-gray-300 dark:border-gray-600'
                  }`}
                  min="0"
                  max="1000"
                />
                {form.getFieldError('evaluationOrder') && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{form.getFieldError('evaluationOrder')}</p>
                )}
              </div>

              {/* Override Settings */}
              <div>
                <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  <input
                    type="checkbox"
                    checked={form.formData.allowOverride}
                    onChange={(e) => form.updateField('allowOverride', e.target.checked)}
                    className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  />
                  Allow Override
                </label>
              </div>

              {form.formData.allowOverride && (
                <div>
                  <label htmlFor="requiredAuthority" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Required Authority
                  </label>
                  <select
                    id="requiredAuthority"
                    value={form.formData.requiredAuthority || ''}
                    onChange={(e) => form.updateField('requiredAuthority', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">Select authority level</option>
                    <option value="ADVISOR">Advisor</option>
                    <option value="SUPERVISOR">Supervisor</option>
                    <option value="COMPLIANCE">Compliance</option>
                    <option value="EXECUTIVE">Executive</option>
                  </select>
                </div>
              )}

              {/* Form Footer */}
              <div className="sticky bottom-0 flex items-center justify-end gap-3 p-6 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 -m-6 mt-6">
                <button
                  type="button"
                  onClick={() => {
                    setShowForm(false);
                    form.resetToBlank();
                  }}
                  className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={form.isSubmitting || saving}
                  className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Save className="w-5 h-5" />
                  {form.isSubmitting || saving ? 'Saving...' : 'Save Rule'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default ValidationRulesBuilderPage;
