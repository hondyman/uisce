// Fabric-related type definitions
export interface FabricModel {
  id: string;
  name: string;
  description: string;
  tenant_id: string;
  datasource_id: string;
  schema: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface Extension {
  id: string;
  name: string;
  type: string;
  tenant_id: string;
  datasource_id: string;
  config: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface ValidationResult {
  valid: boolean;
  errors: string[];
  model: FabricModel;
}

export interface CompatibilityReport {
  datasource_id: string;
  compatible: boolean;
  report: string;
}