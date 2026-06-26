// Edge Type related types for catalog management

export interface EdgeProperty {
  name: string;
  label: string;
  data_type: 'string' | 'integer' | 'boolean' | 'date' | 'float' | 'json' | 'text';
  nullable: boolean;
  default_value?: any;
  input_type: 'text' | 'textarea' | 'select' | 'checkbox' | 'date-picker' | 'number' | 'json-editor' | 'code-editor';
  format?: string; // display format or validation pattern
  validation?: {
    min?: number;
    max?: number;
    minLength?: number;
    maxLength?: number;
    pattern?: string;
    required?: boolean;
    [key: string]: any;
  };
  options?: string[]; // for select/dropdown inputs
  // Optional syntax language for code-editor input type (sql, yaml, json)
  syntax_language?: 'sql' | 'yaml' | 'json' | null;
  order: number;
  is_array?: boolean;
  lookup_config?: {
    node_type_id: string; // The ID of the node type to look up
    filter?: any; // Future: specialized filters
  };
}

export interface EdgeType {
  id: string;
  tenant_id: string;
  edge_type_name: string; // The relationship name (e.g., "contains", "references", "has_parent")
  description: string;
  is_active: boolean;
  subject_node_type_id: string; // The "from" node type
  object_node_type_id: string; // The "to" node type
  subject_node_type_name?: string; // Display name of subject node type
  object_node_type_name?: string; // Display name of object node type
  type?: 'core' | 'custom'; // Core = created by gold copy tenant, Custom = custom tenant
  properties?: EdgeProperty[]; // JSONB array of properties
  config?: Record<string, any>; // Configuration including color
  created_at: string;
  updated_at: string;
  core_id?: string | null;
}

export interface CreateEdgeTypeRequest {
  tenant_id: string;
  edge_type_name: string;
  description: string;
  is_active?: boolean;
  subject_node_type_id: string;
  object_node_type_id: string;
  properties?: EdgeProperty[]; // JSONB array of properties
}

export interface UpdateEdgeTypeRequest {
  edge_type_name?: string;
  description?: string;
  is_active?: boolean;
  subject_node_type_id?: string;
  object_node_type_id?: string;
  properties?: EdgeProperty[]; // JSONB array of properties
  config?: Record<string, any>; // Configuration including color
}

export interface CreatePropertyRequest {
  edge_type_id: string;
  tenant_id: string;
  property: EdgeProperty;
}

export interface UpdatePropertyRequest {
  edge_type_id: string;
  tenant_id: string;
  property_name: string;
  property: Partial<EdgeProperty>;
}

export interface DeletePropertyRequest {
  edge_type_id: string;
  tenant_id: string;
  property_name: string;
}
