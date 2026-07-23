package backtest

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Portfolio Domain Models
// ============================================================================

type Portfolio struct {
	ID                     uuid.UUID       `db:"id" json:"id"`
	TenantID               uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	ClientID               uuid.UUID       `db:"client_id" json:"client_id"`
	Type                   string          `db:"type" json:"type"`
	Benchmark              string          `db:"benchmark" json:"benchmark"`
	AssetAllocationTargets json.RawMessage `db:"asset_allocation_targets" json:"asset_allocation_targets"`
	PerformanceMetrics     json.RawMessage `db:"performance_metrics" json:"performance_metrics"`
	AdvisorDiscretion      bool            `db:"advisor_discretion" json:"advisor_discretion"`
	ClientApprovalRequired bool            `db:"client_approval_required" json:"client_approval_required"`
	TemplateID             *uuid.UUID      `db:"template_id" json:"template_id,omitempty"`
	CustomFields           json.RawMessage `db:"custom_fields" json:"custom_fields"`
	Holdings               []Holding       `json:"holdings,omitempty"`
	CreatedAt              time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time       `db:"updated_at" json:"updated_at"`
}

type Holding struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	PortfolioID   uuid.UUID      `db:"portfolio_id" json:"portfolio_id"`
	Symbol        string         `db:"symbol" json:"symbol"`
	Name          string         `db:"name" json:"name"`
	AssetClass    string         `db:"asset_class" json:"asset_class"`
	Quantity      float64        `db:"quantity" json:"quantity"`
	AverageCost   float64        `db:"average_cost" json:"average_cost"`
	CurrentPrice  float64        `db:"current_price" json:"current_price"`
	CurrentValue  float64        `db:"current_value" json:"current_value"`
	AllocationPct float64        `json:"allocation_pct"`
	GainLoss      float64        `json:"gain_loss"`
	GainLossPct   float64        `json:"gain_loss_pct"`
	Sector        string         `db:"sector" json:"sector"`
	Geography     string         `db:"geography" json:"geography"`
	Metrics       HoldingMetrics `json:"metrics"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
}

type HoldingMetrics struct {
	Beta          float64 `json:"beta"`
	DividendYield float64 `json:"dividend_yield"`
	PERatio       float64 `json:"pe_ratio"`
	Volatility    float64 `json:"volatility"`
	Duration      float64 `json:"duration"` // For bonds
}

// ============================================================================
// Recommendation Models
// ============================================================================

type Recommendation struct {
	ID                uuid.UUID              `db:"id" json:"id"`
	PortfolioID       uuid.UUID              `db:"portfolio_id" json:"portfolio_id"`
	CreatedBy         uuid.UUID              `db:"created_by" json:"created_by"`
	Title             string                 `db:"title" json:"title"`
	Description       string                 `db:"description" json:"description"`
	Type              string                 `db:"type" json:"type"`         // rebalance, tactical, strategic
	Status            string                 `db:"status" json:"status"`     // draft, proposed, accepted, rejected, implemented
	Priority          string                 `db:"priority" json:"priority"` // high, medium, low
	TargetAllocations []TargetAllocation     `json:"target_allocations"`
	Actions           []RecommendationAction `json:"actions"`
	Rationale         string                 `db:"rationale" json:"rationale"`
	BacktestResults   *uuid.UUID             `db:"backtest_id" json:"backtest_id"`
	RiskScore         float64                `db:"risk_score" json:"risk_score"`
	ExpectedReturn    float64                `db:"expected_return" json:"expected_return"`
	TimeHorizon       int                    `db:"time_horizon" json:"time_horizon"` // days
	Metadata          json.RawMessage        `db:"metadata" json:"metadata"`
	CreatedAt         time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time              `db:"updated_at" json:"updated_at"`
}

type TargetAllocation struct {
	Symbol            string  `json:"symbol"`
	Name              string  `json:"name"`
	AssetClass        string  `json:"asset_class"`
	CurrentAllocation float64 `json:"current_allocation"`
	TargetAllocation  float64 `json:"target_allocation"`
	AllocationChange  float64 `json:"allocation_change"`
	Rationale         string  `json:"rationale"`
}

type RecommendationAction struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"` // buy, sell, hold, swap
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	Cost     float64 `json:"cost"`
	Notes    string  `json:"notes"`
}

// ============================================================================
// Backtest Models
// ============================================================================

type BacktestRequest struct {
	RecommendationID string    `json:"recommendation_id"`
	PortfolioID      string    `json:"portfolio_id"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	SimulationDays   int       `json:"simulation_days"`
	MonteCarloCount  int       `json:"monte_carlo_count"` // Number of paths
}

type BacktestResult struct {
	ID                     uuid.UUID       `db:"id" json:"id"`
	RecommendationID       uuid.UUID       `db:"recommendation_id" json:"recommendation_id"`
	PortfolioID            uuid.UUID       `db:"portfolio_id" json:"portfolio_id"`
	SimulationType         string          `db:"simulation_type" json:"simulation_type"` // historical, monte_carlo, stress_test
	StartDate              time.Time       `db:"start_date" json:"start_date"`
	EndDate                time.Time       `db:"end_date" json:"end_date"`
	BaselineReturn         float64         `db:"baseline_return" json:"baseline_return"`
	RecommendationReturn   float64         `db:"recommendation_return" json:"recommendation_return"`
	AlphaGenerated         float64         `db:"alpha_generated" json:"alpha_generated"`
	BetaAdjustedReturn     float64         `db:"beta_adjusted_return" json:"beta_adjusted_return"`
	SharpeRatioBaseline    float64         `db:"sharpe_ratio_baseline" json:"sharpe_ratio_baseline"`
	SharpeRatioRecommended float64         `db:"sharpe_ratio_recommended" json:"sharpe_ratio_recommended"`
	MaxDrawdownBaseline    float64         `db:"max_drawdown_baseline" json:"max_drawdown_baseline"`
	MaxDrawdownRecommended float64         `db:"max_drawdown_recommended" json:"max_drawdown_recommended"`
	TaxSavingsAccumulated  float64         `db:"tax_savings_accumulated" json:"tax_savings_accumulated"`
	TransactionCosts       float64         `db:"transaction_costs" json:"transaction_costs"`
	NetBenefit             float64         `db:"net_benefit" json:"net_benefit"`
	Confidence             float64         `db:"confidence" json:"confidence"`
	SimulationData         json.RawMessage `db:"simulation_data" json:"simulation_data"`
	CreatedAt              time.Time       `db:"created_at" json:"created_at"`
}

type DailySimulation struct {
	Date                 time.Time `json:"date"`
	BaselineValue        float64   `json:"baseline_value"`
	RecommendationValue  float64   `json:"recommendation_value"`
	BaselineReturn       float64   `json:"baseline_return"`
	RecommendationReturn float64   `json:"recommendation_return"`
	AlphaAccumulated     float64   `json:"alpha_accumulated"`
	TaxEventOccurred     bool      `json:"tax_event_occurred"`
	TaxAmount            float64   `json:"tax_amount"`
	TransactionOccurred  bool      `json:"transaction_occurred"`
	TransactionAmount    float64   `json:"transaction_amount"`
	Volatility           float64   `json:"volatility"`
	DrawdownBaseline     float64   `json:"drawdown_baseline"`
	DrawdownRecommended  float64   `json:"drawdown_recommended"`
}

type MonteCarloPath struct {
	PathID      int       `json:"path_id"`
	FinalValue  float64   `json:"final_value"`
	FinalReturn float64   `json:"final_return"`
	MaxDrawdown float64   `json:"max_drawdown"`
	Percentile  float64   `json:"percentile"`
	VaR         float64   `json:"var"`  // Value at Risk
	CVaR        float64   `json:"cvar"` // Conditional Value at Risk
	SharpeRatio float64   `json:"sharpe_ratio"`
	DailyValues []float64 `json:"daily_values"`
}

type ComparisonRequest struct {
	PortfolioID       string `json:"portfolio_id"`
	RecommendationID1 string `json:"recommendation_id_1"`
	RecommendationID2 string `json:"recommendation_id_2"`
}

type ComparisonResult struct {
	ID                uuid.UUID `db:"id" json:"id"`
	PortfolioID       uuid.UUID `db:"portfolio_id" json:"portfolio_id"`
	RecommendationID1 uuid.UUID `db:"recommendation_id_1" json:"recommendation_id_1"`
	RecommendationID2 uuid.UUID `db:"recommendation_id_2" json:"recommendation_id_2"`
	Winner            string    `db:"winner" json:"winner"` // rec1, rec2, tie
	PerformanceDiff   float64   `db:"performance_diff" json:"performance_diff"`
	RiskDiff          float64   `db:"risk_diff" json:"risk_diff"`
	SharpeRatioDiff   float64   `db:"sharpe_ratio_diff" json:"sharpe_ratio_diff"`
	DrawdownDiff      float64   `db:"drawdown_diff" json:"drawdown_diff"`
	TaxDiff           float64   `db:"tax_diff" json:"tax_diff"`
	CostDiff          float64   `db:"cost_diff" json:"cost_diff"`
	Reasoning         string    `db:"reasoning" json:"reasoning"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

// ============================================================================
// Risk Analytics Models
// ============================================================================

type PortfolioRiskMetrics struct {
	ID                   uuid.UUID       `db:"id" json:"id"`
	PortfolioID          uuid.UUID       `db:"portfolio_id" json:"portfolio_id"`
	AsOfDate             time.Time       `db:"as_of_date" json:"as_of_date"`
	ExpectedReturn       float64         `db:"expected_return" json:"expected_return"`
	Volatility           float64         `db:"volatility" json:"volatility"`
	SharpeRatio          float64         `db:"sharpe_ratio" json:"sharpe_ratio"`
	SortinoRatio         float64         `db:"sortino_ratio" json:"sortino_ratio"`
	Beta                 float64         `db:"beta" json:"beta"`
	Alpha                float64         `db:"alpha" json:"alpha"`
	MaxDrawdown          float64         `db:"max_drawdown" json:"max_drawdown"`
	VaR95                float64         `db:"var_95" json:"var_95"`
	CVaR95               float64         `db:"cvar_95" json:"cvar_95"`
	DiversificationRatio float64         `db:"diversification_ratio" json:"diversification_ratio"`
	HerfindahlIndex      float64         `db:"herfindahl_index" json:"herfindahl_index"`
	Concentration        Concentration   `json:"concentration"`
	Correlation          json.RawMessage `db:"correlation_matrix" json:"correlation_matrix"`
	Metadata             json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt            time.Time       `db:"created_at" json:"created_at"`
}

type Concentration struct {
	Top10Holdings float64 `json:"top_10_holdings"`
	Top5Holdings  float64 `json:"top_5_holdings"`
	Top1Holding   float64 `json:"top_1_holding"`
}

type RiskFactor struct {
	ID           uuid.UUID `db:"id" json:"id"`
	PortfolioID  uuid.UUID `db:"portfolio_id" json:"portfolio_id"`
	FactorName   string    `db:"factor_name" json:"factor_name"` // equity, rates, credit, fx, etc.
	Exposure     float64   `db:"exposure" json:"exposure"`
	Sensitivity  float64   `db:"sensitivity" json:"sensitivity"`
	Contribution float64   `db:"contribution" json:"contribution"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// ============================================================================
// Rebalancing Models
// ============================================================================

type RebalancingPlan struct {
	ID                   uuid.UUID             `db:"id" json:"id"`
	PortfolioID          uuid.UUID             `db:"portfolio_id" json:"portfolio_id"`
	CreatedBy            uuid.UUID             `db:"created_by" json:"created_by"`
	Status               string                `db:"status" json:"status"`                     // draft, proposed, approved, executed, canceled
	RebalancingType      string                `db:"rebalancing_type" json:"rebalancing_type"` // threshold, tactical, strategic, rehedging
	TargetDeviationPct   float64               `db:"target_deviation_pct" json:"target_deviation_pct"`
	ProposedTransactions []ProposedTransaction `json:"proposed_transactions"`
	EstimatedCost        float64               `db:"estimated_cost" json:"estimated_cost"`
	EstimatedTaxImpact   float64               `db:"estimated_tax_impact" json:"estimated_tax_impact"`
	ApprovedAt           *time.Time            `db:"approved_at" json:"approved_at"`
	ExecutedAt           *time.Time            `db:"executed_at" json:"executed_at"`
	CreatedAt            time.Time             `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time             `db:"updated_at" json:"updated_at"`
}

type ProposedTransaction struct {
	ID              string  `json:"id"`
	Symbol          string  `json:"symbol"`
	Action          string  `json:"action"` // buy, sell
	CurrentHolding  float64 `json:"current_holding"`
	ProposedHolding float64 `json:"proposed_holding"`
	TransactionSize float64 `json:"transaction_size"`
	EstimatedPrice  float64 `json:"estimated_price"`
	EstimatedCost   float64 `json:"estimated_cost"`
	Priority        int     `json:"priority"`
	Rationale       string  `json:"rationale"`
}

// ============================================================================
// Response Models
// ============================================================================

type BacktestResponse struct {
	Success bool            `json:"success"`
	Data    *BacktestResult `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Message string          `json:"message,omitempty"`
}

type PortfolioResponse struct {
	Success bool       `json:"success"`
	Data    *Portfolio `json:"data,omitempty"`
	Error   string     `json:"error,omitempty"`
}

type RecommendationResponse struct {
	Success bool            `json:"success"`
	Data    *Recommendation `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type RiskMetricsResponse struct {
	Success bool                  `json:"success"`
	Data    *PortfolioRiskMetrics `json:"data,omitempty"`
	Error   string                `json:"error,omitempty"`
}

// ============================================================================
// API Request Models
// ============================================================================

type CreatePortfolioRequest struct {
	Type                   string                 `json:"type"`
	Benchmark              string                 `json:"benchmark,omitempty"`
	AssetAllocationTargets json.RawMessage        `json:"asset_allocation_targets,omitempty"`
	PerformanceMetrics     json.RawMessage        `json:"performance_metrics,omitempty"`
	AdvisorDiscretion      bool                   `json:"advisor_discretion"`
	ClientApprovalRequired bool                   `json:"client_approval_required"`
	CustomFields           json.RawMessage        `json:"custom_fields,omitempty"`
	Holdings               []CreateHoldingRequest `json:"holdings,omitempty"`
}

type CreateHoldingRequest struct {
	Symbol      string  `json:"symbol"`
	Name        string  `json:"name"`
	AssetClass  string  `json:"asset_class"`
	Quantity    float64 `json:"quantity"`
	AverageCost float64 `json:"average_cost"`
	Sector      string  `json:"sector,omitempty"`
	Geography   string  `json:"geography,omitempty"`
}

type CreateRecommendationRequest struct {
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	Type              string                 `json:"type"`
	Priority          string                 `json:"priority"`
	TargetAllocations []TargetAllocation     `json:"target_allocations"`
	Actions           []RecommendationAction `json:"actions"`
	Rationale         string                 `json:"rationale"`
	TimeHorizon       int                    `json:"time_horizon"`
}

type UpdateRecommendationStatusRequest struct {
	Status string `json:"status"`
	Notes  string `json:"notes"`
}
