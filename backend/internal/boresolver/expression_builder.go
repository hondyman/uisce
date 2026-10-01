package boresolver

import (
	"fmt"
	"strings"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// This file adapts filter clauses and field_bindings transformations
// (JSON_PATH/EXPRESSION) to the BO SQL generator. Filter operator semantics
// are the rule engine's (internal/rules/vm, CompileConditionSQL) - one
// condition vocabulary whether the VM or the database evaluates it. This
// file only binds parameters for the dialect and resolves field references;
// it never interpolates a caller-supplied value into SQL.

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
// for the field (e.g. "t0.email"). Operator semantics belong to the rule
// engine (vm.CompileConditionSQL) - this only binds parameters for the
// dialect and resolves cross-field comparisons. It never interpolates
// filter.Value or filter.Operator into the SQL string.
func CompileFilterPredicate(g *BOSQLGenerator, ctx *GenerationContext, sqlExpr string, filter FilterClause) (string, error) {
	// Cross-field comparison ("shipped_date >= order_date"): the right-hand
	// side is another resolved field's SQL expression, never a bound value,
	// so only plain comparison operators are allowed. Value-less operators
	// (IS NULL, ...) ignore it.
	if filter.ValueFieldID != "" && !valueless(filter.Operator) {
		sym, err := vm.ComparisonSQL(filter.Operator)
		if err != nil {
			return "", err
		}
		otherExpr, err := g.ResolvePath(ctx, filter.ValueFieldID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve comparison field %s: %w", filter.ValueFieldID, err)
		}
		return fmt.Sprintf("%s %s %s", sqlExpr, sym, otherExpr), nil
	}
	return vm.CompileConditionSQL(sqlExpr, filter.Operator, filter.Value, func(v interface{}) string {
		return nextParam(g, ctx, v)
	})
}

func valueless(op string) bool {
	switch vm.CanonicalOperator(op) {
	case "is_null", "is_not_null", "is_true", "is_false":
		return true
	}
	return false
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
