// src/components/DimensionBuilder/DimensionBuilder.tsx
import { useState } from 'react';
import { Plus, Code, Type } from 'lucide-react';
import { Dimension } from './types';
import { DimensionCard } from './DimensionCard';
import { NewDimensionForm } from './NewDimensionForm';
import { EmptyState } from './EmptyState';

interface DimensionBuilderProps {
  dimensions: Dimension[];
  onDimensionsChange: (dimensions: Dimension[]) => void;
  onGenerateCode?: () => void;
}

export default function DimensionBuilder({
  dimensions,
  onDimensionsChange,
  onGenerateCode
}: DimensionBuilderProps) {
  const [showNewDimensionForm, setShowNewDimensionForm] = useState(false);
  const [expandedSections, setExpandedSections] = useState<{[key: string]: boolean}>({});

  const generateDimensionId = () => `dimension_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

  const createDimension = (newDimension: Partial<Dimension>) => {
    if (!newDimension.name || !newDimension.sql || !newDimension.type) return;

    const dimension: Dimension = {
      id: generateDimensionId(),
      name: newDimension.name,
      sql: newDimension.sql,
      type: newDimension.type,
      title: newDimension.title,
      description: newDimension.description,
      format: newDimension.format || undefined,
      meta: newDimension.meta || {},
      primary_key: newDimension.primary_key,
      public: newDimension.public !== false,
      sub_query: newDimension.sub_query,
      propagate_filters_to_sub_query: newDimension.propagate_filters_to_sub_query,
      case: undefined,
      granularities: []
    };

    onDimensionsChange([...dimensions, dimension]);
    setShowNewDimensionForm(false);
  };

  const updateDimension = (id: string, updates: Partial<Dimension>) => {
    onDimensionsChange(dimensions.map(d =>
      d.id === id ? { ...d, ...updates } : d
    ));
  };

  const deleteDimension = (id: string) => {
    onDimensionsChange(dimensions.filter(d => d.id !== id));
  };

  const toggleEdit = (id: string) => {
    onDimensionsChange(dimensions.map(d =>
      d.id === id ? { ...d, isEditing: !d.isEditing } : d
    ));
  };

  const toggleSection = (sectionId: string) => {
    setExpandedSections(prev => ({
      ...prev,
      [sectionId]: !prev[sectionId]
    }));
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  return (
    <div className="dimension-builder">
      {/* Header */}
      <div className="mb-6 bg-gradient-to-r from-indigo-500 to-purple-600 text-white p-6 rounded-xl shadow-lg">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Type className="w-8 h-8" />
            <div>
              <h2 className="text-2xl font-bold">Dimension Builder</h2>
              <p className="text-indigo-100">Create and manage Cube.js dimensions with advanced features</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            <span className="bg-white/20 px-3 py-1 rounded-full text-sm font-medium">
              {dimensions.length} dimension{dimensions.length !== 1 ? 's' : ''}
            </span>
            <button
              onClick={() => setShowNewDimensionForm(prev => !prev)}
              className="bg-white/20 hover:bg-white/30 px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
            >
              <Plus className="w-4 h-4" />
              Add Dimension
            </button>
          </div>
        </div>
      </div>

      {/* New Dimension Form */}
      {showNewDimensionForm && (
        <NewDimensionForm
          onCreateDimension={createDimension}
          onCancel={() => setShowNewDimensionForm(false)}
        />
      )}

      {/* Dimensions List */}
      <div className="space-y-4">
        {dimensions.length === 0 ? (
          <EmptyState onShowForm={() => setShowNewDimensionForm(true)} />
        ) : (
          dimensions.map((dimension) => (
            <DimensionCard
              key={dimension.id}
              dimension={dimension}
              onUpdate={updateDimension}
              onDelete={deleteDimension}
              onToggleEdit={toggleEdit}
              onDimensionsChange={onDimensionsChange}
              expandedSections={expandedSections}
              onToggleSection={toggleSection}
              onCopyCode={copyToClipboard}
            />
          ))
        )}
      </div>

      {/* Generate All Code */}
      {dimensions.length > 0 && onGenerateCode && (
        <div className="mt-8 p-4 bg-gray-50 rounded-xl">
          <button
            onClick={onGenerateCode}
            className="w-full bg-green-500 hover:bg-green-600 text-white px-4 py-3 rounded-lg transition-colors flex items-center gap-2 justify-center font-medium"
          >
            <Code className="w-5 h-5" />
            Generate Complete Cube Schema
          </button>
        </div>
      )}
    </div>
  );
}