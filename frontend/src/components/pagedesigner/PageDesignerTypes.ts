import { FormTemplateSpec } from '../reporting/form/FormManagerTypes';
import { CellStyle } from '../reporting/tableColumnModel';

export type WidgetType = 
  | 'BO_FORM'               // CRUD Form container using FormTemplateSpec
  | 'BO_GRID'               // Interactive virtual table grid
  | 'QUERY_VISUALIZATION'   // Visual chart (ECharts) bound to Query Explorer query
  | 'METRIC_CARD'           // Single KPI stat card with formula evaluation
  | 'FILTER_BAR';           // Parameter filter shelf for global page filtering

export interface EventAction {
  targetChannel: string;    // e.g. "selected_account_id" or "filter_region"
  sourcePropertyKey: string;// Field key from event payload (e.g. "account_bk" or "id")
  actionType: 'SET_PARAMETER' | 'CLEAR_PARAMETER' | 'NAVIGATE' | 'TRIGGER_REFETCH' | 'LAUNCH_MODAL_FORM';
  targetPageKey?: string;   // For cross-page drill-downs
}

export interface WidgetEventConfig {
  onRowSelect?: EventAction[];
  onRowDoubleClick?: EventAction[];
  onRowClick?: EventAction[];
  onChartSelect?: EventAction[];
  onFormSubmit?: EventAction[];
}

export interface PageWidgetDef {
  id: string;
  type: WidgetType;
  title: string;
  boKey?: string;                     // Primary BO binding for forms/grids
  queryId?: string;                   // Query Explorer asset ID for charts
  customQueryDef?: Record<string, any>; // Inline query AST
  formSpec?: FormTemplateSpec;        // Embedded form specification
  containerStyle?: CellStyle;
  gridSpan: { xs: number; sm?: number; md: number; lg: number }; // 12-column span
  subscribedParams: string[];         // Re-executes/updates when these parameters mutate
  events?: WidgetEventConfig;
  entitlements?: {
    requiredRoles?: string[];
    allowCreate?: boolean;
    allowUpdate?: boolean;
    allowDelete?: boolean;
  };
}

export interface PageLayoutSpec {
  pageKey: string;
  title: string;
  description?: string;
  isGoldCopy?: boolean;               // 80/10/10 Gold copy baseline flag
  declaredParameters: Array<{
    key: string;                      // e.g. "selected_account_id"
    displayName: string;
    dataType: 'string' | 'number' | 'date' | 'boolean';
    defaultValue?: any;
  }>;
  sections: Array<{
    id: string;
    flow: 'ROW' | 'COLUMN';           // Side-by-side vs stacked layout
    header?: { title: string; show: boolean };
    widgets: PageWidgetDef[];
  }>;
}
