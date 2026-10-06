package library

import "fmt"

// getPhase1PostTradeScenarios returns the deterministic scenario vectors for Phase 1 Post-Trade Rules.
func getPhase1PostTradeScenarios() []Scenario {
	return []Scenario{
		// ====================================================================
		// 51. UCITS_5_10_40 (Post-Trade 5/10/40 concentration limits)
		// ====================================================================
		{
			RuleCode:    "UCITS_5_10_40",
			Code:        "UCITS_5_10_40:PASS",
			Description: "Compliant aggregate exposure of >5% issuers (35% <= 40%)",
			Input: OrderContext{
				UcitsAggregateAbove5PctExposure: d("0.350000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "UCITS_5_10_40",
			Code:        "UCITS_5_10_40:BOUNDARY",
			Description: "Boundary aggregate exposure exactly at 40% limit",
			Input: OrderContext{
				UcitsAggregateAbove5PctExposure: d("0.400000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "UCITS_5_10_40",
			Code:        "UCITS_5_10_40:FAIL",
			Description: "Breached aggregate exposure above 40% (42% > 40%)",
			Input: OrderContext{
				UcitsAggregateAbove5PctExposure: d("0.420000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "ucits_aggregate_above_5pct_exposure"},
		},
		{
			RuleCode:    "UCITS_5_10_40",
			Code:        "UCITS_5_10_40:ADVERSARIAL",
			Description: "Extreme concentration above 40% (65% > 40%)",
			Input: OrderContext{
				UcitsAggregateAbove5PctExposure: d("0.650000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "ucits_aggregate_above_5pct_exposure"},
		},

		// ====================================================================
		// 52. SEC_144A_QIB_HOLDING (Post-Trade 15% QIB / Illiquid Asset Limit)
		// ====================================================================
		{
			RuleCode:    "SEC_144A_QIB_HOLDING",
			Code:        "SEC_144A_QIB_HOLDING:PASS",
			Description: "Compliant restricted 144A non-QIB holding (10% <= 15%)",
			Input: OrderContext{
				Restricted144aExposurePct: d("0.100000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "SEC_144A_QIB_HOLDING",
			Code:        "SEC_144A_QIB_HOLDING:BOUNDARY",
			Description: "Boundary restricted 144A holding exactly at 15% limit",
			Input: OrderContext{
				Restricted144aExposurePct: d("0.150000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "SEC_144A_QIB_HOLDING",
			Code:        "SEC_144A_QIB_HOLDING:FAIL",
			Description: "Breached restricted 144A holding (18% > 15%)",
			Input: OrderContext{
				Restricted144aExposurePct: d("0.180000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "restricted_144a_exposure_pct"},
		},
		{
			RuleCode:    "SEC_144A_QIB_HOLDING",
			Code:        "SEC_144A_QIB_HOLDING:ADVERSARIAL",
			Description: "Extreme restricted 144A holding (50% > 15%)",
			Input: OrderContext{
				Restricted144aExposurePct: d("0.500000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "restricted_144a_exposure_pct"},
		},

		// ====================================================================
		// 53. MARGIN_UTILIZATION_80 (Post-Trade Margin Capacity Warning)
		// ====================================================================
		{
			RuleCode:    "MARGIN_UTILIZATION_80",
			Code:        "MARGIN_UTILIZATION_80:PASS",
			Description: "Compliant margin utilization (70% <= 80%)",
			Input: OrderContext{
				MarginUtilizationPct: d("0.700000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "MARGIN_UTILIZATION_80",
			Code:        "MARGIN_UTILIZATION_80:BOUNDARY",
			Description: "Boundary margin utilization exactly at 80% limit",
			Input: OrderContext{
				MarginUtilizationPct: d("0.800000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "MARGIN_UTILIZATION_80",
			Code:        "MARGIN_UTILIZATION_80:FAIL",
			Description: "Warning margin utilization exceeding 80% (85% > 80%)",
			Input: OrderContext{
				MarginUtilizationPct: d("0.850000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "margin_utilization_pct"},
		},
		{
			RuleCode:    "MARGIN_UTILIZATION_80",
			Code:        "MARGIN_UTILIZATION_80:ADVERSARIAL",
			Description: "Extreme margin utilization (99% > 80%)",
			Input: OrderContext{
				MarginUtilizationPct: d("0.990000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "margin_utilization_pct"},
		},

		// ====================================================================
		// 54. POST_TRADE_GROUP_ISSUER_20 (Group & Related-Party Issuer 20% Limit)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_GROUP_ISSUER_20",
			Code:        "POST_TRADE_GROUP_ISSUER_20:PASS",
			Description: "Compliant corporate group aggregate exposure (16% <= 20%)",
			Input: OrderContext{
				MaxGroupIssuerExposurePct: d("0.160000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_GROUP_ISSUER_20",
			Code:        "POST_TRADE_GROUP_ISSUER_20:BOUNDARY",
			Description: "Boundary corporate group aggregate exposure exactly at 20% limit",
			Input: OrderContext{
				MaxGroupIssuerExposurePct: d("0.200000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_GROUP_ISSUER_20",
			Code:        "POST_TRADE_GROUP_ISSUER_20:FAIL",
			Description: "Breached corporate group aggregate exposure (23% > 20%)",
			Input: OrderContext{
				MaxGroupIssuerExposurePct: d("0.230000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_group_issuer_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_GROUP_ISSUER_20",
			Code:        "POST_TRADE_GROUP_ISSUER_20:ADVERSARIAL",
			Description: "Complex subsidiary tree aggregation pushing group exposure to 35%",
			Input: OrderContext{
				MaxGroupIssuerExposurePct: d("0.350000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_group_issuer_exposure_pct"},
		},

		// ====================================================================
		// 55. POST_TRADE_ISSUER_DEBT_15 (Single-Issuer Debt 15% Limit)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_ISSUER_DEBT_15",
			Code:        "POST_TRADE_ISSUER_DEBT_15:PASS",
			Description: "Compliant fixed income issuer exposure (12% <= 15%)",
			Input: OrderContext{
				MaxIssuerDebtExposurePct: d("0.120000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ISSUER_DEBT_15",
			Code:        "POST_TRADE_ISSUER_DEBT_15:BOUNDARY",
			Description: "Boundary fixed income issuer exposure exactly at 15% limit",
			Input: OrderContext{
				MaxIssuerDebtExposurePct: d("0.150000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_ISSUER_DEBT_15",
			Code:        "POST_TRADE_ISSUER_DEBT_15:FAIL",
			Description: "Breached single-issuer debt holding (18% > 15%)",
			Input: OrderContext{
				MaxIssuerDebtExposurePct: d("0.180000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_issuer_debt_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_ISSUER_DEBT_15",
			Code:        "POST_TRADE_ISSUER_DEBT_15:ADVERSARIAL",
			Description: "Aggregated multi-tranche corporate bond position reaching 25%",
			Input: OrderContext{
				MaxIssuerDebtExposurePct: d("0.250000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_issuer_debt_exposure_pct"},
		},

		// ====================================================================
		// 56. POST_TRADE_COUNTERPARTY_PFE_10 (OTC Counterparty + PFE 10% Limit)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_COUNTERPARTY_PFE_10",
			Code:        "POST_TRADE_COUNTERPARTY_PFE_10:PASS",
			Description: "Compliant net counterparty exposure with PFE (7.5% <= 10%)",
			Input: OrderContext{
				MaxCounterpartyPfeExposurePct: d("0.075000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_COUNTERPARTY_PFE_10",
			Code:        "POST_TRADE_COUNTERPARTY_PFE_10:BOUNDARY",
			Description: "Boundary counterparty exposure with PFE exactly at 10% limit",
			Input: OrderContext{
				MaxCounterpartyPfeExposurePct: d("0.100000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_COUNTERPARTY_PFE_10",
			Code:        "POST_TRADE_COUNTERPARTY_PFE_10:FAIL",
			Description: "Breached counterparty PFE exposure (12.5% > 10%)",
			Input: OrderContext{
				MaxCounterpartyPfeExposurePct: d("0.125000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_counterparty_pfe_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_COUNTERPARTY_PFE_10",
			Code:        "POST_TRADE_COUNTERPARTY_PFE_10:ADVERSARIAL",
			Description: "Cross-currency swap mark-to-market swing pushing counterparty PFE to 22%",
			Input: OrderContext{
				MaxCounterpartyPfeExposurePct: d("0.220000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_counterparty_pfe_exposure_pct"},
		},

		// ====================================================================
		// 57. POST_TRADE_CASH_MIN_5 (5% Cash & Cash Equivalent Liquidity Floor)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_CASH_MIN_5",
			Code:        "POST_TRADE_CASH_MIN_5:PASS",
			Description: "Compliant portfolio cash buffer (8% >= 5%)",
			Input: OrderContext{
				CashAndEquivalentPct: d("0.080000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_CASH_MIN_5",
			Code:        "POST_TRADE_CASH_MIN_5:BOUNDARY",
			Description: "Boundary cash buffer exactly at 5% liquidity floor",
			Input: OrderContext{
				CashAndEquivalentPct: d("0.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_CASH_MIN_5",
			Code:        "POST_TRADE_CASH_MIN_5:FAIL",
			Description: "Warning cash buffer depletion below 5% floor (3.2% < 5%)",
			Input: OrderContext{
				CashAndEquivalentPct: d("0.032000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "cash_and_equivalent_pct"},
		},
		{
			RuleCode:    "POST_TRADE_CASH_MIN_5",
			Code:        "POST_TRADE_CASH_MIN_5:ADVERSARIAL",
			Description: "Complete liquidity depletion with zero unencumbered cash (0.5% < 5%)",
			Input: OrderContext{
				CashAndEquivalentPct: d("0.005000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "cash_and_equivalent_pct"},
		},
	}
}

// evaluatePhase1Scenario handles evaluation dispatch for Phase 1 Post-Trade rules.
func evaluatePhase1Scenario(sc Scenario) (string, string, bool) {
	in := sc.Input
	switch sc.RuleCode {
	case "UCITS_5_10_40":
		limit := d("0.400000")
		if in.UcitsAggregateAbove5PctExposure.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: ucits_aggregate_above_5pct_exposure %s exceeds max_aggregate_above_5pct_pct %s", sc.RuleCode, in.UcitsAggregateAbove5PctExposure, limit), true
		}
		return "PASSED", "Compliant", true

	case "SEC_144A_QIB_HOLDING":
		limit := d("0.150000")
		if in.Restricted144aExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: restricted_144a_exposure_pct %s exceeds max_144a_non_qib_pct %s", sc.RuleCode, in.Restricted144aExposurePct, limit), true
		}
		return "PASSED", "Compliant", true

	case "MARGIN_UTILIZATION_80":
		limit := d("0.800000")
		if in.MarginUtilizationPct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: margin_utilization_pct %s exceeds max_margin_utilization_pct %s", sc.RuleCode, in.MarginUtilizationPct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_GROUP_ISSUER_20":
		limit := d("0.200000")
		if in.MaxGroupIssuerExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: max_group_issuer_exposure_pct %s exceeds max_group_issuer_pct %s", sc.RuleCode, in.MaxGroupIssuerExposurePct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_ISSUER_DEBT_15":
		limit := d("0.150000")
		if in.MaxIssuerDebtExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: max_issuer_debt_exposure_pct %s exceeds max_issuer_debt_pct %s", sc.RuleCode, in.MaxIssuerDebtExposurePct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_COUNTERPARTY_PFE_10":
		limit := d("0.100000")
		if in.MaxCounterpartyPfeExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: max_counterparty_pfe_exposure_pct %s exceeds max_counterparty_pfe_pct %s", sc.RuleCode, in.MaxCounterpartyPfeExposurePct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_CASH_MIN_5":
		limit := d("0.050000")
		if in.CashAndEquivalentPct.LessThan(limit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: cash_and_equivalent_pct %s below min_cash_pct %s", sc.RuleCode, in.CashAndEquivalentPct, limit), true
		}
		return "PASSED", "Compliant", true

	default:
		return "", "", false
	}
}
