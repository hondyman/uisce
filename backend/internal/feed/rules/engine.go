package rules

import (
	"context"
	"fmt"

	clientContext "github.com/hondyman/semlayer/backend/internal/context"
	"github.com/hondyman/semlayer/backend/pkg/policy"
)

// RuleEngine evaluates card rules against client context
type RuleEngine struct {
	rules        []CardRule
	celEvaluator *policy.CELEvaluator
}

func NewRuleEngine() (*RuleEngine, error) {
	celEval, err := policy.NewCELEvaluator()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL evaluator: %w", err)
	}

	return &RuleEngine{
		rules:        getHardcodedRulesWithCEL(),
		celEvaluator: celEval,
	}, nil
}

// EvaluateRule checks if a card rule is eligible
func (e *RuleEngine) EvaluateRule(ctx context.Context, rule CardRule, clientCtx clientContext.ClientContext) (*EvaluationResult, error) {
	// Build CEL evaluation context
	vars := map[string]interface{}{
		"client": map[string]interface{}{
			"Profile": map[string]interface{}{
				"TaxStatus": clientCtx.Profile.TaxStatus,
			},
			"Portfolio": map[string]interface{}{
				"UnrealizedLossPct": clientCtx.Portfolio.UnrealizedLossPct,
				"DriftPct":          clientCtx.Portfolio.DriftPct,
			},
			"Compliance": map[string]interface{}{
				"IsRestricted": clientCtx.Compliance.IsRestricted,
			},
		},
	}

	// Use CEL evaluator if rule has CEL expression
	if rule.CELEligibility != "" {
		eligible, err := e.celEvaluator.EvalBool(ctx, rule.CELEligibility, vars)
		if err != nil {
			return &EvaluationResult{
				CardID:   rule.CardID,
				Eligible: false,
				Reason:   fmt.Sprintf("CEL error: %v", err),
			}, nil
		}

		if !eligible {
			return &EvaluationResult{
				CardID:   rule.CardID,
				Eligible: false,
				Reason:   "CEL check failed",
			}, nil
		}

		// Calculate rank
		rankScore := 1.0
		if rule.CELRankScore != "" {
			score, err := e.celEvaluator.EvalNumber(ctx, rule.CELRankScore, vars)
			if err == nil {
				rankScore = score
			}
		}

		return &EvaluationResult{
			CardID:    rule.CardID,
			Eligible:  true,
			RankScore: rankScore,
			Context:   vars,
		}, nil
	}

	// Fallback to hardcoded
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
	return &EvaluationResult{
		CardID:    rule.CardID,
		Eligible:  true,
		RankScore: 1.0,
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

// CEL-enabled rules  
func getHardcodedRulesWithCEL() []CardRule {
	return []CardRule{
		{
			CardID:         "welcome_message",
			CELEligibility: "true",
			CELRankScore:   "1.0",
		},
		{
			CardID:         "tax_loss_harvest",
			CELEligibility: `client.Portfolio.UnrealizedLossPct < -0.01 && client.Profile.TaxStatus == "taxable" && !client.Compliance.IsRestricted`,
			CELRankScore:   "abs(client.Portfolio.UnrealizedLossPct) * 100.0",
		},
		{
			CardID:         "portfolio_drift",
			CELEligibility: "client.Portfolio.DriftPct > 0.05",
			CELRankScore:   "client.Portfolio.DriftPct * 100.0",
		},
	}
}
