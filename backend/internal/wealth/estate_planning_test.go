package wealth

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestEstatePlanRecommender(t *testing.T) {
	// Test Annual Gifting recommendation
	t.Run("RecommendAnnualGifting", func(t *testing.T) {
		profile := &FamilyProfile{
			FamilyName:       "Test Family",
			TotalNetworth:    decimal.NewFromInt(10000000),
			MarriedCouple:    true,
			HasChildren:      true,
			HasGrandchildren: false,
			GenerationCount:  2,
			OldestMemberAge:  65,
		}

		recommender := NewEstatePlanRecommender(nil, nil)
		score := recommender.scoreAnnualGifting(profile)

		assert.Greater(t, score, 0.7, "Married couple with children should score high for annual gifting")
	})

	// Test SLAT recommendation
	t.Run("RecommendSLAT", func(t *testing.T) {
		profile := &FamilyProfile{
			TotalNetworth:   decimal.NewFromInt(15000000),
			MarriedCouple:   true,
			LiquidAssetPct:  decimal.NewFromInt(50),
			OldestMemberAge: 55,
		}

		recommender := NewEstatePlanRecommender(nil, nil)
		score := recommender.scoreSLAT(profile)

		assert.Greater(t, score, 0.6, "High net worth married couple should score high for SLAT")
	})

	// Test Dynasty Trust for multi-generational families
	t.Run("RecommendDynastyTrust", func(t *testing.T) {
		profile := &FamilyProfile{
			TotalNetworth:    decimal.NewFromInt(30000000),
			HasGrandchildren: true,
			GenerationCount:  3,
		}

		recommender := NewEstatePlanRecommender(nil, nil)
		score := recommender.scoreDynastyTrust(profile)

		assert.GreaterOrEqual(t, score, 0.7, "Ultra-high net worth multi-gen family should score high for Dynasty")
	})
}

func TestTaxCalculations(t *testing.T) {
	t.Run("CalculateAnnualGiftingSavings", func(t *testing.T) {
		profile := &FamilyProfile{
			Members: make([]FamilyMember, 4), // 2 parents + 2 children
		}

		recommender := NewEstatePlanRecommender(nil, nil)
		savings := recommender.calculateAnnualGiftingSavings(profile)

		assert.True(t, savings.GreaterThan(decimal.Zero), "Should calculate positive tax savings")
		// With 2 children, $37K/child/year for 10 years with growth
		expected := decimal.NewFromInt(74000).Mul(decimal.NewFromInt(10)).Mul(decimal.NewFromFloat(1.967)).Mul(decimal.NewFromFloat(0.40))
		// Allow some margin for floating point logic differences
		assert.True(t, savings.Sub(expected).Abs().LessThan(decimal.NewFromInt(10000)), "Should be within $10K of expected")
	})
}

func TestScenarioOptimizer(t *testing.T) {
	t.Run("FilterByConstraints", func(t *testing.T) {
		recommendations := []StrategyRecommendation{
			{
				StrategyType:          "SLAT",
				ComplexityScore:       6,
				ImplementationWeeks:   8,
				AnnualMaintenanceCost: decimal.NewFromInt(5000),
				EstimatedTaxSavings:   decimal.NewFromInt(2000000),
			},
			{
				StrategyType:          "ANNUAL_GIFTING",
				ComplexityScore:       2,
				ImplementationWeeks:   1,
				AnnualMaintenanceCost: decimal.NewFromInt(500),
				EstimatedTaxSavings:   decimal.NewFromInt(500000),
			},
		}

		constraints := OptimizationConstraints{
			MaxComplexity:          3,
			MaxImplementationWeeks: 4,
			MaxAnnualCost:          decimal.NewFromInt(1000),
			MinTaxSavings:          decimal.NewFromInt(100000),
		}

		optimizer := NewScenarioOptimizer(nil)
		filtered := optimizer.filterByConstraints(recommendations, constraints)

		assert.Equal(t, 1, len(filtered), "Should filter to only Annual Gifting")
		assert.Equal(t, "ANNUAL_GIFTING", filtered[0].StrategyType)
	})

	t.Run("CalculateSynergyBonus", func(t *testing.T) {
		strategies := []StrategyRecommendation{
			{StrategyType: "ANNUAL_GIFTING"},
			{StrategyType: "SLAT"},
		}

		optimizer := NewScenarioOptimizer(nil)
		bonus := optimizer.calculateSynergyBonus(strategies, &FamilyProfile{})

		assert.Equal(t, 0.10, bonus, "Annual Gifting + SLAT should have 10% synergy bonus")
	})

	t.Run("OptimizationScore", func(t *testing.T) {
		result := OptimizationResult{
			TotalTaxSavings:          decimal.NewFromInt(5000000),
			TotalComplexity:          10,
			TotalImplementationWeeks: 12,
			TotalAnnualCost:          decimal.NewFromInt(10000),
		}

		profile := &FamilyProfile{
			TotalNetworth: decimal.NewFromInt(20000000),
		}

		optimizer := NewScenarioOptimizer(nil)
		score := optimizer.calculateScore(result, profile)

		assert.Greater(t, score, 0.0, "Should calculate positive score")
		assert.Less(t, score, 1.0, "Score should be normalized to 0-1")
	})
}

func TestNarrativeGenerator(t *testing.T) {
	t.Run("GenerateScenarioNarrative", func(t *testing.T) {
		scenario := &EstateScenario{
			ScenarioName:    "Annual Exclusion Gifting",
			StrategyType:    "ANNUAL_GIFTING",
			TaxSavings:      decimal.NewFromInt(1500000),
			TaxSavingsPct:   15.0,
			ComplexityScore: 2,
			StructuresUsed:  []string{"None"},
			Implementation: []string{
				"Establish annual gifting calendar",
				"Execute gifts before year-end",
			},
			AnnualCost: decimal.NewFromInt(500),
		}

		profile := &FamilyProfile{
			FamilyName:    "Test Family",
			TotalNetworth: decimal.NewFromInt(10000000),
		}

		generator := NewEstatePlanNarrativeGenerator(nil)
		narrative, err := generator.GenerateScenarioNarrative(context.Background(), scenario, profile)

		assert.NoError(t, err)
		assert.Contains(t, narrative, "Annual Exclusion Gifting")
		assert.Contains(t, narrative, "Test Family")
		assert.Contains(t, narrative, "$1.5M")
	})

	t.Run("GenerateExecutiveSummary", func(t *testing.T) {
		scenario := EstateScenario{
			ScenarioName:    "SLAT",
			TaxSavings:      decimal.NewFromInt(3000000),
			TaxSavingsPct:   30.0,
			ComplexityScore: 6,
		}

		profile := &FamilyProfile{
			FamilyName:      "Smith Family",
			TotalNetworth:   decimal.NewFromInt(15000000),
			GenerationCount: 2,
			PrimaryState:    "CA",
		}

		generator := NewEstatePlanNarrativeGenerator(nil)
		summary, err := generator.GenerateExecutiveSummary(context.Background(), scenario, profile)

		assert.NoError(t, err)
		assert.Contains(t, summary, "Smith Family")
		assert.Contains(t, summary, "Executive Summary")
		assert.Contains(t, summary, "Next Steps")
	})
}

func TestCompatibilityChecks(t *testing.T) {
	t.Run("IncompatibleStrategies", func(t *testing.T) {
		s1 := StrategyRecommendation{StrategyType: "SLAT"}
		s2 := StrategyRecommendation{StrategyType: "SLAT"}

		optimizer := NewScenarioOptimizer(nil)
		compatible := optimizer.isCompatible(s1, s2)

		assert.False(t, compatible, "Cannot have multiple SLATs")
	})

	t.Run("CompatibleStrategies", func(t *testing.T) {
		s1 := StrategyRecommendation{StrategyType: "ANNUAL_GIFTING"}
		s2 := StrategyRecommendation{StrategyType: "GRAT"}

		optimizer := NewScenarioOptimizer(nil)
		compatible := optimizer.isCompatible(s1, s2)

		assert.True(t, compatible, "Annual Gifting and GRAT are compatible")
	})
}
