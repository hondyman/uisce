package bo

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// ============================================================================
// CORE BUSINESS OBJECT TYPES
// ============================================================================

// BusinessObject represents a Workday-style business object definition
type BusinessObject struct {
	ID            string    `json:"id" db:"id"`
	TenantID      string    `json:"tenant_id" db:"tenant_id"`
	Key           string    `json:"key" db:"key"`
	Name          string    `json:"name" db:"name"`
	DisplayName   string    `json:"display_name" db:"display_name"`
	TechnicalName string    `json:"technical_name" db:"technical_name"`
	Description   string    `json:"description" db:"description"`
	Icon          string    `json:"icon" db:"icon"`
	IsCore        bool      `json:"is_core" db:"is_core"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	IsSystem      bool      `json:"is_system" db:"is_system"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`

	// Config payload (jsonb in DB) - may be used by various consumers
	Config     json.RawMessage `json:"config,omitempty" db:"config"`
	ClonesFrom string          `json:"clones_from,omitempty" db:"clones_from"`

	// Relationships
	Fields    []*BOField         `json:"fields,omitempty"`
	Layouts   []*PageLayout      `json:"layouts,omitempty"`
	Processes []*BusinessProcess `json:"processes,omitempty"`
}

// BOField represents a field within a business object
type BOField struct {
	ID               string          `json:"id" db:"id"`
	TenantID         string          `json:"tenant_id" db:"tenant_id"`
	BusinessObjectID string          `json:"business_object_id" db:"business_object_id"`
	Key              string          `json:"key" db:"key"`
	Name             string          `json:"name" db:"name"`
	DisplayName      string          `json:"display_name" db:"display_name"`
	TechnicalName    string          `json:"technical_name" db:"technical_name"`
	Type             FieldType       `json:"type" db:"type"`
	IsCore           bool            `json:"is_core" db:"is_core"`
	IsRequired       bool            `json:"is_required" db:"is_required"`
	IsReadOnly       bool            `json:"is_readonly" db:"is_readonly"`
	IsSearchable     bool            `json:"is_searchable" db:"is_searchable"`
	Description      string          `json:"description" db:"description"`
	Sequence         int             `json:"sequence" db:"sequence"`
	Section          string          `json:"section,omitempty" db:"section"`
	DefaultValue     string          `json:"default_value,omitempty" db:"default_value"`
	ValidationRules  json.RawMessage `json:"validation_rules,omitempty" db:"validation_rules"`
	ReferenceBO      string          `json:"reference_bo,omitempty" db:"reference_bo"`
	PicklistValues   pq.StringArray  `json:"picklist_values,omitempty" db:"picklist_values"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

// FieldType represents the type of a business object field
type FieldType string

const (
	FieldTypeString       FieldType = "string"
	FieldTypeText         FieldType = "text"
	FieldTypeNumber       FieldType = "number"
	FieldTypeDecimal      FieldType = "decimal"
	FieldTypeCurrency     FieldType = "currency"
	FieldTypePercentage   FieldType = "percentage"
	FieldTypeDate         FieldType = "date"
	FieldTypeDateTime     FieldType = "datetime"
	FieldTypeBoolean      FieldType = "boolean"
	FieldTypeReference    FieldType = "reference"
	FieldTypePicklist     FieldType = "picklist"
	FieldTypeJSON         FieldType = "json"
	FieldTypeUUID         FieldType = "uuid"
	FieldTypeEmail        FieldType = "email"
	FieldTypeCurrencyCode FieldType = "currency_code"
	FieldTypeCountryCode  FieldType = "country_code"
)

// ============================================================================
// FINANCIAL BUSINESS OBJECTS
// ============================================================================

// Portfolio represents an investment portfolio
type Portfolio struct {
	ID                 string          `json:"id" db:"id"`
	TenantID           string          `json:"tenant_id" db:"tenant_id"`
	Name               string          `json:"name" db:"name"`
	Code               string          `json:"code" db:"code"`
	Type               PortfolioType   `json:"type" db:"type"`
	Strategy           string          `json:"strategy,omitempty" db:"strategy"`
	BenchmarkID        *string         `json:"benchmark_id,omitempty" db:"benchmark_id"`
	InceptionDate      time.Time       `json:"inception_date" db:"inception_date"`
	Currency           string          `json:"currency" db:"currency"`
	MarketValue        *Money          `json:"market_value,omitempty"`
	CostBasis          *Money          `json:"cost_basis,omitempty"`
	UnrealizedGainLoss *Money          `json:"unrealized_gain_loss,omitempty"`
	DayChange          *Money          `json:"day_change,omitempty"`
	DayChangePercent   *float64        `json:"day_change_percent,omitempty"`
	YTDReturn          *float64        `json:"ytd_return,omitempty"`
	InceptionReturn    *float64        `json:"inception_return,omitempty"`
	RiskProfile        string          `json:"risk_profile,omitempty" db:"risk_profile"`
	TargetAllocation   json.RawMessage `json:"target_allocation,omitempty" db:"target_allocation"`
	AdvisorID          *string         `json:"advisor_id,omitempty" db:"advisor_id"`
	Custodian          string          `json:"custodian,omitempty" db:"custodian"`
	Status             string          `json:"status" db:"status"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`

	// Relationships
	Accounts  []*Account  `json:"accounts,omitempty"`
	Positions []*Position `json:"positions,omitempty"`
	Benchmark *Benchmark  `json:"benchmark,omitempty"`
}

type PortfolioType string

const (
	PortfolioTypeIndividual    PortfolioType = "individual"
	PortfolioTypeJoint         PortfolioType = "joint"
	PortfolioTypeTrust         PortfolioType = "trust"
	PortfolioTypeFoundation    PortfolioType = "foundation"
	PortfolioTypeFamilyOffice  PortfolioType = "family_office"
	PortfolioTypeInstitutional PortfolioType = "institutional"
)

// Security represents a financial instrument
type Security struct {
	ID               string       `json:"id" db:"id"`
	Ticker           string       `json:"ticker" db:"ticker"`
	CUSIP            string       `json:"cusip,omitempty" db:"cusip"`
	ISIN             string       `json:"isin,omitempty" db:"isin"`
	SEDOL            string       `json:"sedol,omitempty" db:"sedol"`
	Name             string       `json:"name" db:"name"`
	Type             SecurityType `json:"type" db:"type"`
	Subtype          string       `json:"subtype,omitempty" db:"subtype"`
	AssetClass       string       `json:"asset_class" db:"asset_class"`
	Sector           string       `json:"sector,omitempty" db:"sector"`
	Industry         string       `json:"industry,omitempty" db:"industry"`
	Exchange         string       `json:"exchange,omitempty" db:"exchange"`
	Currency         string       `json:"currency" db:"currency"`
	Country          string       `json:"country,omitempty" db:"country"`
	Price            *float64     `json:"price,omitempty"`
	PriceDate        *time.Time   `json:"price_date,omitempty"`
	DayChange        *float64     `json:"day_change,omitempty"`
	DayChangePercent *float64     `json:"day_change_percent,omitempty"`
	FiftyTwoWeekHigh *float64     `json:"fifty_two_week_high,omitempty"`
	FiftyTwoWeekLow  *float64     `json:"fifty_two_week_low,omitempty"`
	DividendYield    *float64     `json:"dividend_yield,omitempty"`
	PERatio          *float64     `json:"pe_ratio,omitempty"`
	MarketCap        *float64     `json:"market_cap,omitempty"`
	ESGScore         *float64     `json:"esg_score,omitempty"`
	// Bond-specific
	MaturityDate *time.Time `json:"maturity_date,omitempty" db:"maturity_date"`
	CouponRate   *float64   `json:"coupon_rate,omitempty" db:"coupon_rate"`
	CreditRating string     `json:"credit_rating,omitempty" db:"credit_rating"`
	// Fund-specific
	ExpenseRatio *float64  `json:"expense_ratio,omitempty" db:"expense_ratio"`
	NAV          *float64  `json:"nav,omitempty" db:"nav"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type SecurityType string

const (
	SecurityTypeEquity        SecurityType = "equity"
	SecurityTypeFixedIncome   SecurityType = "fixed_income"
	SecurityTypeAlternative   SecurityType = "alternative"
	SecurityTypeCash          SecurityType = "cash"
	SecurityTypeRealEstate    SecurityType = "real_estate"
	SecurityTypePrivateEquity SecurityType = "private_equity"
	SecurityTypeCrypto        SecurityType = "crypto"
	SecurityTypeCommodity     SecurityType = "commodity"
)

// Position represents a holding within an account
type Position struct {
	ID                        string     `json:"id" db:"id"`
	AccountID                 string     `json:"account_id" db:"account_id"`
	SecurityID                string     `json:"security_id" db:"security_id"`
	Quantity                  float64    `json:"quantity" db:"quantity"`
	CostBasis                 *Money     `json:"cost_basis,omitempty"`
	AverageCost               *Money     `json:"average_cost,omitempty"`
	MarketValue               *Money     `json:"market_value,omitempty"`
	UnrealizedGainLoss        *Money     `json:"unrealized_gain_loss,omitempty"`
	UnrealizedGainLossPercent *float64   `json:"unrealized_gain_loss_percent,omitempty"`
	DayChange                 *Money     `json:"day_change,omitempty"`
	DayChangePercent          *float64   `json:"day_change_percent,omitempty"`
	Weight                    *float64   `json:"weight,omitempty"`
	AcquiredDate              *time.Time `json:"acquired_date,omitempty" db:"acquired_date"`
	LotMethod                 string     `json:"lot_method,omitempty" db:"lot_method"`
	Currency                  string     `json:"currency" db:"currency"`
	IncomeYTD                 *Money     `json:"income_ytd,omitempty"`
	AsOfDate                  time.Time  `json:"as_of_date" db:"as_of_date"`
	CreatedAt                 time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at" db:"updated_at"`

	// Relationships
	Security *Security `json:"security,omitempty"`
	Account  *Account  `json:"account,omitempty"`
	TaxLots  []*TaxLot `json:"tax_lots,omitempty"`
}

// TaxLot represents an individual tax lot for cost basis tracking
type TaxLot struct {
	ID                  string    `json:"id" db:"id"`
	PositionID          string    `json:"position_id" db:"position_id"`
	AcquisitionDate     time.Time `json:"acquisition_date" db:"acquisition_date"`
	Quantity            float64   `json:"quantity" db:"quantity"`
	OriginalQuantity    float64   `json:"original_quantity" db:"original_quantity"`
	CostPerShare        *Money    `json:"cost_per_share,omitempty"`
	TotalCost           *Money    `json:"total_cost,omitempty"`
	MarketValue         *Money    `json:"market_value,omitempty"`
	UnrealizedGainLoss  *Money    `json:"unrealized_gain_loss,omitempty"`
	HoldingPeriod       string    `json:"holding_period,omitempty" db:"holding_period"` // short_term, long_term
	WashSaleDisallowed  *Money    `json:"wash_sale_disallowed,omitempty"`
	AdjustedCost        *Money    `json:"adjusted_cost,omitempty"`
	IsCovered           bool      `json:"is_covered" db:"is_covered"`
	SourceTransactionID *string   `json:"source_transaction_id,omitempty" db:"source_transaction_id"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// Account represents an investment account
type Account struct {
	ID               string      `json:"id" db:"id"`
	TenantID         string      `json:"tenant_id" db:"tenant_id"`
	ClientID         string      `json:"client_id" db:"client_id"`
	AccountNumber    string      `json:"account_number" db:"account_number"`
	AccountType      AccountType `json:"account_type" db:"account_type"`
	Status           string      `json:"status" db:"status"`
	Balance          *Money      `json:"balance,omitempty"`
	OpenDate         *time.Time  `json:"open_date,omitempty" db:"open_date"`
	CloseDate        *time.Time  `json:"close_date,omitempty" db:"close_date"`
	Custodian        string      `json:"custodian,omitempty" db:"custodian"`
	CustodianAcctNum string      `json:"custodian_acct_num,omitempty" db:"custodian_acct_num"`
	TaxStatus        string      `json:"tax_status,omitempty" db:"tax_status"` // taxable, tax_deferred, tax_exempt
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at" db:"updated_at"`

	// Relationships
	Positions    []*Position    `json:"positions,omitempty"`
	Transactions []*Transaction `json:"transactions,omitempty"`
}

type AccountType string

const (
	AccountTypeIndividual AccountType = "individual"
	AccountTypeJoint      AccountType = "joint"
	AccountTypeIRA        AccountType = "ira"
	AccountTypeRothIRA    AccountType = "roth_ira"
	AccountType401k       AccountType = "401k"
	AccountTypeTrust      AccountType = "trust"
	AccountTypeCorporate  AccountType = "corporate"
	AccountTypeCustodial  AccountType = "custodial"
)

// Transaction represents a financial transaction
type Transaction struct {
	ID             string          `json:"id" db:"id"`
	AccountID      string          `json:"account_id" db:"account_id"`
	SecurityID     *string         `json:"security_id,omitempty" db:"security_id"`
	Type           TransactionType `json:"type" db:"type"`
	Subtype        string          `json:"subtype,omitempty" db:"subtype"`
	TradeDate      time.Time       `json:"trade_date" db:"trade_date"`
	SettlementDate *time.Time      `json:"settlement_date,omitempty" db:"settlement_date"`
	Quantity       *float64        `json:"quantity,omitempty" db:"quantity"`
	Price          *Money          `json:"price,omitempty"`
	Amount         *Money          `json:"amount,omitempty"`
	Fees           *Money          `json:"fees,omitempty"`
	NetAmount      *Money          `json:"net_amount,omitempty"`
	Currency       string          `json:"currency" db:"currency"`
	Description    string          `json:"description,omitempty" db:"description"`
	Status         string          `json:"status" db:"status"`
	ExternalID     string          `json:"external_id,omitempty" db:"external_id"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`

	// Relationships
	Security *Security `json:"security,omitempty"`
}

type TransactionType string

const (
	TransactionTypeBuy         TransactionType = "buy"
	TransactionTypeSell        TransactionType = "sell"
	TransactionTypeDividend    TransactionType = "dividend"
	TransactionTypeInterest    TransactionType = "interest"
	TransactionTypeDeposit     TransactionType = "deposit"
	TransactionTypeWithdrawal  TransactionType = "withdrawal"
	TransactionTypeFee         TransactionType = "fee"
	TransactionTypeTransferIn  TransactionType = "transfer_in"
	TransactionTypeTransferOut TransactionType = "transfer_out"
	TransactionTypeSplit       TransactionType = "split"
	TransactionTypeSpinoff     TransactionType = "spinoff"
	TransactionTypeMerger      TransactionType = "merger"
)

// Benchmark represents a performance benchmark
type Benchmark struct {
	ID              string          `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	Code            string          `json:"code" db:"code"`
	Type            string          `json:"type" db:"type"` // index, blended, custom
	Description     string          `json:"description,omitempty" db:"description"`
	Currency        string          `json:"currency" db:"currency"`
	Components      json.RawMessage `json:"components,omitempty" db:"components"`
	MTDReturn       *float64        `json:"mtd_return,omitempty"`
	QTDReturn       *float64        `json:"qtd_return,omitempty"`
	YTDReturn       *float64        `json:"ytd_return,omitempty"`
	OneYearReturn   *float64        `json:"one_year_return,omitempty"`
	ThreeYearReturn *float64        `json:"three_year_return,omitempty"`
	FiveYearReturn  *float64        `json:"five_year_return,omitempty"`
	Volatility      *float64        `json:"volatility,omitempty"`
	SharpeRatio     *float64        `json:"sharpe_ratio,omitempty"`
	IsActive        bool            `json:"is_active" db:"is_active"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// Performance represents portfolio performance metrics
type Performance struct {
	ID                 string    `json:"id" db:"id"`
	PortfolioID        string    `json:"portfolio_id" db:"portfolio_id"`
	AsOfDate           time.Time `json:"as_of_date" db:"as_of_date"`
	Period             string    `json:"period" db:"period"` // mtd, qtd, ytd, 1y, 3y, 5y, 10y, itd
	ReturnTWR          *float64  `json:"return_twr,omitempty"`
	ReturnMWR          *float64  `json:"return_mwr,omitempty"`
	BenchmarkReturn    *float64  `json:"benchmark_return,omitempty"`
	Alpha              *float64  `json:"alpha,omitempty"`
	Beta               *float64  `json:"beta,omitempty"`
	SharpeRatio        *float64  `json:"sharpe_ratio,omitempty"`
	SortinoRatio       *float64  `json:"sortino_ratio,omitempty"`
	MaxDrawdown        *float64  `json:"max_drawdown,omitempty"`
	Volatility         *float64  `json:"volatility,omitempty"`
	TrackingError      *float64  `json:"tracking_error,omitempty"`
	InformationRatio   *float64  `json:"information_ratio,omitempty"`
	UpsideCapture      *float64  `json:"upside_capture,omitempty"`
	DownsideCapture    *float64  `json:"downside_capture,omitempty"`
	BeginningValue     *Money    `json:"beginning_value,omitempty"`
	EndingValue        *Money    `json:"ending_value,omitempty"`
	NetContributions   *Money    `json:"net_contributions,omitempty"`
	Income             *Money    `json:"income,omitempty"`
	RealizedGainLoss   *Money    `json:"realized_gain_loss,omitempty"`
	UnrealizedGainLoss *Money    `json:"unrealized_gain_loss,omitempty"`
	Fees               *Money    `json:"fees,omitempty"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

// Money represents a monetary value with currency
type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// ============================================================================
// BUSINESS PROCESS TYPES
// ============================================================================

// BusinessProcess represents a Workday-style business process
type BusinessProcess struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	Key         string    `json:"key" db:"key"`
	Name        string    `json:"name" db:"name"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Description string    `json:"description,omitempty" db:"description"`
	Category    string    `json:"category" db:"category"`
	Status      string    `json:"status" db:"status"` // draft, active, inactive
	Version     int       `json:"version" db:"version"`
	IsSystem    bool      `json:"is_system" db:"is_system"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Relationships
	Steps     []*ProcessStep     `json:"steps,omitempty"`
	Instances []*ProcessInstance `json:"instances,omitempty"`
}

// ProcessStep represents a step in a business process
type ProcessStep struct {
	ID          string          `json:"id" db:"id"`
	TenantID    string          `json:"tenant_id" db:"tenant_id"`
	ProcessID   string          `json:"process_id" db:"process_id"`
	Key         string          `json:"key" db:"key"`
	Name        string          `json:"name" db:"name"`
	DisplayName string          `json:"display_name" db:"display_name"`
	StepType    string          `json:"step_type" db:"step_type"` // initiate, validate, approve, etc.
	Sequence    int             `json:"sequence" db:"sequence"`
	Config      json.RawMessage `json:"config" db:"config"`
	IsRequired  bool            `json:"is_required" db:"is_required"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// ProcessInstance represents a running instance of a business process
type ProcessInstance struct {
	ID            string          `json:"id" db:"id"`
	TenantID      string          `json:"tenant_id" db:"tenant_id"`
	ProcessID     string          `json:"process_id" db:"process_id"`
	EntityType    string          `json:"entity_type" db:"entity_type"` // client, portfolio, etc.
	EntityID      string          `json:"entity_id" db:"entity_id"`
	CurrentStepID *string         `json:"current_step_id,omitempty" db:"current_step_id"`
	Status        string          `json:"status" db:"status"` // pending, in_progress, completed, failed, cancelled
	StartedAt     time.Time       `json:"started_at" db:"started_at"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	Data          json.RawMessage `json:"data,omitempty" db:"data"`
	CreatedBy     string          `json:"created_by" db:"created_by"`

	// Relationships
	Process *BusinessProcess `json:"process,omitempty"`
	History []*StepHistory   `json:"history,omitempty"`
}

// StepHistory represents the history of a step execution
type StepHistory struct {
	ID         string          `json:"id" db:"id"`
	InstanceID string          `json:"instance_id" db:"instance_id"`
	StepID     string          `json:"step_id" db:"step_id"`
	Action     string          `json:"action" db:"action"` // started, completed, approved, rejected, skipped
	Actor      string          `json:"actor" db:"actor"`
	Comments   string          `json:"comments,omitempty" db:"comments"`
	Data       json.RawMessage `json:"data,omitempty" db:"data"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

// PageLayout represents a UI layout for a business object
type PageLayout struct {
	ID          string          `json:"id" db:"id"`
	TenantID    string          `json:"tenant_id" db:"tenant_id"`
	BOID        string          `json:"bo_id" db:"bo_id"`
	Name        string          `json:"name" db:"name"`
	Type        string          `json:"type" db:"type"` // form, grid, detail, wizard
	Description string          `json:"description,omitempty" db:"description"`
	IsDefault   bool            `json:"is_default" db:"is_default"`
	IsActive    bool            `json:"is_active" db:"is_active"`
	Config      json.RawMessage `json:"config,omitempty" db:"config"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`

	// Relationships
	Sections []*LayoutSection `json:"sections,omitempty"`
	Actions  []*LayoutAction  `json:"actions,omitempty"`
}

// LayoutSection represents a section within a layout
type LayoutSection struct {
	ID          string   `json:"id" db:"id"`
	LayoutID    string   `json:"layout_id" db:"layout_id"`
	Title       string   `json:"title" db:"title"`
	Order       int      `json:"order" db:"order"`
	Columns     int      `json:"columns" db:"columns"`
	Collapsible bool     `json:"collapsible" db:"collapsible"`
	FieldIDs    []string `json:"field_ids" db:"field_ids"`
}

// LayoutAction represents an action button in a layout
type LayoutAction struct {
	ID        string `json:"id" db:"id"`
	LayoutID  string `json:"layout_id" db:"layout_id"`
	Label     string `json:"label" db:"label"`
	Type      string `json:"type" db:"type"`   // save, submit, cancel, custom
	Style     string `json:"style" db:"style"` // primary, secondary, danger
	Order     int    `json:"order" db:"order"`
	ProcessID string `json:"process_id,omitempty" db:"process_id"` // Triggers this process
}
