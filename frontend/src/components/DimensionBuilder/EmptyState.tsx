// src/components/DimensionBuilder/EmptyState.tsx
import { Plus, Type } from 'lucide-react';

interface EmptyStateProps {
  onShowForm: () => void;
}

export function EmptyState({ onShowForm }: EmptyStateProps) {
  return (
    <div className="text-center py-12 bg-gray-50 rounded-xl border-2 border-dashed border-gray-300">
      <Type className="w-12 h-12 text-gray-400 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900 mb-2">No dimensions yet</h3>
      <p className="text-gray-500 mb-4">Create your first dimension to get started</p>
      <button
        onClick={onShowForm}
        className="bg-indigo-500 hover:bg-indigo-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2 mx-auto"
      >
        <Plus className="w-4 h-4" />
        Add First Dimension
      </button>
    </div>
  );
}
