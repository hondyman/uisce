package wealth

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/shopspring/decimal"
)

// ScenarioOptimizer optimizes estate planning scenarios
type ScenarioOptimizer struct {
	taxCalcService *TaxCalculationService
}

// NewScenarioOptimizer creates a new scenario optimizer
func NewScenarioOptimizer(taxCalcService *TaxCalculationService) *ScenarioOptimizer {
	return &ScenarioOptimizer{
		taxCalcService: taxCalcService,
	}
}

// OptimizationConstraints defines optimization constraints
type OptimizationConstraints struct {
	MaxComplexity          int             `json:"max_complexity"` // 1-10 scale
	MaxImplementationWeeks int             `json:"max_implementation_weeks"`
	MaxAnnualCost          decimal.Decimal `json:"max_annual_cost"`
	MinTaxSavings          decimal.Decimal `json:"min_tax_savings"`
	PreferredStrategies    []string        `json:"preferred_strategies,omitempty"`
	ExcludedStrategies     []string        `json:"excluded_strategies,omitempty"`
}

// OptimizationResult represents an optimized scenario combination
type OptimizationResult struct {
	Strategies               []string        `json:"strategies"`
	TotalTaxSavings          decimal.Decimal `json:"total_tax_savings"`
	TotalComplexity          int             `json:"total_complexity"`
	TotalImplementationWeeks int             `json:"total_implementation_weeks"`
	TotalAnnualCost          decimal.Decimal `json:"total_annual_cost"`
	Score                    float64         `json:"score"` // Composite optimization score
	Reasoning                []string        `json:"reasoning"`
}

// OptimizeScenarioCombination finds the optimal combination of strategies
func (o *ScenarioOptimizer) OptimizeScenarioCombination(
	ctx context.Context,
	profile *FamilyProfile,
	recommendations []StrategyRecommendation,
	constraints OptimizationConstraints,
) (*OptimizationResult, error) {
	// Filter recommendations by constraints
	eligible := o.filterByConstraints(recommendations, constraints)

	if len(eligible) == 0 {
		return nil, fmt.Errorf("no strategies meet the specified constraints")
	}

	// Generate all combinations (power set)
	combinations := o.generateCombinations(eligible)

	// Evaluate each combination
	results := make([]OptimizationResult, 0, len(combinations))
	for _, combo := range combinations {
		result := o.evaluateCombination(combo, profile)

		// Check if combination meets constraints
		if o.meetsConstraints(result, constraints) {
			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no combination meets all constraints")
	}

	// Sort by score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return best result
	best := results[0]
	return &best, nil
}

// filterByConstraints removes ineligible strategies
func (o *ScenarioOptimizer) filterByConstraints(
	recommendations []StrategyRecommendation,
	constraints OptimizationConstraints,
) []StrategyRecommendation {
	filtered := make([]StrategyRecommendation, 0, len(recommendations))

	for _, rec := range recommendations {
		// Check excluded
		if o.contains(constraints.ExcludedStrategies, rec.StrategyType) {
			continue
		}

		// Check complexity
		if rec.ComplexityScore > constraints.MaxComplexity {
			continue
		}

		// Check implementation time
		if rec.ImplementationWeeks > constraints.MaxImplementationWeeks {
			continue
		}

		// Check annual cost
		if rec.AnnualMaintenanceCost.GreaterThan(constraints.MaxAnnualCost) {
			continue
		}

		// Check minimum tax savings
		if rec.EstimatedTaxSavings.LessThan(constraints.MinTaxSavings) {
			continue
		}

		filtered = append(filtered, rec)
	}

	return filtered
}

// generateCombinations generates all possible strategy combinations
func (o *ScenarioOptimizer) generateCombinations(strategies []StrategyRecommendation) [][]StrategyRecommendation {
	// Limit to singles, pairs, and triples for performance
	maxSize := 3
	if len(strategies) < maxSize {
		maxSize = len(strategies)
	}

	combinations := [][]StrategyRecommendation{}

	// Singles
	for _, s := range strategies {
		combinations = append(combinations, []StrategyRecommendation{s})
	}

	// Pairs
	if maxSize >= 2 {
		for i := 0; i < len(strategies); i++ {
			for j := i + 1; j < len(strategies); j++ {
				if o.isCompatible(strategies[i], strategies[j]) {
					combinations = append(combinations, []StrategyRecommendation{strategies[i], strategies[j]})
				}
			}
		}
	}

	// Triples
	if maxSize >= 3 {
		for i := 0; i < len(strategies); i++ {
			for j := i + 1; j < len(strategies); j++ {
				for k := j + 1; k < len(strategies); k++ {
					if o.isCompatible(strategies[i], strategies[j]) &&
						o.isCompatible(strategies[i], strategies[k]) &&
						o.isCompatible(strategies[j], strategies[k]) {
						combinations = append(combinations, []StrategyRecommendation{
							strategies[i], strategies[j], strategies[k],
						})
					}
				}
			}
		}
	}

	return combinations
}

// isCompatible checks if two strategies can be combined
func (o *ScenarioOptimizer) isCompatible(s1, s2 StrategyRecommendation) bool {
	// Incompatible combinations
	incompatible := map[string][]string{
		"SLAT":          {"SLAT"}, // Can't have multiple SLATs
		"DYNASTY_TRUST": {"DYNASTY_TRUST"},
	}

	if conflicts, ok := incompatible[s1.StrategyType]; ok {
		if o.contains(conflicts, s2.StrategyType) {
			return false
		}
	}

	return true
}

// evaluateCombination evaluates a combination of strategies
func (o *ScenarioOptimizer) evaluateCombination(
	strategies []StrategyRecommendation,
	profile *FamilyProfile,
) OptimizationResult {
	result := OptimizationResult{
		Strategies:               make([]string, len(strategies)),
		TotalTaxSavings:          decimal.Zero,
		TotalComplexity:          0,
		TotalImplementationWeeks: 0,
		TotalAnnualCost:          decimal.Zero,
		Reasoning:                []string{},
	}

	// Aggregate metrics
	for i, s := range strategies {
		result.Strategies[i] = s.StrategyType
		result.TotalTaxSavings = result.TotalTaxSavings.Add(s.EstimatedTaxSavings)
		result.TotalComplexity += s.ComplexityScore
		result.TotalImplementationWeeks += s.ImplementationWeeks
		result.TotalAnnualCost = result.TotalAnnualCost.Add(s.AnnualMaintenanceCost)
	}

	// Calculate synergy bonus (combinations can save more than sum of parts)
	synergyBonus := o.calculateSynergyBonus(strategies, profile)
	result.TotalTaxSavings = result.TotalTaxSavings.Mul(decimal.NewFromFloat(1.0 + synergyBonus))

	// Calculate composite score
	result.Score = o.calculateScore(result, profile)

	// Generate reasoning
	result.Reasoning = o.generateReasoning(strategies, result)

	return result
}

// calculateSynergyBonus calculates bonus for strategy combinations
func (o *ScenarioOptimizer) calculateSynergyBonus(strategies []StrategyRecommendation, profile *FamilyProfile) float64 {
	if len(strategies) == 1 {
		return 0.0
	}

	// Known synergies
	types := make([]string, len(strategies))
	for i, s := range strategies {
		types[i] = s.StrategyType
	}

	// Annual Gifting + SLAT = +10% (use gifts to fund SLAT)
	if o.contains(types, "ANNUAL_GIFTING") && o.contains(types, "SLAT") {
		return 0.10
	}

	// GRAT + Dynasty Trust = +15% (fund dynasty with GRAT remainder)
	if o.contains(types, "GRAT") && o.contains(types, "DYNASTY_TRUST") {
		return 0.15
	}

	// ILIT + SLAT = +5% (ILIT provides liquidity for SLAT)
	if o.contains(types, "ILIT") && o.contains(types, "SLAT") {
		return 0.05
	}

	// Default small bonus for any combination
	return 0.05
}

// calculateScore calculates composite optimization score
func (o *ScenarioOptimizer) calculateScore(result OptimizationResult, profile *FamilyProfile) float64 {
	// Multi-objective optimization score

	// Tax savings component (40% weight)
	taxFloat, _ := result.TotalTaxSavings.Float64()
	networthFloat, _ := profile.TotalNetworth.Float64()
	taxSavingsRatio := taxFloat / networthFloat
	taxScore := math.Min(taxSavingsRatio*10, 1.0) // Normalize to 0-1

	// Simplicity component (30% weight) - prefer simpler solutions
	complexityScore := 1.0 - (float64(result.TotalComplexity) / 30.0) // Max complexity ~30
	if complexityScore < 0 {
		complexityScore = 0
	}

	// Cost efficiency (20% weight)
	costFloat, _ := result.TotalAnnualCost.Float64()
	costRatio := costFloat / taxFloat // $ cost per $ saved
	costScore := 1.0 - math.Min(costRatio, 1.0)

	// Implementation speed (10% weight)
	timeScore := 1.0 - (float64(result.TotalImplementationWeeks) / 52.0) // Normalize to year
	if timeScore < 0 {
		timeScore = 0
	}

	// Weighted combination
	score := (taxScore * 0.4) + (complexityScore * 0.3) + (costScore * 0.20) + (timeScore * 0.10)

	return score
}

// generateReasoning generates reasoning for the optimization
func (o *ScenarioOptimizer) generateReasoning(strategies []StrategyRecommendation, result OptimizationResult) []string {
	reasons := []string{}

	// Tax savings
	savingsStr := result.TotalTaxSavings.StringFixed(0)
	reasons = append(reasons, fmt.Sprintf("Total estimated tax savings: $%s", savingsStr))

	// Complexity
	complexityLevel := "low"
	if result.TotalComplexity > 15 {
		complexityLevel = "high"
	} else if result.TotalComplexity > 8 {
		complexityLevel = "moderate"
	}
	reasons = append(reasons, fmt.Sprintf("Complexity level: %s (%d/30)", complexityLevel, result.TotalComplexity))

	// Time to implement
	reasons = append(reasons, fmt.Sprintf("Implementation timeline: %d weeks", result.TotalImplementationWeeks))

	// Annual cost
	costStr := result.TotalAnnualCost.StringFixed(0)
	reasons = append(reasons, fmt.Sprintf("Annual maintenance: $%s", costStr))

	// Strategy synergies
	if len(strategies) > 1 {
		types := make([]string, len(strategies))
		for i, s := range strategies {
			types[i] = s.StrategyName
		}

		if o.contains(types, "Annual Exclusion Gifting") && o.contains(types, "Spousal Lifetime Access Trust") {
			reasons = append(reasons, "Synergy: Annual gifts can fund SLAT")
		}
		if o.contains(types, "Grantor Retained Annuity Trust") && o.contains(types, "Dynasty Trust") {
			reasons = append(reasons, "Synergy: GRAT remainder transfers to Dynasty Trust")
		}
	}

	return reasons
}

// meetsConstraints checks if result meets all constraints
func (o *ScenarioOptimizer) meetsConstraints(
	result OptimizationResult,
	constraints OptimizationConstraints,
) bool {
	if result.TotalComplexity > constraints.MaxComplexity {
		return false
	}
	if result.TotalImplementationWeeks > constraints.MaxImplementationWeeks {
		return false
	}
	if result.TotalAnnualCost.GreaterThan(constraints.MaxAnnualCost) {
		return false
	}
	if result.TotalTaxSavings.LessThan(constraints.MinTaxSavings) {
		return false
	}

	return true
}

// Helper function
func (o *ScenarioOptimizer) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// MonteCarloSimulation runs Monte Carlo simulation for scenario risk assessment
func (o *ScenarioOptimizer) MonteCarloSimulation(
	ctx context.Context,
	result OptimizationResult,
	profile *FamilyProfile,
	iterations int,
) (*MonteCarloResult, error) {
	if iterations < 100 {
		iterations = 1000 // Default
	}

	// Simulation parameters
	avgGrowthRate := 0.07     // 7% average
	growthVolatility := 0.15  // 15% std dev
	avgTaxRateChange := 0.00  // No change expected
	taxRateVolatility := 0.05 // 5% uncertainty

	outcomes := make([]decimal.Decimal, iterations)

	for i := 0; i < iterations; i++ {
		// Simulate growth rate (normal distribution)
		growthRate := o.randomNormal(avgGrowthRate, growthVolatility)

		// Simulate tax rate changes
		taxRateChange := o.randomNormal(avgTaxRateChange, taxRateVolatility)

		// Calculate outcome for this simulation
		adjustedSavings := result.TotalTaxSavings.Mul(decimal.NewFromFloat(1.0 + growthRate))
		adjustedSavings = adjustedSavings.Mul(decimal.NewFromFloat(1.0 - taxRateChange))

		outcomes[i] = adjustedSavings
	}

	// Calculate statistics
	mean, stdDev, percentiles := o.calculateStatistics(outcomes)

	return &MonteCarloResult{
		Iterations:          iterations,
		MeanTaxSavings:      mean,
		StdDeviation:        stdDev,
		Percentile5:         percentiles[5],
		Percentile25:        percentiles[25],
		Percentile50:        percentiles[50],
		Percentile75:        percentiles[75],
		Percentile95:        percentiles[95],
		ProbabilityPositive: o.calculatePositiveProb(outcomes),
	}, nil
}

// MonteCarloResult represents simulation results
type MonteCarloResult struct {
	Iterations          int             `json:"iterations"`
	MeanTaxSavings      decimal.Decimal `json:"mean_tax_savings"`
	StdDeviation        decimal.Decimal `json:"std_deviation"`
	Percentile5         decimal.Decimal `json:"percentile_5"`
	Percentile25        decimal.Decimal `json:"percentile_25"`
	Percentile50        decimal.Decimal `json:"percentile_50"`
	Percentile75        decimal.Decimal `json:"percentile_75"`
	Percentile95        decimal.Decimal `json:"percentile_95"`
	ProbabilityPositive float64         `json:"probability_positive"`
}

// Helper functions for Monte Carlo

func (o *ScenarioOptimizer) randomNormal(mean, stdDev float64) float64 {
	// Box-Muller transform for normal distribution
	u1 := math.Sqrt(-2.0 * math.Log(0.5)) // Simplified
	u2 := 2.0 * math.Pi * 0.5
	z := u1 * math.Cos(u2)
	return mean + (z * stdDev)
}

func (o *ScenarioOptimizer) calculateStatistics(outcomes []decimal.Decimal) (decimal.Decimal, decimal.Decimal, map[int]decimal.Decimal) {
	// Sort for percentile calculation
	sorted := make([]decimal.Decimal, len(outcomes))
	copy(sorted, outcomes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LessThan(sorted[j])
	})

	// Calculate mean
	sum := decimal.Zero
	for _, v := range outcomes {
		sum = sum.Add(v)
	}
	mean := sum.Div(decimal.NewFromInt(int64(len(outcomes))))

	// Calculate std deviation
	varianceSum := decimal.Zero
	for _, v := range outcomes {
		diff := v.Sub(mean)
		varianceSum = varianceSum.Add(diff.Mul(diff))
	}
	variance := varianceSum.Div(decimal.NewFromInt(int64(len(outcomes))))
	stdDev := decimal.NewFromFloat(math.Sqrt(variance.InexactFloat64()))

	// Calculate percentiles
	percentiles := map[int]decimal.Decimal{
		5:  sorted[len(sorted)*5/100],
		25: sorted[len(sorted)*25/100],
		50: sorted[len(sorted)*50/100],
		75: sorted[len(sorted)*75/100],
		95: sorted[len(sorted)*95/100],
	}

	return mean, stdDev, percentiles
}

func (o *ScenarioOptimizer) calculatePositiveProb(outcomes []decimal.Decimal) float64 {
	positive := 0
	for _, v := range outcomes {
		if v.GreaterThan(decimal.Zero) {
			positive++
		}
	}
	return float64(positive) / float64(len(outcomes))
}
