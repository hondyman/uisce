import React, { useState } from 'react';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import { Plus, Trash2, Save, X } from 'lucide-react';
import ParameterBuilder from './ParameterBuilder';
import { getParameterSchema, getAvailableRuleTypes, validateParameters } from '../lib/parameterSchemas';

interface ReportConfig {
  id?: string;
  name: string;
  description: string;
  reportType: string;
  parameters: Record<string, any>;
  sections: ReportSection[];
  enabled: boolean;
}

interface ReportSection {
  id: string;
  name: string;
  entityType: string;
  filterExpression: string;
}

interface ReportBuilderUIProps {
  onSave?: (config: ReportConfig) => void;
  onDelete?: (id: string) => void;
  initialConfig?: ReportConfig;
}

/**
 * ReportBuilderUI
 * 
 * Schema-driven report configuration builder using unified ParameterBuilder component.
 * Allows users to:
 * - Select report type (determines available parameters)
 * - Configure parameters via ParameterBuilder (8 field types, validation)
 * - Add report sections (entities to include)
 * - Save/delete report configurations
 * 
 * Integration: Uses ParameterBuilder for consistent parameter UI/UX across app
 */
export const ReportBuilderUI: React.FC<ReportBuilderUIProps> = ({
  onSave,
  onDelete,
  initialConfig,
}) => {
  const [formData, setFormData] = useState<ReportConfig>(
    initialConfig || {
      name: '',
      description: '',
      reportType: 'CONCENTRATION',
      parameters: {},
      sections: [],
      enabled: true,
    }
  );

  const [newSection, setNewSection] = useState<ReportSection>({
    id: '',
    name: '',
    entityType: '',
    filterExpression: '',
  });

  const [showSectionForm, setShowSectionForm] = useState(false);
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});
  const [showValidation, setShowValidation] = useState(false);

  // Get available report types
  const reportTypes = getAvailableRuleTypes();

  // Handle report type change - reset parameters when type changes
  const handleReportTypeChange = (newType: string) => {
    setFormData({
      ...formData,
      reportType: newType,
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

  // Add new report section
  const handleAddSection = () => {
    if (!newSection.name || !newSection.entityType) {
      return;
    }

    const section: ReportSection = {
      ...newSection,
      id: `section_${Date.now()}`,
    };

    setFormData({
      ...formData,
      sections: [...formData.sections, section],
    });

    setNewSection({
      id: '',
      name: '',
      entityType: '',
      filterExpression: '',
    });
    setShowSectionForm(false);
  };

  // Remove report section
  const handleRemoveSection = (sectionId: string) => {
    setFormData({
      ...formData,
      sections: formData.sections.filter((s) => s.id !== sectionId),
    });
  };

  // Validate and save
  const handleSave = () => {
    const schema = getParameterSchema(formData.reportType);
    if (!schema) {
      setValidationErrors({ base: 'Invalid report type selected' });
      return;
    }

    // Validate parameters
    const errors = validateParameters(formData.reportType, formData.parameters);
    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors);
      setShowValidation(true);
      return;
    }

    // Validate basic fields
    if (!formData.name.trim()) {
      setValidationErrors({ name: 'Report name is required' });
      setShowValidation(true);
      return;
    }

    if (formData.sections.length === 0) {
      setValidationErrors({ sections: 'At least one section is required' });
      setShowValidation(true);
      return;
    }

    // All validations passed
    setShowValidation(false);
    onSave?.(formData);
  };

  const handleDelete = () => {
    const confirm = useConfirm();
    const notification = useNotification();
    (async () => {
      if (formData.id && (await confirm({ title: 'Delete report config', description: 'Delete this report configuration?' }))) {
        onDelete?.(formData.id);
        notification.success('Report configuration deleted');
      }
    })();
  };

  const schema = getParameterSchema(formData.reportType);

  return (
    <div className="bg-white dark:bg-slate-900 rounded-lg p-6 shadow-lg border dark:border-slate-700">
      <h2 className="text-2xl font-bold mb-6 text-gray-900 dark:text-white">Report Builder</h2>

      {/* Basic Information */}
      <div className="space-y-4 mb-6">
        <div>
          <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
            Report Name *
          </label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            placeholder="e.g., Q4 Portfolio Concentration Report"
            className="w-full px-4 py-2 border dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 outline-none"
          />
          {validationErrors.name && showValidation && (
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{validationErrors.name}</p>
          )}
        </div>

        <div>
          <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
            Description
          </label>
          <textarea
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            placeholder="Optional: Describe the purpose of this report"
            rows={2}
            className="w-full px-4 py-2 border dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 outline-none"
          />
        </div>

        <div className="flex items-center gap-4">
          <div className="flex-1">
            <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
              Report Type *
            </label>
            <select
              value={formData.reportType}
              onChange={(e) => handleReportTypeChange(e.target.value)}
              aria-label="Report Type"
              className="w-full px-4 py-2 border dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 outline-none"
            >
              {reportTypes.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
          </div>

          <div className="flex items-end gap-2 pt-6">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                className="w-4 h-4 rounded border dark:border-slate-600 accent-blue-600"
              />
              <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">Enabled</span>
            </label>
          </div>
        </div>
      </div>

      {/* Parameter Configuration using ParameterBuilder */}
      <div className="mb-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Report Configuration
        </h3>
        <div className="p-4 bg-gray-50 dark:bg-slate-800 rounded-lg border dark:border-slate-700">
          {schema ? (
            <ParameterBuilder
              schema={schema}
              parameters={formData.parameters}
              onChange={handleParametersChange}
              errors={validationErrors}
              showValidation={showValidation}
            />
          ) : (
            <p className="text-gray-600 dark:text-gray-400">Select a report type to configure parameters</p>
          )}
        </div>
      </div>

      {/* Report Sections */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Report Sections</h3>
          <button
            onClick={() => setShowSectionForm(!showSectionForm)}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold transition-colors"
          >
            <Plus size={18} />
            Add Section
          </button>
        </div>

        {/* Section Form */}
        {showSectionForm && (
          <div className="mb-4 p-4 bg-blue-50 dark:bg-slate-800 border border-blue-200 dark:border-slate-700 rounded-lg">
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
                  Section Name *
                </label>
                <input
                  type="text"
                  value={newSection.name}
                  onChange={(e) => setNewSection({ ...newSection, name: e.target.value })}
                  placeholder="e.g., Top 10 Holdings"
                  className="w-full px-3 py-2 border dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 outline-none"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
                  Entity Type *
                </label>
                <input
                  type="text"
                  value={newSection.entityType}
                  onChange={(e) => setNewSection({ ...newSection, entityType: e.target.value })}
                  placeholder="e.g., SecurityHolding"
                  className="w-full px-3 py-2 border dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 outline-none"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-1">
                  Filter Expression
                </label>
                <textarea
                  value={newSection.filterExpression}
                  onChange={(e) => setNewSection({ ...newSection, filterExpression: e.target.value })}
                  placeholder="e.g., weight > 0.05"
                  rows={2}
                  className="w-full px-3 py-2 border dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 outline-none"
                />
              </div>

              <div className="flex gap-2">
                <button
                  onClick={handleAddSection}
                  className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-semibold transition-colors"
                >
                  <Plus size={16} />
                  Add
                </button>
                <button
                  onClick={() => setShowSectionForm(false)}
                  className="flex items-center gap-2 px-4 py-2 bg-gray-400 hover:bg-gray-500 text-white rounded-lg font-semibold transition-colors"
                >
                  <X size={16} />
                  Cancel
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Sections List */}
        <div className="space-y-2">
          {formData.sections.length > 0 ? (
            formData.sections.map((section) => (
              <div
                key={section.id}
                className="p-4 bg-gray-50 dark:bg-slate-800 border dark:border-slate-700 rounded-lg flex justify-between items-start"
              >
                <div className="flex-1">
                  <p className="font-semibold text-gray-900 dark:text-white">{section.name}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">{section.entityType}</p>
                  {section.filterExpression && (
                    <p className="text-xs text-gray-500 dark:text-gray-500 font-mono mt-1">
                      {section.filterExpression}
                    </p>
                  )}
                </div>
                <button
                  onClick={() => handleRemoveSection(section.id)}
                  className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 p-2"
                  aria-label="Delete section"
                >
                  <Trash2 size={18} />
                </button>
              </div>
            ))
          ) : (
            <p className="text-gray-500 dark:text-gray-400 text-center py-4">
              No sections added yet. Add a section to get started.
            </p>
          )}
        </div>

        {validationErrors.sections && showValidation && (
          <p className="mt-2 text-sm text-red-600 dark:text-red-400">{validationErrors.sections}</p>
        )}
      </div>

      {/* Error Display */}
      {validationErrors.base && showValidation && (
        <div className="mb-4 p-4 bg-red-100 dark:bg-red-900 border border-red-400 dark:border-red-700 rounded-lg">
          <p className="text-sm text-red-800 dark:text-red-200">{validationErrors.base}</p>
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-3 justify-end">
        {formData.id && (
          <button
            onClick={handleDelete}
            className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg font-semibold transition-colors"
          >
            <Trash2 size={18} />
            Delete
          </button>
        )}
        <button
          onClick={handleSave}
          className="flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold transition-colors"
        >
          <Save size={18} />
          Save Report
        </button>
      </div>
    </div>
  );
};

export default ReportBuilderUI;
