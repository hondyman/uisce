package reporting

import (
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// SQL Dialect
// ─────────────────────────────────────────────────────────────────────────────

// SQLDialect selects the target SQL engine for dialect-specific emission.
type SQLDialect string

const (
	DialectPostgres   SQLDialect = "postgres"
	DialectStarRocks  SQLDialect = "starrocks"
	DialectTrino      SQLDialect = "trino"   // Iceberg via Trino
	DialectAnsi       SQLDialect = "ansi"    // generic fallback
)

// ─────────────────────────────────────────────────────────────────────────────
// PushdownPlan — output of the pushdown optimizer
// ─────────────────────────────────────────────────────────────────────────────

// PushdownPlan separates filter predicates into SQL-pushable and hybrid buckets.
type PushdownPlan struct {
	// Pushable: emitted directly into the SQL query
	WherePredicates      []string // → WHERE clause
	HavingPredicates     []string // → HAVING clause
	QualifyPredicates    []string // → QUALIFY (StarRocks) or CTE row_number filter (Postgres)
	BitemporalPredicates []string // → system-period temporal guards

	// Non-pushable: evaluated in-memory after data fetch via HybridEvaluator
	HybridFilters []*ExprNode

	// Ordered positional bind arguments ($1, $2 …)
	BindArgs []interface{}

	// CTEs required (e.g. QUALIFY wrappers for Postgres)
	RequiredCTEs []CTESpec
}

// CTESpec describes a WITH clause needed by the pushdown plan.
type CTESpec struct {
	Alias string
	SQL   string
}

// WhereSQL returns the WHERE clause fragment (empty string if no predicates).
func (p *PushdownPlan) WhereSQL() string {
	if len(p.WherePredicates) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(p.WherePredicates, "\n  AND ")
}

// HavingSQL returns the HAVING clause fragment.
func (p *PushdownPlan) HavingSQL() string {
	if len(p.HavingPredicates) == 0 {
		return ""
	}
	return "HAVING " + strings.Join(p.HavingPredicates, "\n  AND ")
}

// QualifySQL returns the QUALIFY clause (StarRocks) or empty (Postgres uses CTE).
func (p *PushdownPlan) QualifySQL(dialect SQLDialect) string {
	if len(p.QualifyPredicates) == 0 {
		return ""
	}
	switch dialect {
	case DialectStarRocks:
		return "QUALIFY " + strings.Join(p.QualifyPredicates, " AND ")
	default:
		return "" // wrapped in RequiredCTEs for Postgres/Trino
	}
}

// HasHybrid returns true if any filters require in-memory evaluation.
func (p *PushdownPlan) HasHybrid() bool { return len(p.HybridFilters) > 0 }

// ─────────────────────────────────────────────────────────────────────────────
// PushdownOptimizer — classifies expression nodes as pushable vs hybrid
// ─────────────────────────────────────────────────────────────────────────────

// PushdownOptimizer walks a resolved ExpressionFilterModel and produces a PushdownPlan.
type PushdownOptimizer struct {
	dialect  SQLDialect
	ser      *DialectSQLSerializer
	bindArgs []interface{}
}

// NewPushdownOptimizer creates an optimizer targeting the given SQL dialect.
func NewPushdownOptimizer(dialect SQLDialect) *PushdownOptimizer {
	return &PushdownOptimizer{
		dialect: dialect,
		ser:     NewDialectSQLSerializer(dialect),
	}
}

// Optimize builds a PushdownPlan for the given model and bind args from macro resolution.
func (o *PushdownOptimizer) Optimize(model *ExpressionFilterModel, bindArgs []interface{}) *PushdownPlan {
	o.bindArgs = bindArgs
	plan := &PushdownPlan{BindArgs: bindArgs}

	for _, grp := range model.Groups {
		var groupWhere, groupHaving, groupQualify []string

		// ── Simple Filter (legacy Filter structs) ────────────────────────
		for i := range grp.Filters {
			f := &grp.Filters[i]
			if !f.Enabled {
				continue
			}
			// Delegate to existing CompileFilterModel logic
			singleModel := &FilterModel{
				Groups:          []FilterGroup{{ID: grp.ID, Combinator: grp.Combinator, Filters: []Filter{*f}}},
				GroupCombinator: "AND",
			}
			clause := CompileFilterModel(singleModel, nil, &TenantDefaults{})
			if clause == "" {
				continue
			}
			switch grp.Category {
			case FilterCategoryHaving:
				groupHaving = append(groupHaving, clause)
			case FilterCategoryQualify:
				groupQualify = append(groupQualify, clause)
			default:
				groupWhere = append(groupWhere, clause)
			}
		}

		// ── Expression Filters (ExprNode) ────────────────────────────────
		for _, ef := range grp.ExprFilters {
			if !ef.Enabled || ef.Predicate == nil {
				continue
			}
			if o.isPushable(ef.Predicate) {
				sql := o.ser.Serialize(ef.Predicate)
				if sql == "" {
					continue
				}
				cat := ef.Category
				if cat == "" {
					cat = grp.Category
				}
				switch cat {
				case FilterCategoryHaving:
					groupHaving = append(groupHaving, sql)
				case FilterCategoryQualify:
					o.handleQualify(sql, plan)
				case FilterCategoryBitemporal:
					plan.BitemporalPredicates = append(plan.BitemporalPredicates, sql)
				default:
					groupWhere = append(groupWhere, sql)
				}
			} else {
				plan.HybridFilters = append(plan.HybridFilters, ef.Predicate)
			}
		}

		// Combine group clauses with the group combinator
		if len(groupWhere) > 0 {
			combined := combineWithOp(groupWhere, grp.Combinator)
			if len(model.Groups) > 1 {
				combined = "(" + combined + ")"
			}
			plan.WherePredicates = append(plan.WherePredicates, combined)
		}
		if len(groupHaving) > 0 {
			plan.HavingPredicates = append(plan.HavingPredicates, combineWithOp(groupHaving, grp.Combinator))
		}
	}

	return plan
}

// handleQualify adds QUALIFY predicates — emits CTE wrapper for non-StarRocks dialects.
func (o *PushdownOptimizer) handleQualify(predicate string, plan *PushdownPlan) {
	plan.QualifyPredicates = append(plan.QualifyPredicates, predicate)
	if o.dialect != DialectStarRocks {
		// Postgres/Trino: wrap in CTE
		cte := CTESpec{
			Alias: fmt.Sprintf("qualify_filter_%d", len(plan.RequiredCTEs)+1),
			SQL:   fmt.Sprintf("SELECT * FROM __source__ WHERE %s", predicate),
		}
		plan.RequiredCTEs = append(plan.RequiredCTEs, cte)
	}
}

// isPushable returns true if the expression subtree can be serialized to SQL.
// Non-pushable cases: application-level UDFs, custom scoring, unrecognised functions.
var nonPushableFuncs = map[string]bool{
	"XIRR":              true,
	"CUSTOM_RISK_SCORE": true,
	"PY_MODEL":          true,
	"ML_SCORE":          true,
}

func (o *PushdownOptimizer) isPushable(node *ExprNode) bool {
	if node == nil {
		return true
	}
	switch node.Kind {
	case ExprLiteral, ExprField:
		return true
	case ExprParam:
		return true // resolved to $N bind var
	case ExprSession:
		return true // already resolved to bind var by MacroResolver
	case ExprMacro:
		return true // already resolved to literal by MacroResolver
	case ExprFunction:
		if nonPushableFuncs[strings.ToUpper(node.FuncName)] {
			return false
		}
		for _, a := range node.Args {
			if !o.isPushable(a) {
				return false
			}
		}
		return true
	case ExprBinaryOp:
		return o.isPushable(node.Left) && o.isPushable(node.Right)
	case ExprUnaryOp:
		return o.isPushable(node.Operand)
	case ExprSubquery:
		return true // correlated subqueries are always pushed
	case ExprCase:
		for _, w := range node.CaseWhen {
			if !o.isPushable(w.When) || !o.isPushable(w.Then) {
				return false
			}
		}
		return o.isPushable(node.CaseElse)
	case ExprAggregate:
		return o.isPushable(node.AggArg)
	case ExprWindow:
		return o.isPushable(node.AggArg)
	default:
		return false
	}
}

func combineWithOp(parts []string, op string) string {
	if op == "" {
		op = "AND"
	}
	return strings.Join(parts, " "+strings.ToUpper(op)+" ")
}

// ─────────────────────────────────────────────────────────────────────────────
// DialectSQLSerializer — emits dialect-aware SQL from a resolved ExprNode tree
// ─────────────────────────────────────────────────────────────────────────────

// DialectSQLSerializer converts a resolved ExprNode tree into a SQL string
// targeting a specific database dialect.
type DialectSQLSerializer struct {
	dialect SQLDialect
}

// NewDialectSQLSerializer creates a serializer for the given dialect.
func NewDialectSQLSerializer(dialect SQLDialect) *DialectSQLSerializer {
	return &DialectSQLSerializer{dialect: dialect}
}

// Serialize converts an ExprNode to a SQL fragment.
func (s *DialectSQLSerializer) Serialize(node *ExprNode) string {
	if node == nil {
		return "NULL"
	}
	switch node.Kind {
	case ExprLiteral:
		return s.serializeLiteral(node.Literal)
	case ExprField:
		return s.serializeField(node.FieldRef)
	case ExprParam:
		if node.ParamRef != nil {
			return node.ParamRef.ParamName // $1, $2 …
		}
		return "NULL"
	case ExprFunction:
		return s.serializeFunction(node)
	case ExprBinaryOp:
		return s.serializeBinaryOp(node)
	case ExprUnaryOp:
		return s.serializeUnaryOp(node)
	case ExprSubquery:
		return s.serializeSubquery(node.Subquery)
	case ExprCase:
		return s.serializeCase(node)
	case ExprAggregate:
		return s.serializeAggregate(node)
	case ExprWindow:
		return s.serializeWindow(node)
	default:
		return "NULL"
	}
}

func (s *DialectSQLSerializer) serializeLiteral(lit *LiteralExpr) string {
	if lit == nil {
		return "NULL"
	}
	if lit.StrVal != nil {
		return "'" + strings.ReplaceAll(*lit.StrVal, "'", "''") + "'"
	}
	if lit.NumVal != nil {
		v := *lit.NumVal
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%f", v)
	}
	if lit.BoolVal != nil {
		if *lit.BoolVal {
			return "TRUE"
		}
		return "FALSE"
	}
	if lit.DateVal != nil {
		return fmt.Sprintf("'%s'::date", *lit.DateVal)
	}
	return "NULL"
}

func (s *DialectSQLSerializer) serializeField(ref *FieldRefExpr) string {
	if ref == nil {
		return `"unknown"`
	}
	alias := ref.TableAlias
	if alias == "" {
		alias = ref.Table
	}
	if alias != "" {
		return fmt.Sprintf(`"%s"."%s"`, alias, ref.Column)
	}
	return quoteField(ref.Column)
}

// serializeFunction handles scalar functions with dialect-specific translation.
func (s *DialectSQLSerializer) serializeFunction(node *ExprNode) string {
	fn := strings.ToUpper(node.FuncName)
	args := make([]string, len(node.Args))
	for i, a := range node.Args {
		args[i] = s.Serialize(a)
	}
	join := func() string { return strings.Join(args, ", ") }

	switch fn {
	// ── String functions ────────────────────────────────────────────────
	case "SUBSTR", "SUBSTRING":
		return fmt.Sprintf("SUBSTR(%s)", join())
	case "UPPER":
		return fmt.Sprintf("UPPER(%s)", join())
	case "LOWER":
		return fmt.Sprintf("LOWER(%s)", join())
	case "TRIM":
		return fmt.Sprintf("TRIM(%s)", join())
	case "LTRIM":
		return fmt.Sprintf("LTRIM(%s)", join())
	case "RTRIM":
		return fmt.Sprintf("RTRIM(%s)", join())
	case "REPLACE":
		return fmt.Sprintf("REPLACE(%s)", join())
	case "LENGTH", "LEN":
		return fmt.Sprintf("LENGTH(%s)", join())
	case "CONCAT":
		switch s.dialect {
		case DialectPostgres:
			return fmt.Sprintf("CONCAT(%s)", join())
		default:
			return fmt.Sprintf("CONCAT(%s)", join())
		}

	// ── Regex ────────────────────────────────────────────────────────────
	case "REGEXP_LIKE":
		switch s.dialect {
		case DialectPostgres:
			if len(args) >= 2 {
				return fmt.Sprintf("(%s ~ %s)", args[0], args[1])
			}
		case DialectStarRocks:
			return fmt.Sprintf("REGEXP(%s)", join())
		default: // Trino, ANSI
			return fmt.Sprintf("REGEXP_LIKE(%s)", join())
		}

	// ── Date / time ───────────────────────────────────────────────────────
	case "DATE_TRUNC":
		return fmt.Sprintf("DATE_TRUNC(%s)", join())
	case "DATE_ADD":
		switch s.dialect {
		case DialectStarRocks:
			// StarRocks: DATE_ADD(date, INTERVAL N unit)
			if len(args) >= 3 {
				return fmt.Sprintf("DATE_ADD(%s, INTERVAL %s %s)", args[0], args[1], strings.Trim(args[2], "'"))
			}
		case DialectTrino:
			if len(args) >= 3 {
				return fmt.Sprintf("DATE_ADD(%s, %s, %s)", strings.Trim(args[2], "'"), args[1], args[0])
			}
		default: // Postgres
			if len(args) >= 3 {
				return fmt.Sprintf("(%s + INTERVAL '%s' %s)", args[0], strings.Trim(args[1], "'"), strings.Trim(args[2], "'"))
			}
		}
	case "EXTRACT":
		return fmt.Sprintf("EXTRACT(%s)", join())
	case "NOW", "CURRENT_TIMESTAMP":
		return "NOW()"
	case "CURRENT_DATE":
		return "CURRENT_DATE"

	// ── Null / conditional ────────────────────────────────────────────────
	case "COALESCE":
		return fmt.Sprintf("COALESCE(%s)", join())
	case "NULLIF":
		return fmt.Sprintf("NULLIF(%s)", join())
	case "IIF", "IF":
		if len(args) == 3 {
			return fmt.Sprintf("CASE WHEN %s THEN %s ELSE %s END", args[0], args[1], args[2])
		}

	// ── Numeric ───────────────────────────────────────────────────────────
	case "ROUND":
		return fmt.Sprintf("ROUND(%s)", join())
	case "FLOOR":
		return fmt.Sprintf("FLOOR(%s)", join())
	case "CEIL", "CEILING":
		return fmt.Sprintf("CEIL(%s)", join())
	case "ABS":
		return fmt.Sprintf("ABS(%s)", join())

	// ── Array / JSON ──────────────────────────────────────────────────────
	case "ARRAY_CONTAINS":
		switch s.dialect {
		case DialectPostgres:
			if len(args) >= 2 {
				return fmt.Sprintf("%s @> ARRAY[%s]", args[0], args[1])
			}
		case DialectStarRocks:
			return fmt.Sprintf("ARRAY_CONTAINS(%s)", join())
		default: // Trino
			return fmt.Sprintf("CONTAINS(%s)", join())
		}
	case "JSON_EXTRACT_SCALAR":
		switch s.dialect {
		case DialectPostgres:
			if len(args) >= 2 {
				// col->>'key' from JSON path $.key
				path := strings.Trim(args[1], "'")
				path = strings.TrimPrefix(path, "$.")
				return fmt.Sprintf("%s->>'%s'", args[0], path)
			}
		default:
			return fmt.Sprintf("JSON_EXTRACT_SCALAR(%s)", join())
		}
	}

	// Default — emit as-is (let the engine validate)
	return fmt.Sprintf("%s(%s)", fn, join())
}

func (s *DialectSQLSerializer) serializeBinaryOp(node *ExprNode) string {
	left := s.Serialize(node.Left)
	right := s.Serialize(node.Right)
	op := strings.ToUpper(node.Op)
	switch op {
	case "=", "!=", "<>", ">", "<", ">=", "<=":
		return fmt.Sprintf("%s %s %s", left, op, right)
	case "AND", "OR":
		return fmt.Sprintf("(%s %s %s)", left, op, right)
	case "IN":
		return fmt.Sprintf("%s IN (%s)", left, right)
	case "NOT IN":
		return fmt.Sprintf("%s NOT IN (%s)", left, right)
	case "LIKE", "ILIKE", "NOT LIKE":
		return fmt.Sprintf("%s %s %s", left, op, right)
	case "+", "-", "*", "/":
		return fmt.Sprintf("(%s %s %s)", left, op, right)
	case "EXISTS":
		return fmt.Sprintf("EXISTS %s", right)
	case "NOT EXISTS":
		return fmt.Sprintf("NOT EXISTS %s", right)
	default:
		return fmt.Sprintf("%s %s %s", left, op, right)
	}
}

func (s *DialectSQLSerializer) serializeUnaryOp(node *ExprNode) string {
	op := strings.ToUpper(node.Op)
	inner := s.Serialize(node.Operand)
	switch op {
	case "NOT":
		return fmt.Sprintf("NOT (%s)", inner)
	case "-":
		return fmt.Sprintf("-(%s)", inner)
	case "IS NULL":
		return fmt.Sprintf("%s IS NULL", inner)
	case "IS NOT NULL":
		return fmt.Sprintf("%s IS NOT NULL", inner)
	default:
		return fmt.Sprintf("%s %s", op, inner)
	}
}

func (s *DialectSQLSerializer) serializeSubquery(sq *SubquerySpec) string {
	if sq == nil {
		return "NULL"
	}
	var sb strings.Builder
	sb.WriteString(sq.Operator)
	sb.WriteString(" (SELECT ")
	if len(sq.SelectFields) > 0 {
		sb.WriteString(strings.Join(sq.SelectFields, ", "))
	} else {
		sb.WriteString("1")
	}
	sb.WriteString(" FROM ")
	sb.WriteString(quoteField(sq.TargetTable))
	var wheres []string
	for _, c := range sq.Correlations {
		wheres = append(wheres, fmt.Sprintf("%s = %s", quoteField(c.OuterField), quoteField(c.InnerField)))
	}
	for _, f := range sq.InnerFilters {
		wheres = append(wheres, s.Serialize(f))
	}
	if len(wheres) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(wheres, " AND "))
	}
	sb.WriteString(")")
	return sb.String()
}

func (s *DialectSQLSerializer) serializeCase(node *ExprNode) string {
	var sb strings.Builder
	sb.WriteString("CASE")
	for _, w := range node.CaseWhen {
		sb.WriteString(fmt.Sprintf(" WHEN %s THEN %s", s.Serialize(w.When), s.Serialize(w.Then)))
	}
	if node.CaseElse != nil {
		sb.WriteString(" ELSE ")
		sb.WriteString(s.Serialize(node.CaseElse))
	}
	sb.WriteString(" END")
	return sb.String()
}

func (s *DialectSQLSerializer) serializeAggregate(node *ExprNode) string {
	fn := strings.ToUpper(node.AggFunc)
	if node.AggArg == nil {
		if fn == "COUNT" {
			return "COUNT(*)"
		}
		return fn + "()"
	}
	return fmt.Sprintf("%s(%s)", fn, s.Serialize(node.AggArg))
}

func (s *DialectSQLSerializer) serializeWindow(node *ExprNode) string {
	fn := strings.ToUpper(node.WindowFunc)
	var inner string
	if node.AggArg != nil {
		inner = s.Serialize(node.AggArg)
	}
	spec := s.serializeWindowSpec(node.WindowSpec)
	return fmt.Sprintf("%s(%s) OVER (%s)", fn, inner, spec)
}

func (s *DialectSQLSerializer) serializeWindowSpec(spec *WindowSpec) string {
	if spec == nil {
		return ""
	}
	var parts []string
	if len(spec.PartitionBy) > 0 {
		pbs := make([]string, len(spec.PartitionBy))
		for i, e := range spec.PartitionBy {
			pbs[i] = s.Serialize(e)
		}
		parts = append(parts, "PARTITION BY "+strings.Join(pbs, ", "))
	}
	if len(spec.OrderBy) > 0 {
		obs := make([]string, len(spec.OrderBy))
		for i, o := range spec.OrderBy {
			dir := ""
			if o.Desc {
				dir = " DESC"
			}
			obs[i] = s.Serialize(o.Expr) + dir
		}
		parts = append(parts, "ORDER BY "+strings.Join(obs, ", "))
	}
	if spec.FrameClause != "" {
		parts = append(parts, spec.FrameClause)
	}
	return strings.Join(parts, " ")
}
