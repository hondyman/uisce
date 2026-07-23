package engine

import (
	"math/rand"
	"sort"
)

// MonteCarloSimulate runs a simulation to estimate tax impact distribution
func MonteCarloSimulate(plan Plan, runs int) MonteCarloSummary {
	// In a real implementation, this would use:
	// - Historical volatility of assets
	// - Correlation matrix
	// - Tax rules (short-term vs long-term rates)
	// to simulate future prices and potential tax events.

	// For this prototype, we simulate a distribution around the estimated tax impact
	// using a normal distribution approximation.

	baseImpact := plan.TaxImpact
	stdDev := 500.0 // Arbitrary standard deviation for the simulation

	results := make([]float64, runs)
	sum := 0.0

	for i := 0; i < runs; i++ {
		// Simulate random noise
		noise := rand.NormFloat64() * stdDev
		simulatedImpact := baseImpact + noise
		results[i] = simulatedImpact
		sum += simulatedImpact
	}

	sort.Float64s(results)

	mean := sum / float64(runs)
	median := results[runs/2]
	pct05 := results[int(float64(runs)*0.05)]
	pct95 := results[int(float64(runs)*0.95)]
	conf80Min := results[int(float64(runs)*0.10)]
	conf80Max := results[int(float64(runs)*0.90)]

	return MonteCarloSummary{
		MeanTaxImpact:   mean,
		MedianTaxImpact: median,
		Pct05:           pct05,
		Pct95:           pct95,
		Confidence80Min: conf80Min,
		Confidence80Max: conf80Max,
		Runs:            runs,
	}
}
