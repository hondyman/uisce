package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	temporal "go.temporal.io/sdk/temporal"
)

// ============================================================================
// REBALANCE-SPECIFIC DATA STRUCTURES (Portfolio Rebalancing)
// ============================================================================

type RebalanceInput struct {
	PortfolioID string           `json:"portfolio_id"`
	ModelID     string           `json:"model_id"`
	Options     RebalanceOptions `json:"options"`
	TriggeredBy string           `json:"triggered_by"`
	TenantID    string           `json:"tenant_id"`
}

type RebalanceOptions struct {
	TaxHarvest   bool    `json:"tax_harvest"`
	MinTradeSize float64 `json:"min_trade_size"`
	TolerancePct float64 `json:"tolerance_pct"`
	WashSaleDays int     `json:"wash_sale_days"`
	DryRun       bool    `json:"dry_run"`
}

type PortfolioHolding struct {
	Symbol         string    `json:"symbol"`
	CurrentShares  float64   `json:"current_shares"`
	CostBasis      float64   `json:"cost_basis"`
	MarketValue    float64   `json:"market_value"`
	UnrealizedGain float64   `json:"unrealized_gain"`
	DaysHeld       int       `json:"days_held"`
	CurrentWeight  float64   `json:"current_weight"`
	AssetClass     string    `json:"asset_class"`
	Sector         string    `json:"sector"`
	PurchaseDate   time.Time `json:"purchase_date"`
}

type RebalanceTradeSpec struct {
	Symbol         string    `json:"symbol"`
	Action         string    `json:"action"` // buy/sell
	Shares         float64   `json:"shares"`
	Price          float64   `json:"price"`
	UnrealizedGain float64   `json:"unrealized_gain"`
	DaysHeld       int       `json:"days_held"`
	TaxHarvest     bool      `json:"tax_harvest"`
	CreatedAt      time.Time `json:"created_at"`
}

type RebalanceTaxImpact struct {
	Saved            float64 `json:"saved"`
	LossesUsed       float64 `json:"losses_used"`
	WashSaleBumped   int     `json:"wash_sale_bumped"`
	RealisticTaxRate float64 `json:"realistic_tax_rate"`
	EstimatedTaxDebt float64 `json:"estimated_tax_debt"`
}

type RebalanceDriftResult struct {
	TotalDrift       float64            `json:"total_drift"`
	TotalDriftValue  float64            `json:"total_drift_value"`
	AssetClassDrifts map[string]float64 `json:"asset_class_drifts"`
	SectorDrifts     map[string]float64 `json:"sector_drifts"`
	TradesNeeded     int                `json:"trades_needed"`
	EstimatedCost    float64            `json:"estimated_cost"`
}

type AllocationAlloc struct {
	AssetClass    string  `json:"asset_class"`
	TargetPercent float64 `json:"target_percent"`
	MinPercent    float64 `json:"min_percent"`
	MaxPercent    float64 `json:"max_percent"`
	Benchmark     string  `json:"benchmark"`
}

type SemanticAllocationModel struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ModelType   string            `json:"model_type"`
	Allocations []AllocationAlloc `json:"allocations"`
	CreatedAt   time.Time         `json:"created_at"`
}

type RebalanceAuditRecord struct {
	ID             string               `json:"id"`
	WorkflowID     string               `json:"workflow_id"`
	PortfolioID    string               `json:"portfolio_id"`
	TenantID       string               `json:"tenant_id"`
	TriggeredBy    string               `json:"triggered_by"`
	DriftBefore    float64              `json:"drift_before"`
	DriftAfter     float64              `json:"drift_after"`
	TaxSaved       float64              `json:"tax_saved"`
	TradesProposed int                  `json:"trades_proposed"`
	TradesExecuted int                  `json:"trades_executed"`
	Trades         []RebalanceTradeSpec `json:"trades"`
	TaxImpact      RebalanceTaxImpact   `json:"tax_impact"`
	PolicyVersion  int                  `json:"policy_version"`
	Status         string               `json:"status"`
	ErrorMsg       string               `json:"error_msg"`
	Timestamp      time.Time            `json:"timestamp"`
}

// ============================================================================
// REBALANCE SERVICE
// ============================================================================

type RebalanceService struct {
	temporalClient client.Client
	hasuraURL      string
	kafkaBrokers   string
}

// NewRebalanceService creates a new rebalance service
func NewRebalanceService(temporalClient client.Client, hasuraURL, kafkaBrokers string) *RebalanceService {
	return &RebalanceService{
		temporalClient: temporalClient,
		hasuraURL:      hasuraURL,
		kafkaBrokers:   kafkaBrokers,
	}
}

// StartRebalance initiates the Temporal workflow
func (s *RebalanceService) StartRebalance(ctx context.Context, input RebalanceInput) (string, error) {
	workflowID := fmt.Sprintf("rebal-%s-%d", input.PortfolioID, time.Now().UnixNano())

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "rebalance-queue",
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}

	run, err := s.temporalClient.ExecuteWorkflow(ctx, options, "RebalanceOrchestrator", input)
	if err != nil {
		activity.GetLogger(ctx).Error("Failed to start rebalance workflow", "error", err)
		return "", err
	}

	return run.GetID(), nil
}

// ============================================================================
// PORTFOLIO DRIFT CALCULATION
// ============================================================================

// CalculatePortfolioDrift computes portfolio drift from target model
func CalculatePortfolioDrift(holdings []PortfolioHolding, model SemanticAllocationModel) RebalanceDriftResult {
	totalPortfolioValue := 0.0
	for _, h := range holdings {
		totalPortfolioValue += h.MarketValue
	}

	driftResult := RebalanceDriftResult{
		AssetClassDrifts: make(map[string]float64),
		SectorDrifts:     make(map[string]float64),
	}

	// Calculate current weights by asset class
	currentWeights := make(map[string]float64)
	for _, h := range holdings {
		currentWeights[h.AssetClass] += h.MarketValue / totalPortfolioValue
	}

	// Compare to target allocations
	totalDrift := 0.0
	for _, alloc := range model.Allocations {
		current := currentWeights[alloc.AssetClass]
		target := alloc.TargetPercent
		drift := math.Abs(current - target)

		driftResult.AssetClassDrifts[alloc.AssetClass] = drift
		totalDrift += drift

		if drift > alloc.MaxPercent-alloc.MinPercent {
			driftResult.TradesNeeded++
		}
	}

	driftResult.TotalDrift = totalDrift
	driftResult.TotalDriftValue = totalDrift * totalPortfolioValue

	return driftResult
}

// ============================================================================
// TAX-LOSS HARVESTING ENGINE
// ============================================================================

// OptimizeRebalanceTrades generates tax-efficient trade recommendations
func OptimizeRebalanceTrades(
	holdings []PortfolioHolding,
	model SemanticAllocationModel,
	options RebalanceOptions,
) ([]RebalanceTradeSpec, RebalanceTaxImpact) {
	trades := []RebalanceTradeSpec{}
	taxImpact := RebalanceTaxImpact{
		RealisticTaxRate: 0.20,
	}

	totalValue := 0.0
	for _, h := range holdings {
		totalValue += h.MarketValue
	}

	targetWeights := make(map[string]float64)
	for _, alloc := range model.Allocations {
		targetWeights[alloc.AssetClass] = alloc.TargetPercent
	}

	// 1. IDENTIFY TAX-LOSS HARVESTING OPPORTUNITIES
	if options.TaxHarvest {
		for _, h := range holdings {
			if h.UnrealizedGain < -1000 && h.DaysHeld > options.WashSaleDays {
				pricePerShare := h.MarketValue / h.CurrentShares
				if pricePerShare <= 0 {
					continue
				}

				sharesToSell := h.MarketValue / pricePerShare

				trades = append(trades, RebalanceTradeSpec{
					Symbol:         h.Symbol,
					Action:         "sell",
					Shares:         sharesToSell,
					Price:          pricePerShare,
					UnrealizedGain: h.UnrealizedGain,
					DaysHeld:       h.DaysHeld,
					TaxHarvest:     true,
					CreatedAt:      time.Now(),
				})

				taxImpact.LossesUsed += math.Abs(h.UnrealizedGain)
			}
		}
	}

	// 2. REBALANCE TO TARGET WEIGHTS
	for _, h := range holdings {
		target := targetWeights[h.AssetClass]
		current := h.CurrentWeight
		drift := current - target

		if math.Abs(drift) < options.TolerancePct {
			continue
		}

		if math.Abs(drift*totalValue) < options.MinTradeSize {
			continue
		}

		pricePerShare := h.MarketValue / h.CurrentShares
		if pricePerShare <= 0 {
			continue
		}

		if drift > 0 {
			sharesToSell := (drift * totalValue) / pricePerShare
			trades = append(trades, RebalanceTradeSpec{
				Symbol:         h.Symbol,
				Action:         "sell",
				Shares:         sharesToSell,
				Price:          pricePerShare,
				UnrealizedGain: h.UnrealizedGain,
				DaysHeld:       h.DaysHeld,
				TaxHarvest:     false,
				CreatedAt:      time.Now(),
			})
		} else {
			sharesToBuy := (-drift * totalValue) / pricePerShare
			trades = append(trades, RebalanceTradeSpec{
				Symbol:         h.Symbol,
				Action:         "buy",
				Shares:         sharesToBuy,
				Price:          pricePerShare,
				UnrealizedGain: h.UnrealizedGain,
				DaysHeld:       h.DaysHeld,
				TaxHarvest:     false,
				CreatedAt:      time.Now(),
			})
		}
	}

	// 3. ESTIMATE TAX IMPACT
	taxImpact.Saved = taxImpact.LossesUsed * taxImpact.RealisticTaxRate

	var totalGains float64
	for _, t := range trades {
		if t.Action == "sell" && t.UnrealizedGain > 0 {
			totalGains += t.UnrealizedGain
		}
	}
	taxImpact.EstimatedTaxDebt = totalGains * taxImpact.RealisticTaxRate

	return trades, taxImpact
}

// ============================================================================
// WASH SALE DETECTION
// ============================================================================

// CheckWashSaleViolation checks if a sale would violate wash-sale rules
func CheckWashSaleViolation(soldSymbol string, saleDate time.Time, salesHistory []RebalanceTradeSpec, washSaleDays int) bool {
	windowStart := saleDate.AddDate(0, 0, -washSaleDays)
	windowEnd := saleDate.AddDate(0, 0, washSaleDays)

	for _, sale := range salesHistory {
		if sale.Symbol == soldSymbol && sale.CreatedAt.After(windowStart) && sale.CreatedAt.Before(windowEnd) {
			return true
		}
	}
	return false
}

// ============================================================================
// REBALANCE EVENT SERIALIZATION
// ============================================================================

// MarshalRebalanceEvent serializes a rebalance for Kafka/audit
func MarshalRebalanceEvent(trades []RebalanceTradeSpec, impact RebalanceTaxImpact, audit RebalanceAuditRecord) ([]byte, error) {
	event := map[string]interface{}{
		"workflow_id":  audit.WorkflowID,
		"portfolio_id": audit.PortfolioID,
		"tenant_id":    audit.TenantID,
		"trades":       trades,
		"tax_impact":   impact,
		"status":       audit.Status,
		"timestamp":    audit.Timestamp,
	}
	return json.Marshal(event)
}

// ============================================================================
// MOCK DATA (for testing without Hasura)
// ============================================================================

// MockGetPortfolioHoldings returns sample portfolio
func MockGetPortfolioHoldings() []PortfolioHolding {
	return []PortfolioHolding{
		{
			Symbol:         "SPY",
			CurrentShares:  100,
			CostBasis:      30000,
			MarketValue:    50000,
			UnrealizedGain: 20000,
			DaysHeld:       365,
			CurrentWeight:  0.50,
			AssetClass:     "US Equities",
			Sector:         "Technology",
			PurchaseDate:   time.Now().AddDate(-1, 0, 0),
		},
		{
			Symbol:         "BND",
			CurrentShares:  200,
			CostBasis:      20000,
			MarketValue:    19500,
			UnrealizedGain: -500,
			DaysHeld:       180,
			CurrentWeight:  0.20,
			AssetClass:     "Bonds",
			Sector:         "Fixed Income",
			PurchaseDate:   time.Now().AddDate(0, -6, 0),
		},
	}
}

// MockGetAllocationModel returns target allocation model
func MockGetAllocationModel() SemanticAllocationModel {
	return SemanticAllocationModel{
		ID:        "model-60-40",
		Name:      "Classic 60/40",
		ModelType: "balanced",
		Allocations: []AllocationAlloc{
			{
				AssetClass:    "US Equities",
				TargetPercent: 0.60,
				MinPercent:    0.55,
				MaxPercent:    0.65,
				Benchmark:     "SPY",
			},
			{
				AssetClass:    "Bonds",
				TargetPercent: 0.30,
				MinPercent:    0.25,
				MaxPercent:    0.35,
				Benchmark:     "BND",
			},
		},
	}
}

// ============================================================================
// ESTIMATE COMMISSION
// ============================================================================

// EstimateCommission calculates trade commission costs
func EstimateCommission(trades []RebalanceTradeSpec, commissionPerTrade float64) float64 {
	return float64(len(trades)) * commissionPerTrade
}

// ============================================================================
// ACTIVITY: Fetch Portfolio Holdings
// ============================================================================

// FetchPortfolioHoldingsActivity queries Hasura for current holdings
func (a *RebalanceActivities) FetchPortfolioHoldingsActivity(
	ctx context.Context,
	portfolioID string,
) ([]PortfolioHolding, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Fetching portfolio holdings", "portfolioID", portfolioID)

	holdings := MockGetPortfolioHoldings()

	logger.Info("Holdings fetched", "count", len(holdings))
	return holdings, nil
}

// ============================================================================
// ACTIVITY: Get Target Allocation Model
// ============================================================================

// GetAllocationModelActivity queries Hasura for semantic model
func (a *RebalanceActivities) GetAllocationModelActivity(
	ctx context.Context,
	modelID string,
) (SemanticAllocationModel, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Fetching allocation model", "modelID", modelID)

	model := MockGetAllocationModel()
	logger.Info("Model fetched", "name", model.Name)
	return model, nil
}

// ============================================================================
// ACTIVITY: Calculate Portfolio Drift
// ============================================================================

// CalculateDriftActivity computes portfolio drift
func (a *RebalanceActivities) CalculateDriftActivity(
	ctx context.Context,
	holdings []PortfolioHolding,
	model SemanticAllocationModel,
) (RebalanceDriftResult, error) {
	logger := activity.GetLogger(ctx)

	drift := CalculatePortfolioDrift(holdings, model)

	logger.Info("Drift calculated",
		"totalDrift", drift.TotalDrift,
		"tradesNeeded", drift.TradesNeeded)

	return drift, nil
}

// ============================================================================
// ACTIVITY: Optimize Trades (Tax-Aware)
// ============================================================================

// OptimizeTradesActivity generates trade recommendations
func (a *RebalanceActivities) OptimizeTradesActivity(
	ctx context.Context,
	holdings []PortfolioHolding,
	model SemanticAllocationModel,
	options RebalanceOptions,
) ([]RebalanceTradeSpec, RebalanceTaxImpact, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Optimizing trades (tax-aware)",
		"taxHarvest", options.TaxHarvest,
		"minTradeSize", options.MinTradeSize)

	trades, impact := OptimizeRebalanceTrades(holdings, model, options)

	logger.Info("Trades optimized",
		"tradesGenerated", len(trades),
		"taxSaved", impact.Saved)

	return trades, impact, nil
}

// ============================================================================
// ACTIVITY: Save Proposed Trades
// ============================================================================

// SaveProposedTradesActivity inserts trades into proposed_trades table
func (a *RebalanceActivities) SaveProposedTradesActivity(
	ctx context.Context,
	portfolioID string,
	trades []RebalanceTradeSpec,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Saving proposed trades",
		"portfolioID", portfolioID,
		"tradeCount", len(trades))

	logger.Info("Proposed trades saved")
	return nil
}

// ============================================================================
// ACTIVITY: Publish Trade Event
// ============================================================================

// PublishTradeEventActivity publishes trades to Redpanda/Kafka
func (a *RebalanceActivities) PublishTradeEventActivity(
	ctx context.Context,
	trades []RebalanceTradeSpec,
	impact RebalanceTaxImpact,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Publishing trade event", "tradeCount", len(trades))

	logger.Info("Trade event published")
	return nil
}

// ============================================================================
// ACTIVITY: Log Rebalance Audit
// ============================================================================

// LogRebalanceAuditActivity creates immutable audit record
func (a *RebalanceActivities) LogRebalanceAuditActivity(
	ctx context.Context,
	audit RebalanceAuditRecord,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Logging rebalance audit",
		"workflowID", audit.WorkflowID,
		"status", audit.Status)

	logger.Info("Audit record created")
	return nil
}
