package portfolio

import (
	"errors"
	"time"

	"go.temporal.io/sdk/workflow"
)

// PortfolioState represents the in-memory state of a portfolio
type PortfolioState struct {
	PortfolioID  string             `json:"portfolio_id"`
	CashBalance  float64            `json:"cash_balance"`
	Positions    map[string]float64 `json:"positions"`
	RiskExposure float64            `json:"risk_exposure"`
}

// DeepCopy creates a deep copy of the state for simulation
func (s *PortfolioState) DeepCopy() *PortfolioState {
	newPositions := make(map[string]float64)
	for k, v := range s.Positions {
		newPositions[k] = v
	}
	return &PortfolioState{
		PortfolioID:  s.PortfolioID,
		CashBalance:  s.CashBalance,
		Positions:    newPositions,
		RiskExposure: s.RiskExposure,
	}
}

// TradeInput represents the input for a trade simulation
type TradeInput struct {
	Symbol string  `json:"symbol"`
	Qty    float64 `json:"qty"`
	Price  float64 `json:"price"`
	Side   string  `json:"side"` // Buy, Sell
}

// PortfolioWorkflow manages the state of a portfolio and handles simulations
func PortfolioWorkflow(ctx workflow.Context, portfolioID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting PortfolioWorkflow", "PortfolioID", portfolioID)

	// Initialize State (Mock loading from DB)
	state := &PortfolioState{
		PortfolioID: portfolioID,
		CashBalance: 100000.0,
		Positions: map[string]float64{
			"AAPL": 100.0,
		},
		RiskExposure: 0.0, // Placeholder
	}

	// Setup Query Handler for "What-If" Analysis
	err := workflow.SetQueryHandler(ctx, "SimulateTrade", func(input TradeInput) (*PortfolioState, error) {
		logger.Info("Handling SimulateTrade Query", "Input", input)

		// 1. CLONE the state (Critical! Do not modify actual state)
		simState := state.DeepCopy()

		// 2. Apply the hypothetical trade logic
		cost := input.Price * input.Qty
		if input.Side == "Buy" {
			simState.CashBalance -= cost
			simState.Positions[input.Symbol] += input.Qty
		} else if input.Side == "Sell" {
			simState.CashBalance += cost
			simState.Positions[input.Symbol] -= input.Qty
		} else {
			return nil, errors.New("invalid side")
		}

		// 3. Recalculate Risk (Validation - Mocked)
		// In a real system, this would call a risk engine or activity
		if simState.Positions["AAPL"] > 1000 {
			simState.RiskExposure = 100.0 // High risk
		} else {
			simState.RiskExposure = 10.0 // Low risk
		}

		// 4. Return the hypothetical future
		return simState, nil
	})
	if err != nil {
		return err
	}

	// Main Event Loop (Keep workflow running)
	// In a real system, this would handle Signals for actual trades, deposits, etc.
	selector := workflow.NewSelector(ctx)
	
	// Example Signal Handler (Placeholder)
	/*
	selector.AddReceive(workflow.GetSignalChannel(ctx, "TradeConfirmed"), func(c workflow.ReceiveChannel, more bool) {
		var trade TradeInput
		c.Receive(ctx, &trade)
		// Update actual state...
	})
	*/

	// Wait indefinitely (or until cancellation)
	// Using a timer loop to keep it alive for demo purposes if no signals
	for {
		selector.AddFuture(workflow.NewTimer(ctx, time.Hour*24), func(f workflow.Future) {
			// Daily maintenance or keep-alive
		})
		selector.Select(ctx)
	}

	return nil
}
