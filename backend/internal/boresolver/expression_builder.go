package boresolver

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
)

// This file centralizes SQL predicate/transformation compilation for the BO
// SQL generator. It is deliberately narrow in scope: compiling a filter
// operator + value into a parameterized SQL fragment, and applying a
// field_bindings transformation (JSON_PATH/EXPRESSION) to a column
// reference. It is NOT a general expression language — for boolean
// rule/policy evaluation against already-materialized values, this codebase
// already has that in backend/pkg/policy/cel_eval.go (CEL). This compiler's
// job is narrower and different: producing a SQL fragment (plus bound
// parameters) that gets pushed into a query, never raw string interpolation
// of a caller-supplied value.

// allowedComparisonOps is the closed set of operator tokens
// CompileFilterPredicate will ever interpolate into SQL text. filter.Operator
// is caller-supplied (ultimately from an authenticated user's filter UI) and
// cannot be bind-parameterized like filter.Value, so anything not in this
// set is rejected rather than passed through.
var allowedComparisonOps = map[string]bool{
	"=": true, "!=": true, "<>": true, ">": true, "<": true, ">=": true, "<=": true,
	"CONTAINS": true, "CONTAIN": true,
	"STARTS WITH": true, "STARTS_WITH": true, "START_WITH": true,
	"ENDS WITH": true, "ENDS_WITH": true, "END_WITH": true,
	"IN": true, "NOT IN": true,
}

// CompiledPredicate is a SQL fragment paired with the parameter values it
// references, in the order its placeholders appear.
type CompiledPredicate struct {
	SQL  string
	Args []interface{}
}

// nextParam allocates the next placeholder token for ctx's dialect and
// records the value in ctx.Args, returning the token to embed in SQL.
func nextParam(g *BOSQLGenerator, ctx *GenerationContext, value interface{}) string {
	if ctx.Args == nil {
		ctx.Args = make([]interface{}, 0)
	}
	ctx.ParamCounter++
	token := paramToken(g.Dialect, ctx.ParamCounter)
	ctx.Args = append(ctx.Args, value)
	return token
}

// CompileFilterPredicate compiles one filter clause into a parameterized SQL
// fragment appended to ctx.Args, given the already-resolved SQL expression
// for the field (e.g. "t0.email"). It never interpolates filter.Value into
// the SQL string directly.
func CompileFilterPredicate(g *BOSQLGenerator, ctx *GenerationContext, sqlExpr string, filter FilterClause) (string, error) {
	op := strings.ToUpper(strings.TrimSpace(filter.Operator))
	switch op {
	case "", "EQ":
		op = "="
	case "NEQ":
		op = "!="
	case "GT":
		op = ">"
	case "LT":
		op = "<"
	case "GTE":
		op = ">="
	case "LTE":
		op = "<="
	}

	switch op {
	case "IS NULL", "NULL", "IS_NULL":
		return fmt.Sprintf("%s IS NULL", sqlExpr), nil
	case "IS NOT NULL", "NOT_NULL", "NOT NULL", "IS_NOT_NULL":
		return fmt.Sprintf("%s IS NOT NULL", sqlExpr), nil
	case "IS TRUE", "IS_TRUE":
		return fmt.Sprintf("%s IS TRUE", sqlExpr), nil
	case "IS FALSE", "IS_FALSE":
		return fmt.Sprintf("%s IS FALSE", sqlExpr), nil
	}

	// Cross-field comparison ("shipped_date >= order_date"): the right-hand
	// side is another resolved field's SQL expression, never a bound value.
	if filter.ValueFieldID != "" {
		otherExpr, err := g.ResolvePath(ctx, filter.ValueFieldID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve comparison field %s: %w", filter.ValueFieldID, err)
		}
		if op == "" {
			op = "="
		}
		return fmt.Sprintf("%s %s %s", sqlExpr, op, otherExpr), nil
	}

	if op == "BETWEEN" || op == "NOT BETWEEN" || op == "NOT_BETWEEN" {
		bounds, ok := filter.Value.([]interface{})
		if !ok || len(bounds) != 2 {
			return "", fmt.Errorf("BETWEEN requires a two-element value array [low, high]")
		}
		lowTok := nextParam(g, ctx, bounds[0])
		highTok := nextParam(g, ctx, bounds[1])
		verb := "BETWEEN"
		if op != "BETWEEN" {
			verb = "NOT BETWEEN"
		}
		return fmt.Sprintf("%s %s %s AND %s", sqlExpr, verb, lowTok, highTok), nil
	}

	// Every remaining branch below interpolates op directly into the SQL
	// text (it can't be a bind parameter), so it must come from a known-safe
	// set — never pass an unrecognized filter.Operator through as-is.
	if !allowedComparisonOps[op] {
		return "", fmt.Errorf("unsupported filter operator: %q", filter.Operator)
	}

	switch v := filter.Value.(type) {
	case string:
		switch op {
		case "CONTAINS", "CONTAIN":
			token := nextParam(g, ctx, "%"+v+"%")
			return fmt.Sprintf("%s ILIKE %s", sqlExpr, token), nil
		case "STARTS WITH", "STARTS_WITH", "START_WITH":
			token := nextParam(g, ctx, v+"%")
			return fmt.Sprintf("%s ILIKE %s", sqlExpr, token), nil
		case "ENDS WITH", "ENDS_WITH", "END_WITH":
			token := nextParam(g, ctx, "%"+v)
			return fmt.Sprintf("%s ILIKE %s", sqlExpr, token), nil
		case "IN", "NOT IN":
			return compileInList(g, ctx, sqlExpr, op, splitCSV(v))
		default:
			token := nextParam(g, ctx, v)
			return fmt.Sprintf("%s %s %s", sqlExpr, op, token), nil
		}
	case []interface{}:
		if op == "" {
			op = "IN"
		}
		return compileInList(g, ctx, sqlExpr, op, v)
	case []string:
		items := make([]interface{}, len(v))
		for i, s := range v {
			items[i] = s
		}
		return compileInList(g, ctx, sqlExpr, op, items)
	default:
		token := nextParam(g, ctx, v)
		return fmt.Sprintf("%s %s %s", sqlExpr, op, token), nil
	}
}

func splitCSV(v string) []interface{} {
	parts := strings.Split(v, ",")
	items := make([]interface{}, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func compileInList(g *BOSQLGenerator, ctx *GenerationContext, sqlExpr, op string, items []interface{}) (string, error) {
	if op != "IN" && op != "NOT IN" {
		op = "IN"
	}
	if len(items) == 0 {
		// An empty IN-list is never true; NOT IN over nothing is always true.
		// Encode without a placeholder since there's no value to bind.
		if op == "NOT IN" {
			return "1=1", nil
		}
		return "1=0", nil
	}
	tokens := make([]string, len(items))
	for i, item := range items {
		tokens[i] = nextParam(g, ctx, item)
	}
	return fmt.Sprintf("%s %s (%s)", sqlExpr, op, strings.Join(tokens, ", ")), nil
}

// CompileFilterGroup recursively compiles a FilterGroup tree into a single
// parameterized SQL fragment, appending bound values to ctx.Args exactly
// like CompileFilterPredicate. An empty group compiles to "" (the caller
// treats that as "no predicate", matching flat Filters' empty-list behavior).
func CompileFilterGroup(g *BOSQLGenerator, ctx *GenerationContext, group FilterGroup) (string, error) {
	conj := strings.ToUpper(strings.TrimSpace(group.Conjunction))
	if conj != "OR" {
		conj = "AND"
	}

	var parts []string
	for _, cond := range group.Conditions {
		sqlExpr, err := g.ResolvePath(ctx, cond.FieldID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve filter field %s: %w", cond.FieldID, err)
		}
		clause, err := CompileFilterPredicate(g, ctx, sqlExpr, cond)
		if err != nil {
			return "", fmt.Errorf("failed to compile filter for field %s: %w", cond.FieldID, err)
		}
		parts = append(parts, clause)
	}
	for _, sub := range group.Groups {
		clause, err := CompileFilterGroup(g, ctx, sub)
		if err != nil {
			return "", err
		}
		if clause != "" {
			parts = append(parts, "("+clause+")")
		}
	}

	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, " "+conj+" "), nil
}

// ---------------------------------------------------------------------------
// CEL bridge: the SAME FilterGroup definition used to compile a SQL WHERE
// clause can also be evaluated in-process against an already-materialized
// row (e.g. for row-level validation before/without a round-trip to the
// database), via CEL (github.com/google/cel-go — the expression engine this
// codebase already uses for policy/rule evaluation, see
// backend/pkg/policy/cel_eval.go). This avoids hand-rolling a second,
// divergent predicate evaluator: one FilterGroup, two backends (SQL or CEL).
//
// This bridge intentionally does NOT reuse policy.CELEvaluator — that type's
// cel.Env is hardcoded to a fixed wealth-eligibility variable schema
// (client/holdings/symbol/...). Row validation needs one DynType variable
// per BO field instead, built fresh per BO shape here.
// ---------------------------------------------------------------------------

// fieldNamer resolves a filter's FieldID/ValueFieldID (a Field UUID) to the
// column-safe variable name CEL should see. Callers typically supply
// BOField.Path or BOField.Name from the BODefinition already loaded for SQL
// generation, so a filter tree written once resolves consistently in both
// backends.
type fieldNamer func(fieldID string) (string, error)

// ToCELExpression compiles a FilterGroup into an equivalent CEL boolean
// expression string, e.g. `(status == "active") && (amount >= 100.0)`.
func ToCELExpression(group FilterGroup, namer fieldNamer) (string, error) {
	conj := " && "
	if strings.ToUpper(strings.TrimSpace(group.Conjunction)) == "OR" {
		conj = " || "
	}

	var parts []string
	for _, cond := range group.Conditions {
		expr, err := conditionToCEL(cond, namer)
		if err != nil {
			return "", err
		}
		parts = append(parts, expr)
	}
	for _, sub := range group.Groups {
		expr, err := ToCELExpression(sub, namer)
		if err != nil {
			return "", err
		}
		if expr != "" {
			parts = append(parts, "("+expr+")")
		}
	}
	if len(parts) == 0 {
		return "true", nil
	}
	return strings.Join(parts, conj), nil
}

func conditionToCEL(cond FilterClause, namer fieldNamer) (string, error) {
	name, err := namer(cond.FieldID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve field %s for CEL: %w", cond.FieldID, err)
	}

	op := strings.ToUpper(strings.TrimSpace(cond.Operator))
	switch op {
	case "", "EQ":
		return fmt.Sprintf("%s == %s", name, celLiteral(cond.Value)), nil
	case "NEQ":
		return fmt.Sprintf("%s != %s", name, celLiteral(cond.Value)), nil
	case "GT":
		return fmt.Sprintf("%s > %s", name, celLiteral(cond.Value)), nil
	case "LT":
		return fmt.Sprintf("%s < %s", name, celLiteral(cond.Value)), nil
	case "GTE":
		return fmt.Sprintf("%s >= %s", name, celLiteral(cond.Value)), nil
	case "LTE":
		return fmt.Sprintf("%s <= %s", name, celLiteral(cond.Value)), nil
	case "IS_NULL", "IS NULL", "NULL":
		return fmt.Sprintf("%s == null", name), nil
	case "IS_NOT_NULL", "IS NOT NULL", "NOT NULL", "NOT_NULL":
		return fmt.Sprintf("%s != null", name), nil
	case "CONTAINS", "CONTAIN":
		return fmt.Sprintf("%s.contains(%s)", name, celLiteral(cond.Value)), nil
	case "STARTS WITH", "STARTS_WITH", "START_WITH":
		return fmt.Sprintf("%s.startsWith(%s)", name, celLiteral(cond.Value)), nil
	case "ENDS WITH", "ENDS_WITH", "END_WITH":
		return fmt.Sprintf("%s.endsWith(%s)", name, celLiteral(cond.Value)), nil
	case "IN":
		return fmt.Sprintf("%s in %s", name, celList(cond.Value)), nil
	case "NOT IN":
		return fmt.Sprintf("!(%s in %s)", name, celList(cond.Value)), nil
	case "BETWEEN", "NOT BETWEEN", "NOT_BETWEEN":
		bounds, ok := cond.Value.([]interface{})
		if !ok || len(bounds) != 2 {
			return "", fmt.Errorf("BETWEEN requires a two-element value array [low, high]")
		}
		expr := fmt.Sprintf("(%s >= %s && %s <= %s)", name, celLiteral(bounds[0]), name, celLiteral(bounds[1]))
		if strings.HasPrefix(op, "NOT") {
			expr = "!" + expr
		}
		return expr, nil
	default:
		return "", fmt.Errorf("unsupported operator for CEL translation: %s", cond.Operator)
	}
}

func celLiteral(v interface{}) string {
	switch val := v.(type) {
	case string:
		return "\"" + strings.ReplaceAll(val, "\"", "\\\"") + "\""
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func celList(v interface{}) string {
	var items []interface{}
	switch val := v.(type) {
	case []interface{}:
		items = val
	case []string:
		for _, s := range val {
			items = append(items, s)
		}
	case string:
		for _, s := range splitCSV(val) {
			items = append(items, s)
		}
	}
	lits := make([]string, len(items))
	for i, it := range items {
		lits[i] = celLiteral(it)
	}
	return "[" + strings.Join(lits, ", ") + "]"
}

// EvaluateFilterGroupCEL evaluates group against an already-materialized row
// (e.g. for validating a record in application code without round-tripping
// through the database). vars maps the same field names namer() produces to
// the row's actual values.
func EvaluateFilterGroupCEL(group FilterGroup, namer fieldNamer, vars map[string]interface{}) (bool, error) {
	expr, err := ToCELExpression(group, namer)
	if err != nil {
		return false, err
	}

	opts := make([]cel.EnvOption, 0, len(vars))
	for name := range vars {
		opts = append(opts, cel.Variable(name, cel.DynType))
	}
	env, err := cel.NewEnv(opts...)
	if err != nil {
		return false, fmt.Errorf("failed to build CEL env: %w", err)
	}

	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("CEL compile error for %q: %w", expr, issues.Err())
	}
	prg, err := env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("CEL program build error: %w", err)
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("CEL eval error: %w", err)
	}
	result, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("CEL expression %q did not evaluate to a bool", expr)
	}
	return result, nil
}
