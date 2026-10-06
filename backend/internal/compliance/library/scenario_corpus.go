package library

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Scenario is a single deterministic evaluation case against a Core rule.
type Scenario struct {
	RuleCode    string
	Code        string // scenario ID: "<rule>:<case>"
	Description string
	Input       OrderContext
	Expected    ExpectedOutcome
}

// ExpectedOutcome represents the expected decision and explainability text.
type ExpectedOutcome struct {
	Status      string // PASSED | BLOCKED | WARNING | APPROVAL_REQUIRED
	MustContain string // substring the explanation must contain (explainability assertion)
}

// OrderContext represents the deterministic inputs evaluated against compliance rules.
type OrderContext struct {
	TenantID   uuid.UUID
	AccountID  uuid.UUID
	SecurityID uuid.UUID
	OrderID    uuid.UUID
	IssuerID   uuid.UUID
	EmployeeID *uuid.UUID

	Side       string
	Quantity   decimal.Decimal
	Price      decimal.Decimal
	OrderType  string
	LimitPrice decimal.Decimal
	Notional   decimal.Decimal

	// Exposure & Valuation Snapshots
	IssuerExposurePct     decimal.Decimal
	DaysOver5Pct          int
	TargetFundExposurePct decimal.Decimal
	DepositExposurePct    decimal.Decimal
	CountryExposurePct    decimal.Decimal
	SectorExposurePct     decimal.Decimal
	NavWeightPct                    decimal.Decimal
	CompliantAssetsPct              decimal.Decimal
	UcitsAggregateAbove5PctExposure decimal.Decimal
	Restricted144aExposurePct       decimal.Decimal
	MaxGroupIssuerExposurePct       decimal.Decimal
	MaxIssuerDebtExposurePct        decimal.Decimal
	MaxCounterpartyPfeExposurePct   decimal.Decimal
	CashAndEquivalentPct            decimal.Decimal
	MaxSovereignExposurePct         decimal.Decimal
	MaxAgencySupraExposurePct       decimal.Decimal
	MaxMuniObligorExposurePct       decimal.Decimal
	MaxCcpExposurePct               decimal.Decimal
	MaxCustodianConcentrationPct    decimal.Decimal
	MaxBankDepositPct               decimal.Decimal
	SecLendingCollateralRatio       decimal.Decimal
	UnclassifiedSecuritiesPct       decimal.Decimal
	MaxSectorExposurePct            decimal.Decimal
	MaxIndustryGroupPct             decimal.Decimal
	CyclicalSectorsAggregatePct     decimal.Decimal
	EmergingMarketsPct              decimal.Decimal
	NonOecdExposurePct              decimal.Decimal
	FrontierMarketsPct              decimal.Decimal
	SanctionedEntityMatchesCount    int
	UnhedgedFxExposurePct           decimal.Decimal
	HighYieldDebtExposurePct        decimal.Decimal
	SplitRatingWorstGradeRank       int
	SplitRatingConservativeGradeRank int
	EsgControversialWeaponsPct      decimal.Decimal
	EsgThermalCoalRevenuePct        decimal.Decimal
	EsgTobaccoRevenuePct            decimal.Decimal
	IlliquidLevel3AssetsPct         decimal.Decimal
	SettlementFailExposurePct       decimal.Decimal
	EsgWaciTco2ePerMRevenue         decimal.Decimal
	EsgEmissionsDataCoveragePct     decimal.Decimal
	EsgGhgScope12Intensity          decimal.Decimal
	EsgBoardGenderDiversityPct      decimal.Decimal
	EsgHazardousWasteRatio          decimal.Decimal
	EuTaxonomyAlignmentPct          decimal.Decimal
	LiquidityCoverageRatio          decimal.Decimal
	Nav                             decimal.Decimal
	ExistingPositionValue decimal.Decimal
	ReferencePrice        decimal.Decimal
	PriceDeviationPct     decimal.Decimal
	AdvRatio              decimal.Decimal
	DuplicateCount        int
	OrdersPerMinute       int

	// List & Eligibility Flags
	InRestrictedList          bool
	InInsiderList             bool
	IssuerIsSanctioned        bool
	Restricted144A            bool
	QIBStatus                 bool
	RegS                      bool
	Jurisdiction              string
	DomesticJurisdictions     []string
	DistributionEligible      bool
	PreclearanceValid         bool
	BlackoutActive            bool
	DaysSincePurchase         int
	EasyToBorrow              bool
	LocateValid               bool
	ShortSaleRestricted       bool
	CoveredOffering           bool
	DaysBeforePricing         int
	RealizedLoss30d           bool
	SameBeneficialOwner       bool
	SettledQty                decimal.Decimal
	UnpaidCash                bool
	PendingOpposingOrder      bool
	CanBeAggregated           bool
	ParentIsBlock             bool
	ApprovedVenues            []string
	Venue                     string
	OutsideSpread             bool
	VarLeverageRatio          decimal.Decimal
	CommitmentRatio           decimal.Decimal
	IlliquidAssetsPct         decimal.Decimal
	DaysToLiquidate50Pct      int
	MarginUtilizationPct      decimal.Decimal
	RatioDeviation            decimal.Decimal
	PriceDispersion           decimal.Decimal
	StatementDiscrepancy      bool
	MissingFieldsCount        int
	LEIInvalid                bool
	MinutesSinceExecution     int
	FailAgeDays               int
	CounterpartyExposure      decimal.Decimal
	EMIRValid                 bool
	Currency                  string
	AccountBaseCurrency       string
	SecurityCurrency          string
	IsPrincipalOrEmployee     bool
	PendingClientOrdersExist  bool
	CancelRate5Min            decimal.Decimal
	OrderToTradeRatio         decimal.Decimal
	ArrivalPriceDeviationPct  decimal.Decimal
	LowerBand                 decimal.Decimal
	UpperBand                 decimal.Decimal
	RoundLot                  decimal.Decimal
	PennyStock                bool
}

// Helper to construct decimals
func d(s string) decimal.Decimal {
	v, err := decimal.NewFromString(s)
	if err != nil {
		panic(fmt.Sprintf("invalid decimal: %s", s))
	}
	return v
}

var (
	testTenantID = uuid.MustParse("00000000-0000-4000-a000-000000000000")
	testSecID    = uuid.MustParse("11111111-1111-4111-a111-111111111111")
	testEmpID    = uuid.MustParse("22222222-2222-4222-a222-222222222222")
)

// CoreScenarioCorpus contains deterministic scenario tests for all Core Gold-Copy and Post-Trade rules.
var CoreScenarioCorpus = buildScenarioCorpus()

func buildScenarioCorpus() []Scenario {
	var corpus []Scenario
	corpus = append(corpus, getCorePreTradeScenarios()...)
	corpus = append(corpus, getPhase1PostTradeScenarios()...)
	corpus = append(corpus, getPhase2PostTradeScenarios()...)
	return corpus
}

// EvaluateScenario evaluates an OrderContext against a rule AST condition and parameter thresholds.
func EvaluateScenario(sc Scenario) (string, string) {
	if status, explain, ok := evaluatePhase1Scenario(sc); ok {
		return status, explain
	}
	if status, explain, ok := evaluatePhase2Scenario(sc); ok {
		return status, explain
	}
	in := sc.Input
	switch sc.RuleCode {
	case "UCITS_ISSUER_5":
		limit := d("0.050000")
		if in.IssuerExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: issuer_exposure_pct %s exceeds limit %s", sc.RuleCode, in.IssuerExposurePct, limit)
		}
		return "PASSED", "Compliant"

	case "UCITS_ISSUER_10_EXCEPTION":
		limit10 := d("0.100000")
		limit5 := d("0.050000")
		maxDays := 180
		if in.IssuerExposurePct.GreaterThan(limit10) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: issuer_exposure_pct %s exceeds iss10_limit_pct %s", sc.RuleCode, in.IssuerExposurePct, limit10)
		}
		if in.IssuerExposurePct.GreaterThan(limit5) && in.DaysOver5Pct > maxDays {
			return "APPROVAL_REQUIRED", fmt.Sprintf("Rule %s requires approval: issuer_exposure_pct %s exceeds 5%% and grandfather_days %d expired", sc.RuleCode, in.IssuerExposurePct, maxDays)
		}
		return "PASSED", "Compliant"

	case "UCITS_ISSUER_40":
		limit := d("0.400000")
		if in.IssuerExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: issuer_exposure_pct %s exceeds iss40_limit_pct %s", sc.RuleCode, in.IssuerExposurePct, limit)
		}
		return "PASSED", "Compliant"

	case "UCITS_DEPOSIT_20":
		limit := d("0.200000")
		if in.DepositExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: deposit_exposure_pct %s exceeds deposit_limit_pct %s", sc.RuleCode, in.DepositExposurePct, limit)
		}
		return "PASSED", "Compliant"

	case "ACT40_DIV_75_5":
		divLimit := d("0.750000")
		concLimit := d("0.050000")
		if in.CompliantAssetsPct.LessThan(divLimit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: compliant_assets_pct %s below %s", sc.RuleCode, in.CompliantAssetsPct, divLimit)
		}
		if in.IssuerExposurePct.GreaterThan(concLimit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: issuer_exposure_pct %s exceeds issuer_limit_pct %s", sc.RuleCode, in.IssuerExposurePct, concLimit)
		}
		return "PASSED", "Compliant"

	case "FOF_20":
		limit := d("0.200000")
		if in.TargetFundExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: target_fund_exposure_pct %s exceeds fof_limit_pct %s", sc.RuleCode, in.TargetFundExposurePct, limit)
		}
		return "PASSED", "Compliant"

	case "SEC_144A_ELIGIBILITY":
		if in.Restricted144A && !in.QIBStatus {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: restricted_144a asset requires qib_status", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "REG_S_OFFSHORE_ONLY":
		if in.RegS {
			for _, dom := range in.DomesticJurisdictions {
				if strings.EqualFold(in.Jurisdiction, dom) {
					return "BLOCKED", fmt.Sprintf("Rule %s breached: reg_s asset cannot be offered in domestic_jurisdictions %s", sc.RuleCode, in.Jurisdiction)
				}
			}
		}
		return "PASSED", "Compliant"

	case "RESTRICTED_LIST_BLOCK":
		if in.InRestrictedList {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: security is in restricted_list", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "INSIDER_LIST_MAR":
		if in.InInsiderList {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: security is in insider_list", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "SANCTIONS_ISSUER_BLOCK":
		if in.IssuerIsSanctioned {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: issuer is sanctioned", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "COUNTRY_EXPOSURE_LIMIT":
		limit := d("0.150000")
		if in.CountryExposurePct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: country_exposure_pct %s exceeds country_limit_pct %s", sc.RuleCode, in.CountryExposurePct, limit)
		}
		return "PASSED", "Compliant"

	case "REG_M_RULE_105":
		window := 5
		if in.Side == "SHORT" && in.CoveredOffering && in.DaysBeforePricing <= window {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: short sale within restricted_window_days %d of offering", sc.RuleCode, window)
		}
		return "PASSED", "Compliant"

	case "WASH_SALE_1091":
		if in.Side == "BUY" && in.RealizedLoss30d && in.SameBeneficialOwner {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: buy order with realized_loss_30d for same beneficial owner", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "SELF_TRADE_PREVENT":
		if in.PendingOpposingOrder {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: matching pending_opposing_order on venue", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "PT_PRECLEARANCE_REQUIRED":
		if in.EmployeeID != nil && !in.PreclearanceValid {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: employee trade without preclearance_valid", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "FAT_FINGER_NOTIONAL":
		limit := d("50000000.00")
		if in.Notional.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: notional %s exceeds max_notional %s", sc.RuleCode, in.Notional, limit)
		}
		return "PASSED", "Compliant"

	case "AGGREGATION_ELIGIBILITY":
		if !in.CanBeAggregated && in.ParentIsBlock {
			return "WARNING", fmt.Sprintf("Rule %s warning: order cannot be aggregated (can_be_aggregated=false) into block parent", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "COUNTERPARTY_OTC_LIMIT":
		limit := d("25000000.000000")
		if in.CounterpartyExposure.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: counterparty_exposure %s exceeds counterparty_limit %s", sc.RuleCode, in.CounterpartyExposure, limit)
		}
		return "PASSED", "Compliant"

	case "CROSS_BORDER_CLIENT_ELIGIBILITY":
		if !in.DistributionEligible {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: account is not distribution_eligible", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "DUPLICATE_ORDER_WINDOW":
		if in.DuplicateCount > 0 {
			return "WARNING", fmt.Sprintf("Rule %s warning: duplicate_count %d exceeds 0 in window", sc.RuleCode, in.DuplicateCount)
		}
		return "PASSED", "Compliant"

	case "EMIR_FIELD_VALIDITY":
		if !in.EMIRValid {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: emir_valid is false", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "EXECUTION_PRICE_DEVIATION":
		maxDev := d("0.020000")
		if in.ArrivalPriceDeviationPct.GreaterThan(maxDev) {
			return "WARNING", fmt.Sprintf("Rule %s warning: arrival_price_deviation_pct %s exceeds max_bench_deviation %s", sc.RuleCode, in.ArrivalPriceDeviationPct, maxDev)
		}
		return "PASSED", "Compliant"

	case "EXECUTION_WITHIN_SPREAD":
		if in.OutsideSpread {
			return "WARNING", fmt.Sprintf("Rule %s warning: execution was outside_spread", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "FAT_FINGER_ADV_RATIO":
		maxADV := d("0.250000")
		if in.AdvRatio.GreaterThan(maxADV) {
			return "WARNING", fmt.Sprintf("Rule %s warning: adv_ratio %s exceeds adv_multiple %s", sc.RuleCode, in.AdvRatio, maxADV)
		}
		return "PASSED", "Compliant"

	case "FREERIDING_REG_T":
		if in.Side == "SELL" && in.SettledQty.LessThan(in.Quantity) && in.UnpaidCash {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: freeriding violation on sell of unsettled shares with unpaid cash", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "FRONT_RUNNING_CLIENT_ORDER":
		if in.IsPrincipalOrEmployee && in.PendingClientOrdersExist {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: front_running prohibited when pending client orders exist", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "FX_SETTLEMENT_CURRENCY_MATCH":
		if in.Currency != in.AccountBaseCurrency && in.Currency != in.SecurityCurrency {
			return "WARNING", fmt.Sprintf("Rule %s warning: currency_match failed: order currency %s matches neither account base %s nor security %s", sc.RuleCode, in.Currency, in.AccountBaseCurrency, in.SecurityCurrency)
		}
		return "PASSED", "Compliant"

	case "ILLIQUID_ASSET_LIMIT":
		limit := d("0.150000")
		if in.IlliquidAssetsPct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: illiquid_assets_pct %s exceeds max_illiquid_pct %s", sc.RuleCode, in.IlliquidAssetsPct, limit)
		}
		return "PASSED", "Compliant"

	case "LARGE_TRADE_THRESHOLD":
		limit := d("500000.000000")
		if in.Notional.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: notional %s exceeds large_trade_threshold %s", sc.RuleCode, in.Notional, limit)
		}
		return "PASSED", "Compliant"

	case "LAYERING_SPOOF_PATTERN":
		maxCancel := d("0.900000")
		maxOTR := d("50.000000")
		if in.CancelRate5Min.GreaterThan(maxCancel) && in.OrderToTradeRatio.GreaterThan(maxOTR) {
			return "WARNING", fmt.Sprintf("Rule %s warning: layering_spoof pattern detected: cancel_rate %s and otr %s exceed thresholds", sc.RuleCode, in.CancelRate5Min, in.OrderToTradeRatio)
		}
		return "PASSED", "Compliant"

	case "LEVERAGE_VAR_COMMIT":
		maxVaR := d("0.200000")
		maxCommit := d("2.000000")
		if in.VarLeverageRatio.GreaterThan(maxVaR) {
			return "APPROVAL_REQUIRED", fmt.Sprintf("Rule %s requires approval: var_leverage_ratio %s exceeds max_leverage_var %s", sc.RuleCode, in.VarLeverageRatio, maxVaR)
		}
		if in.CommitmentRatio.GreaterThan(maxCommit) {
			return "APPROVAL_REQUIRED", fmt.Sprintf("Rule %s requires approval: commitment_ratio %s exceeds max_commitment_ratio %s", sc.RuleCode, in.CommitmentRatio, maxCommit)
		}
		return "PASSED", "Compliant"

	case "LIQUIDITY_BUCKET_DAYS":
		maxDays := 7
		if in.DaysToLiquidate50Pct > maxDays {
			return "WARNING", fmt.Sprintf("Rule %s warning: days_to_liquidate_50pct %d exceeds max_liquid_days %d", sc.RuleCode, in.DaysToLiquidate50Pct, maxDays)
		}
		return "PASSED", "Compliant"

	case "LULD_PRICE_BAND":
		if in.LimitPrice.LessThan(in.LowerBand) || in.LimitPrice.GreaterThan(in.UpperBand) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: luld_price_band violation: limit price %s outside [%s, %s]", sc.RuleCode, in.LimitPrice, in.LowerBand, in.UpperBand)
		}
		return "PASSED", "Compliant"

	case "MARGIN_HOUSE_LIMIT":
		limit := d("0.700000")
		if in.MarginUtilizationPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: margin_utilization_pct %s exceeds house_margin_limit %s", sc.RuleCode, in.MarginUtilizationPct, limit)
		}
		return "PASSED", "Compliant"

	case "ODD_LOT_ABOVE_MIN":
		if in.Quantity.LessThan(in.RoundLot) && !in.PennyStock {
			return "WARNING", fmt.Sprintf("Rule %s warning: odd_lot quantity %s below round lot %s on standard equity", sc.RuleCode, in.Quantity, in.RoundLot)
		}
		return "PASSED", "Compliant"

	case "ORDER_RATE_LIMIT":
		maxOPM := 100
		if in.OrdersPerMinute > maxOPM {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: orders_per_minute %d exceeds max_opm %d", sc.RuleCode, in.OrdersPerMinute, maxOPM)
		}
		return "PASSED", "Compliant"

	case "PRICE_COLLAR_PCT":
		limit := d("0.100000")
		if in.PriceDeviationPct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: price_deviation_pct %s exceeds collar_pct %s", sc.RuleCode, in.PriceDeviationPct, limit)
		}
		return "PASSED", "Compliant"

	case "PRO_RATA_ALLOCATION_FAIRNESS":
		maxDev := d("0.020000")
		maxDisp := d("0.001000")
		if in.RatioDeviation.GreaterThan(maxDev) && in.PriceDispersion.GreaterThan(maxDisp) {
			return "WARNING", fmt.Sprintf("Rule %s warning: allocation_fairness violated: ratio_deviation %s and price_dispersion %s exceed thresholds", sc.RuleCode, in.RatioDeviation, in.PriceDispersion)
		}
		return "PASSED", "Compliant"

	case "PT_ACCESS_PERSON_RECON":
		if in.StatementDiscrepancy {
			return "WARNING", fmt.Sprintf("Rule %s warning: statement_discrepancy detected in access person trade reconciliation", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "PT_BLACKOUT_PERIOD":
		if in.EmployeeID != nil && in.BlackoutActive {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: employee trade during active blackout_active window", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "PT_MIN_HOLDING_30D":
		minDays := 30
		if in.EmployeeID != nil && in.DaysSincePurchase < minDays {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: employee holding period %d days is less than min_holding_days %d", sc.RuleCode, in.DaysSincePurchase, minDays)
		}
		return "PASSED", "Compliant"

	case "SECTOR_CONCENTRATION":
		limit := d("0.250000")
		if in.SectorExposurePct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: sector_exposure_pct %s exceeds max_sector_pct %s", sc.RuleCode, in.SectorExposurePct, limit)
		}
		return "PASSED", "Compliant"

	case "SETTLEMENT_FAIL_AGING":
		maxDays := 3
		if in.FailAgeDays > maxDays {
			return "APPROVAL_REQUIRED", fmt.Sprintf("Rule %s requires approval: fail_age_days %d exceeds max_fail_days %d", sc.RuleCode, in.FailAgeDays, maxDays)
		}
		return "PASSED", "Compliant"

	case "SHORT_POSITION_RESTRICTION":
		if in.ShortSaleRestricted && in.Side == "SHORT" {
			return "APPROVAL_REQUIRED", fmt.Sprintf("Rule %s requires approval: short_sale_restricted circuit breaker active for short order", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "SHORT_SALE_LOCATE":
		if in.Side == "SHORT" && !in.LocateValid && !in.EasyToBorrow {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: short_sale_locate required for hard-to-borrow security", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "SINGLE_POSITION_NAV":
		limit := d("0.100000")
		if in.NavWeightPct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: nav_weight_pct %s exceeds max_single_nav_pct %s", sc.RuleCode, in.NavWeightPct, limit)
		}
		return "PASSED", "Compliant"

	case "TXN_REPORT_COMPLETENESS":
		if in.MissingFieldsCount > 0 {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: missing_fields_count %d in transaction report", sc.RuleCode, in.MissingFieldsCount)
		}
		if in.LEIInvalid {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: lei_invalid in transaction report", sc.RuleCode)
		}
		return "PASSED", "Compliant"

	case "TXN_REPORT_TIMELINESS":
		maxDelay := 1440
		if in.MinutesSinceExecution > maxDelay {
			return "WARNING", fmt.Sprintf("Rule %s warning: minutes_since_execution %d exceeds max_reporting_delay_minutes %d", sc.RuleCode, in.MinutesSinceExecution, maxDelay)
		}
		return "PASSED", "Compliant"

	case "VENUE_APPROVED_LIST":
		approved := map[string]bool{"XNYS": true, "XNAS": true, "XLON": true, "XFRA": true, "XPAR": true}
		if !approved[in.Venue] {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: venue %s not in approved_venues list", sc.RuleCode, in.Venue)
		}
		return "PASSED", "Compliant"

	default:
		return "PASSED", "Compliant"
	}
}
