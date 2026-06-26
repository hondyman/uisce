// src/components/DimensionBuilder/NewDimensionForm.tsx
import { useState } from 'react';
import { Plus, Save, X } from 'lucide-react';
import { Dimension, dimensionTypes, formatOptions } from './types';
import SqlMonacoEditor from '../SqlMonacoEditor';

interface NewDimensionFormProps {
  onCreateDimension: (dimension: Partial<Dimension>) => void;
  onCancel: () => void;
}

export function NewDimensionForm({ onCreateDimension, onCancel }: NewDimensionFormProps) {
  const [newDimension, setNewDimension] = useState<Partial<Dimension>>({
    type: 'string' as const,
    public: true,
    format: '' as const
  });

  const handleCreate = () => {
    onCreateDimension(newDimension);
    setNewDimension({ type: 'string' as const, public: true, format: '' as const });
  };

  return (
    <div className="mb-6 bg-white border border-gray-200 rounded-xl p-6 shadow-sm">
      <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
        <Plus className="w-5 h-5 text-indigo-500" />
        Create New Dimension
      </h3>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Name *</label>
          <input
            type="text"
            value={newDimension.name || ''}
            onChange={(e) => setNewDimension(prev => ({ ...prev, name: e.target.value }))}
            className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
            placeholder="dimension_name"
          />
        </div>
        
        <div>
          <label htmlFor="new-dim-type" className="block text-sm font-medium text-gray-700 mb-1">Type *</label>
          <select
            id="new-dim-type"
            value={newDimension.type || 'string'}
            onChange={(e) => setNewDimension(prev => ({ ...prev, type: e.target.value as Dimension['type'] }))}
            className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
          >
            {dimensionTypes.map(type => (
              <option key={type.value} value={type.value}>{type.label}</option>
            ))}
          </select>
        </div>
        
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">SQL Expression *</label>
          <SqlMonacoEditor
            value={newDimension.sql || ''}
            onChange={(value) => setNewDimension(prev => ({ ...prev, sql: value }))}
            placeholder="{CUBE}.column_name"
            height={80}
          />
        </div>
        
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
          <input
            type="text"
            value={newDimension.title || ''}
            onChange={(e) => setNewDimension(prev => ({ ...prev, title: e.target.value }))}
            className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
            placeholder="Human readable title"
          />
        </div>
      </div>
      
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea
          value={newDimension.description || ''}
          onChange={(e) => setNewDimension(prev => ({ ...prev, description: e.target.value }))}
          className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
          rows={2}
          placeholder="Dimension description"
        />
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
        <div>
          <label htmlFor="new-dim-format" className="block text-sm font-medium text-gray-700 mb-1">Format</label>
          <select
            id="new-dim-format"
            value={newDimension.format || ''}
            onChange={(e) => setNewDimension(prev => ({ ...prev, format: e.target.value as Dimension['format'] }))}
            className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
          >
            {formatOptions.map(format => (
              <option key={format.value} value={format.value}>{format.label}</option>
            ))}
          </select>
        </div>
        
        <div className="flex flex-wrap gap-4 mt-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={newDimension.public !== false}
              onChange={(e) => setNewDimension(prev => ({ ...prev, public: e.target.checked }))}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Public</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!newDimension.primary_key}
              onChange={(e) => setNewDimension(prev => ({ ...prev, primary_key: e.target.checked }))}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Primary Key</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!newDimension.sub_query}
              onChange={(e) => setNewDimension(prev => ({ ...prev, sub_query: e.target.checked }))}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Sub Query</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!newDimension.propagate_filters_to_sub_query}
              onChange={(e) => setNewDimension(prev => ({ ...prev, propagate_filters_to_sub_query: e.target.checked }))}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Propagate Filters</span>
          </label>
        </div>
      </div>
      
      <div className="flex gap-3">
        <button
          onClick={handleCreate}
          disabled={!newDimension.name || !newDimension.sql || !newDimension.type}
          className="bg-indigo-500 hover:bg-indigo-600 disabled:opacity-50 disabled:cursor-not-allowed text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        >
          <Save className="w-4 h-4" />
          Create Dimension
        </button>
        <button
          onClick={onCancel}
          className="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        >
          <X className="w-4 h-4" />
          Cancel
        </button>
      </div>
    </div>
  );
}