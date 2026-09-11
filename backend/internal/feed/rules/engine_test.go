package rules

import (
	"context"
	"testing"

	clientContext "github.com/hondyman/uisce/backend/internal/context"
)

func TestRuleEngine_WelcomeMessage(t *testing.T) {
	engine, err := NewRuleEngine()
	if err != nil {
		t.Fatalf("NewRuleEngine: %v", err)
	}

	loadRule := func() *CardRule {
		rules, err := engine.LoadRules()
		if err != nil {
			t.Fatalf("LoadRules: %v", err)
		}
		for _, r := range rules {
			if r.CardID == "welcome_message" {
				return r
			}
		}
		t.Fatal("welcome_message rule not found")
		return nil
	}

	t.Run("eligible when UnrealizedLossPct is very negative", func(t *testing.T) {
		ctx := clientContext.ClientContext{
			Portfolio: clientContext.PortfolioSummary{
				UnrealizedLossPct: -0.05,
				DriftPct:          0.01,
			},
		}
		rule := loadRule()
		result, err := engine.EvaluateRule(context.Background(), *rule, ctx)
		if err != nil {
			t.Fatalf("EvaluateRule: %v", err)
		}
		if !result.Eligible {
			t.Errorf("expected eligible, got ineligible: %s", result.Reason)
		}
		if result.RankScore != 1.0 {
			t.Errorf("expected RankScore 1.0, got %v", result.RankScore)
		}
	})

	t.Run("eligible when UnrealizedLossPct is positive", func(t *testing.T) {
		ctx := clientContext.ClientContext{
			Portfolio: clientContext.PortfolioSummary{
				UnrealizedLossPct: 0.05,
				DriftPct:          0.01,
			},
		}
		rule := loadRule()
		result, _ := engine.EvaluateRule(context.Background(), *rule, ctx)
		if !result.Eligible {
			t.Errorf("expected eligible (condition always true for real values), got ineligible: %s", result.Reason)
		}
	})
}

func TestRuleEngine_TaxLossHarvest(t *testing.T) {
	engine, err := NewRuleEngine()
	if err != nil {
		t.Fatalf("NewRuleEngine: %v", err)
	}

	loadRule := func() *CardRule {
		rules, _ := engine.LoadRules()
		for _, r := range rules {
			if r.CardID == "tax_loss_harvest" {
				return r
			}
		}
		t.Fatal("tax_loss_harvest rule not found")
		return nil
	}

	t.Run("eligible — loss below threshold, taxable, not restricted", func(t *testing.T) {
		ctx := clientContext.ClientContext{
			Portfolio: clientContext.PortfolioSummary{
				UnrealizedLossPct: -0.05,
				DriftPct:          0.01,
			},
			Profile:    clientContext.ClientProfile{TaxStatus: "taxable"},
			Compliance: clientContext.ComplianceStatus{IsRestricted: false},
		}
		rule := loadRule()
		result, err := engine.EvaluateRule(context.Background(), *rule, ctx)
		if err != nil {
			t.Fatalf("EvaluateRule: %v", err)
		}
		if !result.Eligible {
			t.Errorf("expected eligible, got ineligible: %s", result.Reason)
		}
		if result.RankScore != 5.0 {
			t.Errorf("expected RankScore 5.0 (abs(-0.05)*100), got %v", result.RankScore)
		}
	})

	t.Run("eligible — larger loss yields higher rank score", func(t *testing.T) {
		ctxLarge := clientContext.ClientContext{
			Portfolio:  clientContext.PortfolioSummary{UnrealizedLossPct: -0.15},
			Profile:    clientContext.ClientProfile{TaxStatus: "taxable"},
			Compliance: clientContext.ComplianceStatus{IsRestricted: false},
		}
		ctxSmall := clientContext.ClientContext{
			Portfolio:  clientContext.PortfolioSummary{UnrealizedLossPct: -0.05},
			Profile:    clientContext.ClientProfile{TaxStatus: "taxable"},
			Compliance: clientContext.ComplianceStatus{IsRestricted: false},
		}
		rule := loadRule()

		resultLarge, _ := engine.EvaluateRule(context.Background(), *rule, ctxLarge)
		resultSmall, _ := engine.EvaluateRule(context.Background(), *rule, ctxSmall)

		if !resultLarge.Eligible || !resultSmall.Eligible {
			t.Fatalf("both should be eligible: large=%v small=%v", resultLarge.Eligible, resultSmall.Eligible)
		}
		if resultLarge.RankScore <= resultSmall.RankScore {
			t.Errorf("larger loss should produce higher rank score (%v > %v); got opposite", resultLarge.RankScore, resultSmall.RankScore)
		}
		// abs(-0.15)*100 = 15.0, abs(-0.05)*100 = 5.0
		if resultLarge.RankScore != 15.0 || resultSmall.RankScore != 5.0 {
			t.Errorf("unexpected rank scores: large=%v small=%v", resultLarge.RankScore, resultSmall.RankScore)
		}
	})

	t.Run("ineligible — loss above threshold (not below -1%)", func(t *testing.T) {
		ctx := clientContext.ClientContext{
			Portfolio:  clientContext.PortfolioSummary{UnrealizedLossPct: -0.005},
			Profile:    clientContext.ClientProfile{TaxStatus: "taxable"},
			Compliance: clientContext.ComplianceStatus{IsRestricted: false},
		}
		rule := loadRule()
		result, _ := engine.EvaluateRule(context.Background(), *rule, ctx)
		if result.Eligible {
			t.Error("expected ineligible (-0.005 is not < -0.01), got eligible")
		}
	})

	t.Run("ineligible — non-taxable account", func(t *testing.T) {
		ctx := clientContext.ClientContext{
			Portfolio:  clientContext.PortfolioSummary{UnrealizedLossPct: -0.05},
			Profile:    clientContext.ClientProfile{TaxStatus: "tax_exempt"},
			Compliance: clientContext.ComplianceStatus{IsRestricted: false},
		}
		rule := loadRule()
		result, _ := engine.EvaluateRule(context.Background(), *rule, ctx)
		if result.Eligible {
			t.Error("expected ineligible (tax_exempt), got eligible")
		}
	})

	t.Run("ineligible — restricted account", func(t *testing.T) {
		ctx := clientContext.ClientContext{
			Portfolio:  clientContext.PortfolioSummary{UnrealizedLossPct: -0.05},
			Profile:    clientContext.ClientProfile{TaxStatus: "taxable"},
			Compliance: clientContext.ComplianceStatus{IsRestricted: true},
		}
		rule := loadRule()
		result, _ := engine.EvaluateRule(context.Background(), *rule, ctx)
		if result.Eligible {
			t.Error("expected ineligible (restricted), got eligible")
		}
	})
}

func TestRuleEngine_PortfolioDrift(t *testing.T) {
	engine, err := NewRuleEngine()
	if err != nil {
		t.Fatalf("NewRuleEngine: %v", err)
	}

	loadRule := func() *CardRule {
		rules, _ := engine.LoadRules()
		for _, r := range rules {
			if r.CardID == "portfolio_drift" {
				return r
			}
		}
		t.Fatal("portfolio_drift rule not found")
		return nil
	}

	t.Run("eligible — drift above 5%", func(t *testing.T) {
		ctx := clientContext.ClientContext{
			Portfolio: clientContext.PortfolioSummary{DriftPct: 0.08},
		}
		rule := loadRule()
		result, err := engine.EvaluateRule(context.Background(), *rule, ctx)
		if err != nil {
			t.Fatalf("EvaluateRule: %v", err)
		}
		if !result.Eligible {
			t.Errorf("expected eligible, got ineligible: %s", result.Reason)
		}
		if result.RankScore != 8.0 {
			t.Errorf("expected RankScore 8.0 (0.08*100), got %v", result.RankScore)
		}
	})

	t.Run("eligible — higher drift yields higher rank score", func(t *testing.T) {
		ctxHigh := clientContext.ClientContext{
			Portfolio: clientContext.PortfolioSummary{DriftPct: 0.12},
		}
		ctxLow := clientContext.ClientContext{
			Portfolio: clientContext.PortfolioSummary{DriftPct: 0.06},
		}
		rule := loadRule()

		resultHigh, _ := engine.EvaluateRule(context.Background(), *rule, ctxHigh)
		resultLow, _ := engine.EvaluateRule(context.Background(), *rule, ctxLow)

		if !resultHigh.Eligible || !resultLow.Eligible {
			t.Fatalf("both should be eligible: high=%v low=%v", resultHigh.Eligible, resultLow.Eligible)
		}
		if resultHigh.RankScore <= resultLow.RankScore {
			t.Errorf("higher drift should produce higher rank score (%v > %v); got opposite", resultHigh.RankScore, resultLow.RankScore)
		}
		// 0.12*100 = 12.0, 0.06*100 = 6.0
		if resultHigh.RankScore != 12.0 || resultLow.RankScore != 6.0 {
			t.Errorf("unexpected rank scores: high=%v low=%v", resultHigh.RankScore, resultLow.RankScore)
		}
	})

	t.Run("ineligible — drift at exactly 5%", func(t *testing.T) {
		ctx := clientContext.ClientContext{
			Portfolio: clientContext.PortfolioSummary{DriftPct: 0.05},
		}
		rule := loadRule()
		result, _ := engine.EvaluateRule(context.Background(), *rule, ctx)
		if result.Eligible {
			t.Error("expected ineligible (0.05 is not > 0.05), got eligible")
		}
	})

	t.Run("ineligible — drift below 5%", func(t *testing.T) {
		ctx := clientContext.ClientContext{
			Portfolio: clientContext.PortfolioSummary{DriftPct: 0.03},
		}
		rule := loadRule()
		result, _ := engine.EvaluateRule(context.Background(), *rule, ctx)
		if result.Eligible {
			t.Error("expected ineligible, got eligible")
		}
	})
}

func TestRuleEngine_RankScoreExpressions(t *testing.T) {
	engine, err := NewRuleEngine()
	if err != nil {
		t.Fatalf("NewRuleEngine: %v", err)
	}

	rules, _ := engine.LoadRules()
	for _, rule := range rules {
		if rule.RankScoreExpr == "" {
			t.Errorf("rule %s has empty RankScoreExpr", rule.CardID)
		}
	}

	byID := loadRulesByID(rules, "tax_loss_harvest", "portfolio_drift")
	taxLossRule := byID["tax_loss_harvest"]
	portfolioDriftRule := byID["portfolio_drift"]

	if taxLossRule.RankScoreExpr != "(-client.Portfolio.UnrealizedLossPct) * 100.0" {
		t.Errorf("unexpected tax_loss_harvest rank expr: %q", taxLossRule.RankScoreExpr)
	}
	if portfolioDriftRule.RankScoreExpr != "client.Portfolio.DriftPct * 100.0" {
		t.Errorf("unexpected portfolio_drift rank expr: %q", portfolioDriftRule.RankScoreExpr)
	}
}

func loadRulesByID(rules []*CardRule, ids ...string) map[string]*CardRule {
	result := make(map[string]*CardRule)
	for _, r := range rules {
		for _, id := range ids {
			if r.CardID == id {
				result[id] = r
			}
		}
	}
	return result
}
