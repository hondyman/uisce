// types/ExportTypes.ts
import { Node as FlowNode, Edge } from 'reactflow';

export interface ExportOptions {
  format: 'csv' | 'json' | 'xml';
  delimiter?: string;
  includeRelationships: boolean;
  includeIndexes: boolean;
  includeComments: boolean;
  selectedSchemas: string[];
  selectedTables: string[];
  exportScope: 'all' | 'schemas' | 'tables';
}

export interface ExportDialogProps {
  nodes: FlowNode[];
  edges: Edge[];
  onClose: () => void;
  onExport?: (options: ExportOptions) => void;
}