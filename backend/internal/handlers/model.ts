export interface ModelCatalogNode {
  id: string;
  model_key: string;
  display_name: string;
  description: string;
  status: 'draft' | 'published' | 'archived';
  version: number;
  is_current: boolean;
  is_core: boolean;
  is_custom: boolean;
  can_edit: boolean;
  parent_model_key?: string;
  core_model_exists: boolean;
  custom_model_exists: boolean;
  source_config: any;
  resolved_config: any;
  created_at: string;
  updated_at?: string;
  published_at?: string;
  metadata: {
    generator: string;
    table_count: number;
    measure_count: number;
    dimension_count: number;
    has_custom_version?: boolean;
    inherits_from?: string;
    can_create?: boolean;
  };
}