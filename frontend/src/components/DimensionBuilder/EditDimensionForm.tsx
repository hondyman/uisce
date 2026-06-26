// src/components/DimensionBuilder/EditDimensionForm.tsx
import { Save, X } from 'lucide-react';
import { Dimension, dimensionTypes, formatOptions } from './types';

interface EditDimensionFormProps {
  // Accept the owning dimension as optional so callers can pass it when available.
  dimension?: Dimension;
  editData: Partial<Dimension>;
  setEditData: (data: Partial<Dimension>) => void;
  onSave: () => void;
  onCancel: () => void;
}

export function EditDimensionForm({ 
  dimension: _dimension,
  editData, 
  setEditData, 
  onSave, 
  onCancel 
}: EditDimensionFormProps) {
  // dimension may be optional for callers; not used here

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="dim-name" className="block text-sm font-medium text-gray-700 mb-1">Name</label>
          <input
            type="text"
            id="dim-name"
            value={editData.name || ''}
            onChange={(e) => setEditData({ ...editData, name: e.target.value })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
          />
        </div>
        
        <div>
          <label htmlFor="dim-type" className="block text-sm font-medium text-gray-700 mb-1">Type</label>
          <select
            id="dim-type"
            value={editData.type}
            onChange={(e) => setEditData({ ...editData, type: e.target.value as Dimension['type'] })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
          >
            {dimensionTypes.map(type => (
              <option key={type.value} value={type.value}>{type.label}</option>
            ))}
          </select>
        </div>
        
        <div>
          <label htmlFor="dim-sql" className="block text-sm font-medium text-gray-700 mb-1">SQL</label>
          <input
            type="text"
            id="dim-sql"
            value={editData.sql || ''}
            onChange={(e) => setEditData({ ...editData, sql: e.target.value })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono text-gray-900"
          />
        </div>
        
        <div>
          <label htmlFor="dim-title" className="block text-sm font-medium text-gray-700 mb-1">Title</label>
          <input
            type="text"
            id="dim-title"
            value={editData.title || ''}
            onChange={(e) => setEditData({ ...editData, title: e.target.value })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
          />
        </div>
      </div>
      
      <div>
        <label htmlFor="dim-description" className="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea
          id="dim-description"
          value={editData.description || ''}
          onChange={(e) => setEditData({ ...editData, description: e.target.value })}
          className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
          rows={2}
        />
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="dim-format" className="block text-sm font-medium text-gray-700 mb-1">Format</label>
          <select
            id="dim-format"
            value={editData.format ?? ''}
            onChange={(e) => setEditData({ ...editData, format: e.target.value as Dimension['format'] })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-900"
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
              checked={editData.public !== false}
              onChange={(e) => setEditData({ ...editData, public: e.target.checked })}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Public</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!editData.primary_key}
              onChange={(e) => setEditData({ ...editData, primary_key: e.target.checked })}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Primary Key</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!editData.sub_query}
              onChange={(e) => setEditData({ ...editData, sub_query: e.target.checked })}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Sub Query</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!editData.propagate_filters_to_sub_query}
              onChange={(e) => setEditData({ ...editData, propagate_filters_to_sub_query: e.target.checked })}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Propagate Filters</span>
          </label>
        </div>
      </div>
      
      <div className="flex gap-3 pt-4 border-t border-gray-200">
        <button
          onClick={onSave}
          className="bg-indigo-500 hover:bg-indigo-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        >
          <Save className="w-4 h-4" />
          Save
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
