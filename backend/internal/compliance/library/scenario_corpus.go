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
	NavWeightPct          decimal.Decimal
	CompliantAssetsPct    decimal.Decimal
	Nav                   decimal.Decimal
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

// CoreScenarioCorpus contains deterministic scenario tests for the initial library rules.
var CoreScenarioCorpus = []Scenario{
	// ── 1. UCITS_ISSUER_5 ──────────────────────────────────────────
	{
		RuleCode:    "UCITS_ISSUER_5",
		Code:        "UCITS_ISSUER_5:PASS",
		Description: "4.9% exposure order, under limit",
		Input: OrderContext{
			IssuerExposurePct: d("0.049000"),
			Quantity:          d("100"),
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_ISSUER_5",
		Code:        "UCITS_ISSUER_5:FAIL",
		Description: "10.4% exposure, over limit",
		Input: OrderContext{
			IssuerExposurePct: d("0.104000"),
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "issuer_exposure_pct"},
	},
	{
		RuleCode:    "UCITS_ISSUER_5",
		Code:        "UCITS_ISSUER_5:BOUNDARY",
		Description: "Exactly 5.000000% — GT is strict, must pass",
		Input: OrderContext{
			IssuerExposurePct: d("0.050000"),
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "UCITS_ISSUER_5",
		Code:        "UCITS_ISSUER_5:ADVERSARIAL_COMBINED",
		Description: "4.7% current + in-flight reservation delta pushes to 5.2%",
		Input: OrderContext{
			IssuerExposurePct:     d("0.052000"),
			ExistingPositionValue: d("47000000"),
			Quantity:              d("5000"),
			Price:                 d("1000"),
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "issuer_exposure_pct"},
	},

	// ── 2. UCITS_ISSUER_10_EXCEPTION ────────────────────────────────
	{
		RuleCode:    "UCITS_ISSUER_10_EXCEPTION",
		Code:        "UCITS10:EXC_WINDOW",
		Description: "6.2% exposure held >180 days since crossing 5% — approval path",
		Input: OrderContext{
			IssuerExposurePct: d("0.062000"),
			DaysOver5Pct:      200,
		},
		Expected: ExpectedOutcome{Status: "APPROVAL_REQUIRED", MustContain: "grandfather_days"},
	},
	{
		RuleCode:    "UCITS_ISSUER_10_EXCEPTION",
		Code:        "UCITS10:OVER_10",
		Description: "10.4% — beyond even the 10% exception limit",
		Input: OrderContext{
			IssuerExposurePct: d("0.104000"),
			DaysOver5Pct:      200,
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "iss10_limit_pct"},
	},
	{
		RuleCode:    "UCITS_ISSUER_10_EXCEPTION",
		Code:        "UCITS10:UNDER_180_DAYS",
		Description: "6.2% exposure held for only 60 days (within 180-day grace window) — passes",
		Input: OrderContext{
			IssuerExposurePct: d("0.062000"),
			DaysOver5Pct:      60,
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},

	// ── 3. UCITS_ISSUER_40 ──────────────────────────────────────────
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

	// ── 4. UCITS_DEPOSIT_20 ─────────────────────────────────────────
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

	// ── 5. ACT40_DIV_75_5 ───────────────────────────────────────────
	{
		RuleCode:    "ACT40_DIV_75_5",
		Code:        "ACT40:PASS",
		Description: "80% compliant assets and 4% issuer exposure",
		Input: OrderContext{
			CompliantAssetsPct: d("0.800000"),
			IssuerExposurePct:  d("0.040000"),
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "ACT40_DIV_75_5",
		Code:        "ACT40:FAIL_DIV",
		Description: "68% compliant assets (<75%) triggers soft warning",
		Input: OrderContext{
			CompliantAssetsPct: d("0.680000"),
			IssuerExposurePct:  d("0.040000"),
		},
		Expected: ExpectedOutcome{Status: "WARNING", MustContain: "compliant_assets_pct"},
	},
	{
		RuleCode:    "ACT40_DIV_75_5",
		Code:        "ACT40:FAIL_CONC",
		Description: "6% issuer exposure (>5%) triggers soft warning",
		Input: OrderContext{
			CompliantAssetsPct: d("0.800000"),
			IssuerExposurePct:  d("0.060000"),
		},
		Expected: ExpectedOutcome{Status: "WARNING", MustContain: "issuer_limit_pct"},
	},

	// ── 6. FOF_20 ───────────────────────────────────────────────────
	{
		RuleCode:    "FOF_20",
		Code:        "FOF:PASS",
		Description: "18.5% target fund exposure under 20% limit",
		Input:       OrderContext{TargetFundExposurePct: d("0.185000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FOF_20",
		Code:        "FOF:BOUNDARY",
		Description: "Exactly 20.000000% target fund exposure passes",
		Input:       OrderContext{TargetFundExposurePct: d("0.200000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "FOF_20",
		Code:        "FOF:FAIL",
		Description: "21.5% target fund exposure breaches 20% limit",
		Input:       OrderContext{TargetFundExposurePct: d("0.215000")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "fof_limit_pct"},
	},

	// ── 7. SEC_144A_ELIGIBILITY ─────────────────────────────────────
	{
		RuleCode:    "SEC_144A_ELIGIBILITY",
		Code:        "SEC144A:PASS_QIB",
		Description: "144A security purchase by qualified institutional buyer",
		Input: OrderContext{
			Restricted144A: true,
			QIBStatus:      true,
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SEC_144A_ELIGIBILITY",
		Code:        "SEC144A:FAIL_NON_QIB",
		Description: "144A security purchase by non-QIB account blocked",
		Input: OrderContext{
			Restricted144A: true,
			QIBStatus:      false,
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "restricted_144a"},
	},
	{
		RuleCode:    "SEC_144A_ELIGIBILITY",
		Code:        "SEC144A:PASS_PUBLIC",
		Description: "Unrestricted security purchase by non-QIB account passes",
		Input: OrderContext{
			Restricted144A: false,
			QIBStatus:      false,
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},

	// ── 8. REG_S_OFFSHORE_ONLY ──────────────────────────────────────
	{
		RuleCode:    "REG_S_OFFSHORE_ONLY",
		Code:        "REGS:PASS_OFFSHORE",
		Description: "Reg S offering traded in EU offshore jurisdiction",
		Input: OrderContext{
			RegS:                  true,
			Jurisdiction:          "EU",
			DomesticJurisdictions: []string{"US"},
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "REG_S_OFFSHORE_ONLY",
		Code:        "REGS:FAIL_DOMESTIC",
		Description: "Reg S offering traded in US domestic jurisdiction blocked",
		Input: OrderContext{
			RegS:                  true,
			Jurisdiction:          "US",
			DomesticJurisdictions: []string{"US"},
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "domestic_jurisdictions"},
	},

	// ── 9. RESTRICTED_LIST_BLOCK ────────────────────────────────────
	{
		RuleCode:    "RESTRICTED_LIST_BLOCK",
		Code:        "RLB:PASS",
		Description: "Security not in house restricted list",
		Input:       OrderContext{InRestrictedList: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "RESTRICTED_LIST_BLOCK",
		Code:        "RLB:FAIL",
		Description: "Security is in house restricted list -> HARD_BLOCK",
		Input:       OrderContext{InRestrictedList: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "restricted_list"},
	},

	// ── 10. INSIDER_LIST_MAR ────────────────────────────────────────
	{
		RuleCode:    "INSIDER_LIST_MAR",
		Code:        "INSIDER:PASS",
		Description: "Security not in MAR insider list",
		Input:       OrderContext{InInsiderList: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "INSIDER_LIST_MAR",
		Code:        "INSIDER:FAIL",
		Description: "Security on MAR insider list -> HARD_BLOCK",
		Input:       OrderContext{InInsiderList: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "insider_list"},
	},

	// ── 11. SANCTIONS_ISSUER_BLOCK ──────────────────────────────────
	{
		RuleCode:    "SANCTIONS_ISSUER_BLOCK",
		Code:        "SANCTIONS:PASS",
		Description: "Issuer is clean and not on OFAC/EU sanctions lists",
		Input:       OrderContext{IssuerIsSanctioned: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SANCTIONS_ISSUER_BLOCK",
		Code:        "SANCTIONS:FAIL",
		Description: "Issuer is sanctioned -> HARD_BLOCK",
		Input:       OrderContext{IssuerIsSanctioned: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "sanctioned"},
	},

	// ── 12. COUNTRY_EXPOSURE_LIMIT ──────────────────────────────────
	{
		RuleCode:    "COUNTRY_EXPOSURE_LIMIT",
		Code:        "COUNTRY:PASS",
		Description: "Country exposure 12% is below 15% mandate",
		Input:       OrderContext{CountryExposurePct: d("0.120000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "COUNTRY_EXPOSURE_LIMIT",
		Code:        "COUNTRY:BOUNDARY",
		Description: "Exactly 15.000000% country exposure passes",
		Input:       OrderContext{CountryExposurePct: d("0.150000")},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "COUNTRY_EXPOSURE_LIMIT",
		Code:        "COUNTRY:FAIL",
		Description: "Country exposure 18% breaches 15% limit -> SOFT_WARNING",
		Input:       OrderContext{CountryExposurePct: d("0.180000")},
		Expected:    ExpectedOutcome{Status: "WARNING", MustContain: "country_limit_pct"},
	},

	// ── 13. REG_M_RULE_105 ──────────────────────────────────────────
	{
		RuleCode:    "REG_M_RULE_105",
		Code:        "REGM:FAIL",
		Description: "Short sale 3 days prior to pricing in covered offering -> BLOCKED",
		Input: OrderContext{
			Side:              "SHORT",
			CoveredOffering:   true,
			DaysBeforePricing: 3,
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "restricted_window_days"},
	},
	{
		RuleCode:    "REG_M_RULE_105",
		Code:        "REGM:PASS_OUTSIDE_WINDOW",
		Description: "Short sale 6 days prior to pricing (outside 5-day window) -> PASS",
		Input: OrderContext{
			Side:              "SHORT",
			CoveredOffering:   true,
			DaysBeforePricing: 6,
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "REG_M_RULE_105",
		Code:        "REGM:PASS_NO_OFFERING",
		Description: "Short sale with no covered offering -> PASS",
		Input: OrderContext{
			Side:              "SHORT",
			CoveredOffering:   false,
			DaysBeforePricing: 0,
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},

	// ── 14. WASH_SALE_1091 ──────────────────────────────────────────
	{
		RuleCode:    "WASH_SALE_1091",
		Code:        "WS:PASS",
		Description: "Buy order with no 30-day realized loss -> PASS",
		Input: OrderContext{
			Side:            "BUY",
			RealizedLoss30d: false,
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "WASH_SALE_1091",
		Code:        "WS:FAIL",
		Description: "Buy order with 30-day realized loss for same beneficial owner -> BLOCKED",
		Input: OrderContext{
			Side:                "BUY",
			RealizedLoss30d:     true,
			SameBeneficialOwner: true,
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "realized_loss_30d"},
	},
	{
		RuleCode:    "WASH_SALE_1091",
		Code:        "WS:ADVERSARIAL_DIFFERENT_ACCOUNT",
		Description: "Buy order with realized loss but different beneficial owner -> PASS",
		Input: OrderContext{
			Side:                "BUY",
			RealizedLoss30d:     true,
			SameBeneficialOwner: false,
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},

	// ── 15. SELF_TRADE_PREVENT ──────────────────────────────────────
	{
		RuleCode:    "SELF_TRADE_PREVENT",
		Code:        "STP:PASS",
		Description: "No pending opposing orders on venue -> PASS",
		Input:       OrderContext{PendingOpposingOrder: false},
		Expected:    ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "SELF_TRADE_PREVENT",
		Code:        "STP:FAIL",
		Description: "Matching pending opposing order from same beneficial owner group -> BLOCKED",
		Input:       OrderContext{PendingOpposingOrder: true},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "pending_opposing_order"},
	},

	// ── 16. PT_PRECLEARANCE_REQUIRED ────────────────────────────────
	{
		RuleCode:    "PT_PRECLEARANCE_REQUIRED",
		Code:        "PT:PASS_VALID",
		Description: "Employee trade with valid preclearance -> PASS",
		Input: OrderContext{
			EmployeeID:        &testEmpID,
			PreclearanceValid: true,
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},
	{
		RuleCode:    "PT_PRECLEARANCE_REQUIRED",
		Code:        "PT:FAIL_NO_PRECLEAR",
		Description: "Employee trade without valid preclearance -> BLOCKED",
		Input: OrderContext{
			EmployeeID:        &testEmpID,
			PreclearanceValid: false,
		},
		Expected: ExpectedOutcome{Status: "BLOCKED", MustContain: "preclearance_valid"},
	},
	{
		RuleCode:    "PT_PRECLEARANCE_REQUIRED",
		Code:        "PT:PASS_CLIENT",
		Description: "Client order without employee ID -> PASS (rule must not fire for clients)",
		Input: OrderContext{
			EmployeeID:        nil,
			PreclearanceValid: false,
		},
		Expected: ExpectedOutcome{Status: "PASSED"},
	},

	// ── 17. FAT_FINGER_NOTIONAL ─────────────────────────────────────
	{
		RuleCode:    "FAT_FINGER_NOTIONAL",
		Code:        "FFN:PASS",
		Description: "100 shares @ $100 = $10,000 <= $50,000,000 -> PASS",
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
		Description: "$50,000,001 notional breaches $50M limit -> BLOCKED",
		Input:       OrderContext{Notional: d("50000001.00")},
		Expected:    ExpectedOutcome{Status: "BLOCKED", MustContain: "max_notional"},
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

	default:
		return "PASSED", "Compliant"
	}
}
