/**
 * ParameterBuilder Component
 * Reusable, schema-driven parameter input builder
 * Used by: ValidationRulesBuilderPage, ReportBuilderUI, RuleBuilder, etc.
 */

import React, { useMemo } from 'react';
import {
  ParameterField,
  ParameterSchema,
  validateParameters,
  normalizeParameterValue,
  denormalizeParameterValue,
} from '../lib/parameterSchemas';

interface ParameterBuilderProps {
  /** Schema defining what parameters to show */
  schema: ParameterSchema;

  /** Current parameter values */
  parameters: Record<string, any>;

  /** Callback when parameters change */
  onChange: (parameters: Record<string, any>) => void;

  /** Field-level errors (e.g., from validation) */
  errors?: Record<string, string>;

  /** Whether to show validation errors */
  showValidation?: boolean;

  /** Custom class for the container */
  className?: string;

  /** Custom class for each field */
  fieldClassName?: string;
}

/**
 * Single parameter field renderer
 */
const ParameterFieldInput: React.FC<{
  field: ParameterField;
  value: any;
  onChange: (value: any) => void;
  error?: string;
}> = ({ field, value, onChange, error }) => {
  const displayValue = normalizeParameterValue(field, value);

  const handleChange = (newValue: any) => {
    const denormalized = denormalizeParameterValue(field, newValue);
    onChange(denormalized);
  };

  switch (field.type) {
    case 'text':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {field.label}
            {field.required && <span className="text-red-500">*</span>}
          </label>
          <input
            type="text"
            value={displayValue}
            onChange={(e) => handleChange(e.target.value)}
            placeholder={field.placeholder}
            className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 ${
              error
                ? 'border-red-500 dark:border-red-500'
                : 'border-gray-300 dark:border-gray-600'
            }`}
          />
          {field.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{field.description}</p>
          )}
          {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
        </div>
      );

    case 'number':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {field.label}
            {field.required && <span className="text-red-500">*</span>}
          </label>
          <input
            type="number"
            value={displayValue === '' || displayValue === null ? '' : displayValue}
            onChange={(e) => handleChange(e.target.value)}
            placeholder={field.placeholder}
            min={field.min}
            max={field.max}
            step={field.step}
            className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 ${
              error
                ? 'border-red-500 dark:border-red-500'
                : 'border-gray-300 dark:border-gray-600'
            }`}
          />
          {field.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{field.description}</p>
          )}
          {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
        </div>
      );

    case 'checkbox':
      return (
        <div>
          <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
            <input
              type="checkbox"
              checked={displayValue || false}
              onChange={(e) => handleChange(e.target.checked)}
              className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            {field.label}
            {field.required && <span className="text-red-500">*</span>}
          </label>
          {field.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 ml-6">{field.description}</p>
          )}
          {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
        </div>
      );

    case 'select':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {field.label}
            {field.required && <span className="text-red-500">*</span>}
          </label>
          <select
            title={`Select ${field.label}`}
            value={displayValue || ''}
            onChange={(e) => handleChange(e.target.value)}
            className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 ${
              error
                ? 'border-red-500 dark:border-red-500'
                : 'border-gray-300 dark:border-gray-600'
            }`}
          >
            <option value="">Select {field.label.toLowerCase()}</option>
            {field.options?.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          {field.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{field.description}</p>
          )}
          {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
        </div>
      );

    case 'multiselect':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {field.label}
            {field.required && <span className="text-red-500">*</span>}
          </label>
          <div className="space-y-2">
            {field.options?.map((option) => (
              <label key={option.value} className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={(Array.isArray(value) ? value : []).includes(option.value)}
                  onChange={(e) => {
                    const arr = Array.isArray(value) ? [...value] : [];
                    if (e.target.checked) {
                      arr.push(option.value);
                    } else {
                      arr.splice(arr.indexOf(option.value), 1);
                    }
                    handleChange(arr);
                  }}
                  className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">{option.label}</span>
              </label>
            ))}
          </div>
          {field.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">{field.description}</p>
          )}
          {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
        </div>
      );

    case 'textarea':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {field.label}
            {field.required && <span className="text-red-500">*</span>}
          </label>
          <textarea
            value={displayValue}
            onChange={(e) => handleChange(e.target.value)}
            placeholder={field.placeholder}
            rows={field.rows || 3}
            className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 ${
              error
                ? 'border-red-500 dark:border-red-500'
                : 'border-gray-300 dark:border-gray-600'
            }`}
          />
          {field.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{field.description}</p>
          )}
          {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
        </div>
      );

    case 'slider':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {field.label}
            {field.required && <span className="text-red-500">*</span>}
            <span className="ml-2 text-blue-600 dark:text-blue-400 font-semibold">{displayValue || 0}</span>
          </label>
          <input
            type="range"
            value={displayValue || 0}
            onChange={(e) => handleChange(e.target.value)}
            min={field.min}
            max={field.max}
            step={field.step || 1}
            title={`Adjust ${field.label}`}
            className="w-full h-2 bg-gray-200 dark:bg-gray-600 rounded-lg appearance-none cursor-pointer accent-blue-600"
          />
          {field.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{field.description}</p>
          )}
          {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
        </div>
      );

    case 'comma-list':
      return (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {field.label}
            {field.required && <span className="text-red-500">*</span>}
          </label>
          <input
            type="text"
            value={displayValue}
            onChange={(e) => handleChange(e.target.value)}
            placeholder={field.placeholder}
            className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 ${
              error
                ? 'border-red-500 dark:border-red-500'
                : 'border-gray-300 dark:border-gray-600'
            }`}
          />
          {field.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{field.description}</p>
          )}
          {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
        </div>
      );

    default:
      return null;
  }
};

/**
 * Main ParameterBuilder component
 */
export const ParameterBuilder: React.FC<ParameterBuilderProps> = ({
  schema,
  parameters,
  onChange,
  errors = {},
  showValidation = false,
  className = '',
  fieldClassName = '',
}) => {
  const validationErrors = useMemo(() => {
    if (!showValidation) return {};
    return validateParameters(schema.ruleType, parameters);
  }, [schema.ruleType, parameters, showValidation]);

  const allErrors = { ...validationErrors, ...errors };

  const handleParameterChange = (fieldName: string, value: any) => {
    onChange({
      ...parameters,
      [fieldName]: value,
    });
  };

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Schema description */}
      <div className="bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
        <p className="text-sm text-blue-900 dark:text-blue-200">
          <strong>{schema.name}:</strong> {schema.description}
        </p>
      </div>

      {/* Parameter fields */}
      <div className={`space-y-4 ${fieldClassName}`}>
        {schema.fields.map((field) => (
          <ParameterFieldInput
            key={field.name}
            field={field}
            value={parameters[field.name]}
            onChange={(value) => handleParameterChange(field.name, value)}
            error={allErrors[field.name]}
          />
        ))}
      </div>

      {/* No fields message */}
      {schema.fields.length === 0 && (
        <p className="text-sm text-gray-500 dark:text-gray-400 italic">No parameters required for this rule type.</p>
      )}
    </div>
  );
};

export default ParameterBuilder;
