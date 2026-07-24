package goldcopy

import (
	"time"

	"github.com/google/uuid"
)

// ─── Portfolio Master ─────────────────────────────────────────────────────────

// PortfolioMasterRecord is the full institutional portfolio metadata gold copy.
type PortfolioMasterRecord struct {
	ID                       uuid.UUID  `json:"id"`
	TenantID                 uuid.UUID  `json:"tenant_id"`
	CoreID                   *uuid.UUID `json:"core_id,omitempty"`
	PortfolioID              string     `json:"portfolio_id"`
	PortfolioCode            string     `json:"portfolio_code"`
	PortfolioName            string     `json:"portfolio_name"`
	PortfolioType            string     `json:"portfolio_type"`
	PortfolioCategory        string     `json:"portfolio_category,omitempty"`
	InceptionDate            time.Time  `json:"inception_date"`
	TerminationDate          *time.Time `json:"termination_date,omitempty"`
	BaseCurrency             string     `json:"base_currency"`
	Domicile                 string     `json:"domicile,omitempty"`
	LegalStructure           string     `json:"legal_structure,omitempty"`
	RegulatoryClassification string     `json:"regulatory_classification,omitempty"`
	LiquidityProfile         string     `json:"liquidity_profile,omitempty"`
	RiskProfile              string     `json:"risk_profile,omitempty"`
	InvestmentObjective      string     `json:"investment_objective,omitempty"`
	InvestmentGuidelines     string     `json:"investment_guidelines,omitempty"`
	ValuationFrequency       string     `json:"valuation_frequency,omitempty"`
	PricingSource            string     `json:"pricing_source,omitempty"`
	IsModelPortfolio         bool       `json:"is_model_portfolio"`
	IsCompositeMember        bool       `json:"is_composite_member"`
	BenchmarkID              *uuid.UUID `json:"benchmark_id,omitempty"`
	StrategyID               *uuid.UUID `json:"strategy_id,omitempty"`
	MandateID                *uuid.UUID `json:"mandate_id,omitempty"`
	CompositeID              *uuid.UUID `json:"composite_id,omitempty"`
	PerformanceSettingsID    *uuid.UUID `json:"performance_settings_id,omitempty"`
	PortfolioManagerID       string     `json:"portfolio_manager_id,omitempty"`
	CustodianID              string     `json:"custodian_id,omitempty"`
	ConfidenceScore          int        `json:"confidence_score"`
	// map of field → source system that won survivorship
	SourceSystems map[string]string `json:"source_systems"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	ValidFrom     time.Time         `json:"valid_from"`
	ValidTo       *time.Time        `json:"valid_to,omitempty"`
}

// ─── Performance Settings ─────────────────────────────────────────────────────

// PerformanceSettingsRecord is the master record for calculation policies.
type PerformanceSettingsRecord struct {
	ID                uuid.UUID         `json:"id"`
	TenantID          uuid.UUID         `json:"tenant_id"`
	CoreID            *uuid.UUID        `json:"core_id,omitempty"`
	PortfolioID       string            `json:"portfolio_id"`
	ValuationMethod   string            `json:"valuation_method"`
	FeeTreatment      string            `json:"fee_treatment"`
	CashFlowMethod    string            `json:"cash_flow_method,omitempty"`
	CurrencyHedging   string            `json:"currency_hedging_policy,omitempty"`
	LookthroughPolicy string            `json:"lookthrough_policy,omitempty"`
	DerivativesPolicy string            `json:"treatment_of_derivatives,omitempty"`
	ConfidenceScore   int               `json:"confidence_score"`
	SourceSystems     map[string]string `json:"source_systems"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	ValidFrom         time.Time         `json:"valid_from"`
	ValidTo           *time.Time        `json:"valid_to,omitempty"`
}

// ─── Raw Source Record (input to gold copy engine) ────────────────────────────

// RawPortfolioRecord is one incoming record from a single source system.
type RawPortfolioRecord struct {
	PortfolioID   string            `json:"portfolio_id"`
	SourceSystem  string            `json:"source_system"`
	EffectiveDate time.Time         `json:"effective_date"`
	QualityScore  int               `json:"quality_score"`
	Fields        map[string]string `json:"fields"` // field_name → raw value
}

// ─── Survivorship ─────────────────────────────────────────────────────────────

// SurvivorshipRule is one field-level strategy loaded from edm.survivorship_rules.
type SurvivorshipRule struct {
	EntityType       string   `json:"entity_type"`
	FieldName        string   `json:"field_name"`
	Strategy         string   `json:"strategy"` // prefer_source | earliest_non_null | latest_by | highest_quality
	PreferredSources []string `json:"preferred_sources"`
	TimeField        string   `json:"time_field,omitempty"`
	Priority         int      `json:"priority"`
}

// SurvivorshipResult is the outcome of resolving one field across all source records.
type SurvivorshipResult struct {
	FieldName       string           `json:"field_name"`
	ChosenValue     string           `json:"chosen_value"`
	ChosenSource    string           `json:"chosen_source"`
	Strategy        string           `json:"strategy"`
	RejectedSources []RejectedSource `json:"rejected_sources"`
}

// RejectedSource documents a source whose value was not chosen.
type RejectedSource struct {
	Source string `json:"source"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

// ─── DQ Validation ───────────────────────────────────────────────────────────

// DQViolation is a data quality rule failure.
type DQViolation struct {
	RuleName string `json:"rule_name"`
	Field    string `json:"field"`
	Severity string `json:"severity"` // Hard | Soft | Warning
	Message  string `json:"message"`
}

// ─── Gold Copy Run ────────────────────────────────────────────────────────────

// GoldCopyRunResult summarises one gold copy build run.
type GoldCopyRunResult struct {
	RunID           uuid.UUID              `json:"run_id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	EntityType      string                 `json:"entity_type"`
	PortfolioID     string                 `json:"portfolio_id"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     time.Time              `json:"completed_at"`
	GoldenRecord    *PortfolioMasterRecord `json:"golden_record,omitempty"`
	SurvivorshipLog []SurvivorshipResult   `json:"survivorship_log"`
	DQViolations    []DQViolation          `json:"dq_violations"`
	ConfidenceScore int                    `json:"confidence_score"`
	Success         bool                   `json:"success"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// ─── Lineage ──────────────────────────────────────────────────────────────────

// GoldCopyLineageEntry is one row written to edm.gold_copy_lineage.
type GoldCopyLineageEntry struct {
	ID                     uuid.UUID        `json:"id"`
	TenantID               uuid.UUID        `json:"tenant_id"`
	EntityType             string           `json:"entity_type"`
	EntityID               uuid.UUID        `json:"entity_id"`
	FieldName              string           `json:"field_name"`
	ChosenValue            string           `json:"chosen_value"`
	ChosenSource           string           `json:"chosen_source"`
	RejectedSources        []RejectedSource `json:"rejected_sources"`
	SurvivorshipRule       string           `json:"survivorship_rule"`
	DQRulesPassed          []string         `json:"dq_rules_passed"`
	DQRulesFailed          []string         `json:"dq_rules_failed"`
	ConfidenceContribution int              `json:"confidence_contribution"`
	RunID                  uuid.UUID        `json:"run_id"`
	CreatedAt              time.Time        `json:"created_at"`
}

// ─── Security Master ──────────────────────────────────────────────────────────

// IssuerMasterRecord is the gold copy for a legal entity that issues securities.
type IssuerMasterRecord struct {
	ID                     uuid.UUID         `json:"id"`
	TenantID               uuid.UUID         `json:"tenant_id"`
	CoreID                 *uuid.UUID        `json:"core_id,omitempty"`
	IssuerID               string            `json:"issuer_id"`
	IssuerName             string            `json:"issuer_name"`
	ShortName              string            `json:"short_name,omitempty"`
	LEI                    string            `json:"lei,omitempty"`
	CountryOfIncorporation string            `json:"country_of_incorporation,omitempty"`
	Sector                 string            `json:"sector,omitempty"`
	Industry               string            `json:"industry,omitempty"`
	RatingComposite        string            `json:"rating_composite,omitempty"`
	Status                 string            `json:"status"`
	ConfidenceScore        int               `json:"confidence_score"`
	SourceSystems          map[string]string `json:"source_systems"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
	ValidFrom              time.Time         `json:"valid_from"`
	ValidTo                *time.Time        `json:"valid_to,omitempty"`
}

// SecurityMasterRecord is the root gold copy for any financial instrument.
// Subtype-specific detail lives in the attribute structs below.
type SecurityMasterRecord struct {
	ID                       uuid.UUID         `json:"id"`
	TenantID                 uuid.UUID         `json:"tenant_id"`
	CoreID                   *uuid.UUID        `json:"core_id,omitempty"`
	SecurityID               string            `json:"security_id"`
	PrimaryIdentifier        string            `json:"primary_identifier"`
	ISIN                     string            `json:"isin,omitempty"`
	CUSIP                    string            `json:"cusip,omitempty"`
	SEDOL                    string            `json:"sedol,omitempty"`
	FIGI                     string            `json:"figi,omitempty"`
	Ticker                   string            `json:"ticker,omitempty"`
	LocalTicker              string            `json:"local_ticker,omitempty"`
	RIC                      string            `json:"ric,omitempty"`
	BBGID                    string            `json:"bbg_id,omitempty"`
	VendorIDs                map[string]string `json:"vendor_ids,omitempty"`
	SecurityName             string            `json:"security_name"`
	ShortName                string            `json:"short_name,omitempty"`
	Description              string            `json:"description,omitempty"`
	AssetClass               string            `json:"asset_class"` // Equity | FixedIncome | Fund | Derivative | FX | Commodity
	SubAssetClass            string            `json:"sub_asset_class,omitempty"`
	InstrumentType           string            `json:"instrument_type,omitempty"`
	Sector                   string            `json:"sector,omitempty"`
	Industry                 string            `json:"industry,omitempty"`
	Currency                 string            `json:"currency"`
	SettlementCurrency       string            `json:"settlement_currency,omitempty"`
	CountryOfIssue           string            `json:"country_of_issue,omitempty"`
	CountryOfRisk            string            `json:"country_of_risk,omitempty"`
	Region                   string            `json:"region,omitempty"`
	IssueDate                *time.Time        `json:"issue_date,omitempty"`
	MaturityDate             *time.Time        `json:"maturity_date,omitempty"`
	IssuerID                 *uuid.UUID        `json:"issuer_id,omitempty"`
	ListingExchange          string            `json:"listing_exchange,omitempty"`
	ExchangeCode             string            `json:"exchange_code,omitempty"`
	Status                   string            `json:"status"`
	LiquidityProfile         string            `json:"liquidity_profile,omitempty"`
	RegulatoryClassification string            `json:"regulatory_classification,omitempty"`
	// Subtype payloads — only one will be populated based on AssetClass
	FixedIncome *FixedIncomeAttributes `json:"fixed_income,omitempty"`
	Equity      *EquityAttributes      `json:"equity,omitempty"`
	Fund        *FundAttributes        `json:"fund,omitempty"`
	Derivative  *DerivativeAttributes  `json:"derivative,omitempty"`
	// Gold copy metadata
	ConfidenceScore int               `json:"confidence_score"`
	SourceSystems   map[string]string `json:"source_systems"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	ValidFrom       time.Time         `json:"valid_from"`
	ValidTo         *time.Time        `json:"valid_to,omitempty"`
}

// FixedIncomeAttributes holds bond-specific gold copy fields.
type FixedIncomeAttributes struct {
	SecurityID          uuid.UUID         `json:"security_id"`
	TenantID            uuid.UUID         `json:"tenant_id"`
	CouponType          string            `json:"coupon_type"`
	CouponRate          *float64          `json:"coupon_rate,omitempty"`
	CouponFrequency     string            `json:"coupon_frequency,omitempty"`
	DayCountConvention  string            `json:"day_count_convention,omitempty"`
	IssuePrice          *float64          `json:"issue_price,omitempty"`
	IssueSize           *float64          `json:"issue_size,omitempty"`
	ParValue            *float64          `json:"par_value,omitempty"`
	YieldToMaturity     *float64          `json:"yield_to_maturity,omitempty"`
	YieldToWorst        *float64          `json:"yield_to_worst,omitempty"`
	Seniority           string            `json:"seniority,omitempty"`
	Secured             bool              `json:"secured"`
	CollateralType      string            `json:"collateral_type,omitempty"`
	RatingAgencyRatings map[string]string `json:"rating_agency_ratings,omitempty"`
	RatingComposite     string            `json:"rating_composite,omitempty"`
}

// EquityAttributes holds stock-specific gold copy fields.
type EquityAttributes struct {
	SecurityID        uuid.UUID `json:"security_id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	ShareClass        string    `json:"share_class,omitempty"`
	SharesOutstanding *float64  `json:"shares_outstanding,omitempty"`
	FreeFloat         *float64  `json:"free_float,omitempty"`
	DividendYield     *float64  `json:"dividend_yield,omitempty"`
	DividendFrequency string    `json:"dividend_frequency,omitempty"`
}

// FundAttributes holds ETF/mutual-fund-specific gold copy fields.
type FundAttributes struct {
	SecurityID         uuid.UUID `json:"security_id"`
	TenantID           uuid.UUID `json:"tenant_id"`
	FundType           string    `json:"fund_type"`
	Domicile           string    `json:"domicile,omitempty"`
	ManagementCompany  string    `json:"management_company,omitempty"`
	Administrator      string    `json:"administrator,omitempty"`
	Custodian          string    `json:"custodian,omitempty"`
	TotalExpenseRatio  *float64  `json:"total_expense_ratio,omitempty"`
	ManagementFee      *float64  `json:"management_fee,omitempty"`
	PerformanceFee     *float64  `json:"performance_fee,omitempty"`
	DistributionPolicy string    `json:"distribution_policy,omitempty"`
	ProspectusLink     string    `json:"prospectus_link,omitempty"`
}

// DerivativeAttributes holds option/future-specific gold copy fields.
type DerivativeAttributes struct {
	SecurityID          uuid.UUID  `json:"security_id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	UnderlierSecurityID *uuid.UUID `json:"underlier_security_id,omitempty"`
	UnderlierType       string     `json:"underlier_type,omitempty"`
	ContractSize        *float64   `json:"contract_size,omitempty"`
	ContractMonth       string     `json:"contract_month,omitempty"`
	StrikePrice         *float64   `json:"strike_price,omitempty"`
	OptionType          string     `json:"option_type,omitempty"` // Call | Put
	ExerciseStyle       string     `json:"exercise_style,omitempty"`
	SettlementType      string     `json:"settlement_type,omitempty"`
	ExpiryDate          *time.Time `json:"expiry_date,omitempty"`
}

// RawSecurityRecord is one incoming record from a single source system.
type RawSecurityRecord struct {
	SecurityID    string            `json:"security_id"`
	ISIN          string            `json:"isin,omitempty"`
	CUSIP         string            `json:"cusip,omitempty"`
	FIGI          string            `json:"figi,omitempty"`
	SourceSystem  string            `json:"source_system"`
	EffectiveDate time.Time         `json:"effective_date"`
	QualityScore  int               `json:"quality_score"`
	Fields        map[string]string `json:"fields"`
}

// SecurityGoldCopyRunResult summarises one Security gold copy run.
type SecurityGoldCopyRunResult struct {
	RunID           uuid.UUID             `json:"run_id"`
	TenantID        uuid.UUID             `json:"tenant_id"`
	EntityType      string                `json:"entity_type"`
	ClusterKey      string                `json:"cluster_key"`
	StartedAt       time.Time             `json:"started_at"`
	CompletedAt     time.Time             `json:"completed_at"`
	GoldenRecord    *SecurityMasterRecord `json:"golden_record,omitempty"`
	SurvivorshipLog []SurvivorshipResult  `json:"survivorship_log"`
	DQViolations    []DQViolation         `json:"dq_violations"`
	ConfidenceScore int                   `json:"confidence_score"`
	Success         bool                  `json:"success"`
	ErrorMessage    string                `json:"error_message,omitempty"`
}
