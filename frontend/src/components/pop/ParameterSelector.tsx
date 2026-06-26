import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import { devError } from '../../utils/devLogger';

interface DynamicParameter {
  name: string;
  type: 'dimension' | 'measure' | 'filter' | 'time_range';
  value?: any;
  defaultValue?: any;
  required: boolean;
  options?: string[];
  description: string;
  source?: string;
}

interface ParameterSelectorProps {
  parameters: DynamicParameter[];
  onParameterChange: (name: string, value: any) => void;
  onValidationChange?: (isValid: boolean) => void;
  className?: string;
}

export const ParameterSelector: React.FC<ParameterSelectorProps> = ({
  parameters,
  onParameterChange,
  onValidationChange,
  className = ''
}) => {
  const [values, setValues] = useState<Record<string, any>>({});
  const [availableOptions, setAvailableOptions] = useState<Record<string, string[]>>({});
  const [loading, setLoading] = useState<Record<string, boolean>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Initialize values with defaults
  useEffect(() => {
    const initialValues: Record<string, any> = {};
    parameters.forEach(param => {
      if (param.defaultValue !== undefined) {
        initialValues[param.name] = param.defaultValue;
      }
    });
    setValues(initialValues);
  }, [parameters]);

  // Fetch available options for parameters that need them
  const fetchAvailableOptions = useCallback(async (param: DynamicParameter) => {
    if (!param.source && !param.options) return;

    setLoading(prev => ({ ...prev, [param.name]: true }));

    try {
      let options: string[] = [];

      if (param.options) {
        // Use predefined options
        options = param.options;
      } else if (param.source) {
        // Fetch from API based on source
        const response = await axios.get(`/api/parameters/${param.type}/${param.name}/values`);
        options = response.data.values || [];
      }

      setAvailableOptions(prev => ({ ...prev, [param.name]: options }));
    } catch (error) {
      devError(`Failed to fetch options for ${param.name}:`, error);
      setErrors(prev => ({
        ...prev,
        [param.name]: 'Failed to load options'
      }));
    } finally {
      setLoading(prev => ({ ...prev, [param.name]: false }));
    }
  }, []);

  // Load options when parameters change
  useEffect(() => {
    parameters.forEach(param => {
      if (param.type === 'dimension' || param.type === 'time_range') {
        fetchAvailableOptions(param);
      }
    });
  }, [parameters, fetchAvailableOptions]);

  // Handle parameter value change
  const handleValueChange = (paramName: string, value: any) => {
    const newValues = { ...values, [paramName]: value };
    setValues(newValues);
    onParameterChange(paramName, value);

    // Clear any existing errors
    if (errors[paramName]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[paramName];
        return newErrors;
      });
    }
  };

  // Validate parameters
  const validateParameters = useCallback(() => {
    const newErrors: Record<string, string> = {};
    let isValid = true;

    parameters.forEach(param => {
      if (param.required && (values[param.name] === undefined || values[param.name] === '')) {
        newErrors[param.name] = `${param.name} is required`;
        isValid = false;
      }
    });

    setErrors(newErrors);
    onValidationChange?.(isValid);
    return isValid;
  }, [parameters, values, onValidationChange]);

  // Validate when values change
  useEffect(() => {
    validateParameters();
  }, [validateParameters]);

  // Render parameter input based on type
  const renderParameterInput = (param: DynamicParameter) => {
    const value = values[param.name];
    const error = errors[param.name];
    const isLoadingOptions = loading[param.name];
    const options = availableOptions[param.name] || param.options || [];

    switch (param.type) {
      case 'dimension':
      case 'time_range':
        return (
          <div key={param.name} className="parameter-group">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {param.name}
              {param.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <select
              value={value || ''}
              onChange={(e) => handleValueChange(param.name, e.target.value)}
              className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                error ? 'border-red-500' : 'border-gray-300'
              }`}
              disabled={isLoadingOptions}
              aria-label={param.description || param.name}
            >
              <option value="">
                {isLoadingOptions ? 'Loading...' : `Select ${param.name}`}
              </option>
              {options.map(option => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
            {param.description && (
              <p className="text-xs text-gray-500 mt-1">{param.description}</p>
            )}
            {error && (
              <p className="text-xs text-red-600 mt-1">{error}</p>
            )}
          </div>
        );

      case 'filter':
        if (param.name.includes('only')) {
          // Boolean filter
          return (
            <div key={param.name} className="parameter-group">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={value || false}
                  onChange={(e) => handleValueChange(param.name, e.target.checked)}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="ml-2 text-sm text-gray-700">
                  {param.description || param.name}
                </span>
              </label>
            </div>
          );
        }
        // String filter
        return (
          <div key={param.name} className="parameter-group">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {param.name}
              {param.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <input
              type="text"
              value={value || ''}
              onChange={(e) => handleValueChange(param.name, e.target.value)}
              placeholder={param.description}
              className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                error ? 'border-red-500' : 'border-gray-300'
              }`}
            />
            {error && (
              <p className="text-xs text-red-600 mt-1">{error}</p>
            )}
          </div>
        );

      default:
        return (
          <div key={param.name} className="parameter-group">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {param.name} ({param.type})
              {param.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <input
              type="text"
              value={value || ''}
              onChange={(e) => handleValueChange(param.name, e.target.value)}
              placeholder={param.description}
              className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                error ? 'border-red-500' : 'border-gray-300'
              }`}
            />
            {error && (
              <p className="text-xs text-red-600 mt-1">{error}</p>
            )}
          </div>
        );
    }
  };

  return (
    <div className={`parameter-selector space-y-4 ${className}`}>
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold text-gray-900">Dynamic Parameters</h3>
        <div className="text-sm text-gray-500">
          {Object.keys(errors).length === 0 ? (
            <span className="text-green-600">✓ All parameters valid</span>
          ) : (
            <span className="text-red-600">⚠ Some parameters invalid</span>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {parameters.map(renderParameterInput)}
      </div>

      {/* Parameter Summary */}
      <div className="bg-gray-50 p-4 rounded-lg">
        <h4 className="text-sm font-medium text-gray-700 mb-2">Current Parameter Values:</h4>
        <div className="text-xs text-gray-600 space-y-1">
          {Object.entries(values).map(([key, value]) => (
            <div key={key} className="flex justify-between">
              <span className="font-medium">{key}:</span>
              <span>{String(value)}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default ParameterSelector;
