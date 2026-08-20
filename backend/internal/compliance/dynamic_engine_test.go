package compliance

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestEvaluateRule_ZeroHardcodeThresholds(t *testing.T) {
	engine := NewDynamicComplianceEngine(nil)
	ctx := context.Background()

	ruleID := uuid.New()
	rule := DynamicComplianceRule{
		RuleID:  ruleID,
		RuleKey: "sec_sector_active_weight_limit",
		Tolerance: ToleranceConfig{
			Unit:          "PERCENT",
			AlertAbove:    10.0,
			AlertBelow:    -10.0,
			SeverityAbove: "HARD_BREACH",
			SeverityBelow: "HARD_BREACH",
		},
	}

	rows := []GroupedComparisonRow{
		{
			GroupName:       "Information Technology",
			PortfolioMetric: 35.0,
			BenchmarkMetric: 20.0,
			ActiveDelta:     15.0, // +15.0 > +10.0 -> Breach
		},
		{
			GroupName:       "Financials",
			PortfolioMetric: 12.0,
			BenchmarkMetric: 15.0,
			ActiveDelta:     -3.0, // -3.0 within [-10, +10] -> No breach
		},
		{
			GroupName:       "Health Care",
			PortfolioMetric: 5.0,
			BenchmarkMetric: 20.0,
			ActiveDelta:     -15.0, // -15.0 < -10.0 -> Breach
		},
	}

	breaches := engine.EvaluateRule(ctx, rule, rows)

	if len(breaches) != 2 {
		t.Fatalf("expected 2 breaches, got %d", len(breaches))
	}

	// Verify IT breach
	itBreach := breaches[0]
	if itBreach.GroupKey != "Information Technology" {
		t.Errorf("expected IT breach, got %s", itBreach.GroupKey)
	}
	if itBreach.ActiveDelta != 15.0 {
		t.Errorf("expected active delta 15.0, got %f", itBreach.ActiveDelta)
	}
	if itBreach.BreachType != "HARD_BREACH" {
		t.Errorf("expected HARD_BREACH, got %s", itBreach.BreachType)
	}

	// Verify Health Care breach
	hcBreach := breaches[1]
	if hcBreach.GroupKey != "Health Care" {
		t.Errorf("expected Health Care breach, got %s", hcBreach.GroupKey)
	}
	if hcBreach.ActiveDelta != -15.0 {
		t.Errorf("expected active delta -15.0, got %f", hcBreach.ActiveDelta)
	}
}

func TestCompileBenchmarkSQL_RuleValidation(t *testing.T) {
	engine := NewDynamicComplianceEngine(nil)

	rule := DynamicComplianceRule{
		RuleKey:           "sec_sector_active_weight_limit",
		GroupingDimension: "bloomberg_industry_sector",
	}

	sqlQuery := engine.CompileBenchmarkSQL(rule, "industry_sector", "h.investment_class = 'Equity'")

	// Verify Rule 7 ABAC & Tenant isolation parameter
	if !strings.Contains(sqlQuery, "h.tenant_id = :tenant_id") {
		t.Errorf("expected query to enforce Rule 7 multi-tenant isolation on portfolio CTE")
	}
	if !strings.Contains(sqlQuery, "b.tenant_id = :tenant_id") {
		t.Errorf("expected query to enforce Rule 7 multi-tenant isolation on benchmark CTE")
	}

	// Verify grouping dimension
	if !strings.Contains(sqlQuery, "GROUP BY h.industry_sector") {
		t.Errorf("expected query to group by industry_sector")
	}

	// Verify filter pushdown
	if !strings.Contains(sqlQuery, "h.investment_class = 'Equity'") {
		t.Errorf("expected filter pushdown to be injected")
	}
}
