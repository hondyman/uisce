package boresolver

import (
	"fmt"
	"strings"
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
