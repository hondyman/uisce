// Node Type related types for catalog management

export interface NodeProperty {
  name: string;
  label: string;
  data_type: 'string' | 'integer' | 'boolean' | 'date' | 'float' | 'json' | 'text' | 'array';
  nullable: boolean;
  default_value?: any;
  input_type: 'text' | 'textarea' | 'select' | 'checkbox' | 'date-picker' | 'number' | 'json-editor' | 'chips' | 'lookup' | 'code-editor';
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
  // Optional lookup id for lookup-based properties (references a lookup table id)
  lookup_id?: string | null;
  cascade_from?: string | null;
  // Optional syntax language for code-editor input type (sql, yaml, json)
  syntax_language?: 'sql' | 'yaml' | 'json' | null;
  order: number;
}

export interface NodeType {
  id: string;
  tenant_id: string;
  catalog_type_name: string;
  description: string;
  is_active: boolean;
  type?: string; // "core" or "custom" from API
  parent_type_id?: string | null;
  config?: {
    [key: string]: any;
  };
  // New input type for referencing an external lookup table is represented on NodeProperty above
  properties?: NodeProperty[]; // Top-level properties JSONB field
  created_at: string;
  updated_at: string;
}

export interface CreateNodeTypeRequest {
  tenant_id: string;
  catalog_type_name: string;
  description: string;
  is_active?: boolean;
  parent_type_id?: string | null;
  config?: {
    [key: string]: any;
  };
  properties?: NodeProperty[]; // Top-level properties field
}

export interface UpdateNodeTypeRequest {
  catalog_type_name?: string;
  description?: string;
  is_active?: boolean;
  parent_type_id?: string | null;
  config?: {
    [key: string]: any;
  };
  properties?: NodeProperty[]; // Top-level properties field
}

export interface CreatePropertyRequest extends Omit<NodeProperty, 'order'> {
  order?: number;
}

export interface UpdatePropertyRequest extends Partial<NodeProperty> {
  name: string;
}

// Data type options for property configuration
export const DATA_TYPE_OPTIONS = [
  { value: 'string', label: 'String' },
  { value: 'integer', label: 'Integer' },
  { value: 'float', label: 'Float' },
  { value: 'boolean', label: 'Boolean' },
  { value: 'date', label: 'Date' },
  { value: 'text', label: 'Text (Long)' },
  { value: 'json', label: 'JSON' },
] as const;

// Input type options for property configuration
export const INPUT_TYPE_OPTIONS = [
  { value: 'text', label: 'Text Input' },
  { value: 'textarea', label: 'Text Area' },
  { value: 'number', label: 'Number Input' },
  { value: 'checkbox', label: 'Checkbox' },
  { value: 'select', label: 'Select/Dropdown' },
  { value: 'chips', label: 'Chips / Multi-select' },
  { value: 'date-picker', label: 'Date Picker' },
  { value: 'json-editor', label: 'JSON Editor' },
  { value: 'lookup', label: 'Lookup (Reference Table)' },
] as const;
