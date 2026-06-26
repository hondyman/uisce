import React, { useState, useEffect } from 'react';
import { DATA_TYPE_OPTIONS, INPUT_TYPE_OPTIONS } from '../../types/nodeTypes';
import type { NodeProperty } from '../../types/nodeTypes';

interface PropertyFormModalProps {
  property: NodeProperty | null;
  existingPropertyNames: string[];
  onSave: (property: NodeProperty) => void;
  onClose: () => void;
}

export const PropertyFormModal: React.FC<PropertyFormModalProps> = ({
  property,
  existingPropertyNames,
  onSave,
  onClose,
}) => {
  const isEditing = !!property;
  const [formData, setFormData] = useState<NodeProperty>({
    name: property?.name || '',
    label: property?.label || '',
    data_type: property?.data_type || 'string',
    nullable: property?.nullable ?? false,
    default_value: property?.default_value,
    input_type: property?.input_type || 'text',
    format: property?.format || '',
    validation: property?.validation || {},
    options: property?.options || [],
    order: property?.order || 0,
  });

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [optionsText, setOptionsText] = useState(property?.options?.join('\n') || '');

  useEffect(() => {
    // Update label automatically from name if label is empty
    if (!formData.label && formData.name) {
      const autoLabel = formData.name
        .replace(/_/g, ' ')
        .replace(/\b\w/g, (c) => c.toUpperCase());
      setFormData((prev) => ({ ...prev, label: autoLabel }));
    }
  }, [formData.name, formData.label]);

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Property name is required';
    } else if (!/^[a-z][a-z0-9_]*$/i.test(formData.name)) {
      newErrors.name = 'Property name must be alphanumeric with underscores, starting with a letter';
    } else if (!isEditing && existingPropertyNames.includes(formData.name)) {
      newErrors.name = 'A property with this name already exists';
    }

    if (!formData.label.trim()) {
      newErrors.label = 'Label is required';
    }

    if (formData.input_type === 'select' && (!formData.options || formData.options.length === 0)) {
      newErrors.options = 'Options are required for select input type';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    onSave(formData);
  };

  const handleChange = (field: keyof NodeProperty, value: any) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    // Clear error for this field
    if (errors[field]) {
      setErrors((prev) => {
        const newErrors = { ...prev };
        delete newErrors[field];
        return newErrors;
      });
    }
  };

  const handleOptionsChange = (text: string) => {
    setOptionsText(text);
    const options = text.split('\n').map((line) => line.trim()).filter((line) => line.length > 0);
    handleChange('options', options);
  };

  const handleValidationChange = (key: string, value: any) => {
    const newValidation = { ...formData.validation, [key]: value };
    handleChange('validation', newValidation);
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
          <h3 className="text-xl font-bold text-gray-900">
            {isEditing ? 'Edit Property' : 'Add Property'}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
            aria-label="Close modal"
          >
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          {/* Property Name */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Property Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => handleChange('name', e.target.value)}
              disabled={isEditing}
              className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${
                errors.name ? 'border-red-500' : 'border-gray-300'
              } ${isEditing ? 'bg-gray-100 cursor-not-allowed' : ''}`}
              placeholder="e.g., business_owner, created_date"
            />
            {errors.name && <p className="mt-1 text-sm text-red-600">{errors.name}</p>}
            <p className="mt-1 text-xs text-gray-500">
              Use lowercase letters, numbers, and underscores. This is the internal field name.
            </p>
          </div>

          {/* Label */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Display Label <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={formData.label}
              onChange={(e) => handleChange('label', e.target.value)}
              className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${
                errors.label ? 'border-red-500' : 'border-gray-300'
              }`}
              placeholder="e.g., Business Owner, Created Date"
            />
            {errors.label && <p className="mt-1 text-sm text-red-600">{errors.label}</p>}
          </div>

          {/* Data Type and Input Type */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="data_type" className="block text-sm font-medium text-gray-700 mb-2">
                Data Type <span className="text-red-500">*</span>
              </label>
              <select
                id="data_type"
                value={formData.data_type}
                onChange={(e) => handleChange('data_type', e.target.value as any)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                {DATA_TYPE_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label htmlFor="input_type" className="block text-sm font-medium text-gray-700 mb-2">
                Input Type <span className="text-red-500">*</span>
              </label>
              <select
                id="input_type"
                value={formData.input_type}
                onChange={(e) => handleChange('input_type', e.target.value as any)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                {INPUT_TYPE_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Nullable */}
          <div className="flex items-center">
            <input
              type="checkbox"
              id="nullable"
              checked={formData.nullable}
              onChange={(e) => handleChange('nullable', e.target.checked)}
              className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
            />
            <label htmlFor="nullable" className="ml-2 block text-sm text-gray-700">
              Nullable (allow empty/null values)
            </label>
          </div>

          {/* Default Value */}
          <div>
            <label htmlFor="default_value" className="block text-sm font-medium text-gray-700 mb-2">
              Default Value
            </label>
            {formData.data_type === 'boolean' ? (
              <select
                id="default_value"
                value={String(formData.default_value ?? '')}
                onChange={(e) => handleChange('default_value', e.target.value === '' ? undefined : e.target.value === 'true')}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">None</option>
                <option value="true">True</option>
                <option value="false">False</option>
              </select>
            ) : (
              <input
                id="default_value"
                type={formData.data_type === 'integer' || formData.data_type === 'float' ? 'number' : 'text'}
                value={formData.default_value ?? ''}
                onChange={(e) => handleChange('default_value', e.target.value || undefined)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                placeholder="Optional default value"
              />
            )}
          </div>

          {/* Format */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Format / Pattern
            </label>
            <input
              type="text"
              value={formData.format}
              onChange={(e) => handleChange('format', e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="e.g., YYYY-MM-DD, email, url"
            />
            <p className="mt-1 text-xs text-gray-500">
              Display format, validation pattern, or special format hint
            </p>
          </div>

          {/* Options (for select input type) */}
          {formData.input_type === 'select' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Options <span className="text-red-500">*</span>
              </label>
              <textarea
                value={optionsText}
                onChange={(e) => handleOptionsChange(e.target.value)}
                rows={5}
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm ${
                  errors.options ? 'border-red-500' : 'border-gray-300'
                }`}
                placeholder="Enter one option per line"
              />
              {errors.options && <p className="mt-1 text-sm text-red-600">{errors.options}</p>}
              <p className="mt-1 text-xs text-gray-500">
                One option per line. {formData.options?.length || 0} option(s) configured.
              </p>
            </div>
          )}

          {/* Validation Rules */}
          {(formData.data_type === 'string' || formData.data_type === 'text') && (
            <div className="border-t border-gray-200 pt-6">
              <h4 className="text-sm font-semibold text-gray-700 mb-3">Validation Rules</h4>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label htmlFor="minLength" className="block text-xs font-medium text-gray-600 mb-1">
                    Min Length
                  </label>
                  <input
                    id="minLength"
                    type="number"
                    value={formData.validation?.minLength ?? ''}
                    onChange={(e) => handleValidationChange('minLength', e.target.value ? parseInt(e.target.value) : undefined)}
                    className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                    min="0"
                    placeholder="Min"
                  />
                </div>
                <div>
                  <label htmlFor="maxLength" className="block text-xs font-medium text-gray-600 mb-1">
                    Max Length
                  </label>
                  <input
                    id="maxLength"
                    type="number"
                    value={formData.validation?.maxLength ?? ''}
                    onChange={(e) => handleValidationChange('maxLength', e.target.value ? parseInt(e.target.value) : undefined)}
                    className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                    min="0"
                    placeholder="Max"
                  />
                </div>
              </div>
            </div>
          )}

          {(formData.data_type === 'integer' || formData.data_type === 'float') && (
            <div className="border-t border-gray-200 pt-6">
              <h4 className="text-sm font-semibold text-gray-700 mb-3">Validation Rules</h4>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label htmlFor="minValue" className="block text-xs font-medium text-gray-600 mb-1">
                    Minimum Value
                  </label>
                  <input
                    id="minValue"
                    type="number"
                    value={formData.validation?.min ?? ''}
                    onChange={(e) => handleValidationChange('min', e.target.value ? parseFloat(e.target.value) : undefined)}
                    className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                    step={formData.data_type === 'float' ? '0.01' : '1'}
                    placeholder="Min"
                  />
                </div>
                <div>
                  <label htmlFor="maxValue" className="block text-xs font-medium text-gray-600 mb-1">
                    Maximum Value
                  </label>
                  <input
                    id="maxValue"
                    type="number"
                    value={formData.validation?.max ?? ''}
                    onChange={(e) => handleValidationChange('max', e.target.value ? parseFloat(e.target.value) : undefined)}
                    className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                    step={formData.data_type === 'float' ? '0.01' : '1'}
                    placeholder="Max"
                  />
                </div>
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center justify-end gap-3 border-t border-gray-200 pt-6">
            <button
              type="button"
              onClick={onClose}
              className="px-6 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              {isEditing ? 'Update' : 'Add'} Property
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
