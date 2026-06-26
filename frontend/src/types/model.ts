import type { JSONValue } from './json';
import type { SemanticModel } from '../components/UnifiedSemanticBuilder/types';

export interface ModelCatalogNode {
  id: string;
  model_key: string;
  // Title is the canonical name stored in backend (sometimes also surfaced as display_name)
  title?: string;
  display_name?: string;
  description?: string;
  status: 'draft' | 'published' | 'archived';
  version: number;
  is_current: boolean;
  is_core: boolean;
  is_custom: boolean;
  can_edit: boolean;
  parent_model_key?: string;
  core_model_exists: boolean;
  custom_model_exists: boolean;
  // Optional source/resolved configs (structure defined elsewhere)
  source_config?: JSONValue;
  // resolved_config is expected to match the internal SemanticModel shape for the builder,
  // but may also be a raw JSONValue from GraphQL. Use a union to accept both.
  resolved_config?: SemanticModel | JSONValue;
  created_at?: string;
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

export default ModelCatalogNode;
