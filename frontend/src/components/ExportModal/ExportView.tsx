import { useState, useMemo, useEffect, FC } from 'react';
import { devError } from '../../utils/devLogger';
import { Node as FlowNode, Edge } from 'reactflow';
import { ExportModalHeader } from './ExportModalHeader';
import { ExportModalFooter } from './ExportModalFooter';
import { FormatSelector } from './FormatSelector';
import { AdvancedOptions } from './AdvancedOptions';
import { ScopeSelector } from './ScopeSelector';
import { SelectionPanels } from './SelectionPanels';
import { ExportPreview } from './ExportPreview';
import { ExportOptions } from '../../types/ExportTypes';
import { getNodeSchema } from '../../utils/exportUtils';

export interface ExportViewProps {
  nodes: FlowNode[];
  edges: Edge[];
  onExport: (options: ExportOptions) => Promise<void>;
  onCancel: () => void;
}

export const ExportView: FC<ExportViewProps> = ({ nodes, edges, onExport, onCancel }) => {
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
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    setIsVisible(true);
  }, []);

  const handleClose = () => {
    setIsVisible(false);
    setTimeout(onCancel, 300);
  };

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

  const filteredEdges = useMemo(() => {
    const nodeIds = new Set(filteredNodes.map(n => n.id));
    return edges.filter(edge => nodeIds.has(edge.source) && nodeIds.has(edge.target));
  }, [filteredNodes, edges]);

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await onExport(exportOptions);
      handleClose();
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

  const estimatedSize = (filteredNodes.length * 0.05).toFixed(1); // Rough estimate

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
    handleOptionChange('selectedSchemas', availableSchemas);
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

  return (
    <div className={`export-modal-root ${isVisible ? 'visible' : 'hidden'}`}>
      <div className={`export-modal-dialog ${isVisible ? 'visible' : 'hidden'}`}>
        <ExportModalHeader
          totalTables={nodes.length}
          totalSchemas={availableSchemas.length}
          onClose={handleClose}
        />

        <div className="flex max-h-[calc(85vh-160px)]">
          <div className="w-1/2 p-8 border-r border-gray-200 overflow-y-auto">
            <div className="space-y-8">
              <FormatSelector
                selectedFormat={exportOptions.format}
                onFormatChange={value => handleOptionChange('format', value)}
              />
              <AdvancedOptions
                options={exportOptions}
                onOptionChange={handleOptionChange}
              />
              <ScopeSelector
                selectedScope={exportOptions.exportScope}
                onScopeChange={value => handleOptionChange('exportScope', value)}
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
            </div>
          </div>

          <div className="w-1/2 p-8 bg-gray-50 overflow-y-auto">
            <ExportPreview
              filteredNodes={filteredNodes}
              filteredEdges={filteredEdges}
              getNodeSchema={getNodeSchema}
              exportOptions={exportOptions}
              estimatedSize={estimatedSize}
            />
          </div>
        </div>

        <ExportModalFooter
          onClose={handleClose}
          onExport={handleExport}
          canExport={canExport}
          isExporting={isExporting}
          estimatedSize={estimatedSize}
        />
      </div>
    </div>
  );
};