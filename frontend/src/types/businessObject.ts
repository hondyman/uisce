/**
 * Business Object Types and Interfaces
 * Defines the driving table pattern for business objects
 */

export interface BusinessObjectDefinition {
  bo_def_id: string;
  tenant_id: string;
  bo_key: string;          // e.g., 'customer', 'portfolio'
  name: string;
  display_name: string;
  description?: string;

  // Driver table
  driver_table_id?: string | null;
  driver_table_name?: string;

  status: 'draft' | 'active' | 'deprecated';
  enableHistory?: boolean;
  historyMode?: 'EXPLICIT_RANGE' | 'EVENT_LOG';
  config?: {
    is_active?: boolean;
    icons?: string;
    ui_hints?: Record<string, any>;
    governance?: Record<string, any>;
  };

  created_at: Date;
  created_by?: string;
  updated_at: Date;
  updated_by?: string;
}

export interface FieldDefinition {
  field_def_id: string;
  tenant_id: string;
  bo_def_id: string;

  field_key: string;           // 'name', 'email', 'risk_score'
  display_name: string;
  technical_name?: string;
  field_type: 'string' | 'number' | 'date' | 'boolean' | 'json' | 'array';
  role?: 'DIMENSION' | 'MEASURE' | 'VALIDITY_START' | 'VALIDITY_END' | 'EVENT_DATE' | 'PARTITION_KEY';
  semantic_term_id?: string;

  is_required: boolean;
  is_multi_value: boolean;
  is_core: boolean;

  json_schema?: {
    minimum?: number;
    maximum?: number;
    pattern?: string;
    enum?: any[];
    format?: string;
    [key: string]: any;
  };

  created_at: Date;
}

export interface SubtypeDefinition {
  subtype_def_id: string;
  tenant_id: string;
  bo_def_id: string;

  subtype_key: string;      // 'taxable', 'ira', 'trust'
  name: string;
  display_name: string;
  description?: string;

  created_at: Date;
  created_by?: string;

  // Includes applicable fields
  fields?: FieldDefinition[];
}

export interface SubtypeFieldMapping {
  tenant_id: string;
  subtype_def_id: string;
  field_def_id: string;
  is_required_override?: boolean;  // null means use base is_required
}

export interface BusinessObjectInstance {
  bo_instance_id: string;
  tenant_id: string;

  bo_def_id: string;
  subtype_def_id?: string | null;

  external_ref?: string;      // Optional external ID (CRM, etc.)
  title?: string;             // Human-readable label

  core_field_values: Record<string, any>;
  custom_field_values: Record<string, any>;

  status: 'active' | 'inactive' | 'archived';

  created_at: Date;
  created_by?: string;
  updated_at: Date;
  updated_by?: string;

  is_deleted: boolean;
  deleted_at?: Date;
}

export interface RelatedObject {
  relationship_id: string;
  tenant_id: string;

  from_instance_id: string;
  to_instance_id?: string | null;

  // For linking to hard tables
  to_hard_table_name?: string;
  to_hard_table_id?: string;

  relationship_type: string;  // 'owns', 'depends_on', 'member_of', etc.
  properties?: Record<string, any>;

  created_at: Date;
  created_by?: string;
}

export interface AuditLogEntry {
  audit_id: string;
  tenant_id: string;

  entity_type: 'business_object' | 'instance' | 'relationship';
  entity_id: string;

  action: 'CREATE' | 'UPDATE' | 'DELETE';
  changes: Record<string, any>;

  created_by?: string;
  created_at: Date;
}

// API Request/Response Types
export interface CreateBusinessObjectRequest {
  bo_key: string;
  name: string;
  display_name: string;
  description?: string;
  driver_table_id?: string;
  driver_table_name?: string;
  status?: 'draft' | 'active' | 'deprecated';
  enable_history?: boolean;
  history_mode?: 'EXPLICIT_RANGE' | 'EVENT_LOG';
  config?: Record<string, any>;
}

export interface UpdateBusinessObjectRequest {
  name?: string;
  display_name?: string;
  description?: string;
  driver_table_id?: string;
  driver_table_name?: string;
  status?: 'draft' | 'active' | 'deprecated';
  enable_history?: boolean;
  history_mode?: 'EXPLICIT_RANGE' | 'EVENT_LOG';
  config?: Record<string, any>;
}

export interface CreateInstanceRequest {
  bo_def_id: string;
  subtype_def_id?: string;
  external_ref?: string;
  title?: string;
  core_field_values: Record<string, any>;
  custom_field_values?: Record<string, any>;
}

export interface UpdateInstanceRequest {
  core_field_values?: Record<string, any>;
  custom_field_values?: Record<string, any>;
  status?: string;
  title?: string;
}

export interface CreateRelationshipRequest {
  from_instance_id: string;
  to_instance_id?: string;
  to_hard_table_name?: string;
  to_hard_table_id?: string;
  relationship_type: string;
  properties?: Record<string, any>;
}

// Enums
export const FIELD_TYPES = ['string', 'number', 'date', 'boolean', 'json', 'array'] as const;
export const BO_STATUS = ['draft', 'active', 'deprecated'] as const;
export const INSTANCE_STATUS = ['active', 'inactive', 'archived'] as const;
export const RELATIONSHIP_TYPES = [
  'owns',
  'depends_on',
  'member_of',
  'generated_by',
  'linked_to',
  'derived_from',
  'contains',
  'is_contained_in',
] as const;
