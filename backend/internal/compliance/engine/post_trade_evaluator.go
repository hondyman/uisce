package engine

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PortfolioPosition represents a security holding within an evaluated portfolio.
type PortfolioPosition struct {
	SecurityID    string          `json:"security_id"`
	Symbol        string          `json:"symbol"`
	IssuerID      string          `json:"issuer_id"`
	IssuerName    string          `json:"issuer_name"`
	MarketValue   decimal.Decimal `json:"market_value"`
	Weight        decimal.Decimal `json:"weight"` // MarketValue / NAV
	AssetClass    string          `json:"asset_class"`
	Sector                string          `json:"sector"`
	IndustryGroup         string          `json:"industry_group,omitempty"`
	CountryOfRisk         string          `json:"country_of_risk"`
	CountryClassification string          `json:"country_classification,omitempty"` // DEVELOPED, EMERGING, FRONTIER
	Is144A                bool            `json:"is_144a"`
	IsQIBEligible         bool            `json:"is_qib_eligible"`
	CreditRating          string          `json:"credit_rating"`
	IssuerType            string          `json:"issuer_type,omitempty"` // SOVEREIGN, AGENCY, SUPRANATIONAL, MUNICIPAL, CORPORATE
	ParentEntityID        string          `json:"parent_entity_id,omitempty"`
	CounterpartyID        string          `json:"counterparty_id,omitempty"`
	CustodianID           string          `json:"custodian_id,omitempty"`
	BankID                string          `json:"bank_id,omitempty"`
	CCPID                 string          `json:"ccp_id,omitempty"`
	PFEAmount             decimal.Decimal `json:"pfe_amount,omitempty"`
	SecLendingOnLoanValue     decimal.Decimal `json:"sec_lending_on_loan_value,omitempty"`
	SecLendingCollateralValue decimal.Decimal `json:"sec_lending_collateral_value,omitempty"`
	IsSanctioned              bool            `json:"is_sanctioned,omitempty"`
	IsUnhedgedFX              bool            `json:"is_unhedged_fx,omitempty"`
	UnhedgedFXAmount          decimal.Decimal `json:"unhedged_fx_amount,omitempty"`
	IsHighYield               bool            `json:"is_high_yield,omitempty"`
	ConservativeRatingRank    int             `json:"conservative_rating_rank,omitempty"`
	IsControversialWeapons    bool            `json:"is_controversial_weapons,omitempty"`
	ThermalCoalRevenuePct     decimal.Decimal `json:"thermal_coal_revenue_pct,omitempty"`
	TobaccoRevenuePct         decimal.Decimal `json:"tobacco_revenue_pct,omitempty"`
	FairValueLevel            int             `json:"fair_value_level,omitempty"` // 1, 2, 3
	IsSettlementFailed        bool            `json:"is_settlement_failed,omitempty"`
	WaciIntensity             decimal.Decimal `json:"waci_intensity,omitempty"`
	HasEmissionsData          bool            `json:"has_emissions_data,omitempty"`
	GhgScope12Intensity       decimal.Decimal `json:"ghg_scope_1_2_intensity,omitempty"`
	BoardGenderDiversityPct   decimal.Decimal `json:"board_gender_diversity_pct,omitempty"`
	HazardousWasteRatio       decimal.Decimal `json:"hazardous_waste_ratio,omitempty"`
	EuTaxonomyAlignmentPct    decimal.Decimal `json:"eu_taxonomy_alignment_pct,omitempty"`
	LiquidityCoverageRatio    decimal.Decimal `json:"liquidity_coverage_ratio,omitempty"`
}

// PortfolioState represents the aggregated point-in-time state of an account's portfolio.
type PortfolioState struct {
	TenantID                        uuid.UUID                  `json:"tenant_id"`
	AccountID                       uuid.UUID                  `json:"account_id"`
	AsOfDate                        time.Time                  `json:"as_of_date"`
	NAV                             decimal.Decimal            `json:"nav"`
	GrossExposure                   decimal.Decimal            `json:"gross_exposure"`
	NetExposure                     decimal.Decimal            `json:"net_exposure"`
	CashBalance                     decimal.Decimal            `json:"cash_balance"`
	MarginLimit                     decimal.Decimal            `json:"margin_limit"`
	SecLendingTotalLoanValue        decimal.Decimal            `json:"sec_lending_total_loan_value,omitempty"`
	SecLendingTotalCollateralValue  decimal.Decimal            `json:"sec_lending_total_collateral_value,omitempty"`
	LiquidityCoverageRatio          decimal.Decimal            `json:"liquidity_coverage_ratio,omitempty"`
	CyclicalSectors                 []string                   `json:"cyclical_sectors,omitempty"`
	ClassificationOverrides         map[string]string          `json:"classification_overrides,omitempty"`
	Positions                       []PortfolioPosition        `json:"positions"`
	Metrics                         map[string]decimal.Decimal `json:"metrics,omitempty"`
}

// PostTradeEvaluationResult represents a compliance decision and generated finding.
type PostTradeEvaluationResult struct {
	FindingID           uuid.UUID              `json:"finding_id"`
	TenantID            uuid.UUID              `json:"tenant_id"`
	RuleID              uuid.UUID              `json:"rule_id"`
	RuleVersion         int                    `json:"rule_version"`
	RuleCode            string                 `json:"rule_code"`
	AccountID           uuid.UUID              `json:"account_id"`
	AsOfDate            time.Time              `json:"as_of_date"`
	EvaluationTier      string                 `json:"evaluation_tier"`
	Status              string                 `json:"status"` // OPEN | SUPERSEDED | RESOLVED | DISMISSED
	Action              string                 `json:"action"` // BREACHED | BREACH_CONFIRMED | WITHIN_LIMITS | WARNING
	FindingSeverity     string                 `json:"finding_severity"`
	SupersedesFindingID *uuid.UUID             `json:"supersedes_finding_id,omitempty"`
	SupersededReason    string                 `json:"superseded_reason,omitempty"`
	LineageHash         string                 `json:"lineage_hash"`
	PortfolioSnapshotID uuid.UUID              `json:"portfolio_snapshot_id"`
	Details             map[string]interface{} `json:"details"`
}

// PostTradeEvaluator orchestrates post-trade portfolio metric calculation, rule evaluation, and finding persistence.
type PostTradeEvaluator struct {
	db *sql.DB
}

// NewPostTradeEvaluator constructs an evaluator backed by Postgres.
func NewPostTradeEvaluator(db *sql.DB) *PostTradeEvaluator {
	return &PostTradeEvaluator{db: db}
}

// ComputePortfolioMetrics calculates aggregate compliance metrics across positions.
func (e *PostTradeEvaluator) ComputePortfolioMetrics(state *PortfolioState) map[string]decimal.Decimal {
	metrics := make(map[string]decimal.Decimal)
	if state.NAV.IsZero() {
		return metrics
	}

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

	// 6. OTC Derivative Counterparty Net Exposure + PFE
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

	// 7. Cash and Cash Equivalent Liquidity Ratio
	if state.NAV.GreaterThan(decimal.Zero) {
		cashRatio := state.CashBalance.Div(state.NAV)
		metrics["portfolio.cash_and_equivalent_pct"] = cashRatio
	} else {
		metrics["portfolio.cash_and_equivalent_pct"] = decimal.Zero
	}

	// 8. Sovereign Debt Exposure (grouped by country of risk or issuer)
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

	// 9. Agency & Supranational Exposure (aggregate across portfolio)
	var agencySupraTotal decimal.Decimal
	for _, pos := range state.Positions {
		if pos.IssuerType == "AGENCY" || pos.IssuerType == "SUPRANATIONAL" || pos.Sector == "AGENCY" || pos.Sector == "SUPRANATIONAL" {
			agencySupraTotal = agencySupraTotal.Add(pos.Weight)
		}
	}
	metrics["portfolio.max_agency_supra_exposure_pct"] = agencySupraTotal

	// 10. Municipal Single-Obligor Exposure
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

	// 11. Central Counterparty (CCP) Clearing Exposure
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

	// 12. Custodian Safekeeping Concentration
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

	// 13. Single-Bank Cash Deposit Exposure
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

	// 14. Securities Lending Collateral Coverage Ratio
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

	// 15. Unclassified / Uncategorized Securities Exposure
	var unclassifiedWeight decimal.Decimal
	for _, pos := range state.Positions {
		if pos.Sector == "" || strings.EqualFold(pos.Sector, "UNCLASSIFIED") || strings.EqualFold(pos.Sector, "UNKNOWN") {
			unclassifiedWeight = unclassifiedWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.unclassified_securities_pct"] = unclassifiedWeight

	// 16. GICS / ICB Sector Concentration (Max Sector Exposure)
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

	// 17. Industry Group Concentration (Max Industry Group Exposure)
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

	// 18. Cyclical Sector Aggregate Exposure (Reference Data / Tenant Overridable)
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

	// 19. Emerging Market Country Exposure
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

	// 20. Non-OECD Country Aggregate Exposure
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

	// 21. Frontier Market Country Sub-Ceiling Exposure
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

	// 22. Sanctioned Entity Holding Matches Count
	var sanctionedMatches int64
	for _, pos := range state.Positions {
		if pos.IsSanctioned {
			sanctionedMatches++
		}
	}
	metrics["portfolio.sanctioned_entity_matches_count"] = decimal.NewFromInt(sanctionedMatches)

	// 23. Unhedged Foreign Currency Exposure
	var unhedgedFX decimal.Decimal
	for _, pos := range state.Positions {
		if pos.IsUnhedgedFX {
			unhedgedFX = unhedgedFX.Add(pos.Weight)
		}
	}
	metrics["portfolio.unhedged_fx_exposure_pct"] = unhedgedFX

	// 24. High-Yield & Sub-Investment Grade Debt Exposure
	var highYieldDebt decimal.Decimal
	for _, pos := range state.Positions {
		if pos.IsHighYield {
			highYieldDebt = highYieldDebt.Add(pos.Weight)
		}
	}
	metrics["portfolio.high_yield_debt_exposure_pct"] = highYieldDebt

	// 25. Split Credit Rating Conservative Grade Floor Rank (Worst rank across portfolio, 1=AAA ... 10=BBB- ... 22=D)
	worstRatingRank := 1 // Sentinel: default AAA if no debt holdings
	for _, pos := range state.Positions {
		if pos.ConservativeRatingRank > worstRatingRank {
			worstRatingRank = pos.ConservativeRatingRank
		}
	}
	metrics["portfolio.split_rating_worst_grade_rank"] = decimal.NewFromInt(int64(worstRatingRank))

	// 26. Controversial Weapons Zero-Tolerance Exposure
	var weaponsWeight decimal.Decimal
	for _, pos := range state.Positions {
		if pos.IsControversialWeapons {
			weaponsWeight = weaponsWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.esg_controversial_weapons_pct"] = weaponsWeight

	// 27. Thermal Coal Revenue Exposure
	var maxCoalRevenue decimal.Decimal
	for _, pos := range state.Positions {
		if pos.ThermalCoalRevenuePct.GreaterThan(maxCoalRevenue) {
			maxCoalRevenue = pos.ThermalCoalRevenuePct
		}
	}
	metrics["portfolio.esg_thermal_coal_revenue_pct"] = maxCoalRevenue

	// 28. Tobacco Revenue Exposure
	var maxTobaccoRevenue decimal.Decimal
	for _, pos := range state.Positions {
		if pos.TobaccoRevenuePct.GreaterThan(maxTobaccoRevenue) {
			maxTobaccoRevenue = pos.TobaccoRevenuePct
		}
	}
	metrics["portfolio.esg_tobacco_revenue_pct"] = maxTobaccoRevenue

	// 29. Level 3 Illiquid Fair-Value Assets Exposure
	var illiquidWeight decimal.Decimal
	for _, pos := range state.Positions {
		if pos.FairValueLevel == 3 {
			illiquidWeight = illiquidWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.illiquid_level3_assets_pct"] = illiquidWeight

	// 30. Settlement Failure Exposure
	var failedSettlementWeight decimal.Decimal
	for _, pos := range state.Positions {
		if pos.IsSettlementFailed {
			failedSettlementWeight = failedSettlementWeight.Add(pos.Weight)
		}
	}
	metrics["portfolio.settlement_fail_exposure_pct"] = failedSettlementWeight

	// 31-35. ESG & Sustainability Metrics (Normalized over Invested Portfolio Assets)
	var totalInvestedMV decimal.Decimal
	for _, pos := range state.Positions {
		totalInvestedMV = totalInvestedMV.Add(pos.MarketValue)
	}

	if totalInvestedMV.GreaterThan(decimal.Zero) {
		var waciSum, coveredEmissionsMV, scope12Sum, diversitySum, wasteSum, taxonomySum decimal.Decimal
		for _, pos := range state.Positions {
			invWeight := pos.MarketValue.Div(totalInvestedMV)
			if pos.HasEmissionsData || pos.WaciIntensity.GreaterThan(decimal.Zero) {
				coveredEmissionsMV = coveredEmissionsMV.Add(pos.MarketValue)
				waciSum = waciSum.Add(invWeight.Mul(pos.WaciIntensity))
			}
			if pos.GhgScope12Intensity.GreaterThan(decimal.Zero) {
				scope12Sum = scope12Sum.Add(invWeight.Mul(pos.GhgScope12Intensity))
			}
			if pos.BoardGenderDiversityPct.GreaterThan(decimal.Zero) {
				diversitySum = diversitySum.Add(invWeight.Mul(pos.BoardGenderDiversityPct))
			}
			if pos.HazardousWasteRatio.GreaterThan(decimal.Zero) {
				wasteSum = wasteSum.Add(invWeight.Mul(pos.HazardousWasteRatio))
			}
			if pos.EuTaxonomyAlignmentPct.GreaterThan(decimal.Zero) {
				taxonomySum = taxonomySum.Add(invWeight.Mul(pos.EuTaxonomyAlignmentPct))
			}
		}
		metrics["portfolio.esg_waci_tco2e_per_m_revenue"] = waciSum
		metrics["portfolio.esg_emissions_data_coverage_pct"] = coveredEmissionsMV.Div(totalInvestedMV)
		metrics["portfolio.esg_ghg_scope_1_2_intensity"] = scope12Sum
		metrics["portfolio.esg_board_gender_diversity_pct"] = diversitySum
		metrics["portfolio.esg_hazardous_waste_ratio"] = wasteSum
		metrics["portfolio.eu_taxonomy_alignment_pct"] = taxonomySum
	} else {
		metrics["portfolio.esg_waci_tco2e_per_m_revenue"] = decimal.Zero
		metrics["portfolio.esg_emissions_data_coverage_pct"] = decimal.RequireFromString("1.000000")
		metrics["portfolio.esg_ghg_scope_1_2_intensity"] = decimal.Zero
		metrics["portfolio.esg_board_gender_diversity_pct"] = decimal.RequireFromString("0.500000")
		metrics["portfolio.esg_hazardous_waste_ratio"] = decimal.Zero
		metrics["portfolio.eu_taxonomy_alignment_pct"] = decimal.RequireFromString("0.500000")
	}

	// 36. Liquidity Coverage Ratio (LCR) Stress Buffer Ratio
	if state.LiquidityCoverageRatio.GreaterThan(decimal.Zero) {
		metrics["portfolio.liquidity_coverage_ratio"] = state.LiquidityCoverageRatio
	} else {
		var lcrSum decimal.Decimal
		for _, pos := range state.Positions {
			if pos.LiquidityCoverageRatio.GreaterThan(decimal.Zero) {
				lcrSum = lcrSum.Add(pos.Weight.Mul(pos.LiquidityCoverageRatio))
			}
		}
		if lcrSum.GreaterThan(decimal.Zero) {
			metrics["portfolio.liquidity_coverage_ratio"] = lcrSum
		} else {
			// Compliant default sentinel (1.200000 = 120%)
			metrics["portfolio.liquidity_coverage_ratio"] = decimal.RequireFromString("1.200000")
		}
	}

	state.Metrics = metrics
	return metrics
}

// ComputeStateContentHash produces a deterministic SHA-256 fingerprint of the portfolio input.
func (e *PostTradeEvaluator) ComputeStateContentHash(state PortfolioState) string {
	raw, _ := json.Marshal(struct {
		TenantID      string              `json:"tenant_id"`
		AccountID     string              `json:"account_id"`
		AsOfDate      string              `json:"as_of_date"`
		NAV           string              `json:"nav"`
		GrossExposure string              `json:"gross_exposure"`
		NetExposure   string              `json:"net_exposure"`
		CashBalance   string              `json:"cash_balance"`
		Positions     []PortfolioPosition `json:"positions"`
	}{
		TenantID:      state.TenantID.String(),
		AccountID:     state.AccountID.String(),
		AsOfDate:      state.AsOfDate.Format("2006-01-02"),
		NAV:           state.NAV.StringFixed(6),
		GrossExposure: state.GrossExposure.StringFixed(6),
		NetExposure:   state.NetExposure.StringFixed(6),
		CashBalance:   state.CashBalance.StringFixed(6),
		Positions:     state.Positions,
	})

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// ComputeDeterministicFindingID generates an RFC 4122 UUIDv5 lineage identifier.
func (e *PostTradeEvaluator) ComputeDeterministicFindingID(
	tenantID uuid.UUID,
	ruleCode string,
	accountID uuid.UUID,
	asOfDate time.Time,
	inputHash string,
) (uuid.UUID, string) {
	lineageSeed := fmt.Sprintf(
		"tenant:%s:rule:%s:account:%s:asOfDate:%s:inputHash:%s",
		tenantID.String(),
		ruleCode,
		accountID.String(),
		asOfDate.Format("2006-01-02"),
		inputHash,
	)
	findingID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(lineageSeed))
	return findingID, lineageSeed
}

// EvaluateAndPersist executes post-trade evaluation against an account's portfolio state.
func (e *PostTradeEvaluator) EvaluateAndPersist(
	ctx context.Context,
	state PortfolioState,
) ([]PostTradeEvaluationResult, error) {
	metrics := e.ComputePortfolioMetrics(&state)
	contentHash := e.ComputeStateContentHash(state)

	tx, err := e.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin tx failed: %w", err)
	}
	defer tx.Rollback()

	// Fence tenant context for RLS
	_, err = tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true)", state.TenantID.String())
	if err != nil {
		return nil, fmt.Errorf("set tenant context failed: %w", err)
	}

	// 1. Upsert Portfolio Snapshot
	var snapshotID uuid.UUID
	positionsJSON, _ := json.Marshal(state.Positions)
	metricsJSON, _ := json.Marshal(metrics)

	err = tx.QueryRowContext(ctx, `
		INSERT INTO compliance.compliance_portfolio_snapshot (
			tenant_id, account_id, as_of_date, nav, gross_exposure, net_exposure,
			cash_balance, positions, metrics, content_hash
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10
		)
		ON CONFLICT (tenant_id, account_id, as_of_date) DO UPDATE SET
			nav = EXCLUDED.nav,
			gross_exposure = EXCLUDED.gross_exposure,
			net_exposure = EXCLUDED.net_exposure,
			cash_balance = EXCLUDED.cash_balance,
			positions = EXCLUDED.positions,
			metrics = EXCLUDED.metrics,
			content_hash = EXCLUDED.content_hash
		RETURNING id
	`, state.TenantID, state.AccountID, state.AsOfDate.Format("2006-01-02"),
		state.NAV.StringFixed(6), state.GrossExposure.StringFixed(6), state.NetExposure.StringFixed(6),
		state.CashBalance.StringFixed(6), positionsJSON, metricsJSON, contentHash,
	).Scan(&snapshotID)
	if err != nil {
		return nil, fmt.Errorf("upsert portfolio snapshot failed: %w", err)
	}

	// 2. Fetch Active Post-Trade Rules for Tenant
	rows, err := tx.QueryContext(ctx, `
		SELECT r.id, r.rule_code, r.current_version, r.severity, r.parameter_thresholds
		FROM compliance.compliance_rule r
		WHERE (r.tenant_id = $1 OR r.tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
		  AND r.rule_phase = 'POST_TRADE'
		  AND r.is_active = true
		  AND r.valid_to IS NULL
		ORDER BY r.priority DESC
	`, state.TenantID)
	if err != nil {
		return nil, fmt.Errorf("query post-trade rules failed: %w", err)
	}
	defer rows.Close()

	type RuleEntry struct {
		ID         uuid.UUID
		Code       string
		Version    int
		Severity   string
		Thresholds map[string]interface{}
	}
	var rules []RuleEntry
	for rows.Next() {
		var re RuleEntry
		var threshBytes []byte
		if err := rows.Scan(&re.ID, &re.Code, &re.Version, &re.Severity, &threshBytes); err != nil {
			return nil, fmt.Errorf("scan rule entry failed: %w", err)
		}
		_ = json.Unmarshal(threshBytes, &re.Thresholds)
		rules = append(rules, re)
	}

	var results []PostTradeEvaluationResult

	// 3. Evaluate Each Post-Trade Rule
	for _, rule := range rules {
		res, evalErr := e.evaluateRule(rule.ID, rule.Code, rule.Version, rule.Severity, rule.Thresholds, state, metrics, snapshotID, contentHash)
		if evalErr != nil {
			return nil, evalErr
		}

		// 4. Handle Finding State Persistence with Deterministic Lineage and Supersession
		err = e.persistFindingLifecycle(ctx, tx, res)
		if err != nil {
			return nil, fmt.Errorf("persist finding lifecycle for %s failed: %w", rule.Code, err)
		}

		results = append(results, *res)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit post-trade evaluation tx failed: %w", err)
	}

	return results, nil
}

// evaluateRule evaluates a single post-trade rule against the computed portfolio metrics.
func (e *PostTradeEvaluator) evaluateRule(
	ruleID uuid.UUID,
	ruleCode string,
	ruleVersion int,
	severity string,
	thresholds map[string]interface{},
	state PortfolioState,
	metrics map[string]decimal.Decimal,
	snapshotID uuid.UUID,
	contentHash string,
) (*PostTradeEvaluationResult, error) {
	findingID, lineageHash := e.ComputeDeterministicFindingID(state.TenantID, ruleCode, state.AccountID, state.AsOfDate, contentHash)

	res := &PostTradeEvaluationResult{
		FindingID:           findingID,
		TenantID:            state.TenantID,
		RuleID:              ruleID,
		RuleVersion:         ruleVersion,
		RuleCode:            ruleCode,
		AccountID:           state.AccountID,
		AsOfDate:            state.AsOfDate,
		EvaluationTier:      "POST_TRADE",
		FindingSeverity:     severity,
		LineageHash:         lineageHash,
		PortfolioSnapshotID: snapshotID,
		Details:             make(map[string]interface{}),
	}

	switch ruleCode {
	case "UCITS_5_10_40":
		maxAggLimit := decimal.RequireFromString("0.400000")
		if v, ok := thresholds["max_aggregate_above_5pct_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxAggLimit = d
			}
		}
		maxSingleLimit := decimal.RequireFromString("0.100000")
		if v, ok := thresholds["max_single_issuer_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxSingleLimit = d
			}
		}

		aggExposure := metrics["portfolio.ucits_aggregate_above_5pct_exposure"]
		maxSingle := metrics["portfolio.max_single_issuer_exposure"]

		res.Details["ucits_aggregate_above_5pct_exposure"] = aggExposure.StringFixed(6)
		res.Details["max_single_issuer_exposure"] = maxSingle.StringFixed(6)
		res.Details["max_aggregate_above_5pct_pct"] = maxAggLimit.StringFixed(6)
		res.Details["max_single_issuer_pct"] = maxSingleLimit.StringFixed(6)

		if maxSingle.GreaterThan(maxSingleLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Single issuer exposure %s exceeds max limit %s", maxSingle.StringFixed(4), maxSingleLimit.StringFixed(4))
		} else if aggExposure.GreaterThan(maxAggLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Aggregate exposure of >5%% issuers (%s) exceeds 40%% limit (%s)", aggExposure.StringFixed(4), maxAggLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "SEC_144A_QIB_HOLDING":
		maxLimit := decimal.RequireFromString("0.150000")
		if v, ok := thresholds["max_144a_non_qib_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}

		exp := metrics["portfolio.restricted_144a_exposure_pct"]
		res.Details["restricted_144a_exposure_pct"] = exp.StringFixed(6)
		res.Details["max_144a_non_qib_pct"] = maxLimit.StringFixed(6)

		if exp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Restricted 144A non-QIB exposure %s exceeds 15%% limit %s", exp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "MARGIN_UTILIZATION_80":
		maxLimit := decimal.RequireFromString("0.800000")
		if v, ok := thresholds["max_margin_utilization_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}

		utilization := metrics["portfolio.margin_utilization_pct"]
		res.Details["margin_utilization_pct"] = utilization.StringFixed(6)
		res.Details["max_margin_utilization_pct"] = maxLimit.StringFixed(6)

		if utilization.GreaterThan(maxLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Margin utilization %s exceeds 80%% warning threshold %s", utilization.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_GROUP_ISSUER_20":
		maxLimit := decimal.RequireFromString("0.200000")
		if v, ok := thresholds["max_group_issuer_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}

		groupExp := metrics["portfolio.max_group_issuer_exposure_pct"]
		res.Details["max_group_issuer_exposure_pct"] = groupExp.StringFixed(6)
		res.Details["max_group_issuer_pct"] = maxLimit.StringFixed(6)

		if groupExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Corporate group aggregate exposure %s exceeds 20%% limit %s", groupExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_ISSUER_DEBT_15":
		maxLimit := decimal.RequireFromString("0.150000")
		if v, ok := thresholds["max_issuer_debt_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}

		debtExp := metrics["portfolio.max_issuer_debt_exposure_pct"]
		res.Details["max_issuer_debt_exposure_pct"] = debtExp.StringFixed(6)
		res.Details["max_issuer_debt_pct"] = maxLimit.StringFixed(6)

		if debtExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Single-issuer debt exposure %s exceeds 15%% limit %s", debtExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_COUNTERPARTY_PFE_10":
		maxLimit := decimal.RequireFromString("0.100000")
		if v, ok := thresholds["max_counterparty_pfe_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}

		cpExp := metrics["portfolio.max_counterparty_pfe_exposure_pct"]
		res.Details["max_counterparty_pfe_exposure_pct"] = cpExp.StringFixed(6)
		res.Details["max_counterparty_pfe_pct"] = maxLimit.StringFixed(6)

		if cpExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("OTC counterparty net + PFE exposure %s exceeds 10%% limit %s", cpExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_CASH_MIN_5":
		minLimit := decimal.RequireFromString("0.050000")
		if v, ok := thresholds["min_cash_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				minLimit = d
			}
		}

		cashExp := metrics["portfolio.cash_and_equivalent_pct"]
		res.Details["cash_and_equivalent_pct"] = cashExp.StringFixed(6)
		res.Details["min_cash_pct"] = minLimit.StringFixed(6)

		if cashExp.LessThan(minLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Cash & cash equivalent ratio %s is below 5%% liquidity floor %s", cashExp.StringFixed(4), minLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_SOVEREIGN_EXPOSURE_35":
		maxLimit := decimal.RequireFromString("0.350000")
		if v, ok := thresholds["max_sovereign_exposure_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		sovExp := metrics["portfolio.max_sovereign_exposure_pct"]
		res.Details["max_sovereign_exposure_pct"] = sovExp.StringFixed(6)
		res.Details["max_sovereign_exposure_pct_threshold"] = maxLimit.StringFixed(6)
		if sovExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Sovereign debt exposure %s exceeds 35%% limit %s", sovExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_AGENCY_SUPRA_25":
		maxLimit := decimal.RequireFromString("0.250000")
		if v, ok := thresholds["max_agency_supra_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		agencyExp := metrics["portfolio.max_agency_supra_exposure_pct"]
		res.Details["max_agency_supra_exposure_pct"] = agencyExp.StringFixed(6)
		res.Details["max_agency_supra_pct"] = maxLimit.StringFixed(6)
		if agencyExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Agency and supranational exposure %s exceeds 25%% limit %s", agencyExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_MUNI_OBLIGOR_10":
		maxLimit := decimal.RequireFromString("0.100000")
		if v, ok := thresholds["max_muni_obligor_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		muniExp := metrics["portfolio.max_muni_obligor_exposure_pct"]
		res.Details["max_muni_obligor_exposure_pct"] = muniExp.StringFixed(6)
		res.Details["max_muni_obligor_pct"] = maxLimit.StringFixed(6)
		if muniExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Municipal single-obligor exposure %s exceeds 10%% limit %s", muniExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_CCP_CLEARING_EXPOSURE_15":
		maxLimit := decimal.RequireFromString("0.150000")
		if v, ok := thresholds["max_ccp_exposure_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		ccpExp := metrics["portfolio.max_ccp_exposure_pct"]
		res.Details["max_ccp_exposure_pct"] = ccpExp.StringFixed(6)
		res.Details["max_ccp_exposure_pct_threshold"] = maxLimit.StringFixed(6)
		if ccpExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Central counterparty clearing exposure %s exceeds 15%% limit %s", ccpExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_CUSTODIAN_CONCENTRATION_20":
		maxLimit := decimal.RequireFromString("0.200000")
		if v, ok := thresholds["max_custodian_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		custExp := metrics["portfolio.max_custodian_concentration_pct"]
		res.Details["max_custodian_concentration_pct"] = custExp.StringFixed(6)
		res.Details["max_custodian_pct"] = maxLimit.StringFixed(6)
		if custExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Custodian safekeeping concentration %s exceeds 20%% limit %s", custExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_BANK_DEPOSIT_20":
		maxLimit := decimal.RequireFromString("0.200000")
		if v, ok := thresholds["max_bank_deposit_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		bankExp := metrics["portfolio.max_bank_deposit_pct"]
		res.Details["max_bank_deposit_pct"] = bankExp.StringFixed(6)
		res.Details["max_bank_deposit_pct_threshold"] = maxLimit.StringFixed(6)
		if bankExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Single-bank cash deposit %s exceeds 20%% limit %s", bankExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_SEC_LENDING_COLLATERAL_102":
		minLimit := decimal.RequireFromString("1.020000")
		if v, ok := thresholds["min_collateral_ratio"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				minLimit = d
			}
		}
		ratio := metrics["portfolio.sec_lending_collateral_ratio"]
		res.Details["sec_lending_collateral_ratio"] = ratio.StringFixed(6)
		res.Details["min_collateral_ratio"] = minLimit.StringFixed(6)
		if ratio.LessThan(minLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Securities lending collateral coverage ratio %s is below 102%% floor %s", ratio.StringFixed(4), minLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_UNCLASSIFIED_CEILING_5":
		maxLimit := decimal.RequireFromString("0.050000")
		if v, ok := thresholds["max_unclassified_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		unclassExp := metrics["portfolio.unclassified_securities_pct"]
		res.Details["unclassified_securities_pct"] = unclassExp.StringFixed(6)
		res.Details["max_unclassified_pct"] = maxLimit.StringFixed(6)
		if unclassExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["finding_category"] = "DATA_QUALITY_INCIDENT"
			res.Details["resolution_path"] = "DATA_REMEDIATION_REQUIRED"
			res.Details["breach_reason"] = fmt.Sprintf("DATA_REMEDIATION_REQUIRED: Unclassified or unjoined security holdings %s exceed 5%% ceiling %s. Update security master reference data.", unclassExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_SECTOR_CONCENTRATION_25":
		maxLimit := decimal.RequireFromString("0.250000")
		if v, ok := thresholds["max_sector_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		secExp := metrics["portfolio.max_sector_exposure_pct"]
		res.Details["max_sector_exposure_pct"] = secExp.StringFixed(6)
		res.Details["max_sector_pct"] = maxLimit.StringFixed(6)
		if secExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("GICS/ICB sector concentration %s exceeds 25%% limit %s", secExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_INDUSTRY_GROUP_15":
		maxLimit := decimal.RequireFromString("0.150000")
		if v, ok := thresholds["max_industry_group_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		indExp := metrics["portfolio.max_industry_group_pct"]
		res.Details["max_industry_group_pct"] = indExp.StringFixed(6)
		res.Details["max_industry_group_pct_threshold"] = maxLimit.StringFixed(6)
		if indExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Industry group concentration %s exceeds 15%% limit %s", indExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_CYCLICAL_SECTOR_35":
		maxLimit := decimal.RequireFromString("0.350000")
		if v, ok := thresholds["max_cyclical_sector_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		cycExp := metrics["portfolio.cyclical_sectors_aggregate_pct"]
		res.Details["cyclical_sectors_aggregate_pct"] = cycExp.StringFixed(6)
		res.Details["max_cyclical_sector_pct"] = maxLimit.StringFixed(6)
		if cycExp.GreaterThan(maxLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Cyclical sector aggregate exposure %s exceeds 35%% warning threshold %s", cycExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_EMERGING_MARKET_20":
		maxLimit := decimal.RequireFromString("0.200000")
		if v, ok := thresholds["max_emerging_market_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		emExp := metrics["portfolio.emerging_markets_pct"]
		res.Details["emerging_markets_pct"] = emExp.StringFixed(6)
		res.Details["max_emerging_market_pct"] = maxLimit.StringFixed(6)
		if emExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Emerging market country exposure %s exceeds 20%% limit %s", emExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_NON_OECD_EXPOSURE_10":
		maxLimit := decimal.RequireFromString("0.100000")
		if v, ok := thresholds["max_non_oecd_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		nonOecdExp := metrics["portfolio.non_oecd_exposure_pct"]
		res.Details["non_oecd_exposure_pct"] = nonOecdExp.StringFixed(6)
		res.Details["max_non_oecd_pct"] = maxLimit.StringFixed(6)
		if nonOecdExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Non-OECD country exposure %s exceeds 10%% ceiling %s", nonOecdExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_FRONTIER_MARKET_5":
		maxLimit := decimal.RequireFromString("0.050000")
		if v, ok := thresholds["max_frontier_market_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		frontierExp := metrics["portfolio.frontier_markets_pct"]
		res.Details["frontier_markets_pct"] = frontierExp.StringFixed(6)
		res.Details["max_frontier_market_pct"] = maxLimit.StringFixed(6)
		if frontierExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Frontier market country sub-ceiling %s exceeds 5%% limit %s", frontierExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_SANCTION_LIST_ZERO_TOLERANCE":
		matches := metrics["portfolio.sanctioned_entity_matches_count"]
		res.Details["sanctioned_entity_matches_count"] = matches.String()
		res.Details["max_sanctioned_matches"] = "0"
		if matches.GreaterThan(decimal.Zero) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Sanctions list zero-tolerance violation: %s sanctioned entity holdings detected", matches.String())
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_FX_FORWARD_UNHEDGED_30":
		maxLimit := decimal.RequireFromString("0.300000")
		if v, ok := thresholds["max_unhedged_fx_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		fxExp := metrics["portfolio.unhedged_fx_exposure_pct"]
		res.Details["unhedged_fx_exposure_pct"] = fxExp.StringFixed(6)
		res.Details["max_unhedged_fx_pct"] = maxLimit.StringFixed(6)
		if fxExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Unhedged FX exposure %s exceeds 30%% limit %s", fxExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_HIGH_YIELD_CEILING_10":
		maxLimit := decimal.RequireFromString("0.100000")
		if v, ok := thresholds["max_high_yield_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		hyExp := metrics["portfolio.high_yield_debt_exposure_pct"]
		res.Details["high_yield_debt_exposure_pct"] = hyExp.StringFixed(6)
		res.Details["max_high_yield_pct"] = maxLimit.StringFixed(6)
		if hyExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("High-yield debt exposure %s exceeds 10%% ceiling %s", hyExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_SPLIT_RATING_CONSERVATIVE_FLOOR":
		maxRank := int64(10) // default BBB-/Baa3 rank (1=AAA ... 10=BBB- ... 22=D)
		if v, ok := thresholds["worst_permissible_rank"].(float64); ok {
			maxRank = int64(v)
		} else if v, ok := thresholds["max_grade_rank"].(float64); ok {
			maxRank = int64(v)
		}
		rank := metrics["portfolio.split_rating_worst_grade_rank"]
		res.Details["split_rating_worst_grade_rank"] = rank.String()
		res.Details["worst_permissible_rank"] = fmt.Sprintf("%d", maxRank)
		if rank.IntPart() > maxRank {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Conservative split credit rating rank %s exceeds maximum permissible grade rank %d (BBB-/Baa3 floor breached; sub-investment grade holding detected)", rank.String(), maxRank)
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_ESG_CONTROVERSIAL_WEAPONS_0":
		weapExp := metrics["portfolio.esg_controversial_weapons_pct"]
		res.Details["esg_controversial_weapons_pct"] = weapExp.StringFixed(6)
		res.Details["max_weapons_exposure_pct"] = "0.000000"
		if weapExp.GreaterThan(decimal.Zero) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Controversial weapons zero-tolerance exclusion breached: %s exposure detected", weapExp.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_ESG_THERMAL_COAL_REVENUE_5":
		maxLimit := decimal.RequireFromString("0.050000")
		if v, ok := thresholds["max_coal_revenue_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		coalExp := metrics["portfolio.esg_thermal_coal_revenue_pct"]
		res.Details["esg_thermal_coal_revenue_pct"] = coalExp.StringFixed(6)
		res.Details["max_coal_revenue_pct"] = maxLimit.StringFixed(6)
		if coalExp.GreaterThan(maxLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Thermal coal revenue exposure %s exceeds 5%% threshold %s", coalExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_ESG_TOBACCO_REVENUE_5":
		maxLimit := decimal.RequireFromString("0.050000")
		if v, ok := thresholds["max_tobacco_revenue_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		tobaccoExp := metrics["portfolio.esg_tobacco_revenue_pct"]
		res.Details["esg_tobacco_revenue_pct"] = tobaccoExp.StringFixed(6)
		res.Details["max_tobacco_revenue_pct"] = maxLimit.StringFixed(6)
		if tobaccoExp.GreaterThan(maxLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Tobacco revenue exposure %s exceeds 5%% threshold %s", tobaccoExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_ILLIQUID_TIER3_ASSETS_10":
		maxLimit := decimal.RequireFromString("0.100000")
		if v, ok := thresholds["max_illiquid_tier3_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		illiquidExp := metrics["portfolio.illiquid_level3_assets_pct"]
		res.Details["illiquid_level3_assets_pct"] = illiquidExp.StringFixed(6)
		res.Details["max_illiquid_tier3_pct"] = maxLimit.StringFixed(6)
		if illiquidExp.GreaterThan(maxLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Level 3 illiquid assets exposure %s exceeds 10%% ceiling %s", illiquidExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_SETTLEMENT_FAIL_CONCENTRATION_5":
		maxLimit := decimal.RequireFromString("0.050000")
		if v, ok := thresholds["max_settlement_fail_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		failExp := metrics["portfolio.settlement_fail_exposure_pct"]
		res.Details["settlement_fail_exposure_pct"] = failExp.StringFixed(6)
		res.Details["max_settlement_fail_pct"] = maxLimit.StringFixed(6)
		if failExp.GreaterThan(maxLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Settlement failure exposure %s exceeds 5%% threshold %s", failExp.StringFixed(4), maxLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_ESG_WACI_PORTFOLIO_CEILING":
		maxWaci := decimal.RequireFromString("150.000000")
		minCov := decimal.RequireFromString("0.750000")
		if v, ok := thresholds["max_waci_tco2e_per_m_revenue"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxWaci = d
			}
		}
		if v, ok := thresholds["min_emissions_data_coverage_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				minCov = d
			}
		}
		waci := metrics["portfolio.esg_waci_tco2e_per_m_revenue"]
		cov := metrics["portfolio.esg_emissions_data_coverage_pct"]
		res.Details["esg_waci_tco2e_per_m_revenue"] = waci.StringFixed(2)
		res.Details["max_waci_tco2e_per_m_revenue"] = maxWaci.StringFixed(2)
		res.Details["esg_emissions_data_coverage_pct"] = cov.StringFixed(4)
		res.Details["min_emissions_data_coverage_pct"] = minCov.StringFixed(4)

		if cov.LessThan(minCov) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Emissions data coverage %s is below mandatory threshold %s (understates portfolio WACI)", cov.StringFixed(4), minCov.StringFixed(4))
		} else if waci.GreaterThan(maxWaci) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("WACI %s tCO2e/M$ exceeds portfolio carbon ceiling %s tCO2e/M$", waci.StringFixed(2), maxWaci.StringFixed(2))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_ESG_SCOPE_1_2_EMISSIONS_CEILING":
		maxLimit := decimal.RequireFromString("100.000000")
		if v, ok := thresholds["max_ghg_scope_1_2_intensity"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		scope12 := metrics["portfolio.esg_ghg_scope_1_2_intensity"]
		res.Details["esg_ghg_scope_1_2_intensity"] = scope12.StringFixed(2)
		res.Details["max_ghg_scope_1_2_intensity"] = maxLimit.StringFixed(2)
		if scope12.GreaterThan(maxLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Scope 1+2 emissions intensity %s tCO2e/M$ exceeds ceiling %s tCO2e/M$", scope12.StringFixed(2), maxLimit.StringFixed(2))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_ESG_BOARD_GENDER_DIVERSITY_FLOOR":
		minLimit := decimal.RequireFromString("0.300000")
		if v, ok := thresholds["min_board_gender_diversity_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				minLimit = d
			}
		}
		div := metrics["portfolio.esg_board_gender_diversity_pct"]
		res.Details["esg_board_gender_diversity_pct"] = div.StringFixed(4)
		res.Details["min_board_gender_diversity_pct"] = minLimit.StringFixed(4)
		if div.LessThan(minLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Board female representation %s is below minimum diversity floor %s (SFDR PAI 13)", div.StringFixed(4), minLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_ESG_HAZARDOUS_WASTE_RATIO_CEILING":
		maxLimit := decimal.RequireFromString("5.000000")
		if v, ok := thresholds["max_hazardous_waste_ratio"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				maxLimit = d
			}
		}
		waste := metrics["portfolio.esg_hazardous_waste_ratio"]
		res.Details["esg_hazardous_waste_ratio"] = waste.StringFixed(2)
		res.Details["max_hazardous_waste_ratio"] = maxLimit.StringFixed(2)
		if waste.GreaterThan(maxLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Hazardous waste ratio %s tonnes/M$ exceeds ceiling %s tonnes/M$ (SFDR PAI 9)", waste.StringFixed(2), maxLimit.StringFixed(2))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_EU_TAXONOMY_GREEN_REVENUE_FLOOR":
		minLimit := decimal.RequireFromString("0.150000")
		if v, ok := thresholds["min_taxonomy_alignment_pct"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				minLimit = d
			}
		}
		tax := metrics["portfolio.eu_taxonomy_alignment_pct"]
		res.Details["eu_taxonomy_alignment_pct"] = tax.StringFixed(4)
		res.Details["min_taxonomy_alignment_pct"] = minLimit.StringFixed(4)
		if tax.LessThan(minLimit) {
			res.Action = "WARNING"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("EU Taxonomy green revenue alignment %s is below minimum floor %s", tax.StringFixed(4), minLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	case "POST_TRADE_LIQUIDITY_COVERAGE_RATIO_BUFFER":
		minLimit := decimal.RequireFromString("1.050000")
		if v, ok := thresholds["min_lcr_buffer_ratio"].(string); ok {
			if d, err := decimal.NewFromString(v); err == nil {
				minLimit = d
			}
		}
		lcr := metrics["portfolio.liquidity_coverage_ratio"]
		res.Details["liquidity_coverage_ratio"] = lcr.StringFixed(4)
		res.Details["min_lcr_buffer_ratio"] = minLimit.StringFixed(4)
		if lcr.LessThan(minLimit) {
			res.Action = "BREACHED"
			res.Status = "OPEN"
			res.Details["breach_reason"] = fmt.Sprintf("Stress Liquidity Coverage Ratio %s is below mandatory buffer %s (Basel III / UCITS)", lcr.StringFixed(4), minLimit.StringFixed(4))
		} else {
			res.Action = "WITHIN_LIMITS"
			res.Status = "RESOLVED"
		}

	default:
		res.Action = "WITHIN_LIMITS"
		res.Status = "RESOLVED"
	}

	return res, nil
}

// persistFindingLifecycle manages the deterministic insertion, supersession, or resolution of compliance findings.
func (e *PostTradeEvaluator) persistFindingLifecycle(
	ctx context.Context,
	tx *sql.Tx,
	res *PostTradeEvaluationResult,
) error {
	// Look for existing OPEN finding for this rule and account on as_of_date
	var existingID uuid.UUID
	var existingLineage string
	var existingSeverity, existingAction string

	err := tx.QueryRowContext(ctx, `
		SELECT id, lineage_hash, finding_severity, action
		FROM compliance.compliance_finding
		WHERE tenant_id = $1 AND rule_code = $2 AND account_id = $3 AND as_of_date = $4 AND status = 'OPEN'
		FOR UPDATE
	`, res.TenantID, res.RuleCode, res.AccountID, res.AsOfDate.Format("2006-01-02")).Scan(
		&existingID, &existingLineage, &existingSeverity, &existingAction,
	)

	hasExistingOpen := (err == nil)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("query existing open finding failed: %w", err)
	}

	detailsJSON, _ := json.Marshal(res.Details)

	if res.Action == "BREACHED" || res.Action == "WARNING" {
		// Rule is currently in breach / warning
		if hasExistingOpen {
			if existingID == res.FindingID {
				// Exact same input and lineage -> idempotent no-op
				return nil
			}

			// Restatement detected: Old finding is superseded by new calculation
			_, err = tx.ExecContext(ctx, `
				UPDATE compliance.compliance_finding
				SET status = 'SUPERSEDED',
					superseded_reason = 'SUPERSEDED_BY_RESTATEMENT',
					updated_at = now()
				WHERE id = $1
			`, existingID)
			if err != nil {
				return fmt.Errorf("supersede existing finding failed: %w", err)
			}

			res.SupersedesFindingID = &existingID
			res.SupersededReason = "SUPERSEDED_BY_RESTATEMENT"
		}

		// Insert the new finding
		_, err = tx.ExecContext(ctx, `
			INSERT INTO compliance.compliance_finding (
				id, tenant_id, rule_id, rule_version, rule_code, account_id,
				as_of_date, evaluation_tier, status, action, finding_severity,
				supersedes_finding_id, superseded_reason, lineage_hash,
				portfolio_snapshot_id, details
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, 'OPEN', $9, $10,
				$11, $12, $13,
				$14, $15::jsonb
			)
			ON CONFLICT (id) DO UPDATE SET
				details = EXCLUDED.details,
				updated_at = now()
		`, res.FindingID, res.TenantID, res.RuleID, res.RuleVersion, res.RuleCode, res.AccountID,
			res.AsOfDate.Format("2006-01-02"), res.EvaluationTier, res.Action, res.FindingSeverity,
			res.SupersedesFindingID, res.SupersededReason, res.LineageHash,
			res.PortfolioSnapshotID, detailsJSON,
		)
		if err != nil {
			return fmt.Errorf("insert new compliance finding failed: %w", err)
		}
	} else {
		// Rule is WITHIN_LIMITS
		if hasExistingOpen {
			// Portfolio restatement resolved prior breach
			_, err = tx.ExecContext(ctx, `
				UPDATE compliance.compliance_finding
				SET status = 'RESOLVED',
					resolution_notes = 'Remediated/resolved by portfolio restatement',
					resolved_by = 'system_post_trade_evaluator',
					resolved_at = now(),
					updated_at = now()
				WHERE id = $1
			`, existingID)
			if err != nil {
				return fmt.Errorf("resolve existing finding failed: %w", err)
			}
		}
	}

	return nil
}
