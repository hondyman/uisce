package rulefabric

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// =============================================================================
// CONDITION TREE -> CEL COMPILER
// =============================================================================
//
// RuleFabric is standardizing on CEL as its single condition language: no
// more AdvancedConditionBuilder JSON trees, no CUE, no Starlark. This file
// compiles an existing ConditionGroup/Condition tree (the shape
// AdvancedConditionBuilder.tsx produces, and what every rule authored before
// this migration has stored in rule_logic.condition_json) into an equivalent
// CEL boolean expression string, so it can be evaluated the same way as a
// hand-authored policy expression - one evaluator, one language.
//
// Field access compiles to `data.<field>` (or `data["field.with.dots"]` for
// a leaf field name that itself isn't a valid CEL identifier chain);
// EntityPath cross-entity conditions have no direct `data.X` equivalent
// without a join, so they are reported in the returned `unsupported` list
// rather than silently compiled to something wrong - callers should treat a
// non-empty unsupported list as "this rule needs manual attention", not
// discard it.

// CompileConditionTreeToCEL compiles a ConditionGroup (or a raw condition_json
// blob containing one) into a CEL boolean expression. Returns the compiled
// expression, a list of human-readable notes about anything it could not
// translate (empty if everything compiled cleanly), and an error only for
// structurally invalid input (not for individual unsupported conditions).
func CompileConditionTreeToCEL(raw json.RawMessage) (string, []string, error) {
	var group ConditionGroup
	if err := json.Unmarshal(raw, &group); err != nil {
		return "", nil, fmt.Errorf("failed to parse condition tree: %w", err)
	}
	var unsupported []string
	expr, err := compileGroup(&group, &unsupported)
	if err != nil {
		return "", unsupported, err
	}
	if expr == "" {
		expr = "true"
	}
	return expr, unsupported, nil
}

func compileGroup(group *ConditionGroup, unsupported *[]string) (string, error) {
	if group == nil || len(group.Conditions) == 0 {
		return "true", nil
	}

	parts := make([]string, 0, len(group.Conditions))
	for _, raw := range group.Conditions {
		expr, err := compileNode(raw, unsupported)
		if err != nil {
			return "", err
		}
		if expr != "" {
			parts = append(parts, expr)
		}
	}
	if len(parts) == 0 {
		return "true", nil
	}

	op := strings.ToUpper(group.Operator)
	switch op {
	case "NOT":
		// NOT is unary: negate the (assumed single) child.
		return fmt.Sprintf("!(%s)", parts[0]), nil
	case "OR":
		return "(" + strings.Join(parts, " || ") + ")", nil
	default: // AND, or unspecified
		return "(" + strings.Join(parts, " && ") + ")", nil
	}
}

// compileNode compiles one element of a ConditionGroup.Conditions slice,
// which per the JSON shape may itself be a nested group or a leaf condition.
func compileNode(raw interface{}, unsupported *[]string) (string, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}

	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return "", err
	}

	switch probe.Type {
	case "group":
		var nested ConditionGroup
		if err := json.Unmarshal(b, &nested); err != nil {
			return "", err
		}
		return compileGroup(&nested, unsupported)
	default:
		var cond Condition
		if err := json.Unmarshal(b, &cond); err != nil {
			return "", err
		}
		return compileCondition(&cond, unsupported)
	}
}

func compileCondition(cond *Condition, unsupported *[]string) (string, error) {
	if cond.EntityPath != nil {
		*unsupported = append(*unsupported, fmt.Sprintf(
			"condition on field %q uses a cross-entity EntityPath (from %s to %s via %s) with no direct CEL equivalent - needs manual review",
			cond.Field, cond.EntityPath.FromEntity, cond.EntityPath.ToEntity, cond.EntityPath.Relationship,
		))
		return "true", nil // neutral element for AND; doesn't silently fail the whole rule
	}

	field := celFieldAccess(cond.Field)

	switch cond.Operator {
	case "equals":
		return fmt.Sprintf("(%s == %s)", field, celLiteral(cond.Value)), nil
	case "not_equals":
		return fmt.Sprintf("(%s != %s)", field, celLiteral(cond.Value)), nil
	case "is_null":
		return fmt.Sprintf("(!has(%s) || string(%s) == \"\")", field, field), nil
	case "is_not_null":
		return fmt.Sprintf("(has(%s) && string(%s) != \"\")", field, field), nil
	case "greater_than":
		return fmt.Sprintf("(double(%s) > %s)", field, celNumberLiteral(cond.Value)), nil
	case "greater_than_or_equals":
		return fmt.Sprintf("(double(%s) >= %s)", field, celNumberLiteral(cond.Value)), nil
	case "less_than":
		return fmt.Sprintf("(double(%s) < %s)", field, celNumberLiteral(cond.Value)), nil
	case "less_than_or_equals":
		return fmt.Sprintf("(double(%s) <= %s)", field, celNumberLiteral(cond.Value)), nil
	case "between":
		lo, hi, err := betweenBounds(cond.Value)
		if err != nil {
			*unsupported = append(*unsupported, fmt.Sprintf("condition on field %q: %v", cond.Field, err))
			return "true", nil
		}
		return fmt.Sprintf("(double(%s) >= %s && double(%s) <= %s)", field, lo, field, hi), nil
	case "contains":
		return fmt.Sprintf("string(%s).contains(%s)", field, celLiteral(cond.Value)), nil
	case "not_contains":
		return fmt.Sprintf("!string(%s).contains(%s)", field, celLiteral(cond.Value)), nil
	case "starts_with":
		return fmt.Sprintf("string(%s).startsWith(%s)", field, celLiteral(cond.Value)), nil
	case "ends_with":
		return fmt.Sprintf("string(%s).endsWith(%s)", field, celLiteral(cond.Value)), nil
	case "matches_regex":
		return fmt.Sprintf("string(%s).matches(%s)", field, celLiteral(cond.Value)), nil
	case "in":
		return fmt.Sprintf("(%s in %s)", field, celListLiteral(cond.Value)), nil
	case "not_in":
		return fmt.Sprintf("!(%s in %s)", field, celListLiteral(cond.Value)), nil
	case "date_before":
		return fmt.Sprintf("(timestamp(string(%s)) < timestamp(%s))", field, celLiteral(cond.Value)), nil
	case "date_after":
		return fmt.Sprintf("(timestamp(string(%s)) > timestamp(%s))", field, celLiteral(cond.Value)), nil
	case "days_ago_less_than":
		hours, err := daysToHoursLiteral(cond.Value)
		if err != nil {
			*unsupported = append(*unsupported, fmt.Sprintf("condition on field %q: %v", cond.Field, err))
			return "true", nil
		}
		return fmt.Sprintf("(now - timestamp(string(%s)) < duration(%q))", field, hours), nil
	default:
		*unsupported = append(*unsupported, fmt.Sprintf("condition on field %q uses unknown operator %q - left as always-true, needs manual review", cond.Field, cond.Operator))
		return "true", nil
	}
}

// celFieldAccess turns a (possibly dotted) field path into a CEL expression
// rooted at the `data` variable, e.g. "status" -> "data.status",
// "customer.kyc_status" -> "data.customer.kyc_status".
func celFieldAccess(field string) string {
	if field == "" {
		return "data"
	}
	// Every path segment must be a valid CEL identifier for dot access to
	// parse; fall back to a single map index with the raw field string
	// otherwise (still correct, just not as readable).
	for _, seg := range strings.Split(field, ".") {
		if !isValidCELIdent(seg) {
			return fmt.Sprintf("data[%s]", strconv.Quote(field))
		}
	}
	return "data." + field
}

func isValidCELIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		isAlpha := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
		isDigit := r >= '0' && r <= '9'
		if i == 0 && !isAlpha {
			return false
		}
		if i > 0 && !isAlpha && !isDigit {
			return false
		}
	}
	return true
}

// celLiteral renders a Go value (decoded from condition_json) as a CEL
// literal. Strings are quoted; numbers/bools are rendered as-is; anything
// else falls back to its JSON encoding, which is valid CEL for maps/lists.
func celLiteral(v interface{}) string {
	switch val := v.(type) {
	case string:
		return strconv.Quote(val)
	case bool:
		return strconv.FormatBool(val)
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64)
	case nil:
		return "null"
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return strconv.Quote(fmt.Sprintf("%v", val))
		}
		return string(b)
	}
}

// celNumberLiteral renders a value as a CEL double literal, explicitly
// wrapped in double(...) so it type-checks against double(field) regardless
// of whether the JSON number happened to be whole (which CEL would
// otherwise parse as an int literal, incompatible with a double operand -
// CEL has no int/double overload for comparison operators). Also coerces
// strings that look numeric (a common condition_json authoring artifact).
func celNumberLiteral(v interface{}) string {
	switch val := v.(type) {
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return fmt.Sprintf("double(%s)", strconv.FormatFloat(f, 'g', -1, 64))
		}
		return strconv.Quote(val)
	case float64:
		return fmt.Sprintf("double(%s)", strconv.FormatFloat(val, 'g', -1, 64))
	default:
		return celLiteral(v)
	}
}

func celListLiteral(v interface{}) string {
	list, ok := v.([]interface{})
	if !ok {
		// A single non-array value used with in/not_in: treat as a
		// one-element list rather than producing invalid CEL syntax.
		return "[" + celLiteral(v) + "]"
	}
	parts := make([]string, len(list))
	for i, item := range list {
		parts[i] = celLiteral(item)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// daysToHoursLiteral converts a "days ago" numeric threshold into a CEL
// duration literal string (e.g. 3 days -> "72h"). CEL's duration() parser
// has no day unit, only up to hours.
func daysToHoursLiteral(v interface{}) (string, error) {
	var days float64
	switch val := v.(type) {
	case float64:
		days = val
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return "", fmt.Errorf("days_ago_less_than requires a number, got %q", val)
		}
		days = f
	default:
		return "", fmt.Errorf("days_ago_less_than requires a number, got %T", v)
	}
	return fmt.Sprintf("%gh", days*24), nil
}

func betweenBounds(v interface{}) (string, string, error) {
	switch val := v.(type) {
	case []interface{}:
		if len(val) != 2 {
			return "", "", fmt.Errorf("between requires exactly 2 values, got %d", len(val))
		}
		return celNumberLiteral(val[0]), celNumberLiteral(val[1]), nil
	case map[string]interface{}:
		min, hasMin := val["min"]
		max, hasMax := val["max"]
		if !hasMin || !hasMax {
			return "", "", fmt.Errorf("between map requires min and max keys")
		}
		return celNumberLiteral(min), celNumberLiteral(max), nil
	default:
		return "", "", fmt.Errorf("between requires a [min, max] array or {min, max} map, got %T", v)
	}
}
