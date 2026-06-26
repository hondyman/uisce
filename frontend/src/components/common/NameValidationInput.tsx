import React, { useState, useEffect } from 'react';
import { validateEntityName, getExampleNames, type NameValidationResult } from '../../utils/nameValidation';
import { IconCheck, IconAlertTriangle, IconX } from '@tabler/icons-react';
import IconInfoCircle from '@tabler/icons-react/dist/esm/icons/IconInfoCircle.mjs';

interface NameValidationInputProps {
  value: string;
  onChange: (value: string) => void;
  onValidationChange?: (validation: NameValidationResult) => void;
  type?: 'cube' | 'view' | 'measure' | 'dimension' | 'pre-aggregation';
  label?: string;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  className?: string;
  showExamples?: boolean;
}

const NameValidationInput: React.FC<NameValidationInputProps> = ({
  value,
  onChange,
  onValidationChange,
  type = 'cube',
  label,
  placeholder,
  required = false,
  disabled = false,
  className = '',
  showExamples = true
}) => {
  const [validation, setValidation] = useState<NameValidationResult>({ isValid: true, errors: [], warnings: [] });
  const [showTooltip, setShowTooltip] = useState(false);

  useEffect(() => {
    const result = validateEntityName(value, type);
    setValidation(result);
    onValidationChange?.(result);
  }, [value, type, onValidationChange]);

  const getValidationIcon = () => {
    if (!value) return null;
    
    if (validation.errors.length > 0) {
      return <IconX className="w-4 h-4 text-red-500" />;
    } else if (validation.warnings.length > 0) {
      return <IconAlertTriangle className="w-4 h-4 text-yellow-500" />;
    } else {
      return <IconCheck className="w-4 h-4 text-green-500" />;
    }
  };

  const getInputBorderColor = () => {
    if (!value) return 'border-gray-300 focus:border-blue-500';
    
    if (validation.errors.length > 0) {
      return 'border-red-300 focus:border-red-500';
    } else if (validation.warnings.length > 0) {
      return 'border-yellow-300 focus:border-yellow-500';
    } else {
      return 'border-green-300 focus:border-green-500';
    }
  };

  const applySuggestion = () => {
    if (validation.suggestions) {
      onChange(validation.suggestions);
    }
  };

  const examples = getExampleNames(type);

  return (
    <div className={`space-y-2 ${className}`}>
      {label && (
        <label className="block text-sm font-medium text-gray-700">
          {label} {required && <span className="text-red-500">*</span>}
        </label>
      )}
      
      <div className="relative">
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder || `Enter ${type} name...`}
          disabled={disabled}
          className={`
            w-full px-3 py-2 pr-10 border rounded-md shadow-sm 
            focus:outline-none focus:ring-1 focus:ring-blue-500
            ${getInputBorderColor()}
            ${disabled ? 'bg-gray-100 cursor-not-allowed' : 'bg-white'}
          `}
        />
        
        <div className="absolute inset-y-0 right-0 pr-3 flex items-center">
          {getValidationIcon()}
        </div>
      </div>

      {/* Validation Messages */}
      {value && (validation.errors.length > 0 || validation.warnings.length > 0) && (
        <div className="space-y-1">
          {validation.errors.map((error, index) => (
            <div key={`error-${index}`} className="flex items-start space-x-2 text-sm text-red-600">
              <IconX className="w-4 h-4 mt-0.5 flex-shrink-0" />
              <span>{error}</span>
            </div>
          ))}
          
          {validation.warnings.map((warning, index) => (
            <div key={`warning-${index}`} className="flex items-start space-x-2 text-sm text-yellow-600">
              <IconAlertTriangle className="w-4 h-4 mt-0.5 flex-shrink-0" />
              <span>{warning}</span>
            </div>
          ))}
          
          {validation.suggestions && (
            <button
              type="button"
              onClick={applySuggestion}
              className="flex items-center space-x-1 text-sm text-blue-600 hover:text-blue-800 underline"
            >
              <span>Apply suggestion: "{validation.suggestions}"</span>
            </button>
          )}
        </div>
      )}

      {/* Examples */}
      {showExamples && (
        <div className="relative">
          <button
            type="button"
            onClick={() => setShowTooltip(!showTooltip)}
            className="flex items-center space-x-1 text-sm text-gray-500 hover:text-gray-700"
          >
            <IconInfoCircle className="w-4 h-4" />
            <span>See examples</span>
          </button>
          
          {showTooltip && (
            <div className="absolute top-full left-0 z-10 mt-1 p-3 bg-white border border-gray-200 rounded-md shadow-lg min-w-max">
              <div className="text-sm">
                <div className="font-medium text-gray-700 mb-2">
                  Good examples for {type}s:
                </div>
                <div className="space-y-1">
                  {examples.map((example, index) => (
                    <button
                      key={index}
                      type="button"
                      onClick={() => {
                        onChange(example);
                        setShowTooltip(false);
                      }}
                      className="block w-full text-left px-2 py-1 text-gray-600 hover:bg-gray-100 rounded text-xs font-mono"
                    >
                      {example}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Naming Rules Help */}
      {showTooltip && (
        <div className="text-xs text-gray-500 space-y-1 bg-gray-50 p-2 rounded">
          <div className="font-medium">Naming Rules:</div>
          <ul className="list-disc list-inside space-y-0.5">
            <li>Must start with a letter</li>
            <li>Only letters, numbers, and underscores</li>
            <li>Cannot be a Python reserved keyword</li>
            <li>Use snake_case (recommended)</li>
          </ul>
        </div>
      )}
    </div>
  );
};

export default NameValidationInput;
