import React, { useState } from 'react';
import type { NodeProperty } from '../../types/nodeTypes';

interface NodeTypePreviewPanelProps {
  properties: NodeProperty[];
  typeName: string;
}

export const NodeTypePreviewPanel: React.FC<NodeTypePreviewPanelProps> = ({ properties, typeName }) => {
  const [previewValues, setPreviewValues] = useState<Record<string, any>>({});

  const handlePreviewChange = (propertyName: string, value: any) => {
    setPreviewValues((prev) => ({ ...prev, [propertyName]: value }));
  };

  const renderPropertyInput = (property: NodeProperty) => {
    const value = previewValues[property.name] ?? property.default_value ?? '';

    switch (property.input_type) {
      case 'textarea':
        return (
          <textarea
            value={value}
            onChange={(e) => handlePreviewChange(property.name, e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            rows={3}
            placeholder={property.nullable ? 'Optional' : 'Required'}
          />
        );

      case 'checkbox':
        return (
          <div className="flex items-center">
            <input
              type="checkbox"
              checked={!!value}
              onChange={(e) => handlePreviewChange(property.name, e.target.checked)}
              className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              aria-label={property.label}
            />
            <span className="ml-2 text-sm text-gray-600">
              {value ? 'Yes' : 'No'}
            </span>
          </div>
        );

      case 'select':
        return (
          <select
            value={value}
            onChange={(e) => handlePreviewChange(property.name, e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            aria-label={property.label}
          >
            <option value="">Select an option...</option>
            {property.options?.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        );

      case 'date-picker':
        return (
          <input
            type="date"
            value={value}
            onChange={(e) => handlePreviewChange(property.name, e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            aria-label={property.label}
          />
        );

      case 'number':
        return (
          <input
            type="number"
            value={value}
            onChange={(e) => handlePreviewChange(property.name, e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder={property.nullable ? 'Optional' : 'Required'}
            min={property.validation?.min}
            max={property.validation?.max}
            step={property.data_type === 'float' ? '0.01' : '1'}
          />
        );

      case 'json-editor':
        return (
          <textarea
            value={value}
            onChange={(e) => handlePreviewChange(property.name, e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
            rows={5}
            placeholder='{"key": "value"}'
          />
        );

      case 'text':
      default:
        return (
          <input
            type="text"
            value={value}
            onChange={(e) => handlePreviewChange(property.name, e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder={property.nullable ? 'Optional' : 'Required'}
            minLength={property.validation?.minLength}
            maxLength={property.validation?.maxLength}
          />
        );
    }
  };

  if (properties.length === 0) {
    return (
      <div className="bg-gray-50 border border-gray-200 rounded-lg p-8 text-center">
        <p className="text-gray-600">No properties to preview.</p>
        <p className="text-sm text-gray-500 mt-1">Add properties to see how they will appear in the UI.</p>
      </div>
    );
  }

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-6">
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Form Preview</h3>
        <p className="text-sm text-gray-600">
          This is how the {typeName} form will appear to users
        </p>
      </div>

      <div className="space-y-4">
        {properties.map((property) => (
          <div key={property.name} className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">
              {property.label}
              {!property.nullable && <span className="text-red-500 ml-1">*</span>}
            </label>
            {renderPropertyInput(property)}
            {property.format && (
              <p className="text-xs text-gray-500">Format: {property.format}</p>
            )}
            {property.validation && Object.keys(property.validation).length > 0 && (
              <p className="text-xs text-gray-500">
                Validation: {JSON.stringify(property.validation)}
              </p>
            )}
          </div>
        ))}
      </div>

      <div className="mt-6 pt-6 border-t border-gray-200">
        <h4 className="text-sm font-semibold text-gray-700 mb-2">Current Values</h4>
        <pre className="bg-gray-50 border border-gray-200 rounded p-3 text-xs overflow-x-auto">
          {JSON.stringify(previewValues, null, 2)}
        </pre>
      </div>
    </div>
  );
};
