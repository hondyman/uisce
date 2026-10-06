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

// CoreScenarioCorpus contains deterministic scenario tests for all 50 Core Gold-Copy rules.
var CoreScenarioCorpus = []Scenario{
	// ====================================================================
	// 1. UCITS_ISSUER_5 (Concentration & Diversification)
	// ====================================================================
	{
		RuleCode:    "UCITS_ISSUER_5",
		Code:        "UCITS_ISSUER_5:PASS",
		Description: "4.9% exposure order, under limit",
		Input:       OrderContext{IssuerExposurePct: d("0.049000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_ISSUER_5",
		Code:        "UCITS_ISSUER_5:BOUNDARY",
		Description: "Exactly 5.000000% — GT is strict, must pass",
		Input:       OrderContext{IssuerExposurePct: d("0.050000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_ISSUER_5",
		Code:        "UCITS_ISSUER_5:FAIL",
		Description: "10.4% exposure, over limit",
		Input:       OrderContext{IssuerExposurePct: d("0.104000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "issuer_exposure_pct"},
	},
	{
		RuleCode:    "UCITS_ISSUER_5",
		Code:        "UCITS_ISSUER_5:ADVERSARIAL",
		Description: "4.7% current + in-flight reservation delta pushes to 5.2%",
		Input: OrderContext{
			IssuerExposurePct:     d("0.052000"),
			ExistingPositionValue: d("47000000"),
			Quantity:              d("5000"),
			Price:                 d("1000"),
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "issuer_exposure_pct"},
	},

	// ====================================================================
	// 2. UCITS_ISSUER_10_EXCEPTION (Concentration & Diversification)
	// ====================================================================
	{
		RuleCode:    "UCITS_ISSUER_10_EXCEPTION",
		Code:        "UCITS10:PASS_GRACE",
		Description: "6.2% exposure held for 60 days (within 180-day grace window) — passes",
		Input:       OrderContext{IssuerExposurePct: d("0.062000"), DaysOver5Pct: 60},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_ISSUER_10_EXCEPTION",
		Code:        "UCITS10:BOUNDARY_DAYS",
		Description: "6.2% exposure held for exactly 180 days (GT strict) — passes",
		Input:       OrderContext{IssuerExposurePct: d("0.062000"), DaysOver5Pct: 180},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_ISSUER_10_EXCEPTION",
		Code:        "UCITS10:EXC_WINDOW_EXPIRED",
		Description: "6.2% exposure held >180 days since crossing 5% — approval path",
		Input:       OrderContext{IssuerExposurePct: d("0.062000"), DaysOver5Pct: 200},
		Expected:    ExpectedOutcome{Status: "APPROVAL_REQUIRED", MustContain: "grandfather_days"},
	},
	{
		RuleCode:    "UCITS_ISSUER_10_EXCEPTION",
		Code:        "UCITS10:OVER_10_HARD_BREACH",
		Description: "10.4% — beyond even the 10% exception limit",
		Input:       OrderContext{IssuerExposurePct: d("0.104000"), DaysOver5Pct: 200},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "iss10_limit_pct"},
	},

	// ====================================================================
	// 3. UCITS_ISSUER_40 (Concentration & Diversification)
	// ====================================================================
	{
		RuleCode:    "UCITS_ISSUER_40",
		Code:        "UCITS40:PASS",
		Description: "38.0% single basket exposure under 40% cap",
		Input:       OrderContext{IssuerExposurePct: d("0.380000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_ISSUER_40",
		Code:        "UCITS40:BOUNDARY",
		Description: "Exactly 40.000000% — GT is strict, passes",
		Input:       OrderContext{IssuerExposurePct: d("0.400000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_ISSUER_40",
		Code:        "UCITS40:FAIL",
		Description: "42.0% single basket exposure breaches 40% cap",
		Input:       OrderContext{IssuerExposurePct: d("0.420000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "iss40_limit_pct"},
	},
	{
		RuleCode:    "UCITS_ISSUER_40",
		Code:        "UCITS40:ADVERSARIAL",
		Description: "Aggregated basket across sub-funds at 41.5% breaches cap",
		Input:       OrderContext{IssuerExposurePct: d("0.415000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "iss40_limit_pct"},
	},

	// ====================================================================
	// 4. UCITS_DEPOSIT_20 (Concentration & Diversification)
	// ====================================================================
	{
		RuleCode:    "UCITS_DEPOSIT_20",
		Code:        "UCITS_DEP20:PASS",
		Description: "18.0% deposit exposure under 20% limit",
		Input:       OrderContext{DepositExposurePct: d("0.180000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_DEPOSIT_20",
		Code:        "UCITS_DEP20:BOUNDARY",
		Description: "Exactly 20.000000% deposit exposure passes",
		Input:       OrderContext{DepositExposurePct: d("0.200000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_DEPOSIT_20",
		Code:        "UCITS_DEP20:FAIL",
		Description: "22.5% deposit exposure breaches 20% limit",
		Input:       OrderContext{DepositExposurePct: d("0.225000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "deposit_limit_pct"},
	},
	{
		RuleCode:    "UCITS_DEPOSIT_20",
		Code:        "UCITS_DEP20:ADVERSARIAL",
		Description: "Split deposit across branch accounts aggregating to 24.0% breaches limit",
		Input:       OrderContext{DepositExposurePct: d("0.240000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "deposit_limit_pct"},
	},

	// ====================================================================
	// 5. ACT40_DIV_75_5 (Concentration & Diversification)
	// ====================================================================
	{
		RuleCode:    "ACT40_DIV_75_5",
		Code:        "ACT40:PASS",
		Description: "80% compliant assets, 4.5% issuer exposure",
		Input:       OrderContext{CompliantAssetsPct: d("0.800000"), IssuerExposurePct: d("0.045000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "ACT40_DIV_75_5",
		Code:        "ACT40:BOUNDARY",
		Description: "Exactly 75% compliant assets and 5% issuer exposure passes",
		Input:       OrderContext{CompliantAssetsPct: d("0.750000"), IssuerExposurePct: d("0.050000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "ACT40_DIV_75_5",
		Code:        "ACT40:FAIL_ASSETS",
		Description: "72% compliant assets breaches 75% floor",
		Input:       OrderContext{CompliantAssetsPct: d("0.720000"), IssuerExposurePct: d("0.040000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "compliant_assets_pct"},
	},
	{
		RuleCode:    "ACT40_DIV_75_5",
		Code:        "ACT40:FAIL_ISSUER",
		Description: "78% compliant assets but 5.5% issuer exposure breaches 5% cap",
		Input:       OrderContext{CompliantAssetsPct: d("0.780000"), IssuerExposurePct: d("0.055000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "issuer_limit_pct"},
	},

	// ====================================================================
	// 6. FOF_20 (Cross-Border & Asset Eligibility)
	// ====================================================================
	{
		RuleCode:    "FOF_20",
		Code:        "FOF20:PASS",
		Description: "15% target fund exposure under 20% limit",
		Input:       OrderContext{TargetFundExposurePct: d("0.150000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FOF_20",
		Code:        "FOF20:BOUNDARY",
		Description: "Exactly 20.000000% target fund exposure passes",
		Input:       OrderContext{TargetFundExposurePct: d("0.200000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FOF_20",
		Code:        "FOF20:FAIL",
		Description: "24.5% target fund exposure breaches 20% limit",
		Input:       OrderContext{TargetFundExposurePct: d("0.245000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "fof_limit_pct"},
	},
	{
		RuleCode:    "FOF_20",
		Code:        "FOF20:ADVERSARIAL",
		Description: "Master-feeder structured exposure totaling 28.0% breaches limit",
		Input:       OrderContext{TargetFundExposurePct: d("0.280000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "fof_limit_pct"},
	},

	// ====================================================================
	// 7. SEC_144A_ELIGIBILITY (Cross-Border & Asset Eligibility)
	// ====================================================================
	{
		RuleCode:    "SEC_144A_ELIGIBILITY",
		Code:        "144A:PASS_QIB",
		Description: "144A asset with verified QIB account",
		Input:       OrderContext{Restricted144A: true, QIBStatus: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SEC_144A_ELIGIBILITY",
		Code:        "144A:PASS_UNRESTRICTED",
		Description: "Unrestricted security for non-QIB account",
		Input:       OrderContext{Restricted144A: false, QIBStatus: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SEC_144A_ELIGIBILITY",
		Code:        "144A:FAIL_NON_QIB",
		Description: "144A asset ordered by non-QIB account",
		Input:       OrderContext{Restricted144A: true, QIBStatus: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "qib_status"},
	},
	{
		RuleCode:    "SEC_144A_ELIGIBILITY",
		Code:        "144A:ADVERSARIAL",
		Description: "Attempted routing of 144A asset to retail account with expired QIB cert",
		Input:       OrderContext{Restricted144A: true, QIBStatus: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "qib_status"},
	},

	// ====================================================================
	// 8. REG_S_OFFSHORE_ONLY (Cross-Border & Asset Eligibility)
	// ====================================================================
	{
		RuleCode:    "REG_S_OFFSHORE_ONLY",
		Code:        "REGS:PASS_OFFSHORE",
		Description: "Reg S asset offered to UK investor",
		Input: OrderContext{
			RegS:                  true,
			Jurisdiction:          "GB",
			DomesticJurisdictions: []string{"US"},
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "REG_S_OFFSHORE_ONLY",
		Code:        "REGS:PASS_NON_REGS",
		Description: "Non-Reg S asset offered to US domestic investor",
		Input: OrderContext{
			RegS:                  false,
			Jurisdiction:          "US",
			DomesticJurisdictions: []string{"US"},
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "REG_S_OFFSHORE_ONLY",
		Code:        "REGS:FAIL_DOMESTIC",
		Description: "Reg S offshore asset offered to US domestic jurisdiction",
		Input: OrderContext{
			RegS:                  true,
			Jurisdiction:          "US",
			DomesticJurisdictions: []string{"US"},
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "domestic_jurisdictions"},
	},
	{
		RuleCode:    "REG_S_OFFSHORE_ONLY",
		Code:        "REGS:ADVERSARIAL",
		Description: "Reg S asset routed to US domestic entity via offshore shell address",
		Input: OrderContext{
			RegS:                  true,
			Jurisdiction:          "US",
			DomesticJurisdictions: []string{"US"},
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "domestic_jurisdictions"},
	},

	// ====================================================================
	// 9. RESTRICTED_LIST_BLOCK (Restricted Lists & Insider Lists)
	// ====================================================================
	{
		RuleCode:    "RESTRICTED_LIST_BLOCK",
		Code:        "RL:PASS",
		Description: "Security not on restricted list",
		Input:       OrderContext{InRestrictedList: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "RESTRICTED_LIST_BLOCK",
		Code:        "RL:BOUNDARY",
		Description: "Security recently de-listed from restricted list",
		Input:       OrderContext{InRestrictedList: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "RESTRICTED_LIST_BLOCK",
		Code:        "RL:FAIL",
		Description: "Security is actively on restricted list",
		Input:       OrderContext{InRestrictedList: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "restricted_list"},
	},
	{
		RuleCode:    "RESTRICTED_LIST_BLOCK",
		Code:        "RL:ADVERSARIAL",
		Description: "Order for equity linked note on restricted underlying",
		Input:       OrderContext{InRestrictedList: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "restricted_list"},
	},

	// ====================================================================
	// 10. INSIDER_LIST_MAR (Restricted Lists & Insider Lists)
	// ====================================================================
	{
		RuleCode:    "INSIDER_LIST_MAR",
		Code:        "IL:PASS",
		Description: "Security not in insider list",
		Input:       OrderContext{InInsiderList: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "INSIDER_LIST_MAR",
		Code:        "IL:BOUNDARY",
		Description: "Security with expired insider deal project tag",
		Input:       OrderContext{InInsiderList: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "INSIDER_LIST_MAR",
		Code:        "IL:FAIL",
		Description: "Security is in insider list",
		Input:       OrderContext{InInsiderList: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "insider_list"},
	},
	{
		RuleCode:    "INSIDER_LIST_MAR",
		Code:        "IL:ADVERSARIAL",
		Description: "Desk trader attempting basket execution containing insider issuer",
		Input:       OrderContext{InInsiderList: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "insider_list"},
	},

	// ====================================================================
	// 11. SANCTIONS_ISSUER_BLOCK (Restricted Lists & Insider Lists)
	// ====================================================================
	{
		RuleCode:    "SANCTIONS_ISSUER_BLOCK",
		Code:        "SANC:PASS",
		Description: "Issuer clean from sanctions lists",
		Input:       OrderContext{IssuerIsSanctioned: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SANCTIONS_ISSUER_BLOCK",
		Code:        "SANC:BOUNDARY",
		Description: "Issuer in review but not sanctioned",
		Input:       OrderContext{IssuerIsSanctioned: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SANCTIONS_ISSUER_BLOCK",
		Code:        "SANC:FAIL",
		Description: "Issuer is on OFAC / EU sanctions list",
		Input:       OrderContext{IssuerIsSanctioned: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "sanctioned"},
	},
	{
		RuleCode:    "SANCTIONS_ISSUER_BLOCK",
		Code:        "SANC:ADVERSARIAL",
		Description: "Sanctioned parent company disguised through overseas SPV",
		Input:       OrderContext{IssuerIsSanctioned: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "sanctioned"},
	},

	// ====================================================================
	// 12. COUNTRY_EXPOSURE_LIMIT (Cross-Border & Asset Eligibility)
	// ====================================================================
	{
		RuleCode:    "COUNTRY_EXPOSURE_LIMIT",
		Code:        "CTRY:PASS",
		Description: "12% country exposure under 15% limit",
		Input:       OrderContext{CountryExposurePct: d("0.120000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "COUNTRY_EXPOSURE_LIMIT",
		Code:        "CTRY:BOUNDARY",
		Description: "Exactly 15.000000% country exposure passes",
		Input:       OrderContext{CountryExposurePct: d("0.150000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "COUNTRY_EXPOSURE_LIMIT",
		Code:        "CTRY:FAIL",
		Description: "18.5% country exposure breaches 15% limit",
		Input:       OrderContext{CountryExposurePct: d("0.185000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "country_limit_pct"},
	},
	{
		RuleCode:    "COUNTRY_EXPOSURE_LIMIT",
		Code:        "CTRY:ADVERSARIAL",
		Description: "Dual-domiciled issuer exposure pushing country total to 21.0%",
		Input:       OrderContext{CountryExposurePct: d("0.210000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "country_limit_pct"},
	},

	// ====================================================================
	// 13. REG_M_RULE_105 (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "REG_M_RULE_105",
		Code:        "REGM:PASS_BUY",
		Description: "Long buy order during offering window is permitted",
		Input:       OrderContext{Side: "BUY", CoveredOffering: true, DaysBeforePricing: 2},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "REG_M_RULE_105",
		Code:        "REGM:BOUNDARY",
		Description: "Short sale 6 days before pricing (>5 day restricted window) passes",
		Input:       OrderContext{Side: "SHORT", CoveredOffering: true, DaysBeforePricing: 6},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "REG_M_RULE_105",
		Code:        "REGM:FAIL_SHORT_WINDOW",
		Description: "Short sale 3 days before offering pricing breaches Rule 105",
		Input:       OrderContext{Side: "SHORT", CoveredOffering: true, DaysBeforePricing: 3},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "restricted_window_days"},
	},
	{
		RuleCode:    "REG_M_RULE_105",
		Code:        "REGM:ADVERSARIAL",
		Description: "Short sale on pricing day (0 days before pricing)",
		Input:       OrderContext{Side: "SHORT", CoveredOffering: true, DaysBeforePricing: 0},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "restricted_window_days"},
	},

	// ====================================================================
	// 14. WASH_SALE_1091 (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "WASH_SALE_1091",
		Code:        "WS:PASS_NO_LOSS",
		Description: "Buy order with no realized loss in previous 30 days",
		Input:       OrderContext{Side: "BUY", RealizedLoss30d: false, SameBeneficialOwner: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "WASH_SALE_1091",
		Code:        "WS:PASS_DIFF_OWNER",
		Description: "Buy order for separate independent beneficial owner",
		Input:       OrderContext{Side: "BUY", RealizedLoss30d: true, SameBeneficialOwner: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "WASH_SALE_1091",
		Code:        "WS:FAIL",
		Description: "Buy order for same beneficial owner within 30 days of realized loss",
		Input:       OrderContext{Side: "BUY", RealizedLoss30d: true, SameBeneficialOwner: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "realized_loss_30d"},
	},
	{
		RuleCode:    "WASH_SALE_1091",
		Code:        "WS:ADVERSARIAL",
		Description: "Buy order routed through related trust account with same beneficial owner",
		Input:       OrderContext{Side: "BUY", RealizedLoss30d: true, SameBeneficialOwner: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "realized_loss_30d"},
	},

	// ====================================================================
	// 15. SELF_TRADE_PREVENT (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "SELF_TRADE_PREVENT",
		Code:        "STP:PASS",
		Description: "No pending opposing orders on venue",
		Input:       OrderContext{PendingOpposingOrder: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SELF_TRADE_PREVENT",
		Code:        "STP:BOUNDARY",
		Description: "Opposing orders cancelled prior to new order entry",
		Input:       OrderContext{PendingOpposingOrder: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SELF_TRADE_PREVENT",
		Code:        "STP:FAIL",
		Description: "Matching pending opposing order from same beneficial owner group",
		Input:       OrderContext{PendingOpposingOrder: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "pending_opposing_order"},
	},
	{
		RuleCode:    "SELF_TRADE_PREVENT",
		Code:        "STP:ADVERSARIAL",
		Description: "Concurrent opposing limit orders entered within same millisecond",
		Input:       OrderContext{PendingOpposingOrder: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "pending_opposing_order"},
	},

	// ====================================================================
	// 16. PT_PRECLEARANCE_REQUIRED (Fair Allocation & Personal Trading)
	// ====================================================================
	{
		RuleCode:    "PT_PRECLEARANCE_REQUIRED",
		Code:        "PT:PASS_VALID",
		Description: "Employee trade with valid preclearance",
		Input:       OrderContext{EmployeeID: &testEmpID, PreclearanceValid: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PT_PRECLEARANCE_REQUIRED",
		Code:        "PT:PASS_CLIENT",
		Description: "Client order without employee ID (rule exempt for clients)",
		Input:       OrderContext{EmployeeID: nil, PreclearanceValid: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PT_PRECLEARANCE_REQUIRED",
		Code:        "PT:FAIL_NO_PRECLEAR",
		Description: "Employee trade without valid preclearance",
		Input:       OrderContext{EmployeeID: &testEmpID, PreclearanceValid: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "preclearance_valid"},
	},
	{
		RuleCode:    "PT_PRECLEARANCE_REQUIRED",
		Code:        "PT:ADVERSARIAL",
		Description: "Employee personal trade submitted via corporate account without preclearance",
		Input:       OrderContext{EmployeeID: &testEmpID, PreclearanceValid: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "preclearance_valid"},
	},

	// ====================================================================
	// 17. FAT_FINGER_NOTIONAL (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "FAT_FINGER_NOTIONAL",
		Code:        "FFN:PASS",
		Description: "100 shares @ $100 = $10,000 <= $50,000,000",
		Input:       OrderContext{Notional: d("10000.00")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FAT_FINGER_NOTIONAL",
		Code:        "FFN:BOUNDARY",
		Description: "Exactly $50,000,000 notional — GT strict, passes",
		Input:       OrderContext{Notional: d("50000000.00")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FAT_FINGER_NOTIONAL",
		Code:        "FFN:FAIL",
		Description: "$50,000,001 notional breaches $50M limit",
		Input:       OrderContext{Notional: d("50000001.00")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "max_notional"},
	},
	{
		RuleCode:    "FAT_FINGER_NOTIONAL",
		Code:        "FFN:ADVERSARIAL",
		Description: "Accidental 100x quantity fat-finger creating $120,000,000 notional",
		Input:       OrderContext{Notional: d("120000000.00")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "max_notional"},
	},

	// ====================================================================
	// 18. AGGREGATION_ELIGIBILITY (Fair Allocation & Personal Trading)
	// ====================================================================
	{
		RuleCode:    "AGGREGATION_ELIGIBILITY",
		Code:        "AGG:PASS_ELIGIBLE",
		Description: "Block order with eligible accounts",
		Input:       OrderContext{CanBeAggregated: true, ParentIsBlock: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "AGGREGATION_ELIGIBILITY",
		Code:        "AGG:BOUNDARY_NON_BLOCK",
		Description: "Ineligible account on non-block single order passes",
		Input:       OrderContext{CanBeAggregated: false, ParentIsBlock: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "AGGREGATION_ELIGIBILITY",
		Code:        "AGG:FAIL_INELIGIBLE_BLOCK",
		Description: "Ineligible account included in block parent order",
		Input:       OrderContext{CanBeAggregated: false, ParentIsBlock: true},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "can_be_aggregated"},
	},
	{
		RuleCode:    "AGGREGATION_ELIGIBILITY",
		Code:        "AGG:ADVERSARIAL",
		Description: "Retail wealth account improperly grouped into institutional block execution",
		Input:       OrderContext{CanBeAggregated: false, ParentIsBlock: true},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "can_be_aggregated"},
	},

	// ====================================================================
	// 19. COUNTERPARTY_OTC_LIMIT (Derivatives & Leverage)
	// ====================================================================
	{
		RuleCode:    "COUNTERPARTY_OTC_LIMIT",
		Code:        "CP_OTC:PASS",
		Description: "$18M OTC exposure under $25M limit",
		Input:       OrderContext{CounterpartyExposure: d("18000000.00")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "COUNTERPARTY_OTC_LIMIT",
		Code:        "CP_OTC:BOUNDARY",
		Description: "Exactly $25,000,000.00 exposure passes",
		Input:       OrderContext{CounterpartyExposure: d("25000000.00")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "COUNTERPARTY_OTC_LIMIT",
		Code:        "CP_OTC:FAIL",
		Description: "$25,000,001.00 OTC exposure breaches $25M limit",
		Input:       OrderContext{CounterpartyExposure: d("25000001.00")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "counterparty_limit"},
	},
	{
		RuleCode:    "COUNTERPARTY_OTC_LIMIT",
		Code:        "CP_OTC:ADVERSARIAL",
		Description: "Bilateral swap uncollateralized exposure reaching $35M",
		Input:       OrderContext{CounterpartyExposure: d("35000000.00")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "counterparty_limit"},
	},

	// ====================================================================
	// 20. CROSS_BORDER_CLIENT_ELIGIBILITY (Cross-Border & Asset Eligibility)
	// ====================================================================
	{
		RuleCode:    "CROSS_BORDER_CLIENT_ELIGIBILITY",
		Code:        "XBORDER:PASS",
		Description: "Account with valid cross-border distribution eligibility",
		Input:       OrderContext{DistributionEligible: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "CROSS_BORDER_CLIENT_ELIGIBILITY",
		Code:        "XBORDER:BOUNDARY",
		Description: "Passported UCITS fund distributed in authorized host state",
		Input:       OrderContext{DistributionEligible: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "CROSS_BORDER_CLIENT_ELIGIBILITY",
		Code:        "XBORDER:FAIL",
		Description: "Ineligible distribution account",
		Input:       OrderContext{DistributionEligible: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "distribution_eligible"},
	},
	{
		RuleCode:    "CROSS_BORDER_CLIENT_ELIGIBILITY",
		Code:        "XBORDER:ADVERSARIAL",
		Description: "Unregistered foreign client soliciting local fund without PRIIPs KIID",
		Input:       OrderContext{DistributionEligible: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "distribution_eligible"},
	},

	// ====================================================================
	// 21. DUPLICATE_ORDER_WINDOW (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "DUPLICATE_ORDER_WINDOW",
		Code:        "DUP:PASS",
		Description: "First order ingress with 0 duplicates in 60s window",
		Input:       OrderContext{DuplicateCount: 0},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "DUPLICATE_ORDER_WINDOW",
		Code:        "DUP:BOUNDARY",
		Description: "Duplicate count 0 with previous order 61s ago",
		Input:       OrderContext{DuplicateCount: 0},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "DUPLICATE_ORDER_WINDOW",
		Code:        "DUP:FAIL",
		Description: "1 duplicate order detected within 60s window",
		Input:       OrderContext{DuplicateCount: 1},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "duplicate_count"},
	},
	{
		RuleCode:    "DUPLICATE_ORDER_WINDOW",
		Code:        "DUP:ADVERSARIAL",
		Description: "Rapid algorithmic re-submission burst creating 5 duplicates",
		Input:       OrderContext{DuplicateCount: 5},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "duplicate_count"},
	},

	// ====================================================================
	// 22. EMIR_FIELD_VALIDITY (Transaction Reporting)
	// ====================================================================
	{
		RuleCode:    "EMIR_FIELD_VALIDITY",
		Code:        "EMIR:PASS",
		Description: "Valid EMIR ISO 20022 XML report payload",
		Input:       OrderContext{EMIRValid: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "EMIR_FIELD_VALIDITY",
		Code:        "EMIR:BOUNDARY",
		Description: "Valid EMIR record with minimum mandatory tags",
		Input:       OrderContext{EMIRValid: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "EMIR_FIELD_VALIDITY",
		Code:        "EMIR:FAIL",
		Description: "Invalid EMIR record failing schema validation",
		Input:       OrderContext{EMIRValid: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "emir_valid"},
	},
	{
		RuleCode:    "EMIR_FIELD_VALIDITY",
		Code:        "EMIR:ADVERSARIAL",
		Description: "Corrupted XML namespaces and missing UTI identifier",
		Input:       OrderContext{EMIRValid: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "emir_valid"},
	},

	// ====================================================================
	// 23. EXECUTION_PRICE_DEVIATION (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "EXECUTION_PRICE_DEVIATION",
		Code:        "EXEC_DEV:PASS",
		Description: "1.2% arrival price deviation under 2.0% threshold",
		Input:       OrderContext{ArrivalPriceDeviationPct: d("0.012000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "EXECUTION_PRICE_DEVIATION",
		Code:        "EXEC_DEV:BOUNDARY",
		Description: "Exactly 2.000000% arrival price deviation passes",
		Input:       OrderContext{ArrivalPriceDeviationPct: d("0.020000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "EXECUTION_PRICE_DEVIATION",
		Code:        "EXEC_DEV:FAIL",
		Description: "2.8% arrival price deviation breaches 2.0% threshold",
		Input:       OrderContext{ArrivalPriceDeviationPct: d("0.028000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_bench_deviation"},
	},
	{
		RuleCode:    "EXECUTION_PRICE_DEVIATION",
		Code:        "EXEC_DEV:ADVERSARIAL",
		Description: "Severe 7.5% slippage during illiquid opening auction",
		Input:       OrderContext{ArrivalPriceDeviationPct: d("0.075000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_bench_deviation"},
	},

	// ====================================================================
	// 24. EXECUTION_WITHIN_SPREAD (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "EXECUTION_WITHIN_SPREAD",
		Code:        "SPREAD:PASS",
		Description: "Execution cleanly inside NBBO bid-ask spread",
		Input:       OrderContext{OutsideSpread: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "EXECUTION_WITHIN_SPREAD",
		Code:        "SPREAD:BOUNDARY",
		Description: "Execution at touch price on displayed ask",
		Input:       OrderContext{OutsideSpread: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "EXECUTION_WITHIN_SPREAD",
		Code:        "SPREAD:FAIL",
		Description: "Execution printed outside NBBO spread",
		Input:       OrderContext{OutsideSpread: true},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "outside_spread"},
	},
	{
		RuleCode:    "EXECUTION_WITHIN_SPREAD",
		Code:        "SPREAD:ADVERSARIAL",
		Description: "Off-market trade executed 50 bps beyond prevailing spread",
		Input:       OrderContext{OutsideSpread: true},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "outside_spread"},
	},

	// ====================================================================
	// 25. FAT_FINGER_ADV_RATIO (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "FAT_FINGER_ADV_RATIO",
		Code:        "ADV_RATIO:PASS",
		Description: "15% of 30d ADV under 25% threshold",
		Input:       OrderContext{AdvRatio: d("0.150000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FAT_FINGER_ADV_RATIO",
		Code:        "ADV_RATIO:BOUNDARY",
		Description: "Exactly 25.000000% of 30d ADV passes",
		Input:       OrderContext{AdvRatio: d("0.250000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FAT_FINGER_ADV_RATIO",
		Code:        "ADV_RATIO:FAIL",
		Description: "32% of 30d ADV breaches 25% threshold",
		Input:       OrderContext{AdvRatio: d("0.320000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "adv_multiple"},
	},
	{
		RuleCode:    "FAT_FINGER_ADV_RATIO",
		Code:        "ADV_RATIO:ADVERSARIAL",
		Description: "Massive single market order equaling 180% of daily volume",
		Input:       OrderContext{AdvRatio: d("1.800000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "adv_multiple"},
	},

	// ====================================================================
	// 26. FREERIDING_REG_T (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "FREERIDING_REG_T",
		Code:        "REG_T:PASS_SETTLED",
		Description: "Sale of fully settled position with paid cash",
		Input:       OrderContext{Side: "SELL", SettledQty: d("1000"), Quantity: d("500"), UnpaidCash: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FREERIDING_REG_T",
		Code:        "REG_T:BOUNDARY",
		Description: "Sale of exact settled quantity with unpaid cash",
		Input:       OrderContext{Side: "SELL", SettledQty: d("500"), Quantity: d("500"), UnpaidCash: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FREERIDING_REG_T",
		Code:        "REG_T:FAIL",
		Description: "Sale of unsettled shares with unpaid purchase cash",
		Input:       OrderContext{Side: "SELL", SettledQty: d("200"), Quantity: d("500"), UnpaidCash: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "freeriding"},
	},
	{
		RuleCode:    "FREERIDING_REG_T",
		Code:        "REG_T:ADVERSARIAL",
		Description: "Intraday flip of 1,000 unsettled shares with zero paid funds",
		Input:       OrderContext{Side: "SELL", SettledQty: d("0"), Quantity: d("1000"), UnpaidCash: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "freeriding"},
	},

	// ====================================================================
	// 27. FRONT_RUNNING_CLIENT_ORDER (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "FRONT_RUNNING_CLIENT_ORDER",
		Code:        "FRONTRUN:PASS_NO_CLIENTS",
		Description: "Principal trade with no pending client orders in market",
		Input:       OrderContext{IsPrincipalOrEmployee: true, PendingClientOrdersExist: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FRONT_RUNNING_CLIENT_ORDER",
		Code:        "FRONTRUN:BOUNDARY_CLIENT",
		Description: "Client order itself while other client orders exist",
		Input:       OrderContext{IsPrincipalOrEmployee: false, PendingClientOrdersExist: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FRONT_RUNNING_CLIENT_ORDER",
		Code:        "FRONTRUN:FAIL",
		Description: "Principal order placed ahead of pending client block",
		Input:       OrderContext{IsPrincipalOrEmployee: true, PendingClientOrdersExist: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "front_running"},
	},
	{
		RuleCode:    "FRONT_RUNNING_CLIENT_ORDER",
		Code:        "FRONTRUN:ADVERSARIAL",
		Description: "Proprietary market-making fill entered concurrently with institutional client parent",
		Input:       OrderContext{IsPrincipalOrEmployee: true, PendingClientOrdersExist: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "front_running"},
	},

	// ====================================================================
	// 28. FX_SETTLEMENT_CURRENCY_MATCH (Cross-Border & Asset Eligibility)
	// ====================================================================
	{
		RuleCode:    "FX_SETTLEMENT_CURRENCY_MATCH",
		Code:        "FX_MATCH:PASS_BASE",
		Description: "Order currency matches account base currency",
		Input:       OrderContext{Currency: "USD", AccountBaseCurrency: "USD", SecurityCurrency: "EUR"},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FX_SETTLEMENT_CURRENCY_MATCH",
		Code:        "FX_MATCH:BOUNDARY_SEC",
		Description: "Order currency matches underlying security denomination",
		Input:       OrderContext{Currency: "EUR", AccountBaseCurrency: "USD", SecurityCurrency: "EUR"},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FX_SETTLEMENT_CURRENCY_MATCH",
		Code:        "FX_MATCH:FAIL",
		Description: "Order currency matches neither account base nor security denomination",
		Input:       OrderContext{Currency: "JPY", AccountBaseCurrency: "USD", SecurityCurrency: "EUR"},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "currency_match"},
	},
	{
		RuleCode:    "FX_SETTLEMENT_CURRENCY_MATCH",
		Code:        "FX_MATCH:ADVERSARIAL",
		Description: "Three-way currency mismatch across GBP order, CHF account, and CAD security",
		Input:       OrderContext{Currency: "GBP", AccountBaseCurrency: "CHF", SecurityCurrency: "CAD"},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "currency_match"},
	},

	// ====================================================================
	// 29. ILLIQUID_ASSET_LIMIT (Liquidity & Settlement)
	// ====================================================================
	{
		RuleCode:    "ILLIQUID_ASSET_LIMIT",
		Code:        "ILLIQUID:PASS",
		Description: "10% illiquid holdings under 15% threshold",
		Input:       OrderContext{IlliquidAssetsPct: d("0.100000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "ILLIQUID_ASSET_LIMIT",
		Code:        "ILLIQUID:BOUNDARY",
		Description: "Exactly 15.000000% illiquid holdings passes",
		Input:       OrderContext{IlliquidAssetsPct: d("0.150000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "ILLIQUID_ASSET_LIMIT",
		Code:        "ILLIQUID:FAIL",
		Description: "18.2% illiquid holdings breaches 15% threshold",
		Input:       OrderContext{IlliquidAssetsPct: d("0.182000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_illiquid_pct"},
	},
	{
		RuleCode:    "ILLIQUID_ASSET_LIMIT",
		Code:        "ILLIQUID:ADVERSARIAL",
		Description: "Level 3 private credit allocations pushing illiquid ratio to 28.0%",
		Input:       OrderContext{IlliquidAssetsPct: d("0.280000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_illiquid_pct"},
	},

	// ====================================================================
	// 30. LARGE_TRADE_THRESHOLD (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "LARGE_TRADE_THRESHOLD",
		Code:        "LARGE_TRADE:PASS",
		Description: "$250k notional order under $500k threshold",
		Input:       OrderContext{Notional: d("250000.00")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LARGE_TRADE_THRESHOLD",
		Code:        "LARGE_TRADE:BOUNDARY",
		Description: "Exactly $500,000.00 notional passes",
		Input:       OrderContext{Notional: d("500000.00")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LARGE_TRADE_THRESHOLD",
		Code:        "LARGE_TRADE:FAIL",
		Description: "$500,001.00 notional breaches $500k escalation threshold",
		Input:       OrderContext{Notional: d("500001.00")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "large_trade_threshold"},
	},
	{
		RuleCode:    "LARGE_TRADE_THRESHOLD",
		Code:        "LARGE_TRADE:ADVERSARIAL",
		Description: "Block parent order of $4,500,000 notional triggering supervisor escalation",
		Input:       OrderContext{Notional: d("4500000.00")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "large_trade_threshold"},
	},

	// ====================================================================
	// 31. LAYERING_SPOOF_PATTERN (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "LAYERING_SPOOF_PATTERN",
		Code:        "SPOOF:PASS",
		Description: "75% cancel rate, 30 order-to-trade ratio under thresholds",
		Input:       OrderContext{CancelRate5Min: d("0.750000"), OrderToTradeRatio: d("30.000000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LAYERING_SPOOF_PATTERN",
		Code:        "SPOOF:BOUNDARY",
		Description: "Exactly 90% cancel rate and 50 OTR passes",
		Input:       OrderContext{CancelRate5Min: d("0.900000"), OrderToTradeRatio: d("50.000000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LAYERING_SPOOF_PATTERN",
		Code:        "SPOOF:FAIL",
		Description: "95% cancel rate and 75 OTR breaches spoofing thresholds",
		Input:       OrderContext{CancelRate5Min: d("0.950000"), OrderToTradeRatio: d("75.000000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "layering_spoof"},
	},
	{
		RuleCode:    "LAYERING_SPOOF_PATTERN",
		Code:        "SPOOF:ADVERSARIAL",
		Description: "Automated quote stuffing with 99% cancel rate and 400 OTR",
		Input:       OrderContext{CancelRate5Min: d("0.990000"), OrderToTradeRatio: d("400.000000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "layering_spoof"},
	},

	// ====================================================================
	// 32. LEVERAGE_VAR_COMMIT (Derivatives & Leverage)
	// ====================================================================
	{
		RuleCode:    "LEVERAGE_VAR_COMMIT",
		Code:        "LEV_VAR:PASS",
		Description: "15% VaR and 1.5x commitment leverage under limits",
		Input:       OrderContext{VarLeverageRatio: d("0.150000"), CommitmentRatio: d("1.500000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LEVERAGE_VAR_COMMIT",
		Code:        "LEV_VAR:BOUNDARY",
		Description: "Exactly 20% VaR and 2.0x commitment passes",
		Input:       OrderContext{VarLeverageRatio: d("0.200000"), CommitmentRatio: d("2.000000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LEVERAGE_VAR_COMMIT",
		Code:        "LEV_VAR:FAIL_VAR",
		Description: "24% VaR breaches 20% limit",
		Input:       OrderContext{VarLeverageRatio: d("0.240000"), CommitmentRatio: d("1.800000")},
		Expected:    ExpectedOutcome{Status: "APPROVAL_REQUIRED", MustContain: "max_leverage_var"},
	},
	{
		RuleCode:    "LEVERAGE_VAR_COMMIT",
		Code:        "LEV_VAR:FAIL_COMMIT",
		Description: "2.4x commitment ratio breaches 2.0x cap",
		Input:       OrderContext{VarLeverageRatio: d("0.180000"), CommitmentRatio: d("2.400000")},
		Expected:    ExpectedOutcome{Status: "APPROVAL_REQUIRED", MustContain: "max_commitment_ratio"},
	},

	// ====================================================================
	// 33. LIQUIDITY_BUCKET_DAYS (Liquidity & Settlement)
	// ====================================================================
	{
		RuleCode:    "LIQUIDITY_BUCKET_DAYS",
		Code:        "LIQ_DAYS:PASS",
		Description: "4 days to liquidate 50% under 7-day limit",
		Input:       OrderContext{DaysToLiquidate50Pct: 4},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LIQUIDITY_BUCKET_DAYS",
		Code:        "LIQ_DAYS:BOUNDARY",
		Description: "Exactly 7 days to liquidate 50% passes",
		Input:       OrderContext{DaysToLiquidate50Pct: 7},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LIQUIDITY_BUCKET_DAYS",
		Code:        "LIQ_DAYS:FAIL",
		Description: "10 days to liquidate 50% breaches 7-day limit",
		Input:       OrderContext{DaysToLiquidate50Pct: 10},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_liquid_days"},
	},
	{
		RuleCode:    "LIQUIDITY_BUCKET_DAYS",
		Code:        "LIQ_DAYS:ADVERSARIAL",
		Description: "Highly concentrated small-cap position requiring 28 days to liquidate",
		Input:       OrderContext{DaysToLiquidate50Pct: 28},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_liquid_days"},
	},

	// ====================================================================
	// 34. LULD_PRICE_BAND (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "LULD_PRICE_BAND",
		Code:        "LULD:PASS",
		Description: "Limit price $100 cleanly inside $95 - $105 price band",
		Input:       OrderContext{LimitPrice: d("100.00"), LowerBand: d("95.00"), UpperBand: d("105.00")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LULD_PRICE_BAND",
		Code:        "LULD:BOUNDARY_LOWER",
		Description: "Limit price exactly at lower band $95.00 passes",
		Input:       OrderContext{LimitPrice: d("95.00"), LowerBand: d("95.00"), UpperBand: d("105.00")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "LULD_PRICE_BAND",
		Code:        "LULD:FAIL_BELOW",
		Description: "Limit price $94.00 breaches lower band $95.00",
		Input:       OrderContext{LimitPrice: d("94.00"), LowerBand: d("95.00"), UpperBand: d("105.00")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "luld_price_band"},
	},
	{
		RuleCode:    "LULD_PRICE_BAND",
		Code:        "LULD:FAIL_ABOVE",
		Description: "Limit price $106.50 breaches upper band $105.00",
		Input:       OrderContext{LimitPrice: d("106.50"), LowerBand: d("95.00"), UpperBand: d("105.00")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "luld_price_band"},
	},

	// ====================================================================
	// 35. MARGIN_HOUSE_LIMIT (Derivatives & Leverage)
	// ====================================================================
	{
		RuleCode:    "MARGIN_HOUSE_LIMIT",
		Code:        "MARGIN:PASS",
		Description: "55% margin utilization under 70% house capacity limit",
		Input:       OrderContext{MarginUtilizationPct: d("0.550000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "MARGIN_HOUSE_LIMIT",
		Code:        "MARGIN:BOUNDARY",
		Description: "Exactly 70.000000% margin utilization passes",
		Input:       OrderContext{MarginUtilizationPct: d("0.700000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "MARGIN_HOUSE_LIMIT",
		Code:        "MARGIN:FAIL",
		Description: "74% margin utilization breaches 70% limit",
		Input:       OrderContext{MarginUtilizationPct: d("0.740000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "house_margin_limit"},
	},
	{
		RuleCode:    "MARGIN_HOUSE_LIMIT",
		Code:        "MARGIN:ADVERSARIAL",
		Description: "Intraday margin spike pushing account to 92% utilization",
		Input:       OrderContext{MarginUtilizationPct: d("0.920000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "house_margin_limit"},
	},

	// ====================================================================
	// 36. ODD_LOT_ABOVE_MIN (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "ODD_LOT_ABOVE_MIN",
		Code:        "ODD_LOT:PASS_ROUND",
		Description: "100 shares round lot order passes",
		Input:       OrderContext{Quantity: d("100"), RoundLot: d("100"), PennyStock: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "ODD_LOT_ABOVE_MIN",
		Code:        "ODD_LOT:BOUNDARY_PENNY",
		Description: "50 shares odd lot permitted for penny stock exemption",
		Input:       OrderContext{Quantity: d("50"), RoundLot: d("100"), PennyStock: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "ODD_LOT_ABOVE_MIN",
		Code:        "ODD_LOT:FAIL",
		Description: "75 shares odd lot on non-penny equity triggers warning",
		Input:       OrderContext{Quantity: d("75"), RoundLot: d("100"), PennyStock: false},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "odd_lot"},
	},
	{
		RuleCode:    "ODD_LOT_ABOVE_MIN",
		Code:        "ODD_LOT:ADVERSARIAL",
		Description: "1 share micro-lot order on standard equity",
		Input:       OrderContext{Quantity: d("1"), RoundLot: d("100"), PennyStock: false},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "odd_lot"},
	},

	// ====================================================================
	// 37. ORDER_RATE_LIMIT (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "ORDER_RATE_LIMIT",
		Code:        "RATE_LIMIT:PASS",
		Description: "75 orders/min under 100 max OPM",
		Input:       OrderContext{OrdersPerMinute: 75},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "ORDER_RATE_LIMIT",
		Code:        "RATE_LIMIT:BOUNDARY",
		Description: "Exactly 100 orders/min passes",
		Input:       OrderContext{OrdersPerMinute: 100},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "ORDER_RATE_LIMIT",
		Code:        "RATE_LIMIT:FAIL",
		Description: "105 orders/min breaches 100 OPM throttle",
		Input:       OrderContext{OrdersPerMinute: 105},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "max_opm"},
	},
	{
		RuleCode:    "ORDER_RATE_LIMIT",
		Code:        "RATE_LIMIT:ADVERSARIAL",
		Description: "Runaway algo loop emitting 450 orders/min",
		Input:       OrderContext{OrdersPerMinute: 450},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "max_opm"},
	},

	// ====================================================================
	// 38. PRICE_COLLAR_PCT (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "PRICE_COLLAR_PCT",
		Code:        "COLLAR:PASS",
		Description: "5% price deviation under 10% collar threshold",
		Input:       OrderContext{PriceDeviationPct: d("0.050000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PRICE_COLLAR_PCT",
		Code:        "COLLAR:BOUNDARY",
		Description: "Exactly 10.000000% price deviation passes",
		Input:       OrderContext{PriceDeviationPct: d("0.100000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PRICE_COLLAR_PCT",
		Code:        "COLLAR:FAIL",
		Description: "13.5% price deviation breaches 10% collar",
		Input:       OrderContext{PriceDeviationPct: d("0.135000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "collar_pct"},
	},
	{
		RuleCode:    "PRICE_COLLAR_PCT",
		Code:        "COLLAR:ADVERSARIAL",
		Description: "Extreme erroneous limit order at 65% premium over market",
		Input:       OrderContext{PriceDeviationPct: d("0.650000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "collar_pct"},
	},

	// ====================================================================
	// 39. PRO_RATA_ALLOCATION_FAIRNESS (Fair Allocation & Personal Trading)
	// ====================================================================
	{
		RuleCode:    "PRO_RATA_ALLOCATION_FAIRNESS",
		Code:        "PRO_RATA:PASS",
		Description: "1.0% ratio deviation, 0.05 bps dispersion under fairness limits",
		Input:       OrderContext{RatioDeviation: d("0.010000"), PriceDispersion: d("0.000500")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PRO_RATA_ALLOCATION_FAIRNESS",
		Code:        "PRO_RATA:BOUNDARY",
		Description: "Exactly 2% ratio deviation and 10 bps dispersion passes",
		Input:       OrderContext{RatioDeviation: d("0.020000"), PriceDispersion: d("0.001000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PRO_RATA_ALLOCATION_FAIRNESS",
		Code:        "PRO_RATA:FAIL",
		Description: "3.5% ratio deviation and 25 bps dispersion breaches fairness",
		Input:       OrderContext{RatioDeviation: d("0.035000"), PriceDispersion: d("0.002500")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "allocation_fairness"},
	},
	{
		RuleCode:    "PRO_RATA_ALLOCATION_FAIRNESS",
		Code:        "PRO_RATA:ADVERSARIAL",
		Description: "Systematic cherry-picking with 14% deviation and 80 bps dispersion",
		Input:       OrderContext{RatioDeviation: d("0.140000"), PriceDispersion: d("0.008000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "allocation_fairness"},
	},

	// ====================================================================
	// 40. PT_ACCESS_PERSON_RECON (Fair Allocation & Personal Trading)
	// ====================================================================
	{
		RuleCode:    "PT_ACCESS_PERSON_RECON",
		Code:        "ACCESS_RECON:PASS",
		Description: "Employee statement reconciled with zero trade discrepancies",
		Input:       OrderContext{StatementDiscrepancy: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PT_ACCESS_PERSON_RECON",
		Code:        "ACCESS_RECON:BOUNDARY",
		Description: "Confirmed reconciled feed from custodian broker",
		Input:       OrderContext{StatementDiscrepancy: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PT_ACCESS_PERSON_RECON",
		Code:        "ACCESS_RECON:FAIL",
		Description: "Discrepancy detected between trade confirmation and preclearance",
		Input:       OrderContext{StatementDiscrepancy: true},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "statement_discrepancy"},
	},
	{
		RuleCode:    "PT_ACCESS_PERSON_RECON",
		Code:        "ACCESS_RECON:ADVERSARIAL",
		Description: "Unreported external brokerage account trade discovered during monthly audit",
		Input:       OrderContext{StatementDiscrepancy: true},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "statement_discrepancy"},
	},

	// ====================================================================
	// 41. PT_BLACKOUT_PERIOD (Fair Allocation & Personal Trading)
	// ====================================================================
	{
		RuleCode:    "PT_BLACKOUT_PERIOD",
		Code:        "BLACKOUT:PASS",
		Description: "Employee trade outside blackout window",
		Input:       OrderContext{EmployeeID: &testEmpID, BlackoutActive: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PT_BLACKOUT_PERIOD",
		Code:        "BLACKOUT:BOUNDARY_CLIENT",
		Description: "Client order active during earnings blackout window (exempt)",
		Input:       OrderContext{EmployeeID: nil, BlackoutActive: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PT_BLACKOUT_PERIOD",
		Code:        "BLACKOUT:FAIL",
		Description: "Employee trade submitted during active blackout window",
		Input:       OrderContext{EmployeeID: &testEmpID, BlackoutActive: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "blackout_active"},
	},
	{
		RuleCode:    "PT_BLACKOUT_PERIOD",
		Code:        "BLACKOUT:ADVERSARIAL",
		Description: "Access person executing during earnings pre-announcement quiet period",
		Input:       OrderContext{EmployeeID: &testEmpID, BlackoutActive: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "blackout_active"},
	},

	// ====================================================================
	// 42. PT_MIN_HOLDING_30D (Fair Allocation & Personal Trading)
	// ====================================================================
	{
		RuleCode:    "PT_MIN_HOLDING_30D",
		Code:        "HOLDING_30D:PASS",
		Description: "Employee sale after 45 days holding period",
		Input:       OrderContext{EmployeeID: &testEmpID, DaysSincePurchase: 45},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PT_MIN_HOLDING_30D",
		Code:        "HOLDING_30D:BOUNDARY",
		Description: "Client sale after 5 days (client exempt)",
		Input:       OrderContext{EmployeeID: nil, DaysSincePurchase: 5},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PT_MIN_HOLDING_30D",
		Code:        "HOLDING_30D:FAIL",
		Description: "Employee sale after only 14 days breaches 30-day holding requirement",
		Input:       OrderContext{EmployeeID: &testEmpID, DaysSincePurchase: 14},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "min_holding_days"},
	},
	{
		RuleCode:    "PT_MIN_HOLDING_30D",
		Code:        "HOLDING_30D:ADVERSARIAL",
		Description: "Employee day-trade sale after 1 day",
		Input:       OrderContext{EmployeeID: &testEmpID, DaysSincePurchase: 1},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "min_holding_days"},
	},

	// ====================================================================
	// 43. SECTOR_CONCENTRATION (Concentration & Diversification)
	// ====================================================================
	{
		RuleCode:    "SECTOR_CONCENTRATION",
		Code:        "SECTOR:PASS",
		Description: "20% sector exposure under 25% cap",
		Input:       OrderContext{SectorExposurePct: d("0.200000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SECTOR_CONCENTRATION",
		Code:        "SECTOR:BOUNDARY",
		Description: "Exactly 25.000000% sector exposure passes",
		Input:       OrderContext{SectorExposurePct: d("0.250000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SECTOR_CONCENTRATION",
		Code:        "SECTOR:FAIL",
		Description: "28.5% sector exposure breaches 25% cap",
		Input:       OrderContext{SectorExposurePct: d("0.285000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_sector_pct"},
	},
	{
		RuleCode:    "SECTOR_CONCENTRATION",
		Code:        "SECTOR:ADVERSARIAL",
		Description: "Aggressive tech overweight pushing GICS sector to 38.0%",
		Input:       OrderContext{SectorExposurePct: d("0.380000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_sector_pct"},
	},

	// ====================================================================
	// 44. SETTLEMENT_FAIL_AGING (Liquidity & Settlement)
	// ====================================================================
	{
		RuleCode:    "SETTLEMENT_FAIL_AGING",
		Code:        "FAIL_AGE:PASS",
		Description: "Settlement fail aged 2 days under 3-day escalation limit",
		Input:       OrderContext{FailAgeDays: 2},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SETTLEMENT_FAIL_AGING",
		Code:        "FAIL_AGE:BOUNDARY",
		Description: "Settlement fail aged exactly 3 days passes",
		Input:       OrderContext{FailAgeDays: 3},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SETTLEMENT_FAIL_AGING",
		Code:        "FAIL_AGE:FAIL",
		Description: "Settlement fail aged 4 days triggers CSDR buy-in escalation",
		Input:       OrderContext{FailAgeDays: 4},
		Expected:    ExpectedOutcome{Status: "APPROVAL_REQUIRED", MustContain: "max_fail_days"},
	},
	{
		RuleCode:    "SETTLEMENT_FAIL_AGING",
		Code:        "FAIL_AGE:ADVERSARIAL",
		Description: "Chronic un-remedied settlement fail aged 18 days",
		Input:       OrderContext{FailAgeDays: 18},
		Expected:    ExpectedOutcome{Status: "APPROVAL_REQUIRED", MustContain: "max_fail_days"},
	},

	// ====================================================================
	// 45. SHORT_POSITION_RESTRICTION (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "SHORT_POSITION_RESTRICTION",
		Code:        "SSR:PASS",
		Description: "Short sale on security with no active SSR circuit breaker",
		Input:       OrderContext{Side: "SHORT", ShortSaleRestricted: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SHORT_POSITION_RESTRICTION",
		Code:        "SSR:BOUNDARY_LONG",
		Description: "Long sale on SSR-triggered security (long sales exempt)",
		Input:       OrderContext{Side: "SELL", ShortSaleRestricted: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SHORT_POSITION_RESTRICTION",
		Code:        "SSR:FAIL",
		Description: "Short sale order on security with active Reg SHO 201 circuit breaker",
		Input:       OrderContext{Side: "SHORT", ShortSaleRestricted: true},
		Expected:    ExpectedOutcome{Status: "APPROVAL_REQUIRED", MustContain: "short_sale_restricted"},
	},
	{
		RuleCode:    "SHORT_POSITION_RESTRICTION",
		Code:        "SSR:ADVERSARIAL",
		Description: "Short sale order disguised as long sale without locate",
		Input:       OrderContext{Side: "SHORT", ShortSaleRestricted: true},
		Expected:    ExpectedOutcome{Status: "APPROVAL_REQUIRED", MustContain: "short_sale_restricted"},
	},

	// ====================================================================
	// 46. SHORT_SALE_LOCATE (Market Abuse & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "SHORT_SALE_LOCATE",
		Code:        "LOCATE:PASS_VALID",
		Description: "Short sale with confirmed valid borrow locate",
		Input:       OrderContext{Side: "SHORT", LocateValid: true, EasyToBorrow: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SHORT_SALE_LOCATE",
		Code:        "LOCATE:BOUNDARY_ETB",
		Description: "Short sale on Easy-To-Borrow listed stock (locate exempt)",
		Input:       OrderContext{Side: "SHORT", LocateValid: false, EasyToBorrow: true},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SHORT_SALE_LOCATE",
		Code:        "LOCATE:FAIL_NAKED",
		Description: "Naked short sale with no locate on hard-to-borrow stock",
		Input:       OrderContext{Side: "SHORT", LocateValid: false, EasyToBorrow: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "short_sale_locate"},
	},
	{
		RuleCode:    "SHORT_SALE_LOCATE",
		Code:        "LOCATE:ADVERSARIAL",
		Description: "High-volume short sale without pre-borrow locate on threshold security",
		Input:       OrderContext{Side: "SHORT", LocateValid: false, EasyToBorrow: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "short_sale_locate"},
	},

	// ====================================================================
	// 47. SINGLE_POSITION_NAV (Concentration & Diversification)
	// ====================================================================
	{
		RuleCode:    "SINGLE_POSITION_NAV",
		Code:        "POS_NAV:PASS",
		Description: "7.5% NAV single position weight under 10% cap",
		Input:       OrderContext{NavWeightPct: d("0.075000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SINGLE_POSITION_NAV",
		Code:        "POS_NAV:BOUNDARY",
		Description: "Exactly 10.000000% single position weight passes",
		Input:       OrderContext{NavWeightPct: d("0.100000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SINGLE_POSITION_NAV",
		Code:        "POS_NAV:FAIL",
		Description: "12.8% single position weight breaches 10% NAV cap",
		Input:       OrderContext{NavWeightPct: d("0.128000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_single_nav_pct"},
	},
	{
		RuleCode:    "SINGLE_POSITION_NAV",
		Code:        "POS_NAV:ADVERSARIAL",
		Description: "Massive single allocation pushing asset to 22.5% of fund NAV",
		Input:       OrderContext{NavWeightPct: d("0.225000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_single_nav_pct"},
	},

	// ====================================================================
	// 48. TXN_REPORT_COMPLETENESS (Transaction Reporting)
	// ====================================================================
	{
		RuleCode:    "TXN_REPORT_COMPLETENESS",
		Code:        "RPT_COMPL:PASS",
		Description: "Complete transaction report with 0 missing fields and valid LEI",
		Input:       OrderContext{MissingFieldsCount: 0, LEIInvalid: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "TXN_REPORT_COMPLETENESS",
		Code:        "RPT_COMPL:BOUNDARY",
		Description: "Transaction report with all required tags populated",
		Input:       OrderContext{MissingFieldsCount: 0, LEIInvalid: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "TXN_REPORT_COMPLETENESS",
		Code:        "RPT_COMPL:FAIL_FIELDS",
		Description: "Transaction report missing 3 mandatory regulatory fields",
		Input:       OrderContext{MissingFieldsCount: 3, LEIInvalid: false},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "missing_fields_count"},
	},
	{
		RuleCode:    "TXN_REPORT_COMPLETENESS",
		Code:        "RPT_COMPL:FAIL_LEI",
		Description: "Transaction report with invalid / lapsed Legal Entity Identifier (LEI)",
		Input:       OrderContext{MissingFieldsCount: 0, LEIInvalid: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "lei_invalid"},
	},

	// ====================================================================
	// 49. TXN_REPORT_TIMELINESS (Transaction Reporting)
	// ====================================================================
	{
		RuleCode:    "TXN_REPORT_TIMELINESS",
		Code:        "RPT_TIME:PASS",
		Description: "Report generated 360 minutes (6h) after execution (well within T+1 1440m)",
		Input:       OrderContext{MinutesSinceExecution: 360},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "TXN_REPORT_TIMELINESS",
		Code:        "RPT_TIME:BOUNDARY",
		Description: "Report generated at exactly 1440 minutes (24h T+1 deadline) passes",
		Input:       OrderContext{MinutesSinceExecution: 1440},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "TXN_REPORT_TIMELINESS",
		Code:        "RPT_TIME:FAIL",
		Description: "Report generated at 1550 minutes breaches T+1 MiFID deadline",
		Input:       OrderContext{MinutesSinceExecution: 1550},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_reporting_delay_minutes"},
	},
	{
		RuleCode:    "TXN_REPORT_TIMELINESS",
		Code:        "RPT_TIME:ADVERSARIAL",
		Description: "Late report submitted 3 days (4320 minutes) post execution",
		Input:       OrderContext{MinutesSinceExecution: 4320},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "max_reporting_delay_minutes"},
	},

	// ====================================================================
	// 50. VENUE_APPROVED_LIST (Market Conduct & Order Surveillance)
	// ====================================================================
	{
		RuleCode:    "VENUE_APPROVED_LIST",
		Code:        "VENUE:PASS_XNYS",
		Description: "Order routed to approved venue XNYS",
		Input:       OrderContext{Venue: "XNYS"},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "VENUE_APPROVED_LIST",
		Code:        "VENUE:BOUNDARY_XPAR",
		Description: "Order routed to approved European venue XPAR",
		Input:       OrderContext{Venue: "XPAR"},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "VENUE_APPROVED_LIST",
		Code:        "VENUE:FAIL_UNREG",
		Description: "Order routed to unapproved dark pool UNREG_DARK_POOL",
		Input:       OrderContext{Venue: "UNREG_DARK_POOL"},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "approved_venues"},
	},
	{
		RuleCode:    "VENUE_APPROVED_LIST",
		Code:        "VENUE:ADVERSARIAL",
		Description: "Order routed to off-exchange broker crossing network OFF_X_UNKNOWN",
		Input:       OrderContext{Venue: "OFF_X_UNKNOWN"},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "approved_venues"},
	},

	// 51. UCITS_5_10_40 (Post-Trade 5/10/40 concentration limits)
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

	// 52. SEC_144A_QIB_HOLDING (Post-Trade 15% QIB / Illiquid Asset Limit)
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

	// 53. MARGIN_UTILIZATION_80 (Post-Trade Margin Capacity Warning)
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
}

// EvaluateScenario evaluates an OrderContext against a rule AST condition and parameter thresholds.
func EvaluateScenario(sc Scenario) (string, string) {
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

	case "UCITS_5_10_40":
		limit := d("0.400000")
		if in.UcitsAggregateAbove5PctExposure.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: ucits_aggregate_above_5pct_exposure %s exceeds max_aggregate_above_5pct_pct %s", sc.RuleCode, in.UcitsAggregateAbove5PctExposure, limit)
		}
		return "PASSED", "Compliant"

	case "SEC_144A_QIB_HOLDING":
		limit := d("0.150000")
		if in.Restricted144aExposurePct.GreaterThan(limit) {
			return "BLOCKED", fmt.Sprintf("Rule %s breached: restricted_144a_exposure_pct %s exceeds max_144a_non_qib_pct %s", sc.RuleCode, in.Restricted144aExposurePct, limit)
		}
		return "PASSED", "Compliant"

	case "MARGIN_UTILIZATION_80":
		limit := d("0.800000")
		if in.MarginUtilizationPct.GreaterThan(limit) {
			return "WARNING", fmt.Sprintf("Rule %s warning: margin_utilization_pct %s exceeds max_margin_utilization_pct %s", sc.RuleCode, in.MarginUtilizationPct, limit)
		}
		return "PASSED", "Compliant"

	default:
		return "PASSED", "Compliant"
	}
}
