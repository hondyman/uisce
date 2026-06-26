package indexing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/values"
)

type PortfolioConstructionService interface {
	CalculateIdealHoldings(ctx context.Context, req RebalanceRequest) ([]IndexConstituent, error)
	GenerateOrders(ctx context.Context, current *Portfolio, ideal []IndexConstituent) ([]Order, error)
}

type portfolioConstructionServiceImpl struct {
}

func NewPortfolioConstructionService() PortfolioConstructionService {
	return &portfolioConstructionServiceImpl{}
}

// CalculateIdealHoldings determines the target portfolio weights
// 1. Start with Benchmark Index weights
// 2. Apply Constraints (Exclusions)
// 3. Renormalize weights
func (s *portfolioConstructionServiceImpl) CalculateIdealHoldings(ctx context.Context, req RebalanceRequest) ([]IndexConstituent, error) {
	// Map of Ticker -> Constituent
	idealMap := make(map[string]IndexConstituent)
	totalWeight := 0.0

	// 1. Load initial index constituents
	for _, c := range req.Benchmark.Constituents {
		idealMap[c.Ticker] = c
		totalWeight += c.Weight
	}

	// 2. Apply Constraints
	// We need to identify which tickers to exclude based on constraints and signals
	excludedTickers := make(map[string]bool)

	for _, constraint := range req.Constraints {
		if constraint.Operator == values.OperatorExclude {
			// Check each constituent against the constraint
			for _, c := range req.Benchmark.Constituents {
				if isViolation(c, constraint, req.Signals) {
					excludedTickers[c.Ticker] = true
				}
			}
		}
	}

	// Remove excluded tickers from idealMap
	for ticker := range excludedTickers {
		delete(idealMap, ticker)
	}

	// 3. Renormalize weights
	// Calculate remaining total weight
	remainingWeight := 0.0
	for _, c := range idealMap {
		remainingWeight += c.Weight
	}

	if remainingWeight == 0 {
		return nil, fmt.Errorf("all assets excluded, cannot construct portfolio")
	}

	// Scale factor to bring total back to 1.0 (or whatever the original total was, usually 1.0)
	// If original index didn't sum to 1.0, we preserve the scale relative to original.
	// Assuming standard index sums to ~1.0.
	scaleFactor := totalWeight / remainingWeight

	var idealPortfolio []IndexConstituent
	for _, c := range idealMap {
		c.Weight = c.Weight * scaleFactor
		idealPortfolio = append(idealPortfolio, c)
	}

	return idealPortfolio, nil
}

// isViolation checks if a constituent violates a constraint
// This is a simplified matcher. In a real system, this would use a robust rule engine.
func isViolation(c IndexConstituent, constraint values.Constraint, signals []values.ValueSignal) bool {
	// 1. Check direct scope matches (e.g. Sector, Region)
	if constraint.Scope.Sector != "" && c.Sector == constraint.Scope.Sector {
		return true
	}
	if constraint.Scope.Region != "" && c.Region == constraint.Scope.Region {
		return true
	}
	if constraint.Scope.Issuer != "" && c.Ticker == constraint.Scope.Issuer {
		return true
	}

	// 2. Check signal-based violations (e.g. "Labor Practices" score < 50)
	// This requires parsing the 'Condition' JSON which is complex.
	// For this MVP, we will assume if there is a signal for this issuer
	// that matches the constraint's criteria, it's a violation.
	// We'll look for signals for this ticker.
	for _, sig := range signals {
		if sig.IssuerID == c.Ticker {
			// If constraint has a condition, we should evaluate it.
			// Simplified: If signal score is negative, treat as violation for now if no condition logic is implemented.
			if sig.Score < 0 {
				return true
			}
		}
	}

	return false
}

// GenerateOrders compares current holdings with ideal holdings and generates orders
func (s *portfolioConstructionServiceImpl) GenerateOrders(ctx context.Context, current *Portfolio, ideal []IndexConstituent) ([]Order, error) {
	var orders []Order

	// Map ideal weights
	idealWeights := make(map[string]float64)
	for _, c := range ideal {
		idealWeights[c.Ticker] = c.Weight
	}

	// Calculate total portfolio value
	totalValue := current.Cash
	for _, h := range current.Holdings {
		totalValue += h.Value
	}

	// 1. Identify Sells (Overweight or Excluded)
	for _, h := range current.Holdings {
		targetWeight, exists := idealWeights[h.Ticker]
		currentWeight := h.Value / totalValue

		if !exists {
			// Sell all (Excluded)
			orders = append(orders, Order{
				ID:          uuid.New(),
				PortfolioID: current.ID,
				Ticker:      h.Ticker,
				Type:        OrderTypeSell,
				Quantity:    h.Shares,
				Amount:      h.Value,
				Reason:      "Excluded by constraints",
				Status:      "PENDING",
				CreatedAt:   time.Now(),
			})
		} else if currentWeight > targetWeight {
			// Sell difference (Overweight)
			diffWeight := currentWeight - targetWeight
			sellAmount := diffWeight * totalValue
			// Assuming price = Value / Shares
			price := h.Value / h.Shares
			sellShares := sellAmount / price

			orders = append(orders, Order{
				ID:          uuid.New(),
				PortfolioID: current.ID,
				Ticker:      h.Ticker,
				Type:        OrderTypeSell,
				Quantity:    sellShares,
				Amount:      sellAmount,
				Reason:      "Rebalance (Overweight)",
				Status:      "PENDING",
				CreatedAt:   time.Now(),
			})
		}
	}

	// 2. Identify Buys (Underweight or New)
	// Note: In a real system, we'd account for cash generated from sells.
	// Here we assume we can just generate the orders and the OMS handles settlement/cash.
	for _, c := range ideal {
		targetWeight := c.Weight

		// Find current holding
		var currentHolding *Holding
		for _, h := range current.Holdings {
			if h.Ticker == c.Ticker {
				currentHolding = &h
				break
			}
		}

		currentWeight := 0.0
		if currentHolding != nil {
			currentWeight = currentHolding.Value / totalValue
		}

		if currentWeight < targetWeight {
			// Buy difference
			diffWeight := targetWeight - currentWeight
			buyAmount := diffWeight * totalValue

			// We don't have current price for new assets in 'current.Holdings'.
			// In a real system, we'd fetch quote.
			// For MVP, we'll set Amount and leave Quantity 0 if price unknown, or assume $100 placeholder.
			price := 100.0
			if currentHolding != nil {
				price = currentHolding.Value / currentHolding.Shares
			}

			buyShares := buyAmount / price

			orders = append(orders, Order{
				ID:          uuid.New(),
				PortfolioID: current.ID,
				Ticker:      c.Ticker,
				Type:        OrderTypeBuy,
				Quantity:    buyShares,
				Amount:      buyAmount,
				Reason:      "Rebalance (Underweight)",
				Status:      "PENDING",
				CreatedAt:   time.Now(),
			})
		}
	}

	return orders, nil
}
