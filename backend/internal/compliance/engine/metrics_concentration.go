package engine

import (
	"strings"

	"github.com/shopspring/decimal"
)

// computeConcentrationMetrics calculates concentration and exposure metrics across portfolio positions.
func computeConcentrationMetrics(state *PortfolioState, metrics map[string]decimal.Decimal) {
	// 1. Issuer aggregations for UCITS 5/10/40
	issuerExposureMap := make(map[string]decimal.Decimal)
	for i := range state.Positions {
		pos := &state.Positions[i]
		if state.NAV.GreaterThan(decimal.Zero) {
			pos.Weight = pos.MarketValue.Div(state.NAV)
		}
		issuerExposureMap[pos.IssuerID] = issuerExposureMap[pos.IssuerID].Add(pos.Weight)
	}

	var maxSingleIssuer decimal.Decimal
	var ucitsAggregateAbove5Pct decimal.Decimal
	fivePct := decimal.RequireFromString("0.050000")

	for _, exp := range issuerExposureMap {
		if exp.GreaterThan(maxSingleIssuer) {
			maxSingleIssuer = exp
		}
		if exp.GreaterThan(fivePct) {
			ucitsAggregateAbove5Pct = ucitsAggregateAbove5Pct.Add(exp)
		}
	}

	metrics["portfolio.max_single_issuer_exposure"] = maxSingleIssuer
	metrics["portfolio.ucits_aggregate_above_5pct_exposure"] = ucitsAggregateAbove5Pct

	// 2. SEC 144A / QIB Illiquid Asset Exposure
	var restricted144aExposure decimal.Decimal
	for _, pos := range state.Positions {
		if pos.Is144A && !pos.IsQIBEligible {
			restricted144aExposure = restricted144aExposure.Add(pos.Weight)
		}
	}
	metrics["portfolio.restricted_144a_exposure_pct"] = restricted144aExposure

	// 3. Margin Utilization
	if state.MarginLimit.GreaterThan(decimal.Zero) {
		borrowed := state.GrossExposure.Sub(state.CashBalance)
		if borrowed.IsNegative() {
			borrowed = decimal.Zero
		}
		marginUtilization := borrowed.Div(state.MarginLimit)
		metrics["portfolio.margin_utilization_pct"] = marginUtilization
	} else {
		metrics["portfolio.margin_utilization_pct"] = decimal.Zero
	}

	// 4. Corporate Group / Related-Party Issuer Exposure
	groupExposureMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		groupID := pos.ParentEntityID
		if groupID == "" {
			groupID = pos.IssuerID
		}
		if groupID != "" {
			groupExposureMap[groupID] = groupExposureMap[groupID].Add(pos.Weight)
		}
	}
	var maxGroupExposure decimal.Decimal
	for _, exp := range groupExposureMap {
		if exp.GreaterThan(maxGroupExposure) {
			maxGroupExposure = exp
		}
	}
	metrics["portfolio.max_group_issuer_exposure_pct"] = maxGroupExposure

	// 5. Single-Issuer Debt / Fixed Income Exposure
	debtExposureMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		if pos.AssetClass == "FIXED_INCOME" || pos.AssetClass == "DEBT" || pos.AssetClass == "BOND" {
			debtExposureMap[pos.IssuerID] = debtExposureMap[pos.IssuerID].Add(pos.Weight)
		}
	}
	var maxDebtExposure decimal.Decimal
	for _, exp := range debtExposureMap {
		if exp.GreaterThan(maxDebtExposure) {
			maxDebtExposure = exp
		}
	}
	metrics["portfolio.max_issuer_debt_exposure_pct"] = maxDebtExposure

	// 6. Unclassified / Uncategorized Securities Exposure
	var unclassifiedWeight decimal.Decimal
	for _, pos := range state.Positions {
		if pos.Sector == "" || strings.EqualFold(pos.Sector, "UNCLASSIFIED") || strings.EqualFold(pos.Sector, "UNKNOWN") {
			unclassifiedWeight = unclassifiedWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.unclassified_securities_pct"] = unclassifiedWeight

	// 7. GICS / ICB Sector Concentration (Max Sector Exposure)
	sectorMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		if pos.Sector != "" && !strings.EqualFold(pos.Sector, "UNCLASSIFIED") && !strings.EqualFold(pos.Sector, "UNKNOWN") {
			sectorMap[pos.Sector] = sectorMap[pos.Sector].Add(pos.Weight)
		}
	}
	var maxSector decimal.Decimal
	for _, exp := range sectorMap {
		if exp.GreaterThan(maxSector) {
			maxSector = exp
		}
	}
	metrics["portfolio.max_sector_exposure_pct"] = maxSector

	// 8. Industry Group Concentration (Max Industry Group Exposure)
	industryMap := make(map[string]decimal.Decimal)
	for _, pos := range state.Positions {
		ind := pos.IndustryGroup
		if ind == "" {
			ind = pos.Sector
		}
		if ind != "" && !strings.EqualFold(ind, "UNCLASSIFIED") && !strings.EqualFold(ind, "UNKNOWN") {
			industryMap[ind] = industryMap[ind].Add(pos.Weight)
		}
	}
	var maxIndustry decimal.Decimal
	for _, exp := range industryMap {
		if exp.GreaterThan(maxIndustry) {
			maxIndustry = exp
		}
	}
	metrics["portfolio.max_industry_group_pct"] = maxIndustry

	// 9. Cyclical Sector Aggregate Exposure (Reference Data / Tenant Overridable)
	var cyclicalWeight decimal.Decimal
	cyclicalSectors := make(map[string]bool)
	if len(state.CyclicalSectors) > 0 {
		for _, s := range state.CyclicalSectors {
			cyclicalSectors[strings.ToUpper(strings.TrimSpace(s))] = true
		}
	} else {
		for _, s := range []string{"ENERGY", "MATERIALS", "INDUSTRIALS", "CONSUMER DISCRETIONARY", "FINANCIALS", "REAL ESTATE"} {
			cyclicalSectors[s] = true
		}
	}
	for _, pos := range state.Positions {
		sector := pos.Sector
		if ovr, ok := state.ClassificationOverrides[pos.SecurityID]; ok {
			sector = ovr
		}
		if cyclicalSectors[strings.ToUpper(sector)] {
			cyclicalWeight = cyclicalWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.cyclical_sectors_aggregate_pct"] = cyclicalWeight

	// 10. Emerging Market Country Exposure
	emergingCountries := map[string]bool{
		"BR": true, "CN": true, "IN": true, "MX": true, "ZA": true, "KR": true, "TW": true, "SA": true, "ID": true, "TR": true, "PL": true, "TH": true, "MY": true, "CL": true, "CO": true, "EG": true, "PH": true, "HU": true, "CZ": true, "GR": true, "QA": true, "AE": true, "KW": true,
	}
	var emergingWeight decimal.Decimal
	for _, pos := range state.Positions {
		if emergingCountries[strings.ToUpper(pos.CountryOfRisk)] || strings.EqualFold(pos.CountryClassification, "EMERGING") {
			emergingWeight = emergingWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.emerging_markets_pct"] = emergingWeight

	// 11. Non-OECD Country Aggregate Exposure
	oecdCountries := map[string]bool{
		"US": true, "GB": true, "DE": true, "FR": true, "JP": true, "CA": true, "AU": true, "IT": true, "ES": true, "NL": true, "CH": true, "SE": true, "NO": true, "DK": true, "FI": true, "BE": true, "AT": true, "IE": true, "NZ": true, "PT": true, "LU": true, "IL": true, "KR": true, "MX": true, "CL": true, "CO": true, "CR": true, "CZ": true, "EE": true, "GR": true, "HU": true, "IS": true, "LV": true, "LT": true, "PL": true, "SK": true, "SI": true, "TR": true,
	}
	var nonOECDWeight decimal.Decimal
	for _, pos := range state.Positions {
		c := strings.ToUpper(pos.CountryOfRisk)
		if c != "" && !oecdCountries[c] {
			nonOECDWeight = nonOECDWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.non_oecd_exposure_pct"] = nonOECDWeight

	// 12. Frontier Market Country Sub-Ceiling Exposure
	frontierCountries := map[string]bool{
		"VN": true, "NG": true, "KE": true, "BD": true, "PK": true, "RO": true, "KZ": true, "MA": true, "HR": true, "RS": true, "SI": true, "IS": true, "BH": true, "OM": true, "JO": true, "TN": true, "MU": true,
	}
	var frontierWeight decimal.Decimal
	for _, pos := range state.Positions {
		if frontierCountries[strings.ToUpper(pos.CountryOfRisk)] || strings.EqualFold(pos.CountryClassification, "FRONTIER") {
			frontierWeight = frontierWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.frontier_markets_pct"] = frontierWeight
}
