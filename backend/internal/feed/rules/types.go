package rules

import (
	clientContext "github.com/hondyman/semlayer/backend/internal/context"
)

// CardRule defines a rule for showing a feed card
type CardRule struct {
	CardID         string
	Conditions     []RuleCondition
	CELEligibility string  // CEL expression for eligibility (if set, overrides Conditions)
	CELRankScore   string  // CEL expression for calculating rank score
}

// RuleCondition defines a single condition within a rule
type RuleCondition struct {
	Field    string      // e.g., "Portfolio.UnrealizedLossPct", "Portfolio.DriftPct"
	Operator string      // "lt", "lte", "gt", "gte", "eq", "contains"
	Value    interface{} // comparison value
}

// EvaluationResult contains the result of rule evaluation
type EvaluationResult struct {
	CardID    string
	Eligible  bool
	Reason    string
	RankScore float64
	Context   map[string]interface{} // CEL evaluation context for debugging
}

// RuleEvaluator evaluates rules against client context
type RuleEvaluator interface {
	Evaluate(rule *CardRule, ctx *clientContext.ClientContext) *EvaluationResult
	LoadRules() ([]*CardRule, error)
}
