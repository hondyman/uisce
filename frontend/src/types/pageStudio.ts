export interface ComponentDefinition {
  id: string;
  type: string;
  label?: string;
  icon?: string;
  defaultProps?: Record<string, unknown>;
  // The actual per-instance prop bag LayoutCanvas/PageComponentRenderer read
  // and write at runtime (defaultProps above is the palette's template, not
  // an instance's live values).
  props?: Record<string, unknown>;
  category?: string;
}

export interface LayoutNode {
  id: string;
  componentId: string;
  props?: Record<string, unknown>;
  children?: LayoutNode[];
  style?: Record<string, string>;
}

export interface DataSourceDefinition {
  id: string;
  name: string;
  type: 'api' | 'database' | 'static' | 'business_object';
  config: Record<string, unknown>;
}

// config shape when type === 'business_object'
export interface BusinessObjectDataSourceConfig {
  boId: string;
  boKey: string;
  bindingId: string;
  displayName: string;
  // BOs related to boId (via GET /api/business-objects/{boId}/relationships)
  // that this page also binds to, keyed by their catalog node id.
  relatedBoIds?: string[];
}

export interface CorePageDefinition {
  id: string;
  name: string;
  slug: string;
  description?: string;
  layout: LayoutNode[];
  components: ComponentDefinition[];
  dataSources: DataSourceDefinition[];
  createdAt: string;
  updatedAt: string;
  version?: number;
}
