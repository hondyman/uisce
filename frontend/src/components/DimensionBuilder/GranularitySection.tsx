// src/components/DimensionBuilder/GranularitySection.tsx
import { Calendar, ChevronDown, ChevronRight } from 'lucide-react';
import { Dimension, Granularity } from './types';
import { GranularityForm } from './GranularityForm';
import { GranularityEditor } from './GranularityEditor';

interface GranularitySectionProps {
  dimension: Dimension;
  onUpdate: (id: string, updates: Partial<Dimension>) => void;
  expandedSections: { [key: string]: boolean };
  onToggleSection: (sectionId: string) => void;
}

export default function GranularitySection({
  dimension,
  onUpdate,
  expandedSections,
  onToggleSection,
}: GranularitySectionProps) {
  const getSectionKey = (suffix: string) => `${dimension.id}_${suffix}`;
  const sectionKey = getSectionKey('granularities');

  const addGranularity = (granularity: Omit<Granularity, 'id'>) => {
    const newGranularity = { ...granularity, id: `gran_${Date.now()}` };
    const newGranularities = [...(dimension.granularities || []), newGranularity];
    onUpdate(dimension.id, { granularities: newGranularities });
  };

  const updateGranularity = (granularityId: string, updates: Partial<Granularity>) => {
    const newGranularities = (dimension.granularities || []).map(g =>
      g.id === granularityId ? { ...g, ...updates } : g
    );
    onUpdate(dimension.id, { granularities: newGranularities });
  };

  const removeGranularity = (granularityId: string) => {
    const newGranularities = (dimension.granularities || []).filter(g => g.id !== granularityId);
    onUpdate(dimension.id, { granularities: newGranularities });
  };

  return (
    <div className="border border-gray-200 rounded-lg">
      <button
        onClick={() => onToggleSection(sectionKey)}
        className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
      >
        <div className="flex items-center gap-2">
          <Calendar className="w-5 h-5 text-green-500" />
          <span className="font-medium">Granularities</span>
          <span className="text-sm text-gray-500">
            ({(dimension.granularities || []).length})
          </span>
        </div>
        {expandedSections[sectionKey] ? (
          <ChevronDown className="w-5 h-5 text-gray-400" />
        ) : (
          <ChevronRight className="w-5 h-5 text-gray-400" />
        )}
      </button>

      {expandedSections[sectionKey] && (
        <div className="border-t border-gray-200 p-4 space-y-4">
          <GranularityForm onAdd={addGranularity} />

          {dimension.granularities && dimension.granularities.length > 0 ? (
            <div className="space-y-3">
              {dimension.granularities.map(gran => (
                <GranularityEditor
                  key={gran.id}
                  granularity={gran}
                  onUpdate={(updates) => updateGranularity(gran.id, updates)}
                  onRemove={() => removeGranularity(gran.id)}
                />
              ))}
            </div>
          ) : (
            <p className="text-gray-500 text-sm italic text-center py-4">
              No custom granularities configured
            </p>
          )}
        </div>
      )}
    </div>
  );
}