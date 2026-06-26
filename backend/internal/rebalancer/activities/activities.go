package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"gonum.org/v1/gonum/mat"

	"github.com/hondyman/semlayer/backend/internal/rebalancer/engine"
)

type RebalancerActivities struct {
	db *sqlx.DB
}

func NewRebalancerActivities(db *sqlx.DB) *RebalancerActivities {
	return &RebalancerActivities{
		db: db,
	}
}

// TaxAwareOptimizeActivity builds the rebalancing plan with factor-aware replacements
func (a *RebalancerActivities) TaxAwareOptimizeActivity(ctx context.Context, tenantID, portfolioID string) (engine.Plan, error) {
	fmt.Printf("Executing TaxAwareOptimize for %s\n", portfolioID)

	// 1. Mock Data Fetching (would be from Repos)
	// drift := ...

	// 2. Mock Optimization & Factor Replacement Logic
	// 2. Run Stratified Sampling with Restrictions
	// Mock Restrictions (e.g., Client hates Tech)
	restrictions := &engine.Restrictions{
		ExcludedSectors: []string{"Tobacco"}, // Example
	}
	
	// Mock targetIndex for compilation, in a real scenario this would be fetched
	// For now, let's assume targetIndex is an empty slice of engine.Security
	targetIndex := []engine.Security{} 

	sampler := engine.NewStratifiedSampler(50)
	sampledTarget, err := sampler.Sample(targetIndex, restrictions)
	if err != nil {
		return engine.Plan{}, fmt.Errorf("sampling failed: %w", err)
	}
	fmt.Printf("Sampled Target Size (after restrictions): %d\n", len(sampledTarget))
	
	// Construct a sample plan
	plan := engine.Plan{
		ID:          fmt.Sprintf("prop_%d", time.Now().Unix()),
		PortfolioID: portfolioID,
		Explanation: "Reduce IVV overweight; harvest BND losses; buy VOO/SPY replacements",
		TEBefore:    2.0,
		TEAfter:     1.3,
		TaxImpact:   -1800.0, // Benefit
		Trades: []engine.Trade{
			{Side: "SELL", Symbol: "IVV", Qty: 50, EstValueUSD: 22500, Reason: "reduce_overweight"},
			{Side: "SELL", Symbol: "BND", Qty: 100, EstValueUSD: 7000, Reason: "harvest_loss"},
			{Side: "BUY", Symbol: "VOO", Qty: 20, EstValueUSD: 8000, Reason: "factor_replacement"},
			{Side: "BUY", Symbol: "SPY", Qty: 15, EstValueUSD: 7500, Reason: "factor_replacement"},
		},
		Citations: []engine.Citation{
			{ID: "C1", Source: "positions_snapshot", Excerpt: "IVV overweight 35% vs target 30%"},
			{ID: "C2", Source: "factor_metadata", Excerpt: "VOO/SPY factor vectors closely match IVV"},
		},
	}

	return plan, nil
}

// MonteCarloSimActivity runs the tax impact simulation
func (a *RebalancerActivities) MonteCarloSimActivity(ctx context.Context, plan engine.Plan, runs int) (engine.Plan, error) {
	fmt.Printf("Running Monte Carlo Simulation for Plan %s with %d runs\n", plan.ID, runs)
	
	summary := engine.MonteCarloSimulate(plan, runs)
	plan.MonteCarlo = summary
	
	// Simple confidence computation
	// e.g., if 80% CI is entirely negative (tax benefit), confidence is high
	if summary.Confidence80Max < 0 {
		plan.Confidence = 0.95
	} else {
		plan.Confidence = 0.70
	}

	return plan, nil
}

// NotifyAdvisorActivity pushes the proposal to the GenUI
func (a *RebalancerActivities) NotifyAdvisorActivity(ctx context.Context, tenantID string, plan engine.Plan) error {
	fmt.Printf("Pushing Proposal %s to Advisor Dashboard for Tenant %s\n", plan.ID, tenantID)
	
	// In reality, this would write to a DB table that the Frontend polls or subscribes to via WebSocket
	// For now, we just log it.
	
	return nil
}

// AnalyzePortfolio checks for drift (Legacy/Simple version)
func (a *RebalancerActivities) AnalyzePortfolio(ctx context.Context, portfolioID string) (float64, error) {
	teCalc := engine.NewTrackingErrorCalculator()
	wP := []float64{0.45, 0.20, 0.10}
	wB := []float64{0.30, 0.30, 0.20}
	covData := []float64{0.04, 0.00, 0.00, 0.00, 0.04, 0.00, 0.00, 0.00, 0.04}
	cov := mat.NewDense(3, 3, covData)
	te := teCalc.CalculateTE(wP, wB, cov)
	return te, nil
}
