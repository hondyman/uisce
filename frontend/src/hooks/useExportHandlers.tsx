// hooks/useExportHandlers.ts
import { Node as FlowNode, Edge } from 'reactflow';
import { devError } from '../utils/devLogger';
import { ExportOptions } from '../types/ExportTypes';
// helper utilities for export (getTableName removed as unused in this hook)
import { exportToCSV, exportToJSON, exportToXML } from '../services/exportService';

interface UseExportHandlersProps {
  exportOptions: ExportOptions;
  setExportOptions: React.Dispatch<React.SetStateAction<ExportOptions>>;
  nodes: FlowNode[];
  filteredNodes: FlowNode[];
  filteredEdges: Edge[];
  schemaGroups: { [schema: string]: FlowNode[] };
  setIsExporting: React.Dispatch<React.SetStateAction<boolean>>;
  onClose: () => void;
}

export const useExportHandlers = ({
  exportOptions,
  setExportOptions,
  nodes,
  filteredNodes,
  filteredEdges,
  schemaGroups,
  setIsExporting,
  onClose
}: UseExportHandlersProps) => {
  
  const handleSchemaToggle = (schema: string) => {
    setExportOptions(prev => ({
      ...prev,
      selectedSchemas: prev.selectedSchemas.includes(schema)
        ? prev.selectedSchemas.filter(s => s !== schema)
        : [...prev.selectedSchemas, schema]
    }));
  };

  const handleTableToggle = (tableId: string) => {
    setExportOptions(prev => ({
      ...prev,
      selectedTables: prev.selectedTables.includes(tableId)
        ? prev.selectedTables.filter(t => t !== tableId)
        : [...prev.selectedTables, tableId]
    }));
  };

  const selectAllSchemas = () => {
    setExportOptions(prev => ({
      ...prev,
      selectedSchemas: Object.keys(schemaGroups)
    }));
  };

  const selectAllTables = () => {
    setExportOptions(prev => ({
      ...prev,
      selectedTables: nodes.map(n => n.id)
    }));
  };

  const clearSchemaSelection = () => {
    setExportOptions(prev => ({
      ...prev,
      selectedSchemas: []
    }));
  };

  const clearTableSelection = () => {
    setExportOptions(prev => ({
      ...prev,
      selectedTables: []
    }));
  };

  const handleExport = async () => {
    setIsExporting(true);
    
    try {
      let content = '';
      let filename = '';
      let mimeType = '';
      
      const timestamp = new Date().toISOString().split('T')[0];
      const scopePrefix = exportOptions.exportScope === 'all' ? 'full' : 
                         exportOptions.exportScope === 'schemas' ? `${exportOptions.selectedSchemas.length}schemas` :
                         `${exportOptions.selectedTables.length}tables`;
      
      switch (exportOptions.format) {
        case 'csv':
          content = exportToCSV(filteredNodes, filteredEdges, exportOptions);
          filename = `data-catalog-${scopePrefix}-${timestamp}.csv`;
          mimeType = 'text/csv';
          break;
        case 'json':
          content = exportToJSON(filteredNodes, filteredEdges, exportOptions);
          filename = `data-catalog-${scopePrefix}-${timestamp}.json`;
          mimeType = 'application/json';
          break;
        case 'xml':
          content = exportToXML(filteredNodes, filteredEdges, exportOptions);
          filename = `data-catalog-${scopePrefix}-${timestamp}.xml`;
          mimeType = 'application/xml';
          break;
      }
      
      const blob = new Blob([content], { type: mimeType });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      
      onClose();
    } catch (error) {
      devError('Export failed:', error);
    } finally {
      setIsExporting(false);
    }
  };

  return {
    handleSchemaToggle,
    handleTableToggle,
    selectAllSchemas,
    selectAllTables,
    clearSchemaSelection,
    clearTableSelection,
    handleExport
  };
};