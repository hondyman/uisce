package reporting

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// MacroResolver resolves dynamic references in an ExprNode tree before SQL
// compilation, ensuring pushdown-safe deterministic SQL fragments.
//
// Resolution order (highest to lowest priority):
//   1. Session variables  (@Session.TenantID) → JWT claim, NEVER from client
//   2. Report parameters  (@AsOfDate) → bind variable $N
//   3. Bitemporal macros  → system-period predicates
//   4. Date macros        (TODAY, MTD, LAST_N_DAYS) → UTC date literals
// ─────────────────────────────────────────────────────────────────────────────

// ExecutionContext carries runtime values injected at query planning time.
type ExecutionContext struct {
	// TenantID extracted from verified JWT — injected as bind var, never literal
	TenantID string
	// UserID from JWT
	UserID string
	// UserRoles from JWT claims
	UserRoles []string
	// AllowedDesks from ABAC policy evaluation
	AllowedDesks []string
	// AsOfDate from X-Uisce-As-Of-Date header (real-world event time)
	// Falls back to NOW() if empty.
	AsOfDate time.Time
	// KnowledgeDate from X-Uisce-Knowledge-Date header (system transaction time)
	KnowledgeDate time.Time
	// ReportParameterValues maps param name → runtime value
	ReportParameterValues map[string]interface{}
	// CalendarCode for financial business-day resolution
	CalendarCode string
	// Now is the wall-clock time for the query plan (UTC, set once per request)
	Now time.Time
}

// ResolvedBinding is a positional bind variable produced during macro resolution.
type ResolvedBinding struct {
	Position int         // 1-based ($1, $2 …)
	Name     string      // human label for debugging
	Value    interface{} // runtime value
}

// MacroResolutionResult holds the resolved expression and collected bind args.
type MacroResolutionResult struct {
	Node     *ExprNode         // resolved node (may replace macros with Literals or Param nodes)
	Bindings []ResolvedBinding // ordered bind variables
}

// MacroResolver walks an ExprNode tree and resolves all dynamic references.
type MacroResolver struct {
	ctx      *ExecutionContext
	bindings []ResolvedBinding
	bindIdx  int
}

// NewMacroResolver creates a resolver for the given execution context.
func NewMacroResolver(ctx *ExecutionContext) *MacroResolver {
	now := ctx.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ctx.Now = now
	return &MacroResolver{ctx: ctx}
}

// Resolve walks the full expression tree and returns a resolved copy.
func (r *MacroResolver) Resolve(node *ExprNode) *MacroResolutionResult {
	if node == nil {
		return &MacroResolutionResult{}
	}
	resolved := r.resolve(node)
	return &MacroResolutionResult{Node: resolved, Bindings: r.bindings}
}

func (r *MacroResolver) resolve(node *ExprNode) *ExprNode {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case ExprSession:
		return r.resolveSession(node)
	case ExprParam:
		return r.resolveParam(node)
	case ExprMacro:
		return r.resolveMacro(node)
	case ExprBitemporal:
		return r.resolveBitemporal(node)
	case ExprFunction:
		args := make([]*ExprNode, len(node.Args))
		for i, a := range node.Args {
			args[i] = r.resolve(a)
		}
		return &ExprNode{Kind: ExprFunction, FuncName: node.FuncName, Args: args}
	case ExprBinaryOp:
		return &ExprNode{Kind: ExprBinaryOp, Op: node.Op, Left: r.resolve(node.Left), Right: r.resolve(node.Right)}
	case ExprUnaryOp:
		return &ExprNode{Kind: ExprUnaryOp, Op: node.Op, Operand: r.resolve(node.Operand)}
	case ExprCase:
		whens := make([]*WhenClause, len(node.CaseWhen))
		for i, w := range node.CaseWhen {
			whens[i] = &WhenClause{When: r.resolve(w.When), Then: r.resolve(w.Then)}
		}
		return &ExprNode{Kind: ExprCase, CaseWhen: whens, CaseElse: r.resolve(node.CaseElse)}
	case ExprAggregate:
		return &ExprNode{Kind: ExprAggregate, AggFunc: node.AggFunc, AggArg: r.resolve(node.AggArg)}
	case ExprWindow:
		spec := resolveWindowSpec(node.WindowSpec, r)
		return &ExprNode{Kind: ExprWindow, WindowFunc: node.WindowFunc, WindowSpec: spec, AggFunc: node.AggFunc, AggArg: r.resolve(node.AggArg)}
	default:
		// ExprLiteral, ExprField — no resolution needed
		return node
	}
}

// resolveSession injects a JWT-verified session variable as a positional bind var.
// SECURITY: session values MUST come from the server-side ExecutionContext (JWT/ABAC),
// never from client-supplied ExprNode data.
func (r *MacroResolver) resolveSession(node *ExprNode) *ExprNode {
	var val interface{}
	var name string
	switch node.SessionVar {
	case "TenantID":
		val, name = r.ctx.TenantID, "@Session.TenantID"
	case "UserID":
		val, name = r.ctx.UserID, "@Session.UserID"
	case "UserRoles":
		val, name = r.ctx.UserRoles, "@Session.UserRoles"
	case "AllowedDesks":
		val, name = r.ctx.AllowedDesks, "@Session.AllowedDesks"
	default:
		// Unknown session var → emit NULL safely
		return LiteralStr("NULL")
	}
	return r.addBinding(name, val)
}

// resolveParam injects a report parameter as a positional bind variable.
func (r *MacroResolver) resolveParam(node *ExprNode) *ExprNode {
	if node.ParamRef == nil {
		return LiteralStr("NULL")
	}
	val, ok := r.ctx.ReportParameterValues[node.ParamRef.ParamName]
	if !ok {
		val, ok = r.ctx.ReportParameterValues[node.ParamRef.ParamID]
		if !ok {
			val = nil
		}
	}
	return r.addBinding(node.ParamRef.ParamName, val)
}

// resolveMacro resolves date/time macros to UTC date literals or intervals.
func (r *MacroResolver) resolveMacro(node *ExprNode) *ExprNode {
	now := r.ctx.Now
	switch strings.ToUpper(node.MacroName) {
	case "TODAY":
		return LiteralStr(now.Format("2006-01-02"))
	case "YESTERDAY":
		return LiteralStr(now.AddDate(0, 0, -1).Format("2006-01-02"))
	case "TOMORROW":
		return LiteralStr(now.AddDate(0, 0, 1).Format("2006-01-02"))
	case "MTD": // Month-to-date start
		return LiteralStr(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
	case "QTD": // Quarter-to-date start
		qStart := quarterStart(now)
		return LiteralStr(qStart.Format("2006-01-02"))
	case "YTD": // Year-to-date start
		return LiteralStr(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
	case "LAST_N_DAYS":
		n := macroArgInt(node.MacroArgs, 0, 7)
		return LiteralStr(now.AddDate(0, 0, -n).Format("2006-01-02"))
	case "NEXT_N_DAYS":
		n := macroArgInt(node.MacroArgs, 0, 7)
		return LiteralStr(now.AddDate(0, 0, n).Format("2006-01-02"))
	case "T_MINUS", "BUSINESS_DAYS_AGO":
		// Financial settlement calendar — resolved to UTC literal
		// TODO: wire to CalendarEvaluator when database is available; use naive for now
		n := macroArgInt(node.MacroArgs, 0, 2)
		return LiteralStr(naiveSubtractBusinessDays(now, n).Format("2006-01-02"))
	case "START_OF_WEEK":
		diff := int(now.Weekday())
		if diff == 0 {
			diff = 7
		}
		return LiteralStr(now.AddDate(0, 0, -(diff - 1)).Format("2006-01-02"))
	case "END_OF_WEEK":
		diff := 7 - int(now.Weekday())
		if now.Weekday() == 0 {
			diff = 0
		}
		return LiteralStr(now.AddDate(0, 0, diff).Format("2006-01-02"))
	case "START_OF_MONTH":
		return LiteralStr(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
	case "END_OF_MONTH":
		return LiteralStr(time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
	case "START_OF_QUARTER":
		return LiteralStr(quarterStart(now).Format("2006-01-02"))
	case "END_OF_QUARTER":
		return LiteralStr(quarterEnd(now).Format("2006-01-02"))
	case "START_OF_YEAR":
		return LiteralStr(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
	case "END_OF_YEAR":
		return LiteralStr(time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
	default:
		// Unknown macro — emit NULL
		return LiteralStr("NULL")
	}
}

// resolveBitemporal injects system-period predicates as bind variables.
func (r *MacroResolver) resolveBitemporal(node *ExprNode) *ExprNode {
	switch node.BitemporalMacro {
	case BitemporalAsOf:
		asOf := r.ctx.AsOfDate
		if asOf.IsZero() {
			asOf = r.ctx.Now
		}
		return r.addBinding("@AsOfDate", asOf.Format("2006-01-02T15:04:05Z"))
	case BitemporalKnowledgeDate:
		kd := r.ctx.KnowledgeDate
		if kd.IsZero() {
			kd = r.ctx.Now
		}
		return r.addBinding("@KnowledgeDate", kd.Format("2006-01-02T15:04:05Z"))
	case BitemporalPeriod:
		asOf := r.ctx.AsOfDate
		if asOf.IsZero() {
			asOf = r.ctx.Now
		}
		return r.addBinding("@PeriodAsOf", asOf.Format("2006-01-02T15:04:05Z"))
	default:
		return LiteralStr("NULL")
	}
}

// addBinding registers a new positional bind variable and returns a Param ExprNode.
func (r *MacroResolver) addBinding(name string, val interface{}) *ExprNode {
	r.bindIdx++
	b := ResolvedBinding{Position: r.bindIdx, Name: name, Value: val}
	r.bindings = append(r.bindings, b)
	return &ExprNode{
		Kind:     ExprParam,
		ParamRef: &ParamRefExpr{ParamName: fmt.Sprintf("$%d", r.bindIdx), DataType: "string"},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Date arithmetic helpers
// ─────────────────────────────────────────────────────────────────────────────

func quarterStart(t time.Time) time.Time {
	m := t.Month()
	q := ((int(m) - 1) / 3) * 3
	return time.Date(t.Year(), time.Month(q+1), 1, 0, 0, 0, 0, time.UTC)
}

func quarterEnd(t time.Time) time.Time {
	qs := quarterStart(t)
	return qs.AddDate(0, 3, -1)
}

// naiveSubtractBusinessDays subtracts n business days (Mon-Fri) from t.
// For production use, the CalendarEvaluator with actual holiday tables is wired in.
func naiveSubtractBusinessDays(t time.Time, n int) time.Time {
	result := t
	subtracted := 0
	for subtracted < n {
		result = result.AddDate(0, 0, -1)
		if result.Weekday() != time.Saturday && result.Weekday() != time.Sunday {
			subtracted++
		}
	}
	return result
}

func macroArgInt(args []*ExprNode, idx, defaultVal int) int {
	if idx >= len(args) || args[idx] == nil {
		return defaultVal
	}
	n := args[idx]
	if n.Kind == ExprLiteral && n.Literal != nil && n.Literal.NumVal != nil {
		return int(*n.Literal.NumVal)
	}
	return defaultVal
}

func resolveWindowSpec(spec *WindowSpec, r *MacroResolver) *WindowSpec {
	if spec == nil {
		return nil
	}
	pb := make([]*ExprNode, len(spec.PartitionBy))
	for i, e := range spec.PartitionBy {
		pb[i] = r.resolve(e)
	}
	ob := make([]*OrderExpr, len(spec.OrderBy))
	for i, o := range spec.OrderBy {
		ob[i] = &OrderExpr{Expr: r.resolve(o.Expr), Desc: o.Desc}
	}
	return &WindowSpec{PartitionBy: pb, OrderBy: ob, FrameClause: spec.FrameClause}
}

// ─────────────────────────────────────────────────────────────────────────────
// BitemporalPredicateBuilder emits system-period WHERE predicates for temporal
// queries against OLTP (Postgres), StarRocks, and Iceberg cold store.
// ─────────────────────────────────────────────────────────────────────────────

type BitemporalPredicateBuilder struct {
	Dialect SQLDialect
}

// BuildSystemPeriodPredicate returns the temporal WHERE clause fragment.
//
//	Postgres:     system_valid_from <= $1 AND system_valid_to > $1
//	StarRocks:    same (no native temporal syntax at time of writing)
//	Iceberg/Trino: FOR SYSTEM_TIME AS OF TIMESTAMP '$asOf' (table scan hint)
func (b *BitemporalPredicateBuilder) BuildSystemPeriodPredicate(asOfBind, validFromCol, validToCol string) string {
	switch b.Dialect {
	case DialectTrino:
		// Iceberg: injected as table scan hint, not a WHERE predicate
		return fmt.Sprintf("/* FOR SYSTEM_TIME AS OF TIMESTAMP %s */", asOfBind)
	default:
		// Postgres / StarRocks
		return fmt.Sprintf("%s <= %s AND %s > %s", validFromCol, asOfBind, validToCol, asOfBind)
	}
}

// BuildKnowledgeDatePredicate returns the knowledge-date fence predicate.
func (b *BitemporalPredicateBuilder) BuildKnowledgeDatePredicate(kdBind, createdAtCol string) string {
	return fmt.Sprintf("%s <= %s", createdAtCol, kdBind)
}

// ─────────────────────────────────────────────────────────────────────────────
// ABACInjector automatically appends RLS predicates to any filter model.
// ─────────────────────────────────────────────────────────────────────────────

// InjectABACPredicates appends mandatory Row-Level Security predicates to the
// ExpressionFilterModel based on the verified ExecutionContext.  These predicates
// are always injected server-side and cannot be overridden by client payloads.
func InjectABACPredicates(model *ExpressionFilterModel, ctx *ExecutionContext, resolver *MacroResolver) *ExpressionFilterModel {
	if model == nil {
		model = &ExpressionFilterModel{GroupCombinator: "AND"}
	}

	abacFilters := []ExpressionFilter{}

	// Rule 7: Hard tenant isolation — tenant_id = @Session.TenantID
	tenantNode := resolver.resolve(SessionNode("TenantID"))
	abacFilters = append(abacFilters, ExpressionFilter{
		ID:        "abac_tenant_isolation",
		Category:  FilterCategoryWhere,
		Enabled:   true,
		Predicate: BinaryNode("=", FieldNode("tenant_id"), tenantNode),
	})

	// Desk-level access control (if AllowedDesks is populated)
	if len(ctx.AllowedDesks) > 0 {
		deskNode := resolver.resolve(SessionNode("AllowedDesks"))
		abacFilters = append(abacFilters, ExpressionFilter{
			ID:        "abac_desk_filter",
			Category:  FilterCategoryWhere,
			Enabled:   true,
			Predicate: BinaryNode("IN", FieldNode("desk_id"), deskNode),
		})
	}

	// Prepend mandatory ABAC group
	abacGroup := ExpressionFilterGroup{
		ID:          "abac_mandatory",
		Combinator:  "AND",
		Category:    FilterCategoryWhere,
		ExprFilters: abacFilters,
	}
	model.Groups = append([]ExpressionFilterGroup{abacGroup}, model.Groups...)
	return model
}

// SanitizeFieldName ensures a field name contains only safe characters to prevent
// SQL injection via field names.  Returns a sanitized identifier or an error.
func SanitizeFieldName(name string) (string, error) {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			return r
		}
		return -1
	}, name)
	if safe == "" {
		return "", fmt.Errorf("invalid field name: %q", name)
	}
	return safe, nil
}
