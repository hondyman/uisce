// src/components/DimensionBuilder/DimensionCard.tsx
import { useState, useEffect } from 'react';
import { Code, Copy, Edit3, Trash2, EyeOff } from 'lucide-react';
import { Dimension, dimensionTypes } from './types';
import { EditDimensionForm } from './EditDimensionForm';
import { ViewDimensionDetails } from './ViewDimensionDetails';
import { generateDimensionCode } from './utils';

interface DimensionCardProps {
  dimension: Dimension;
  onUpdate: (id: string, updates: Partial<Dimension>) => void;
  onDelete: (id: string) => void;
  onToggleEdit: (id: string) => void;
  onDimensionsChange: (_dimensions: Dimension[]) => void;
  expandedSections: {[key: string]: boolean};
  onToggleSection: (sectionId: string) => void;
  onCopyCode: (code: string) => void;
}

export function DimensionCard({
  dimension,
  onUpdate,
  onDelete,
  onToggleEdit,
  onDimensionsChange: _onDimensionsChange,
  expandedSections,
  onToggleSection,
  onCopyCode
}: DimensionCardProps) {
  const [editData, setEditData] = useState<Partial<Dimension>>(dimension);
  // use explicit 'json' | null to match the global code-panel API and avoid boolean toggles
  const [showCode, setShowCode] = useState<string | null>(null);

  useEffect(() => {
    setEditData(dimension);
  }, [dimension]);

  const getDimensionTypeIcon = (type: string) => {
    const typeConfig = dimensionTypes.find(t => t.value === type);
    return typeConfig?.icon || dimensionTypes[0].icon;
  };

  const saveEdit = () => {
    onUpdate(dimension.id, editData);
  };

  const cancelEdit = () => {
    setEditData(dimension);
    onToggleEdit(dimension.id);
  };

  const TypeIcon = getDimensionTypeIcon(dimension.type);

  return (
    <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
      {/* Header */}
      <div className="p-4 bg-gradient-to-r from-gray-50 to-gray-100 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-indigo-100 rounded-lg">
              <TypeIcon className="w-5 h-5 text-indigo-600" />
            </div>
            <div>
              <h3 className="font-semibold text-gray-900 flex items-center gap-2">
                {dimension.title || dimension.name}
                {!dimension.public && <EyeOff className="w-4 h-4 text-gray-400" />}
              </h3>
              <div className="flex items-center gap-3 text-sm text-gray-600">
                <span className="font-mono bg-gray-200 text-gray-800 px-2 py-1 rounded text-xs">
                  {dimension.type}
                </span>
                {dimension.format && (
                  <span className="flex items-center gap-1">
                    {dimension.format}
                  </span>
                )}
                <span className="font-mono">{dimension.name}</span>
              </div>
            </div>
          </div>
          
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowCode(prev => (prev ? null : 'json'))}
              className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
              title="Show/Hide Code"
            >
              <Code className="w-4 h-4" />
            </button>
            <button
              onClick={() => onCopyCode(generateDimensionCode(dimension))}
              className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
              title="Copy Code"
            >
              <Copy className="w-4 h-4" />
            </button>
            <button
              onClick={() => onToggleEdit(dimension.id)}
              className="p-2 text-indigo-500 hover:text-indigo-700 hover:bg-indigo-100 rounded-lg transition-colors"
              aria-label="Edit dimension"
              title="Edit"
            >
              <Edit3 className="w-4 h-4" />
            </button>
            <button
              onClick={() => onDelete(dimension.id)}
              className="p-2 text-red-500 hover:text-red-700 hover:bg-red-100 rounded-lg transition-colors"
              aria-label="Delete dimension"
              title="Delete"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Code Preview */}
      {showCode && (
        <div className="p-4 bg-gray-900 text-gray-100">
          <pre className="text-sm font-mono overflow-x-auto whitespace-pre">
            {generateDimensionCode(dimension)}
          </pre>
        </div>
      )}

      {/* Content */}
      <div className="p-4">
        {dimension.isEditing ? (
          <EditDimensionForm
            dimension={dimension}
            editData={editData}
            setEditData={setEditData}
            onSave={saveEdit}
            onCancel={cancelEdit}
          />
        ) : (
          <ViewDimensionDetails
            dimension={dimension}
            onUpdate={onUpdate}
            expandedSections={expandedSections}
            onToggleSection={onToggleSection}
          />
        )}
      </div>
    </div>
  );
}