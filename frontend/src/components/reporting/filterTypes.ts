export type FilterOperator =
  | 'equals' | 'not_equals' | 'greater_than' | 'less_than'
  | 'greater_equal' | 'less_equal' | 'between' | 'not_between'
  | 'in' | 'not_in' | 'is_null' | 'is_not_null'
  | 'contains' | 'starts_with' | 'ends_with'
  | 'is_business_day' | 'is_holiday'
  | 'next_business_day' | 'previous_business_day' | 'add_business_days'
  | 'today' | 'yesterday' | 'tomorrow'
  | 'start_of_week' | 'end_of_week'
  | 'start_of_month' | 'end_of_month'
  | 'start_of_quarter' | 'end_of_quarter'
  | 'start_of_year' | 'end_of_year'
  | 'last_n_days' | 'last_n_business_days' | 'next_n_business_days'
  | 'previous' | 'next'
  // ── New expression operators ───────────────────────────────────────────
  | 'regexp_like'           // REGEXP_LIKE(field, pattern)
  | 'array_contains'        // ARRAY_CONTAINS(field, value)
  | 'json_extract'          // JSON_EXTRACT_SCALAR(field, '$.path')
  | 'exists'                // EXISTS (SELECT …)
  | 'not_exists'            // NOT EXISTS (SELECT …)
  | 'having_sum'            // HAVING SUM(field) op value
  | 'having_count'          // HAVING COUNT(*) op value
  | 'having_avg'            // HAVING AVG(field) op value
  | 'qualify_row_number'    // QUALIFY ROW_NUMBER() OVER(…) = 1
  | 'bitemporal_as_of'      // AS OF date injection
  | 'expression';           // raw expression mode

export type ValueSourceKind =
  | 'constant' | 'parameter' | 'function'
  | 'tenant_default' | 'instance_default' | 'calendar';

export interface ValueSource {
  kind: ValueSourceKind;
  value?: string;
  parameterId?: string;
  parameterName?: string;
  expression?: string;
  defaultKey?: 'default_calendar' | 'default_fiscal_year' | 'default_region';
  calendarCode?: string;
}

export interface Filter {
  id: string;
  field: string;
  fieldScope?: 'root' | 'subtype';
  fieldSubtypeKey?: string;
  operator: FilterOperator;
  valueSource: ValueSource;
  values?: string[];
  enabled: boolean;
}

export interface FilterGroup {
  id: string;
  combinator: 'AND' | 'OR';
  filters: Filter[];
}

export interface FilterModel {
  groups: FilterGroup[];
  groupCombinator: 'AND' | 'OR';
}

export interface TenantDefaults {
  defaultCalendarCode: string;
  defaultFiscalYear: number;
  defaultRegion: string;
}

export interface TenantCalendar {
  code: string;
  name: string;
  active: boolean;
}

export interface ReportParameter {
  id: string;
  name: string;
  type: 'string' | 'number' | 'date' | 'boolean';
  prompt: string;
  defaultValue?: string;
  allowBlank?: boolean;
  allowMultiple?: boolean;
}

// ─────────────────────────────────────────────────────────────────────────────
// Expression AST — mirrors backend filter_ast.go
// ─────────────────────────────────────────────────────────────────────────────

export type ExprNodeKind =
  | 'Literal' | 'Field' | 'Param' | 'Session'
  | 'Function' | 'BinaryOp' | 'UnaryOp'
  | 'Subquery' | 'Case' | 'Window' | 'Bitemporal' | 'Macro' | 'Aggregate';

export interface LiteralExpr {
  strVal?: string;
  numVal?: number;
  boolVal?: boolean;
  dateVal?: string;
}

export interface FieldRefExpr {
  schema?: string;
  table?: string;
  column: string;
  tableAlias?: string;
}

export interface ParamRefExpr {
  paramId: string;
  paramName: string; // @AsOfDate, @MinNAV …
  dataType: 'string' | 'number' | 'date' | 'boolean';
}

export interface CorrelationPair {
  outerField: string;
  innerField: string;
}

export interface SubquerySpec {
  operator: 'EXISTS' | 'NOT EXISTS' | 'IN' | 'NOT IN';
  targetBOType: string;
  targetTable: string;
  correlations: CorrelationPair[];
  innerFilters?: ExprNode[];
  selectFields?: string[];
}

export interface WhenClause {
  when: ExprNode;
  then: ExprNode;
}

export interface OrderExpr {
  expr: ExprNode;
  desc?: boolean;
}

export interface WindowSpec {
  partitionBy?: ExprNode[];
  orderBy?: OrderExpr[];
  frameClause?: string;
}

export type BitemporalMacroKind = 'AS_OF' | 'KNOWLEDGE_DATE' | 'PERIOD';

export interface ExprNode {
  kind: ExprNodeKind;
  // Leaf
  literal?: LiteralExpr;
  fieldRef?: FieldRefExpr;
  paramRef?: ParamRefExpr;
  sessionVar?: string; // 'TenantID' | 'UserRoles' | 'AllowedDesks'
  macroName?: string;  // 'TODAY' | 'LAST_N_DAYS' | 'T_MINUS' …
  macroArgs?: ExprNode[];
  // Composite
  funcName?: string;
  args?: ExprNode[];
  op?: string;
  left?: ExprNode;
  right?: ExprNode;
  operand?: ExprNode;
  // Subquery
  subquery?: SubquerySpec;
  // CASE
  caseWhen?: WhenClause[];
  caseElse?: ExprNode;
  // Window / Aggregate
  windowFunc?: string;
  windowSpec?: WindowSpec;
  aggFunc?: string;
  aggArg?: ExprNode;
  // Bitemporal
  bitemporalMacro?: BitemporalMacroKind;
}

// ─────────────────────────────────────────────────────────────────────────────
// Filter Category — discriminates WHERE / HAVING / QUALIFY / BITEMPORAL groups
// ─────────────────────────────────────────────────────────────────────────────

export type FilterCategory = 'WHERE' | 'HAVING' | 'QUALIFY' | 'BITEMPORAL';

export interface ExpressionFilter {
  id: string;
  category: FilterCategory;
  enabled: boolean;
  predicate?: ExprNode;
  rawExpression?: string; // freeform SQL expression entered by the user
}

export interface ExpressionFilterGroup {
  id: string;
  combinator: 'AND' | 'OR';
  category: FilterCategory;
  filters?: Filter[];
  exprFilters?: ExpressionFilter[];
}

export interface ExpressionFilterModel {
  groups: ExpressionFilterGroup[];
  groupCombinator: 'AND' | 'OR';
}

// ─────────────────────────────────────────────────────────────────────────────
// ExprNode builder helpers (frontend mirrors of Go constructor helpers)
// ─────────────────────────────────────────────────────────────────────────────

export const LiteralStr = (s: string): ExprNode => ({ kind: 'Literal', literal: { strVal: s } });
export const LiteralNum = (n: number): ExprNode => ({ kind: 'Literal', literal: { numVal: n } });
export const FieldNode = (column: string): ExprNode => ({ kind: 'Field', fieldRef: { column } });
export const FuncNode = (funcName: string, ...args: ExprNode[]): ExprNode => ({ kind: 'Function', funcName, args });
export const BinaryNode = (op: string, left: ExprNode, right: ExprNode): ExprNode => ({ kind: 'BinaryOp', op, left, right });
export const MacroNode = (macroName: string, ...macroArgs: ExprNode[]): ExprNode => ({ kind: 'Macro', macroName, macroArgs });
export const SessionNode = (sessionVar: string): ExprNode => ({ kind: 'Session', sessionVar });
export const ParamNode = (paramId: string, paramName: string, dataType: 'string' | 'number' | 'date' | 'boolean'): ExprNode =>
  ({ kind: 'Param', paramRef: { paramId, paramName, dataType } });
export const AggNode = (aggFunc: string, aggArg: ExprNode): ExprNode => ({ kind: 'Aggregate', aggFunc, aggArg });


