package wealth

import (
	"context"
	"time"

	"github.com/hondyman/semlayer/backend/pkg/custodian"
	"github.com/hondyman/semlayer/backend/pkg/optimizer"
	"github.com/hondyman/semlayer/backend/pkg/saga"
	"go.temporal.io/sdk/activity"
)

// FetchRebalanceInputsActivity retrieves portfolio data and market prices
func (a *WealthActivities) FetchRebalanceInputsActivity(ctx context.Context, portfolioID string, tenantID string) (optimizer.Inputs, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Fetching rebalance inputs", "PortfolioID", portfolioID)

	// Mock data for demo
	// In production: query database for positions, market data provider for prices
	
	inputs := optimizer.Inputs{
		Drift: optimizer.DriftReport{
			PortfolioID:   portfolioID,
			Timestamp:     time.Now(),
			DriftPercent:  3.5,
			TrackingError: 2.1,
			Exposures: []optimizer.Exposure{
				{Symbol: "IVV", CurrentWgt: 0.35, TargetWgt: 0.30, MarketValue: 350000, Sector: "Equity"},
				{Symbol: "BND", CurrentWgt: 0.35, TargetWgt: 0.40, MarketValue: 350000, Sector: "Fixed Income"},
			},
		},
		Lots: []optimizer.Lot{
			{LotID: "l1", Symbol: "IVV", Quantity: 100, CostBasis: 400, MarketPrice: 450, AcquiredAt: time.Now().AddDate(-2, 0, 0), AccountType: "taxable", UnrealizedPNL: 5000, Term: "long"},
			{LotID: "l2", Symbol: "BND", Quantity: 200, CostBasis: 80, MarketPrice: 70, AcquiredAt: time.Now().AddDate(0, -6, 0), AccountType: "taxable", UnrealizedPNL: -2000, Term: "short"},
		},
		Prices: map[string]float64{
			"IVV": 450.0,
			"BND": 70.0,
			"AGG": 98.0, // replacement
		},
		Rules: optimizer.TaxRules{
			WashSaleDays:            30,
			ShortTermPenaltyWeight:  0.5,
			TransactionCostPerShare: 0.01,
			HarvestBudgetUSD:        5000,
			MinTradeUSD:             1000,
			AllowedReplacementMap: map[string][]string{
				"BND": {"AGG"},
			},
		},
		Weights: optimizer.ScoreWeights{
			TEWeight:         10.0,
			TaxAlphaWeight:   1.0,
			TransCostWeight:  5.0,
			ShortTermPenalty: 2.0,
		},
	}

	return inputs, nil
}

// RunOptimizerActivity executes the rebalancing optimizer
func (a *WealthActivities) RunOptimizerActivity(ctx context.Context, inputs optimizer.Inputs) (optimizer.Plan, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running optimizer")

	plan := optimizer.Optimize(inputs)
	return plan, nil
}

// CheckAutonomyActivity evaluates if the plan can be executed autonomously
func (a *WealthActivities) CheckAutonomyActivity(ctx context.Context, plan optimizer.Plan, inputs optimizer.Inputs) (bool, error) {
	// In production: evaluate CEL policy
	// For demo: simple logic
	
	drift := inputs.Drift.DriftPercent
	confidence := plan.Confidence
	
	isAutonomous := drift < 5.0 && confidence > 0.7
	return isAutonomous, nil
}

// ExecuteTradesActivity executes the plan using the saga executor
func (a *WealthActivities) ExecuteTradesActivity(ctx context.Context, plan optimizer.Plan, tenantID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing trades", "PlanID", plan.ID)

	// Initialize saga executor with mock adapter
	// In production: choose adapter based on tenant/custodian config
	adapter := custodian.MockIBKR{}
	executor := saga.Executor{Adapter: adapter}

	// Use workflow ID as correlation ID (mocking it here)
	workflowID := "wf_" + plan.ID 
	
	res, err := executor.Execute(ctx, workflowID, plan.ID, plan)
	if err != nil {
		logger.Error("Trade execution failed", "Error", err)
		return "FAILED", err
	}

	if res.Status == "compensated" {
		return "COMPENSATED", nil
	}

	return "COMPLETED", nil
}
