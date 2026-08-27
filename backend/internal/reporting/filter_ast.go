package reporting

// ─────────────────────────────────────────────────────────────────────────────
// Expression AST — arbitrary expression trees for filter predicates.
//
// This layer sits above the simpler Filter/ValueSource model and supports:
//   - Scalar function calls  (SUBSTR, UPPER, COALESCE, DATE_ADD, ROUND …)
//   - Cross-field arithmetic (unrealized_pnl / market_value > 0.05)
//   - Parameterised bindings (@AsOfDate, @Session.TenantID → $N)
//   - Semi-joins / EXISTS subqueries
//   - CASE WHEN … THEN … END expressions
//   - Window / ROW_NUMBER predicates for QUALIFY / HAVING
//   - Bitemporal system-period macros
//
// The AST is compiled by:
//   1. MacroResolver  → resolves date macros and session vars to literals/binds
//   2. PushdownOptimizer → classifies nodes as SQL-pushable vs hybrid
//   3. DialectSQLSerializer → emits dialect-aware SQL for pushable nodes
//   4. HybridEvaluator (CEL) → evaluates non-pushable nodes in-memory
// ─────────────────────────────────────────────────────────────────────────────

// ExprNodeKind discriminates the AST node type.
type ExprNodeKind string

const (
	ExprLiteral      ExprNodeKind = "Literal"      // 'US', 42, true
	ExprField        ExprNodeKind = "Field"         // schema.table.column or fieldName
	ExprParam        ExprNodeKind = "Param"         // @AsOfDate → $N bind var
	ExprSession      ExprNodeKind = "Session"       // @Session.TenantID → JWT claim
	ExprFunction     ExprNodeKind = "Function"      // SUBSTR(f,1,3), COALESCE(a,b), DATE_ADD(…)
	ExprBinaryOp     ExprNodeKind = "BinaryOp"      // +, -, *, /, =, !=, >, <, AND, OR, IN
	ExprUnaryOp      ExprNodeKind = "UnaryOp"       // NOT, -
	ExprSubquery     ExprNodeKind = "Subquery"      // EXISTS(SELECT …) / IN(SELECT …)
	ExprCase         ExprNodeKind = "Case"          // CASE WHEN … THEN … ELSE … END
	ExprWindow       ExprNodeKind = "Window"        // ROW_NUMBER() OVER(…), SUM(…) OVER(…)
	ExprBitemporal   ExprNodeKind = "Bitemporal"    // AS_OF / KNOWLEDGE_DATE macro
	ExprMacro        ExprNodeKind = "Macro"         // TODAY, MTD, QTD, LAST_N_DAYS(n), T_MINUS(n)
	ExprAggregate    ExprNodeKind = "Aggregate"     // SUM(amount), COUNT(*) — for HAVING
)

// ExprNode is the recursive expression tree node.
type ExprNode struct {
	Kind ExprNodeKind `json:"kind"`

	// ── Leaf nodes ────────────────────────────────────────────────────────
	// ExprLiteral
	Literal *LiteralExpr `json:"literal,omitempty"`
	// ExprField
	FieldRef *FieldRefExpr `json:"fieldRef,omitempty"`
	// ExprParam
	ParamRef *ParamRefExpr `json:"paramRef,omitempty"`
	// ExprSession — value resolved from JWT/request context, never from client payload
	SessionVar string `json:"sessionVar,omitempty"` // e.g. "TenantID", "UserRoles", "AllowedDesks"
	// ExprMacro
	MacroName string     `json:"macroName,omitempty"` // "TODAY","YESTERDAY","MTD","QTD","YTD","LAST_N_DAYS","T_MINUS","BUSINESS_DAYS_AGO"
	MacroArgs []*ExprNode `json:"macroArgs,omitempty"`

	// ── Composite nodes ───────────────────────────────────────────────────
	// ExprFunction
	FuncName string     `json:"funcName,omitempty"` // SUBSTR, UPPER, TRIM, ROUND, COALESCE, DATE_ADD …
	Args     []*ExprNode `json:"args,omitempty"`

	// ExprBinaryOp / ExprUnaryOp
	Op      string   `json:"op,omitempty"` // +,-,*,/,=,!=,>,<,>=,<=,AND,OR,IN,NOT IN,LIKE,~
	Left    *ExprNode `json:"left,omitempty"`
	Right   *ExprNode `json:"right,omitempty"`
	Operand *ExprNode `json:"operand,omitempty"` // unary

	// ExprSubquery
	Subquery *SubquerySpec `json:"subquery,omitempty"`

	// ExprCase
	CaseWhen []*WhenClause `json:"caseWhen,omitempty"`
	CaseElse *ExprNode     `json:"caseElse,omitempty"`

	// ExprWindow / ExprAggregate
	WindowFunc string      `json:"windowFunc,omitempty"` // ROW_NUMBER, RANK, SUM, COUNT, AVG …
	WindowSpec *WindowSpec `json:"windowSpec,omitempty"`
	AggFunc    string      `json:"aggFunc,omitempty"` // SUM, COUNT, AVG, MAX, MIN — HAVING only
	AggArg     *ExprNode   `json:"aggArg,omitempty"`

	// ExprBitemporal
	BitemporalMacro BitemporalMacroKind `json:"bitemporalMacro,omitempty"`
}

// LiteralExpr holds a typed constant value.
type LiteralExpr struct {
	StrVal  *string  `json:"strVal,omitempty"`
	NumVal  *float64 `json:"numVal,omitempty"`
	BoolVal *bool    `json:"boolVal,omitempty"`
	DateVal *string  `json:"dateVal,omitempty"` // ISO-8601 date string
}

// FieldRefExpr references a column, optionally schema-qualified.
type FieldRefExpr struct {
	Schema string `json:"schema,omitempty"`
	Table  string `json:"table,omitempty"`
	Column string `json:"column"`
	// Alias used in generated SQL (e.g. table alias in CTE)
	TableAlias string `json:"tableAlias,omitempty"`
}

// ParamRefExpr references a report parameter, resolved to a positional bind variable.
type ParamRefExpr struct {
	ParamID   string `json:"paramId"`
	ParamName string `json:"paramName"` // e.g. "@AsOfDate", "@MinNAV"
	DataType  string `json:"dataType"`  // string | number | date | boolean
}

// SubquerySpec carries an EXISTS / NOT EXISTS / IN-subquery predicate.
type SubquerySpec struct {
	Operator string `json:"operator"` // "EXISTS" | "NOT EXISTS" | "IN" | "NOT IN"
	// Reference to a Business Object type for the subquery
	TargetBOType  string            `json:"targetBOType"`
	TargetTable   string            `json:"targetTable"`
	// Correlated join conditions: outer_field = inner_field
	Correlations  []CorrelationPair `json:"correlations"`
	// Additional WHERE conditions inside the subquery
	InnerFilters  []*ExprNode       `json:"innerFilters,omitempty"`
	// Select list — for IN(subquery); empty means SELECT 1 (EXISTS)
	SelectFields  []string          `json:"selectFields,omitempty"`
}

// CorrelationPair links an outer column to an inner column for correlated subqueries.
type CorrelationPair struct {
	OuterField string `json:"outerField"`
	InnerField string `json:"innerField"`
}

// WhenClause is one arm of a CASE expression.
type WhenClause struct {
	When *ExprNode `json:"when"`
	Then *ExprNode `json:"then"`
}

// WindowSpec holds PARTITION BY / ORDER BY / FRAME for window functions.
type WindowSpec struct {
	PartitionBy []*ExprNode `json:"partitionBy,omitempty"`
	OrderBy     []*OrderExpr `json:"orderBy,omitempty"`
	FrameClause string       `json:"frameClause,omitempty"` // e.g. "ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW"
}

// OrderExpr is a single ORDER BY expression with direction.
type OrderExpr struct {
	Expr *ExprNode `json:"expr"`
	Desc bool      `json:"desc,omitempty"`
}

// BitemporalMacroKind discriminates bitemporal injection modes.
type BitemporalMacroKind string

const (
	BitemporalAsOf          BitemporalMacroKind = "AS_OF"           // real-world event time
	BitemporalKnowledgeDate BitemporalMacroKind = "KNOWLEDGE_DATE"  // system transaction time
	BitemporalPeriod        BitemporalMacroKind = "PERIOD"          // FOR SYSTEM_TIME AS OF
)

// ─────────────────────────────────────────────────────────────────────────────
// FilterCategory discriminates WHERE / HAVING / QUALIFY / BITEMPORAL groups.
// ─────────────────────────────────────────────────────────────────────────────

type FilterCategory string

const (
	FilterCategoryWhere      FilterCategory = "WHERE"      // standard row predicate
	FilterCategoryHaving     FilterCategory = "HAVING"     // post-aggregation predicate
	FilterCategoryQualify    FilterCategory = "QUALIFY"    // window deduplication (StarRocks native / Postgres CTE)
	FilterCategoryBitemporal FilterCategory = "BITEMPORAL" // system-period injection
)

// ─────────────────────────────────────────────────────────────────────────────
// ExpressionFilter — full expression-tree based filter (supplement to Filter).
// Used when a simple field/operator/value is insufficient.
// ─────────────────────────────────────────────────────────────────────────────

type ExpressionFilter struct {
	ID       string         `json:"id"`
	Category FilterCategory `json:"category"`
	Enabled  bool           `json:"enabled"`
	// The predicate: a binary ExprNode whose .Op is =, !=, >, LIKE, IN, EXISTS …
	// e.g. BinaryOp{Left: Function{SUBSTR, [Field{isin}, Literal{1}, Literal{2}]}, Op: "=", Right: Literal{"US"}}
	Predicate *ExprNode `json:"predicate"`
	// Raw expression string — shown in UI expression mode; compiled to ExprNode server-side
	RawExpression string `json:"rawExpression,omitempty"`
}

// ExpressionFilterGroup extends FilterGroup with category and ExpressionFilter support.
type ExpressionFilterGroup struct {
	ID          string              `json:"id"`
	Combinator  string              `json:"combinator"` // AND | OR
	Category    FilterCategory      `json:"category"`
	Filters     []Filter            `json:"filters,omitempty"`
	ExprFilters []ExpressionFilter  `json:"exprFilters,omitempty"`
}

// ExpressionFilterModel is the complete, extended filter model.
type ExpressionFilterModel struct {
	Groups          []ExpressionFilterGroup `json:"groups"`
	GroupCombinator string                  `json:"groupCombinator"` // AND | OR
}

// ─────────────────────────────────────────────────────────────────────────────
// Constructor helpers
// ─────────────────────────────────────────────────────────────────────────────

func LiteralStr(s string) *ExprNode {
	return &ExprNode{Kind: ExprLiteral, Literal: &LiteralExpr{StrVal: &s}}
}

func LiteralNum(n float64) *ExprNode {
	return &ExprNode{Kind: ExprLiteral, Literal: &LiteralExpr{NumVal: &n}}
}

func LiteralBool(b bool) *ExprNode {
	return &ExprNode{Kind: ExprLiteral, Literal: &LiteralExpr{BoolVal: &b}}
}

func FieldNode(column string) *ExprNode {
	return &ExprNode{Kind: ExprField, FieldRef: &FieldRefExpr{Column: column}}
}

func FuncNode(name string, args ...*ExprNode) *ExprNode {
	return &ExprNode{Kind: ExprFunction, FuncName: name, Args: args}
}

func BinaryNode(op string, left, right *ExprNode) *ExprNode {
	return &ExprNode{Kind: ExprBinaryOp, Op: op, Left: left, Right: right}
}

func ParamNode(id, name, dtype string) *ExprNode {
	return &ExprNode{Kind: ExprParam, ParamRef: &ParamRefExpr{ParamID: id, ParamName: name, DataType: dtype}}
}

func SessionNode(varName string) *ExprNode {
	return &ExprNode{Kind: ExprSession, SessionVar: varName}
}

func MacroNode(name string, args ...*ExprNode) *ExprNode {
	return &ExprNode{Kind: ExprMacro, MacroName: name, MacroArgs: args}
}

func AggNode(fn string, arg *ExprNode) *ExprNode {
	return &ExprNode{Kind: ExprAggregate, AggFunc: fn, AggArg: arg}
}
