package rules

import (
	"context"
	"fmt"

	clientContext "github.com/hondyman/uisce/backend/internal/context"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// RuleEngine evaluates card rules against client context
type RuleEngine struct {
	rules       []CardRule
	vmEvaluator *vm.AdvancedEvaluator
}

func NewRuleEngine() (*RuleEngine, error) {
	return &RuleEngine{
		rules:       getHardcodedRules(),
		vmEvaluator: vm.NewAdvancedEvaluator(),
	}, nil
}

// EvaluateRule checks if a card rule is eligible
func (e *RuleEngine) EvaluateRule(ctx context.Context, rule CardRule, clientCtx clientContext.ClientContext) (*EvaluationResult, error) {
	return e.evaluateHardcoded(rule, clientCtx)
}

func (e *RuleEngine) evaluateHardcoded(rule CardRule, clientCtx clientContext.ClientContext) (*EvaluationResult, error) {
	for _, cond := range rule.Conditions {
		met, err := e.evaluateCondition(cond, clientCtx)
		if err != nil || !met {
			return &EvaluationResult{
				CardID:   rule.CardID,
				Eligible: false,
				Reason:   fmt.Sprintf("Condition not met: %s", cond.Field),
			}, err
		}
	}
		rankScore := 1.0
	if rule.RankScoreExpr != "" {
		vars := map[string]interface{}{
			"client": map[string]interface{}{
				"Portfolio": map[string]interface{}{
					"UnrealizedLossPct": clientCtx.Portfolio.UnrealizedLossPct,
					"DriftPct":          clientCtx.Portfolio.DriftPct,
				},
				"Profile": map[string]interface{}{
					"TaxStatus": clientCtx.Profile.TaxStatus,
				},
				"Compliance": map[string]interface{}{
					"IsRestricted": clientCtx.Compliance.IsRestricted,
				},
			},
		}
		expr, err := vm.ParseExpression(rule.RankScoreExpr)
		if err != nil {
			return &EvaluationResult{
				CardID:   rule.CardID,
				Eligible: false,
				Reason:   fmt.Sprintf("rank expression parse error: %v", err),
			}, err
		}
		node := vm.RuleNode{Type: vm.NodeTypeExpression, Expression: expr}
		score, err := e.vmEvaluator.EvaluateNumeric(node, vars)
		if err != nil {
			return &EvaluationResult{
				CardID:   rule.CardID,
				Eligible: false,
				Reason:   fmt.Sprintf("rank expression eval error: %v", err),
			}, err
		}
		rankScore = score
	}
	return &EvaluationResult{
		CardID:    rule.CardID,
		Eligible:  true,
		RankScore: rankScore,
	}, nil
}

func (e *RuleEngine) evaluateCondition(cond RuleCondition, ctx clientContext.ClientContext) (bool, error) {
	var fieldValue interface{}
	switch cond.Field {
	case "Portfolio.UnrealizedLossPct":
		fieldValue = ctx.Portfolio.UnrealizedLossPct
	case "Profile.TaxStatus":
		fieldValue = ctx.Profile.TaxStatus
	case "Compliance.IsRestricted":
		fieldValue = ctx.Compliance.IsRestricted
	case "Portfolio.DriftPct":
		fieldValue = ctx.Portfolio.DriftPct
	default:
		return false, fmt.Errorf("unknown field: %s", cond.Field)
	}

	switch cond.Operator {
	case "lt":
		if fv, ok := fieldValue.(float64); ok {
			if cv, ok := cond.Value.(float64); ok {
				return fv < cv, nil
			}
		}
	case "eq":
		return fieldValue == cond.Value, nil
	case "gt":
		if fv, ok := fieldValue.(float64); ok {
			if cv, ok := cond.Value.(float64); ok {
				return fv > cv, nil
			}
		}
	}
	return false, nil
}

// GetEligibleRules returns all eligible rules
func (e *RuleEngine) GetEligibleRules(ctx context.Context, clientCtx clientContext.ClientContext) ([]EvaluationResult, error) {
	var results []EvaluationResult
	for _, rule := range e.rules {
		result, err := e.EvaluateRule(ctx, rule, clientCtx)
		if err != nil {
			return nil, err
		}
		if result.Eligible {
			results = append(results, *result)
		}
	}
	return results, nil
}

// LoadRules returns the configured rules
func (e *RuleEngine) LoadRules() ([]*CardRule, error) {
	result := make([]*CardRule, len(e.rules))
	for i := range e.rules {
		result[i] = &e.rules[i]
	}
	return result, nil
}

// Evaluate evaluates a single rule against client context
func (e *RuleEngine) Evaluate(rule *CardRule, ctx *clientContext.ClientContext) *EvaluationResult {
	result, _ := e.EvaluateRule(context.Background(), *rule, *ctx)
	if result == nil {
		return &EvaluationResult{
			CardID:   rule.CardID,
			Eligible: false,
			Reason:   "Error evaluating rule",
		}
	}
	return result
}

// getHardcodedRules returns card rules using the Conditions field,
// evaluated via evaluateHardcoded (no CEL dependency). RankScore is
// computed via vm expression (see RankScoreExpr on CardRule).
func getHardcodedRules() []CardRule {
	return []CardRule{
		{
			CardID:        "welcome_message",
			RankScoreExpr: "1.0",
			Conditions: []RuleCondition{
				{Field: "Portfolio.UnrealizedLossPct", Operator: "gt", Value: -999999.0},
			},
		},
		{
			CardID:        "tax_loss_harvest",
			RankScoreExpr: "(-client.Portfolio.UnrealizedLossPct) * 100.0",
			Conditions: []RuleCondition{
				{Field: "Portfolio.UnrealizedLossPct", Operator: "lt", Value: -0.01},
				{Field: "Profile.TaxStatus", Operator: "eq", Value: "taxable"},
				{Field: "Compliance.IsRestricted", Operator: "eq", Value: false},
			},
		},
		{
			CardID:        "portfolio_drift",
			RankScoreExpr: "client.Portfolio.DriftPct * 100.0",
			Conditions: []RuleCondition{
				{Field: "Portfolio.DriftPct", Operator: "gt", Value: 0.05},
			},
		},
	}
}
