import { useState, useMemo, FC } from 'react';
import { devError } from '../../utils/devLogger';
import { Node as FlowNode, Edge } from 'reactflow';
import { ExportOptions } from '../../types/ExportTypes';
import { getNodeSchema } from '../../utils/exportUtils';

export interface ExportOverlayProps {
  nodes: FlowNode[];
  edges: Edge[];
  onExport: (options: ExportOptions) => Promise<void>;
  onCancel: () => void;
}

export const ExportOverlay: FC<ExportOverlayProps> = ({ 
  nodes, 
  edges, 
  onExport, 
  onCancel 
}) => {
  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    format: 'csv',
    delimiter: ',',
    includeRelationships: false,
    selectedSchemas: [],
    selectedTables: [],
    exportScope: 'all',
    includeIndexes: false,
    includeComments: false,
  });

  const [isExporting, setIsExporting] = useState(false);

  const schemaGroups = useMemo(() => {
    const groups: { [schema: string]: FlowNode[] } = {};
    nodes.forEach(node => {
      const schema = getNodeSchema(node);
      if (!groups[schema]) groups[schema] = [];
      groups[schema].push(node);
    });
    return groups;
  }, [nodes]);

  const availableSchemas = Object.keys(schemaGroups);
  const totalColumns = nodes.reduce((sum, node) => sum + (node.data?.columns?.length || 0), 0);
  const totalRelations = edges.length;

  const filteredNodes = useMemo(() => {
    if (exportOptions.exportScope === 'all') return nodes;
    if (exportOptions.exportScope === 'schemas') {
      return nodes.filter(node => {
        const schema = getNodeSchema(node);
        return exportOptions.selectedSchemas.includes(schema);
      });
    }
    if (exportOptions.exportScope === 'tables') {
      return nodes.filter(node => exportOptions.selectedTables.includes(node.id));
    }
    return nodes;
  }, [nodes, exportOptions]);

  const estimatedSize = (filteredNodes.length * 0.05).toFixed(1);

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await onExport(exportOptions);
    } catch (error) {
      try { devError('Export failed:', error); } catch {}
    } finally {
      setIsExporting(false);
    }
  };

  const canExport =
    exportOptions.exportScope === 'all' ||
    (exportOptions.exportScope === 'schemas' && exportOptions.selectedSchemas.length > 0) ||
    (exportOptions.exportScope === 'tables' && exportOptions.selectedTables.length > 0);

  const handleOptionChange = (key: keyof ExportOptions, value: any) => {
    setExportOptions(prev => ({ ...prev, [key]: value }));
  };

  // Format options
  const formatOptions = [
    { id: 'csv', label: 'CSV', subtitle: 'Excel Ready', icon: '📊' },
    { id: 'json', label: 'JSON', subtitle: 'API Friendly', icon: '</>' },
    { id: 'xml', label: 'XML', subtitle: 'Enterprise', icon: '📄' }
  ];

  // CSV delimiter options
  const delimiterOptions = [
    { value: ',', label: 'Comma (,)' },
    { value: ';', label: 'Semicolon (;)' },
    { value: '\t', label: 'Tab' },
    { value: '|', label: 'Pipe (|)' }
  ];

  // Sample data for preview
  const sampleData = [
    { schema: 'public', table: 'users', column: 'id', type: 'integer', primaryKey: true, foreignKey: false },
    { schema: 'public', table: 'users', column: 'username', type: 'varchar(255)', primaryKey: false, foreignKey: false },
    { schema: 'public', table: 'orders', column: 'user_id', type: 'integer', primaryKey: false, foreignKey: true }
  ];

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-6xl max-h-[90vh] overflow-hidden flex flex-col">
        
        {/* Header */}
        <div className="bg-blue-600 text-white p-6 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="bg-white bg-opacity-20 p-2 rounded-lg">
              <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6z"/>
                <path d="M14 2v6h6"/>
                <path d="M16 13H8"/>
                <path d="M16 17H8"/>
                <path d="M10 9H8"/>
              </svg>
            </div>
            <div>
              <h2 className="text-xl font-semibold">Export Data Catalog</h2>
              <p className="text-blue-100 text-sm">Configure and download your database schema</p>
            </div>
          </div>
          <div className="text-right">
            <div className="text-lg font-semibold">{nodes.length} Tables • {availableSchemas.length} Schemas</div>
            <div className="text-blue-100 text-sm">Ready to Export</div>
          </div>
        </div>

        <div className="flex flex-1 overflow-hidden">
          {/* Left Panel */}
          <div className="w-1/2 p-6 overflow-y-auto border-r border-gray-200">
            
            {/* 1. Export Format */}
            <div className="mb-8">
              <div className="flex items-center space-x-2 mb-4">
                <div className="w-6 h-6 bg-blue-600 text-white rounded-full flex items-center justify-center text-sm font-semibold">1</div>
                <h3 className="text-lg font-semibold text-gray-900">Export Format</h3>
              </div>
              
              <div className="grid grid-cols-3 gap-3 mb-4">
                {formatOptions.map((format) => (
                  <button
                    key={format.id}
                    onClick={() => handleOptionChange('format', format.id)}
                    className={`p-4 border-2 rounded-lg text-center transition-colors ${
                      exportOptions.format === format.id
                        ? 'border-blue-500 bg-blue-50'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <div className="text-2xl mb-1">{format.icon}</div>
                    <div className="font-semibold text-sm">{format.label}</div>
                    <div className="text-xs text-gray-500">{format.subtitle}</div>
                  </button>
                ))}
              </div>

              {/* CSV Configuration */}
              {exportOptions.format === 'csv' && (
                <div className="bg-gray-50 p-4 rounded-lg">
                  <div className="flex items-center space-x-2 mb-3">
                    <svg className="w-4 h-4 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
                      <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
                    </svg>
                    <span className="font-medium text-gray-700">CSV Configuration</span>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    {delimiterOptions.map((delim) => (
                      <button
                        key={delim.value}
                        onClick={() => handleOptionChange('delimiter', delim.value)}
                        className={`p-2 text-sm border rounded ${
                          exportOptions.delimiter === delim.value
                            ? 'border-blue-500 bg-blue-50 text-blue-700'
                            : 'border-gray-200 hover:border-gray-300'
                        }`}
                      >
                        {delim.label}
                      </button>
                    ))}
                  </div>
                  <input
                    type="text"
                    placeholder="Custom delimiter..."
                    className="w-full mt-2 p-2 text-sm border border-gray-200 rounded"
                    onChange={(e) => handleOptionChange('delimiter', e.target.value)}
                  />
                </div>
              )}
            </div>

            {/* 2. Data Scope */}
            <div className="mb-8">
              <div className="flex items-center space-x-2 mb-4">
                <div className="w-6 h-6 bg-purple-600 text-white rounded-full flex items-center justify-center text-sm font-semibold">2</div>
                <h3 className="text-lg font-semibold text-gray-900">Data Scope</h3>
              </div>
              
              <div className="space-y-3">
                <label className="flex items-start space-x-3 p-3 border rounded-lg hover:bg-gray-50 cursor-pointer">
                  <input
                    type="radio"
                    name="scope"
                    checked={exportOptions.exportScope === 'all'}
                    onChange={() => handleOptionChange('exportScope', 'all')}
                    className="mt-1"
                  />
                  <div>
                    <div className="flex items-center space-x-2">
                      <svg className="w-4 h-4 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                      </svg>
                      <span className="font-medium">Export Everything</span>
                    </div>
                    <div className="text-sm text-gray-500">All schemas, tables, and relationships</div>
                  </div>
                </label>

                <label className="flex items-start space-x-3 p-3 border rounded-lg hover:bg-gray-50 cursor-pointer">
                  <input
                    type="radio"
                    name="scope"
                    checked={exportOptions.exportScope === 'schemas'}
                    onChange={() => handleOptionChange('exportScope', 'schemas')}
                    className="mt-1"
                  />
                  <div>
                    <div className="flex items-center space-x-2">
                      <svg className="w-4 h-4 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6zm16-4H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm-1 9H9V9h10v2zm-4 4H9v-2h6v2zm4-8H9V5h10v2z"/>
                      </svg>
                      <span className="font-medium">Select Schemas</span>
                    </div>
                    <div className="text-sm text-gray-500">Choose specific database schemas</div>
                  </div>
                </label>

                <label className="flex items-start space-x-3 p-3 border rounded-lg hover:bg-gray-50 cursor-pointer">
                  <input
                    type="radio"
                    name="scope"
                    checked={exportOptions.exportScope === 'tables'}
                    onChange={() => handleOptionChange('exportScope', 'tables')}
                    className="mt-1"
                  />
                  <div>
                    <div className="flex items-center space-x-2">
                      <svg className="w-4 h-4 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M3 3v18h18V3H3zm16 16H5V5h14v14zm-8-2v-2h2v2h-2zm0-4V9h2v4h-2zm4 4v-2h2v2h-2zm0-4V9h2v4h-2z"/>
                      </svg>
                      <span className="font-medium">Select Tables</span>
                    </div>
                    <div className="text-sm text-gray-500">Pick individual tables</div>
                  </div>
                </label>
              </div>
            </div>

            {/* 3. Advanced Options */}
            <div className="mb-6">
              <div className="flex items-center space-x-2 mb-4">
                <div className="w-6 h-6 bg-green-600 text-white rounded-full flex items-center justify-center text-sm font-semibold">3</div>
                <h3 className="text-lg font-semibold text-gray-900">Advanced Options</h3>
              </div>
              
              <div className="space-y-4">
                <label className="flex items-center space-x-3 cursor-pointer">
                  <div className="flex items-center space-x-2">
                    <svg className="w-4 h-4 text-blue-500" fill="currentColor" viewBox="0 0 24 24">
                      <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                    </svg>
                    <span className="font-medium">Include Relationships</span>
                  </div>
                  <input
                    type="checkbox"
                    checked={exportOptions.includeRelationships}
                    onChange={(e) => handleOptionChange('includeRelationships', e.target.checked)}
                    className="ml-auto"
                  />
                </label>
                <div className="text-sm text-gray-500 ml-6">Export foreign keys and constraints</div>

                <label className="flex items-center space-x-3 cursor-pointer">
                  <div className="flex items-center space-x-2">
                    <svg className="w-4 h-4 text-orange-500" fill="currentColor" viewBox="0 0 24 24">
                      <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.94-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
                    </svg>
                    <span className="font-medium">Include Indexes</span>
                  </div>
                  <input
                    type="checkbox"
                    checked={exportOptions.includeIndexes}
                    onChange={(e) => handleOptionChange('includeIndexes', e.target.checked)}
                    className="ml-auto"
                  />
                </label>
                <div className="text-sm text-gray-500 ml-6">Export database indexes information</div>

                <label className="flex items-center space-x-3 cursor-pointer">
                  <div className="flex items-center space-x-2">
                    <svg className="w-4 h-4 text-purple-500" fill="currentColor" viewBox="0 0 24 24">
                      <path d="M21.99 4c0-1.1-.89-2-2-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h14l4 4-.01-18zM18 14H6v-2h12v2zm0-3H6V9h12v2zm0-3H6V6h12v2z"/>
                    </svg>
                    <span className="font-medium">Include Comments</span>
                  </div>
                  <input
                    type="checkbox"
                    checked={exportOptions.includeComments}
                    onChange={(e) => handleOptionChange('includeComments', e.target.checked)}
                    className="ml-auto"
                  />
                </label>
                <div className="text-sm text-gray-500 ml-6">Export table and column descriptions</div>
              </div>
            </div>
          </div>

          {/* Right Panel - Preview */}
          <div className="w-1/2 p-6 bg-gray-50 overflow-y-auto">
            
            {/* Export Preview Stats */}
            <div className="mb-6">
              <div className="flex items-center space-x-2 mb-4">
                <div className="w-6 h-6 bg-blue-600 text-white rounded-full flex items-center justify-center text-sm">👁</div>
                <h3 className="text-lg font-semibold text-gray-900">Export Preview</h3>
              </div>
              
              <div className="grid grid-cols-4 gap-4 mb-6">
                <div className="text-center p-4 bg-white rounded-lg border">
                  <div className="w-8 h-8 bg-blue-100 rounded-lg flex items-center justify-center mx-auto mb-2">
                    <svg className="w-5 h-5 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
                      <path d="M4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6zm16-4H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/>
                    </svg>
                  </div>
                  <div className="text-2xl font-bold text-gray-900">{availableSchemas.length}</div>
                  <div className="text-sm text-gray-500">SCHEMAS</div>
                </div>
                
                <div className="text-center p-4 bg-white rounded-lg border">
                  <div className="w-8 h-8 bg-purple-100 rounded-lg flex items-center justify-center mx-auto mb-2">
                    <svg className="w-5 h-5 text-purple-600" fill="currentColor" viewBox="0 0 24 24">
                      <path d="M3 3v18h18V3H3zm16 16H5V5h14v14z"/>
                    </svg>
                  </div>
                  <div className="text-2xl font-bold text-gray-900">{nodes.length}</div>
                  <div className="text-sm text-gray-500">TABLES</div>
                </div>
                
                <div className="text-center p-4 bg-white rounded-lg border">
                  <div className="w-8 h-8 bg-green-100 rounded-lg flex items-center justify-center mx-auto mb-2">
                    <svg className="w-5 h-5 text-green-600" fill="currentColor" viewBox="0 0 24 24">
                      <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
                    </svg>
                  </div>
                  <div className="text-2xl font-bold text-gray-900">{totalColumns}</div>
                  <div className="text-sm text-gray-500">COLUMNS</div>
                </div>
                
                <div className="text-center p-4 bg-white rounded-lg border">
                  <div className="w-8 h-8 bg-orange-100 rounded-lg flex items-center justify-center mx-auto mb-2">
                    <svg className="w-5 h-5 text-orange-600" fill="currentColor" viewBox="0 0 24 24">
                      <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                    </svg>
                  </div>
                  <div className="text-2xl font-bold text-gray-900">{totalRelations}</div>
                  <div className="text-sm text-gray-500">RELATIONS</div>
                </div>
              </div>
            </div>

            {/* File Output */}
            <div className="mb-6">
              <h4 className="font-semibold text-gray-900 mb-3">File Output</h4>
              <div className="bg-white p-4 rounded-lg border">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm text-gray-500">Ready to download</span>
                  <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded-full">
                    {exportOptions.format.toUpperCase()} Format
                  </span>
                </div>
                <div className="flex items-center space-x-2">
                  <svg className="w-5 h-5 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6z"/>
                  </svg>
                  <div>
                    <div className="font-medium">data-catalog-{new Date().toISOString().split('T')[0]}.{exportOptions.format}</div>
                    <div className="text-sm text-gray-500">Estimated size: ~{estimatedSize} MB</div>
                  </div>
                </div>
              </div>
            </div>

            {/* Data Sample Preview */}
            <div>
              <div className="flex items-center space-x-2 mb-3">
                <svg className="w-4 h-4 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
                </svg>
                <h4 className="font-semibold text-gray-900">Data Sample Preview</h4>
              </div>
              
              <div className="bg-white rounded-lg border overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-3 py-2 text-left font-medium text-gray-700">Schema</th>
                        <th className="px-3 py-2 text-left font-medium text-gray-700">Table</th>
                        <th className="px-3 py-2 text-left font-medium text-gray-700">Column</th>
                        <th className="px-3 py-2 text-left font-medium text-gray-700">Type</th>
                        <th className="px-3 py-2 text-center font-medium text-gray-700">Primary Key</th>
                        <th className="px-3 py-2 text-center font-medium text-gray-700">Foreign Key</th>
                      </tr>
                    </thead>
                    <tbody>
                      {sampleData.map((row, i) => (
                        <tr key={i} className="border-t">
                          <td className="px-3 py-2 text-gray-600">{row.schema}</td>
                          <td className="px-3 py-2 text-gray-900">{row.table}</td>
                          <td className="px-3 py-2 text-gray-900">{row.column}</td>
                          <td className="px-3 py-2 text-gray-600">{row.type}</td>
                          <td className="px-3 py-2 text-center">
                            {row.primaryKey && <div className="w-2 h-2 bg-red-500 rounded-full mx-auto"></div>}
                          </td>
                          <td className="px-3 py-2 text-center">
                            {row.foreignKey && <div className="w-2 h-2 bg-blue-500 rounded-full mx-auto"></div>}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <div className="px-3 py-2 bg-gray-50 text-xs text-gray-500 border-t">
                  Showing 3 of {totalColumns} columns
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="border-t border-gray-200 p-4 bg-gray-50 flex items-center justify-between">
          <div className="flex items-center space-x-4 text-sm text-gray-600">
            <span>⏱ Estimated export time: ~30 seconds</span>
            <span>📁 File size: ~{estimatedSize} MB</span>
          </div>
          
          <div className="flex items-center space-x-3">
            <button
              onClick={onCancel}
              className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            
            <button
              onClick={handleExport}
              disabled={!canExport || isExporting}
              className={`px-6 py-2 rounded-lg font-medium transition-colors flex items-center space-x-2 ${
                canExport && !isExporting
                  ? 'bg-blue-600 text-white hover:bg-blue-700'
                  : 'bg-gray-300 text-gray-500 cursor-not-allowed'
              }`}
            >
              {isExporting ? (
                <>
                  <div className="w-4 h-4 border-2 border-gray-300 border-t-white rounded-full animate-spin"></div>
                  <span>Exporting...</span>
                </>
              ) : (
                <>
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6z"/>
                    <path d="M14 2v6h6"/>
                    <path d="M16 13H8"/>
                    <path d="M16 17H8"/>
                  </svg>
                  <span>Export Data Catalog</span>
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};