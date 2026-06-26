package simulation

import (
	"context"
	"fmt"
	"math"
)

// RebalanceEngine handles the logic for generating trades to align portfolios with targets
type RebalanceEngine struct {
	// In a real system, would need access to MarketData, etc.
}

func NewRebalanceEngine() *RebalanceEngine {
	return &RebalanceEngine{}
}

// GenerateDeltas calculates the required trades to meet the rebalance rule
func (e *RebalanceEngine) GenerateDeltas(ctx context.Context, currentPositions map[string]float64, prices map[string]float64, rule *RebalanceRule) ([]*SimulationDelta, error) {
	var deltas []*SimulationDelta

	if rule.Type == "TO_TARGET_WEIGHTS" {
		// 1. Calculate Total Portfolio Value
		totalValue := 0.0
		for assetID, qty := range currentPositions {
			price, ok := prices[assetID]
			if !ok {
				return nil, fmt.Errorf("missing price for asset: %s", assetID)
			}
			totalValue += qty * price
		}

		// 2. Iterate Targets and Calculate Deltas
		for assetID, targetWeight := range rule.Targets {
			currentQty := currentPositions[assetID]
			price := prices[assetID]

			// If asset not in current portfolio, assume 0 quantity (buy)
			// (Assuming prices map has it, or we fetch it)
			if price == 0 {
				continue // Cannot trade without price
			}

			targetValue := totalValue * targetWeight
			currentValue := currentQty * price
			diffValue := targetValue - currentValue

			// Threshold check (e.g., ignore drift < 1%)
			// For MVP, simplistic check: if abs(diff) > X
			if math.Abs(diffValue) > 0.01 { // trivial threshold
				deltaQty := diffValue / price

				// Apply Constraints (Mock implementation)
				// e.g. if rule.Constraints.MaxAssetWeight ...

				changeJSON := []byte(fmt.Sprintf(`{"quantity": %.4f, "value": %.2f}`, deltaQty, diffValue))

				deltas = append(deltas, &SimulationDelta{
					BOID:      assetID,
					DeltaType: DeltaTypePosition, // Rebalance turns into position deltas
					Changes:   changeJSON,
					// ScenarioID filled by caller
				})
			}
		}
	} else {
		return nil, fmt.Errorf("unsupported rebalance type: %s", rule.Type)
	}

	return deltas, nil
}
