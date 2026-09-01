/**
 * Data Explorer — Unified Query State contract.
 *
 * This file is the single source of truth for the JSON the UI sends to the
 * backend. No SQL is ever constructed in the UI; QueryState is translated
 * into dialect-specific SQL by the backend SQL Generator.
 */

export type FieldCategory = 'dimension' | 'measure' | 'time';
export type FieldType = 'string' | 'number' | 'date' | 'boolean' | 'unknown';

export type Granularity = 'raw' | 'day' | 'week' | 'month' | 'quarter' | 'year';

export type AggFn =
  | 'SUM'
  | 'AVG'
  | 'MIN'
  | 'MAX'
  | 'COUNT'
  | 'COUNT_DISTINCT'
  | 'ROW_NUMBER'
  | 'RANK'
  | 'DENSE_RANK'
  | 'LEAD'
  | 'LAG'
  | 'RUNNING_TOTAL'
  | 'PERCENT_OF_TOTAL'
  | 'HAVING'
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

export type SortDirection = 'asc' | 'desc';

export type ViewMode = 'table' | 'bar' | 'line' | 'area' | 'pie' | 'scatter' | 'kpi';

export type SourceKind = 'business_object';

export interface ExplorerSubtype {
  id: string;
  key: string;
  name: string;
  displayName: string;
  description?: string;
  fields: ExplorerField[];
}

export interface ExplorerRelatedBO {
  boName: string;
  edge: string;
  description?: string;
}

/**
 * Lightweight field representation exposed to the explorer picker.
 * Derived from backend BO term list and cached per-binding.
 */
export interface ExplorerField {
  id: string;
  name: string;
  displayName: string;
  technicalName?: string;
  category: FieldCategory;
  type: FieldType;
  description?: string;
  defaultAggregation?: AggFn;
  isCore?: boolean;
  isCustom?: boolean;
  provenanceScope?: 'core' | 'custom';
  _scope?: 'root' | 'subtype';
  _subtypeKey?: string;
  _subtypeName?: string;
}

/**
 * A resolved source the explorer runs queries against. Phase 1 only
 * supports Business Objects; expand via SourceKind union later.
 */
export interface ExplorerSource {
  kind: SourceKind;
  id: string;
  bindingId: string;
  datasourceId: string;
  displayName: string;
  description?: string;
  isCore?: boolean;
  isCustom?: boolean;
  fields: ExplorerField[];
  fieldAllowlist?: string[];
  subtypes?: Record<string, ExplorerSubtype>;
  relatedBOs?: ExplorerRelatedBO[];
  rawBO?: any;
  createdAt?: string;
  updatedAt?: string;
  selectedSubtypeKey?: string | null;
}


export interface DimensionSelection {
  fieldId: string;
  granularity?: Granularity;
  expression?: string;
  alias?: string;
}

export interface MeasureSelection {
  fieldId: string;
  agg: AggFn;
  expression?: string;
  alias?: string;
}

export interface FilterSelection {
  fieldId: string;
  operator: FilterOperator;
  values: string[];
}

export interface SortSelection {
  fieldId: string;
  direction: SortDirection;
}

export interface CalculationDefinition {
  id: string;
  name: string;
  displayName: string;
  formula: string; // e.g. "SUM([revenue]) / SUM([volume])" or "[price] * [quantity]"
  returnType: FieldType;
  description?: string;
  format?: 'currency' | 'percent' | 'number';
}

export interface QueryParameter {
  id: string;
  name: string;
  displayName: string;
  type: 'string' | 'number' | 'date' | 'daterange' | 'select' | 'multiselect';
  defaultValue?: any;
  currentValue?: any;
  options?: { label: string; value: any }[];
  description?: string;
}

export interface ScheduleConfig {
  scheduleName: string;
  cronExpression: string;
  timezone: string;
  outputFormats: ('csv' | 'json' | 'pdf' | 'excel')[];
  deliveryChannels: {
    type: 'email' | 'webhook' | 'storage';
    target: string;
  }[];
  isActive: boolean;
}

export interface ShareConfig {
  sharedWith: {
    recipientId: string;
    recipientName: string;
    permission: 'view' | 'edit' | 'admin';
    watermark: boolean;
  }[];
}

export interface ExplorerQueryState {
  sourceId: string;
  bindingId: string;
  dimensions: DimensionSelection[];
  measures: MeasureSelection[];
  timeDimensions: DimensionSelection[];
  calculations: CalculationDefinition[];
  parameters: QueryParameter[];
  filters: FilterSelection[];
  sorts: SortSelection[];
  limit: number;
}

export interface QueryTabState {
  id: string;
  name: string;
  source: ExplorerSource | null;
  queryState: ExplorerQueryState;
  result: ExplorerResult | null;
  viewMode: ViewMode;
  isExecuting: boolean;
  error: string | null;
  lastRunAt: string | null;
  isSaved?: boolean;
}

export interface ExplorerResultColumn {
  name: string;
  type?: FieldType;
  format?: string;
}

export interface ExplorerResult {
  columns: ExplorerResultColumn[];
  rows: Record<string, unknown>[];
  sql: string;
  plan?: import('../../query-builder/types/queryDef').FederatedPlan;
  rowCount: number;
  executionTimeMs: number;
  warnings?: string[];
}

export interface SavedExplorerQuery {
  id: string;
  name: string;
  sourceKind: SourceKind;
  sourceId: string;
  queryState: ExplorerQueryState;
  createdBy?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface BusinessObjectSummary {
  id: string;
  name: string;
  displayName: string;
  description?: string;
  defaultBindingId?: string;
  fieldCount?: number;
  updatedAt?: string;
  // Populated on-demand by UnifiedBOPickerModal from the BO's `with_bindings`
  // detail response — not part of the summary list payload.
  subtypes?: Record<string, any>;
  relatedBOs?: any[];
  fields?: any[];
  coreFields?: any[];
  core_fields?: any[];
}

export const EXPLORER_ACCENT = '#f9f506';
export const EXPLORER_ACCENT_DARK = '#e6e205';
export const EXPLORER_BG = '#f8f8f5';
export const EXPLORER_BORDER = '#e6e6db';
export const EXPLORER_TEXT = '#181811';
export const EXPLORER_MUTED = '#8c8b5f';

export interface ExplorerThemeColors {
  accent: string;
  accentDark: string;
  background: string;
  backgroundElevated: string;
  border: string;
  text: string;
  textMuted: string;
}

export const EXPLORER_THEME_LIGHT: ExplorerThemeColors = {
  accent: '#00C9C8',
  accentDark: '#009E9D',
  background: '#F0FAFC',
  backgroundElevated: '#FFFFFF',
  border: 'rgba(0, 149, 155, 0.15)',
  text: '#071526',
  textMuted: 'rgba(7, 21, 38, 0.40)',
};

export const EXPLORER_THEME_DARK: ExplorerThemeColors = {
  accent: '#00C9C8',
  accentDark: '#00706F',
  background: '#050D1A',
  backgroundElevated: '#071526',
  border: 'rgba(0, 201, 200, 0.12)',
  text: '#E8F4FF',
  textMuted: 'rgba(232, 244, 255, 0.50)',
};

export const DEFAULT_QUERY_STATE: Omit<ExplorerQueryState, 'sourceId' | 'bindingId'> = {
  dimensions: [],
  measures: [],
  timeDimensions: [],
  calculations: [],
  parameters: [],
  filters: [],
  sorts: [],
  limit: 1000,
};

export function emptyExplorerState(
  sourceId: string,
  bindingId: string
): ExplorerQueryState {
  return {
    sourceId,
    bindingId,
    ...DEFAULT_QUERY_STATE,
  };
}

export function createNewQueryTab(id?: string, name?: string, source?: ExplorerSource | null): QueryTabState {
  const tabId = id || `tab-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`;
  return {
    id: tabId,
    name: name || 'Query 1',
    source: source || null,
    queryState: emptyExplorerState(source?.id || '', source?.bindingId || ''),
    result: null,
    viewMode: 'table',
    isExecuting: false,
    error: null,
    lastRunAt: null,
  };
}
