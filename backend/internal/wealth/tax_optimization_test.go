package wealth

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateResidencyOptimizer(t *testing.T) {
	service := NewTaxOptimizationService(nil)

	t.Run("CompareCAtoFL", func(t *testing.T) {
		result, err := service.CompareStateResidencies(
			context.Background(),
			"test-family-id",
			"CA",
			decimal.NewFromInt(1000000),  // $1M gross income
			decimal.NewFromInt(500000),   // $500K investment income
			decimal.NewFromInt(200000),   // $200K capital gains
			decimal.NewFromInt(50000000), // $50M estate
			[]string{"FL", "TX", "NV"},
		)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "CA", result.CurrentState)
		assert.Len(t, result.StateComparisons, 4) // CA + 3 comparison states

		// Florida should show significant savings vs California
		flDetail := findStateDetail(result.StateComparisons, "FL")
		require.NotNil(t, flDetail)
		t.Logf("FL Savings: %s", flDetail.AnnualSavingsVsCurrent)
		assert.True(t, flDetail.AnnualSavingsVsCurrent.GreaterThan(decimal.NewFromInt(40000)),
			"FL should save >$40K/year vs CA")
		assert.Equal(t, false, flDetail.HasIncomeTax)
		assert.Equal(t, false, flDetail.HasEstateTax)

		// Verify California has highest tax
		caDetail := findStateDetail(result.StateComparisons, "CA")
		require.NotNil(t, caDetail)
		assert.True(t, caDetail.HasIncomeTax)
		assert.True(t, caDetail.AnnualSavingsVsCurrent.Equal(decimal.Zero), "CA vs CA savings should be zero")
	})

	t.Run("Top3Recommendations", func(t *testing.T) {
		result, err := service.CompareStateResidencies(
			context.Background(),
			"test-family-id",
			"NY",
			decimal.NewFromInt(2000000),
			decimal.NewFromInt(1000000),
			decimal.NewFromInt(500000),
			decimal.NewFromInt(100000000),
			[]string{"FL", "TX", "NV", "WA", "CA"},
		)

		require.NoError(t, err)
		assert.Len(t, result.TopRecommendations, 3)

		// First recommendation should be highest savings
		assert.True(t, result.TopRecommendations[0].AnnualSavingsVsCurrent.GreaterThan(
			result.TopRecommendations[1].AnnualSavingsVsCurrent,
		))
	})
}

func TestNIITCalculator(t *testing.T) {
	service := NewTaxOptimizationService(nil)

	t.Run("Above Threshold", func(t *testing.T) {
		components := InvestmentIncomeBreakdown{
			Interest:            decimal.NewFromInt(50000),
			Dividends:           decimal.NewFromInt(100000),
			CapitalGains:        decimal.NewFromInt(150000),
			RentalIncome:        decimal.NewFromInt(50000),
			NetInvestmentIncome: decimal.NewFromInt(350000),
		}

		result, err := service.CalculateNIIT(
			context.Background(),
			"test-family-id",
			"test-member-id",
			2025,
			"MARRIED_JOINT",
			decimal.NewFromInt(600000), // MAGI
			components,
		)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, decimal.NewFromInt(250000), result.NIITThreshold)

		// Excess MAGI = $600K - $250K = $350K
		assert.Equal(t, decimal.NewFromInt(350000), result.ExcessOverThreshold)

		// Taxable NII = min(Net Investment Income, Excess MAGI) = min($350K, $350K) = $350K
		assert.Equal(t, decimal.NewFromInt(350000), result.TaxableNII)

		// NIIT = $350K * 3.8% = $13,300
		expectedNIIT := decimal.NewFromInt(13300)
		assert.True(t, result.NIITax.Equal(expectedNIIT), "Expected $13,300 NIIT, got %s", result.NIITax)

		// Should have mitigation strategies
		assert.NotEmpty(t, result.MitigationStrategies)
	})

	t.Run("BelowThreshold", func(t *testing.T) {
		components := InvestmentIncomeBreakdown{
			Interest:            decimal.NewFromInt(20000),
			Dividends:           decimal.NewFromInt(30000),
			NetInvestmentIncome: decimal.NewFromInt(50000),
		}

		result, err := service.CalculateNIIT(
			context.Background(),
			"test-family-id",
			"test-member-id",
			2025,
			"SINGLE",
			decimal.NewFromInt(180000), // Below $200K threshold
			components,
		)

		require.NoError(t, err)
		assert.True(t, result.ExcessOverThreshold.Equal(decimal.Zero), "Excess should be zero")
		assert.True(t, result.NIITax.Equal(decimal.Zero), "Tax should be zero")
	})
}

func TestCharitableBunchingAnalyzer(t *testing.T) {
	service := NewTaxOptimizationService(nil)

	t.Run("BunchingBetter", func(t *testing.T) {
		result, err := service.AnalyzeCharitableBunching(
			context.Background(),
			"test-family-id",
			"test-member-id",
			6,                          // 6 years
			decimal.NewFromInt(50000),  // $50K annual giving
			decimal.NewFromInt(29200),  // 2025 standard deduction (married)
			decimal.NewFromInt(20000),  // $20K other itemized deductions
			decimal.NewFromFloat(0.37), // 37% marginal tax rate
		)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 6, result.AnalysisYears)

		// Baseline: $50K + $20K = $70K total, above standard deduction
		assert.NotNil(t, result.BaselineScenario)
		assert.Len(t, result.BaselineScenario.YearByYearBreakdown, 6)

		// Bunching: 3-year bunching strategy
		assert.NotNil(t, result.BunchingScenario)
		assert.Len(t, result.BunchingScenario.YearByYearBreakdown, 6)

		// Should recommend bunching if savings > $5K
		if result.EstimatedTaxSavings.GreaterThan(decimal.NewFromInt(5000)) {
			assert.Equal(t, "BUNCHING_3YR", result.RecommendedStrategy)
			assert.NotNil(t, result.DAFRecommendation)
		} else {
			assert.Equal(t, "ANNUAL", result.RecommendedStrategy)
		}

		// Tax savings should be calculated
		assert.True(t, result.EstimatedTaxSavings.GreaterThanOrEqual(decimal.Zero))
	})

	t.Run("AnnualBetter", func(t *testing.T) {
		result, err := service.AnalyzeCharitableBunching(
			context.Background(),
			"test-family-id",
			"test-member-id",
			3,
			decimal.NewFromInt(5000),   // Low annual giving
			decimal.NewFromInt(14600),  // Single standard deduction
			decimal.NewFromInt(8000),   // Limited other deductions
			decimal.NewFromFloat(0.24), // 24% marginal rate
		)

		require.NoError(t, err)

		// With low giving, annual should be recommended
		assert.Equal(t, "ANNUAL", result.RecommendedStrategy)
		assert.Nil(t, result.DAFRecommendation)
	})
}

func TestGetStateTaxData(t *testing.T) {
	service := NewTaxOptimizationService(nil)
	data := service.getStateTaxData()

	t.Run("AllStates", func(t *testing.T) {
		// Verify all states are present
		assert.Contains(t, data, "CA")
		assert.Contains(t, data, "NY")
		assert.Contains(t, data, "FL")
		assert.Contains(t, data, "TX")
		assert.Contains(t, data, "NV")
		assert.Contains(t, data, "WA")
	})

	t.Run("CaliforniaData", func(t *testing.T) {
		ca := data["CA"]
		assert.Equal(t, "California", ca.Name)
		assert.True(t, ca.IncomeTaxRate.GreaterThan(decimal.NewFromFloat(0.12)))
		assert.Equal(t, decimal.Zero, ca.EstateTaxRate) // No estate tax
		assert.Equal(t, 183, ca.ResidencyDays)
	})

	t.Run("FloridaData", func(t *testing.T) {
		fl := data["FL"]
		assert.Equal(t, "Florida", fl.Name)
		assert.Equal(t, decimal.Zero, fl.IncomeTaxRate) // No income tax
		assert.Equal(t, decimal.Zero, fl.EstateTaxRate) // No estate tax
	})

	t.Run("NewYorkData", func(t *testing.T) {
		ny := data["NY"]
		assert.Equal(t, "New York", ny.Name)
		assert.True(t, ny.IncomeTaxRate.GreaterThan(decimal.Zero))
		assert.True(t, ny.EstateTaxRate.GreaterThan(decimal.Zero)) // Has estate tax
		assert.True(t, ny.EstateTaxExemption.GreaterThan(decimal.Zero))
	})
}

func TestNIITThresholds(t *testing.T) {
	service := NewTaxOptimizationService(nil)

	tests := []struct {
		filingStatus string
		expected     int64
	}{
		{"SINGLE", 200000},
		{"MARRIED_JOINT", 250000},
		{"MARRIED_SEPARATE", 125000},
		{"HEAD_OF_HOUSEHOLD", 200000},
	}

	for _, tt := range tests {
		t.Run(tt.filingStatus, func(t *testing.T) {
			threshold := service.getNIITThreshold(tt.filingStatus)
			assert.Equal(t, decimal.NewFromInt(tt.expected), threshold)
		})
	}
}

// Helper function to find state detail by code
func findStateDetail(details []StateResidencyDetail, stateCode string) *StateResidencyDetail {
	for _, d := range details {
		if d.StateCode == stateCode {
			return &d
		}
	}
	return nil
}
