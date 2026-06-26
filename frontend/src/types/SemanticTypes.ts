import { Node as FlowNode, Edge } from 'reactflow';

export { type Node as ReactFlowNode } from 'reactflow';

export interface ColumnData {
  id?: string; // Added missing id property
  name: string;
  type: string;
  data_type?: string;
  description?: string;
  isCore?: boolean;
  nullable?: boolean;
  qualifiedPath?: string; // Added missing qualifiedPath property
  isPrimaryKey?: boolean;
  isForeignKey?: boolean;
}

export interface EnhancedSelectedAsset {
  type: 'table' | 'column' | 'business_term' | 'semantic_term' | 'semantic_column' | 'database_column' | 'semantic_model' | 'schema';
  id: string;
  nodeId: string;
  name: string;
  isCore?: boolean;
  description?: string;
  // Optional properties for specific asset types
  columnName?: string;
  tableName?: string;
  qualifiedPath?: string; // Added missing qualifiedPath property
  businessTerm?: any; // Define more specifically based on your business term structure
  semanticTerm?: any; // Define more specifically based on your semantic term structure
  semanticModel?: any; // Define more specifically based on your semantic model structure
  // Additional properties used in BusinessTermsTree
  node?: any; // The original node data
  column?: any; // Column data for column assets
  columns?: ColumnData[]; // For table nodes, an array of columns
}

export interface TableNodeData {
  schemaName?: string;
  tableName?: string;
  label?: string;
  isCore?: boolean;
  columns?: ColumnData[];
}

export interface TechnicalLineageData {
  nodes: Array<{
    id: string;
    data?: TableNodeData;
    position?: { x: number; y: number };
  }>;
  edges: Edge[];
  viewport: Record<string, unknown>;
  metadata: Record<string, any>;
}

export interface SemanticNode {
  id: string;
  node_name: string;
  node_type: 'business_term' | 'semantic_term' | 'semantic_column' | 'database_column' | 'semantic_model' | 'schema';
  description: string;
  qualified_path: string;
  properties: Record<string, any>;
}

export interface SemanticEdge {
  id: string;
  source_node_id: string;
  target_node_id: string;
  edge_type_id: string;
  relationship_type: string;
  properties: Record<string, unknown>;
}

export interface RawSemanticChart {
  businessTerms: SemanticNode[];
  semanticTerms: SemanticNode[];
  semanticColumns: SemanticNode[];
  databaseColumns: SemanticNode[];
  edges: SemanticEdge[];
  viewport: Record<string, unknown>;
  metadata: Record<string, any>;
}

export interface ReactFlowSemanticChart {
  nodes: FlowNode[];
  edges: Edge[];
  viewport: Record<string, unknown>;
  metadata: Record<string, any>;
}

export type SemanticLineageData = RawSemanticChart | ReactFlowSemanticChart;