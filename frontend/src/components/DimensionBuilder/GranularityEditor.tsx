// src/components/DimensionBuilder/GranularityEditor.tsx
import { useState } from 'react';
import { Save, X, Edit3, Trash2 } from 'lucide-react';
import { Granularity } from './types';

interface GranularityEditorProps {
  granularity: Granularity;
  onUpdate: (updates: Partial<Granularity>) => void;
  onRemove: () => void;
}

export function GranularityEditor({ granularity, onUpdate, onRemove }: GranularityEditorProps) {
  const [editData, setEditData] = useState(granularity);
  const [isEditing, setIsEditing] = useState(false);

  const saveEdit = () => {
    onUpdate(editData);
    setIsEditing(false);
  };

  const cancelEdit = () => {
    setEditData(granularity);
    setIsEditing(false);
  };

  return (
    <div className="bg-green-50 border border-green-200 p-4 rounded-lg">
      {isEditing ? (
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input
              type="text"
              value={editData.name}
              onChange={(e) => setEditData({ ...editData, name: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500 text-gray-900"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Interval</label>
            <input
              type="text"
              value={editData.interval}
              onChange={(e) => setEditData({ ...editData, interval: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500 text-gray-900"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Offset</label>
            <input
              type="text"
              value={editData.offset || ''}
              onChange={(e) => setEditData({ ...editData, offset: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500 text-gray-900"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Origin</label>
            <input
              type="text"
              value={editData.origin || ''}
              onChange={(e) => setEditData({ ...editData, origin: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500 text-gray-900"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
            <input
              type="text"
              value={editData.title || ''}
              onChange={(e) => setEditData({ ...editData, title: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500 text-gray-900"
            />
          </div>
          <div className="flex gap-2">
            <button
              onClick={saveEdit}
              className="bg-green-500 hover:bg-green-600 text-white px-3 py-1 rounded text-sm transition-colors flex items-center gap-1"
            >
              <Save className="w-3 h-3" />
              Save
            </button>
            <button
              onClick={cancelEdit}
              className="bg-gray-500 hover:bg-gray-600 text-white px-3 py-1 rounded text-sm transition-colors flex items-center gap-1"
            >
              <X className="w-3 h-3" />
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <span className="font-medium block mb-1">{granularity.name}</span>
            <p className="text-sm text-gray-600">Interval: {granularity.interval}</p>
            {granularity.offset && <p className="text-sm text-gray-600">Offset: {granularity.offset}</p>}
            {granularity.origin && <p className="text-sm text-gray-600">Origin: {granularity.origin}</p>}
            {granularity.title && <p className="text-sm text-gray-600">Title: {granularity.title}</p>}
          </div>
          <div className="flex gap-1">
            <button
              onClick={() => setIsEditing(true)}
              className="text-green-600 hover:text-green-800 p-1 rounded transition-colors"
            >
              <Edit3 className="w-3 h-3" />
            </button>
            <button
              onClick={onRemove}
              className="text-red-500 hover:text-red-700 p-1 rounded transition-colors"
            >
              <Trash2 className="w-3 h-3" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
