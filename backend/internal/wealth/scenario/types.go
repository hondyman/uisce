package scenario

import (
"time"
)

// ScenarioType defines the type of what-if scenario
type ScenarioType string

const (
RebalanceScenario   ScenarioType = "rebalance"
TradeImpactScenario ScenarioType = "trade_impact"
TaxOptScenario      ScenarioType = "tax_optimization"
WithdrawalScenario  ScenarioType = "withdrawal"
ContributionScenario ScenarioType = "contribution"
AllocationScenario  ScenarioType = "allocation_change"
)

// ScenarioConfig configures a what-if scenario analysis
type ScenarioConfig struct {
	ScenarioID    string
	ScenarioType  ScenarioType
	PortfolioID   string
	HouseholdID   string // optional, for household-level scenarios
	AsOfDate      time.Time
	ProjectionYears int
	Currency      string
	
	// Rebalancing parameters
	RebalanceConfig *RebalanceConfig
	
	// Trade parameters
	TradeConfig *TradeConfig
	
	// Tax optimization parameters
	TaxConfig *TaxOptConfig
	
	// Cash flow parameters
	CashFlowConfig *CashFlowConfig
	
	// Allocation change parameters
	AllocationConfig *AllocationChangeConfig
}

// RebalanceConfig configures portfolio rebalancing scenario
type RebalanceConfig struct {
	TargetAllocations map[string]float64 // asset class -> target weight
	RebalanceMethod   RebalanceMethod
	Tolerance         float64 // drift tolerance before rebalancing
	MinTradeSize      float64 // minimum trade amount
	TaxAware          bool    // consider tax implications
	AvoidWashSales    bool
	PreferLongTerm    bool    // prefer long-term gains
	HarvestLosses     bool    // opportunistic tax loss harvesting
}

// RebalanceMethod defines how rebalancing is executed
type RebalanceMethod string

const (
ProportionalRebalance RebalanceMethod = "proportional"
ThresholdRebalance    RebalanceMethod = "threshold"
CalendarRebalance     RebalanceMethod = "calendar"
TaxEfficientRebalance RebalanceMethod = "tax_efficient"
)

// TradeConfig configures trade impact analysis
type TradeConfig struct {
	ProposedTrades    []ProposedTrade
	MarketImpactModel MarketImpactModel
	ExecutionStrategy ExecutionStrategy
	TimeHorizon       int // days to execute
}

// ProposedTrade represents a proposed trade for analysis
type ProposedTrade struct {
	SecurityID    string
	SecurityName  string
	TradeType     TradeType // buy, sell
	Quantity      float64
	NotionalValue float64
	LimitPrice    float64 // optional
	Urgency       TradeUrgency
}

// TradeType defines buy or sell
type TradeType string

const (
TradeBuy  TradeType = "buy"
TradeSell TradeType = "sell"
)

// TradeUrgency defines execution urgency
type TradeUrgency string

const (
UrgencyLow    TradeUrgency = "low"
UrgencyMedium TradeUrgency = "medium"
UrgencyHigh   TradeUrgency = "high"
)

// MarketImpactModel defines the market impact model
type MarketImpactModel string

const (
AlmgrenChrissModel MarketImpactModel = "almgren_chriss"
LinearImpactModel  MarketImpactModel = "linear"
SquareRootModel    MarketImpactModel = "square_root"
)

// ExecutionStrategy defines how trades are executed
type ExecutionStrategy string

const (
VWAPStrategy ExecutionStrategy = "vwap"
TWAPStrategy ExecutionStrategy = "twap"
ISStrategy   ExecutionStrategy = "implementation_shortfall"
POVStrategy  ExecutionStrategy = "participation"
)

// TaxOptConfig configures tax optimization scenario
type TaxOptConfig struct {
	TaxRates          TaxRates
	HarvestingTarget  float64 // target loss amount to harvest
	GainBudget        float64 // maximum gains to realize
	WashSaleWindow    int     // days for wash sale rule
	PreferQualified   bool    // prefer qualified dividends
	StateOfResidence  string  // for state tax calculations
	FilingStatus      string  // single, married_joint, etc.
}

// TaxRates contains applicable tax rates
type TaxRates struct {
	ShortTermRate     float64
	LongTermRate      float64
	QualifiedDivRate  float64
	OrdinaryDivRate   float64
	StateRate         float64
	NIITRate          float64 // Net Investment Income Tax
	AMTRate           float64 // Alternative Minimum Tax
}

// CashFlowConfig configures withdrawal/contribution scenarios
type CashFlowConfig struct {
	CashFlowType      CashFlowType
	Amount            float64
	Frequency         CashFlowFrequency
	StartDate         time.Time
	EndDate           time.Time
	InflationAdjusted bool
	InflationRate     float64
	SourceStrategy    WithdrawalStrategy // for withdrawals
	TargetStrategy    ContributionStrategy // for contributions
}

// CashFlowType defines cash flow direction
type CashFlowType string

const (
CashFlowWithdrawal   CashFlowType = "withdrawal"
CashFlowContribution CashFlowType = "contribution"
)

// CashFlowFrequency defines frequency of cash flows
type CashFlowFrequency string

const (
FrequencyOneTime  CashFlowFrequency = "one_time"
FrequencyMonthly  CashFlowFrequency = "monthly"
FrequencyQuarterly CashFlowFrequency = "quarterly"
FrequencyAnnual   CashFlowFrequency = "annual"
)

// WithdrawalStrategy defines how withdrawals are sourced
type WithdrawalStrategy string

const (
WithdrawProRata      WithdrawalStrategy = "pro_rata"
WithdrawTaxEfficient WithdrawalStrategy = "tax_efficient"
WithdrawFromCash     WithdrawalStrategy = "cash_first"
WithdrawFromGains    WithdrawalStrategy = "gains_first"
WithdrawFromLosses   WithdrawalStrategy = "losses_first"
)

// ContributionStrategy defines how contributions are invested
type ContributionStrategy string

const (
ContributeProRata    ContributionStrategy = "pro_rata"
ContributeUnderweight ContributionStrategy = "underweight_first"
ContributeToCash     ContributionStrategy = "cash_first"
ContributeToTarget   ContributionStrategy = "to_target"
)

// AllocationChangeConfig configures allocation change scenarios
type AllocationChangeConfig struct {
	CurrentAllocations map[string]float64
	TargetAllocations  map[string]float64
	TransitionPeriod   int // months to transition
	TransitionMethod   TransitionMethod
}

// TransitionMethod defines how allocation changes are implemented
type TransitionMethod string

const (
TransitionImmediate TransitionMethod = "immediate"
TransitionGradual   TransitionMethod = "gradual"
TransitionTaxAware  TransitionMethod = "tax_aware"
)
