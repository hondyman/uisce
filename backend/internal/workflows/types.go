package workflows

import (
	"time"
)

// Phases 4-6: Workflow Integration for Front, Middle, and Back Office

// --- PHASE 4: FRONT OFFICE ---

// PreTradeRiskCheck represents a pre-trade compliance check
type PreTradeRiskCheck struct {
	PortfolioID   string            `json:"portfolio_id"`
	ProposedTrade ProposedTrade     `json:"proposed_trade"`
	CheckResults  []RiskCheckResult `json:"check_results"`
	OverallStatus string            `json:"overall_status"` // pass, warning, fail
	Timestamp     time.Time         `json:"timestamp"`
}

// ProposedTrade represents a trade being evaluated
type ProposedTrade struct {
	InstrumentID string  `json:"instrument_id"`
	Quantity     float64 `json:"quantity"`
	Direction    string  `json:"direction"` // buy, sell
	Price        float64 `json:"price,omitempty"`
}

// ProcessResult represents the result of a business process execution
type ProcessResult struct {
	ProcessID string                 `json:"process_id"`
	Status    string                 `json:"status"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// PerformanceInput defines input for performance calculation activities
type PerformanceInput struct {
	TenantID     string `json:"tenant_id"`
	InvestmentID string `json:"investment_id"`
	AsOfDate     string `json:"as_of_date,omitempty"`
}

// RiskCheckResult represents a single risk check outcome
type RiskCheckResult struct {
	CheckName    string                 `json:"check_name"`
	Status       string                 `json:"status"` // pass, warning, fail
	Message      string                 `json:"message"`
	Limit        float64                `json:"limit,omitempty"`
	CurrentVal   float64                `json:"current_value,omitempty"`
	PostTradeVal float64                `json:"post_trade_value,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// PLDriverAnalysis identifies contributors to P&L changes
type PLDriverAnalysis struct {
	PortfolioID string     `json:"portfolio_id"`
	Period      string     `json:"period"`
	TotalPL     float64    `json:"total_pl"`
	Drivers     []PLDriver `json:"drivers"`
	AsOfTime    time.Time  `json:"as_of_time"`
}

// PLDriver represents a component of P&L
type PLDriver struct {
	Category     string  `json:"category"` // price, fx, yield, etc.
	Contribution float64 `json:"contribution"`
	Percentage   float64 `json:"percentage"`
	Description  string  `json:"description"`
}

// --- PHASE 5: MIDDLE OFFICE ---

// GIPSPerformance represents GIPS-compliant performance metrics
type GIPSPerformance struct {
	CompositeID        string    `json:"composite_id"`
	Period             string    `json:"period"`
	GrossReturn        float64   `json:"gross_return"`
	NetReturn          float64   `json:"net_return"`
	BenchmarkReturn    float64   `json:"benchmark_return"`
	NumberOfPortfolios int       `json:"number_of_portfolios"`
	CompositeAssets    float64   `json:"composite_assets"`
	Dispersion         float64   `json:"dispersion,omitempty"`
	StandardDeviation  float64   `json:"standard_deviation,omitempty"`
	Methodology        string    `json:"methodology"`
	AsOfDate           time.Time `json:"as_of_date"`
}

// PeriodLinking handles multi-period return calculations
type PeriodLinking struct {
	Periods      []PeriodReturn `json:"periods"`
	LinkedReturn float64        `json:"linked_return"`
	Methodology  string         `json:"methodology"` // geometric, arithmetic
}

// PeriodReturn represents a single period's return
type PeriodReturn struct {
	StartDate time.Time  `json:"start_date"`
	EndDate   time.Time  `json:"end_date"`
	Return    float64    `json:"return"`
	CashFlows []CashFlow `json:"cash_flows,omitempty"`
}

// CashFlow represents a cash movement
type CashFlow struct {
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"`
	Type   string    `json:"type"` // contribution, withdrawal, dividend, etc.
}

// ReconciliationBreak represents a data discrepancy
type ReconciliationBreak struct {
	ID           string                 `json:"id"`
	PortfolioID  string                 `json:"portfolio_id"`
	InstrumentID string                 `json:"instrument_id,omitempty"`
	BreakType    string                 `json:"break_type"` // position, cash, price, etc.
	SourceA      string                 `json:"source_a"`
	SourceB      string                 `json:"source_b"`
	ValueA       float64                `json:"value_a"`
	ValueB       float64                `json:"value_b"`
	Difference   float64                `json:"difference"`
	Status       string                 `json:"status"` // open, investigating, resolved
	RootCause    string                 `json:"root_cause,omitempty"`
	Resolution   string                 `json:"resolution,omitempty"`
	DetectedAt   time.Time              `json:"detected_at"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// --- PHASE 6: BACK OFFICE ---

// AccountingPolicy represents an accounting treatment
type AccountingPolicy struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	PolicyType      string                 `json:"policy_type"` // accrual, amortization, fx_translation, etc.
	Description     string                 `json:"description"`
	EffectiveDate   time.Time              `json:"effective_date"`
	ApplicableFor   []string               `json:"applicable_for"` // asset classes, regions, etc.
	Parameters      map[string]interface{} `json:"parameters"`
	RegulatoryBasis string                 `json:"regulatory_basis,omitempty"` // GAAP, IFRS, etc.
}

// CorporateAction represents a corporate action event
type CorporateAction struct {
	ID                string                 `json:"id"`
	InstrumentID      string                 `json:"instrument_id"`
	ActionType        string                 `json:"action_type"` // dividend, split, merger, etc.
	ExDate            time.Time              `json:"ex_date"`
	RecordDate        time.Time              `json:"record_date"`
	PaymentDate       time.Time              `json:"payment_date,omitempty"`
	Parameters        map[string]interface{} `json:"parameters"`        // ratio, amount, etc.
	ProcessingStatus  string                 `json:"processing_status"` // pending, processed, reconciled
	AffectedPositions []string               `json:"affected_positions"`
}

// ComplianceRule represents a regulatory or firm policy rule
type ComplianceRule struct {
	ID              string                 `json:"id"`
	RuleName        string                 `json:"rule_name"`
	RuleType        string                 `json:"rule_type"`    // position_limit, trade_restriction, etc.
	Jurisdiction    string                 `json:"jurisdiction"` // US_SEC, EU_MiFID, etc.
	Description     string                 `json:"description"`
	Parameters      map[string]interface{} `json:"parameters"`
	Severity        string                 `json:"severity"` // info, warning, error, critical
	EffectiveDate   time.Time              `json:"effective_date"`
	PolicyReference string                 `json:"policy_reference,omitempty"`
}
