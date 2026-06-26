// frontend/src/types/index.ts

import { ReactNode as _ReactNode } from "react";

export interface DataSource {
  id: string;
  alpha_tenant_instance_id: string;
  is_active: boolean;
  config: any; // config can be complex, 'any' is safer for now
  source_name: string;
  alpha_datasource: {
    id: string; // This is the key fix
    datasource_name: string;
    datasource_type: string;
    datasource_code?: string; // Also used in Connections.tsx
  };
  core_id?: string;
}

export interface Product {
  id: string;
  version: number;
  tenant_id?: string;
  tenant_instance_id?: string;  // Legacy, may be removed
  product_id?: string;  // References products table
  alpha_product_id?: string;  // Legacy: references alpha_product table
  is_active?: boolean;
  tenant_product_datasources: DataSource[];
  alpha_product?: {
    id: string;
    product_name: string;
    product_code?: string;
    is_active?: boolean;
  };
  product?: {
    product_id: string;
    product_name: string;
  };
}

export interface TenantInstance {
  id: string;
  display_name: string;
  instance_name: string;
  description: string | null;
  is_active: boolean;
  url: string | null;
  config: any;
  tenant_id: string;
  tenant_products: Product[];
}

export interface Tenant {
  id: string;
  gold_copy: boolean;
  name: string;
  display_name: string;
  description: string | null;
  is_active: boolean;
  region?: string;
  tenant_instances: TenantInstance[];
  tenant_products?: Product[];  // Products now at tenant level
  allowed_regions?: string[];
}

export interface DataSourceConfig {
  host: string;
  port: number;
  credentials?: {
    username: string;
    password?: string;
  };
  options?: Record<string, string>;
}

export interface BusinessTermRelationship {
  businessTerm: any;
  semanticTerm: any;
  semanticView: any;
  edge?: any;
}

export interface EnhancedSearchResult {
  id: string;
  type: 'table' | 'column' | 'business_term' | 'semantic_term' | 'semantic_view';
  label: string;
  nodeId: string;
  tableName?: string;
  businessTerm?: string;
  semanticTerm?: string;
  semanticView?: string;
}

// Diff-related types used by snapshot and semantic diff viewers
export interface SnapshotDiffItem {
  field: string;
  before: string;
  after: string;
  change_type: 'added' | 'removed' | 'modified';
}

export interface SnapshotDiff {
  filters_diff?: SnapshotDiffItem[];
  metrics_diff?: SnapshotDiffItem[];
  layout_diff?: SnapshotDiffItem[];
  semantic_diff?: SnapshotDiffItem[];
}

export interface MemberDiffItem {
  name: string;
  change_type: 'added' | 'removed' | 'modified';
  before?: string;
  after?: string;
}

export interface SemanticDiff {
  from_version: number;
  to_version: number;
  dimensions?: MemberDiffItem[];
  metrics?: MemberDiffItem[];
}

// Re-export bundle types for convenience
export * from './bundles';