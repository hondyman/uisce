/**
 * types.ts
 * TypeScript types and interfaces for Business Process Designer
 */

export interface ProcessStepType {
  id: string;
  key: string;
  label: string;
  description?: string;
  icon_svg?: string;
  default_data: Record<string, unknown>;
}

export interface ValidationOperator {
  id: string;
  key: string;
  label: string;
  description?: string;
  value_type: 'string' | 'number' | 'boolean' | 'list' | 'date' | 'currency';
  config?: Record<string, unknown>;
}

export interface WorkflowEvent {
  id: string;
  key: string;
  label: string;
  description?: string;
  event_type: 'on_start' | 'on_update' | 'on_submit' | 'on_approval' | 'on_completion' | 'custom';
  config?: Record<string, unknown>;
}

export interface BusinessObjectField {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'date' | 'currency';
  label: string;
}

export interface BusinessObject {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  fields: BusinessObjectField[];
  icon?: string;
  config?: Record<string, unknown>;
}

export interface ValidationRule {
  id?: string;
  process_id?: string;
  node_id?: string;
  field: string;
  field_label?: string;
  op: string;
  op_label?: string;
  value: string;
  message: string;
  severity?: 'block' | 'warning' | 'info';
  order_index?: number;
  enabled?: boolean;
  config?: Record<string, unknown>;
}

export interface ProcessNode {
  id: string;
  type: string;
  data: {
    label: string;
    stepKey?: string;
    config?: Record<string, unknown>;
    eventId?: string;
    rules?: ValidationRule[];
    onFailure?: 'reject' | 'route' | 'escalate';
  };
  position: {
    x: number;
    y: number;
  };
}

export interface ProcessEdge {
  id: string;
  source: string;
  target: string;
}

export interface Process {
  id: string;
  name: string;
  description?: string;
  version: number;
  nodes: ProcessNode[];
  edges: ProcessEdge[];
  config?: Record<string, unknown>;
  status: 'draft' | 'published' | 'archived';
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
  tenant_id?: string;
  tenant_instance_id?: string;
}

export interface ValidationConfig {
  eventId?: string;
  rules: ValidationRule[];
  onFailure?: 'reject' | 'route' | 'escalate';
  escalationRole?: string;
}
