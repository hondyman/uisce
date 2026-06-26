import React, { useState, useEffect } from 'react';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import { Plus, Trash2, Save, Eye, Download } from 'lucide-react';
import { ParameterBuilder } from './ParameterBuilder';
import { PARAMETER_SCHEMAS } from '../lib/parameterSchemas';

// ============================================================================
// TYPES
// ============================================================================

export interface ReportConfig {
  id?: string;
  householdId: string;
  reportName: string;
  description?: string;
  reportType: string; // 'summary', 'detailed', 'performance', 'allocation'
  parameters: Record<string, any>;
  semanticViewId?: string;
  enabled?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface Household {
  id: string;
  name: string;
  description?: string;
  householdType: string;
  status: string;
  createdAt?: string;
}

export interface SemanticView {
  id: string;
  name: string;
  description?: string;
  entity_count?: number;
}

export interface HouseholdReportBuilderProps {
  onSave?: (config: ReportConfig) => void;
  onDelete?: (id: string) => void;
  initialConfig?: ReportConfig;
  households?: Household[];
  semanticViews?: SemanticView[];
}

// ============================================================================
// COMPONENT
// ============================================================================

export const HouseholdReportBuilder: React.FC<HouseholdReportBuilderProps> = ({
  onSave,
  onDelete,
  initialConfig,
  households = [],
  semanticViews = [],
}) => {
  // ========================================================================
  // STATE
  // ========================================================================

  const [config, setConfig] = useState<ReportConfig>(
    initialConfig || {
      householdId: '',
      reportName: '',
      description: '',
      reportType: 'summary',
      parameters: {},
      semanticViewId: '',
      enabled: true,
    }
  );

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [showPreview, setShowPreview] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  // ========================================================================
  // HANDLERS
  // ========================================================================

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setConfig((prev) => ({
      ...prev,
      reportName: e.target.value,
    }));
    setErrors((prev) => ({ ...prev, reportName: '' }));
  };

  const handleDescriptionChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setConfig((prev) => ({
      ...prev,
      description: e.target.value,
    }));
  };

  const handleReportTypeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newType = e.target.value;
    setConfig((prev) => ({
      ...prev,
      reportType: newType,
      parameters: {}, // Reset parameters when type changes
    }));
    setErrors((prev) => ({ ...prev, reportType: '' }));
  };

  const handleHouseholdChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setConfig((prev) => ({
      ...prev,
      householdId: e.target.value,
    }));
    setErrors((prev) => ({ ...prev, householdId: '' }));
  };

  const handleSemanticViewChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setConfig((prev) => ({
      ...prev,
      semanticViewId: e.target.value,
    }));
  };

  const handleParameterChange = (updates: Record<string, any>) => {
    setConfig((prev) => ({
      ...prev,
      parameters: {
        ...prev.parameters,
        ...updates,
      },
    }));
  };

  const handleEnabledChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setConfig((prev) => ({
      ...prev,
      enabled: e.target.checked,
    }));
  };

  // ========================================================================
  // VALIDATION
  // ========================================================================

  const validateConfig = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!config.reportName.trim()) {
      newErrors.reportName = 'Report name is required';
    }

    if (!config.householdId) {
      newErrors.householdId = 'Household is required';
    }

    if (!config.reportType) {
      newErrors.reportType = 'Report type is required';
    }

    // Validate parameters
    const schema = PARAMETER_SCHEMAS[config.reportType];
    if (schema) {
      for (const field of schema.fields) {
        if (field.required && !config.parameters[field.name]) {
          newErrors[field.name] = `${field.label} is required`;
        }
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // ========================================================================
  // ACTIONS
  // ========================================================================

  const handleSave = async () => {
    if (!validateConfig()) {
      return;
    }

    setIsSaving(true);
    try {
      const configToSave: ReportConfig = {
        ...config,
        id: config.id || `report_${Date.now()}`,
        createdAt: config.createdAt || new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };

      onSave?.(configToSave);

      // Show success feedback
      setTimeout(() => {
        setIsSaving(false);
      }, 500);
    } catch (error) {
      console.error('Error saving report config:', error);
      setErrors((prev) => ({
        ...prev,
        save: 'Failed to save report configuration',
      }));
      setIsSaving(false);
    }
  };

  const handleDelete = () => {
    const confirm = useConfirm();
    const notification = useNotification();
    (async () => {
      if (config.id && (await confirm({ title: 'Delete report', description: 'Are you sure you want to delete this report?' }))) {
        onDelete?.(config.id);
        notification.success('Report removed');
      }
    })();
  };

  const handleGeneratePreview = () => {
    if (!validateConfig()) {
      return;
    }
    setShowPreview(true);
  };

  // ========================================================================
  // RENDER
  // ========================================================================

  const selectedHousehold = households.find((h) => h.id === config.householdId);
  const selectedSchema = PARAMETER_SCHEMAS[config.reportType];

  return (
    <div className="max-w-4xl mx-auto p-6 bg-white dark:bg-slate-900 rounded-lg shadow-lg">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-slate-900 dark:text-white mb-2">
          Household Report Builder
        </h1>
        <p className="text-slate-600 dark:text-slate-400">
          Configure and generate household reports with AI semantic cubes
        </p>
      </div>

      {/* Error Banner */}
      {errors.save && (
        <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p className="text-red-700 dark:text-red-200">{errors.save}</p>
        </div>
      )}

      {/* Main Form */}
      <div className="space-y-6">
        {/* Household Selection */}
        <div className="border-b border-slate-200 dark:border-slate-700 pb-6">
          <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
            Household <span className="text-red-500">*</span>
          </label>
          <select
            value={config.householdId}
            onChange={handleHouseholdChange}
            className={`w-full px-4 py-2 rounded-lg border dark:bg-slate-800 dark:text-white dark:border-slate-600 ${
              errors.householdId
                ? 'border-red-500 focus:ring-red-500'
                : 'border-slate-300 focus:ring-blue-500'
            } focus:ring-2 focus:outline-none`}
            aria-label="Select household"
          >
            <option value="">-- Select Household --</option>
            {households.map((h) => (
              <option key={h.id} value={h.id}>
                {h.name} ({h.householdType})
              </option>
            ))}
          </select>
          {errors.householdId && (
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.householdId}</p>
          )}
          {selectedHousehold && (
            <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
              {selectedHousehold.description}
            </p>
          )}
        </div>

        {/* Report Metadata */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Report Name */}
          <div>
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
              Report Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={config.reportName}
              onChange={handleNameChange}
              placeholder="e.g., Q4 2024 Holdings Summary"
              className={`w-full px-4 py-2 rounded-lg border dark:bg-slate-800 dark:text-white dark:border-slate-600 ${
                errors.reportName
                  ? 'border-red-500 focus:ring-red-500'
                  : 'border-slate-300 focus:ring-blue-500'
              } focus:ring-2 focus:outline-none`}
              aria-label="Report name"
            />
            {errors.reportName && (
              <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.reportName}</p>
            )}
          </div>

          {/* Report Type */}
          <div>
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
              Report Type <span className="text-red-500">*</span>
            </label>
            <select
              value={config.reportType}
              onChange={handleReportTypeChange}
              className={`w-full px-4 py-2 rounded-lg border dark:bg-slate-800 dark:text-white dark:border-slate-600 ${
                errors.reportType
                  ? 'border-red-500 focus:ring-red-500'
                  : 'border-slate-300 focus:ring-blue-500'
              } focus:ring-2 focus:outline-none`}
              aria-label="Report type"
            >
              <option value="summary">Summary (Executive Overview)</option>
              <option value="detailed">Detailed (Holdings Breakdown)</option>
              <option value="performance">Performance (Analysis)</option>
              <option value="allocation">Allocation (Breakdown)</option>
            </select>
            {errors.reportType && (
              <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.reportType}</p>
            )}
          </div>
        </div>

        {/* Description */}
        <div>
          <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
            Description
          </label>
          <textarea
            value={config.description || ''}
            onChange={handleDescriptionChange}
            placeholder="Additional notes about this report..."
            rows={3}
            className="w-full px-4 py-2 rounded-lg border border-slate-300 dark:bg-slate-800 dark:text-white dark:border-slate-600 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:outline-none"
            aria-label="Report description"
          />
        </div>

        {/* Semantic View Selection */}
        {semanticViews.length > 0 && (
          <div>
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
              Data Source (Semantic View)
            </label>
            <select
              value={config.semanticViewId || ''}
              onChange={handleSemanticViewChange}
              className="w-full px-4 py-2 rounded-lg border border-slate-300 dark:bg-slate-800 dark:text-white dark:border-slate-600 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:outline-none"
              aria-label="Select semantic view"
            >
              <option value="">-- Auto-detect --</option>
              {semanticViews.map((v) => (
                <option key={v.id} value={v.id}>
                  {v.name} ({v.entity_count || 0} entities)
                </option>
              ))}
            </select>
          </div>
        )}

        {/* Parameters Section */}
        {selectedSchema && (
          <div className="bg-slate-50 dark:bg-slate-800 p-6 rounded-lg border border-slate-200 dark:border-slate-700">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
              Report Parameters
            </h3>
            <ParameterBuilder
              parameters={config.parameters}
              schema={selectedSchema}
              onChange={handleParameterChange}
            />
          </div>
        )}

        {/* Status */}
        <div className="flex items-center space-x-3">
          <input
            type="checkbox"
            id="enabled"
            checked={config.enabled !== false}
            onChange={handleEnabledChange}
            className="w-4 h-4 rounded cursor-pointer"
            aria-label="Enable report"
          />
          <label htmlFor="enabled" className="text-sm text-slate-700 dark:text-slate-300 cursor-pointer">
            Enable this report
          </label>
        </div>

        {/* Action Buttons */}
        <div className="flex flex-wrap gap-3 pt-6 border-t border-slate-200 dark:border-slate-700">
          <button
            onClick={handleSave}
            disabled={isSaving}
            className="flex items-center space-x-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg font-medium transition-colors"
          >
            <Save size={18} />
            <span>{isSaving ? 'Saving...' : 'Save Report'}</span>
          </button>

          <button
            onClick={handleGeneratePreview}
            className="flex items-center space-x-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-medium transition-colors"
          >
            <Eye size={18} />
            <span>Preview</span>
          </button>

          {config.id && (
            <>
              <button
                onClick={() => handleDelete()}
                className="flex items-center space-x-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg font-medium transition-colors"
              >
                <Trash2 size={18} />
                <span>Delete</span>
              </button>

              <button
                className="flex items-center space-x-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded-lg font-medium transition-colors"
              >
                <Download size={18} />
                <span>Download PDF</span>
              </button>
            </>
          )}
        </div>
      </div>

      {/* Preview Modal */}
      {showPreview && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-slate-900 rounded-lg shadow-xl max-w-2xl w-full max-h-[80vh] overflow-y-auto">
            <div className="sticky top-0 bg-slate-100 dark:bg-slate-800 p-4 border-b border-slate-200 dark:border-slate-700 flex justify-between items-center">
              <h2 className="text-xl font-bold text-slate-900 dark:text-white">Report Preview</h2>
              <button
                onClick={() => setShowPreview(false)}
                className="text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white text-xl"
              >
                ✕
              </button>
            </div>

            <div className="p-6">
              <div className="space-y-4 text-slate-700 dark:text-slate-300">
                <div>
                  <p className="font-semibold text-slate-900 dark:text-white">Report Configuration:</p>
                  <pre className="mt-2 p-3 bg-slate-50 dark:bg-slate-800 rounded border border-slate-200 dark:border-slate-700 text-sm overflow-x-auto">
                    {JSON.stringify(config, null, 2)}
                  </pre>
                </div>

                <div className="text-sm text-slate-600 dark:text-slate-400">
                  <p>✓ Configuration is valid and ready to generate</p>
                  <p>✓ {config.parameters ? Object.keys(config.parameters).length : 0} parameters configured</p>
                  {selectedHousehold && <p>✓ Household: {selectedHousehold.name}</p>}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default HouseholdReportBuilder;
