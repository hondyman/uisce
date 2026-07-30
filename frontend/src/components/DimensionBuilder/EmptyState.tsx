// src/components/DimensionBuilder/EmptyState.tsx
import AddIcon from '@mui/icons-material/Add';
import TextFieldsIcon from '@mui/icons-material/TextFields';

interface EmptyStateProps {
  onShowForm: () => void;
}

export function EmptyState({ onShowForm }: EmptyStateProps) {
  return (
    <div className="text-center py-12 bg-gray-50 rounded-xl border-2 border-dashed border-gray-300">
      <TextFieldsIcon sx={{ fontSize: 48 }} className="mx-auto mb-4" color="disabled" />
      <h3 className="text-lg font-medium text-gray-900 mb-2">No dimensions yet</h3>
      <p className="text-gray-500 mb-4">Create your first dimension to get started</p>
      <button
        onClick={onShowForm}
        className="bg-indigo-500 hover:bg-indigo-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2 mx-auto"
      >
        <AddIcon fontSize="small" />
        Add First Dimension
      </button>
    </div>
  );
}
