package scenario

import (
"time"
)

// ScenarioResult contains the full what-if scenario analysis
type ScenarioResult struct {
	Config           ScenarioConfig
	AnalysisDate     time.Time
	
	// Portfolio state
	CurrentValue     float64
	ProjectedValue   float64
	
	// Scenario-specific results
	RebalanceResult  *RebalanceResult
	TradeImpactResult *TradeImpactResult
	TaxOptResult     *TaxOptResult
	CashFlowResult   *CashFlowResult
	AllocationResult *AllocationChangeResult
	
	// Summary metrics
	Summary          *ScenarioSummary
	Recommendations  []Recommendation
}

// ScenarioSummary provides high-level summary metrics
type ScenarioSummary struct {
	TotalTrades       int
	TotalTradeValue   float64
	EstimatedCosts    float64 // commissions + impact
	TaxImpact         float64
	NetBenefit        float64
	RiskChange        float64 // change in portfolio risk
	ExpectedReturn    float64
	SharpeRatioBefore float64
	SharpeRatioAfter  float64
	BreakevenPeriod   int     // months to breakeven on costs
}

// Recommendation provides actionable advice
type Recommendation struct {
	Priority    int
	Category    string
	Title       string
	Description string
	Impact      float64
	Urgency     string
}

// RebalanceResult contains rebalancing scenario results
type RebalanceResult struct {
	CurrentAllocations  map[string]float64
	TargetAllocations   map[string]float64
	ProposedAllocations map[string]float64
	DriftAnalysis       []DriftAnalysis
	ProposedTrades      []RebalanceTrade
	TotalBuys           float64
	TotalSells          float64
	TurnoverPct         float64
	TaxCost             float64
	TransactionCosts    float64
	ExpectedTrackingError float64
}

// DriftAnalysis shows drift from target for each asset class
type DriftAnalysis struct {
	AssetClass      string
	CurrentWeight   float64
	TargetWeight    float64
	Drift           float64
	DriftPct        float64
	AbsoluteDrift   float64
	RequiredAction  string // "buy", "sell", "none"
	RequiredAmount  float64
}

// RebalanceTrade represents a single rebalancing trade
type RebalanceTrade struct {
	SecurityID      string
	SecurityName    string
	AssetClass      string
	Action          string // "buy" or "sell"
	CurrentValue    float64
	CurrentWeight   float64
	TargetValue     float64
	TargetWeight    float64
	TradeValue      float64
	Shares          float64
	TaxLots         []TaxLotImpact
	EstimatedTax    float64
	TransactionCost float64
	Rationale       string
}

// TaxLotImpact shows tax impact of selling specific lots
type TaxLotImpact struct {
	LotID           string
	AcquisitionDate time.Time
	CostBasis       float64
	CurrentValue    float64
	GainLoss        float64
	IsLongTerm      bool
	TaxRate         float64
	TaxAmount       float64
	HoldingPeriod   int // days
}

// TradeImpactResult contains trade impact analysis
type TradeImpactResult struct {
	Trades              []TradeAnalysis
	TotalMarketImpact   float64
	TotalCommissions    float64
	TotalSpreadCosts    float64
	TotalCost           float64
	CostBps             float64
	OptimalExecutionPlan *ExecutionPlan
	AlternativeStrategies []ExecutionStrategy
}

// TradeAnalysis analyzes a single proposed trade
type TradeAnalysis struct {
	Trade              ProposedTrade
	CurrentPrice       float64
	AverageDailyVolume float64
	Volatility         float64
	BidAskSpread       float64
	MarketImpact       float64
	MarketImpactBps    float64
	Commission         float64
	SpreadCost         float64
	TotalCost          float64
	TotalCostBps       float64
	DaysToExecute      float64
	RiskDuringExecution float64
	OptimalSlices      int
}

// ExecutionPlan provides optimal trade execution
type ExecutionPlan struct {
	Strategy         ExecutionStrategy
	TotalDays        int
	DailySchedule    []DailyTradeSchedule
	ExpectedCost     float64
	ExpectedRisk     float64
	OptimalTradeRate float64 // % of ADV
}

// DailyTradeSchedule shows trades for a single day
type DailyTradeSchedule struct {
	Day              int
	Date             time.Time
	Trades           []ScheduledTrade
	CumulativeProgress float64
}

// ScheduledTrade represents a scheduled trade slice
type ScheduledTrade struct {
	SecurityID    string
	TradeValue    float64
	ProgressPct   float64
	ExpectedPrice float64
	ExpectedCost  float64
}

// TaxOptResult contains tax optimization results
type TaxOptResult struct {
	CurrentYearGains    float64
	CurrentYearLosses   float64
	NetGainLoss         float64
	HarvestablePositions []HarvestablePosition
	GainPositions       []GainPosition
	Recommendations     []TaxRecommendation
	ProjectedTaxSavings float64
	WashSaleRisks       []WashSaleRisk
}

// HarvestablePosition is a position with harvestable losses
type HarvestablePosition struct {
	SecurityID       string
	SecurityName     string
	CurrentValue     float64
	CostBasis        float64
	UnrealizedLoss   float64
	IsLongTerm       bool
	TaxBenefit       float64
	ReplacementOptions []ReplacementSecurity
	WashSaleDate     time.Time // date wash sale expires
}

// ReplacementSecurity suggests replacement for harvested position
type ReplacementSecurity struct {
	SecurityID    string
	SecurityName  string
	Correlation   float64
	ExpenseRatio  float64
	TaxEfficiency float64
	Rationale     string
}

// GainPosition is a position with unrealized gains
type GainPosition struct {
	SecurityID      string
	SecurityName    string
	CurrentValue    float64
	CostBasis       float64
	UnrealizedGain  float64
	IsLongTerm      bool
	TaxCost         float64
	DaysToLongTerm  int // days until long-term treatment
}

// TaxRecommendation provides tax optimization advice
type TaxRecommendation struct {
	Action          string
	SecurityID      string
	SecurityName    string
	Amount          float64
	TaxImpact       float64
	Priority        int
	Rationale       string
	Deadline        time.Time
}

// WashSaleRisk identifies potential wash sale violations
type WashSaleRisk struct {
	SecurityID       string
	SecurityName     string
	SaleDate         time.Time
	LossAmount       float64
	ConflictingTrade string
	ConflictDate     time.Time
}

// CashFlowResult contains cash flow scenario analysis
type CashFlowResult struct {
	TotalCashFlow       float64
	CashFlowSchedule    []ScheduledCashFlow
	SourceAllocation    []CashFlowSource
	PortfolioProjection []PortfolioProjection
	SustainabilityYears float64 // years portfolio can sustain withdrawals
	DepletionDate       time.Time
	SuccessProbability  float64 // Monte Carlo success rate
}

// ScheduledCashFlow represents a single cash flow event
type ScheduledCashFlow struct {
	Date            time.Time
	Amount          float64
	InflationAdjusted float64
	CumulativeTotal float64
	PortfolioValue  float64
}

// CashFlowSource shows where cash flow is sourced
type CashFlowSource struct {
	SecurityID    string
	SecurityName  string
	AssetClass    string
	Amount        float64
	Percentage    float64
	TaxImpact     float64
	Rationale     string
}

// PortfolioProjection shows projected portfolio over time
type PortfolioProjection struct {
	Date           time.Time
	Value          float64
	CumulativeFlow float64
	Growth         float64
	Allocations    map[string]float64
}

// AllocationChangeResult contains allocation shift analysis
type AllocationChangeResult struct {
	CurrentRisk       float64
	TargetRisk        float64
	CurrentReturn     float64
	TargetReturn      float64
	TransitionPlan    []TransitionStep
	TotalTrades       int
	TotalTradeValue   float64
	TaxCost           float64
	TransactionCosts  float64
	GlidePath         []GlidePathPoint
}

// TransitionStep shows a step in allocation transition
type TransitionStep struct {
	StepNumber     int
	Date           time.Time
	Allocations    map[string]float64
	Trades         []RebalanceTrade
	CumulativeCost float64
}

// GlidePathPoint shows allocation at a point in time
type GlidePathPoint struct {
	Date        time.Time
	Allocations map[string]float64
	Risk        float64
	Return      float64
}
