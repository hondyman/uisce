/**
 * Data Explorer & Report Builder Shared Semantic Types & Adapter Bridge.
 */
import { ReportParameter, ReportBuilderState } from '../../../components/reporting/builderSerialization';

export type FieldType = 'string' | 'number' | 'date' | 'boolean' | 'unknown';
export type SemanticCategory = 'dimension' | 'measure' | 'time';
export type Granularity = 'raw' | 'day' | 'week' | 'month' | 'quarter' | 'year';
export type ChartViewMode = 'table' | 'bar' | 'line' | 'area' | 'pie' | 'scatter' | 'kpi';

export type AggFn =
  | 'SUM'
  | 'AVG'
  | 'MIN'
  | 'MAX'
  | 'COUNT'
  | 'COUNT_DISTINCT'
  | 'NONE';

export type FilterOperator =
  | 'equals'
  | 'not_equals'
  | 'contains'
  | 'starts_with'
  | 'ends_with'
  | 'gt'
  | 'gte'
  | 'lt'
  | 'lte'
  | 'in'
  | 'not_in'
  | 'is_set'
  | 'is_not_set'
  | 'between';

export interface SemanticField {
  id: string;
  name: string;
  label: string;
  category: SemanticCategory;
  type: FieldType;
  table?: string;
  aggregation?: AggFn;
  format?: 'currency' | 'percent' | 'number' | 'date';
  description?: string;
}

export interface ExplorerFilter {
  id: string;
  fieldId: string;
  operator: FilterOperator | '=' | '!=' | 'IN' | 'NOT IN' | '>' | '<' | '>=' | '<=' | 'LIKE' | 'BETWEEN';
  value: any;
  isParameter?: boolean; // When true, resolves to @ParamName in Report Builder
}

export interface MeasureSelection {
  fieldId: string;
  agg: AggFn;
}

export interface TimeDimensionSelection {
  fieldId: string;
  granularity: Granularity;
}

export interface DimensionSelection {
  fieldId: string;
  granularity?: Granularity;
}

export interface SortSelection {
  fieldId: string;
  direction: 'asc' | 'desc';
}

export interface ExplorerQueryDefinition {
  id?: string;
  title: string;
  dimensions: string[];       // Field IDs
  measures: MeasureSelection[];
  timeDimensions: TimeDimensionSelection[];
  filters: ExplorerFilter[];
  parameters?: ReportParameter[];
  sorts?: SortSelection[];
  limit: number;
  suggestedChart?: ChartViewMode;
}

export type MutationIntent =
  | 'new_query'
  | 'drill_down'
  | 'drill_across'
  | 'add_filter'
  | 'add_measure'
  | 'remove_element'
  | 'unknown';

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
  querySnapshot?: ExplorerQueryDefinition;
  suggestedFollowUps?: string[];
  insights?: string[];
  mutationIntent?: MutationIntent;
}

export interface ColumnDefinition {
  key: string;
  label: string;
  type?: FieldType;
  category?: SemanticCategory;
  format?: 'currency' | 'percent' | 'number' | 'date';
}

export interface QueryExecutionResponse {
  columns: ColumnDefinition[];
  rows: Record<string, any>[];
  totalCount?: number;
  durationMs?: number;
  sql?: string;
  warnings?: string[];
}

export interface AIInsightSummary {
  summaryText: string;
  anomalies?: string[];
  topDriver?: string;
  keyMetrics?: { label: string; value: string | number }[];
}

/**
 * 1-Click Bridge: Transforms Data Explorer query definition directly into 
 * a Report Builder state with parameters and data bindings.
 */
export function convertExplorerToReportBuilder(
  query: ExplorerQueryDefinition,
  _sourceId: string,
  _bindingId?: string
): ReportBuilderState {
  return {
    reportTitle: query.title || 'Exported Explorer Query',
    parameters: query.parameters || [],
    elements: [],
    sectionConfig: {
      header: { visible: true },
      body: { visible: true },
      footer: { visible: true },
    },
    layoutSettings: {
      pageBreakBeforeGroup: false,
      pageBreakAfterGroup: false,
      pageBreakBetweenRegions: false,
      fixedPageSize: true,
      columns: 1,
      columnSpacing: 10,
      headerTokens: [query.title || 'Exported Report'],
      footerTokens: ['Page {PageNumber} of {TotalPages}'],
      includeExecutionTime: true,
      includeUserName: true,
    },
  };
}
