package wealth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// ComplianceService handles regulatory compliance and reporting
type ComplianceService struct {
	db *pgxpool.Pool
}

// NewComplianceService creates a new compliance service
func NewComplianceService(db *pgxpool.Pool) *ComplianceService {
	return &ComplianceService{
		db: db,
	}
}

// ============================================================================
// FORM ADV GENERATION
// ============================================================================

// FormADV represents a Form ADV filing
type FormADV struct {
	FormID        string         `json:"form_id"`
	FirmID        string         `json:"firm_id"`
	FormType      string         `json:"form_type"` // INITIAL, AMENDMENT, ANNUAL_UPDATE
	FilingDate    time.Time      `json:"filing_date"`
	EffectiveDate time.Time      `json:"effective_date"`
	Part1         Part1Data      `json:"part1"`
	Part2         Part2Data      `json:"part2"`
	Schedules     []ScheduleData `json:"schedules"`
	Status        string         `json:"status"`      // DRAFT, FILED, APPROVED
	IARDNumber    string         `json:"iard_number"` // Investment Adviser Registration Depository
	CreatedAt     time.Time      `json:"created_at"`
}

// Part1Data represents Form ADV Part 1
type Part1Data struct {
	FirmName            string          `json:"firm_name"`
	FirmCRDNumber       string          `json:"firm_crd_number"`
	SECFileNumber       string          `json:"sec_file_number"`
	PrimaryBusinessName string          `json:"primary_business_name"`
	MainOfficeAddress   Address         `json:"main_office_address"`
	TotalAUM            decimal.Decimal `json:"total_aum"`
	DiscretionaryAUM    decimal.Decimal `json:"discretionary_aum"`
	NonDiscretionaryAUM decimal.Decimal `json:"non_discretionary_aum"`
	NumberOfClients     int             `json:"number_of_clients"`
	Employees           int             `json:"employees"`
	RegisteredStates    []string        `json:"registered_states"`
	Custody             bool            `json:"custody"`
}

// Part2Data represents Form ADV Part 2 (Brochure)
type Part2Data struct {
	AdvisoryBusinessSummary  string               `json:"advisory_business_summary"`
	FeesAndCompensation      FeesSection          `json:"fees_and_compensation"`
	PerformanceBasedFees     bool                 `json:"performance_based_fees"`
	TypesOfClients           []string             `json:"types_of_clients"`
	MethodsOfAnalysis        []string             `json:"methods_of_analysis"`
	InvestmentStrategies     []string             `json:"investment_strategies"`
	RiskOfLoss               string               `json:"risk_of_loss"`
	DisciplinaryInformation  []DisciplinaryEvent  `json:"disciplinary_information"`
	OtherFinancialActivities []FinancialActivity  `json:"other_financial_activities"`
	CodeOfEthics             bool                 `json:"code_of_ethics"`
	Brokerage                BrokerageSection     `json:"brokerage"`
	ClientReferrals          bool                 `json:"client_referrals"`
	Custody                  bool                 `json:"custody"`
	InvestmentDiscretion     string               `json:"investment_discretion"`
	VotingClientSecurities   bool                 `json:"voting_client_securities"`
	FinancialInformation     FinancialInfoSection `json:"financial_information"`
}

// Address represents a physical address
type Address struct {
	Street1 string `json:"street1"`
	Street2 string `json:"street2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

// FeesSection represents fee information
type FeesSection struct {
	ManagementFee           bool            `json:"management_fee"`
	ManagementFeePercentage decimal.Decimal `json:"management_fee_percentage"`
	PerformanceFee          bool            `json:"performance_fee"`
	FixedFee                bool            `json:"fixed_fee"`
	HourlyFee               bool            `json:"hourly_fee"`
	CommissionBased         bool            `json:"commission_based"`
	FeeDescription          string          `json:"fee_description"`
}

// DisciplinaryEvent represents a regulatory event
type DisciplinaryEvent struct {
	EventType   string    `json:"event_type"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Resolution  string    `json:"resolution"`
}

// FinancialActivity represents other financial industry activities
type FinancialActivity struct {
	ActivityType string `json:"activity_type"`
	Description  string `json:"description"`
}

// BrokerageSection represents brokerage practices
type BrokerageSection struct {
	SoftDollarBenefits        bool   `json:"soft_dollar_benefits"`
	BrokerageSelectionFactors string `json:"brokerage_selection_factors"`
	OrderAggregation          bool   `json:"order_aggregation"`
}

// FinancialInfoSection represents financial condition
type FinancialInfoSection struct {
	FinancialConditionImpaired bool `json:"financial_condition_impaired"`
	Bankruptcy                 bool `json:"bankruptcy"`
	BondRequired               bool `json:"bond_required"`
}

// ScheduleData represents various schedules (A, B, C, D, etc.)
type ScheduleData struct {
	ScheduleType string                 `json:"schedule_type"` // A, B, C, D
	Data         map[string]interface{} `json:"data"`
}

// GenerateFormADV generates a Form ADV filing
func (s *ComplianceService) GenerateFormADV(
	ctx context.Context,
	firmID string,
	formType string,
	effectiveDate time.Time,
) (*FormADV, error) {
	// TODO: Pull firm data from database
	// TODO: Calculate AUM totals
	// TODO: Aggregate client counts
	// TODO: Compile disciplinary history

	form := &FormADV{
		FormID:        uuid.New().String(),
		FirmID:        firmID,
		FormType:      formType,
		FilingDate:    time.Now(),
		EffectiveDate: effectiveDate,
		Status:        "DRAFT",
		CreatedAt:     time.Now(),
	}

	// Mock Part 1 data
	form.Part1 = Part1Data{
		FirmName:            "WealthVision Advisory Group",
		FirmCRDNumber:       "123456",
		SECFileNumber:       "801-12345",
		PrimaryBusinessName: "WealthVision",
		TotalAUM:            decimal.NewFromInt(5000000000), // $5B AUM
		DiscretionaryAUM:    decimal.NewFromInt(4500000000),
		NonDiscretionaryAUM: decimal.NewFromInt(500000000),
		NumberOfClients:     150,
		Employees:           45,
		RegisteredStates:    []string{"CA", "NY", "FL", "TX"},
		Custody:             false,
	}

	// Mock Part 2 data
	form.Part2 = Part2Data{
		AdvisoryBusinessSummary: "Comprehensive wealth management for UHNW families",
		FeesAndCompensation: FeesSection{
			ManagementFee:           true,
			ManagementFeePercentage: decimal.NewFromFloat(0.75),
			FeeDescription:          "0.75% annually on assets under management",
		},
		TypesOfClients:         []string{"HIGH_NET_WORTH", "ULTRA_HIGH_NET_WORTH", "PENSION_PLANS"},
		MethodsOfAnalysis:      []string{"FUNDAMENTAL", "TECHNICAL", "CYCLICAL"},
		InvestmentStrategies:   []string{"LONG_TERM_PURCHASES", "SHORT_TERM_PURCHASES", "TRADING"},
		CodeOfEthics:           true,
		InvestmentDiscretion:   "DISCRETIONARY",
		VotingClientSecurities: true,
	}

	return form, nil
}

// ============================================================================
// GIPS COMPLIANCE
// ============================================================================

// GIPSCompliance represents GIPS (Global Investment Performance Standards) compliance
type GIPSCompliance struct {
	ComplianceID     string          `json:"compliance_id"`
	FirmID           string          `json:"firm_id"`
	VerificationDate time.Time       `json:"verification_date"`
	Verifier         string          `json:"verifier"`
	ComplianceStatus string          `json:"compliance_status"` // COMPLIANT, NON_COMPLIANT, IN_REVIEW
	Composites       []GIPSComposite `json:"composites"`
	Violations       []GIPSViolation `json:"violations"`
	CreatedAt        time.Time       `json:"created_at"`
}

// GIPSComposite represents a GIPS composite
type GIPSComposite struct {
	CompositeID          string              `json:"composite_id"`
	CompositeName        string              `json:"composite_name"`
	CompositeDescription string              `json:"composite_description"`
	CreationDate         time.Time           `json:"creation_date"`
	TotalAssets          decimal.Decimal     `json:"total_assets"`
	NumberOfAccounts     int                 `json:"number_of_accounts"`
	MinimumAccountSize   decimal.Decimal     `json:"minimum_account_size"`
	PerformanceReturns   []PerformanceReturn `json:"performance_returns"`
	Dispersion           decimal.Decimal     `json:"dispersion"` // Asset-weighted std dev
	ThreeYearStdDev      decimal.Decimal     `json:"three_year_std_dev"`
}

// PerformanceReturn represents performance for a period
type PerformanceReturn struct {
	Period           string          `json:"period"` // YYYY-MM or YYYY
	GrossReturn      decimal.Decimal `json:"gross_return"`
	NetReturn        decimal.Decimal `json:"net_return"`
	BenchmarkReturn  decimal.Decimal `json:"benchmark_return"`
	NumberOfAccounts int             `json:"number_of_accounts"`
	TotalAssets      decimal.Decimal `json:"total_assets"`
}

// GIPSViolation represents a GIPS compliance violation
type GIPSViolation struct {
	ViolationID   string     `json:"violation_id"`
	ViolationType string     `json:"violation_type"`
	Description   string     `json:"description"`
	Severity      string     `json:"severity"` // MINOR, MAJOR, CRITICAL
	DetectedDate  time.Time  `json:"detected_date"`
	ResolvedDate  *time.Time `json:"resolved_date,omitempty"`
	Resolution    string     `json:"resolution,omitempty"`
}

// CheckGIPSCompliance performs GIPS compliance check
func (s *ComplianceService) CheckGIPSCompliance(
	ctx context.Context,
	firmID string,
) (*GIPSCompliance, error) {
	compliance := &GIPSCompliance{
		ComplianceID:     uuid.New().String(),
		FirmID:           firmID,
		VerificationDate: time.Now(),
		Verifier:         "Independent Third-Party Verifier",
		ComplianceStatus: "COMPLIANT",
		Violations:       []GIPSViolation{},
		CreatedAt:        time.Now(),
	}

	// TODO: Check composite construction rules
	// TODO: Validate return calculations
	// TODO: Verify disclosure requirements
	// TODO: Check verification independence

	// Mock composite data
	compliance.Composites = []GIPSComposite{
		{
			CompositeID:          uuid.New().String(),
			CompositeName:        "UHNW Balanced",
			CompositeDescription: "Balanced portfolio for UHNW clients",
			TotalAssets:          decimal.NewFromInt(1500000000),
			NumberOfAccounts:     45,
			MinimumAccountSize:   decimal.NewFromInt(25000000),
			ThreeYearStdDev:      decimal.NewFromFloat(8.5),
			Dispersion:           decimal.NewFromFloat(2.3),
		},
	}

	return compliance, nil
}

// ============================================================================
// TRADE SURVEILLANCE
// ============================================================================

// TradeSurveillanceAlert represents a trade compliance alert
type TradeSurveillanceAlert struct {
	AlertID      string       `json:"alert_id"`
	FirmID       string       `json:"firm_id"`
	AlertType    string       `json:"alert_type"` // FRONT_RUNNING, MARKET_MANIPULATION, INSIDER_TRADING, etc.
	Severity     string       `json:"severity"`   // LOW, MEDIUM, HIGH, CRITICAL
	Description  string       `json:"description"`
	TradeDetails TradeDetails `json:"trade_details"`
	Status       string       `json:"status"` // OPEN, INVESTIGATING, RESOLVED, FALSE_POSITIVE
	AssignedTo   string       `json:"assigned_to"`
	DetectedAt   time.Time    `json:"detected_at"`
	ResolvedAt   *time.Time   `json:"resolved_at,omitempty"`
	Resolution   string       `json:"resolution,omitempty"`
}

// TradeDetails represents details of a flagged trade
type TradeDetails struct {
	TradeID        string          `json:"trade_id"`
	AccountID      string          `json:"account_id"`
	Symbol         string          `json:"symbol"`
	Side           string          `json:"side"` // BUY, SELL
	Quantity       int             `json:"quantity"`
	Price          decimal.Decimal `json:"price"`
	TradeTime      time.Time       `json:"trade_time"`
	PreTradePrice  decimal.Decimal `json:"pre_trade_price"`
	PostTradePrice decimal.Decimal `json:"post_trade_price"`
}

// SurveilleTrades performs trade surveillance
func (s *ComplianceService) SurveilleTrades(
	ctx context.Context,
	firmID string,
	startDate time.Time,
	endDate time.Time,
) ([]TradeSurveillanceAlert, error) {
	var alerts []TradeSurveillanceAlert

	// TODO: Fetch trades from database
	// TODO: Check for front-running patterns
	// TODO: Detect unusual price movements
	// TODO: Check for wash sales
	// TODO: Verify best execution
	// TODO: Check for churning

	// Mock alert for demonstration
	alerts = append(alerts, TradeSurveillanceAlert{
		AlertID:     uuid.New().String(),
		FirmID:      firmID,
		AlertType:   "EXCESSIVE_TRADING",
		Severity:    "MEDIUM",
		Description: "Account shows high turnover ratio (450% annualized)",
		Status:      "OPEN",
		DetectedAt:  time.Now(),
	})

	return alerts, nil
}

// ============================================================================
// SUITABILITY ANALYSIS
// ============================================================================

// SuitabilityAnalysis represents investment suitability assessment
type SuitabilityAnalysis struct {
	AnalysisID          string                 `json:"analysis_id"`
	AccountID           string                 `json:"account_id"`
	FamilyID            string                 `json:"family_id"`
	MemberID            string                 `json:"member_id"`
	AnalysisDate        time.Time              `json:"analysis_date"`
	ClientProfile       ClientProfile          `json:"client_profile"`
	PortfolioAllocation PortfolioAllocation    `json:"portfolio_allocation"`
	SuitabilityScore    decimal.Decimal        `json:"suitability_score"`  // 0-100
	SuitabilityStatus   string                 `json:"suitability_status"` // SUITABLE, WARNING, UNSUITABLE
	Violations          []SuitabilityViolation `json:"violations"`
	Recommendations     []string               `json:"recommendations"`
	CreatedAt           time.Time              `json:"created_at"`
}

// ClientProfile represents client risk/investment profile
type ClientProfile struct {
	Age                  int             `json:"age"`
	RiskTolerance        string          `json:"risk_tolerance"`       // CONSERVATIVE, MODERATE, AGGRESSIVE
	InvestmentObjective  string          `json:"investment_objective"` // INCOME, GROWTH, PRESERVATION
	TimeHorizon          int             `json:"time_horizon"`         // Years
	LiquidityNeeds       string          `json:"liquidity_needs"`      // LOW, MODERATE, HIGH
	NetWorth             decimal.Decimal `json:"net_worth"`
	AnnualIncome         decimal.Decimal `json:"annual_income"`
	InvestmentExperience string          `json:"investment_experience"` // NONE, LIMITED, GOOD, EXTENSIVE
	ConcentrationLimits  decimal.Decimal `json:"concentration_limits"`  // Max % in single position
}

// PortfolioAllocation represents current portfolio allocation
type PortfolioAllocation struct {
	EquityPct           decimal.Decimal `json:"equity_pct"`
	FixedIncomePct      decimal.Decimal `json:"fixed_income_pct"`
	AlternativesPct     decimal.Decimal `json:"alternatives_pct"`
	CashPct             decimal.Decimal `json:"cash_pct"`
	InternationalPct    decimal.Decimal `json:"international_pct"`
	MaxSinglePosition   decimal.Decimal `json:"max_single_position"`
	AverageMaturity     int             `json:"average_maturity"` // Days for bonds
	PortfolioVolatility decimal.Decimal `json:"portfolio_volatility"`
}

// SuitabilityViolation represents a suitability rule violation
type SuitabilityViolation struct {
	ViolationType string `json:"violation_type"`
	Description   string `json:"description"`
	Severity      string `json:"severity"`
}

// AnalyzeSuitability performs suitability analysis
func (s *ComplianceService) AnalyzeSuitability(
	ctx context.Context,
	accountID string,
	familyID string,
	memberID string,
) (*SuitabilityAnalysis, error) {
	analysis := &SuitabilityAnalysis{
		AnalysisID:   uuid.New().String(),
		AccountID:    accountID,
		FamilyID:     familyID,
		MemberID:     memberID,
		AnalysisDate: time.Now(),
		CreatedAt:    time.Now(),
	}

	// TODO: Fetch client profile from database
	// TODO: Calculate current allocation
	// TODO: Check suitability rules

	// Mock client profile
	analysis.ClientProfile = ClientProfile{
		Age:                  55,
		RiskTolerance:        "MODERATE",
		InvestmentObjective:  "GROWTH",
		TimeHorizon:          15,
		LiquidityNeeds:       "LOW",
		NetWorth:             decimal.NewFromInt(50000000),
		AnnualIncome:         decimal.NewFromInt(2000000),
		InvestmentExperience: "EXTENSIVE",
		ConcentrationLimits:  decimal.NewFromFloat(10.0), // Max 10% in single position
	}

	// Mock portfolio allocation
	analysis.PortfolioAllocation = PortfolioAllocation{
		EquityPct:           decimal.NewFromFloat(60.0),
		FixedIncomePct:      decimal.NewFromFloat(25.0),
		AlternativesPct:     decimal.NewFromFloat(10.0),
		CashPct:             decimal.NewFromFloat(5.0),
		MaxSinglePosition:   decimal.NewFromFloat(8.5),
		PortfolioVolatility: decimal.NewFromFloat(12.5),
	}

	// Calculate suitability score
	analysis.SuitabilityScore = decimal.NewFromFloat(85.0) // 85/100 - suitable
	analysis.SuitabilityStatus = "SUITABLE"

	// Check for violations
	analysis.Violations = []SuitabilityViolation{}

	// Generate recommendations
	analysis.Recommendations = []string{
		"Portfolio allocation aligns with moderate risk tolerance",
		"Consider increasing international exposure to 20% for diversification",
		"Current volatility (12.5%) is appropriate for profile",
	}

	return analysis, nil
}

// ============================================================================
// AUDIT TRAIL
// ============================================================================

// AuditEntry represents an audit log entry
type AuditEntry struct {
	EntryID      string                 `json:"entry_id"`
	Timestamp    time.Time              `json:"timestamp"`
	UserID       string                 `json:"user_id"`
	UserEmail    string                 `json:"user_email"`
	Action       string                 `json:"action"`        // CREATE, UPDATE, DELETE, VIEW, EXPORT
	ResourceType string                 `json:"resource_type"` // ACCOUNT, TRADE, REPORT, etc.
	ResourceID   string                 `json:"resource_id"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Changes      map[string]interface{} `json:"changes,omitempty"` // Before/after values
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// LogAuditEntry creates an audit log entry
func (s *ComplianceService) LogAuditEntry(
	ctx context.Context,
	userID string,
	action string,
	resourceType string,
	resourceID string,
	changes map[string]interface{},
) error {
	entry := AuditEntry{
		EntryID:      uuid.New().String(),
		Timestamp:    time.Now(),
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Changes:      changes,
		Success:      true,
	}

	// TODO: Persist to database
	// TODO: Trigger compliance alerts if needed
	_ = entry // Suppress unused warning

	return nil
}

// QueryAuditTrail queries the audit trail
func (s *ComplianceService) QueryAuditTrail(
	ctx context.Context,
	filters AuditFilters,
) ([]AuditEntry, error) {
	// TODO: Query database with filters
	// TODO: Apply pagination

	var entries []AuditEntry
	return entries, nil
}

// AuditFilters represents audit trail query filters
type AuditFilters struct {
	UserID       string
	ResourceType string
	ResourceID   string
	Action       string
	StartDate    time.Time
	EndDate      time.Time
	Limit        int
	Offset       int
}
