// components/ExportModal/SelectionPanels.tsx
import type { FC } from 'react';
import { Node as FlowNode } from 'reactflow';
import { ExportOptions } from '../../types/ExportTypes';
import { getTableName } from '../../utils/exportUtils';

interface SelectionPanelsProps {
  exportScope: ExportOptions['exportScope'];
  schemaGroups: { [schema: string]: FlowNode[] };
  selectedSchemas: string[];
  selectedTables: string[];
  onSchemaToggle: (schema: string) => void;
  onTableToggle: (tableId: string) => void;
  onSelectAllSchemas: () => void;
  onClearSchemaSelection: () => void;
  onSelectAllTables: () => void;
  onClearTableSelection: () => void;
}

export const SelectionPanels: FC<SelectionPanelsProps> = ({
  exportScope,
  schemaGroups,
  selectedSchemas,
  selectedTables,
  onSchemaToggle,
  onTableToggle,
  onSelectAllSchemas,
  onClearSchemaSelection,
  onSelectAllTables,
  onClearTableSelection
}) => {
  if (exportScope === 'all') return null;

  if (exportScope === 'schemas') {
  const availableSchemas = Object.keys(schemaGroups);
    
    return (
      <div>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Schema Selection</h3>
          <div className="flex space-x-2">
            <button
              onClick={onSelectAllSchemas}
              className="text-xs px-3 py-2 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 transition-colors"
            >
              Select All
            </button>
            <button
              onClick={onClearSchemaSelection}
              className="text-xs px-3 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
            >
              Clear
            </button>
          </div>
        </div>
        
        <div className="bg-white border border-gray-200 rounded-xl max-h-64 overflow-y-auto">
          {availableSchemas.map(schema => (
            <label key={schema} className="flex items-center justify-between p-4 border-b border-gray-100 last:border-b-0 hover:bg-gray-50 cursor-pointer">
              <div className="flex items-center space-x-3">
                <input
                  type="checkbox"
                  checked={selectedSchemas.includes(schema)}
                  onChange={() => onSchemaToggle(schema)}
                  className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <div>
                  <span className="text-sm font-medium text-gray-900">{schema}</span>
                  <div className="text-xs text-gray-500">
                    {schemaGroups[schema].map(t => getTableName(t)).slice(0, 3).join(', ')}
                    {schemaGroups[schema].length > 3 && '...'}
                  </div>
                </div>
              </div>
              <span className="text-xs text-blue-700 bg-blue-100 px-2 py-1 rounded-full">
                {schemaGroups[schema].length}
              </span>
            </label>
          ))}
        </div>
      </div>
    );
  }

  if (exportScope === 'tables') {
    return (
      <div>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Table Selection</h3>
          <div className="flex space-x-2">
            <button
              onClick={onSelectAllTables}
              className="text-xs px-3 py-2 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 transition-colors"
            >
              Select All
            </button>
            <button
              onClick={onClearTableSelection}
              className="text-xs px-3 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
            >
              Clear
            </button>
          </div>
        </div>
        
        <div className="bg-white border border-gray-200 rounded-xl max-h-64 overflow-y-auto">
          {Object.entries(schemaGroups).map(([schema, tables]) => (
            <div key={schema}>
              <div className="px-4 py-2 bg-gray-100 border-b border-gray-200 text-sm font-medium text-gray-700">
                {schema} ({tables.length})
              </div>
              {tables.map(table => (
                <label key={table.id} className="flex items-center justify-between p-3 border-b border-gray-100 last:border-b-0 hover:bg-gray-50 cursor-pointer">
                  <div className="flex items-center space-x-3">
                    <input
                      type="checkbox"
                      checked={selectedTables.includes(table.id)}
                      onChange={() => onTableToggle(table.id)}
                      className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-900">{getTableName(table)}</span>
                  </div>
                  <span className="text-xs text-gray-500">
                    {table.data?.columns?.length || 0} cols
                  </span>
                </label>
              ))}
            </div>
          ))}
        </div>
      </div>
    );
  }

  return null;
};