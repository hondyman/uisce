import React, { useState, useEffect } from 'react';
import { ArrowRight, Check, X, Sparkles } from 'lucide-react';

interface FieldMapping {
  bo_field_name: string;
  bo_field_type: string;
  view_field_name: string;
  view_field_type: string;
  mapping_type: 'direct' | 'computed' | 'derived';
}

interface BOToViewMapping {
  tenant_id: string;
  bo_key: string;
  bo_name: string;
  view_id: string;
  view_name: string;
  field_mappings: FieldMapping[];
}

const FieldMappingVisualizer: React.FC = () => {
  const [mappings, setMappings] = useState<BOToViewMapping[]>([]);
  const [selectedMapping, setSelectedMapping] = useState<BOToViewMapping | null>(null);
  const [loading, setLoading] = useState(false);

  // Mock data for demonstration
  useEffect(() => {
    const mockMapping: BOToViewMapping = {
      tenant_id: 'default-tenant',
      bo_key: 'Worker',
      bo_name: 'Worker',
      view_id: 'employee_view',
      view_name: 'Employee View',
      field_mappings: [
        {
          bo_field_name: 'email',
          bo_field_type: 'string',
          view_field_name: 'email',
          view_field_type: 'string',
          mapping_type: 'direct',
        },
        {
          bo_field_name: 'first_name',
          bo_field_type: 'string',
          view_field_name: 'first_name',
          view_field_type: 'string',
          mapping_type: 'direct',
        },
        {
          bo_field_name: 'last_name',
          bo_field_type: 'string',
          view_field_name: 'last_name',
          view_field_type: 'string',
          mapping_type: 'direct',
        },
        {
          bo_field_name: 'hire_date',
          bo_field_type: 'date',
          view_field_name: 'hire_date',
          view_field_type: 'timestamp',
          mapping_type: 'computed',
        },
      ],
    };
    setMappings([mockMapping]);
    setSelectedMapping(mockMapping);
  }, []);

  const createMapping = async (boKey: string, viewId: string) => {
    setLoading(true);
    try {
      const response = await fetch('/api/metadata/unified/mappings', {
        method: 'POST',
        headers: {
          'X-Tenant-ID': 'default-tenant',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ bo_key: boKey, view_id: viewId }),
      });
      const newMapping = await response.json();
      setMappings([...mappings, newMapping]);
      setSelectedMapping(newMapping);
    } catch (error) {
      console.error('Failed to create mapping:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-slate-900 dark:text-white mb-2">
            Field Mapping Visualizer
          </h2>
          <p className="text-slate-600 dark:text-slate-400">
            Visualize and manage field mappings between business objects and semantic views
          </p>
        </div>
        <button
          onClick={() => createMapping('Worker', 'employee_view')}
          disabled={loading}
          className="px-4 py-2 bg-gradient-to-r from-blue-500 to-indigo-600 text-white rounded-lg hover:shadow-lg transition-all disabled:opacity-50"
        >
          <span className="flex items-center space-x-2">
            <Sparkles className="w-4 h-4" />
            <span>Auto-Map Fields</span>
          </span>
        </button>
      </div>

      {/* Mapping Selector */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {mappings.map((mapping, idx) => (
          <button
            key={idx}
            onClick={() => setSelectedMapping(mapping)}
            className={`p-4 rounded-xl border-2 text-left transition-all ${
              selectedMapping === mapping
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/20'
                : 'border-slate-200 dark:border-slate-700 hover:border-blue-300'
            }`}
          >
            <div className="font-semibold text-slate-900 dark:text-white mb-1">
              {mapping.bo_name}
            </div>
            <div className="flex items-center space-x-2 text-sm text-slate-600 dark:text-slate-400">
              <ArrowRight className="w-4 h-4" />
              <span>{mapping.view_name}</span>
            </div>
            <div className="mt-2 text-xs text-slate-500">
              {mapping.field_mappings.length} field mappings
            </div>
          </button>
        ))}
      </div>

      {/* Mapping Visualization */}
      {selectedMapping && (
        <div className="bg-white dark:bg-slate-800 rounded-2xl border border-slate-200 dark:border-slate-700 p-8">
          <div className="grid grid-cols-3 gap-8">
            {/* Business Object Fields */}
            <div>
              <h3 className="text-sm font-semibold text-slate-600 dark:text-slate-400 mb-4 uppercase">
                Business Object
              </h3>
              <div className="space-y-3">
                {selectedMapping.field_mappings.map((mapping, idx) => (
                  <div
                    key={idx}
                    className="p-3 bg-blue-50 dark:bg-blue-950/20 rounded-lg border-2 border-blue-200 dark:border-blue-800"
                  >
                    <div className="font-medium text-slate-900 dark:text-white text-sm">
                      {mapping.bo_field_name}
                    </div>
                    <div className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                      {mapping.bo_field_type}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Mapping Arrows */}
            <div className="flex flex-col justify-center">
              {selectedMapping.field_mappings.map((mapping, idx) => (
                <div key={idx} className="flex items-center justify-center py-3">
                  <div className="flex items-center space-x-2">
                    <div className={`w-16 h-0.5 ${
                      mapping.mapping_type === 'direct' 
                        ? 'bg-green-500' 
                        : mapping.mapping_type === 'computed'
                        ? 'bg-yellow-500'
                        : 'bg-purple-500'
                    }`}></div>
                    <div className={`p-1.5 rounded-full ${
                      mapping.mapping_type === 'direct'
                        ? 'bg-green-500'
                        : mapping.mapping_type === 'computed'
                        ? 'bg-yellow-500'
                        : 'bg-purple-500'
                    }`}>
                      {mapping.mapping_type === 'direct' ? (
                        <Check className="w-3 h-3 text-white" />
                      ) : (
                        <Sparkles className="w-3 h-3 text-white" />
                      )}
                    </div>
                    <div className={`w-16 h-0.5 ${
                      mapping.mapping_type === 'direct'
                        ? 'bg-green-500'
                        : mapping.mapping_type === 'computed'
                        ? 'bg-yellow-500'
                        : 'bg-purple-500'
                    }`}></div>
                  </div>
                </div>
              ))}
            </div>

            {/* Semantic View Fields */}
            <div>
              <h3 className="text-sm font-semibold text-slate-600 dark:text-slate-400 mb-4 uppercase">
                Semantic View
              </h3>
              <div className="space-y-3">
                {selectedMapping.field_mappings.map((mapping, idx) => (
                  <div
                    key={idx}
                    className="p-3 bg-indigo-50 dark:bg-indigo-950/20 rounded-lg border-2 border-indigo-200 dark:border-indigo-800"
                  >
                    <div className="font-medium text-slate-900 dark:text-white text-sm">
                      {mapping.view_field_name}
                    </div>
                    <div className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                      {mapping.view_field_type}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Legend */}
          <div className="mt-8 pt-6 border-t border-slate-200 dark:border-slate-700">
            <div className="flex items-center space-x-6 text-sm">
              <div className="flex items-center space-x-2">
                <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                <span className="text-slate-600 dark:text-slate-400">Direct Mapping</span>
              </div>
              <div className="flex items-center space-x-2">
                <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                <span className="text-slate-600 dark:text-slate-400">Computed</span>
              </div>
              <div className="flex items-center space-x-2">
                <div className="w-3 h-3 bg-purple-500 rounded-full"></div>
                <span className="text-slate-600 dark:text-slate-400">Derived</span>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default FieldMappingVisualizer;
