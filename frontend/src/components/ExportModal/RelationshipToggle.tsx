// components/ExportModal/RelationshipToggle.tsx
import type { FC } from 'react';

interface RelationshipToggleProps {
  includeRelationships: boolean;
  onToggle: (include: boolean) => void;
  showToggle: boolean;
}

export const RelationshipToggle: FC<RelationshipToggleProps> = ({
  includeRelationships,
  onToggle,
  showToggle
}) => {
  if (!showToggle) return null;

  return (
    <div className="bg-purple-50 border border-purple-200 rounded-xl p-6">
      <label className="flex items-center space-x-4 cursor-pointer">
        <input
          type="checkbox"
          checked={includeRelationships}
          onChange={(e) => onToggle(e.target.checked)}
          className="w-5 h-5 text-purple-600 border-purple-300 rounded focus:ring-purple-500"
        />
        <div className="flex items-center space-x-3">
          <span className="text-2xl">🔗</span>
          <div>
            <div className="font-semibold text-purple-900">Include Relationships</div>
            <div className="text-sm text-purple-700">Export foreign keys and constraints</div>
          </div>
        </div>
      </label>
    </div>
  );
};