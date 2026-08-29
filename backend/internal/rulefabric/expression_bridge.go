package rulefabric

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// This file bridges rulefabric's Condition/ConditionGroup model to
// boresolver's FilterClause/FilterGroup (internal/boresolver/expression_builder.go),
// so a compliance/MDM rule authored once can be pushed down to SQL to
// pre-filter candidate rows at the database layer instead of pulling every
// row into Go and evaluating rulefabric's per-condition evaluator on each
// one - the difference between "scan and evaluate a million rows in
// process" and "let the database eliminate the 999,000 that can't possibly
// match, then run the detailed per-record evaluator (with its audit-quality
// ConditionResult/FailureReasons) on the handful that remain."
//
// rulefabric's own evaluateCondition/evaluateConditionGroup are NOT
// replaced - they still produce the per-field Actual/Expected/Message audit
// trail a compliance rule needs. This bridge is the bulk pre-filter stage
// that makes running that detailed evaluator at MDM/compliance scale
// (millions of rows, hundreds of thousands of rules) tractable.

// ruleOperatorToFilterOperator maps rulefabric's OperatorRegistry names to
// boresolver's operator vocabulary. Only operators boresolver can express as
// SQL are listed - anything absent here (matches_regex, not_contains,
// date_before/date_after/days_ago_less_than) is not SQL-pushable today, so
// ToFilterGroup reports it via the returned unsupported list rather than
// guessing at a lossy translation.
var ruleOperatorToFilterOperator = map[string]string{
	"equals":                 "EQ",
	"not_equals":             "NEQ",
	"is_null":                "IS_NULL",
	"is_not_null":            "IS_NOT_NULL",
	"greater_than":           "GT",
	"greater_than_or_equals": "GTE",
	"less_than":              "LT",
	"less_than_or_equals":    "LTE",
	"between":                "BETWEEN",
	"contains":               "CONTAINS",
	"starts_with":            "STARTS_WITH",
	"ends_with":              "ENDS_WITH",
	"in":                     "IN",
	"not_in":                 "NOT IN",
}

// ToFilterGroup converts a rulefabric ConditionGroup into a boresolver
// FilterGroup for SQL pushdown. fieldToBOFieldID resolves a rulefabric
// Condition.Field (e.g. "customer.kyc_status") to the Business Object field
// UUID boresolver's ResolvePath expects. Any condition using an operator or
// cross-entity path not representable in SQL is collected into `unsupported`
// (by field name) rather than silently dropped or mistranslated - callers
// should treat a non-empty unsupported list as "don't trust this as a
// complete pre-filter, only as a narrowing hint" (or skip pushdown entirely
// for that rule and run the full per-record evaluator).
func ToFilterGroup(group *ConditionGroup, fieldToBOFieldID func(fieldName string) (string, error)) (*boresolver.FilterGroup, []string, error) {
	if group == nil {
		return nil, nil, nil
	}

	conj := strings.ToUpper(strings.TrimSpace(group.Operator))
	if conj != "OR" {
		conj = "AND" // rulefabric's "NOT" single-child group has no direct
		// FilterGroup equivalent (FilterGroup has no unary negation);
		// treated as AND over its (unsupported-flagged) child instead of
		// silently inverting the wrong thing.
	}

	out := &boresolver.FilterGroup{Conjunction: conj}
	var unsupported []string

	for _, raw := range group.Conditions {
		condMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := condMap["type"].(string)

		if condType == "group" {
			var sub ConditionGroup
			b, _ := json.Marshal(condMap)
			if err := json.Unmarshal(b, &sub); err != nil {
				return nil, nil, fmt.Errorf("failed to decode nested condition group: %w", err)
			}
			subFilter, subUnsupported, err := ToFilterGroup(&sub, fieldToBOFieldID)
			if err != nil {
				return nil, nil, err
			}
			unsupported = append(unsupported, subUnsupported...)
			if subFilter != nil {
				out.Groups = append(out.Groups, *subFilter)
			}
			continue
		}

		var cond Condition
		b, _ := json.Marshal(condMap)
		if err := json.Unmarshal(b, &cond); err != nil {
			return nil, nil, fmt.Errorf("failed to decode condition: %w", err)
		}

		if cond.EntityPath != nil {
			// Cross-entity conditions need a join boresolver's field
			// resolution alone can't infer from a field name string.
			unsupported = append(unsupported, cond.Field)
			continue
		}

		filterOp, ok := ruleOperatorToFilterOperator[strings.ToLower(strings.TrimSpace(cond.Operator))]
		if !ok {
			unsupported = append(unsupported, cond.Field)
			continue
		}

		fieldID, err := fieldToBOFieldID(cond.Field)
		if err != nil {
			unsupported = append(unsupported, cond.Field)
			continue
		}

		out.Conditions = append(out.Conditions, boresolver.FilterClause{
			FieldID:  fieldID,
			Operator: filterOp,
			Value:    cond.Value,
		})
	}

	return out, unsupported, nil
}

// CompilePreFilterSQL is the MDM/compliance-scale entry point: given a rule's
// ConditionGroup, produce a parameterized SQL WHERE fragment (via
// boresolver.CompileFilterGroup) that narrows a bulk scan to only rows that
// could possibly satisfy the rule, before the detailed per-record evaluator
// runs. ok=false means the rule has conditions this bridge can't safely
// express in SQL (see ToFilterGroup) - callers must fall back to evaluating
// every row through the normal in-process evaluator for that rule rather
// than trust a partial/incorrect filter.
func CompilePreFilterSQL(g *boresolver.BOSQLGenerator, ctx *boresolver.GenerationContext, group *ConditionGroup, fieldToBOFieldID func(fieldName string) (string, error)) (sqlFragment string, ok bool, err error) {
	filterGroup, unsupported, err := ToFilterGroup(group, fieldToBOFieldID)
	if err != nil {
		return "", false, err
	}
	if filterGroup == nil || len(unsupported) > 0 {
		return "", false, nil
	}
	sqlFragment, err = boresolver.CompileFilterGroup(g, ctx, *filterGroup)
	if err != nil {
		return "", false, err
	}
	return sqlFragment, true, nil
}
