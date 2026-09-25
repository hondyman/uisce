package vm

import (
	"fmt"
	"strings"
)

// SQL pushdown for conditions. A filter is a RuleCondition the database
// evaluates instead of the VM: one operator vocabulary (the engine's), one
// place that decides what each operator means in SQL. Every value is bound
// through param - never interpolated - and an operator with no faithful SQL
// form is an error, never a silently dropped predicate.

// UnsupportedOperatorError is returned for an operator that has no SQL
// pushdown (unknown, or evaluable only by the VM, e.g. length_equals).
type UnsupportedOperatorError struct {
	Operator string
}

func (e *UnsupportedOperatorError) Error() string {
	return fmt.Sprintf("unsupported filter operator %q", e.Operator)
}

// operatorAliases maps the spellings callers send (SQL symbols, short
// codes, the query-builder's upper-case forms after normalisation) to the
// engine's canonical operator names.
var operatorAliases = map[string]string{
	"":                      "equals",
	"=":                     "equals",
	"==":                    "equals",
	"eq":                    "equals",
	"equal":                 "equals",
	"!=":                    "not_equals",
	"<>":                    "not_equals",
	"neq":                   "not_equals",
	"not_equal":             "not_equals",
	">":                     "greater_than",
	"gt":                    "greater_than",
	"<":                     "less_than",
	"lt":                    "less_than",
	">=":                    "greater_equal",
	"gte":                   "greater_equal",
	"greater_than_or_equal": "greater_equal",
	"<=":                    "less_equal",
	"lte":                   "less_equal",
	"less_than_or_equal":    "less_equal",
	"contain":               "contains",
	"like":                  "contains",
	"start_with":            "starts_with",
	"end_with":              "ends_with",
	"null":                  "is_null",
	"not_null":              "is_not_null",
	"before":                "less_than",
	"after":                 "greater_than",
	"on_or_before":          "less_equal",
	"on_or_after":           "greater_equal",
}

// CanonicalOperator normalises an operator spelling to the engine's name:
// case-insensitive, internal whitespace as underscores ("IS NOT NULL" ->
// "is_not_null", "NOT IN" -> "not_in"), then aliases ("GTE" -> "greater_equal").
// Unknown operators are returned normalised but otherwise unchanged.
func CanonicalOperator(op string) string {
	k := strings.ToLower(strings.Join(strings.Fields(op), "_"))
	if c, ok := operatorAliases[k]; ok {
		return c
	}
	return k
}

var comparisonSQL = map[string]string{
	"equals":        "=",
	"not_equals":    "!=",
	"greater_than":  ">",
	"less_than":     "<",
	"greater_equal": ">=",
	"less_equal":    "<=",
}

// ComparisonSQL returns the SQL symbol for a plain comparison operator (any
// spelling CanonicalOperator accepts), or an UnsupportedOperatorError. Use it
// wherever the right-hand side is not a bound value (e.g. column vs column).
func ComparisonSQL(op string) (string, error) {
	if sym, ok := comparisonSQL[CanonicalOperator(op)]; ok {
		return sym, nil
	}
	return "", &UnsupportedOperatorError{Operator: op}
}

// CompileConditionSQL compiles one condition over the SQL expression colExpr
// into a predicate. param binds a value and returns its placeholder token.
//
// Semantics (matching the VM where SQL can express them):
//   - equals/not_equals with a list value compile to IN/NOT IN; with nil,
//     to IS NULL/IS NOT NULL.
//   - contains/not_contains/starts_with/ends_with are case-insensitive
//     (ILIKE) and match the value literally (% and _ are escaped).
//   - in/not_in take an array or a comma-separated string; an empty list
//     makes IN false and NOT IN true.
//   - between/not_between take a two-element array [low, high].
func CompileConditionSQL(colExpr, operator string, value interface{}, param func(interface{}) string) (string, error) {
	op := CanonicalOperator(operator)
	switch op {
	case "is_null":
		return colExpr + " IS NULL", nil
	case "is_not_null":
		return colExpr + " IS NOT NULL", nil
	case "is_true":
		return colExpr + " IS TRUE", nil
	case "is_false":
		return colExpr + " IS FALSE", nil

	case "between", "not_between":
		bounds, ok := value.([]interface{})
		if !ok || len(bounds) != 2 {
			return "", fmt.Errorf("%s requires a two-element value array [low, high]", op)
		}
		verb := "BETWEEN"
		if op == "not_between" {
			verb = "NOT BETWEEN"
		}
		low := param(bounds[0])
		high := param(bounds[1])
		return fmt.Sprintf("%s %s %s AND %s", colExpr, verb, low, high), nil

	case "in", "not_in":
		items, ok := sqlList(value)
		if !ok {
			return "", fmt.Errorf("%s requires a list value (array or comma-separated string)", op)
		}
		return inListSQL(colExpr, op == "not_in", items, param), nil

	case "contains", "not_contains", "starts_with", "ends_with":
		if value == nil || isList(value) {
			return "", fmt.Errorf("%s requires a single value", op)
		}
		lit := escapeLike(jsString(value))
		pattern := map[string]string{
			"contains": "%" + lit + "%", "not_contains": "%" + lit + "%",
			"starts_with": lit + "%", "ends_with": "%" + lit,
		}[op]
		verb := "ILIKE"
		if op == "not_contains" {
			verb = "NOT ILIKE"
		}
		return fmt.Sprintf("%s %s %s", colExpr, verb, param(pattern)), nil
	}

	sym, ok := comparisonSQL[op]
	if !ok {
		return "", &UnsupportedOperatorError{Operator: operator}
	}
	if isList(value) {
		items, _ := sqlList(value)
		if op != "equals" && op != "not_equals" {
			return "", fmt.Errorf("%s requires a single value, not a list", op)
		}
		return inListSQL(colExpr, op == "not_equals", items, param), nil
	}
	if value == nil {
		switch op {
		case "equals":
			return colExpr + " IS NULL", nil
		case "not_equals":
			return colExpr + " IS NOT NULL", nil
		}
		return "", fmt.Errorf("%s requires a value", op)
	}
	return fmt.Sprintf("%s %s %s", colExpr, sym, param(value)), nil
}

func isList(v interface{}) bool {
	switch v.(type) {
	case []interface{}, []string:
		return true
	}
	return false
}

// sqlList returns the items of a list value. Comma-separated strings drop
// blank entries ("a, ,b" is two items).
func sqlList(v interface{}) ([]interface{}, bool) {
	switch x := v.(type) {
	case []interface{}, []string:
		return listOf(x), true
	case string:
		var out []interface{}
		for _, p := range strings.Split(x, ",") {
			if t := strings.TrimSpace(p); t != "" {
				out = append(out, t)
			}
		}
		return out, true
	}
	return nil, false
}

func inListSQL(colExpr string, negate bool, items []interface{}, param func(interface{}) string) string {
	if len(items) == 0 {
		// No placeholder: there is no value to bind.
		if negate {
			return "1=1"
		}
		return "1=0"
	}
	tokens := make([]string, len(items))
	for i, it := range items {
		tokens[i] = param(it)
	}
	verb := "IN"
	if negate {
		verb = "NOT IN"
	}
	return fmt.Sprintf("%s %s (%s)", colExpr, verb, strings.Join(tokens, ", "))
}

// escapeLike makes s match literally inside a LIKE pattern (backslash is the
// default LIKE escape character).
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
