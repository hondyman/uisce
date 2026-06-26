import React, { useState } from 'react';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import { PropertyFormModal } from './PropertyFormModal';
import type { NodeProperty } from '../../types/nodeTypes';

interface PropertyListEditorProps {
  properties: NodeProperty[];
  onChange: (properties: NodeProperty[]) => void;
}

export const PropertyListEditor: React.FC<PropertyListEditorProps> = ({ properties, onChange }) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingProperty, setEditingProperty] = useState<NodeProperty | null>(null);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);

  const handleAdd = () => {
    setEditingProperty(null);
    setEditingIndex(null);
    setIsModalOpen(true);
  };

  const confirm = useConfirm();
  const notification = useNotification();

  const handleEdit = (property: NodeProperty, index: number) => {
    setEditingProperty(property);
    setEditingIndex(index);
    setIsModalOpen(true);
  };

  const handleDelete = async (index: number) => {
    if (!confirm) return; // safety (shouldn't happen)
    if (!notification) return;

    if (!(await confirm({ title: 'Delete property', description: 'Are you sure you want to delete this property?' }))) {
      return;
    }

    const newProperties = properties.filter((_, i) => i !== index);
    onChange(newProperties);
  };

  const handleSave = (property: NodeProperty) => {
    if (editingIndex !== null) {
      // Update existing property
      const newProperties = [...properties];
      newProperties[editingIndex] = property;
      onChange(newProperties);
    } else {
      // Add new property
      onChange([...properties, property]);
    }
    setIsModalOpen(false);
  };

  const handleMoveUp = (index: number) => {
    if (index === 0) return;
    const newProperties = [...properties];
    [newProperties[index - 1], newProperties[index]] = [newProperties[index], newProperties[index - 1]];
    // Update order values
    newProperties.forEach((prop, i) => {
      prop.order = i;
    });
    onChange(newProperties);
  };

  const handleMoveDown = (index: number) => {
    if (index === properties.length - 1) return;
    const newProperties = [...properties];
    [newProperties[index], newProperties[index + 1]] = [newProperties[index + 1], newProperties[index]];
    // Update order values
    newProperties.forEach((prop, i) => {
      prop.order = i;
    });
    onChange(newProperties);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600">
          {properties.length} {properties.length === 1 ? 'property' : 'properties'} configured
        </p>
        <button
          type="button"
          onClick={handleAdd}
          className="px-4 py-2 bg-green-600 text-white text-sm rounded-lg hover:bg-green-700 transition-colors"
        >
          + Add Property
        </button>
      </div>

      {properties.length === 0 ? (
        <div className="bg-gray-50 border-2 border-dashed border-gray-300 rounded-lg p-8 text-center">
          <p className="text-gray-600">No properties configured yet.</p>
          <p className="text-sm text-gray-500 mt-1">Click "Add Property" to get started.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {properties.map((property, index) => (
            <div
              key={index}
              className="bg-gray-50 border border-gray-200 rounded-lg p-4 flex items-center justify-between hover:bg-gray-100 transition-colors"
            >
              <div className="flex-1">
                <div className="flex items-center gap-3">
                  <span className="text-sm font-semibold text-gray-900">{property.label || property.name}</span>
                  <span className="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded">
                    {property.data_type}
                  </span>
                  <span className="px-2 py-1 text-xs font-medium bg-purple-100 text-purple-800 rounded">
                    {property.input_type}
                  </span>
                  {!property.nullable && (
                    <span className="px-2 py-1 text-xs font-medium bg-red-100 text-red-800 rounded">
                      Required
                    </span>
                  )}
                </div>
                <div className="mt-1 text-xs text-gray-600">
                  Field name: <code className="bg-gray-200 px-1 rounded">{property.name}</code>
                  {property.default_value !== undefined && property.default_value !== null && (
                    <span className="ml-2">
                      Default: <code className="bg-gray-200 px-1 rounded">{String(property.default_value)}</code>
                    </span>
                  )}
                </div>
              </div>

              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => handleMoveUp(index)}
                  disabled={index === 0}
                  className="p-1 text-gray-500 hover:text-gray-700 disabled:opacity-30 disabled:cursor-not-allowed"
                  title="Move up"
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                  </svg>
                </button>
                <button
                  type="button"
                  onClick={() => handleMoveDown(index)}
                  disabled={index === properties.length - 1}
                  className="p-1 text-gray-500 hover:text-gray-700 disabled:opacity-30 disabled:cursor-not-allowed"
                  title="Move down"
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
                <button
                  type="button"
                  onClick={() => handleEdit(property, index)}
                  className="px-3 py-1 text-sm text-blue-600 hover:text-blue-800"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={() => handleDelete(index)}
                  className="px-3 py-1 text-sm text-red-600 hover:text-red-800"
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {isModalOpen && (
        <PropertyFormModal
          property={editingProperty}
          existingPropertyNames={properties.map((p) => p.name)}
          onSave={handleSave}
          onClose={() => setIsModalOpen(false)}
        />
      )}
    </div>
  );
};
