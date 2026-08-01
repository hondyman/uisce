package fix

import (
	"github.com/hondyman/uisce/backend/internal/rules"
)

type ruleEngineAdapter struct {
	engine *rules.RuleEngine
}

func (a *ruleEngineAdapter) EvaluateTrade(trade ExternalTradeItem) EvaluateResult {
	hybridRecord := map[string]any{
		"order.quantity": trade.Quantity,
		"order.price":    trade.Price,
	}
	if trade.Symbol != "" {
		hybridRecord["security.symbol"] = trade.Symbol
	}
	if trade.ISIN != "" {
		hybridRecord["security.isin"] = trade.ISIN
	}
	if trade.ExternalOrderID != "" {
		hybridRecord["order.external_id"] = trade.ExternalOrderID
	}
	if trade.PortfolioID != "" {
		hybridRecord["portfolio.id"] = trade.PortfolioID
	}
	if trade.Side != "" {
		hybridRecord["order.side"] = trade.Side
	}
	if trade.Account != "" {
		hybridRecord["account.id"] = trade.Account
	}

	batchResult, _ := a.engine.EvaluateGroup(nil, "", nil, hybridRecord)

	approved := batchResult != nil && batchResult.PassedAll
	violations := extractViolationsFromBatch(batchResult)

	return EvaluateResult{
		Approved:       approved,
		CanOverride:    !approved && len(violations) > 0,
		HighestSeverity: rollupSeverity(batchResult),
		Violations:     violations,
	}
}

func extractViolationsFromBatch(batchResult *rules.BatchResult) []Violation {
	if batchResult == nil {
		return nil
	}
	var violations []Violation
	for _, res := range batchResult.Results {
		if !res.Passed {
			for _, v := range res.Violations {
				violations = append(violations, Violation{
					RuleID:    v.RuleID,
					FieldPath: v.FieldPath,
					Message:   v.Message,
				})
			}
		}
	}
	return violations
}

func rollupSeverity(batchResult *rules.BatchResult) string {
	if batchResult == nil {
		return "INFORMATIONAL"
	}
	for _, res := range batchResult.Results {
		if res.Severity == rules.SeverityHardBlock {
			return "HARD_BLOCK"
		}
	}
	for _, res := range batchResult.Results {
		if res.Severity == rules.SeverityError {
			return "HARD_BLOCK"
		}
	}
	for _, res := range batchResult.Results {
		if res.Severity == rules.SeverityWarning {
			return "SOFT_WARNING"
		}
	}
	return "INFORMATIONAL"
}

func RuleEngineToComplianceEvaluator(engine *rules.RuleEngine) ComplianceEvaluator {
	return &ruleEngineAdapter{engine: engine}
}

