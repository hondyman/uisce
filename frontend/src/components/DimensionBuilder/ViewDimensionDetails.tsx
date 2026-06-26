// src/components/DimensionBuilder/ViewDimensionDetails.tsx
import { Info, Code, ChevronDown, ChevronRight } from 'lucide-react';
import { Dimension } from './types';
import { CaseEditor } from './CaseEditor';
import GranularitySection from './GranularitySection';

interface ViewDimensionDetailsProps {
  dimension: Dimension;
  onUpdate: (id: string, updates: Partial<Dimension>) => void;
  expandedSections: {[key: string]: boolean};
  onToggleSection: (sectionId: string) => void;
}

export function ViewDimensionDetails({
  dimension,
  onUpdate,
  expandedSections,
  onToggleSection
}: ViewDimensionDetailsProps) {
  const getSectionKey = (suffix: string) => `${dimension.id}_${suffix}`;

  const updateCase = (caseObj: Dimension['case']) => {
    onUpdate(dimension.id, { case: caseObj });
  };

  return (
    <div className="space-y-6">
      {/* Basic Info */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-gray-50 p-3 rounded-lg">
          <label className="block text-xs font-medium text-gray-500 mb-1">SQL Expression</label>
          <code className="text-sm font-mono text-gray-900 break-all">{dimension.sql}</code>
        </div>
        
        {dimension.description && (
          <div className="bg-gray-50 p-3 rounded-lg">
            <label className="block text-xs font-medium text-gray-500 mb-1">Description</label>
            <p className="text-sm text-gray-700">{dimension.description}</p>
          </div>
        )}
      </div>

      {/* Case Section */}
      <div className="border border-gray-200 rounded-lg">
        <button
          onClick={() => onToggleSection(getSectionKey('case'))}
          className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
        >
          <div className="flex items-center gap-2">
            <Info className="w-5 h-5 text-blue-500" />
            <span className="font-medium">Case Statement</span>
            {dimension.case && (
              <span className="text-sm text-gray-500">
                ({dimension.case.when.length} conditions)
              </span>
            )}
          </div>
          {expandedSections[getSectionKey('case')] ? (
            <ChevronDown className="w-5 h-5 text-gray-400" />
          ) : (
            <ChevronRight className="w-5 h-5 text-gray-400" />
          )}
        </button>
        
        {expandedSections[getSectionKey('case')] && (
          <div className="border-t border-gray-200 p-4">
            <CaseEditor
              caseObj={dimension.case}
              onUpdate={updateCase}
            />
          </div>
        )}
      </div>

      {/* Granularities Section (only for time dimensions) */}
      {dimension.type === 'time' && (
        <GranularitySection
          dimension={dimension}
          onUpdate={onUpdate}
          expandedSections={expandedSections}
          onToggleSection={onToggleSection}
        />
      )}

      {/* Meta Information */}
      {dimension.meta && Object.keys(dimension.meta).length > 0 && (
        <div className="border border-gray-200 rounded-lg">
          <button
            onClick={() => onToggleSection(getSectionKey('meta'))}
            className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Code className="w-5 h-5 text-purple-500" />
              <span className="font-medium">Meta Information</span>
            </div>
            {expandedSections[getSectionKey('meta')] ? (
              <ChevronDown className="w-5 h-5 text-gray-400" />
            ) : (
              <ChevronRight className="w-5 h-5 text-gray-400" />
            )}
          </button>
          
          {expandedSections[getSectionKey('meta')] && (
            <div className="border-t border-gray-200 p-4">
              <div className="space-y-2">
                {Object.entries(dimension.meta as Record<string, any>).map(([key, value]) => {
                  // UUID regex pattern
                  const uuidRegex = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
                  
                  let displayValue: any = value;
                  
                  // Special handling for 'type' field
                  if (key === 'type' && typeof value === 'string') {
                    // If it's a UUID, try to map to dimension type or show readable format
                    if (uuidRegex.test(value)) {
                      // Map common type UUIDs to readable names if we have them
                      // Otherwise show the primary dimension type as fallback
                      const typeMapping: Record<string, string> = {
                        'string': 'string',
                        'number': 'number', 
                        'time': 'time',
                        'boolean': 'boolean',
                        'geo': 'geo'
                      };
                      // Use the dimension's own type as reference, otherwise show UUID with indicator
                      displayValue = `${dimension.type || 'unknown'} (from ${value.substring(0, 8)}...)`;
                    }
                  } else if (typeof value === 'string' && uuidRegex.test(value)) {
                    // Other UUID fields - show abbreviated UUID
                    displayValue = `${value.substring(0, 8)}... (UUID)`;
                  } else if (typeof value === 'boolean') {
                    displayValue = value ? 'Yes' : 'No';
                  } else if (typeof value === 'object') {
                    displayValue = JSON.stringify(value, null, 2);
                  }
                  
                  return (
                    <div key={key} className="flex justify-between items-start py-2 border-b border-gray-100 last:border-b-0">
                      <span className="text-sm font-mono text-gray-600 capitalize">{key}:</span>
                      <span className="text-sm text-gray-900 font-mono text-right max-w-xs break-words">{displayValue}</span>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
