import React, { useState, useMemo } from 'react';
import { devError } from '../../utils/devLogger';
import { Node as FlowNode } from 'reactflow';
import { ExportDialogProps, ExportOptions } from '../../types/ExportTypes';
import { estimateFileSize } from '../../services/exportService';
import { getNodeSchema } from '../../utils/exportUtils';

// Import all the child components
import { ExportModalHeader } from './ExportModalHeader';
import { FormatSelector } from './FormatSelector';
import { CSVConfiguration } from './CSVConfiguration';
import { ScopeSelector } from './ScopeSelector';
import { SelectionPanels } from './SelectionPanels';
import { AdvancedOptions } from './AdvancedOptions';
import { ExportPreview } from './ExportPreview';
import { ExportModalFooter } from './ExportModalFooter';

const ExportModal: React.FC<ExportDialogProps> = ({ nodes, edges, onClose, onExport }) => {
  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    format: 'csv',
    delimiter: ',',
    exportScope: 'all',
    includeRelationships: true,
    includeIndexes: false,
    includeComments: false,
    selectedSchemas: [],
    selectedTables: [],
  });

  const [isExporting, setIsExporting] = useState(false);

  // --- FIX: RESTORED LOGIC FOR ALL MEMOIZED CALCULATIONS ---
  const schemaGroups = useMemo(() => {
    const groups: { [schema: string]: FlowNode[] } = {};
    nodes.forEach(node => {
      const schema = getNodeSchema(node);
      if (!groups[schema]) groups[schema] = [];
      groups[schema].push(node);
    });
    return groups; // This return statement is crucial
  }, [nodes]);

  const filteredNodes = useMemo(() => {
    if (exportOptions.exportScope === 'all') {
      return nodes;
    }
    if (exportOptions.exportScope === 'schemas') {
      return nodes.filter(node => exportOptions.selectedSchemas.includes(getNodeSchema(node)));
    }
    if (exportOptions.exportScope === 'tables') {
      return nodes.filter(node => exportOptions.selectedTables.includes(node.id));
    }
    return nodes; // This return statement is crucial
  }, [nodes, exportOptions]);

  const filteredEdges = useMemo(() => {
    const nodeIds = new Set(filteredNodes.map(n => n.id));
    return edges.filter(edge => nodeIds.has(edge.source) && nodeIds.has(edge.target)); // This return statement is crucial
  }, [filteredNodes, edges]);

  const estimatedSize = useMemo(() => {
    return estimateFileSize(filteredNodes, exportOptions.format); // This return statement is crucial
  }, [filteredNodes, exportOptions.format]);
  
  const canExport = useMemo(() => (
    exportOptions.exportScope === 'all' ||
    (exportOptions.exportScope === 'schemas' && exportOptions.selectedSchemas.length > 0) ||
    (exportOptions.exportScope === 'tables' && exportOptions.selectedTables.length > 0)
  ), [exportOptions]);


  // --- EVENT HANDLERS ---
  const handleOptionChange = (key: keyof ExportOptions, value: any) => {
    setExportOptions(prev => ({ ...prev, [key]: value }));
  };

  const handleSchemaToggle = (schema: string) => {
    const newSelection = exportOptions.selectedSchemas.includes(schema)
      ? exportOptions.selectedSchemas.filter(s => s !== schema)
      : [...exportOptions.selectedSchemas, schema];
    handleOptionChange('selectedSchemas', newSelection);
  };

  const handleTableToggle = (tableId: string) => {
    const newSelection = exportOptions.selectedTables.includes(tableId)
      ? exportOptions.selectedTables.filter(t => t !== tableId)
      : [...exportOptions.selectedTables, tableId];
    handleOptionChange('selectedTables', newSelection);
  };
  
  const handleSelectAllSchemas = () => {
    handleOptionChange('selectedSchemas', Object.keys(schemaGroups));
  };

  const handleClearSchemaSelection = () => {
      handleOptionChange('selectedSchemas', []);
  };
  
  const handleSelectAllTables = () => {
    handleOptionChange('selectedTables', nodes.map(n => n.id));
  };
  
  const handleClearTableSelection = () => {
      handleOptionChange('selectedTables', []);
  };

  const handleExport = async () => {
    setIsExporting(true);
    try {
      if (onExport) {
        await onExport(exportOptions);
      }
      onClose();
    } catch (error) {
      try { devError('Export failed:', error); } catch {}
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <div className="w-full max-w-6xl mx-auto bg-white rounded-2xl overflow-hidden flex flex-col max-h-[90vh]">
      <ExportModalHeader
        totalTables={nodes.length}
        totalSchemas={Object.keys(schemaGroups).length}
        onClose={onClose}
      />

      <div className="flex flex-grow min-h-0">
        <div className="w-1/2 p-8 border-r border-gray-200 overflow-y-auto space-y-8">
          <FormatSelector
            selectedFormat={exportOptions.format}
            onFormatChange={(value) => handleOptionChange('format', value)}
          />

          {exportOptions.format === 'csv' && (
            <CSVConfiguration
              delimiter={exportOptions.delimiter || ','}
              onDelimiterChange={(value) => handleOptionChange('delimiter', value)}
            />
          )}

          <ScopeSelector
            selectedScope={exportOptions.exportScope}
            onScopeChange={(value) => handleOptionChange('exportScope', value)}
          />

          <SelectionPanels
            exportScope={exportOptions.exportScope}
            schemaGroups={schemaGroups}
            selectedSchemas={exportOptions.selectedSchemas}
            selectedTables={exportOptions.selectedTables}
            onSchemaToggle={handleSchemaToggle}
            onTableToggle={handleTableToggle}
            onSelectAllSchemas={handleSelectAllSchemas}
            onClearSchemaSelection={handleClearSchemaSelection}
            onSelectAllTables={handleSelectAllTables}
            onClearTableSelection={handleClearTableSelection}
          />
          
          <AdvancedOptions
            options={exportOptions}
            onOptionChange={handleOptionChange}
          />
        </div>
        <div className="w-1/2 p-8 bg-gray-50 overflow-y-auto">
          <ExportPreview
            filteredNodes={filteredNodes}
            filteredEdges={filteredEdges}
            estimatedSize={estimatedSize}
            exportOptions={exportOptions}
            getNodeSchema={getNodeSchema}
          />
        </div>
      </div>

      <ExportModalFooter
        estimatedSize={estimatedSize}
        canExport={canExport}
        isExporting={isExporting}
        onClose={onClose}
        onExport={handleExport}
      />
    </div>
  );
};

export default ExportModal;