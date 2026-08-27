/**
 * Alpha Query Builder — Query Definition (QueryDef) contract.
 *
 * This file is the single source of truth for the JSON the UI sends to the
 * backend. No SQL is ever constructed in the UI; the QueryDef is translated
 * into dialect-specific SQL by the backend SQL Generator.
 */

export type QueryRole = 'DIMENSION' | 'MEASURE' | 'CALCULATED';

export type BindingStatus = 'RESOLVED' | 'PARTIAL' | 'UNRESOLVED';

export type AggregateFunction =
  | 'SUM'
  | 'AVG'
  | 'MIN'
  | 'MAX'
  | 'COUNT'
  | 'COUNT_DISTINCT'
  | 'NONE';

export type FilterOperator =
  | 'eq'
  | 'neq'
  | 'gt'
  | 'gte'
  | 'lt'
  | 'lte'
  | 'contains'
  | 'starts_with'
  | 'ends_with'
  | 'in'
  | 'not_in'
  | 'is_null'
  | 'is_not_null'
  | 'between';

/**
 * Security + binding context. The backend must resolve the binding and
 * enforce tenant ownership before translating or executing the query.
 */
export interface QueryContext {
  /** Business Object that provides the semantic entry point. */
  boId: string;
  /** Resolved binding for the selected tenant/product/datasource triplet. */
  bindingId: string;
  /** Tenant that owns the binding and the data. */
  tenantId: string;
  /** Optional STI subtype discriminator. Drives WHERE t0.subtype_code = ... pushdown. */
  selectedSubtypeKey?: string | null;
}

export interface DimensionDef {
  /** Stable semantic identifier from the BO term graph. */
  termNodeId: string;
  /** Output alias; must be unique within the query. */
  alias: string;
}

export interface MeasureDef {
  termNodeId: string;
  alias: string;
  /** Aggregation to apply. For a bare column use 'NONE' or omit. */
  agg: AggregateFunction;
}

export interface FilterDef {
  termNodeId: string;
  operator: FilterOperator;
  /** Scalar or array value depending on operator. For between, use [min, max]. */
  value?: string | number | boolean | string[] | number[] | null;
}

export interface QueryModel {
  dimensions: DimensionDef[];
  measures: MeasureDef[];
  filters: FilterDef[];
  /** Optional list of aliases to GROUP BY. Defaults to all dimensions. */
  groupBy?: string[];
  /** Optional row limit. */
  limit?: number;
}

export interface QueryDef {
  context: QueryContext;
  query: QueryModel;
}

/**
 * Lightweight term representation exposed to the query builder picker.
 */
export interface SemanticTermView {
  termNodeId: string;
  termKey: string;
  termName: string;
  displayName: string;
  description?: string;
  dataType?: string;
  role: QueryRole;
  bindingStatus: BindingStatus;
  /** Default aggregation suggested for MEASURE terms. */
  defaultAggregation?: AggregateFunction;
}

export interface BindingView {
  bindingId: string;
  bindingName: string;
  backendId: string;
  backendName: string;
  drivingTableId?: string;
  drivingTableName?: string;
  isDefault?: boolean;
}

export interface GeneratedSQL {
  sql: string;
  dialect?: string;
  parameters?: Array<string | number | boolean | null>;
}

// Federated Explain Plan types (mirror backend internal/engine/explain/models.go)
export interface PlanNode {
  id: string;
  nodeType: string;
  dataSource: string;
  cost: number;
  details: Record<string, unknown>;
  children: PlanNode[];
  isSecured: boolean;
}

export interface PlanMetrics {
  totalLatencyMs: number;
  dataScannedBytes: number;
}

export interface FederatedPlan {
  tenantId: string;
  root: PlanNode;
  metrics: PlanMetrics;
  warnings?: string[];
}

export interface PreviewResult {
  sql: string;
  dialect?: string;
  parameters?: Array<string | number | boolean | null>;
  plan?: FederatedPlan;
}

// Meta-API schema types (mirror backend boresolver.BODefinition)
export interface BOSchemaField {
  id: string;
  name: string;
  displayName?: string;
  path?: string;
  semanticTermId?: string;
  physicalColumn?: string;
  override?: boolean;
  type?: string;
  referenceBoId?: string;
  aggregation?: string;
}

export interface BOSchemaRelationship {
  targetBoId: string;
  joinType: string;
  conditions: string[];
}

export interface BOSchema {
  id: string;
  drivingTable: string;
  datasourceId?: string;
  fields: BOSchemaField[];
  relationships: BOSchemaRelationship[];
}

export interface QueryResultColumn {
  name: string;
  type?: string;
}

export interface QueryExecuteResult {
  sql: string;
  columns: QueryResultColumn[];
  rows: Record<string, unknown>[];
  rowCount?: number;
  executionTimeMs?: number;
}

/**
 * Helper to build a blank QueryDef for a given context.
 */
export function createEmptyQueryDef(context: QueryContext): QueryDef {
  return {
    context,
    query: {
      dimensions: [],
      measures: [],
      filters: [],
      groupBy: [],
      limit: 1000,
    },
  };
}

/**
 * Return all aliases currently used in the query model.
 */
export function getUsedAliases(query: QueryModel): Set<string> {
  return new Set([
    ...query.dimensions.map((d) => d.alias),
    ...query.measures.map((m) => m.alias),
  ]);
}

/**
 * Generate a unique alias from a term name.
 */
export function makeAlias(termName: string, used: Set<string>): string {
  const base = termName
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    || 'field';

  if (!used.has(base)) return base;

  let counter = 2;
  while (used.has(`${base}_${counter}`)) {
    counter += 1;
  }
  return `${base}_${counter}`;
}
