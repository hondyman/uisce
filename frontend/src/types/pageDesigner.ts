export type GridSpan = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

export interface PageLayoutSpec {
  id: string;
  tenant_id: string;
  key: string;
  title: string;
  domain: 'MDM' | 'ACCOUNTING' | 'ORDER_MGMT' | 'PORTFOLIO';
  target_bo_id: string;
  layout: LayoutRegion[];
  rules: PageRule[];
}

export interface LayoutRegion {
  id: string;
  name: string;
  rows: LayoutRow[];
}

export interface LayoutRow {
  id: string;
  columns: LayoutColumn[];
}

export interface LayoutColumn {
  id: string;
  span: GridSpan;
  components: PageComponent[];
}

export type ComponentType =
  | 'BO_GRID'
  | 'BO_FORM'
  | 'BO_ANALYTICS_CHART'
  | 'BO_METRIC_CARD'
  | 'BO_TIMELINE'
  | 'RELATIONSHIP_TREE'
  | 'BO_LOOKBACK_MANAGER'
  | 'AI_RECOMMENDATION_BAR';

export interface PageComponent {
  id: string;
  type: ComponentType;
  title: string;
  bo_id: string;
  bindings: {
    fields?: string[];
    measures?: string[];
    dimensions?: string[];
    relationship_id?: string;
  };
  interactions?: {
    emits_context?: {
      source_field: string;
      target_context_key: string;
    }[];
    subscribes_to_context?: {
      context_key: string;
      filter_field: string;
      operator: 'EQ' | 'IN' | 'GT' | 'LT' | 'CONTAINS';
    }[];
  };
  config: Record<string, any>;
}

export interface PageRule {
  id: string;
  name: string;
  condition: {
    field: string;
    operator: 'EQUALS' | 'NOT_EQUALS' | 'GREATER_THAN' | 'IS_NULL';
    value: any;
  };
  actions: {
    target_component_id?: string;
    target_field_id?: string;
    effect: 'HIDE' | 'DISABLE' | 'READ_ONLY' | 'HIGHLIGHT' | 'SET_DEFAULT_VALUE';
    payload?: any;
  }[];
}

export interface ParameterDefinition {
  name: string;
  label: string;
  type: 'STRING' | 'NUMBER' | 'DATE' | 'SELECT';
  defaultValue: any;
  optionsEndpoint?: string;
}

export interface SmartLinkSuggestion {
  sourceComponentId: string;
  targetComponentId: string;
  suggestedKey: string;
  confidence: 'HIGH' | 'MEDIUM' | 'MANUAL_REQUIRED';
}
