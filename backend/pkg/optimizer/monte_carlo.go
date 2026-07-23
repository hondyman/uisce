package optimizer

import (
	"math/rand"
	"sort"

	"gonum.org/v1/gonum/stat"
)

// ScenarioResult holds the outcome of a single Monte Carlo run
type ScenarioResult struct {
	ScenarioID int
	TaxImpact  float64 // net tax cost/benefit
	TEAfter    float64
	TransCost  float64
}

// MonteCarloSummary aggregates the results of the simulation
type MonteCarloSummary struct {
	MeanTaxImpact   float64
	MedianTaxImpact float64
	Pct05           float64
	Pct95           float64
	Confidence80Min float64
	Confidence80Max float64
	Runs            int
}

// MonteCarloSimulate runs the tax impact simulation
func MonteCarloSimulate(plan Plan, lots []Lot, prices map[string]float64, rules TaxRules, runs int) MonteCarloSummary {
	results := make([]ScenarioResult, 0, runs)
	for i := 0; i < runs; i++ {
		// perturb prices with random shocks
		perturbed := make(map[string]float64)
		for sym, p := range prices {
			shock := rand.NormFloat64() * 0.02 // 2% std dev shock
			perturbed[sym] = p * (1 + shock)
		}
		
		// In a real implementation, we would re-run the optimizer or at least re-evaluate the trade execution prices
		// For this demo, we'll approximate by adjusting the tax impact based on price shocks
		
		// recompute impact (mocking the re-evaluation logic for simplicity)
		// We assume trades execute at perturbed prices
		
		// Simple approximation: Tax impact varies by +/- 10% due to price slippage/volatility
		impactShock := 1.0 + (rand.NormFloat64() * 0.1)
		taxImpact := plan.TaxImpact * impactShock
		
		// adjust TEAfter with random noise
		teAfter := plan.TEAfter + rand.NormFloat64()*0.1
		if teAfter < 0 { teAfter = 0 }
		
		results = append(results, ScenarioResult{
			ScenarioID: i, 
			TaxImpact:  taxImpact, 
			TEAfter:    teAfter, 
			TransCost:  plan.TransCost,
		})
	}
	return summarize(results)
}

func summarize(results []ScenarioResult) MonteCarloSummary {
	n := len(results)
	impacts := make([]float64, n)
	for i, r := range results {
		impacts[i] = r.TaxImpact
	}
	sort.Float64s(impacts)
	
	mean := stat.Mean(impacts, nil)
	median := stat.Quantile(0.5, stat.Empirical, impacts, nil)
	pct05 := stat.Quantile(0.05, stat.Empirical, impacts, nil)
	pct95 := stat.Quantile(0.95, stat.Empirical, impacts, nil)
	
	return MonteCarloSummary{
		MeanTaxImpact:   mean,
		MedianTaxImpact: median,
		Pct05:           pct05,
		Pct95:           pct95,
		Confidence80Min: stat.Quantile(0.1, stat.Empirical, impacts, nil),
		Confidence80Max: stat.Quantile(0.9, stat.Empirical, impacts, nil),
		Runs:            n,
	}
}
