package engine

import (
	"github.com/shopspring/decimal"
)

// computeCreditAndCounterpartyMetrics calculates credit, fixed income, counterparty, and sovereign exposure metrics.
func computeCreditAndCounterpartyMetrics(state *PortfolioState, metrics map[string]decimal.Decimal) {
	// 1. OTC Derivative Counterparty Net Exposure + PFE
	counterpartyExposureMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		if pos.CounterpartyID != "" {
			totalExp := pos.Weight
			if state.NAV.GreaterThan(decimal.Zero) && pos.PFEAmount.GreaterThan(decimal.Zero) {
				pfeWeight := pos.PFEAmount.Div(state.NAV)
				totalExp = totalExp.Add(pfeWeight)
			}
			counterpartyExposureMap[pos.CounterpartyID] = counterpartyExposureMap[pos.CounterpartyID].Add(totalExp)
		}
	}
	var maxCounterpartyPFE decimal.Decimal
	for _, exp := range counterpartyExposureMap {
		if exp.GreaterThan(maxCounterpartyPFE) {
			maxCounterpartyPFE = exp
		}
	}
	metrics["portfolio.max_counterparty_pfe_exposure_pct"] = maxCounterpartyPFE

	// 2. Sovereign Debt Exposure (grouped by country of risk or issuer)
	sovereignMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		if pos.IssuerType == "SOVEREIGN" || pos.Sector == "SOVEREIGN" || pos.AssetClass == "SOVEREIGN_DEBT" {
			key := pos.CountryOfRisk
			if key == "" {
				key = pos.IssuerID
			}
			sovereignMap[key] = sovereignMap[key].Add(pos.Weight)
		}
	}
	var maxSovereign decimal.Decimal
	for _, exp := range sovereignMap {
		if exp.GreaterThan(maxSovereign) {
			maxSovereign = exp
		}
	}
	metrics["portfolio.max_sovereign_exposure_pct"] = maxSovereign

	// 3. Agency & Supranational Exposure (aggregate across portfolio)
	var agencySupraTotal decimal.Decimal
	for _, pos := range state.Positions {
		if pos.IssuerType == "AGENCY" || pos.IssuerType == "SUPRANATIONAL" || pos.Sector == "AGENCY" || pos.Sector == "SUPRANATIONAL" {
			agencySupraTotal = agencySupraTotal.Add(pos.Weight)
		}
	}
	metrics["portfolio.max_agency_supra_exposure_pct"] = agencySupraTotal

	// 4. Municipal Single-Obligor Exposure
	muniMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		if pos.IssuerType == "MUNICIPAL" || pos.Sector == "MUNICIPAL" || pos.AssetClass == "MUNICIPAL_BOND" {
			muniMap[pos.IssuerID] = muniMap[pos.IssuerID].Add(pos.Weight)
		}
	}
	var maxMuni decimal.Decimal
	for _, exp := range muniMap {
		if exp.GreaterThan(maxMuni) {
			maxMuni = exp
		}
	}
	metrics["portfolio.max_muni_obligor_exposure_pct"] = maxMuni

	// 5. Central Counterparty (CCP) Clearing Exposure
	ccpMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		if pos.CCPID != "" {
			ccpMap[pos.CCPID] = ccpMap[pos.CCPID].Add(pos.Weight)
		}
	}
	var maxCcp decimal.Decimal
	for _, exp := range ccpMap {
		if exp.GreaterThan(maxCcp) {
			maxCcp = exp
		}
	}
	metrics["portfolio.max_ccp_exposure_pct"] = maxCcp

	// 6. Custodian Safekeeping Concentration
	custodianMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		if pos.CustodianID != "" {
			custodianMap[pos.CustodianID] = custodianMap[pos.CustodianID].Add(pos.Weight)
		}
	}
	var maxCustodian decimal.Decimal
	for _, exp := range custodianMap {
		if exp.GreaterThan(maxCustodian) {
			maxCustodian = exp
		}
	}
	metrics["portfolio.max_custodian_concentration_pct"] = maxCustodian

	// 7. Single-Bank Cash Deposit Exposure
	bankMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		if pos.BankID != "" {
			bankMap[pos.BankID] = bankMap[pos.BankID].Add(pos.Weight)
		}
	}
	var maxBank decimal.Decimal
	for _, exp := range bankMap {
		if exp.GreaterThan(maxBank) {
			maxBank = exp
		}
	}
	metrics["portfolio.max_bank_deposit_pct"] = maxBank

	// 8. Securities Lending Collateral Coverage Ratio
	if state.SecLendingTotalLoanValue.GreaterThan(decimal.Zero) {
		ratio := state.SecLendingTotalCollateralValue.Div(state.SecLendingTotalLoanValue)
		metrics["portfolio.sec_lending_collateral_ratio"] = ratio
	} else {
		// Calculate from position level if state totals not set
		var totalLoan, totalCol decimal.Decimal
		for _, pos := range state.Positions {
			totalLoan = totalLoan.Add(pos.SecLendingOnLoanValue)
			totalCol = totalCol.Add(pos.SecLendingCollateralValue)
		}
		if totalLoan.GreaterThan(decimal.Zero) {
			metrics["portfolio.sec_lending_collateral_ratio"] = totalCol.Div(totalLoan)
		} else {
			// No active loans -> 100% compliant sentinel 1.050000
			metrics["portfolio.sec_lending_collateral_ratio"] = decimal.RequireFromString("1.050000")
		}
	}

	// 9. Sanctioned Entity Holding Matches Count
	var sanctionedMatches int64
	for _, pos := range state.Positions {
		if pos.IsSanctioned {
			sanctionedMatches++
		}
	}
	metrics["portfolio.sanctioned_entity_matches_count"] = decimal.NewFromInt(sanctionedMatches)

	// 10. Unhedged Foreign Currency Exposure
	var unhedgedFX decimal.Decimal
	for _, pos := range state.Positions {
		if pos.IsUnhedgedFX {
			unhedgedFX = unhedgedFX.Add(pos.Weight)
		}
	}
	metrics["portfolio.unhedged_fx_exposure_pct"] = unhedgedFX

	// 11. High-Yield & Sub-Investment Grade Debt Exposure
	var highYieldDebt decimal.Decimal
	for _, pos := range state.Positions {
		if pos.IsHighYield {
			highYieldDebt = highYieldDebt.Add(pos.Weight)
		}
	}
	metrics["portfolio.high_yield_debt_exposure_pct"] = highYieldDebt

	// 12. Split Credit Rating Conservative Grade Floor Rank (Worst rank across portfolio, 1=AAA ... 10=BBB- ... 22=D)
	worstRatingRank := 1 // Sentinel: default AAA if no debt holdings
	for _, pos := range state.Positions {
		if pos.ConservativeRatingRank > worstRatingRank {
			worstRatingRank = pos.ConservativeRatingRank
		}
	}
	metrics["portfolio.split_rating_worst_grade_rank"] = decimal.NewFromInt(int64(worstRatingRank))
}
