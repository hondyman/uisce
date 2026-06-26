import { ColumnData } from '../../types/SemanticTypes';

export interface DrillDownLocator {
  xValues: any[];
  yValues: any[];
}

export interface PivotConfig {
  x?: string[];
  y?: string[];
}

export interface Query {
  measures: string[];
  dimensions: string[];
  filters: any[];
  timeDimensions: any[];
}

export interface PivotRow {
  xValues: any[];
  yValuesArray: any[][];
}

export interface ColumnMapping {
  nodeId: string;
  tableName: string;
  columnName: string;
  columnType: string;
  mappingType: 'dimension' | 'measure' | 'filter' | 'none';
  mappingId?: string;
}

export interface SemanticElement {
  id: string;
  name: string;
  type: string;
  sql: string;
  title?: string;
  description?: string;
  sourceTable: string;
  sourceColumn: string;
  businessTermId?: string;
  isEditing?: boolean;
  format?: string;
  public?: boolean;
  is_custom?: boolean;
  drillMembers?: string[];
  drillDown?: (drillDownLocator: DrillDownLocator, pivotConfig?: PivotConfig) => Query | null;
  pivot?: (pivotConfig?: PivotConfig) => PivotRow[];
}

export interface SemanticModel {
  id?: string;
  name: string;
  dimensions: SemanticElement[];
  measures: SemanticElement[];
  filters: SemanticElement[];
  joins: Array<{
    id: string;
    name: string;
    sql: string;
    relationship: string;
    leftTable: string;
    rightTable: string;
    is_custom?: boolean;
    isEditing?: boolean;
  }>;
  is_custom?: boolean;
  // Optional backend/catalog properties (added to satisfy ModelCatalog usage)
  model_key?: string;
  display_name?: string;
  resolved_config?: any;
  source_config?: any;
  parent_model_key?: string;
  is_core?: boolean;
  status?: string;
  version?: string | number;
  is_current?: boolean;
  metadata?: any;
  custom_model_exists?: boolean;
  description?: string;
  created_at?: string;
  created_by?: string;
  can_edit?: boolean; // Indicates if the model can be edited
  core_model_exists?: boolean; // Indicates if a core model exists

  // NEW: Basic Identification & Description
  technical_name?: string;  // Internal unique identifier

  // NEW: Core vs. Custom Status
  model_type?: 'core' | 'custom';  // Core (read-only) or Custom (user-defined)

  // NEW: Data Source Information
  data_source_description?: string;  // Description of underlying data source(s)
  schema_table_reference?: string;  // Database schema.table or file paths

  // NEW: Extension & Inheritance
  extends_model_id?: string;  // Reference to Core or Custom model this extends
  extends_model_name?: string;  // Resolved name for display

  // NEW: Relationship to Semantic Terms
  linked_semantic_terms?: string[];  // Array of semantic term IDs
  linked_semantic_term_names?: string[];  // Resolved names for display

  // NEW: Model-Specific Transformations
  overridden_properties?: Record<string, any>;  // Override properties for inherited semantic terms
  model_calculations?: Record<string, any>;  // Complex calculations combining semantic terms
}

export interface BusinessTerm {
  id: string;
  node_name: string;
  description?: string;
  properties?: any;
  qualified_path?: string;
  isCore?: boolean;
}

export interface SemanticTerm {
  id: string;
  node_name: string;
  description?: string;
  properties?: any;
  qualified_path?: string;
  parent_id?: string;
}

export interface SemanticView {
  id: string;
  node_name: string;
  description?: string;
  properties?: any;
  qualified_path?: string;
  parent_id?: string;
}

export interface BusinessEdge {
  id: string;
  source_node_id: string;
  target_node_id: string;
  relationship_type: string;
  properties?: any;
}

export interface CollapsedTables {
  [tableId: string]: boolean;
}

export type ShowCode = 'json' | 'yaml' | null;

export interface SelectedColumn {
  nodeId: string;
  tableName: string;
  column: ColumnData;
  id: string;
}
