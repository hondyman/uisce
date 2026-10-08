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

		// ====================================================================
		// 58. POST_TRADE_SOVEREIGN_EXPOSURE_35 (Sovereign Debt 35% Limit)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_SOVEREIGN_EXPOSURE_35",
			Code:        "POST_TRADE_SOVEREIGN_EXPOSURE_35:PASS",
			Description: "Compliant single-country sovereign debt holding (28% <= 35%)",
			Input: OrderContext{
				MaxSovereignExposurePct: d("0.280000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SOVEREIGN_EXPOSURE_35",
			Code:        "POST_TRADE_SOVEREIGN_EXPOSURE_35:BOUNDARY",
			Description: "Boundary sovereign debt holding exactly at 35% limit",
			Input: OrderContext{
				MaxSovereignExposurePct: d("0.350000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SOVEREIGN_EXPOSURE_35",
			Code:        "POST_TRADE_SOVEREIGN_EXPOSURE_35:FAIL",
			Description: "Breached sovereign debt concentration (38.5% > 35%)",
			Input: OrderContext{
				MaxSovereignExposurePct: d("0.385000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_sovereign_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_SOVEREIGN_EXPOSURE_35",
			Code:        "POST_TRADE_SOVEREIGN_EXPOSURE_35:ADVERSARIAL",
			Description: "Single emerging market sovereign debt holding reaching 60% of NAV",
			Input: OrderContext{
				MaxSovereignExposurePct: d("0.600000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_sovereign_exposure_pct"},
		},

		// ====================================================================
		// 59. POST_TRADE_AGENCY_SUPRA_25 (Agency & Supranational Debt 25% Limit)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_AGENCY_SUPRA_25",
			Code:        "POST_TRADE_AGENCY_SUPRA_25:PASS",
			Description: "Compliant agency/supranational debt holding (18% <= 25%)",
			Input: OrderContext{
				MaxAgencySupraExposurePct: d("0.180000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_AGENCY_SUPRA_25",
			Code:        "POST_TRADE_AGENCY_SUPRA_25:BOUNDARY",
			Description: "Boundary agency/supranational holding exactly at 25% limit",
			Input: OrderContext{
				MaxAgencySupraExposurePct: d("0.250000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_AGENCY_SUPRA_25",
			Code:        "POST_TRADE_AGENCY_SUPRA_25:FAIL",
			Description: "Breached agency/supranational exposure (28% > 25%)",
			Input: OrderContext{
				MaxAgencySupraExposurePct: d("0.280000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_agency_supra_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_AGENCY_SUPRA_25",
			Code:        "POST_TRADE_AGENCY_SUPRA_25:ADVERSARIAL",
			Description: "Concentrated multilateral development bank holding at 45%",
			Input: OrderContext{
				MaxAgencySupraExposurePct: d("0.450000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_agency_supra_exposure_pct"},
		},

		// ====================================================================
		// 60. POST_TRADE_MUNI_OBLIGOR_10 (Municipal Single-Obligor 10% Limit)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_MUNI_OBLIGOR_10",
			Code:        "POST_TRADE_MUNI_OBLIGOR_10:PASS",
			Description: "Compliant municipal single-obligor exposure (7.5% <= 10%)",
			Input: OrderContext{
				MaxMuniObligorExposurePct: d("0.075000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_MUNI_OBLIGOR_10",
			Code:        "POST_TRADE_MUNI_OBLIGOR_10:BOUNDARY",
			Description: "Boundary municipal obligor exposure exactly at 10% limit",
			Input: OrderContext{
				MaxMuniObligorExposurePct: d("0.100000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_MUNI_OBLIGOR_10",
			Code:        "POST_TRADE_MUNI_OBLIGOR_10:FAIL",
			Description: "Breached municipal obligor exposure (12.5% > 10%)",
			Input: OrderContext{
				MaxMuniObligorExposurePct: d("0.125000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_muni_obligor_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_MUNI_OBLIGOR_10",
			Code:        "POST_TRADE_MUNI_OBLIGOR_10:ADVERSARIAL",
			Description: "Revenue bond project concentration reaching 22% of portfolio NAV",
			Input: OrderContext{
				MaxMuniObligorExposurePct: d("0.220000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_muni_obligor_exposure_pct"},
		},

		// ====================================================================
		// 61. POST_TRADE_CCP_CLEARING_EXPOSURE_15 (CCP Clearing 15% Limit)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_CCP_CLEARING_EXPOSURE_15",
			Code:        "POST_TRADE_CCP_CLEARING_EXPOSURE_15:PASS",
			Description: "Compliant CCP clearing margin exposure (11% <= 15%)",
			Input: OrderContext{
				MaxCcpExposurePct: d("0.110000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_CCP_CLEARING_EXPOSURE_15",
			Code:        "POST_TRADE_CCP_CLEARING_EXPOSURE_15:BOUNDARY",
			Description: "Boundary CCP clearing margin exposure exactly at 15% limit",
			Input: OrderContext{
				MaxCcpExposurePct: d("0.150000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_CCP_CLEARING_EXPOSURE_15",
			Code:        "POST_TRADE_CCP_CLEARING_EXPOSURE_15:FAIL",
			Description: "Breached CCP clearing exposure (18.5% > 15%)",
			Input: OrderContext{
				MaxCcpExposurePct: d("0.185000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_ccp_exposure_pct"},
		},
		{
			RuleCode:    "POST_TRADE_CCP_CLEARING_EXPOSURE_15",
			Code:        "POST_TRADE_CCP_CLEARING_EXPOSURE_15:ADVERSARIAL",
			Description: "Heavy cleared swap margin spike reaching 32% at single CCP",
			Input: OrderContext{
				MaxCcpExposurePct: d("0.320000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_ccp_exposure_pct"},
		},

		// ====================================================================
		// 62. POST_TRADE_CUSTODIAN_CONCENTRATION_20 (Custodian Safekeeping 20% Limit)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_CUSTODIAN_CONCENTRATION_20",
			Code:        "POST_TRADE_CUSTODIAN_CONCENTRATION_20:PASS",
			Description: "Compliant custodian safekeeping concentration (14% <= 20%)",
			Input: OrderContext{
				MaxCustodianConcentrationPct: d("0.140000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_CUSTODIAN_CONCENTRATION_20",
			Code:        "POST_TRADE_CUSTODIAN_CONCENTRATION_20:BOUNDARY",
			Description: "Boundary custodian concentration exactly at 20% limit",
			Input: OrderContext{
				MaxCustodianConcentrationPct: d("0.200000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_CUSTODIAN_CONCENTRATION_20",
			Code:        "POST_TRADE_CUSTODIAN_CONCENTRATION_20:FAIL",
			Description: "Breached custodian concentration (24.5% > 20%)",
			Input: OrderContext{
				MaxCustodianConcentrationPct: d("0.245000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_custodian_concentration_pct"},
		},
		{
			RuleCode:    "POST_TRADE_CUSTODIAN_CONCENTRATION_20",
			Code:        "POST_TRADE_CUSTODIAN_CONCENTRATION_20:ADVERSARIAL",
			Description: "Prime broker unsegregated custody concentration at 42%",
			Input: OrderContext{
				MaxCustodianConcentrationPct: d("0.420000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_custodian_concentration_pct"},
		},

		// ====================================================================
		// 63. POST_TRADE_BANK_DEPOSIT_20 (Single-Bank Cash Deposit 20% Limit)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_BANK_DEPOSIT_20",
			Code:        "POST_TRADE_BANK_DEPOSIT_20:PASS",
			Description: "Compliant single-bank cash deposit (15% <= 20%)",
			Input: OrderContext{
				MaxBankDepositPct: d("0.150000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_BANK_DEPOSIT_20",
			Code:        "POST_TRADE_BANK_DEPOSIT_20:BOUNDARY",
			Description: "Boundary bank deposit exactly at 20% limit",
			Input: OrderContext{
				MaxBankDepositPct: d("0.200000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_BANK_DEPOSIT_20",
			Code:        "POST_TRADE_BANK_DEPOSIT_20:FAIL",
			Description: "Breached single-bank deposit concentration (22.5% > 20%)",
			Input: OrderContext{
				MaxBankDepositPct: d("0.225000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_bank_deposit_pct"},
		},
		{
			RuleCode:    "POST_TRADE_BANK_DEPOSIT_20",
			Code:        "POST_TRADE_BANK_DEPOSIT_20:ADVERSARIAL",
			Description: "Single credit institution cash deposit holding reaching 38%",
			Input: OrderContext{
				MaxBankDepositPct: d("0.380000"),
			},
			Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "max_bank_deposit_pct"},
		},

		// ====================================================================
		// 64. POST_TRADE_SEC_LENDING_COLLATERAL_102 (Sec Lending Collateral Floor 102%)
		// ====================================================================
		{
			RuleCode:    "POST_TRADE_SEC_LENDING_COLLATERAL_102",
			Code:        "POST_TRADE_SEC_LENDING_COLLATERAL_102:PASS",
			Description: "Compliant securities lending collateral coverage (105% >= 102%)",
			Input: OrderContext{
				SecLendingCollateralRatio: d("1.050000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_LENDING_COLLATERAL_102",
			Code:        "POST_TRADE_SEC_LENDING_COLLATERAL_102:BOUNDARY",
			Description: "Boundary collateral coverage exactly at 102% minimum floor",
			Input: OrderContext{
				SecLendingCollateralRatio: d("1.020000"),
			},
			Expected: ExpectedOutcome{Status: "PASSED", MustContain: "Compliant"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_LENDING_COLLATERAL_102",
			Code:        "POST_TRADE_SEC_LENDING_COLLATERAL_102:FAIL",
			Description: "Warning securities lending undercollateralization (98.5% < 102%)",
			Input: OrderContext{
				SecLendingCollateralRatio: d("0.985000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "sec_lending_collateral_ratio"},
		},
		{
			RuleCode:    "POST_TRADE_SEC_LENDING_COLLATERAL_102",
			Code:        "POST_TRADE_SEC_LENDING_COLLATERAL_102:ADVERSARIAL",
			Description: "Severe collateral shortfall during market dislocation (80% < 102%)",
			Input: OrderContext{
				SecLendingCollateralRatio: d("0.800000"),
			},
			Expected: ExpectedOutcome{Status: "WARNING", MustContain: "sec_lending_collateral_ratio"},
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

	case "POST_TRADE_SOVEREIGN_EXPOSURE_35":
		limit := d("0.350000")
		if in.MaxSovereignExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: max_sovereign_exposure_pct %s exceeds max_sovereign_exposure_pct %s", sc.RuleCode, in.MaxSovereignExposurePct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_AGENCY_SUPRA_25":
		limit := d("0.250000")
		if in.MaxAgencySupraExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: max_agency_supra_exposure_pct %s exceeds max_agency_supra_pct %s", sc.RuleCode, in.MaxAgencySupraExposurePct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_MUNI_OBLIGOR_10":
		limit := d("0.100000")
		if in.MaxMuniObligorExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: max_muni_obligor_exposure_pct %s exceeds max_muni_obligor_pct %s", sc.RuleCode, in.MaxMuniObligorExposurePct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_CCP_CLEARING_EXPOSURE_15":
		limit := d("0.150000")
		if in.MaxCcpExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: max_ccp_exposure_pct %s exceeds max_ccp_exposure_pct %s", sc.RuleCode, in.MaxCcpExposurePct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_CUSTODIAN_CONCENTRATION_20":
		limit := d("0.200000")
		if in.MaxCustodianConcentrationPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: max_custodian_concentration_pct %s exceeds max_custodian_pct %s", sc.RuleCode, in.MaxCustodianConcentrationPct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_BANK_DEPOSIT_20":
		limit := d("0.200000")
		if in.MaxBankDepositPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: max_bank_deposit_pct %s exceeds max_bank_deposit_pct %s", sc.RuleCode, in.MaxBankDepositPct, limit), true
		}
		return "PASSED", "Compliant", true

	case "POST_TRADE_SEC_LENDING_COLLATERAL_102":
		limit := d("1.020000")
		if in.SecLendingCollateralRatio.LessThan(limit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: sec_lending_collateral_ratio %s below min_collateral_ratio %s", sc.RuleCode, in.SecLendingCollateralRatio, limit), true
		}
		return "PASSED", "Compliant", true

	default:
		return "", "", false
	}
}
