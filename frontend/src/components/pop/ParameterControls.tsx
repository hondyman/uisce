import React, { useState, useEffect } from 'react';
import { DynamicParameter } from '../../types/dynamic';

interface ParameterControlsProps {
  parameters: DynamicParameter[];
  onParameterChange: (name: string, value: any) => void;
  onValidationChange?: (isValid: boolean) => void;
  className?: string;
}

export const ParameterControls: React.FC<ParameterControlsProps> = ({
  parameters,
  onParameterChange,
  onValidationChange,
  className = ''
}) => {
  const [values, setValues] = useState<Record<string, any>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});

  // Initialize values with defaults
  useEffect(() => {
    const initialValues: Record<string, any> = {};
    parameters.forEach(param => {
      initialValues[param.name] = param.value !== undefined ? param.value : param.defaultValue;
    });
    setValues(initialValues);
  }, [parameters]);

  // Validate all parameters
  useEffect(() => {
    const newErrors: Record<string, string> = {};
    let isValid = true;

    parameters.forEach(param => {
      const value = values[param.name];
      const error = validateParameter(param, value);
      if (error) {
        newErrors[param.name] = error;
        isValid = false;
      }
    });

    setErrors(newErrors);
    onValidationChange?.(isValid);
  }, [values, parameters, onValidationChange]);

  const validateParameter = (param: DynamicParameter, value: any): string | null => {
    if (param.required && (value === undefined || value === null || value === '')) {
      return `${param.name} is required`;
    }

    if (param.validation) {
      const { min, max, pattern, custom } = param.validation;

      if (typeof value === 'number') {
        if (min !== undefined && value < min) {
          return `Value must be at least ${min}`;
        }
        if (max !== undefined && value > max) {
          return `Value must be at most ${max}`;
        }
      }

      if (typeof value === 'string' && pattern) {
        const regex = new RegExp(pattern);
        if (!regex.test(value)) {
          return `Value must match pattern: ${pattern}`;
        }
      }

      if (custom && !custom(value)) {
        return 'Value failed custom validation';
      }
    }

    return null;
  };

  const handleValueChange = (name: string, value: any) => {
    const newValues = { ...values, [name]: value };
    setValues(newValues);
    setTouched({ ...touched, [name]: true });
    onParameterChange(name, value);
  };

  const renderParameterInput = (param: DynamicParameter) => {
    const value = values[param.name];
    const error = errors[param.name];
    const isTouched = touched[param.name];
    const showError = isTouched && error;

    const baseProps = {
      id: param.name,
      value: value || '',
      onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
        let newValue: any = e.target.value;
        if (param.type === 'number') {
          newValue = e.target.value === '' ? undefined : Number(e.target.value);
        } else if (param.type === 'boolean' && 'checked' in e.target) {
          newValue = (e.target as HTMLInputElement).checked;
        }
        handleValueChange(param.name, newValue);
      },
      className: `w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 ${
        showError ? 'border-red-500' : 'border-gray-300'
      }`,
      placeholder: param.description
    };

    switch (param.type) {
      case 'dimension':
      case 'filter':
        if (param.options && param.options.length > 0) {
          return (
            <select {...baseProps} onChange={(e) => handleValueChange(param.name, e.target.value)}>
              <option value="">Select {param.name}</option>
              {param.options.map(option => (
                <option key={option} value={option}>{option}</option>
              ))}
            </select>
          );
        }
        return <input {...baseProps} type="text" />;

      case 'measure':
        return <input {...baseProps} type="text" />;

      case 'time_range':
        return (
          <div className="flex space-x-2">
            <input
              type="date"
              value={value?.start || ''}
              onChange={(e) => handleValueChange(param.name, { ...value, start: e.target.value })}
              className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              aria-label={`${param.name} start date`}
              placeholder="Start date"
            />
            <input
              type="date"
              value={value?.end || ''}
              onChange={(e) => handleValueChange(param.name, { ...value, end: e.target.value })}
              className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              aria-label={`${param.name} end date`}
              placeholder="End date"
            />
          </div>
        );

      case 'number':
        return <input {...baseProps} type="number" />;

      case 'boolean':
        return (
          <input
            type="checkbox"
            id={param.name}
            checked={value || false}
            onChange={(e) => handleValueChange(param.name, e.target.checked)}
            className="w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500"
            aria-label={param.description}
          />
        );

      case 'string':
      default:
        return <input {...baseProps} type="text" />;
    }
  };

  return (
    <div className={`space-y-4 ${className}`}>
      <h3 className="text-lg font-semibold text-gray-800">Dynamic Parameters</h3>

      {parameters.length === 0 ? (
        <p className="text-gray-500">No parameters available</p>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {parameters.map(param => (
            <div key={param.name} className="space-y-2">
              <label htmlFor={param.name} className="block text-sm font-medium text-gray-700">
                {param.name}
                {param.required && <span className="text-red-500 ml-1">*</span>}
              </label>

              {renderParameterInput(param)}

              {param.description && (
                <p className="text-xs text-gray-500">{param.description}</p>
              )}

              {touched[param.name] && errors[param.name] && (
                <p className="text-xs text-red-600">{errors[param.name]}</p>
              )}

              {param.defaultValue !== undefined && (
                <p className="text-xs text-gray-400">
                  Default: {typeof param.defaultValue === 'object'
                    ? JSON.stringify(param.defaultValue)
                    : param.defaultValue}
                </p>
              )}
            </div>
          ))}
        </div>
      )}

      <div className="flex justify-end space-x-2 pt-4">
        <button
          onClick={() => {
            const defaults: Record<string, any> = {};
            parameters.forEach(param => {
              defaults[param.name] = param.defaultValue;
            });
            setValues(defaults);
            setTouched({});
            Object.entries(defaults).forEach(([name, value]) => {
              onParameterChange(name, value);
            });
          }}
          className="px-4 py-2 text-sm text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50"
        >
          Reset to Defaults
        </button>
      </div>
    </div>
  );
};
