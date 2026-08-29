export interface ComponentDefinition {
  id: string;
  type: string;
  label: string;
  icon?: string;
  defaultProps?: Record<string, unknown>;
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
  type: 'api' | 'database' | 'static';
  config: Record<string, unknown>;
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
